package productive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
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

- **Default channel**: Emits an envelope with a ` + "`data`" + ` object that holds the task ` + "`id`" + ` and
  ` + "`attributes`" + ` such as ` + "`title`" + ` and ` + "`description`" + `, and a ` + "`meta.event`" + ` field that names the
  change (` + "`task.created`" + ` or ` + "`task.updated`" + `).

## Webhook Setup

This trigger registers Productive.io webhooks automatically when configured, and removes them when the
trigger is deleted. Productive.io webhooks are organization-wide and need the Ultimate plan. SuperPlane
registers one remote webhook for task created and one for task updated, both pointing at
` + "`{WEBHOOKS_BASE_URL}/api/v1/webhooks/{id}`" + `. Deliveries for other projects are ignored.

Productive.io puts the task resource under ` + "`object.data`" + `. Each remote webhook has its own
signature, and SuperPlane uses that signature to tell a created task from an updated task.

Productive.io rejects registration with a 403 "webhooks_limit_exceeded" response on plans that do not
include webhooks, in which case setup fails until the organization upgrades.`
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
	config, err := decodeOnTaskConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	project, err := client.GetProject(config.Project)
	if err != nil {
		return fmt.Errorf("error finding project: %v", err)
	}

	metadata := NodeMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to parse metadata: %v", err)
	}

	metadata.Project = project

	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("error setting node metadata: %v", err)
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{ProjectID: config.Project})
}

func (t *OnTask) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnTask) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

// HandleWebhook emits the productive.task envelope for a task.created or
// task.updated delivery, filtered to the actions this node was configured
// to listen for.
func (t *OnTask) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	config, err := decodeOnTaskConfiguration(ctx.Configuration)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	//
	// Productive.io does not send an event header. The signature token of
	// the remote webhook that delivered the body is the event name.
	//
	event, code, err := signedWebhookEvent(ctx)
	if err != nil {
		return code, nil, err
	}
	if event == "" {
		event = deliveryEventName(ctx)
	}
	if event == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("missing %s header", EventHeader)
	}

	//
	// A webhook is shared by every node watching the same project, so a
	// delivery this node was not configured to listen for is not an error;
	// it is simply not this node's news.
	//
	action, known := actionForEvent(event)
	if !known || !slices.Contains(config.Actions, action) {
		return http.StatusOK, nil, nil
	}

	document, err := taskDocument(ctx.Body)
	if err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %v", err)
	}
	if document == nil {
		return http.StatusBadRequest, nil, fmt.Errorf("missing task data")
	}

	//
	// Productive.io webhooks are organization-wide. A delivery for another
	// project is not this node's news.
	//
	if taskProjectID(document) != config.Project {
		return http.StatusOK, nil, nil
	}

	// Productive.io task webhooks often omit the task list. The intake
	// filter reads that relationship, so load it before the event is emitted.
	if err := ensureTaskList(ctx, document); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if err := ctx.Events.Emit(TaskPayloadType, TaskEnvelope(event, document)); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (t *OnTask) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// productiveDelivery is the body Productive.io posts for a task webhook.
// The JSON:API task is under object.data. A fetch of the task uses a
// top-level data field, and a real delivery does not.
type productiveDelivery struct {
	Object struct {
		Data map[string]any `json:"data"`
	} `json:"object"`
}

func ensureTaskList(ctx core.WebhookRequestContext, document map[string]any) error {
	if taskListID(document) != "" || ctx.HTTP == nil || ctx.Integration == nil {
		return nil
	}

	id, _ := document["id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	task, err := client.GetTask(id)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.WithError(err).Warnf("productive task %s: task list unavailable, continuing without it", id)
		}
		return nil
	}

	setTaskListID(document, task.TaskListID)
	return nil
}

func taskDocument(body []byte) (map[string]any, error) {
	delivery := productiveDelivery{}
	if err := json.Unmarshal(body, &delivery); err != nil {
		return nil, err
	}
	return delivery.Object.Data, nil
}

// deliveryEventName reads the event SuperPlane put on the webhook URL, then
// the custom header. The URL event is a path segment or a query value.
// The signature token is checked before this name is trusted.
func deliveryEventName(ctx core.WebhookRequestContext) string {
	if ctx.Query != nil {
		if event := strings.TrimSpace(ctx.Query.Get("event")); event != "" {
			return event
		}
	}
	return strings.TrimSpace(ctx.Headers.Get(EventHeader))
}

func decodeOnTaskConfiguration(raw any) (OnTaskConfiguration, error) {
	config := OnTaskConfiguration{}
	if err := mapstructure.Decode(raw, &config); err != nil {
		return config, fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return config, fmt.Errorf("project is required")
	}

	//
	// The shared multi-select validation accepts an empty list for a required
	// field, so reject it here rather than saving a trigger that can never match.
	//
	if len(config.Actions) == 0 {
		return config, fmt.Errorf("at least one action is required")
	}

	return config, nil
}
