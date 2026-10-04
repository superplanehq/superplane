package factory

import (
	"fmt"
	"net/http"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const OnPullRequestConflictTriggerName = "onPullRequestConflict"
const OnPullRequestConflictPayloadType = "pullRequest.conflictDetected"

func init() {
	registry.RegisterTrigger(OnPullRequestConflictTriggerName, &OnPullRequestConflict{})
}

// OnPullRequestConflict starts a workflow when SuperPlane detects a merge
// conflict on a pull request. SuperPlane emits the event. The trigger has no webhook.
type OnPullRequestConflict struct{}

func (t *OnPullRequestConflict) Name() string {
	return OnPullRequestConflictTriggerName
}

func (t *OnPullRequestConflict) Label() string {
	return "On Merge Conflict"
}

func (t *OnPullRequestConflict) Description() string {
	return "Start when a pull request has merge conflicts"
}

func (t *OnPullRequestConflict) Documentation() string {
	return `The On Merge Conflict trigger starts a workflow when SuperPlane detects a merge conflict on a pull request.

SuperPlane emits the event. GitHub does not send a webhook when a sibling pull request becomes conflicted.

## Event Data

Each event has type ` + "`pullRequest.conflictDetected`" + `. The payload uses the same pull request fields as a GitHub pull request event:

` + "```" + `
{
  "repository": {
    "full_name": "acme/app",
    "html_url": "https://github.com/acme/app"
  },
  "pull_request": {
    "number": 12,
    "html_url": "https://github.com/acme/app/pull/12",
    "head": { "sha": "abc123", "ref": "feature" },
    "base": { "ref": "main" }
  }
}
` + "```" + `
`
}

func (t *OnPullRequestConflict) Icon() string {
	return "factory"
}

func (t *OnPullRequestConflict) Color() string {
	return "orange"
}

func (t *OnPullRequestConflict) ExampleData() map[string]any {
	return map[string]any{
		"type":      OnPullRequestConflictPayloadType,
		"timestamp": "2026-01-01T00:00:00Z",
		"data": map[string]any{
			"repository": map[string]any{
				"full_name": "acme/app",
				"html_url":  "https://github.com/acme/app",
			},
			"pull_request": map[string]any{
				"number":   12,
				"html_url": "https://github.com/acme/app/pull/12",
				"head": map[string]any{
					"sha": "abc123def456",
					"ref": "feature",
				},
				"base": map[string]any{
					"ref": "main",
				},
			},
		},
	}
}

func (t *OnPullRequestConflict) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "repository",
			Label:       "Repository",
			Description: "Repository in owner/name format.",
			Type:        configuration.FieldTypeString,
			Required:    true,
		},
	}
}

func (t *OnPullRequestConflict) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (t *OnPullRequestConflict) Setup(ctx core.TriggerContext) error {
	return nil
}

func (t *OnPullRequestConflict) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnPullRequestConflict) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, fmt.Errorf("hook %s not supported", ctx.Name)
}

func (t *OnPullRequestConflict) Cleanup(ctx core.TriggerContext) error {
	return nil
}
