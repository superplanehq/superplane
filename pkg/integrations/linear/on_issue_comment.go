package linear

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnIssueComment struct{}

type OnIssueCommentConfiguration struct {
	Team          string                    `json:"team" mapstructure:"team"`
	Actions       []string                  `json:"actions" mapstructure:"actions"`
	ContentFilter []configuration.Predicate `json:"contentFilter" mapstructure:"contentFilter"`
}

func (i *OnIssueComment) Name() string {
	return "linear.onIssueComment"
}

func (i *OnIssueComment) Label() string {
	return "On Issue Comment"
}

func (i *OnIssueComment) Description() string {
	return "Listen to comment events on Linear issues"
}

func (i *OnIssueComment) Documentation() string {
	return `The On Issue Comment trigger starts a workflow execution when a comment is created,
updated or deleted on an issue in a Linear team.

## Use Cases

- **Run a command** when someone comments a keyword like ` + "`/deploy`" + ` on an issue
- **Sync discussion** to another tool when a comment is added
- **Notify a channel** when an issue gets new activity

## Configuration

- **Team** (required): Linear team to monitor
- **Actions** (required): Which comment actions to listen for (created, updated, deleted). Default: created.
- **Content Filter** (optional): Only trigger for comments whose body matches one of these predicates.
  A comment delivered without a body never matches.

## Outputs

- **Default channel**: Emits the Linear webhook payload, including ` + "`action`" + `, ` + "`actor`" + `, the comment
  ` + "`url`" + `, and a ` + "`data`" + ` object with the comment ` + "`body`" + `, its ` + "`user`" + `, and the ` + "`issue`" + ` it belongs to.

## Webhook Setup

This trigger registers a Linear webhook automatically when configured, and removes it when the
trigger is deleted. Linear only allows webhook management for workspace admins or OAuth tokens with
the **admin** scope, so the Linear connection must be authorized by a **workspace admin**.`
}

func (i *OnIssueComment) Icon() string {
	return "linear"
}

func (i *OnIssueComment) Color() string {
	return "indigo"
}

func (i *OnIssueComment) Configuration() []configuration.Field {
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
			Name:        "contentFilter",
			Label:       "Content Filter",
			Type:        configuration.FieldTypeAnyPredicateList,
			Required:    false,
			Description: "Only trigger for comments whose body matches one of these predicates",
			TypeOptions: &configuration.TypeOptions{
				AnyPredicateList: &configuration.AnyPredicateListTypeOptions{
					Operators: configuration.AllPredicateOperators,
				},
			},
		},
	}
}

func (i *OnIssueComment) Setup(ctx core.TriggerContext) error {
	config := OnIssueCommentConfiguration{}
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

	//
	// Predicate evaluation swallows regex errors and never matches, so reject an
	// uncompilable pattern here instead of saving a silently dead trigger.
	//
	for _, predicate := range config.ContentFilter {
		if predicate.Type != configuration.PredicateTypeMatches {
			continue
		}

		if _, err := regexp.Compile(predicate.Value); err != nil {
			return fmt.Errorf("invalid content filter pattern %q: %w", predicate.Value, err)
		}
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
		ResourceType: CommentResourceType,
	})
}

func (i *OnIssueComment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (i *OnIssueComment) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (i *OnIssueComment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	config := OnIssueCommentConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get(EventHeader)
	if eventType == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("missing %s header", EventHeader)
	}

	if eventType != CommentResourceType {
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

	if len(config.ContentFilter) > 0 && !i.matchesContentFilter(ctx.Logger, data, config.ContentFilter) {
		return http.StatusOK, nil, nil
	}

	if err := ctx.Events.Emit(CommentPayloadType, data); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (i *OnIssueComment) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (i *OnIssueComment) whitelistedAction(logger *log.Entry, data map[string]any, allowedActions []string) bool {
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

func (i *OnIssueComment) matchesContentFilter(logger *log.Entry, data map[string]any, predicates []configuration.Predicate) bool {
	comment, ok := data["data"].(map[string]any)
	if !ok {
		return false
	}

	body, ok := comment["body"].(string)
	if !ok {
		return false
	}

	if !configuration.MatchesAnyPredicate(predicates, body) {
		logger.Infof("Comment body does not match the content filter: %v", predicates)
		return false
	}

	return true
}
