package giteabot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	gitea "code.gitea.io/gitea/modules/structs"
	"github.com/keybase/go-keybase-chat-bot/kbchat/types/chat1"
	"github.com/keybase/managed-bots/base"
)

// EventType represents a Gitea webhook event
type EventType string

// List of supported events
//
// Find them in Gitea's source at /models/webhook.go as HookEventType.
// To correlate with when each of these trigger, see the Trigger On -> Custom Events options
// when editing a repo's webhook in a Gitea project. Those descriptions are helpful.
const (
	EventTypeCreate              EventType = "create"
	EventTypeDelete              EventType = "delete"
	EventTypeFork                EventType = "fork"
	EventTypePush                EventType = "push"
	EventTypeIssues              EventType = "issues"
	EventTypeIssueComment        EventType = "issue_comment"
	EventTypeRepository          EventType = "repository"
	EventTypeRelease             EventType = "release"
	EventTypePullRequest         EventType = "pull_request"
	EventTypePullRequestApproved EventType = "pull_request_approved"
	EventTypePullRequestRejected EventType = "pull_request_rejected"
	EventTypePullRequestComment  EventType = "pull_request_comment"
)

const eventTypeHeader = "X-Gitea-Event"

// WebhookEventType returns the event type for the given request.
func WebhookEventType(r *http.Request) EventType {
	return EventType(r.Header.Get(eventTypeHeader))
}

// ParseWebhook parses the event payload. For recognized event types, a
// value of the corresponding struct type will be returned. An error will
// be returned for unrecognized event types.
//
// The payloads are defined in Gitea's source at modules/structs/hook.go
//   https://github.com/go-gitea/gitea/blob/master/modules/structs/hook.go
//
// Example usage:
//
// func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
//     payload, err := ioutil.ReadAll(r.Body)
//     if err != nil { ... }
//     event, err := ParseWebhook(EventType(r), payload)
//     if err != nil { ... }
//     switch event := event.(type) {
//     case *gitea.PushPayload:
//         processPushEvent(event)
//     case *gitea.ForkPayload:
//         processForkEvent(event)
//     ...
//     }
// }
func ParseWebhook(eventType EventType, payload []byte) (event interface{}, err error) {
	switch eventType {
	case EventTypePush:
		event = &gitea.PushPayload{}
	case EventTypeCreate:
		event = &gitea.CreatePayload{}
	case EventTypeDelete:
		event = &gitea.DeletePayload{}
	case EventTypeFork:
		event = &gitea.ForkPayload{}
	case EventTypeIssues:
		event = &gitea.IssuePayload{}
	case EventTypeIssueComment:
		event = &gitea.IssueCommentPayload{}
	case EventTypeRepository:
		event = &gitea.RepositoryPayload{}
	case EventTypeRelease:
		event = &gitea.ReleasePayload{}
	case EventTypePullRequest, EventTypePullRequestApproved, EventTypePullRequestRejected, EventTypePullRequestComment:
		event = &gitea.PullRequestPayload{}
	default:
		return nil, fmt.Errorf("unexpected event type: %s", eventType)
	}

	if err := json.Unmarshal(payload, event); err != nil {
		return nil, err
	}

	return event,nil
}

// Return a list of all commit messages from an event
func getCommitMessages(event *gitea.PushPayload) []string {
	var commitMsgs = make([]string, 0)
	for _, commit := range event.Commits {
		commitMsgs = append(commitMsgs, commit.Message)
	}
	return commitMsgs
}

// Convert a ref like "refs/head/master" to a branch like "master"
func refToBranch(ref string) string {
	refFields := strings.Split(ref, "/")
	branch := strings.Join(refFields[2:], "/")

	return branch
}

// Formatters
func formatSetupInstructions(giteaURL string, repo string, msg chat1.MsgSummary, httpAddress string, secret string) (res string) {
	back := "`"
	message := fmt.Sprintf(`
To configure your project to send notifications, go to %s/%s/settings/hooks and add a new Gitea webhook.
For “Target URL”, enter %s%s/giteabot/webhook%s.
"HTTP Method" is POST and the "Content Type" is application/json.
For “Secret”, enter %s%s%s.
Remember to check all the triggers you would like me to update you on.

Happy coding!`,
		giteaURL, repo, back, httpAddress, back, back, base.MakeSecret(repo, msg.ConvID, secret), back)
	return message
}

func formatCommitString(commit string, maxLen int) string {
	firstLine := strings.Split(commit, "\n")[0]
	if len(firstLine) > maxLen {
		firstLine = strings.TrimSpace(firstLine[:maxLen]) + "..."
	}
	return firstLine
}

