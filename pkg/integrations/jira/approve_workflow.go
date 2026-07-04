package jira

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const ApproveWorkflowPayloadType = "jira.approval"

const (
	approvalSelectorLatestPending = "latestPending"
	approvalSelectorByID          = "byId"
)

type ApproveWorkflow struct{}

type ApproveWorkflowSpec struct {
	IssueKey         string `json:"issueKey" mapstructure:"issueKey"`
	Decision         string `json:"decision" mapstructure:"decision"`
	ApprovalSelector string `json:"approvalSelector" mapstructure:"approvalSelector"`
	ApprovalID       string `json:"approvalId" mapstructure:"approvalId"`
	Comment          string `json:"comment" mapstructure:"comment"`
}

func (c *ApproveWorkflow) Name() string {
	return "jira.approveWorkflow"
}

func (c *ApproveWorkflow) Label() string {
	return "Approve Workflow"
}

func (c *ApproveWorkflow) Description() string {
	return "Approve or decline a Jira Service Management request approval"
}

func (c *ApproveWorkflow) Documentation() string {
	return `The Approve Workflow component approves or declines a Jira Service Management request approval.

## Use Cases

- **Automated approval routing**: submit a JSM approval decision after external checks pass
- **Escalation handling**: decline requests when a SuperPlane workflow detects a failed precondition
- **Audit context**: add a customer request comment before submitting the approval decision

## Configuration

- **Issue Key**: JSM request issue key, for example ` + "`ITSM-123`" + `.
- **Decision**: Approve or decline.
- **Approval Selector**: Choose the latest pending approval or pick a specific one from the list.
- **Approval**: The pending approval to decide. Required when picking a specific approval.
- **Comment**: Optional public customer request comment posted before the approval decision.

## Output

Returns the updated approval payload from Jira Service Management.

## Notes

- Requires the API token's user to be in the approver list.
- This component only works on Jira Service Management customer requests, not standard Jira issues.`
}

func (c *ApproveWorkflow) Icon() string {
	return "jira"
}

func (c *ApproveWorkflow) Color() string {
	return "green"
}

func (c *ApproveWorkflow) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ApproveWorkflow) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "issueKey",
			Label:       "Issue Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Jira Service Management request issue key",
			Placeholder: "ITSM-123",
		},
		{
			Name:        "decision",
			Label:       "Decision",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "Approval decision",
			Default:     "approve",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Approve", Value: "approve"},
						{Label: "Decline", Value: "decline"},
					},
				},
			},
		},
		{
			Name:        "approvalSelector",
			Label:       "Approval Selector",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "How to choose the approval",
			Default:     approvalSelectorLatestPending,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Latest pending", Value: approvalSelectorLatestPending},
						{Label: "Pick approval", Value: approvalSelectorByID},
					},
				},
			},
		},
		{
			Name:        "approvalId",
			Label:       "Approval",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Pending approval to decide",
			Placeholder: "Select an approval",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "jsmApproval",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "issueKey",
							ValueFrom: &configuration.ParameterValueFrom{Field: "issueKey"},
						},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "approvalSelector", Values: []string{approvalSelectorByID}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "approvalSelector", Values: []string{approvalSelectorByID}},
			},
		},
		{
			Name:        "comment",
			Label:       "Comment",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Optional public customer request comment to post before the decision",
		},
	}
}

func (c *ApproveWorkflow) Setup(ctx core.SetupContext) error {
	spec := ApproveWorkflowSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	return validateApproveWorkflowSpec(spec)
}

func (c *ApproveWorkflow) Execute(ctx core.ExecutionContext) error {
	spec := ApproveWorkflowSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}
	if err := validateApproveWorkflowSpec(spec); err != nil {
		return err
	}

	issueKey := strings.TrimSpace(spec.IssueKey)
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	request, err := client.GetCustomerRequest(issueKey)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return fmt.Errorf("issue %s is not a Jira Service Management request; approvals only work on JSM service requests", issueKey)
		}
		return fmt.Errorf("failed to load JSM request: %v", err)
	}
	if strings.TrimSpace(request.ServiceDeskID) == "" {
		return fmt.Errorf("issue %s is not a Jira Service Management request; approvals only work on JSM service requests", issueKey)
	}

	approvalID, err := c.resolveApprovalID(client, issueKey, spec)
	if err != nil {
		return err
	}

	if comment := strings.TrimSpace(spec.Comment); comment != "" {
		// public=true makes the comment visible to the JSM customer alongside the decision.
		if err := client.AddCustomerRequestComment(issueKey, comment, true); err != nil && ctx.Logger != nil {
			ctx.Logger.Warnf("jira.approveWorkflow: failed to add request comment before approval decision: %v", err)
		}
	}

	approval, err := client.SubmitApprovalDecision(issueKey, approvalID, strings.TrimSpace(spec.Decision))
	if err != nil {
		if strings.Contains(err.Error(), "403") {
			return fmt.Errorf("approve a JSM request requires the API token's user to be in the approver list")
		}
		return fmt.Errorf("failed to submit approval decision: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		ApproveWorkflowPayloadType,
		[]any{approval},
	)
}

