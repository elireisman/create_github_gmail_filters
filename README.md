### GitHub Gmail Labels & Filters Generator

#### Purpose
Uses Google Gmail API to create labels and Inbox filters for each repo you have checked out at `~/gihtub` on your laptop. Labels all incoming mail notifications from each repo into its own email label. Labels with unread mail will appear in the Gmail sidebar.

#### Usage
Clone the repo, `cd` into it, and run `make`. The script is idempotent and can be rerun anytime if you've checked out more repos since last run. The first run, the script will require you to drop a link into your browser for OAuth, and will prompt you for a code when auth is complete which will be cached locally until expiration.

#### Requirements
You'll need to have `Golang`, `Git` and `make` installed. Also assumes you have some GitHub repos checked out to your local machine's `$HOME/github` dir, and that you have notifications set up in GitHub for these repos.

Finally, you'll need to visit Google's [Console Developers page](https://console.developers.google.com) and create some Gmail API credentials. Make sure you set the `redirect_uris` setting in the generated credentials to a value of `http://localhost`. Afterwards, download the secrets file Google generated as JSON from the Credentials page. Locally, place this file in the root directory of this repo, and rename it `client_secret.json`.

