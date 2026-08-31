package productive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

type OnTask struct{}

type OnTaskConfiguration struct {
	Project string   `json:"project" mapstructure:"project"`
	Actions []string `json:"actions" mapstructure:"actions"`
}

func (t *OnTask) Name() string {
	return "productive.onTask"
}

func (t *OnTask) Label() string {
	return "On Task"
}

func (t *OnTask) Description() string {
	return "Listen to task events from Productive.io"
}

func (t *OnTask) Documentation() string {
	return `The On Task trigger starts a workflow execution when task events occur in a Productive.io project.

## Use Cases

- **Backlog intake**: Create a work order when a new task is added to a project
- **Sync workflows**: Mirror Productive.io tasks into another tracker
- **Notifications**: Alert a channel when a task is created or updated

## Configuration

- **Project** (required): Productive.io project to monitor
- **Actions** (required): Which task actions to listen for (created, updated). Default: created.

## Outputs

- **Default channel**: Emits the Productive.io webhook envelope, including a ` + "`data`" + ` object with the
  task ` + "`id`" + ` and ` + "`attributes`" + ` such as ` + "`title`" + ` and ` + "`description`" + `.

## Webhook Setup

This trigger registers a Productive.io webhook automatically when configured, and removes it when the
trigger is deleted.`
}

func (t *OnTask) Icon() string {
	return "productive"
}

func (t *OnTask) Color() string {
	return "indigo"
}

func (t *OnTask) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Productive.io project to monitor",
			Placeholder: "Select a project",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeProject,
				},
			},
		},
		{
			Name:     "actions",
			Label:    "Actions",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"created"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Created", Value: "created"},
						{Label: "Updated", Value: "updated"},
					},
				},
			},
		},
	}
}

func (t *OnTask) Setup(ctx core.TriggerContext) error {
	config := OnTaskConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return fmt.Errorf("project is required")
	}

	//
	// The shared multi-select validation accepts an empty list for a required
	// field, so reject it here rather than saving a trigger that can never match.
	//
	if len(config.Actions) == 0 {
		return fmt.Errorf("at least one action is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	project, err := client.GetProject(config.Project)
	if err != nil {
		return fmt.Errorf("error finding project: %v", err)
	}

	if err := ctx.Metadata.Set(NodeMetadata{Project: project}); err != nil {
		return fmt.Errorf("error setting node metadata: %v", err)
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		ProjectID: config.Project,
		Events:    eventsForActions(config.Actions),
	})
}

func (t *OnTask) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnTask) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnTask) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	config := OnTaskConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get(EventHeader)
	if eventType == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("missing %s header", EventHeader)
	}

	//
	// A shared webhook can carry events this trigger does not care about
	// (e.g. it was widened to satisfy another trigger on the same project),
	// so anything that is not a task event is ignored before checking the
	// signature or the configured actions.
	//
	if eventType != TaskCreatedEvent && eventType != TaskUpdatedEvent {
		return http.StatusOK, nil, nil
	}

	if code, err := verifyWebhookSignature(ctx); err != nil {
		return code, nil, err
	}

	if !whitelistedEvent(ctx.Logger, eventType, config.Actions) {
		return http.StatusOK, nil, nil
	}

	data := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &data); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %v", err)
	}

	if err := ctx.Events.Emit(TaskPayloadType, data); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (t *OnTask) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// whitelistedEvent reports whether the delivered event matches one of the
// trigger's configured actions. Fail closed: an empty or unknown action list
// matches nothing, so a trigger that somehow reaches this state stays silent
// instead of emitting everything.
func whitelistedEvent(logger *log.Entry, eventType string, allowedActions []string) bool {
	allowedEvents := eventsForActions(allowedActions)
	if !slices.Contains(allowedEvents, eventType) {
		logger.Infof("event %s is not in the allowed list: %v", eventType, allowedEvents)
		return false
	}

	return true
}

// verifyWebhookSignature checks the X-Productive-Signature header against an
// HMAC-SHA256 of the raw request body, signed with the secret SuperPlane gave
// Productive.io when the webhook was created.
func verifyWebhookSignature(ctx core.WebhookRequestContext) (int, error) {
	signature := ctx.Headers.Get(SignatureHeader)
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("missing %s header", SignatureHeader)
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error getting webhook secret: %v", err)
	}

	if len(secret) == 0 {
		return http.StatusInternalServerError, fmt.Errorf("missing webhook secret")
	}

	if err := crypto.VerifySignature(secret, ctx.Body, signature); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid webhook signature")
	}

	return http.StatusOK, nil
}
