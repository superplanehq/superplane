package sentry

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateIssue struct{}

type UpdateIssueConfiguration struct {
	Project      string  `json:"project" mapstructure:"project"`
	IssueID      string  `json:"issueId" mapstructure:"issueId"`
	Status       *string `json:"status,omitempty" mapstructure:"status"`
	AssignedTo   *string `json:"assignedTo,omitempty" mapstructure:"assignedTo"`
	HasSeen      *bool   `json:"hasSeen,omitempty" mapstructure:"hasSeen"`
	IsBookmarked *bool   `json:"isBookmarked,omitempty" mapstructure:"isBookmarked"`
}

type UpdateIssueMetadata struct {
	IssueID     string `json:"issueId"`
	ShortID     string `json:"shortId"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Level       string `json:"level"`
	Culprit     string `json:"culprit"`
	FirstSeen   string `json:"firstSeen"`
	LastSeen    string `json:"lastSeen"`
	Count       string `json:"count"`
	UserCount   int    `json:"userCount"`
	Permalink   string `json:"permalink"`
	ProjectSlug string `json:"projectSlug"`
	ProjectName string `json:"projectName"`
	AssignedTo  string `json:"assignedTo,omitempty"`
}

func (c *UpdateIssue) Name() string {
	return "sentry.updateIssue"
}

func (c *UpdateIssue) Label() string {
	return "Update Issue"
}

func (c *UpdateIssue) Description() string {
	return "Update a Sentry issue"
}

func (c *UpdateIssue) Documentation() string {
	return `The Update Issue component modifies an existing Sentry issue's status, assignment, or other properties.

## Use Cases

- **Auto-resolve on deploy**: Automatically resolve issues after a successful production deployment
- **Escalation workflows**: Assign issues to on-call engineers when they meet certain criteria
- **Bulk triage**: Mark multiple issues as seen or ignored based on patterns
- **Cross-platform sync**: Update Sentry issues when tickets are closed in Jira or Linear
- **SLA enforcement**: Auto-ignore low-priority issues that haven't been addressed

## Configuration

| Field | Description | Example Expression |
|-------|-------------|-------------------|
| **Issue ID** | The Sentry issue ID to update (required) | ` + "`$['trigger'].data.issue.id`" + ` |
| **Status** | New issue status | ` + "`resolved`" + `, ` + "`ignored`" + `, ` + "`unresolved`" + ` |
| **Assigned To** | User ID or username to assign | ` + "`me`" + `, ` + "`user:123`" + `, ` + "`john.doe`" + ` |
| **Mark as Seen** | Mark the issue as viewed | ` + "`true`" + ` or ` + "`false`" + ` |
| **Bookmark** | Add/remove bookmark | ` + "`true`" + ` or ` + "`false`" + ` |

## Status Values

| Status | Description |
|--------|-------------|
| ` + "`resolved`" + ` | Mark as resolved (issue is fixed) |
| ` + "`resolvedInNextRelease`" + ` | Resolve in the next release |
| ` + "`unresolved`" + ` | Reopen a resolved issue |
| ` + "`ignored`" + ` | Ignore the issue (stops alerts) |

## Output Payload

The component outputs the updated issue object:

` + "```" + `
$['updateIssue'].id           # Issue ID
$['updateIssue'].shortId      # Short ID like "PROJ-123"
$['updateIssue'].title        # Issue title
$['updateIssue'].status       # Updated status
$['updateIssue'].assignedTo   # Assigned user object (if set)
$['updateIssue'].permalink    # Direct link to issue in Sentry
` + "```" + `

## Example: Auto-resolve After Deploy

1. Use a **Semaphore On Pipeline Done** trigger filtered to production deploys
2. Add this **Update Issue** component with:
   - Issue ID: ` + "`$['trigger'].data.issue.id`" + ` (from a previous step)
   - Status: ` + "`resolved`" + `

## Notes

- Only fields you specify will be updated; others remain unchanged
- The ` + "`assignedTo`" + ` field accepts user IDs, usernames, or ` + "`me`" + ` for the token owner
- Resolving an issue in Sentry marks it as fixed but it can regress if the error recurs
- Use ` + "`ignored`" + ` for issues you don't want to be alerted about anymore`
}

