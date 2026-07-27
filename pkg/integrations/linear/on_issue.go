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

type OnIssue struct{}

type OnIssueConfiguration struct {
	Team    string                    `json:"team" mapstructure:"team"`
	Actions []string                  `json:"actions" mapstructure:"actions"`
	Labels  []configuration.Predicate `json:"labels" mapstructure:"labels"`
}

func (i *OnIssue) Name() string {
	return "linear.onIssue"
}

func (i *OnIssue) Label() string {
	return "On Issue"
}

func (i *OnIssue) Description() string {
	return "Listen to issue events from Linear"
}

func (i *OnIssue) Documentation() string {
	return `The On Issue trigger starts a workflow execution when issue events occur in a Linear team.

## Use Cases

- **Notify Slack** when an issue is created so the team can triage it
- **Create a GitHub issue** when a Linear issue is created, for traceability
- **Update external dashboards** when an issue is completed or deleted

## Configuration

- **Team** (required): Linear team to monitor
- **Actions** (required): Which issue actions to listen for (created, updated, deleted). Default: created.
- **Labels** (optional): Only trigger for issues carrying specific labels

## Outputs

- **Default channel**: Emits the Linear webhook payload, including ` + "`action`" + `, ` + "`actor`" + `, the issue
  ` + "`url`" + `, and a ` + "`data`" + ` object with the issue ` + "`identifier`" + `, ` + "`title`" + `, ` + "`state`" + `, ` + "`team`" + ` and ` + "`labels`" + `.

## Webhook Setup

This trigger registers a Linear webhook automatically when configured, and removes it when the
trigger is deleted. Linear only allows webhook management for workspace admins or OAuth tokens with
the **admin** scope, so the Linear connection must be authorized by a **workspace admin**.`
}

func (i *OnIssue) Icon() string {
	return "linear"
}

func (i *OnIssue) Color() string {
	return "indigo"
}

func (i *OnIssue) Configuration() []configuration.Field {
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
						{Label: "Created", Value: "create"},
						{Label: "Updated", Value: "update"},
						{Label: "Deleted", Value: "remove"},
					},
				},
			},
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeAnyPredicateList,
			Required:    false,
			Description: "Only trigger for issues carrying one of these labels",
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
	}
}

func (i *OnIssue) Setup(ctx core.TriggerContext) error {
	config := OnIssueConfiguration{}
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
		ResourceType: IssueResourceType,
	})
}

func (i *OnIssue) Hooks() []core.Hook {
	return []core.Hook{}
}

func (i *OnIssue) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (i *OnIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	config := OnIssueConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get(EventHeader)
	if eventType == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("missing %s header", EventHeader)
	}

	//
	// A Linear webhook can carry several resource types,
	// so ignore anything that is not an issue event.
	//
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

	//
	// Fail closed: an empty action list matches nothing, so a trigger that
	// somehow reaches this state stays silent instead of emitting everything.
	//
	if !i.whitelistedAction(ctx.Logger, data, config.Actions) {
		return http.StatusOK, nil, nil
	}

	if len(config.Labels) > 0 && !i.hasWhitelistedLabel(ctx.Logger, data, config.Labels) {
		return http.StatusOK, nil, nil
	}

	if err := ctx.Events.Emit(IssuePayloadType, data); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (i *OnIssue) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (i *OnIssue) whitelistedAction(logger *log.Entry, data map[string]any, allowedActions []string) bool {
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

func (i *OnIssue) hasWhitelistedLabel(logger *log.Entry, data map[string]any, allowedLabels []configuration.Predicate) bool {
	issue, ok := data["data"].(map[string]any)
	if !ok {
		return false
	}

	labels, ok := issue["labels"].([]any)
	if !ok {
		return false
	}

	labelNames := []string{}
	for _, label := range labels {
		labelMap, ok := label.(map[string]any)
		if !ok {
			continue
		}

		name, ok := labelMap["name"].(string)
		if !ok {
			continue
		}

		labelNames = append(labelNames, name)
	}

	for _, labelName := range labelNames {
		if configuration.MatchesAnyPredicate(allowedLabels, labelName) {
			return true
		}
	}

	logger.Infof("Labels do not match the allowed list: Received: %v, Allowed: %v", labelNames, allowedLabels)
	return false
}
