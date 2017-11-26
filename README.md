### GitHub Gmail Labels & Filters Generator

#### Purpose
Uses Google Gmail API to create labels and Inbox filters for all your GitHub-org repos that fill up your Inbox! By default, labels and filters are created for every Github org repo you are currently watching or getting notifications from. If the `-local=true` arg is supplied, the labels and filters are generated for each repo you have checked out at `~/gihtub` on your local machine. The script buckets all incoming mail notifications from each repo into its own email label. Labels with unread mail will appear prominently (with an unread count) in the left-hand Gmail menu panel.


#### Setup
Clone the repo, `cd` into it, and run `make`. The script is idempotent and can be rerun anytime if you've checked out more repos since last run. The first run, the script will require you to drop a link into your browser for OAuth, and will prompt you for a code when auth is complete which will be cached locally until expiration.

You'll need to have `Golang`, `Git` and `make` installed. Also assumes you have some GitHub repos checked out to your local machine's `$HOME/github` dir, and that you have notifications set up in GitHub for these repos. You can set or change individual notifications on various repos on github.com _after_ the script has run.

To obtain API credentials, you'll need to visit Google's [Console Developers page](https://console.developers.google.com) and create some Gmail API credentials. You can use them for this purpose only and destroy them after. When creating new creds in Google, remember to set the following attributes:
    * `redirect_uris` setting in the generated credentials must be `http://localhost:9292`
    * The credentials will need the following 3 Gmail permissions: gmail.GmailReadonlyScope, gmail.GmailLabelsScope, gmail.GmailSettingsBasicScope


### Usage
Afterwards, download the secrets file from the Google Credentials page as JSON, and place this file in the root directory of this repo, renaming it to `client_secret.json`. When the `client_secret.json` file is in place, run one of the following:
    * `make` will create a Gmail label + routing filter for each Github-org repo in you're subscribed to for notifications
    * `make local` will create a Gmail label + routing filter for each Github-org repo you have checked out locally at `~/github`
    * `make clean` clears old auth token; useful if the script fails with expired auth token errors
If your local machine prompts you to allow `create_filters` app to accept connections, _be sure to click_ `Allow`. When you are prompted with a long URL, copy and paste it into your web browser. Upom browsing this URL, the app will receive it's authorization tokens (which are cached locally for easy rerun of the script) and begin creating the labels and filters.

If the script fails saying your token is expired, rerun the script after running: `make clean` to clear it. After, you will be re-prompted for auth, and can log into a different Google account if desired.


### TODOs
    * add OAuth2 lib for GitHub repo subscriptions-for-user API calls
    * get new filters to retroactively scan and re-label older emails at filter creation time, as you can in the Gmail Settings -> Filters menus
    * paramterize more things
    * HTTPS for Gmail OAuth2 callback server

