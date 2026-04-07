package approval

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	StatePending  = "pending"
	StateApproved = "approved"
	StateRejected = "rejected"

	ItemTypeAnyone = "anyone"
	ItemTypeUser   = "user"
	ItemTypeRole   = "role"
	ItemTypeGroup  = "group"

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

type NodeMetadata struct {
	Records []Record `mapstructure:"records" json:"records"`
}

/*
 * Metadata for the component execution.
 */
type Metadata struct {
	Result  string   `mapstructure:"result" json:"result"`
	Records []Record `mapstructure:"records" json:"records"`
}

type Record struct {
	Index     int            `mapstructure:"index" json:"index"`
	Type      string         `mapstructure:"type" json:"type"`
	State     string         `mapstructure:"state" json:"state"`
	User      *core.User     `mapstructure:"user" json:"user,omitempty"`
	RoleRef   *core.RoleRef  `mapstructure:"roleRef" json:"roleRef,omitempty"`
	GroupRef  *core.GroupRef `mapstructure:"groupRef" json:"groupRef,omitempty"`
	Approval  *ApprovalInfo  `mapstructure:"approval" json:"approval,omitempty"`
	Rejection *RejectionInfo `mapstructure:"rejection" json:"rejection,omitempty"`
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
	//
	// If there is a pending record, the result is pending.
	//
	for _, record := range m.Records {
		if record.State == StatePending {
			m.Result = StatePending
			return
		}
	}

	//
	// If there is a rejected record, the result is rejected.
	//
	for _, record := range m.Records {
		if record.State == StateRejected {
			m.Result = StateRejected
			return
		}
	}

	m.Result = StateApproved
}

func (m *Metadata) hasGivenInputInAnyRecord(user *core.User) bool {
	if user == nil {
		return false
	}

	return slices.ContainsFunc(m.Records, func(record Record) bool {
		return record.State != StatePending && record.User != nil && record.User.ID == user.ID
	})
}

func (m *Metadata) Approve(record *Record, index int, ctx core.ActionContext) error {
	user := ctx.Auth.AuthenticatedUser()
	if m.hasGivenInputInAnyRecord(user) {
		return fmt.Errorf("user has already approved/rejected another requirement")
	}

	err := m.validateAction(record, ctx)
	if err != nil {
		return err
	}

	record.State = StateApproved
	record.Approval = &ApprovalInfo{ApprovedAt: time.Now().Format(time.RFC3339)}
	record.User = user
	comment, ok := ctx.Parameters["comment"].(string)
	if ok {
		record.Approval.Comment = comment
	}

	m.Records[index] = *record
	return nil
}

func (m *Metadata) Reject(record *Record, index int, ctx core.ActionContext) error {
	user := ctx.Auth.AuthenticatedUser()
	if m.hasGivenInputInAnyRecord(user) {
		return fmt.Errorf("user has already approved/rejected another requirement")
	}

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
	record.User = user
	record.Rejection = &RejectionInfo{
		RejectedAt: time.Now().Format(time.RFC3339),
		Reason:     reasonStr,
	}

	m.Records[index] = *record
	return nil
}

func (m *Metadata) validateAction(record *Record, ctx core.ActionContext) error {
	authenticatedUser := ctx.Auth.AuthenticatedUser()
	switch record.Type {
	case ItemTypeAnyone:
		// Any authenticated user can approve
		return nil

	case ItemTypeUser:
		if record.User.ID != authenticatedUser.ID {
			return fmt.Errorf("item must be approved by %s", authenticatedUser.ID)
		}

		return nil

	case ItemTypeRole:
		hasRole, err := ctx.Auth.HasRole(record.RoleRef.Name)
		if err != nil {
			return fmt.Errorf("error checking role %s: %v", record.RoleRef.Name, err)
		}

		if !hasRole {
			return fmt.Errorf("item must be approved by %s", record.RoleRef.Name)
		}

		return nil

	case ItemTypeGroup:
		inGroup, err := ctx.Auth.InGroup(record.GroupRef.Name)
		if err != nil {
			return fmt.Errorf("error checking group %s: %v", record.GroupRef.Name, err)
		}

		if !inGroup {
			return fmt.Errorf("item must be approved by %s", record.GroupRef.Name)
		}

		return nil
	}

	return fmt.Errorf("unknown record type: %s", record.Type)
}

func (a *Approval) newNodeMetadata(auth core.AuthContext, items []Item) (*NodeMetadata, error) {
	records := []Record{}

	for i, item := range items {
		record, err := approvalItemToRecord(auth, item, i)
		if err != nil {
			return nil, err
		}

		records = append(records, *record)
	}

	return &NodeMetadata{
		Records: records,
	}, nil
}

