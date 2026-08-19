package linear

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnIssueLabel struct{}

type OnIssueLabelConfiguration struct {
	Team   string   `json:"team" mapstructure:"team"`
	Labels []string `json:"labels" mapstructure:"labels"`
}

func (i *OnIssueLabel) Name() string {
	return "linear.onIssueLabel"
}

func (i *OnIssueLabel) Label() string {
	return "On Issue Label"
}

func (i *OnIssueLabel) Description() string {
	return "Listen for a target label being added to a Linear issue"
}

func (i *OnIssueLabel) Documentation() string {
	return `The On Issue Label trigger starts a workflow execution when one of the selected labels is
added to an issue in a Linear team.

## Use Cases

- **Kick off automation** when a ` + "`factory`" + ` label is added to an issue
- **Route work** to a pipeline when a ` + "`ready-for-deploy`" + ` label is applied
- **Notify a team** when a triage label lands on an issue

## Configuration

- **Team** (required): Linear team to monitor
- **Labels** (required): The label(s) to watch for. The trigger fires only when one of these is
  added to an issue, not on later edits of an already-labelled issue.

## Outputs

- **Default channel**: Emits the Linear webhook payload, including ` + "`action`" + `, ` + "`actor`" + `, the issue
  ` + "`url`" + `, and a ` + "`data`" + ` object with the issue ` + "`identifier`" + `, ` + "`title`" + `, ` + "`state`" + `, ` + "`team`" + ` and ` + "`labels`" + `.

## Webhook Setup

This trigger registers a Linear webhook automatically when configured, and removes it when the
trigger is deleted. Linear only allows webhook management for workspace admins or OAuth tokens with
the **admin** scope, so the Linear connection must be authorized by a **workspace admin**.`
}

func (i *OnIssueLabel) Icon() string {
	return "linear"
}

func (i *OnIssueLabel) Color() string {
	return "indigo"
}

func (i *OnIssueLabel) Configuration() []configuration.Field {
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
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Trigger when one of these labels is added to an issue",
			Placeholder: "Select labels",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  ResourceTypeLabel,
					Multi: true,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "team",
							ValueFrom: &configuration.ParameterValueFrom{Field: "team"},
						},
					},
				},
			},
		},
	}
}

func (i *OnIssueLabel) Setup(ctx core.TriggerContext) error {
	config := OnIssueLabelConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Team == "" {
		return fmt.Errorf("team is required")
	}

	//
	// A picker can save an empty slot, so reject an empty selection here rather
	// than saving a trigger that can never match.
	//
	if len(trimmedLabels(config.Labels)) == 0 {
		return fmt.Errorf("at least one label is required")
	}

	team, err := requireTeam(ctx.Integration, config.Team)
	if err != nil {
		return err
	}

	if err := ctx.Metadata.Set(NodeMetadata{Team: team}); err != nil {
		return err
	}

	//
	// Linear delivers label additions as Issue events, so subscribe to the same
	// resource type as onIssue and detect the labelling moment when handling it.
	//
	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		TeamID:       config.Team,
		ResourceType: IssueResourceType,
	})
}

func (i *OnIssueLabel) Hooks() []core.Hook {
	return []core.Hook{}
}

func (i *OnIssueLabel) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (i *OnIssueLabel) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	config := OnIssueLabelConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get(EventHeader)
	if eventType == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("missing %s header", EventHeader)
	}

	if eventType != IssueResourceType {
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

	targets := trimmedLabels(config.Labels)
	if len(targets) == 0 || !i.hasNewlyAddedLabel(ctx.Logger, data, targets) {
		return http.StatusOK, nil, nil
	}

	if err := ctx.Events.Emit(IssuePayloadType, data); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (i *OnIssueLabel) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// hasNewlyAddedLabel reports whether a target label was just added to the issue.
// It fires when the issue is created already carrying a target label, or when an
// update adds one that was not on the issue before - so a later edit of an
// already-labelled issue stays silent.
func (i *OnIssueLabel) hasNewlyAddedLabel(logger *log.Entry, data map[string]any, targetLabelIDs []string) bool {
	issue, ok := data["data"].(map[string]any)
	if !ok {
		return false
	}

	current := labelIDSet(issue["labelIds"])
	matched := []string{}
	for _, id := range targetLabelIDs {
		if current[id] {
			matched = append(matched, id)
		}
	}

	if len(matched) == 0 {
		return false
	}

	action, _ := data["action"].(string)
	if action == "create" {
		return true
	}

	//
	// On update, Linear includes updatedFrom.labelIds only when the label set
	// changed. Its absence means labels were untouched, so this is not a
	// labelling event.
	//
	updatedFrom, ok := data["updatedFrom"].(map[string]any)
	if !ok {
		return false
	}

	previousValue, ok := updatedFrom["labelIds"]
	if !ok {
		return false
	}

	previous := labelIDSet(previousValue)
	for _, id := range matched {
		if !previous[id] {
			return true
		}
	}

	logger.Infof("Target labels %v were already on the issue, not newly added", matched)
	return false
}

// labelIDSet collects the string IDs from a webhook labelIds array into a set,
// tolerating the loosely-typed []any that JSON decoding produces.
func labelIDSet(value any) map[string]bool {
	set := map[string]bool{}

	items, ok := value.([]any)
	if !ok {
		return set
	}

	for _, item := range items {
		if id, ok := item.(string); ok {
			set[id] = true
		}
	}

	return set
}
