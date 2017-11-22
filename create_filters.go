package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/elireisman/create_github_gmail_filters/gutils"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/googleapi"
)

func callbackServer(out chan string) *http.Server {
	server := &http.Server{
		Addr:           ":9292",
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			log.Fatalf("Failed to parse incoming params from auth server, err=%s", err)
		}
		out <- r.FormValue("code")

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		io.WriteString(w, "<body><h2>Success!</h2><h3>Auth completed, check your console window for status during Gmail API calls</h2></body>")
	})

	return server
}

func getLocalGHRepos() []string {
	out := []string{}
	ghPath := strings.Join([]string{os.Getenv("HOME"), "github"}, "/")
	log.Printf("Listing repos from: %s", ghPath)
	contents, err := ioutil.ReadDir(ghPath)
	if err != nil {
		log.Fatalf("couldn't list contents of %s, err=%s", ghPath, err)
	}

	for _, target := range contents {
		if target.IsDir() {
			out = append(out, target.Name())
		}
	}

	return out
}

func createLabel(svc *gmail.UsersLabelsService, labelName string) (*gmail.Label, error) {
	log.Printf("- Attempting to create label '%s'", labelName)
	label := &gmail.Label{
		Name: labelName,
		MessageListVisibility: "show",
		LabelListVisibility:   "labelShowIfUnread",
		Type:                  "user",
	}
	resp, err := svc.Create("me", label).Do()

	// if the label already exists, don't blow up (repeated runs should be idempotent anyway)
	if err != nil {
		gerr := err.(*googleapi.Error)
		if gerr.Code == http.StatusConflict {
			log.Printf("Label already exists: %s ", gerr.Message)
			err = nil
		}
	}
	return resp, err
}

func createFilter(svc *gmail.UsersSettingsFiltersService, label *gmail.Label, repo string) (*gmail.Filter, error) {
	noReply := fmt.Sprintf("%s@noreply.github.com", repo)
	subjectTag := fmt.Sprintf("[%s]", label.Name)

	log.Printf("- Attempting to create new inbox filter (id %s) routing mail to label '%s'", label.Id, label.Name)
	filter := &gmail.Filter{
		Action: &gmail.FilterAction{
			AddLabelIds:    []string{label.Id},
			RemoveLabelIds: []string{"INBOX"},
		},
		Criteria: &gmail.FilterCriteria{
			To:      noReply,
			Subject: subjectTag,
		},
	}
	resp, err := svc.Create("me", filter).Do()

	// if the filter already exists, don't blow up (repeated runs should be idempotent anyway)
	if err != nil {
		gerr := err.(*googleapi.Error)
		if gerr.Code == http.StatusConflict {
			log.Printf("Label already exists: %s ", gerr.Message)
			err = nil
		}
	}
	return resp, err
}

func main() {
	ctx := context.Background()

	b, err := ioutil.ReadFile("client_secret.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file, err=%s", err)
	}

	codeChan := make(chan string)
	server := callbackServer(codeChan)
	go server.ListenAndServe()

	// If modifying these scopes, delete your previously saved credentials
	// at ~/.credentials/gmail-go-quickstart.json
	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope, gmail.GmailLabelsScope, gmail.GmailSettingsBasicScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config, err=%s", err)
	}
	client := gutils.GetClient(ctx, config, codeChan)

	srv, err := gmail.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve gmail Client, err=%s", err)
	}

	labelSvc := srv.Users.Labels
	filterSvc := srv.Users.Settings.Filters
	for _, repo := range getLocalGHRepos() {
		// create a label for each repo name
		labelName := "github/" + repo
		labelResp, err := createLabel(labelSvc, labelName)
		if err != nil {
			if labelResp != nil {
				log.Printf("Response: %v", *labelResp)
			}
			log.Fatalf("Failed to create label '%s', err=%s", labelName, err)
		}

		// inlcude response from label-create as it's a *Label populated with metadata
		// if all went well, and we'll need it's ID field to associate the filter with it
		if labelResp != nil {
			if resp, err := createFilter(filterSvc, labelResp, repo); err != nil {
				if resp != nil {
					log.Printf("Response: %v", *resp)
				}
				log.Fatalf("Failed to create mail filter + routing to label '%s' failed, err=%s", labelName, err)
			}
		}
	}

	log.Printf("Run complete, thanks for playing - now go check your email!")
}