func approvalItemToRecord(auth core.AuthContext, item Item, index int) (*Record, error) {
	switch item.Type {
	case ItemTypeAnyone:
		return &Record{
			Type:  item.Type,
			Index: index,
			State: StatePending,
		}, nil

	case ItemTypeUser:
		userID, err := uuid.Parse(item.User)
		if err != nil {
			return nil, err
		}

		user, err := auth.GetUser(userID)
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
		role, err := auth.GetRole(item.Role)
		if err != nil {
			return nil, err
		}

		return &Record{
			Type:    item.Type,
			Index:   index,
			RoleRef: role,
			State:   StatePending,
		}, nil

	case ItemTypeGroup:
		group, err := auth.GetGroup(item.Group)
		if err != nil {
			return nil, err
		}

		return &Record{
			Type:     item.Type,
			Index:    index,
			GroupRef: group,
			State:    StatePending,
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

func (a *Approval) Documentation() string {
	return `The Approval component pauses workflow execution and waits for manual approval from specified users, groups, or roles before continuing.

## Use Cases

- **Deployment approvals**: Require approval before deploying to production
- **Financial transactions**: Get approval for high-value operations
- **Content moderation**: Review content before publishing
- **Compliance workflows**: Ensure regulatory approvals are obtained

## How It Works

1. When the Approval component executes, it creates approval requirements based on the configured approvers
2. The workflow pauses and waits for all required approvals
3. Approvers receive notifications and can approve or reject from the workflow UI
4. Once all approvals are collected, the workflow continues:
   - **Approved channel**: All required approvers approved
   - **Rejected channel**: At least one approver rejected

## Configuration

- **Approvers**: List of users, groups, or roles who must approve
  - **Everyone**: Any authenticated user can approve
  - **Specific user**: Only the specified user can approve
  - **Group**: Any member of the specified group can approve
  - **Role**: Any user with the specified role can approve

## Output Channels

- **Approved**: Emitted when all required approvers have approved
- **Rejected**: Emitted when at least one approver rejects (after all have responded)

## Actions

- **approve**: Approve a pending requirement (can include an optional comment)
- **reject**: Reject a pending requirement (requires a reason)`
}

func (a *Approval) Icon() string {
	return "hand"
}

func (a *Approval) Color() string {
	return "orange"
}

func (a *Approval) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: ChannelApproved, Label: "Approved", Description: "All required actors approved"},
		{Name: ChannelRejected, Label: "Rejected", Description: "At least one actor rejected (after everyone responded)"},
	}
}

func (a *Approval) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "items",
			Label:       "Approvers",
			Description: "List of users, groups, or roles who must approve before the workflow continues",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Default:     `[{"type":"anyone"}]`,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Approver",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "type",
								Label:    "Request approval from",
								Type:     configuration.FieldTypeSelect,
								Required: true,
								Default:  "anyone",
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Value: ItemTypeAnyone, Label: "Any one"},
											{Value: ItemTypeUser, Label: "Specific user"},
											{Value: ItemTypeGroup, Label: "Group"},
											{Value: ItemTypeRole, Label: "Role"},
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
										Values: []string{ItemTypeUser},
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
										Values: []string{ItemTypeRole},
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
										Values: []string{ItemTypeGroup},
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

func (a *Approval) Setup(ctx core.SetupContext) error {
	config := Config{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return err
	}

	metadata, err := a.newNodeMetadata(ctx.Auth, config.Items)
	if err != nil {
		return err
	}

	return ctx.Metadata.Set(metadata)
}

func (a *Approval) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (a *Approval) Execute(ctx core.ExecutionContext) error {
	config := Config{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return err
	}

	nodeMetadata := NodeMetadata{}
	err = mapstructure.Decode(ctx.NodeMetadata.Get(), &nodeMetadata)
	if err != nil {
		return fmt.Errorf("failed to parse node metadata: %w", err)
	}

	//
	// Set execution metadata.
	// If node metadata is set, we use it.
	// Otherwise, we create a new metadata with the configured items.
	//
	var executionMetadata *Metadata
	if len(nodeMetadata.Records) > 0 {
		executionMetadata = &Metadata{Records: nodeMetadata.Records}
		executionMetadata.UpdateResult()
		err = ctx.Metadata.Set(executionMetadata)
		if err != nil {
			return fmt.Errorf("error setting metadata: %v", err)
		}
	} else {
		nodeMetadata, err := a.newNodeMetadata(ctx.Auth, config.Items)
		if err != nil {
			return fmt.Errorf("error creating new metadata: %v", err)
		}

		executionMetadata = &Metadata{Records: nodeMetadata.Records}
		executionMetadata.UpdateResult()
		err = ctx.Metadata.Set(executionMetadata)
		if err != nil {
			return fmt.Errorf("error setting metadata: %v", err)
		}
	}

	//
	// If no items are specified, just finish the execution.
	//
	if executionMetadata.Completed() {
		return ctx.ExecutionState.Emit(
			ChannelApproved,
			"approval.finished",
			[]any{executionMetadata},
		)
	}

	if ctx.Notifications != nil {
		if err := a.notifyApprovers(ctx, executionMetadata); err != nil {
			if ctx.Logger != nil {
				ctx.Logger.Warnf("failed to send approval notification: %v", err)
			}
		}
	}

	return nil
}

