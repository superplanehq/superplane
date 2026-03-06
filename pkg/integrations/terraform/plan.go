package terraform

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const PollInterval = 10 * time.Second

type Plan struct{}

type PlanSpec struct {
	WorkspaceID string `mapstructure:"workspaceId"`
	Message     string `mapstructure:"message"`
}

type RunStateEntry struct {
	Status    string `json:"status" mapstructure:"status"`
	Timestamp string `json:"timestamp" mapstructure:"timestamp"`
	Message   string `json:"message,omitempty" mapstructure:"message"`
}

type ExecutionMetadata struct {
	RunID         string          `json:"runId" mapstructure:"runId"`
	WorkspaceName string          `json:"workspaceName" mapstructure:"workspaceName"`
	CurrentStatus string          `json:"currentStatus" mapstructure:"currentStatus"`
	StartedAt     string          `json:"startedAt" mapstructure:"startedAt"`
	FinishedAt    string          `json:"finishedAt,omitempty" mapstructure:"finishedAt"`
	RunURL        string          `json:"runUrl,omitempty" mapstructure:"runUrl"`
	StateHistory  []RunStateEntry `json:"stateHistory" mapstructure:"stateHistory"`
	Additions     *int            `json:"additions,omitempty" mapstructure:"additions"`
	Changes       *int            `json:"changes,omitempty" mapstructure:"changes"`
	Destructions  *int            `json:"destructions,omitempty" mapstructure:"destructions"`
	PlanLog       string          `json:"planLog,omitempty" mapstructure:"planLog"`
	PlanJSON      string          `json:"planJson,omitempty" mapstructure:"planJson"`
}

func (c *Plan) Name() string        { return "terraform.plan" }
func (c *Plan) Label() string       { return "Run plan" }
func (c *Plan) Description() string { return "Create a Terraform run and wait for its plan" }
func (c *Plan) Icon() string        { return "file-text" }
func (c *Plan) Color() string       { return "purple" }
func (c *Plan) Documentation() string {
	return `Creates a Terraform run, polls its status, and waits for the plan to complete. It creates a plan only and does not allow applying.`
}

func (c *Plan) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:  "workspaceId",
			Label: "Workspace",
			Type:  configuration.FieldTypeIntegrationResource,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: "workspace"},
			},
			Required: true,
		},
		{
			Name:     "message",
			Label:    "Run Message",
			Type:     configuration.FieldTypeString,
			Required: false,
		},
	}
}

func (c *Plan) Setup(ctx core.SetupContext) error {
	return ensureWorkspaceInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	)
}

func (c *Plan) Execute(ctx core.ExecutionContext) error {
	client, err := getClientFromIntegration(ctx.Integration)
	if err != nil {
		return err
	}

	spec := PlanSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}

	resolvedWsID, err := client.ResolveWorkspaceID(spec.WorkspaceID)
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	msg := fmt.Sprintf("⚙ %s", spec.Message)

	run, err := client.CreateRun(resolvedWsID, msg, true)
	if err != nil {
		return fmt.Errorf("failed to queue run: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	workspaceName := ""
	if run.Workspace != nil {
		workspaceName = run.Workspace.Attributes.Name
	}

	metadata := ExecutionMetadata{
		RunID:         run.ID,
		WorkspaceName: workspaceName,
		CurrentStatus: run.Attributes.Status,
		StartedAt:     now,
		StateHistory: []RunStateEntry{
			{Status: run.Attributes.Status, Timestamp: now, Message: "Run created"},
		},
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, PollInterval)
}

func (c *Plan) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *Plan) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			Description:    "Poll for run status updates",
			UserAccessible: false,
		},
	}
}

