package productive

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/logging"
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
	return "Listen to task events from Productive"
}

func (t *OnTask) Documentation() string {
	return `The On Task trigger starts a workflow execution when task events occur in a Productive project.

## Use Cases

- **Backlog intake**: Create a work order when a new task is added to a project
- **Sync workflows**: Mirror Productive tasks into another tracker
- **Notifications**: Alert a channel when a task is created or updated

## Configuration

- **Project** (required): Productive project to monitor
- **Actions** (required): Which task actions to listen for (created, updated). Default: created.

## Outputs

- **Default channel**: Emits an envelope with a ` + "`data`" + ` object that holds the task ` + "`id`" + ` and
  ` + "`attributes`" + ` such as ` + "`title`" + ` and ` + "`description`" + `, and a ` + "`meta.event`" + ` field that names the
  change (` + "`task.created`" + ` or ` + "`task.updated`" + `).

## Webhook Setup

This trigger registers Productive webhooks automatically when configured, and removes them when the
trigger is deleted. Productive webhooks are organization-wide and need the Ultimate plan. SuperPlane
registers one remote webhook for task created and one for task updated, both pointing at
` + "`{WEBHOOKS_BASE_URL}/api/v1/webhooks/{id}`" + `. Deliveries for other projects are ignored.

Productive puts the task resource under ` + "`object.data`" + `. Each remote webhook has its own
signature, and SuperPlane uses that signature to tell a created task from an updated task.

Productive rejects registration with a 403 "webhooks_limit_exceeded" response on plans that do not
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
			Description: "The Productive project to monitor",
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
	if err := ensureTaskList(ctx, document, event); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	envelope := TaskEnvelope(event, document, webhookOrganizationID(ctx))
	if event == TaskUpdatedEvent {
		// Intake tells a list move from an edit by the changeset for this
		// delivery. A transient lookup error fails the delivery so
		// Productive.io retries. An update with no changeset activity is
		// emitted without a list move.
		if err := stampTaskListMove(ctx, document, envelope); err != nil {
			return http.StatusInternalServerError, nil, err
		}
	}

	if err := ctx.Events.Emit(TaskPayloadType, envelope); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil, nil
}

func (t *OnTask) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func stampTaskListMove(ctx core.WebhookRequestContext, document map[string]any, envelope map[string]any) error {
	id, _ := document["id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}

	failure := taskWebhookFailure{
		event:      TaskUpdatedEvent,
		taskID:     id,
		projectID:  taskProjectID(document),
		taskListID: taskListID(document),
	}
	if ctx.HTTP == nil || ctx.Integration == nil {
		failure.err = fmt.Errorf("productive task %s: %w", id, errTaskUpdateActivityUnavailable)
		return failTaskWebhook(ctx, failure)
	}

	deliveredAt, ok := deliveryCreatedAt(ctx.Body)
	if !ok {
		failure.err = fmt.Errorf("productive task %s: %w: delivery has no created time", id, errTaskUpdateActivityUnavailable)
		warnTaskListMoveUnknown(ctx, failure)
		return nil
	}
	failure.deliveredAt = deliveredAt

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		failure.err = err
		return failTaskWebhook(ctx, failure)
	}

	//
	// Productive.io sends task.updated for changes that write no changeset
	// activity, such as placement or status. Retrying such a delivery can
	// never find one, so only a transient API error fails the delivery.
	//
	var lastErr error
	for failure.attempts < taskListFetchAttempts {
		waitBeforeTaskFetch(failure.attempts)
		failure.attempts++
		changeset, found, err := client.taskUpdateChangesetAt(id, deliveredAt, document)
		if err != nil {
			lastErr = err
			if !isRetryableProductiveError(err) {
				break
			}
			continue
		}
		if !found {
			lastErr = nil
			continue
		}
		if move, ok := taskListMoveFromChangeset(changeset); ok {
			setTaskListMove(envelope, move)
		}
		return nil
	}

	if lastErr == nil {
		failure.err = fmt.Errorf("productive task %s: %w", id, errTaskUpdateActivityUnavailable)
		warnTaskListMoveUnknown(ctx, failure)
		return nil
	}

	failure.err = fmt.Errorf("productive task %s: %w: %w", id, errTaskUpdateActivityUnavailable, lastErr)
	if !isRetryableProductiveError(lastErr) {
		warnTaskListMoveUnknown(ctx, failure)
		return nil
	}
	return failTaskWebhook(ctx, failure)
}

// warnTaskListMoveUnknown records a task.updated delivery that is emitted
// without meta.task_list_move. Intake ignores such an update.
func warnTaskListMoveUnknown(ctx core.WebhookRequestContext, failure taskWebhookFailure) {
	logging.LogProductiveWebhookWarning(
		failure.event,
		"task update emitted without task list move",
		taskWebhookLogFields(ctx, failure, http.StatusOK),
		failure.err,
	)
}

func setTaskListMove(envelope map[string]any, move TaskListMove) {
	meta, _ := envelope["meta"].(map[string]any)
	if meta == nil {
		meta = map[string]any{}
		envelope["meta"] = meta
	}
	meta["task_list_move"] = map[string]any{
		"from": move.From,
		"to":   move.To,
	}
}

func webhookOrganizationID(ctx core.WebhookRequestContext) string {
	if ctx.Integration == nil {
		return ""
	}
	value, err := ctx.Integration.GetConfig("organizationId")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(value))
}

// productiveDelivery is the body Productive.io posts for a task webhook.
// The JSON:API task is under object.data. A fetch of the task uses a
// top-level data field, and a real delivery does not.
type productiveDelivery struct {
	Object struct {
		Data map[string]any `json:"data"`
	} `json:"object"`
}

// taskFetchRetryDelays is the wait before each retry of a task or activity
// fetch. Productive.io can send a webhook before the activity for that
// change is readable.
var taskFetchRetryDelays = []time.Duration{250 * time.Millisecond, 750 * time.Millisecond}

var taskListFetchAttempts = len(taskFetchRetryDelays) + 1

var sleep = time.Sleep

func waitBeforeTaskFetch(attempt int) {
	if attempt <= 0 || attempt > len(taskFetchRetryDelays) {
		return
	}
	sleep(taskFetchRetryDelays[attempt-1])
}

// isRetryableProductiveError reports whether a later retry of the same
// request can succeed. A 4xx other than 429 answers the same every time.
func isRetryableProductiveError(err error) bool {
	status, ok := responseStatus(err)
	if !ok {
		return true
	}
	return status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
}

func ensureTaskList(ctx core.WebhookRequestContext, document map[string]any, event string) error {
	if taskListID(document) != "" || ctx.HTTP == nil || ctx.Integration == nil {
		return nil
	}

	id, _ := document["id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}

	failure := taskWebhookFailure{
		event:     event,
		taskID:    id,
		projectID: taskProjectID(document),
	}
	if deliveredAt, ok := deliveryCreatedAt(ctx.Body); ok {
		failure.deliveredAt = deliveredAt
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		failure.err = fmt.Errorf("error creating client: %v", err)
		return failTaskWebhook(ctx, failure)
	}

	var lastErr error
	for failure.attempts < taskListFetchAttempts {
		waitBeforeTaskFetch(failure.attempts)
		failure.attempts++
		task, err := client.GetTask(id)
		if err == nil {
			setTaskListID(document, task.TaskListID)
			return nil
		}
		lastErr = err
		if !isRetryableProductiveError(err) {
			break
		}
	}

	failure.err = fmt.Errorf("productive task %s: task list unavailable: %w", id, lastErr)
	if !isRetryableProductiveError(lastErr) {
		logging.LogProductiveWebhookWarning(
			failure.event,
			"task emitted without task list",
			taskWebhookLogFields(ctx, failure, http.StatusOK),
			failure.err,
		)
		return nil
	}
	return failTaskWebhook(ctx, failure)
}

type taskWebhookFailure struct {
	event       string
	taskID      string
	projectID   string
	taskListID  string
	deliveredAt time.Time
	attempts    int
	err         error
}

func failTaskWebhook(ctx core.WebhookRequestContext, failure taskWebhookFailure) error {
	logTaskWebhookFailure(ctx, failure)
	return failure.err
}

func logTaskWebhookFailure(ctx core.WebhookRequestContext, failure taskWebhookFailure) {
	if failure.err == nil {
		return
	}
	logging.LogProductiveWebhookFailure(
		failure.event,
		taskWebhookLogFields(ctx, failure, http.StatusInternalServerError),
		failure.err,
	)
}

func taskWebhookLogFields(ctx core.WebhookRequestContext, failure taskWebhookFailure, status int) log.Fields {
	fields := log.Fields{}
	if ctx.Logger != nil {
		maps.Copy(fields, ctx.Logger.Data)
	}
	fields["status"] = status
	fields["attempts"] = failure.attempts
	if failure.taskID != "" {
		fields["productive_task_id"] = failure.taskID
	}
	if failure.projectID != "" {
		fields["project_id"] = failure.projectID
	}
	if failure.taskListID != "" {
		fields["task_list_id"] = failure.taskListID
	}
	if organizationID := webhookOrganizationID(ctx); organizationID != "" {
		fields["productive_organization_id"] = organizationID
	}
	if !failure.deliveredAt.IsZero() {
		fields["delivered_at"] = failure.deliveredAt.Format(time.RFC3339Nano)
	}
	if status, ok := responseStatus(failure.err); ok {
		fields["upstream_status"] = status
	}
	return fields
}

func responseStatus(err error) (int, bool) {
	var httpErr *responseError
	if err == nil || !errors.As(err, &httpErr) {
		return 0, false
	}
	return httpErr.statusCode, true
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
