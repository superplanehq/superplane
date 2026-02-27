package terraform

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-tfe"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ChannelApproved = "approved"
	ChannelRejected = "rejected"
	ChannelTimeout  = "timeout"

	ActionApprove = "approve"
	ActionReject  = "reject"
	ActionTimeout = "timeout"
)

type WaitForApproval struct{}

type WaitForApprovalConfiguration struct {
	RunID          string `json:"runId" mapstructure:"runId"`
	Timeout        *int   `json:"timeout,omitempty" mapstructure:"timeout,omitempty"`
	ApproveLabel   string `json:"approveLabel,omitempty" mapstructure:"approveLabel,omitempty"`
	RejectLabel    string `json:"rejectLabel,omitempty" mapstructure:"rejectLabel,omitempty"`
	ApplyOnApprove bool   `json:"applyOnApprove" mapstructure:"applyOnApprove"`
}

type WaitForApprovalMetadata struct {
	RunID         string  `json:"runId"`
	RunURL        string  `json:"runUrl,omitempty"`
	RunStatus     string  `json:"runStatus,omitempty"`
	WorkspaceName string  `json:"workspaceName,omitempty"`
	Decision      *string `json:"decision,omitempty"`
	DecidedAt     *string `json:"decidedAt,omitempty"`
	DecidedBy     *string `json:"decidedBy,omitempty"`
	AppliedToTFC  bool    `json:"appliedToTFC"`
}

func (c *WaitForApproval) Name() string {
	return "terraform.waitForApproval"
}

func (c *WaitForApproval) Label() string {
	return "Wait for Approval"
}

func (c *WaitForApproval) Description() string {
	return "Display approve/reject buttons and optionally apply the run in Terraform Cloud"
}

func (c *WaitForApproval) Documentation() string {
	return `The Wait for Approval component pauses the workflow and displays Approve/Reject buttons in the SuperPlane UI.

## Use Cases

- **Manual approval gates**: Require human approval before applying Terraform changes
- **Review workflow**: Allow team members to review planned changes before applying
- **Controlled deployments**: Implement approval workflows for infrastructure changes

## Configuration

- **Run ID**: The Terraform run ID to approve (required, usually from trigger data)
- **Apply on Approve**: If enabled, automatically applies the run in Terraform Cloud when approved
- **Timeout**: Maximum time to wait in seconds (optional, leave empty to wait indefinitely)
- **Approve Label**: Custom label for the approve button (default: "Approve & Apply")
- **Reject Label**: Custom label for the reject button (default: "Reject")

## Output Channels

- **Approved**: Emits when the user clicks Approve
- **Rejected**: Emits when the user clicks Reject
- **Timeout**: Emits when no decision is made within the configured timeout

## Behavior

- The workflow pauses until a decision is made or timeout occurs
- If Apply on Approve is enabled, clicking Approve will also apply the run in Terraform Cloud
- Only the first decision is processed; subsequent clicks are ignored`
}

func (c *WaitForApproval) Icon() string {
	return "terraform"
}

func (c *WaitForApproval) Color() string {
	return "orange"
}

func (c *WaitForApproval) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: ChannelApproved, Label: "Approved", Description: "Emits when the run is approved"},
		{Name: ChannelRejected, Label: "Rejected", Description: "Emits when the run is rejected"},
		{Name: ChannelTimeout, Label: "Timeout", Description: "Emits when timeout is reached"},
	}
}

func (c *WaitForApproval) DefaultOutputChannel() core.OutputChannel {
	return core.OutputChannel{Name: ChannelApproved, Label: "Approved"}
}

func (c *WaitForApproval) ExampleOutput() map[string]any {
	return map[string]any{
		"runId":        "run-xxxxx",
		"decision":     "approved",
		"decidedAt":    "2024-01-01T12:00:00Z",
		"appliedToTFC": true,
	}
}

func (c *WaitForApproval) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "runId",
			Label:       "Run ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The Terraform run ID (e.g., run-xxxxx). Use {{trigger.runId}} to get from trigger.",
		},
		{
			Name:        "applyOnApprove",
			Label:       "Apply on Approve",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "If enabled, automatically applies the run in Terraform Cloud when approved",
		},
		{
			Name:        "timeout",
			Label:       "Timeout (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     "3600",
			Description: "Maximum time to wait in seconds (leave empty to wait indefinitely)",
		},
		{
			Name:        "approveLabel",
			Label:       "Approve Button Label",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "Approve & Apply",
			Description: "Custom label for the approve button",
		},
		{
			Name:        "rejectLabel",
			Label:       "Reject Button Label",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "Reject",
			Description: "Custom label for the reject button",
		},
	}
}