func (c *Plan) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case "poll":
		return c.poll(ctx)
	}

	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (c *Plan) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata ExecutionMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	client, err := getClientFromIntegration(ctx.Integration)
	if err != nil {
		return err
	}

	run, err := client.ReadRun(metadata.RunID)
	if err != nil {
		return fmt.Errorf("failed to read run %s: %w", metadata.RunID, err)
	}

	currentStatus := run.Attributes.Status
	now := time.Now().Format(time.RFC3339)

	metadataChanged := false

	if run.Workspace != nil {
		if metadata.WorkspaceName == "" && run.Workspace.Attributes.Name != "" {
			metadata.WorkspaceName = run.Workspace.Attributes.Name
			metadataChanged = true
		}
		if metadata.RunURL == "" && client.BaseURL != "" {
			orgName := ""
			if run.Workspace.Organization != nil {
				orgName = run.Workspace.Organization.Attributes.Name
			}
			if orgName == "" {
				orgName = run.Workspace.Relationships.Organization.Data.ID
			}
			if orgName != "" {
				metadata.RunURL = fmt.Sprintf("%s/app/%s/workspaces/%s/runs/%s", client.BaseURL, orgName, run.Workspace.Attributes.Name, run.ID)
				metadataChanged = true
			}
		}
	}

	if currentStatus != metadata.CurrentStatus {
		metadata.CurrentStatus = currentStatus
		metadata.StateHistory = append(metadata.StateHistory, RunStateEntry{
			Status:    currentStatus,
			Timestamp: now,
		})
		metadataChanged = true
	}

	if isTerminalStatePlanTarget(currentStatus) {
		metadata.FinishedAt = now
		metadataChanged = true
	}

	if metadataChanged {
		if err := ctx.Metadata.Set(metadata); err != nil {
			return fmt.Errorf("failed to update metadata: %w", err)
		}
	}

	if isTerminalStatePlanTarget(currentStatus) {
		return c.emitFinalState(ctx, metadata, run)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, PollInterval)
}

func (c *Plan) emitFinalState(ctx core.ActionContext, metadata ExecutionMetadata, run *RunPayload) error {
	payload := map[string]any{
		"runId":        metadata.RunID,
		"finalStatus":  metadata.CurrentStatus,
		"stateHistory": metadata.StateHistory,
		"runUrl":       metadata.RunURL,
	}

	if run != nil && run.Plan != nil && run.Plan.ID != "" {
		client, err := getClientFromIntegration(ctx.Integration)
		if err == nil {
			plan, err := client.ReadPlan(run.Plan.ID)
			if err == nil && plan != nil {
				additions := plan.Attributes.ResourceAdditions
				changes := plan.Attributes.ResourceChanges
				destructions := plan.Attributes.ResourceDestructions
				payload["additions"] = additions
				payload["changes"] = changes
				payload["destructions"] = destructions
				metadata.Additions = &additions
				metadata.Changes = &changes
				metadata.Destructions = &destructions

				if plan.Attributes.LogReadURL != "" {
					logText, err := client.DownloadLog(plan.Attributes.LogReadURL)
					if err == nil {
						payload["planLog"] = logText
						metadata.PlanLog = logText
					}
				}

				if plan.Links.JSONOutput != "" {
					jsonText, err := client.DownloadJSONOutput(plan.Links.JSONOutput)
					if err == nil {
						payload["planJson"] = jsonText
						metadata.PlanJSON = jsonText
					}
				}

				if err := ctx.Metadata.Set(metadata); err != nil {
					return fmt.Errorf("failed to save final metadata: %w", err)
				}
			}
		}
	}

	channel := "passed"

	if isFailedState(metadata.CurrentStatus) {
		channel = "failed"
	}

	return ctx.ExecutionState.Emit(channel, "terraform.run.planned", []any{payload})
}

func (c *Plan) HandleWebhook(ctx core.WebhookRequestContext) (int, error) { return http.StatusOK, nil }

func (c *Plan) Cancel(ctx core.ExecutionContext) error {
	var metadata ExecutionMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return err
	}
	if metadata.RunID == "" {
		return nil
	}

	client, err := getClientFromIntegration(ctx.Integration)
	if err != nil {
		return err
	}

	err = client.CancelRun(metadata.RunID, "Cancelled via SuperPlane Workflow")
	if err != nil {
		return fmt.Errorf("failed to cancel terraform run: %w", err)
	}
	return nil
}

func (c *Plan) Cleanup(ctx core.SetupContext) error { return nil }

func (c *Plan) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: "passed", Label: "Passed", Description: "Run completed successfully or paused at planned"},
		{Name: "failed", Label: "Failed", Description: "Run failed, errored, or was canceled"},
	}
}

func (c *Plan) DefaultOutputChannel() core.OutputChannel {
	return core.OutputChannel{Name: "passed", Label: "Passed"}
}

func isFailedState(status string) bool {
	switch status {
	case "discarded", "errored", "canceled", "policy_soft_failed", "force_canceled":
		return true
	}
	return false
}

func (c *Plan) ExampleOutput() map[string]any {
	return map[string]any{
		"runId":       "run-xxxxx",
		"finalStatus": "planned",
	}
}

func isTerminalStatePlanTarget(status string) bool {
	switch status {
	case "planned", "planned_and_finished", "discarded", "errored", "canceled", "policy_soft_failed", "force_canceled", "applied":
		return true
	}
	return false
}