// Borrowed from https://github.com/keybase/managed-bots/blob/master/base/git/git.go (3dbf0f6)
func FormatPushMsg(username string, repo string, branch string, numCommits int, messages []string, commitURL string) (res string) {
	res = fmt.Sprintf("%s pushed %d commit", username, numCommits)
	if numCommits != 1 {
		res += "s"
	}
	res += fmt.Sprintf(" to %s %s:\n", repo, branch)
	for _, msg := range messages {
		res += fmt.Sprintf("- `%s`\n", formatCommitString(msg, 50))
	}

	res += fmt.Sprintf("\n%s", commitURL)
	return res
}

func FormatCreateMsg(ref string, refType string, repo string) string {
	return fmt.Sprintf("Created new %s %s in repo %s", refType, ref, repo)
}

func FormatDeleteMsg(ref string, refType string, repo string) string {
	return fmt.Sprintf("Deleted %s %s in repo %s", refType, ref, repo)
}

func FormatForkMsg(original string, newFork string) string {
	return fmt.Sprintf("%s has been forked to %s", original, newFork)
}

func FormatIssueMsg(action gitea.HookIssueAction, username string, issueNum int64, repo string, assignee string, title string, issueURL string) (message string) {
	// We intentionally don't handle every issue action here
	switch action {
	case gitea.HookIssueOpened, gitea.HookIssueClosed, gitea.HookIssueReOpened, gitea.HookIssueEdited:
		message = fmt.Sprintf("%s %s issue \"%s\" (#%d) on %s: %s", username, action, title, issueNum, repo, issueURL)
	case gitea.HookIssueAssigned:
		message = fmt.Sprintf("%s %s issue \"%s\" (#%d) on %s to %s: %s", username, action, title, issueNum, repo, assignee, issueURL)
	default:
		message = fmt.Sprintf("%s %s issue #%d", username, action, issueNum)
	}

	return message
}

func FormatIssueCommentMsg(action gitea.HookIssueCommentAction, username string, issueNum int64, repo string, comment string, issueTitle string, commentURL string) (message string) {
	switch action {
	case gitea.HookIssueCommentCreated:
		message = fmt.Sprintf("%s commented on issue \"%s\" (#%d) on %s:\n%s\n%s", username, issueTitle, issueNum, repo, comment, commentURL)
	case gitea.HookIssueCommentDeleted:
		message = fmt.Sprintf("%s deleted their comment on issue \"%s\" (#%d) on %s:\n%s", username, issueTitle, issueNum, repo, comment)
	case gitea.HookIssueCommentEdited:
		message = fmt.Sprintf("%s edited their comment on issue \"%s\" (#%d) on %s:\n%s\n%s", username, issueTitle, issueNum, repo, comment, commentURL)
	}

	return message
}

func FormatRepositoryMsg(action gitea.HookRepoAction, username string, repo string) (message string) {
	switch action {
	case gitea.HookRepoCreated, gitea.HookRepoDeleted:
		message = fmt.Sprintf("%s %s repository %s", username, action, repo)
	}

	return message
}

func FormatReleaseMsg(action gitea.HookReleaseAction, username string, repo string, release string, tag string, tarURL string) (message string) {
	switch action {
	case gitea.HookReleasePublished, gitea.HookReleaseUpdated:
		message = fmt.Sprintf("%s %s release \"%s\" (%s) in %s: %s", username, action, release, tag, repo, tarURL)
	case gitea.HookReleaseDeleted:
		message = fmt.Sprintf("%s %s release \"%s\" (%s) in %s", username, action, release, tag, repo)
	}

	return message
}

func FormatPullRequestMsg(action gitea.HookIssueAction, username string, repo string, prNum int64, title string, sourceBranch string, assignee string, URL string) (message string) {
	// We intentionally don't handle every action here
	// Note that PRs use "issue actions"
	switch action {
	case gitea.HookIssueOpened, gitea.HookIssueClosed, gitea.HookIssueReOpened, gitea.HookIssueEdited:
		message = fmt.Sprintf("%s %s PR \"%s\" (#%d) on %s from source %s: %s", username, action, title, prNum, repo, sourceBranch, URL)
	case gitea.HookIssueAssigned:
		message = fmt.Sprintf("%s %s PR \"%s\" (#%d) on %s to %s: %s", username, action, title, prNum, repo, assignee, URL)
	default:
		message = fmt.Sprintf("%s %s PR #%d", username, action, prNum)
	}

	return message
}