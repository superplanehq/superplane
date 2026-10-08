package bitbucket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// Bitbucket comment webhook event keys. Only new comments start runs;
// updates and deletions never do.
const (
	pullRequestCommentCreated = "pullrequest:comment_created"
	pullRequestCommentUpdated = "pullrequest:comment_updated"
	pullRequestCommentDeleted = "pullrequest:comment_deleted"
)

type OnPullRequestComment struct{}

type OnPullRequestCommentConfiguration struct {
	Repository              string `json:"repository" mapstructure:"repository"`
	ContentFilter           string `json:"contentFilter" mapstructure:"contentFilter"`
	OnlyFactoryPullRequests bool   `json:"onlyFactoryPullRequests" mapstructure:"onlyFactoryPullRequests"`
}

func (p *OnPullRequestComment) Name() string {
	return "bitbucket.onPullRequestComment"
}

func (p *OnPullRequestComment) Label() string {
	return "On Pull Request Comment"
}

func (p *OnPullRequestComment) Description() string {
	return "Listen to Bitbucket pull request comments"
}

func (p *OnPullRequestComment) Documentation() string {
	return `The On Pull Request Comment trigger starts a workflow execution when comments are added on a pull request.

## Use Cases

- **PR feedback**: React to reviewer comments on a pull request
- **Bot interactions**: Respond to mentions in pull request comments
- **Notification systems**: Notify teams when pull requests receive comments

## Configuration

- **Repository**: Select the Bitbucket repository to monitor
- **Content Filter**: Optional filter on comment content. Mentions that start with @ match that mention in the comment body. Other values are regular expressions.
- **Only pull requests in this factory**: Start a run only when the comment belongs to a pull request in this factory.

## Event Data

Each comment event includes:
- **comment**: Comment information including body, author, timestamps, parent thread, and inline file/line context
- **pullrequest**: The pull request the comment belongs to
- **repository**: Repository information
- **actor**: User who added the comment

Bitbucket payloads carry no bot marker, so this trigger has no bot filter. Feedback automations exclude their own replies through the replying integration instead.

## Webhook Setup

This trigger automatically sets up a Bitbucket webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed. One repository webhook is shared between triggers.`
}

func (p *OnPullRequestComment) Icon() string {
	return "bitbucket"
}

func (p *OnPullRequestComment) Color() string {
	return "blue"
}

func (p *OnPullRequestComment) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "contentFilter",
			Label:       "Content Filter",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., /solve or @ada",
			Description: "Optional filter on comment content. Mentions that start with @ match that mention in the comment body. Other values are regular expressions.",
		},
		{
			Name:        "onlyFactoryPullRequests",
			Label:       "Only pull requests in this factory",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Start a run only when the comment belongs to a pull request in this factory.",
		},
	}
}

func (p *OnPullRequestComment) Setup(ctx core.TriggerContext) error {
	config := OnPullRequestCommentConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	repo, err := ensureRepoInMetadata(ctx.HTTP, ctx.Metadata, ctx.Integration, config.Repository)
	if err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventTypes:     []string{pullRequestCommentCreated},
		RepositorySlug: repo.Slug,
	})
}

func (p *OnPullRequestComment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (p *OnPullRequestComment) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (p *OnPullRequestComment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	eventKey := ctx.Headers.Get("X-Event-Key")
	if eventKey == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-Event-Key header")
	}

	if eventKey != pullRequestCommentCreated {
		return http.StatusOK, nil, nil
	}

	if code, err := verifyBitbucketSignature(ctx); err != nil {
		return code, nil, err
	}

	data := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &data); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %v", err)
	}

	config := OnPullRequestCommentConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	if !payloadRepositoryMatchesTrigger(ctx, config.Repository) {
		return http.StatusOK, nil, nil
	}

	event, ok := normalizePullRequestCommentEvent(data)
	if !ok {
		return http.StatusOK, nil, nil
	}

	if !matchBitbucketContentFilter(config.ContentFilter, event) {
		return http.StatusOK, nil, nil
	}

	if config.OnlyFactoryPullRequests {
		matched, code, matchErr := matchFactoryPullRequest(ctx, event)
		if matchErr != nil || !matched {
			return code, nil, matchErr
		}
	}

	if err := ctx.Events.Emit("bitbucket.pullRequestComment", event); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (p *OnPullRequestComment) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// normalizePullRequestCommentEvent builds the comment event contract.
