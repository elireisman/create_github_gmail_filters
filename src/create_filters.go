package main


import (
  "encoding/json"
  "fmt"
  "io/ioutil"
  "log"
  "net/http"
  "net/url"
  "os"
  "os/exec"
  "os/user"
  "path/filepath"
  "strings"

  "golang.org/x/net/context"
  "golang.org/x/oauth2"
  "golang.org/x/oauth2/google"
  "google.golang.org/api/gmail/v1"
  "google.golang.org/api/googleapi"
)


// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
  cacheFile, err := tokenCacheFile()
  if err != nil {
    log.Fatalf("Unable to get path to cached credential file. %v", err)
  }
  tok, err := tokenFromFile(cacheFile)
  if err != nil {
    tok = getTokenFromWeb(config)
    saveToken(cacheFile, tok)
  }
  return config.Client(ctx, tok)
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
  authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
  log.Printf("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)
  if err := exec.Command("open", authURL); err != nil {
    log.Fatalf("Failed to open auth page at URL: %s", authURL)
  }

  fmt.Printf("enter code> ")
  var code string
  if _, err := fmt.Scan(&code); err != nil {
    log.Fatalf("Unable to read authorization code %v", err)
  }

  tok, err := config.Exchange(oauth2.NoContext, code)
  if err != nil {
    log.Fatalf("Unable to retrieve token from web %v", err)
  }
  return tok
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile() (string, error) {
  usr, err := user.Current()
  if err != nil {
    return "", err
  }
  tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
  os.MkdirAll(tokenCacheDir, 0700)
  return filepath.Join(tokenCacheDir,
    url.QueryEscape("gmail-go-quickstart.json")), err
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
  f, err := os.Open(file)
  if err != nil {
    return nil, err
  }
  t := &oauth2.Token{}
  err = json.NewDecoder(f).Decode(t)
  defer f.Close()
  return t, err
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) {
  log.Printf("Saving credential file to: %s\n", file)
  f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
  if err != nil {
    log.Fatalf("Unable to cache oauth token: %v", err)
  }
  defer f.Close()
  json.NewEncoder(f).Encode(token)
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
  log.Printf("* Attempting to create label '%s' for user '%s'", labelName)
  label := &gmail.Label{
    Name: labelName,
    MessageListVisibility: "show",
    LabelListVisibility: "labelShowIfUnread",
    Type: "user",
  }
  resp, err := svc.Create("me", label).Do()

  // if the label already exists, don't blow up (repeated runs should be idempotent anyway)
  if err != nil && googleapi.IsNotModified(err) {
    err = nil
  }
  return resp, err
}

func createFilter(svc *gmail.UsersSettingsFiltersService, label *gmail.Label, repo string) (*gmail.Filter, error) {
  noReply := fmt.Sprintf("%s@noreply.github.com", repo)
  subjectTag := fmt.Sprintf("[%s]", label.Name)

  log.Printf("* Attempting to create new inbox filter routing mail to you + '%s' to label '%s' (with ID %s)", noReply, label.Name, label.Id)
  filter := &gmail.Filter{
    Action: &gmail.FilterAction{
      AddLabelIds: []string{label.Id},
    },
    Criteria: &gmail.FilterCriteria{
      To: noReply,
      Subject: subjectTag,
    },
  }
  resp, err := svc.Create("me", filter).Do()

  // if the filter already exists, don't blow up (repeated runs should be idempotent anyway)
  if err != nil && googleapi.IsNotModified(err) {
    err = nil
  }
  return resp, err
}

func main() {
  ctx := context.Background()

  b, err := ioutil.ReadFile("client_secret.json")
  if err != nil {
    log.Fatalf("Unable to read client secret file, err=%s", err)
  }

  // If modifying these scopes, delete your previously saved credentials
  // at ~/.credentials/gmail-go-quickstart.json
  config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope, gmail.GmailLabelsScope, gmail.GmailSettingsBasicScope)
  if err != nil {
    log.Fatalf("Unable to parse client secret file to config, err=%s", err)
  }
  client := getClient(ctx, config)

  srv, err := gmail.New(client)
  if err != nil {
    log.Fatalf("Unable to retrieve gmail Client, err=%s", err)
  }

  labelSvc := srv.Users.Labels
  filterSvc := srv.Users.Settings.Filters
  for _, repo := range getLocalGHRepos() {
    // create a label for each repo name
    labelName := "github/" + repo
    log.Printf("Repo:", repo, "\t", "Label:", labelName)

    labelResp, err := createLabel(labelSvc, labelName)
    if err != nil {
      if labelResp != nil {
        log.Printf("Response: %v", *labelResp)
      }
      log.Fatalf("Failed to create label '%s', err=%s", labelName, err)
    }

    // inlcude response from label-create as it's a *Label populated with metadata
    // if all went well, and we'll need it's ID field to associate the filter with it
    if resp, err := createFilter(filterSvc, labelResp, repo); err != nil {
      if resp != nil {
        log.Printf("Response: %v", *resp)
      }
      log.Fatalf("Failed to create mail filter + routing to label '%s' failed, err=%s", labelName, err)
    }
  }

  log.Printf("Run complete, thanks for playing - now go check your email!")
}

