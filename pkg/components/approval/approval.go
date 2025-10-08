package approval

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
)

const ComponentName = "approval"

type Config struct {
	Count int `json:"count"`
}

type ApprovalMetadata struct {
	RequiredCount int              `mapstructure:"required_count"`
	Approvals     []ApprovalRecord `mapstructure:"approvals"`
}

type ApprovalRecord struct {
	ApprovedAt string `mapstructure:"approved_at"`
	Comment    string `mapstructure:"comment"`
}

type Approval struct{}

func (a *Approval) Name() string {
	return ComponentName
}

func (a *Approval) Description() string {
	return "Wait for approvals before continuing execution. Execution moves to waiting state until required approvals are received."
}

func (a *Approval) Outputs(configuration any) []string {
	return []string{components.DefaultBranchName}
}

func (a *Approval) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{
		{
			Name:        "count",
			Type:        "number",
			Description: "Number of approvals required before execution continues",
			Required:    true,
		},
	}
}

func (a *Approval) Execute(ctx components.ExecutionContext) error {
	config := Config{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return err
	}

	if config.Count < 1 {
		return fmt.Errorf("count must be at least 1")
	}

	// Initialize approval state
	metadata := ApprovalMetadata{
		RequiredCount: config.Count,
		Approvals:     []ApprovalRecord{},
	}

	ctx.Metadata.Set("required_count", metadata.RequiredCount)
	ctx.Metadata.Set("approvals", metadata.Approvals)

	// Move to waiting state
	return ctx.State.Wait()
}

func (a *Approval) Actions() []components.Action {
	return []components.Action{
		{
			Name:        "approve",
			Description: "Approve this execution",
			Parameters: []components.ConfigurationField{
				{
					Name:        "comment",
					Type:        "string",
					Description: "Optional comment for the approval",
					Required:    false,
				},
			},
		},
		{
			Name:        "reject",
			Description: "Reject this execution",
			Parameters: []components.ConfigurationField{
				{
					Name:        "reason",
					Type:        "string",
					Description: "Reason for rejection",
					Required:    true,
				},
			},
		},
	}
}

func (a *Approval) HandleAction(ctx components.ActionContext) error {
	switch ctx.Name {
	case "approve":
		return a.handleApprove(ctx)
	case "reject":
		return a.handleReject(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (a *Approval) handleApprove(ctx components.ActionContext) error {
	// Parse metadata into structured format
	var metadata ApprovalMetadata
	err := mapstructure.Decode(ctx.Metadata.GetAll(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Add new approval
	approval := ApprovalRecord{
		ApprovedAt: time.Now().Format(time.RFC3339),
	}

	if comment, ok := ctx.Parameters["comment"]; ok && comment != nil {
		if commentStr, ok := comment.(string); ok {
			approval.Comment = commentStr
		}
	}

	metadata.Approvals = append(metadata.Approvals, approval)
	ctx.Metadata.Set("approvals", metadata.Approvals)

	// Check if we have enough approvals
	if len(metadata.Approvals) >= metadata.RequiredCount {
		// Complete the execution - pass input data through
		return ctx.State.Finish(map[string][]any{
			components.DefaultBranchName: {ctx.Metadata.GetAll()},
		})
	}

	// Still waiting for more approvals
	return nil
}

func (a *Approval) handleReject(ctx components.ActionContext) error {
	reason, ok := ctx.Parameters["reason"]
	if !ok || reason == nil {
		return fmt.Errorf("reason is required for rejection")
	}

	reasonStr, ok := reason.(string)
	if !ok {
		return fmt.Errorf("reason must be a string")
	}

	return ctx.State.Fail(fmt.Sprintf("Rejected: %s", reasonStr))
}
