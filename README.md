# Keybase Gitea Bot

Parses incoming webhook notifications from a Gitea server
and publishes those messages to a given conversation (team or person or channel).

Inspired by [keybase/managed-bots gitlabbot](https://github.com/keybase/managed-bots/tree/master/gitlabbot).

## Running

First ensure the Keybase CLI client is installed and you are logged in with the user the bot will run as.

Then add that account as a bot to your team.

1. Build the bot using Go 1.13+, like such (in this directory):
   ```
   go install .
   ```
2. Gitea Bot sets itself up to serve HTTP requests on port 8080. You can configure nginx or any other reverse proxy software in front of it if you want.
3. To start the bot, run a command like this:
   ```
   $GOPATH/bin/giteabot --http-prefix "$HOSTNAME:8080" --dsn "user:pass@host/database" --secret "some_nonce" --announcement "convid" --err-report-conv "convid" --gitea-url "http://git.internal"
   ```
4. Run `giteabot --help` for more options.

### Helpful Tips

- If you accidentally run the bot under your own username and wish to clear the `!` commands, run the following:
  ```
  keybase chat api -m '{"method": "clearcommands"}'
  ```
- To get the conversation ID (convid) so you can tell this bot where to publish announcements and errors to, look through the list of chats you are in:
  ```
  keybase chat api -p -m '{"method": "list"}' | less
  ```
- The bot will reply with a friendly message to GET requests at `$HOSTNAME:8080/giteabot`. This is its health check interface.
- The webhook handler lives at `$HOSTNAME:8080/giteabot/webhook`.

### Docker

There are a few complications running a Keybase chat bot, and it is likely easiest to deploy using Docker. See https://hub.docker.com/r/keybaseio/client for our preferred client image to get started.