// Inline file/line context passes through when the comment anchors to code;
// stale inline commits are filtered by feedback consumers, not here.
func normalizePullRequestCommentEvent(payload map[string]any) (map[string]any, bool) {
	comment, ok := payload["comment"].(map[string]any)
	if !ok {
		return nil, false
	}
	pullRequest, ok := payload["pullrequest"].(map[string]any)
	if !ok {
		if nested, ok := comment["pullrequest"].(map[string]any); ok {
			pullRequest = nested
		} else {
			return nil, false
		}
	}

	normalized := map[string]any{
		"comment":     normalizeComment(comment, payload),
		"pullrequest": pullRequest,
	}
	if repository, ok := payload["repository"].(map[string]any); ok {
		normalized["repository"] = repository
	}
	if actor, ok := payload["actor"].(map[string]any); ok {
		normalized["actor"] = actor
	} else if user, ok := comment["user"].(map[string]any); ok {
		normalized["actor"] = user
	}
	return normalized, true
}

func normalizeComment(comment, payload map[string]any) map[string]any {
	normalized := map[string]any{}
	if id, ok := comment["id"]; ok {
		normalized["id"] = id
	}
	if content, ok := comment["content"].(map[string]any); ok {
		normalized["body"] = content["raw"]
		normalized["content"] = content
	}
	if user, ok := comment["user"].(map[string]any); ok {
		normalized["author"] = user
		normalized["nickname"] = user["nickname"]
		// ponytail: aliases keep shared canvas expressions working;
		// Bitbucket nicknames stand in for GitHub logins
		if _, ok := user["login"]; !ok {
			if nickname, _ := user["nickname"].(string); nickname != "" {
				user["login"] = nickname
			}
		}
		if _, ok := user["html_url"]; !ok {
			user["html_url"] = bitbucketUserURL(user)
		}
	}
	for _, key := range []string{"created_on", "updated_on", "deleted_on"} {
		if value, ok := comment[key]; ok {
			normalized[key] = value
		}
	}
	if parent, ok := comment["parent"].(map[string]any); ok {
		normalized["parent"] = map[string]any{"id": parent["id"]}
	}
	if inline, ok := comment["inline"].(map[string]any); ok {
		normalized["inline"] = inlineCommentContext(inline)
		if path, ok := inline["path"].(string); ok {
			normalized["path"] = path
		}
	}
	if url := commentURL(payload, comment); url != "" {
		normalized["url"] = url
		normalized["html_url"] = url
	}
	return normalized
}

func bitbucketUserURL(user map[string]any) string {
	if links, ok := user["links"].(map[string]any); ok {
		if html, ok := links["html"].(map[string]any); ok {
			if href, _ := html["href"].(string); strings.TrimSpace(href) != "" {
				return strings.TrimSpace(href)
			}
		}
	}
	if nickname, _ := user["nickname"].(string); strings.TrimSpace(nickname) != "" {
		return "https://bitbucket.org/" + strings.TrimSpace(nickname) + "/"
	}
	return ""
}

// inlineCommentContext carries file/line context for code-anchored comments.
func inlineCommentContext(inline map[string]any) map[string]any {
	context := map[string]any{}
	for _, key := range []string{"path", "from", "to"} {
		if value, ok := inline[key]; ok {
			context[key] = value
		}
	}
	return context
}

// commentURL best-effort links back to the comment. Bitbucket anchors PR
// comments with #comment-<id>.
func commentURL(payload, comment map[string]any) string {
	id, _ := comment["id"].(float64)
	if id == 0 {
		if asInt, ok := comment["id"].(int); ok {
			id = float64(asInt)
		}
	}
	if id == 0 {
		return ""
	}
	pr, _ := payload["pullrequest"].(map[string]any)
	if pr == nil {
		pr, _ = comment["pullrequest"].(map[string]any)
	}
	links, _ := pr["links"].(map[string]any)
	html, _ := links["html"].(map[string]any)
	href, _ := html["href"].(string)
	if href == "" {
		return ""
	}
	return fmt.Sprintf("%s#comment-%d", strings.TrimRight(href, "/"), int64(id))
}

// matchBitbucketContentFilter matches @mentions in the comment body and
// everything else as a regular expression against that body.
func matchBitbucketContentFilter(filter string, event map[string]any) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return true
	}
	comment, _ := event["comment"].(map[string]any)
	body, _ := comment["body"].(string)
	if strings.HasPrefix(filter, "@") {
		return bitbucketMentionMatches(filter, body)
	}
	matched, err := regexp.MatchString(filter, body)
	if err != nil {
		return strings.Contains(body, filter)
	}
	return matched
}

// bitbucketMentionMatches reports whether body contains mention as a whole
// nickname. A longer nickname that only starts with the filter does not match.
func bitbucketMentionMatches(mention, body string) bool {
	mention = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(mention), "@"))
	if mention == "" {
		return false
	}

	lower := strings.ToLower(body)
	needle := "@" + mention
	start := 0
	for {
		index := strings.Index(lower[start:], needle)
		if index < 0 {
			return false
		}
		index += start
		after := index + len(needle)
		if after == len(lower) || !isBitbucketNicknameChar(lower[after]) {
			return true
		}
		start = after
	}
}

func isBitbucketNicknameChar(char byte) bool {
	return (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' || char == '_'
}
