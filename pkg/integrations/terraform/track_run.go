package terraform

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type TrackRun struct{}

type TrackRunConfiguration struct {
	RunID        string `json:"runId" mapstructure:"runId"`
	PollInterval *int   `json:"pollInterval,omitempty" mapstructure:"pollInterval,omitempty"`
}

type TrackRunMetadata struct {
	RunID         string          `json:"runId"`
	RunURL        string          `json:"runUrl,omitempty"`
	WorkspaceName string          `json:"workspaceName,omitempty"`
	CurrentStatus string          `json:"currentStatus,omitempty"`
	StateHistory  []RunStateEntry `json:"stateHistory,omitempty"`
	StartedAt     string          `json:"startedAt,omitempty"`
	FinishedAt    string          `json:"finishedAt,omitempty"`
	PollInterval  int             `json:"pollInterval,omitempty"`
}

type RunStateEntry struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message,omitempty"`
}

func (c *TrackRun) Name() string {
	return "terraform.trackRun"
}

func (c *TrackRun) Label() string {
	return "Track Run"
}

func (c *TrackRun) Description() string {
	return "Track a Terraform run through its lifecycle states (planning, applying, etc.)"
}

func (c *TrackRun) Documentation() string {
	return `The Track Run component monitors a Terraform run and tracks its progression through states.

## States Tracked

- **Pending** → **Planning** → **Planned** → **Applying** → **Applied**
- Or: **Planned** → **Discarded** / **Errored** / **Canceled**

## Output Channels

- **Completed**: Run finished successfully (applied or planned_and_finished)
- **Failed**: Run errored, was canceled, or was discarded
- **Needs Attention**: Run requires manual confirmation or policy override

## Use Cases

- Track infrastructure changes through their lifecycle
- Get notified when runs complete or fail
- Chain actions based on run completion`
}

func (c *TrackRun) Icon() string {
	return "terraform"
}

func (c *TrackRun) Color() string {
	return "purple"
}

func (c *TrackRun) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: "completed", Label: "Completed", Description: "Run completed successfully"},
		{Name: "failed", Label: "Failed", Description: "Run failed, errored, or was canceled"},
		{Name: "needsAttention", Label: "Needs Attention", Description: "Run requires confirmation or policy override"},
	}
}

func (c *TrackRun) DefaultOutputChannel() core.OutputChannel {
	return core.OutputChannel{Name: "completed", Label: "Completed"}
}

func (c *TrackRun) ExampleOutput() map[string]any {
	return map[string]any{
		"runId":       "run-xxxxx",
		"finalStatus": "applied",
		"stateHistory": []map[string]string{
			{"status": "pending", "timestamp": "2024-01-01T12:00:00Z"},
			{"status": "planning", "timestamp": "2024-01-01T12:00:05Z"},
			{"status": "planned", "timestamp": "2024-01-01T12:01:00Z"},
			{"status": "applying", "timestamp": "2024-01-01T12:01:05Z"},
			{"status": "applied", "timestamp": "2024-01-01T12:02:00Z"},
		},
	}
}

func (c *TrackRun) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "runId",
			Label:       "Run ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The Terraform run ID to track (e.g., run-xxxxx). Use {{trigger.runId}} from trigger.",
		},
		{
			Name:        "pollInterval",
			Label:       "Poll Interval (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     "10",
			Description: "How often to check for status updates (default: 10 seconds)",
		},
	}
}

func (c *TrackRun) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *TrackRun) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *TrackRun) Execute(ctx core.ExecutionContext) error {
	var config TrackRunConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.RunID == "" {
		return fmt.Errorf("runId is required but got empty string")
	}

	if len(config.RunID) < 4 || config.RunID[:4] != "run-" {
		return fmt.Errorf("invalid run ID format: %q (expected format: run-xxxxxxxx)", config.RunID)
	}

	client, err := getClientFromIntegration(ctx.Integration)
	if err != nil {
		return err
	}

	ctx.Logger.Infof("Reading run details for: %s", config.RunID)

	run, err := client.ReadRun(context.Background(), config.RunID)
	if err != nil {
		return fmt.Errorf("failed to read run %s: %w", config.RunID, err)
	}

	var workspaceName string
	if run.Workspace != nil {
		workspaceName = run.Workspace.Attributes.Name
	}

	pollInterval := 10
	if config.PollInterval != nil && *config.PollInterval > 0 {
		pollInterval = *config.PollInterval
	}

	now := time.Now().Format(time.RFC3339)
	metadata := TrackRunMetadata{
		RunID:         config.RunID,
		WorkspaceName: workspaceName,
		CurrentStatus: run.Attributes.Status,
		StartedAt:     now,
		PollInterval:  pollInterval,
		StateHistory: []RunStateEntry{
			{Status: run.Attributes.Status, Timestamp: now, Message: "Started tracking"},
		},
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	ctx.Logger.Infof("Tracking run %s - current status: %s", config.RunID, run.Attributes.Status)

	if isTerminalState(run.Attributes.Status) {
		return c.emitFinalState(ctx, config.RunID, run.Attributes.Status, metadata)
	}

	if needsAttention(run.Attributes.Status, run.Workspace) {
		ctx.Logger.Infof("Run %s needs attention: %s", config.RunID, run.Attributes.Status)
		return c.emitNeedsAttentionFromExecution(ctx, config.RunID, run.Attributes.Status, metadata)
	}

	if err := ctx.Requests.ScheduleActionCall("poll", map[string]any{}, time.Duration(pollInterval)*time.Second); err != nil {
		return fmt.Errorf("failed to schedule poll: %w", err)
	}

	return nil
}

func (c *TrackRun) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *TrackRun) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			Description:    "Poll for run status updates",
			UserAccessible: false,
		},
	}
}

