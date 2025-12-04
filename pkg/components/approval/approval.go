package approval

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	StatePending  = "pending"
	StateApproved = "approved"
	StateRejected = "rejected"

	ItemTypeUser  = "user"
	ItemTypeRole  = "role"
	ItemTypeGroup = "group"

	ChannelApproved = "approved"
	ChannelRejected = "rejected"
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
	Result  string   `mapstructure:"result" json:"result"`
	Records []Record `mapstructure:"records" json:"records"`
}

type Record struct {
	Index     int              `mapstructure:"index" json:"index"`
	Type      string           `mapstructure:"type" json:"type"`
	State     string           `mapstructure:"state" json:"state"`
	User      *components.User `mapstructure:"user" json:"user,omitempty"`
	Role      *string          `mapstructure:"role" json:"role,omitempty"`
	Group     *string          `mapstructure:"group" json:"group,omitempty"`
	Approval  *ApprovalInfo    `mapstructure:"approval" json:"approval,omitempty"`
	Rejection *RejectionInfo   `mapstructure:"rejection" json:"rejection,omitempty"`
}

type ApprovalInfo struct {
	ApprovedAt string `mapstructure:"approvedAt" json:"approvedAt"`
	Comment    string `mapstructure:"comment" json:"comment"`
}

type RejectionInfo struct {
	RejectedAt string `mapstructure:"rejectedAt" json:"rejectedAt"`
	Reason     string `mapstructure:"reason" json:"reason"`
}

func (m *Metadata) Completed() bool {
	for _, record := range m.Records {
		if record.State == StatePending {
			return false
		}
	}

	return true
}

func (m *Metadata) UpdateResult() {
	for _, record := range m.Records {
		if record.State == StateRejected {
			m.Result = StateRejected
			return
		}
	}

	m.Result = StateApproved
}

func (m *Metadata) Approve(record *Record, index int, ctx components.ActionContext) error {
	err := m.validateAction(record, ctx)
	if err != nil {
		return err
	}

	record.State = StateApproved
	record.Approval = &ApprovalInfo{ApprovedAt: time.Now().Format(time.RFC3339)}
	record.User = ctx.AuthContext.AuthenticatedUser()
	comment, ok := ctx.Parameters["comment"].(string)
	if ok {
		record.Approval.Comment = comment
	}

	m.Records[index] = *record
	return nil
}

func (m *Metadata) Reject(record *Record, index int, ctx components.ActionContext) error {
	err := m.validateAction(record, ctx)
	if err != nil {
		return err
	}

	reason, ok := ctx.Parameters["reason"]
	if !ok || reason == nil {
		return fmt.Errorf("reason is required for rejection")
	}

	reasonStr, ok := reason.(string)
	if !ok {
		return fmt.Errorf("reason must be a string")
	}

	record.State = StateRejected
	record.User = ctx.AuthContext.AuthenticatedUser()
	record.Rejection = &RejectionInfo{
		RejectedAt: time.Now().Format(time.RFC3339),
		Reason:     reasonStr,
	}

	m.Records[index] = *record
	return nil
}

func (m *Metadata) validateAction(record *Record, ctx components.ActionContext) error {
	authenticatedUser := ctx.AuthContext.AuthenticatedUser()
	switch record.Type {
	case ItemTypeUser:
		if record.User.ID != authenticatedUser.ID {
			return fmt.Errorf("item must be approved by %s", authenticatedUser.ID)
		}

		return nil

	case ItemTypeRole:
		hasRole, err := ctx.AuthContext.HasRole(*record.Role)
		if err != nil {
			return fmt.Errorf("error checking role %s: %v", *record.Role, err)
		}

		if !hasRole {
			return fmt.Errorf("item must be approved by %s", *record.Role)
		}

		return nil

	case ItemTypeGroup:
		inGroup, err := ctx.AuthContext.InGroup(*record.Group)
		if err != nil {
			return fmt.Errorf("error checking group %s: %v", *record.Group, err)
		}

		if !inGroup {
			return fmt.Errorf("item must be approved by %s", *record.Group)
		}

		return nil
	}

	return fmt.Errorf("unknown record type: %s", record.Type)
}

func NewMetadata(ctx components.ExecutionContext, items []Item) (*Metadata, error) {
	records := []Record{}

	for i, item := range items {
		record, err := approvalItemToRecord(ctx, item, i)
		if err != nil {
			return nil, err
		}

		records = append(records, *record)
	}

	return &Metadata{
		Result:  StatePending,
		Records: records,
	}, nil
}

func approvalItemToRecord(ctx components.ExecutionContext, item Item, index int) (*Record, error) {
	switch item.Type {
	case ItemTypeUser:
		userID, err := uuid.Parse(item.User)
		if err != nil {
			return nil, err
		}

		user, err := ctx.AuthContext.GetUser(userID)
		if err != nil {
			return nil, err
		}

		return &Record{
			Type:  item.Type,
			Index: index,
			State: StatePending,
			User:  user,
		}, nil

	case ItemTypeRole:
		return &Record{
			Type:  item.Type,
			Index: index,
			Role:  &item.Role,
			State: StatePending,
		}, nil

	case ItemTypeGroup:
		return &Record{
			Type:  item.Type,
			Index: index,
			Group: &item.Group,
			State: StatePending,
		}, nil
	}

	return nil, fmt.Errorf("unsupport item type: %s", item.Type)
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
	return "hand"
}

