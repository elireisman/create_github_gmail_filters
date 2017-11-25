package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"encoding/json"
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

// this server will receive the OAuth callback and extract the code we need to hit the Gmail API
func callbackServer(out chan string) *http.Server {
	return &http.Server{
		Addr:           ":9292",
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := r.ParseForm(); err != nil {
				log.Fatalf("Failed to parse incoming params from auth server, err=%s", err)
			}
			out <- r.FormValue("code")

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "text/html; charset=UTF-8")
			io.WriteString(w, "<body><h2>Success!</h2><h3>Auth completed, check your console window for status during Gmail API calls</h2></body>")
		}),
	}
}

func getWatchedGHRepos() []string {
	out := []string{}
	user := os.Getenv("USER")
	endpt := "https://api.github.com/users/" + user  + "/subscriptions"
	resp, err := http.Get(endpt)
	if err != nil {
		log.Fatalf("Failed to read watchlist from github.com for user %q", user)
	}
	subs := []map[string]interface{}{}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read watches list from %q, err=%s", endpt, err)
	}
	if err := json.Unmarshal(bytes, &subs); err != nil {
		log.Fatalf("Failed to unmarshal watches list, err=%s", err)
	}

	for _, repo := range subs {
		if fullName, ok := repo["full_name"]; ok {
			if strings.HasPrefix(fullName.(string), "github/") {
				out = append(out, repo["name"].(string))
			}
		}
	}
	return out
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
		switch err.(type) {
		case *googleapi.Error:
			gerr := err.(*googleapi.Error)
			if gerr.Code == http.StatusConflict {
				log.Printf("Label already exists: %s ", gerr.Message)
				err = nil
			}

		//case *url.Error:
		default:
			log.Printf("%#v", err)
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
	localFlag := flag.Bool("local", false, "Build repo list from local checkouts instead of GitHub subscriptions")
        flag.Parse()

	b, err := ioutil.ReadFile("client_secret.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file, err=%s", err)
	}

	ctx := context.Background()
	codeChan := make(chan string)
	server := callbackServer(codeChan)
	go server.ListenAndServe()

	// If modifying these scopes, delete your previously saved credentials at ~/.credentials/gmail-go-quickstart.json
	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope, gmail.GmailLabelsScope, gmail.GmailSettingsBasicScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config, err=%s", err)
	}
	client := gutils.GetClient(ctx, config, codeChan)

	srv, err := gmail.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve gmail Client, err=%s", err)
	}

	var targets []string
	switch *localFlag {
	case true: targets = getLocalGHRepos()
	default:   targets = getWatchedGHRepos()
	}

	labelSvc := srv.Users.Labels
	filterSvc := srv.Users.Settings.Filters
	for _, repo := range targets {
		// create a label for each repo name
		labelName := "github/" + repo
		labelResp, err := createLabel(labelSvc, labelName)
		if labelResp != nil {
			log.Printf("Response: %#v", *labelResp)
		}
		if err != nil {
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

	log.Printf("\n")
	log.Printf("Run complete, thanks for playing - now go check your email!")
}