func (c *WaitForApproval) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *WaitForApproval) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *WaitForApproval) Execute(ctx core.ExecutionContext) error {
	var config WaitForApprovalConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.RunID == "" {
		return errors.New("runId is required")
	}

	client, err := getClientFromIntegration(ctx.Integration)
	if err != nil {
		return err
	}

	run, err := client.TFE.Runs.ReadWithOptions(context.Background(), config.RunID, &tfe.RunReadOptions{
		Include: []tfe.RunIncludeOpt{tfe.RunWorkspace},
	})
	if err != nil {
		return fmt.Errorf("failed to read run: %w", err)
	}

	var workspaceName string
	if run.Workspace != nil {
		workspaceName = run.Workspace.Name
	}

	metadata := WaitForApprovalMetadata{
		RunID:         config.RunID,
		RunStatus:     string(run.Status),
		WorkspaceName: workspaceName,
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	if config.Timeout != nil && *config.Timeout > 0 {
		timeout := time.Duration(*config.Timeout) * time.Second
		if err := ctx.Requests.ScheduleActionCall(ActionTimeout, map[string]any{}, timeout); err != nil {
			return fmt.Errorf("failed to schedule timeout: %w", err)
		}
	}

	return nil
}

func (c *WaitForApproval) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *WaitForApproval) Actions() []core.Action {
	return []core.Action{
		{
			Name:           ActionApprove,
			Description:    "Approve the Terraform run",
			UserAccessible: true,
			Parameters: []configuration.Field{
				{
					Name:        "comment",
					Label:       "Comment",
					Type:        configuration.FieldTypeString,
					Required:    false,
					Description: "Optional comment to add when approving",
				},
			},
		},
		{
			Name:           ActionReject,
			Description:    "Reject and discard the Terraform run",
			UserAccessible: true,
			Parameters: []configuration.Field{
				{
					Name:        "comment",
					Label:       "Comment",
					Type:        configuration.FieldTypeString,
					Required:    false,
					Description: "Optional comment to add when rejecting",
				},
			},
		},
		{
			Name:           ActionTimeout,
			Description:    "Handle timeout",
			UserAccessible: false,
		},
	}
}

func (c *WaitForApproval) HandleAction(ctx core.ActionContext) error {
	switch ctx.Name {
	case ActionApprove:
		return c.handleApprove(ctx)
	case ActionReject:
		return c.handleReject(ctx)
	case ActionTimeout:
		return c.handleTimeout(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (c *WaitForApproval) handleApprove(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var config WaitForApprovalConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	var metadata WaitForApprovalMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	decision := "approved"
	metadata.Decision = &decision
	metadata.DecidedAt = &now

	// Apply the run in Terraform Cloud if configured
	if config.ApplyOnApprove {
		client, err := getClientFromIntegration(ctx.Integration)
		if err != nil {
			return err
		}

		comment, _ := ctx.Parameters["comment"].(string)
		err = client.TFE.Runs.Apply(context.Background(), config.RunID, tfe.RunApplyOptions{
			Comment: &comment,
		})
		if err != nil {
			return fmt.Errorf("failed to apply run in Terraform Cloud: %w", err)
		}
		metadata.AppliedToTFC = true
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	payload := map[string]any{
		"runId":        config.RunID,
		"decision":     "approved",
		"decidedAt":    now,
		"appliedToTFC": metadata.AppliedToTFC,
	}

	return ctx.ExecutionState.Emit(
		ChannelApproved,
		"terraform.run.approved",
		[]any{payload},
	)
}

func (c *WaitForApproval) handleReject(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var config WaitForApprovalConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	var metadata WaitForApprovalMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	decision := "rejected"
	metadata.Decision = &decision
	metadata.DecidedAt = &now

	// Discard the run in Terraform Cloud
	client, err := getClientFromIntegration(ctx.Integration)
	if err != nil {
		return err
	}

	comment, _ := ctx.Parameters["comment"].(string)
	err = client.TFE.Runs.Discard(context.Background(), config.RunID, tfe.RunDiscardOptions{
		Comment: &comment,
	})
	if err != nil {
		return fmt.Errorf("failed to discard run in Terraform Cloud: %w", err)
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	payload := map[string]any{
		"runId":     config.RunID,
		"decision":  "rejected",
		"decidedAt": now,
	}

	return ctx.ExecutionState.Emit(
		ChannelRejected,
		"terraform.run.rejected",
		[]any{payload},
	)
}

func (c *WaitForApproval) handleTimeout(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var config WaitForApprovalConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	payload := map[string]any{
		"runId":     config.RunID,
		"timeoutAt": time.Now().Format(time.RFC3339),
	}

	return ctx.ExecutionState.Emit(
		ChannelTimeout,
		"terraform.approval.timeout",
		[]any{payload},
	)
}

func (c *WaitForApproval) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *WaitForApproval) Cleanup(ctx core.SetupContext) error {
	return nil
}
