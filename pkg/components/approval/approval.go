package approval

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	StatePending  = "pending"
	StateApproved = "approved"
	StateRejected = "rejected"
)

func init() {
	registry.RegisterComponent("approval", &Approval{})
}

/*
 * Configuration for the component.
 * Filled when the component is added to a blueprint/workflow.
 */
type Config struct {
	Items []Item `json:"items" mapstructure:"items"`
}

type Item struct {
	Type  string `mapstructure:"type" json:"type"`
	User  string `mapstructure:"user" json:"user,omitempty"`
	Role  string `mapstructure:"role" json:"role,omitempty"`
	Group string `mapstructure:"group" json:"group,omitempty"`
}

/*
 * Metadata for the component.
 */
type Metadata struct {
	Records []Record `mapstructure:"records" json:"records"`
}

func (m *Metadata) Completed() bool {
	for _, record := range m.Records {
		if record.State == StatePending {
			return false
		}
	}

	return true
}

func (m *Metadata) Approve(record *Record, index int, ctx components.ActionContext) {
	record.State = "approved"
	record.At = time.Now().Format(time.RFC3339)

	comment, ok := ctx.Parameters["comment"].(string)
	if ok {
		record.Comment = comment
	}

	user := ctx.UserContext.Get()
	record.By = &user
	m.Records[index] = *record
}

func NewMetadata(items []Item) Metadata {
	records := []Record{}

	for i, item := range items {
		records = append(records, Record{
			Type:  item.Type,
			User:  item.User,
			Role:  item.Role,
			Group: item.Group,
			Index: i,
			State: StatePending,
		})
	}

	return Metadata{
		Records: records,
	}
}

type Record struct {
	Type    string           `mapstructure:"type" json:"type"`
	User    string           `mapstructure:"user" json:"user,omitempty"`
	Role    string           `mapstructure:"role" json:"role,omitempty"`
	Group   string           `mapstructure:"group" json:"group,omitempty"`
	Index   int              `mapstructure:"index" json:"index"`
	State   string           `mapstructure:"state" json:"state"`
	At      string           `mapstructure:"at" json:"at"`
	By      *components.User `mapstructure:"by" json:"by,omitempty"`
	Comment string           `mapstructure:"comment" json:"comment"`
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
			Name:  "items",
			Label: "Items",
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
	// Initialize metadata for the execution.
	//
	ctx.MetadataContext.Set(NewMetadata(config.Items))
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
					Name:        "index",
					Label:       "Item Index",
					Type:        components.FieldTypeNumber,
					Description: "Index of the item being fulfilled",
					Required:    true,
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
					Name:        "index",
					Label:       "Item Index",
					Type:        components.FieldTypeNumber,
					Description: "Index of the item being rejected",
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
	//
	// TODO: check if user can perform this action
	//

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
	var metadata Metadata
	err := mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	record, err := a.findPendingRecord(metadata, ctx.Parameters)
	if err != nil {
		return fmt.Errorf("failed to find requirement: %w", err)
	}

	metadata.Approve(record, record.Index, ctx)
	ctx.MetadataContext.Set(metadata)

	//
	// Check if there are pending records yet.
	//
	if !metadata.Completed() {
		return nil
	}

	return ctx.ExecutionStateContext.Pass(map[string][]any{
		components.DefaultOutputChannel.Name: {metadata},
	})
}

func (a *Approval) findPendingRecord(metadata Metadata, parameters map[string]any) (*Record, error) {
	i, ok := parameters["index"].(float64)
	if !ok {
		return nil, fmt.Errorf("index is required")
	}

	index := int(i)
	if index < 0 || index >= len(metadata.Records) {
		return nil, fmt.Errorf("invalid index: %d", index)
	}

	record := metadata.Records[index]
	if record.State != StatePending {
		return nil, fmt.Errorf("record at index %d is not pending", index)
	}

	return &record, nil
}

// TODO: when I reject an item, does the execution fail?
func (a *Approval) handleReject(ctx components.ActionContext) error {
	var metadata Metadata
	err := mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	record, err := a.findPendingRecord(metadata, ctx.Parameters)
	if err != nil {
		return fmt.Errorf("failed to find requirement: %w", err)
	}

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
	if record.Type == "user" && record.User != "" {
		requirementLabel = fmt.Sprintf("User: %s", record.User)
	} else if record.Type == "role" && record.Role != "" {
		requirementLabel = fmt.Sprintf("Role: %s", record.Role)
	} else if record.Type == "group" && record.Group != "" {
		requirementLabel = fmt.Sprintf("Group: %s", record.Group)
	} else {
		requirementLabel = fmt.Sprintf("Requirement #%d", record.Index+1)
	}

	return ctx.ExecutionStateContext.Fail(
		fmt.Sprintf("Rejected by %s: %s", requirementLabel, reasonStr),
		"",
	)
}
