package approval

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
)

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
	return "approval"
}

func (a *Approval) Label() string {
	return "Approval"
}

func (a *Approval) Description() string {
	return "Collect approvals on events"
}

func (a *Approval) OutputBranches(configuration any) []components.OutputBranch {
	return []components.OutputBranch{components.DefaultOutputBranch}
}

func (a *Approval) Configuration() []components.ConfigurationField {
	min := 1
	return []components.ConfigurationField{
		{
			Name:        "count",
			Label:       "Number of approvals",
			Type:        components.FieldTypeNumber,
			Description: "Number of approvals required before execution continues",
			Required:    true,
			Min:         &min,
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
	// Initialize metadata for the execution.
	//
	ctx.MetadataContext.Set(NewMetadata(config.Count))
	return nil
}

func (a *Approval) Actions() []components.Action {
	return []components.Action{
		{
			Name:        "approve",
			Description: "Approve this execution",
			Parameters: []components.ConfigurationField{
				{
					Name:        "comment",
					Label:       "Comment",
					Type:        components.FieldTypeString,
					Description: "Leave a comment before approving",
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
					Label:       "Reason",
					Type:        components.FieldTypeString,
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
	return ctx.ExecutionStateContext.Pass(map[string][]any{
		components.DefaultOutputBranch.Name: {metadata},
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

	return ctx.ExecutionStateContext.Fail(reasonStr, "")
}