func (c *TrackRun) HandleAction(ctx core.ActionContext) error {
	if ctx.Name != "poll" {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}

	return c.poll(ctx)
}

func (c *TrackRun) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata TrackRunMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	runID := metadata.RunID
	if runID == "" {
		return fmt.Errorf("runId not found in metadata - initial execution may have failed")
	}

	client, err := getClientFromIntegration(ctx.Integration)
	if err != nil {
		return err
	}

	run, err := client.ReadRun(context.Background(), runID)
	if err != nil {
		return fmt.Errorf("failed to read run %s: %w", runID, err)
	}

	currentStatus := run.Attributes.Status
	now := time.Now().Format(time.RFC3339)

	if currentStatus != metadata.CurrentStatus {
		ctx.Logger.Infof("Run %s status changed: %s → %s", runID, metadata.CurrentStatus, currentStatus)

		metadata.CurrentStatus = currentStatus
		metadata.StateHistory = append(metadata.StateHistory, RunStateEntry{
			Status:    currentStatus,
			Timestamp: now,
		})

		if err := ctx.Metadata.Set(metadata); err != nil {
			return fmt.Errorf("failed to update metadata: %w", err)
		}
	}

	if isTerminalState(currentStatus) {
		ctx.Logger.Infof("Run %s reached terminal state: %s", runID, currentStatus)
		metadata.FinishedAt = now
		if err := ctx.Metadata.Set(metadata); err != nil {
			return fmt.Errorf("failed to update metadata: %w", err)
		}
		return c.emitFinalStateFromAction(ctx, runID, currentStatus, metadata)
	}

	if needsAttention(currentStatus, run.Workspace) {
		ctx.Logger.Infof("Run %s needs attention: %s", runID, currentStatus)
		return c.emitNeedsAttention(ctx, runID, currentStatus, metadata)
	}

	ctx.Logger.Infof("Run %s still in progress: %s, scheduling next poll", runID, currentStatus)

	pollInterval := metadata.PollInterval
	if pollInterval <= 0 {
		pollInterval = 10
	}

	if err := ctx.Requests.ScheduleActionCall("poll", map[string]any{}, time.Duration(pollInterval)*time.Second); err != nil {
		return fmt.Errorf("failed to schedule next poll: %w", err)
	}

	return nil
}

func (c *TrackRun) emitFinalState(ctx core.ExecutionContext, runID, status string, metadata TrackRunMetadata) error {
	return emitFinalStateToChannel(ctx.ExecutionState, runID, status, metadata)
}

func (c *TrackRun) emitFinalStateFromAction(ctx core.ActionContext, runID, status string, metadata TrackRunMetadata) error {
	return emitFinalStateToChannel(ctx.ExecutionState, runID, status, metadata)
}

func (c *TrackRun) emitNeedsAttention(ctx core.ActionContext, runID, status string, metadata TrackRunMetadata) error {
	return emitNeedsAttentionToChannel(ctx.ExecutionState, runID, status, metadata)
}

func (c *TrackRun) emitNeedsAttentionFromExecution(ctx core.ExecutionContext, runID, status string, metadata TrackRunMetadata) error {
	return emitNeedsAttentionToChannel(ctx.ExecutionState, runID, status, metadata)
}

func emitFinalStateToChannel(state core.ExecutionStateContext, runID, status string, metadata TrackRunMetadata) error {
	payload := map[string]any{
		"runId":        runID,
		"finalStatus":  status,
		"stateHistory": metadata.StateHistory,
		"runUrl":       metadata.RunURL,
	}

	channel := "completed"
	eventType := "terraform.run.completed"

	if isFailedState(status) {
		channel = "failed"
		eventType = "terraform.run.failed"
	}

	return state.Emit(channel, eventType, []any{payload})
}

func emitNeedsAttentionToChannel(state core.ExecutionStateContext, runID, status string, metadata TrackRunMetadata) error {
	payload := map[string]any{
		"runId":        runID,
		"status":       status,
		"stateHistory": metadata.StateHistory,
		"runUrl":       metadata.RunURL,
		"message":      getAttentionMessage(status),
	}

	return state.Emit("needsAttention", "terraform.run.needsAttention", []any{payload})
}

func (c *TrackRun) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *TrackRun) Cleanup(ctx core.SetupContext) error {
	return nil
}

func isTerminalState(status string) bool {
	switch status {
	case "applied", "planned_and_finished",
		"discarded", "errored", "canceled", "policy_soft_failed", "force_canceled":
		return true
	}
	return false
}

func isFailedState(status string) bool {
	switch status {
	case "discarded", "errored", "canceled", "policy_soft_failed", "force_canceled":
		return true
	}
	return false
}

func needsAttention(status string, workspace *WorkspacePayload) bool {
	autoApply := workspace != nil && workspace.Attributes.AutoApply

	switch status {
	case "planned", "cost_estimated", "policy_checked":
		return !autoApply
	case "policy_override", "planned_and_saved":
		return true
	}
	return false
}

func getAttentionMessage(status string) string {
	switch status {
	case "planned":
		return "Run is planned and waiting for confirmation"
	case "policy_override":
		return "Run requires policy override"
	case "cost_estimated":
		return "Cost estimation complete, awaiting confirmation"
	default:
		return "Run requires attention"
	}
}
