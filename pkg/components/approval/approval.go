package approval

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterComponent("approval", &Approval{})
}

/*
 * Configuration for the component.
 * Filled when the component is added to a blueprint/workflow.
 */
type Config struct {
	ApprovalRequirements []ApprovalRequirement `json:"approvals" mapstructure:"approvals"`
}

type ApprovalRequirement struct {
	Type       string                          `json:"type"`
	User       string                          `json:"user"`
	Role       string                          `json:"role"`
	Group      string                          `json:"group"`
	Parameters []components.ConfigurationField `json:"parameters"`
}

/*
 * Metadata for the component.
 */
type Metadata struct {
	Approvals []ApprovalRecord `json:"approvals"`
}

func NewMetadata() Metadata {
	return Metadata{
		Approvals: []ApprovalRecord{},
	}
}

func (m *Metadata) addApproval(requirementIndex int, approvedBy string, parameters map[string]any) {
	record := ApprovalRecord{
		RequirementIndex: requirementIndex,
		ApprovedAt:       time.Now().Format(time.RFC3339),
		ApprovedBy:       approvedBy,
	}

	// Extract comment if provided
	if c, ok := parameters["comment"]; ok && c != nil {
		if comment, ok := c.(string); ok {
			record.Comment = comment
		}
	}

	// Extract data if provided
	if d, ok := parameters["data"]; ok && d != nil {
		if data, ok := d.(map[string]any); ok {
			record.Data = data
		}
	}

	m.Approvals = append(m.Approvals, record)
}

