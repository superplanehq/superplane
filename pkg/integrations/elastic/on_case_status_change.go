package elastic

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	onCaseStatusChangePollAction   = "poll"
	onCaseStatusChangePollInterval = 1 * time.Minute
)

type OnCaseStatusChange struct{}

type OnCaseStatusChangeConfiguration struct {
	Statuses []string `json:"statuses" mapstructure:"statuses"`
}

type OnCaseStatusChangeMetadata struct {
	LastPollTime string `json:"lastPollTime,omitempty" mapstructure:"lastPollTime"`
}

var caseStatusOptions = []configuration.FieldOption{
	{Label: "Open", Value: "open"},
	{Label: "In Progress", Value: "in-progress"},
	{Label: "Closed", Value: "closed"},
}

func (t *OnCaseStatusChange) Name() string  { return "elastic.onCaseStatusChange" }
func (t *OnCaseStatusChange) Label() string { return "When Case Status Changes" }
func (t *OnCaseStatusChange) Description() string {
	return "React when an Elastic Security case status changes"
}
func (t *OnCaseStatusChange) Icon() string  { return "alert-circle" }
func (t *OnCaseStatusChange) Color() string { return "gray" }

func (t *OnCaseStatusChange) Documentation() string {
	return `The When Case Status Changes trigger starts a workflow execution when a Kibana Security case is updated.

## How it works

SuperPlane polls the Kibana Cases API every minute for cases updated since the last poll. Each updated case matching the configured status filter triggers a workflow execution.

## Configuration

- **Statuses** *(optional)*: Only fire when a case transitions to one of these statuses. Leave empty to fire for any case update.

## Event Data

The trigger emits the full case details including id, title, status, severity, version, and timestamps.`
}

func (t *OnCaseStatusChange) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "statuses",
			Label:       "Statuses",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Only fire for cases with one of these statuses. Leave empty to fire for all status values.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: caseStatusOptions,
				},
			},
		},
	}
}

func (t *OnCaseStatusChange) Setup(ctx core.TriggerContext) error {
	if ctx.Metadata != nil {
		var meta OnCaseStatusChangeMetadata
		if err := mapstructure.Decode(ctx.Metadata.Get(), &meta); err != nil || meta.LastPollTime == "" {
			meta.LastPollTime = time.Now().UTC().Format(time.RFC3339Nano)
		}
		if err := ctx.Metadata.Set(meta); err != nil {
			return fmt.Errorf("failed to save metadata: %w", err)
		}
	}

	if ctx.Requests != nil {
		if err := ctx.Requests.ScheduleActionCall(onCaseStatusChangePollAction, map[string]any{}, onCaseStatusChangePollInterval); err != nil {
			return fmt.Errorf("failed to schedule poll: %w", err)
		}
	}

	return nil
}

func (t *OnCaseStatusChange) Actions() []core.Action {
	return []core.Action{
		{
			Name:           onCaseStatusChangePollAction,
			Description:    "Poll Kibana Cases API for status changes",
			UserAccessible: false,
		},
	}
}

func (t *OnCaseStatusChange) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	if ctx.Name == onCaseStatusChangePollAction {
		return nil, t.poll(ctx)
	}
	return nil, nil
}

func (t *OnCaseStatusChange) poll(ctx core.TriggerActionContext) error {
	var config OnCaseStatusChangeConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	var meta OnCaseStatusChangeMetadata
	if ctx.Metadata != nil {
		if err := mapstructure.Decode(ctx.Metadata.Get(), &meta); err != nil || meta.LastPollTime == "" {
			meta.LastPollTime = time.Now().UTC().Format(time.RFC3339Nano)
		}
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("elastic onCaseStatusChange: failed to create client: %v", err)
		}
		return ctx.Requests.ScheduleActionCall(onCaseStatusChangePollAction, map[string]any{}, onCaseStatusChangePollInterval)
	}

	cases, err := client.ListCasesUpdatedSince(meta.LastPollTime, config.Statuses)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("elastic onCaseStatusChange: failed to list cases: %v", err)
		}
		return ctx.Requests.ScheduleActionCall(onCaseStatusChangePollAction, map[string]any{}, onCaseStatusChangePollInterval)
	}

	newLastPollTime := meta.LastPollTime
	for _, c := range cases {
		if len(config.Statuses) > 0 && !slices.Contains(config.Statuses, strings.ToLower(c.Status)) {
			continue
		}

		payload := map[string]any{
			"id":          c.ID,
			"title":       c.Title,
			"status":      c.Status,
			"severity":    c.Severity,
			"version":     c.Version,
			"tags":        c.Tags,
			"description": c.Description,
			"createdAt":   c.CreatedAt,
			"updatedAt":   c.UpdatedAt,
		}
		if err := ctx.Events.Emit("elastic.case.status.changed", payload); err != nil {
			return fmt.Errorf("failed to emit event: %w", err)
		}

		if c.UpdatedAt > newLastPollTime {
			newLastPollTime = c.UpdatedAt
		}
	}

	if newLastPollTime != meta.LastPollTime && ctx.Metadata != nil {
		meta.LastPollTime = newLastPollTime
		if err := ctx.Metadata.Set(meta); err != nil {
			return fmt.Errorf("failed to update metadata: %w", err)
		}
	} else if ctx.Metadata != nil && meta.LastPollTime == "" {
		meta.LastPollTime = time.Now().UTC().Format(time.RFC3339Nano)
		if err := ctx.Metadata.Set(meta); err != nil {
			return fmt.Errorf("failed to update metadata: %w", err)
		}
	}

	return ctx.Requests.ScheduleActionCall(onCaseStatusChangePollAction, map[string]any{}, onCaseStatusChangePollInterval)
}

func (t *OnCaseStatusChange) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (t *OnCaseStatusChange) Cleanup(_ core.TriggerContext) error {
	return nil
}