func (c *UpdateIssue) Icon() string {
	return "sentry"
}

func (c *UpdateIssue) Color() string {
	return "purple"
}

func (c *UpdateIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "project",
					UseNameAsValue: true,
				},
			},
			Description: "The Sentry project containing the issue",
		},
		{
			Name:     "issueId",
			Label:    "Issue",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "issue",
					Parameters: []configuration.ParameterRef{
						{
							Name: "project",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "project",
							},
						},
					},
				},
			},
			Description: "The Sentry issue to update",
		},
		{
			Name:        "status",
			Label:       "Status",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Issue status: resolved, resolvedInNextRelease, unresolved, or ignored",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{
							Label: "Resolved",
							Value: "resolved",
						},
						{
							Label: "Resolved in Next Release",
							Value: "resolvedInNextRelease",
						},
						{
							Label: "Unresolved",
							Value: "unresolved",
						},
						{
							Label: "Ignored",
							Value: "ignored",
						},
					},
				},
			},
		},
		{
			Name:        "assignedTo",
			Label:       "Assigned To",
			Type:        configuration.FieldTypeExpression,
			Required:    false,
			Description: "User ID or username to assign the issue to",
			Placeholder: "$['trigger'].data.assignedTo",
		},
		{
			Name:        "hasSeen",
			Label:       "Mark as Seen",
			Type:        configuration.FieldTypeExpression,
			Required:    false,
			Description: "Mark the issue as seen (true/false)",
			Placeholder: "$['trigger'].data.hasSeen",
		},
		{
			Name:        "isBookmarked",
			Label:       "Bookmark",
			Type:        configuration.FieldTypeExpression,
			Required:    false,
			Description: "Bookmark the issue (true/false)",
			Placeholder: "$['trigger'].data.isBookmarked",
		},
	}
}

func (c *UpdateIssue) Setup(ctx core.SetupContext) error {
	config := UpdateIssueConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.IssueID == "" {
		return fmt.Errorf("issue ID is required")
	}

	if ctx.HTTP != nil && ctx.Integration != nil {
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil
		}

		issue, err := client.GetIssue(config.IssueID)
		if err != nil {
			return nil
		}

		metadata := UpdateIssueMetadata{
			IssueID:     issue.ID,
			ShortID:     issue.ShortID,
			Title:       issue.Title,
			Status:      issue.Status,
			Level:       issue.Level,
			Culprit:     issue.Culprit,
			FirstSeen:   issue.FirstSeen,
			LastSeen:    issue.LastSeen,
			Count:       issue.Count,
			UserCount:   issue.UserCount,
			Permalink:   issue.Permalink,
			ProjectSlug: issue.Project.Slug,
			ProjectName: issue.Project.Name,
		}

		if issue.AssignedTo != nil {
			metadata.AssignedTo = issue.AssignedTo.Name
		}

		_ = ctx.Metadata.Set(metadata)
	}

	return nil
}

func (c *UpdateIssue) Execute(ctx core.ExecutionContext) error {
	var config UpdateIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize Sentry client: %w", err)
	}

	request := UpdateIssueRequest{}

	if config.Status != nil && *config.Status != "" {
		request.Status = *config.Status
	}

	if config.AssignedTo != nil && *config.AssignedTo != "" {
		request.AssignedTo = *config.AssignedTo
	}

	if config.HasSeen != nil {
		request.HasSeen = config.HasSeen
	}

	if config.IsBookmarked != nil {
		request.IsBookmarked = config.IsBookmarked
	}

	issue, err := client.UpdateIssue(config.IssueID, request)
	if err != nil {
		return fmt.Errorf("failed to update issue: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"sentry.issue",
		[]any{issue},
	)
}

func (c *UpdateIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *UpdateIssue) Actions() []core.Action {
	return []core.Action{}
}

func (c *UpdateIssue) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *UpdateIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateIssue) Cleanup(ctx core.SetupContext) error {
	return nil
}