func validateApproveWorkflowSpec(spec ApproveWorkflowSpec) error {
	if strings.TrimSpace(spec.IssueKey) == "" {
		return fmt.Errorf("issueKey is required")
	}
	decision := strings.ToLower(strings.TrimSpace(spec.Decision))
	if decision != "approve" && decision != "decline" {
		return fmt.Errorf("decision must be approve or decline")
	}
	selector := normalizeApprovalSelector(spec.ApprovalSelector)
	if selector == approvalSelectorByID && strings.TrimSpace(spec.ApprovalID) == "" {
		return fmt.Errorf("approvalId is required when approvalSelector is byId")
	}
	return nil
}

func normalizeApprovalSelector(selector string) string {
	if strings.TrimSpace(selector) == approvalSelectorByID {
		return approvalSelectorByID
	}
	return approvalSelectorLatestPending
}

func (c *ApproveWorkflow) resolveApprovalID(client *Client, issueKey string, spec ApproveWorkflowSpec) (string, error) {
	if normalizeApprovalSelector(spec.ApprovalSelector) == approvalSelectorByID {
		return strings.TrimSpace(spec.ApprovalID), nil
	}

	approvals, err := client.ListApprovals(issueKey)
	if err != nil {
		return "", fmt.Errorf("failed to list approvals: %v", err)
	}
	approvalID, ok := latestPendingApprovalID(approvals)
	if !ok {
		return "", fmt.Errorf("no pending approval found for %s", issueKey)
	}
	return approvalID, nil
}

func isPendingApproval(approval Approval) bool {
	// JSM marks open approvals with finalDecision "pending", but for some
	// workflows it leaves the field empty until a decision is made. A completed
	// approval always carries a concrete decision (approved/declined), so treat
	// both "pending" and an empty decision as still pending — otherwise open
	// approvals are skipped and resolution reports "no pending approval".
	decision := strings.TrimSpace(approval.FinalDecision)
	return decision == "" || strings.EqualFold(decision, "PENDING")
}

// latestPendingApprovalID returns the most recently created pending approval.
// Jira lists approvals oldest-first; when createdDate is missing we use list
// position (last pending wins) as a fallback.
func latestPendingApprovalID(approvals []Approval) (string, bool) {
	var (
		bestID    string
		bestTime  time.Time
		bestIndex = -1
		hasTime   bool
	)

	for i, approval := range approvals {
		if !isPendingApproval(approval) {
			continue
		}
		id := approval.ID.String()
		if t, ok := approvalCreatedTime(approval); ok {
			if !hasTime || t.After(bestTime) || (t.Equal(bestTime) && i > bestIndex) {
				bestTime = t
				bestID = id
				bestIndex = i
				hasTime = true
			}
			continue
		}
		if !hasTime && i > bestIndex {
			bestID = id
			bestIndex = i
		}
	}

	if bestIndex < 0 {
		return "", false
	}
	return bestID, true
}

func approvalCreatedTime(approval Approval) (time.Time, bool) {
	if approval.CreatedDate == nil {
		return time.Time{}, false
	}
	for _, key := range []string{"iso8601", "jira"} {
		raw, ok := approval.CreatedDate[key].(string)
		if !ok {
			continue
		}
		if t, ok := parseJiraDateTime(raw); ok {
			return t, true
		}
	}
	return time.Time{}, false
}

func parseJiraDateTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05-0700",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func (c *ApproveWorkflow) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ApproveWorkflow) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ApproveWorkflow) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *ApproveWorkflow) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *ApproveWorkflow) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *ApproveWorkflow) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
