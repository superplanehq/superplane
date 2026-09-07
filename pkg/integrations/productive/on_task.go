package productive

import (
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	// pollTasksHook reads the tasks that changed since the poll before it.
	// Productive.io sells webhooks as a plan feature and answers registration
	// with 403 "webhooks_limit_exceeded" on plans without it, so this trigger
	// polls rather than subscribing.
	pollTasksHook = "pollTasks"

	// pollInterval is the delay between two polls of the same project.
	pollInterval = time.Minute

	// pollPageSize is how many changed tasks one request reads, and
	// maxPollPages is how many such requests one poll makes. Together they cap
	// a poll well above the activity a minute can hold.
	pollPageSize = 50
	maxPollPages = 5
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

## How tasks arrive

The trigger reads the project every minute and emits the tasks that changed since the read before it.
Tasks that already exist when you add the trigger are not reported; only later changes are.

Productive.io offers webhooks on selected plans only, and rejects webhook registration on the other
plans. Polling therefore works on every plan, at the cost of up to one minute of delay.`
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

	//
	// A trigger that started from an empty cursor would report every task it
	// can read as if it had just arrived. The first setup therefore starts at
	// the change the project already carries, and only what happens after that
	// reaches the canvas.
	//
	if metadata.PolledUntil == "" {
		polledUntil, err := newestTaskChange(client, config.Project)
		if err != nil {
			return err
		}

		metadata.PolledUntil = polledUntil
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("error setting node metadata: %v", err)
	}

	return ctx.Requests.ScheduleActionCall(pollTasksHook, map[string]any{}, pollInterval)
}

func (t *OnTask) Hooks() []core.Hook {
	return []core.Hook{
		{
			Name: pollTasksHook,
			Type: core.HookTypeInternal,
		},
	}
}

func (t *OnTask) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	if ctx.Name != pollTasksHook {
		return nil, fmt.Errorf("hook %s not supported", ctx.Name)
	}

	return nil, t.pollTasks(ctx)
}

// HandleWebhook answers the calls every trigger has to accept. This trigger
// polls instead of subscribing, so Productive.io delivers nothing here.
func (t *OnTask) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (t *OnTask) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// pollTasks emits the tasks of the project that changed since the last poll,
// then leaves the next poll behind.
func (t *OnTask) pollTasks(ctx core.TriggerHookContext) error {
	//
	// The next poll is scheduled before anything can fail. A poll that ends
	// early must still leave a successor behind, or one bad response would
	// stop the trigger for good.
	//
	if err := ctx.Requests.ScheduleActionCall(pollTasksHook, map[string]any{}, pollInterval); err != nil {
		return err
	}

	config, err := decodeOnTaskConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	metadata := NodeMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to parse metadata: %v", err)
	}

	polledUntil, ok := parseTaskTime(metadata.PolledUntil)
	if !ok {
		//
		// Without a usable cursor the poll cannot tell new tasks from old
		// ones. Start at the current time rather than reporting the whole
		// project as new.
		//
		metadata.PolledUntil = formatTaskTime(time.Now())
		return ctx.Metadata.Set(metadata)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	documents, err := changedTasks(client, config.Project, polledUntil)
	if err != nil {
		//
		// A failed read must not fail the request: the request is retried at
		// once, which would hammer Productive.io while it is unhappy. The
		// scheduled poll picks the same tasks up one interval later.
		//
		ctx.Logger.Errorf("Error reading the changed tasks of project %s: %v", config.Project, err)
		return nil
	}

	return emitChangedTasks(ctx, config.Actions, documents, metadata, polledUntil)
}

// emitChangedTasks emits one event per task the trigger listens for, and moves
// the cursor to the newest change it handled.
func emitChangedTasks(
	ctx core.TriggerHookContext,
	actions []string,
	documents []map[string]any,
	metadata NodeMetadata,
	polledUntil time.Time,
) error {
	//
	// Productive.io answers newest first. Emitting the oldest task first keeps
	// the newest at the top of a backlog, where the reader expects it.
	//
	slices.Reverse(documents)

	cursor := polledUntil
	for _, document := range documents {
		changedAt, ok := taskTime(document, "updated_at")
		if !ok {
			continue
		}

		event, wanted := taskEvent(document, actions, polledUntil)
		if wanted {
			if err := ctx.Events.Emit(TaskPayloadType, TaskEnvelope(event, document)); err != nil {
				//
				// Stop at the first task that could not be emitted and keep
				// the cursor behind it, so the next poll starts from there
				// instead of skipping it.
				//
				ctx.Logger.Errorf("Error emitting Productive.io task %v: %v", document["id"], err)
				break
			}
		}

		if changedAt.After(cursor) {
			cursor = changedAt
		}
	}

	if !cursor.After(polledUntil) {
		return nil
	}

	metadata.PolledUntil = formatTaskTime(cursor)
	return ctx.Metadata.Set(metadata)
}

// taskEvent names the change a polled task carries, and reports whether the
// trigger was configured to listen for it. A task that appeared after the last
// poll counts as created; a task that was already there counts as updated.
func taskEvent(document map[string]any, actions []string, polledUntil time.Time) (string, bool) {
	if createdAt, ok := taskTime(document, "created_at"); ok && createdAt.After(polledUntil) {
		return TaskCreatedEvent, slices.Contains(actions, ActionCreated)
	}

	return TaskUpdatedEvent, slices.Contains(actions, ActionUpdated)
}

// changedTasks reads the tasks of the project that changed after polledUntil.
// Productive.io sorts them by change time, so paging stops at the first task
// that is not newer than the cursor.
func changedTasks(client *Client, projectID string, polledUntil time.Time) ([]map[string]any, error) {
	changed := []map[string]any{}

	for page := 1; page <= maxPollPages; page++ {
		documents, err := client.ListChangedTaskDocuments(projectID, page, pollPageSize)
		if err != nil {
			return nil, err
		}

		for _, document := range documents {
			//
			// A task without a readable change time cannot be placed against
			// the cursor. Treat it as the end of the new tasks, so a single
			// odd resource cannot make every poll report it again.
			//
			changedAt, ok := taskTime(document, "updated_at")
			if !ok || !changedAt.After(polledUntil) {
				return changed, nil
			}

			changed = append(changed, document)
		}

		if len(documents) < pollPageSize {
			return changed, nil
		}
	}

	return changed, nil
}

// newestTaskChange reports the most recent change the project carries, or the
// current time when it has no tasks to read.
func newestTaskChange(client *Client, projectID string) (string, error) {
	documents, err := client.ListChangedTaskDocuments(projectID, 1, 1)
	if err != nil {
		return "", fmt.Errorf("error reading the tasks of project %s: %v", projectID, err)
	}

	if len(documents) == 0 {
		return formatTaskTime(time.Now()), nil
	}

	changedAt, ok := taskTime(documents[0], "updated_at")
	if !ok {
		return formatTaskTime(time.Now()), nil
	}

	return formatTaskTime(changedAt), nil
}

// taskTime reads one of a task's timestamp attributes.
func taskTime(document map[string]any, attribute string) (time.Time, bool) {
	attributes, ok := document["attributes"].(map[string]any)
	if !ok {
		return time.Time{}, false
	}

	value, ok := attributes[attribute].(string)
	if !ok {
		return time.Time{}, false
	}

	return parseTaskTime(value)
}

func parseTaskTime(value string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false
	}

	return parsed, true
}

func formatTaskTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
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
