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

/*
 * Metadata for the component.
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
	Role      *string        `mapstructure:"role" json:"role,omitempty"`
	Group     *string        `mapstructure:"group" json:"group,omitempty"`
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

func (m *Metadata) hasApprovedAnyRecord(userID string) bool {
	if userID == "" {
		return false
	}

	return slices.ContainsFunc(m.Records, func(record Record) bool {
		if record.State != StateApproved || record.User == nil {
			return false
		}

		return record.User.ID == userID
	})
}

func (m *Metadata) Approve(record *Record, index int, ctx core.ActionContext) error {
	authenticatedUser := ctx.Auth.AuthenticatedUser()
	if authenticatedUser != nil && m.hasApprovedAnyRecord(authenticatedUser.ID) {
		return fmt.Errorf("user has already approved another requirement")
	}

	err := m.validateAction(record, ctx)
	if err != nil {
		return err
	}

	record.State = StateApproved
	record.Approval = &ApprovalInfo{ApprovedAt: time.Now().Format(time.RFC3339)}
	record.User = authenticatedUser
	comment, ok := ctx.Parameters["comment"].(string)
	if ok {
		record.Approval.Comment = comment
	}

	m.Records[index] = *record
	return nil
}

func (m *Metadata) Reject(record *Record, index int, ctx core.ActionContext) error {
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
	record.User = ctx.Auth.AuthenticatedUser()
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
		hasRole, err := ctx.Auth.HasRole(*record.Role)
		if err != nil {
			return fmt.Errorf("error checking role %s: %v", *record.Role, err)
		}

		if !hasRole {
			return fmt.Errorf("item must be approved by %s", *record.Role)
		}

		return nil

	case ItemTypeGroup:
		inGroup, err := ctx.Auth.InGroup(*record.Group)
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

func NewMetadata(ctx core.ExecutionContext, items []Item) (*Metadata, error) {
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

func approvalItemToRecord(ctx core.ExecutionContext, item Item, index int) (*Record, error) {
	switch item.Type {
	case ItemTypeAnyone:
		return &Record{
			Type:  item.Type,
			Index: index,
			State: StatePending,
			User:  nil, // No specific user - anyone can approve
		}, nil

	case ItemTypeUser:
		userID, err := uuid.Parse(item.User)
		if err != nil {
			return nil, err
		}

		user, err := ctx.Auth.GetUser(userID)
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
											{Value: "anyone", Label: "Any user"},
											{Value: "user", Label: "Specific user"},
											{Value: "group", Label: "Group"},
											// TODO: Uncomment after RBAC definitive implementation
											// {Value: "role", Label: "Role"},
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

func (a *Approval) Setup(ctx core.SetupContext) error {
	return nil
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

	metadata, err := NewMetadata(ctx, config.Items)
	if err != nil {
		return err
	}

	metadata.UpdateResult()
	err = ctx.Metadata.Set(metadata)
	if err != nil {
		return fmt.Errorf("error setting metadata: %v", err)
	}

	//
	// If no items are specified, just finish the execution.
	//
	if metadata.Completed() {
		return ctx.ExecutionState.Emit(
			ChannelApproved,
			"approval.finished",
			[]any{metadata},
		)
	}

	if ctx.Notifications != nil {
		if err := a.notifyApprovers(ctx, metadata); err != nil {
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

	record, err := a.resolveApproveRecord(metadata, ctx)
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

func (a *Approval) resolveApproveRecord(metadata Metadata, ctx core.ActionContext) (*Record, error) {
	index, err := getActionIndex(ctx.Parameters, len(metadata.Records))
	if err != nil {
		return nil, err
	}

	requestedRecord := metadata.Records[index]
	authenticatedUser := ctx.Auth.AuthenticatedUser()
	if requestedRecord.Type == ItemTypeAnyone && authenticatedUser != nil {
		userRecord := findPendingUserRecord(metadata, authenticatedUser.ID)
		if userRecord != nil {
			return userRecord, nil
		}
	}

	if requestedRecord.State == StatePending {
		if err := metadata.validateAction(&requestedRecord, ctx); err != nil {
			return nil, err
		}

		return &requestedRecord, nil
	}

	fallback := findPendingEligibleRecord(metadata, ctx)
	if fallback != nil {
		return fallback, nil
	}

	return nil, fmt.Errorf("record at index %d is not pending", index)
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

func findPendingUserRecord(metadata Metadata, userID string) *Record {
	for i, record := range metadata.Records {
		if record.State != StatePending || record.Type != ItemTypeUser || record.User == nil {
			continue
		}

		if record.User.ID == userID {
			record.Index = i
			return &record
		}
	}

	return nil
}

func findPendingEligibleRecord(metadata Metadata, ctx core.ActionContext) *Record {
	for i, record := range metadata.Records {
		if record.State != StatePending {
			continue
		}

		if record.Type == ItemTypeUser && record.User == nil {
			continue
		}

		if record.Type == ItemTypeRole && record.Role == nil {
			continue
		}

		if record.Type == ItemTypeGroup && record.Group == nil {
			continue
		}

		if err := metadata.validateAction(&record, ctx); err != nil {
			continue
		}

		record.Index = i
		return &record
	}

	return nil
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

func (a *Approval) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (a *Approval) notifyApprovers(ctx core.ExecutionContext, metadata *Metadata) error {
	url := ""
	if ctx.BaseURL != "" && ctx.OrganizationID != "" && ctx.WorkflowID != "" && ctx.NodeID != "" {
		url = fmt.Sprintf(
			"%s/%s/workflows/%s?sidebar=1&node=%s",
			strings.TrimRight(ctx.BaseURL, "/"),
			ctx.OrganizationID,
			ctx.WorkflowID,
			ctx.NodeID,
		)
	}

	title := "Approval required"
	body := "A canvas run item is waiting for your approval. Please visit the URL below to handle it."

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
			if record.Role != nil && *record.Role != "" {
				roleSet[*record.Role] = struct{}{}
			}

		case ItemTypeGroup:
			if record.Group != nil && *record.Group != "" {
				groupSet[*record.Group] = struct{}{}
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
