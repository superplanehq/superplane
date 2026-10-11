package bitbucket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

// Bitbucket pull-request webhook event keys. Build/commit events arrive
// separately (repo:commit_status_created/updated) and stay out of this trigger.
const (
	pullRequestEventCreated    = "pullrequest:created"
	pullRequestEventUpdated    = "pullrequest:updated"
	pullRequestEventApproved   = "pullrequest:approved"
	pullRequestEventUnapproved = "pullrequest:unapproved"
	pullRequestEventFulfilled  = "pullrequest:fulfilled"
	pullRequestEventRejected   = "pullrequest:rejected"
)

// pullRequestActionByEventKey maps a Bitbucket webhook event key to the
// normalized pull-request action carried in the shared event contract.
var pullRequestActionByEventKey = map[string]string{
	pullRequestEventCreated:    "created",
	pullRequestEventUpdated:    "updated",
	pullRequestEventApproved:   "approved",
	pullRequestEventUnapproved: "unapproved",
	pullRequestEventFulfilled:  "merged",
	pullRequestEventRejected:   "declined",
}

// pullRequestEventKeysForActions returns the webhook event keys a trigger
// configuration subscribes to.
func pullRequestEventKeysForActions(actions []string) ([]string, error) {
	byAction := map[string]string{}
	for key, action := range pullRequestActionByEventKey {
		byAction[action] = key
	}
	keys := []string{}
	for _, action := range actions {
		key, ok := byAction[strings.ToLower(strings.TrimSpace(action))]
		if !ok {
			return nil, fmt.Errorf("unknown pull request action %q", action)
		}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("at least one action is required")
	}
	return keys, nil
}

// normalizeBitbucketPullRequestEvent builds the shared Bitbucket event
// contract from a webhook payload. Bitbucket PR payloads natively carry
// repository identity, PR identity/state, source branch/hash, actor, and
// timestamps, so normalization is projection plus the mapped action.
func normalizeBitbucketPullRequestEvent(eventKey string, payload map[string]any) (map[string]any, bool) {
	action, ok := pullRequestActionByEventKey[eventKey]
	if !ok {
		return nil, false
	}
	pullRequest, ok := payload["pullrequest"].(map[string]any)
	if !ok {
		return nil, false
	}
	event := map[string]any{
		"action":      action,
		"pullrequest": pullRequest,
		// pull_request mirrors the GitHub event shape so closure canvases
		// share expressions across providers.
		"pull_request": pullRequestView(action, payload, pullRequest),
	}
	if repository, ok := payload["repository"].(map[string]any); ok {
		event["repository"] = repository
	}
	if actor, ok := payload["actor"].(map[string]any); ok {
		event["actor"] = actor
	}
	return event, true
}

// pullRequestView projects a Bitbucket pull request into the GitHub event
// shape closure expressions read: number, merged, state, timestamps, URL.
func pullRequestView(action string, payload, pullRequest map[string]any) map[string]any {
	view := map[string]any{
		"number": pullRequest["id"],
		"merged": action == "merged",
		"state":  "open",
	}
	if source, ok := pullRequest["source"].(map[string]any); ok {
		head := map[string]any{}
		if branch, ok := source["branch"].(map[string]any); ok {
			head["ref"] = branch["name"]
		}
		if commit, ok := source["commit"].(map[string]any); ok {
			head["sha"] = commit["hash"]
		}
		view["head"] = head
	}
	if title, ok := pullRequest["title"]; ok {
		view["title"] = title
	}
	if description, ok := pullRequest["description"]; ok {
		view["description"] = description
	}
	if destination, ok := pullRequest["destination"].(map[string]any); ok {
		if branch, ok := destination["branch"].(map[string]any); ok {
			view["base"] = map[string]any{"ref": branch["name"]}
		}
	}
	if author, ok := pullRequest["author"].(map[string]any); ok {
		if nickname, _ := author["nickname"].(string); strings.TrimSpace(nickname) != "" {
			view["user"] = map[string]any{"login": strings.TrimSpace(nickname)}
		}
	} else if actor, ok := payload["actor"].(map[string]any); ok {
		if nickname, _ := actor["nickname"].(string); strings.TrimSpace(nickname) != "" {
			view["user"] = map[string]any{"login": strings.TrimSpace(nickname)}
		}
	}
	if links, ok := pullRequest["links"].(map[string]any); ok {
		if html, ok := links["html"].(map[string]any); ok {
			view["html_url"] = html["href"]
		}
	}
	updated, _ := pullRequest["updated_on"].(string)
	switch action {
	case "merged":
		view["state"] = "closed"
		view["merged_at"] = updated
	case "declined":
		view["state"] = "closed"
		view["closed_at"] = updated
	}
	return view
}

// verifyBitbucketSignature checks the HMAC webhook signature shared by
// Bitbucket repository webhooks.
func verifyBitbucketSignature(ctx core.WebhookRequestContext) (int, error) {
	signature := ctx.Headers.Get("X-Hub-Signature")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("missing X-Hub-Signature header")
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature format")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error getting webhook secret")
	}

	if err := crypto.VerifySignature(secret, ctx.Body, signature); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}
	return http.StatusOK, nil
}

// payloadRepositoryMatchesTrigger reports whether a webhook payload belongs
// to the trigger subscription. Payloads without repository identity predate
// routing and pass through; payloads carrying it must match the node
// repository resolved at setup, falling back to the configured name.
func payloadRepositoryMatchesTrigger(ctx core.WebhookRequestContext, configuredRepository string) bool {
	payloadRepository := map[string]any{}
	if len(ctx.Body) > 0 {
		raw := map[string]any{}
		if err := json.Unmarshal(ctx.Body, &raw); err == nil {
			payloadRepository, _ = raw["repository"].(map[string]any)
		}
	}
	if len(payloadRepository) == 0 {
		return true
	}

	if ctx.Metadata != nil {
		var nodeMetadata NodeMetadata
		if err := mapstructure.Decode(ctx.Metadata.Get(), &nodeMetadata); err == nil && nodeMetadata.Repository != nil {
			return repositoryMetadataMatchesSlice(*nodeMetadata.Repository, payloadRepository)
		}
	}
	return repositoryNameMatchesPayload(configuredRepository, payloadRepository)
}

func repositoryMetadataMatchesSlice(repo RepositoryMetadata, payload map[string]any) bool {
	uuid, _ := payload["uuid"].(string)
	fullName, _ := payload["full_name"].(string)
	name, _ := payload["name"].(string)
	for _, candidate := range []string{uuid, fullName, name} {
		if candidate != "" && repositoryMetadataMatches(repo, candidate) {
			return true
		}
	}
	return false
}

func repositoryNameMatchesPayload(configured string, payload map[string]any) bool {
	fullName, _ := payload["full_name"].(string)
	if fullName != "" && configured != "" {
		return strings.EqualFold(strings.TrimSpace(fullName), strings.TrimSpace(configured))
	}
	return false
}