func (a *Approval) Actions() []core.Action {
	return []core.Action{
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

func (a *Approval) HandleAction(ctx core.ActionContext) error {
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

	if ctx.Name == "reject" {
		metadata.Result = StateRejected
		if err := ctx.Metadata.Set(metadata); err != nil {
			return err
		}

		return ctx.ExecutionState.Emit(
			ChannelRejected,
			"approval.finished",
			[]any{metadata},
		)
	}

	//
	// If we have pending records yet, just update the metadata,
	// without finishing the execution.
	//
	if !metadata.Completed() {
		return ctx.Metadata.Set(metadata)
	}

	//
	// Here, no more pending records exist,
	// so we update the metadata result.
	// If a single item was rejected,
	// the final state of the execution is rejected.
	//
	metadata.UpdateResult()
	err = ctx.Metadata.Set(metadata)
	if err != nil {
		return err
	}

	var outputChannel string
	if metadata.Result == StateApproved {
		outputChannel = ChannelApproved
	} else if metadata.Result == StateRejected {
		outputChannel = ChannelRejected
	}

	return ctx.ExecutionState.Emit(
		outputChannel,
		"approval.finished",
		[]any{metadata},
	)
}

func (a *Approval) handleApprove(ctx core.ActionContext) (*Metadata, error) {
	var metadata Metadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
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
	index, err := getActionIndex(parameters, len(metadata.Records))
	if err != nil {
		return nil, err
	}

	record := metadata.Records[index]
	if record.State != StatePending {
		return nil, fmt.Errorf("record at index %d is not pending", index)
	}

	return &record, nil
}

func getActionIndex(parameters map[string]any, max int) (int, error) {
	i, ok := parameters["index"].(float64)
	if !ok {
		return 0, fmt.Errorf("index is required")
	}

	index := int(i)
	if index < 0 || index >= max {
		return 0, fmt.Errorf("invalid index: %d", index)
	}

	return index, nil
}

func (a *Approval) handleReject(ctx core.ActionContext) (*Metadata, error) {
	var metadata Metadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
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

func (a *Approval) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (a *Approval) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (a *Approval) notifyApprovers(ctx core.ExecutionContext, metadata *Metadata) error {
	url := fmt.Sprintf(
		"%s/%s/canvases/%s?sidebar=1&node=%s",
		strings.TrimRight(ctx.BaseURL, "/"),
		ctx.OrganizationID,
		ctx.WorkflowID,
		ctx.NodeID,
	)

	title := fmt.Sprintf("Approval required: %s — %s", ctx.CanvasName, ctx.NodeName)
	body := fmt.Sprintf(
		"An approval is waiting for you.\n\nCanvas: %s\nApproval: %s",
		ctx.CanvasName,
		ctx.NodeName,
	)

	receivers := core.NotificationReceivers{}
	emailSet := map[string]struct{}{}
	groupSet := map[string]struct{}{}
	roleSet := map[string]struct{}{}

	for _, record := range metadata.Records {
		if record.State != StatePending {
			continue
		}

		switch record.Type {
		case ItemTypeAnyone:
			roleSet[models.RoleOrgViewer] = struct{}{}
			roleSet[models.RoleOrgAdmin] = struct{}{}
			roleSet[models.RoleOrgOwner] = struct{}{}

		case ItemTypeUser:
			if record.User != nil && record.User.Email != "" {
				emailSet[record.User.Email] = struct{}{}
			}

		case ItemTypeRole:
			if record.RoleRef != nil && record.RoleRef.Name != "" {
				roleSet[record.RoleRef.Name] = struct{}{}
			}

		case ItemTypeGroup:
			if record.GroupRef != nil && record.GroupRef.Name != "" {
				groupSet[record.GroupRef.Name] = struct{}{}
			}
		}
	}

	receivers.Emails = mapKeys(emailSet)
	receivers.Groups = mapKeys(groupSet)
	receivers.Roles = mapKeys(roleSet)

	return ctx.Notifications.Send(title, body, url, "Open approval", receivers)
}

func mapKeys(input map[string]struct{}) []string {
	result := make([]string, 0, len(input))
	for key := range input {
		result = append(result, key)
	}
	return result
}

func (a *Approval) Cleanup(ctx core.SetupContext) error {
	return nil
}
