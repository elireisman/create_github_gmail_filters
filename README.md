### GitHub Gmail Labels & Filters Generator

#### Purpose
Uses Google Gmail API to create labels and Inbox filters for all your GitHub-org repos that fill up your Inbox! By default, labels and filters are created for every Github org repo you are currently subscribed to. Alternately, labels and filters can be generated for each repo you have checked out at `~/gihtub` on your local machine. The script updates the Gmail account you authorize, generating a label and a filter that will route all future notifications for each target repo to its respective label folder. Labels with unread mail will appear prominently, and with an unread message count, in the left-hand Gmail menu panel.


#### Setup
Clone the repo, `cd` into it, and run `make` or `make local` (see Usage section below.) Next, check your Chrome browser and authorize the script with Google for the intended Gmail account. When prompted to permit the `create_filters` app to accept connections, _be sure to click_ `Allow`.

You'll need to have `Golang`, `Git` and `make` installed. You will also need Oauth credentials.

To obtain API credentials, you'll need to visit Google's [Console Developers page](https://console.developers.google.com) and create some Gmail API credentials. Use them for the script run and destroy them after if you like. Remember to set the following attributes on the credentials:
    * `redirect_uris` setting in the generated credentials must be `http://localhost:9292`
    * The credentials will need the following 3 Gmail permissions: gmail.GmailReadonlyScope, gmail.GmailLabelsScope, gmail.GmailSettingsBasicScope

Afterwards, download the secrets file from the Google Credentials page as JSON, and place this file in the root directory of this repo, renaming it to `client_secret.json`. The script is idempotent and can be rerun if your subscriptions or local checkouts change.


### Usage
When the `client_secret.json` file is in place, run one of the following:
    * `make` will create a Gmail label + routing filter for each Github-org repo in you're subscribed to for notifications
    * `make local` will create a Gmail label + routing filter for each Github-org repo you have checked out locally at `~/github`
    * `make clean` clears old auth token; useful if the script fails with expired auth token errors


### Troubleshooting
If the script fails saying your auth token is expired, run `make clean` to clear it, then re-auth with Google on the next run. You could also choose to auth a different Gmail account after running `make clean`.


### TODOs
    * add OAuth2 lib for GitHub repo subscriptions-for-user API calls
    * get new filters to retroactively scan and re-label older emails at filter creation time, as you can in the Gmail Settings -> Filters menus
    * paramterize more things