type ApprovalRecord struct {
	RequirementIndex int            `json:"requirementIndex"`
	ApprovedAt       string         `json:"approvedAt"`
	ApprovedBy       string         `json:"approvedBy"`
	Comment          string         `json:"comment"`
	Data             map[string]any `json:"data"`
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

func (a *Approval) Icon() string {
	return "check"
}

func (a *Approval) Color() string {
	return "green"
}

func (a *Approval) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (a *Approval) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{
		{
			Name:  "approvals",
			Label: "Approvals",
			Type:  components.FieldTypeList,
			TypeOptions: &components.TypeOptions{
				List: &components.ListTypeOptions{
					ItemDefinition: &components.ListItemDefinition{
						Type: components.FieldTypeObject,
						Schema: []components.ConfigurationField{
							{
								Name:     "type",
								Label:    "Type",
								Type:     components.FieldTypeSelect,
								Required: true,
								TypeOptions: &components.TypeOptions{
									Select: &components.SelectTypeOptions{
										Options: []components.FieldOption{
											{Value: "user", Label: "User"},
											{Value: "role", Label: "Role"},
											{Value: "group", Label: "Group"},
										},
									},
								},
							},
							{
								Name:  "user",
								Label: "User",
								Type:  components.FieldTypeUser,
								VisibilityConditions: []components.VisibilityCondition{
									{
										Field:  "type",
										Values: []string{"user"},
									},
								},
							},
							{
								Name:  "role",
								Label: "Role",
								Type:  components.FieldTypeRole,
								VisibilityConditions: []components.VisibilityCondition{
									{
										Field:  "type",
										Values: []string{"role"},
									},
								},
							},
							{
								Name:  "group",
								Label: "Group",
								Type:  components.FieldTypeGroup,
								VisibilityConditions: []components.VisibilityCondition{
									{
										Field:  "type",
										Values: []string{"group"},
									},
								},
							},
						},
					},
				},
			},
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
	// TODO: we should validate the configuration
	// That could be done in a Setup() method too.
	// Here we just initialize the metadata with
	// the approval requirements.
	//

	//
	// Initialize metadata for the execution.
	//
	ctx.MetadataContext.Set(NewMetadata())
	return nil
}

func (a *Approval) Actions() []components.Action {
	return []components.Action{
		{
			Name:           "approve",
			Description:    "Approve this execution",
			UserAccessible: true,
			Parameters: []components.ConfigurationField{
				{
					Name:        "requirementIndex",
					Label:       "Requirement Index",
					Type:        components.FieldTypeNumber,
					Description: "Index of the approval requirement being fulfilled",
					Required:    true,
				},
				{
					Name:        "data",
					Label:       "Approval Data",
					Type:        components.FieldTypeObject,
					Description: "Additional data required for this approval",
					Required:    false,
				},
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
			Name:           "reject",
			Description:    "Reject this approval requirement",
			UserAccessible: true,
			Parameters: []components.ConfigurationField{
				{
					Name:        "requirementIndex",
					Label:       "Requirement Index",
					Type:        components.FieldTypeNumber,
					Description: "Index of the approval requirement being rejected",
					Required:    true,
				},
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
	// Get configuration
	config := Config{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Get metadata
	var metadata Metadata
	err = mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Get requirement index from parameters
	requirementIndexFloat, ok := ctx.Parameters["requirementIndex"].(float64)
	if !ok {
		return fmt.Errorf("requirementIndex is required")
	}
	requirementIndex := int(requirementIndexFloat)

	// Validate index
	if requirementIndex < 0 || requirementIndex >= len(config.ApprovalRequirements) {
		return fmt.Errorf("invalid requirement index: %d (total requirements: %d)", requirementIndex, len(config.ApprovalRequirements))
	}

	// Get the requirement
	requirement := &config.ApprovalRequirements[requirementIndex]

	// Check if this requirement has already been approved
	for _, approval := range metadata.Approvals {
		if approval.RequirementIndex == requirementIndex {
			return fmt.Errorf("requirement at index %d has already been approved", requirementIndex)
		}
	}

	// Validate data against requirement's parameters
	if len(requirement.Parameters) > 0 {
		data, _ := ctx.Parameters["data"].(map[string]any)
		if err := components.ValidateConfiguration(requirement.Parameters, data); err != nil {
			return fmt.Errorf("invalid approval data: %w", err)
		}
	}

	// TODO: Get the actual user ID from the request context
	// For now, we'll use a placeholder
	approvedBy := "unknown-user"

	// Add approval
	metadata.addApproval(requirementIndex, approvedBy, ctx.Parameters)
	ctx.MetadataContext.Set(metadata)

	// Check if all requirements are met
	approvedRequirements := make(map[int]bool)
	for _, approval := range metadata.Approvals {
		approvedRequirements[approval.RequirementIndex] = true
	}

	allApproved := len(approvedRequirements) == len(config.ApprovalRequirements)
	if allApproved {
		// All requirements met - finish the execution
		return ctx.ExecutionStateContext.Pass(map[string][]any{
			components.DefaultOutputChannel.Name: {metadata},
		})
	}

	// Still waiting for more approvals - keep execution in waiting state
	// Don't call Pass() or Fail() to keep it pending
	return nil
}

func (a *Approval) handleReject(ctx components.ActionContext) error {
	// Get configuration
	config := Config{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Get requirement index from parameters
	requirementIndexFloat, ok := ctx.Parameters["requirementIndex"].(float64)
	if !ok {
		return fmt.Errorf("requirementIndex is required")
	}
	requirementIndex := int(requirementIndexFloat)

	// Validate index
	if requirementIndex < 0 || requirementIndex >= len(config.ApprovalRequirements) {
		return fmt.Errorf("invalid requirement index: %d (total requirements: %d)", requirementIndex, len(config.ApprovalRequirements))
	}

	// Get the requirement
	requirement := &config.ApprovalRequirements[requirementIndex]

	// Get rejection reason
	reason, ok := ctx.Parameters["reason"]
	if !ok || reason == nil {
		return fmt.Errorf("reason is required for rejection")
	}

	reasonStr, ok := reason.(string)
	if !ok {
		return fmt.Errorf("reason must be a string")
	}

	// Build rejection message with requirement details
	var requirementLabel string
	if requirement.Type == "user" && requirement.User != "" {
		requirementLabel = fmt.Sprintf("User: %s", requirement.User)
	} else if requirement.Type == "role" && requirement.Role != "" {
		requirementLabel = fmt.Sprintf("Role: %s", requirement.Role)
	} else if requirement.Type == "group" && requirement.Group != "" {
		requirementLabel = fmt.Sprintf("Group: %s", requirement.Group)
	} else {
		requirementLabel = fmt.Sprintf("Requirement #%d", requirementIndex+1)
	}

	rejectionMessage := fmt.Sprintf("Rejected by %s: %s", requirementLabel, reasonStr)

	// Fail the execution with the rejection reason
	return ctx.ExecutionStateContext.Fail(rejectionMessage, "")
}
