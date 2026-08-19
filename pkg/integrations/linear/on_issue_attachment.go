package linear

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnIssueAttachment struct{}

type OnIssueAttachmentConfiguration struct {
	Team    string   `json:"team" mapstructure:"team"`
	Actions []string `json:"actions" mapstructure:"actions"`
}

func (i *OnIssueAttachment) Name() string {
	return "linear.onIssueAttachment"
}

func (i *OnIssueAttachment) Label() string {
	return "On Issue Attachment"
}

func (i *OnIssueAttachment) Description() string {
	return "Listen to attachment events on Linear issues"
}

func (i *OnIssueAttachment) Documentation() string {
	return `The On Issue Attachment trigger starts a workflow execution when a link attachment is
added, updated or removed on an issue in a Linear team.

## Use Cases

- **Triage on evidence**: Raise priority when an alert or error report is attached to an issue
- **React to linked work**: Start a workflow when a pull request is attached
- **Clean-up**: Act when an attachment is removed from an issue

## Configuration

- **Team** (required): Linear team to monitor
- **Actions** (required): Which attachment actions to listen for (added, updated, removed).
  Default: added.

## Outputs

- **Default channel**: Emits the Linear webhook payload, including ` + "`action`" + `,
` + "`actor`" + `, and a ` + "`data`" + ` object with the attachment ` + "`title`" + `,
` + "`subtitle`" + `, ` + "`url`" + ` and ` + "`id`" + `.

Linear's attachment payload identifies the issue only by ` + "`issueId`" + `, so this trigger looks
the issue up and adds an ` + "`issue`" + ` object with its ` + "`identifier`" + `, ` + "`title`" + `
and ` + "`url`" + `. If that lookup fails the event is still delivered, without the ` + "`issue`" + `
object.

## Deduplication

Linear deduplicates attachments by URL within an issue, so re-attaching an existing URL arrives as an
` + "`update`" + ` rather than a ` + "`create`" + `. Select **Updated** to catch those.

## Webhook Setup

This trigger registers a Linear webhook automatically when configured, and removes it when the
trigger is deleted. Linear only allows webhook management for workspace admins or OAuth tokens with
the **admin** scope, so the Linear connection must be authorized by a **workspace admin**.`
}

func (i *OnIssueAttachment) Icon() string {
	return "linear"
}

func (i *OnIssueAttachment) Color() string {
	return "indigo"
}

func (i *OnIssueAttachment) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "team",
			Label:       "Team",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Linear team to monitor",
			Placeholder: "Select a team",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeTeam,
				},
			},
		},
		{
			Name:     "actions",
			Label:    "Actions",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"create"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Added", Value: "create"},
						{Label: "Updated", Value: "update"},
						{Label: "Removed", Value: "remove"},
					},
				},
			},
		},
	}
}

func (i *OnIssueAttachment) Setup(ctx core.TriggerContext) error {
	config := OnIssueAttachmentConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Team == "" {
		return fmt.Errorf("team is required")
	}

	//
	// The shared multi-select validation accepts an empty list for a required
	// field, so reject it here rather than saving a trigger that can never match.
	//
	if len(config.Actions) == 0 {
		return fmt.Errorf("at least one action is required")
	}

	team, err := requireTeam(ctx.Integration, config.Team)
	if err != nil {
		return err
	}

	if err := ctx.Metadata.Set(NodeMetadata{Team: team}); err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		TeamID:       config.Team,
		ResourceType: AttachmentResourceType,
	})
}

func (i *OnIssueAttachment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (i *OnIssueAttachment) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (i *OnIssueAttachment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	config := OnIssueAttachmentConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get(EventHeader)
	if eventType == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("missing %s header", EventHeader)
	}

	if eventType != AttachmentResourceType {
		return http.StatusOK, nil, nil
	}

	code, err := verifyWebhookSignature(ctx)
	if err != nil {
		return code, nil, err
	}

	data := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &data); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %v", err)
	}

	//
	// Fail closed: an empty action list matches nothing, so a trigger that
	// somehow reaches this state stays silent instead of emitting everything.
	//
	if !i.whitelistedAction(ctx.Logger, data, config.Actions) {
		return http.StatusOK, nil, nil
	}

	i.enrichWithIssue(ctx, data)

	if err := ctx.Events.Emit(AttachmentPayloadType, data); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (i *OnIssueAttachment) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (i *OnIssueAttachment) whitelistedAction(logger *log.Entry, data map[string]any, allowedActions []string) bool {
	action, ok := data["action"].(string)
	if !ok {
		return false
	}

	if !slices.Contains(allowedActions, action) {
		logger.Infof("Action %s is not in the allowed list: %v", action, allowedActions)
		return false
	}

	return true
}

// enrichWithIssue adds an issue object to the attachment payload, which Linear
// sends carrying only issueId. A failed lookup is logged and left alone, so a
// Linear hiccup costs the issue context rather than the whole event.
func (i *OnIssueAttachment) enrichWithIssue(ctx core.WebhookRequestContext, data map[string]any) {
	attachment, ok := data["data"].(map[string]any)
	if !ok {
		return
	}

	issueID, ok := attachment["issueId"].(string)
	if !ok || issueID == "" {
		return
	}

	if ctx.HTTP == nil || ctx.Integration == nil {
		return
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		ctx.Logger.Warnf("Skipping issue enrichment, failed to create client: %v", err)
		return
	}

	issue, err := client.GetIssue(issueID)
	if err != nil {
		ctx.Logger.Warnf("Skipping issue enrichment for issue %s: %v", issueID, err)
		return
	}

	attachment["issue"] = map[string]any{
		"id":         issue.ID,
		"identifier": issue.Identifier,
		"title":      issue.Title,
		"url":        issue.URL,
	}
}