func (a *Approval) Color() string {
	return "orange"
}

func (a *Approval) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{
		{Name: ChannelApproved, Label: "Approved", Description: "All required actors approved"},
		{Name: ChannelRejected, Label: "Rejected", Description: "At least one actor rejected (after everyone responded)"},
	}
}

func (a *Approval) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:  "items",
			Label: "Items",
			Type:  configuration.FieldTypeList,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Item",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "type",
								Label:    "Type",
								Type:     configuration.FieldTypeSelect,
								Required: true,
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
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
								Type:  configuration.FieldTypeUser,
								VisibilityConditions: []configuration.VisibilityCondition{
									{
										Field:  "type",
										Values: []string{"user"},
									},
								},
							},
							{
								Name:  "role",
								Label: "Role",
								Type:  configuration.FieldTypeRole,
								VisibilityConditions: []configuration.VisibilityCondition{
									{
										Field:  "type",
										Values: []string{"role"},
									},
								},
							},
							{
								Name:  "group",
								Label: "Group",
								Type:  configuration.FieldTypeGroup,
								VisibilityConditions: []configuration.VisibilityCondition{
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

func (a *Approval) Setup(ctx components.SetupContext) error {
	config := Config{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return err
	}

	if len(config.Items) == 0 {
		return fmt.Errorf("invalid approval configuration: no user/role/group specified")
	}

	return nil
}

func (a *Approval) ProcessQueueItem(ctx components.ProcessQueueContext) (*models.WorkflowNodeExecution, error) {
	return ctx.DefaultProcessing()
}

func (a *Approval) Execute(ctx components.ExecutionContext) error {
	config := Config{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return err
	}

	metadata, err := NewMetadata(ctx, config.Items)
	if err != nil {
		return err
	}

	metadata.UpdateResult()
	ctx.MetadataContext.Set(metadata)

	//
	// If no items are specified, just finish the execution.
	//
	if metadata.Completed() {
		return ctx.ExecutionStateContext.Pass(map[string][]any{
			ChannelApproved: {metadata},
		})
	}

	return nil
}

func (a *Approval) Actions() []components.Action {
	return []components.Action{
		{
			Name:           "approve",
			Description:    "Approve this execution",
			UserAccessible: true,
			Parameters: []configuration.Field{
				{
					Name:        "index",
					Label:       "Item Index",
					Type:        configuration.FieldTypeNumber,
					Description: "Index of the item being fulfilled",
					Required:    true,
				},
				{
					Name:        "comment",
					Label:       "Comment",
					Type:        configuration.FieldTypeString,
					Description: "Leave a comment before approving",
					Required:    false,
				},
			},
		},
		{
			Name:           "reject",
			Description:    "Reject this approval requirement",
			UserAccessible: true,
			Parameters: []configuration.Field{
				{
					Name:        "index",
					Label:       "Item Index",
					Type:        configuration.FieldTypeNumber,
					Description: "Index of the item being rejected",
					Required:    true,
				},
				{
					Name:        "reason",
					Label:       "Reason",
					Type:        configuration.FieldTypeString,
					Description: "Reason for rejection",
					Required:    true,
				},
			},
		},
	}
}

func (a *Approval) HandleAction(ctx components.ActionContext) error {
	var err error
	var metadata *Metadata
	switch ctx.Name {
	case "approve":
		metadata, err = a.handleApprove(ctx)
	case "reject":
		metadata, err = a.handleReject(ctx)
	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}

	if err != nil {
		return err
	}

	//
	// If we have pending records yet, just update the metadata,
	// without finishing the execution.
	//
	if !metadata.Completed() {
		ctx.MetadataContext.Set(metadata)
		return nil
	}

	//
	// Here, no more pending records exist,
	// so we update the metadata result.
	// If a single item was rejected,
	// the final state of the execution is rejected.
	//
	metadata.UpdateResult()
	ctx.MetadataContext.Set(metadata)

	var outputChannel string
	if metadata.Result == StateApproved {
		outputChannel = ChannelApproved
	} else if metadata.Result == StateRejected {
		outputChannel = ChannelRejected
	}

	return ctx.ExecutionStateContext.Pass(map[string][]any{
		outputChannel: {metadata},
	})
}

func (a *Approval) handleApprove(ctx components.ActionContext) (*Metadata, error) {
	var metadata Metadata
	err := mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	record, err := a.findPendingRecord(metadata, ctx.Parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to find requirement: %w", err)
	}

	err = metadata.Approve(record, record.Index, ctx)
	if err != nil {
		return nil, err
	}

	return &metadata, nil
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

func (a *Approval) handleReject(ctx components.ActionContext) (*Metadata, error) {
	var metadata Metadata
	err := mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	record, err := a.findPendingRecord(metadata, ctx.Parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to find requirement: %w", err)
	}

	err = metadata.Reject(record, record.Index, ctx)
	if err != nil {
		return nil, err
	}

	return &metadata, nil
}
