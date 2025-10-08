package approval

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
)

const ComponentName = "approval"

/*
 * Configuration for the component.
 * Filled when the component is added to a blueprint/workflow.
 */
type Config struct {
	Count int `json:"count"`
}

/*
 * Metadata for the component.
 */
type Metadata struct {
	RequiredCount int              `mapstructure:"required_count" json:"required_count"`
	Approvals     []ApprovalRecord `mapstructure:"approvals" json:"approvals"`
}

func NewMetadata(count int) Metadata {
	return Metadata{
		RequiredCount: count,
		Approvals:     []ApprovalRecord{},
	}
}

func (m *Metadata) addApproval(parameters map[string]any) {
	record := ApprovalRecord{
		ApprovedAt: time.Now().Format(time.RFC3339),
	}

	c, ok := parameters["comment"]
	if !ok || c == nil {
		m.Approvals = append(m.Approvals, record)
		return
	}

	comment, ok := c.(string)
	if !ok {
		m.Approvals = append(m.Approvals, record)
		return
	}

	record.Comment = comment
	m.Approvals = append(m.Approvals, record)
}

type ApprovalRecord struct {
	ApprovedAt string `mapstructure:"approved_at" json:"approved_at"`
	Comment    string `mapstructure:"comment" json:"comment"`
}

type Approval struct{}

func (a *Approval) Name() string {
	return ComponentName
}

func (a *Approval) Description() string {
	return "Wait for approvals before continuing execution. Execution moves to waiting state until required approvals are received."
}

func (a *Approval) OutputBranches(configuration any) []string {
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

	//
	// TODO: this should be validated before it even gets here.
	//
	if config.Count < 1 {
		return fmt.Errorf("count must be at least 1")
	}

	//
	// Initialize metadata for the execution,
	// and move it to the waiting state.
	//
	ctx.MetadataContext.Set(NewMetadata(config.Count))
	return ctx.ExecutionStateContext.Wait()
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
	//
	// Add new approval to metadata
	//
	var metadata Metadata
	err := mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	metadata.addApproval(ctx.Parameters)
	ctx.MetadataContext.Set(metadata)

	//
	// If the number of approvals is still below the required amount,
	// do not finish the execution yet.
	//
	if len(metadata.Approvals) < metadata.RequiredCount {
		return nil
	}

	//
	// Required amount of approvals reached - finish the execution.
	//
	return ctx.ExecutionStateContext.Finish(map[string][]any{
		components.DefaultBranchName: {metadata},
	})
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

	return ctx.ExecutionStateContext.Fail(fmt.Sprintf("Rejected: %s", reasonStr))
}
