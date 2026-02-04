package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetRepositoryIssues struct{}

type GetRepositoryIssuesConfiguration struct {
	Repository string  `mapstructure:"repository"`
	State      string  `mapstructure:"state"`
	Labels     *string `mapstructure:"labels,omitempty"`
	Sort       string  `mapstructure:"sort"`
	Direction  string  `mapstructure:"direction"`
	PerPage    *int    `mapstructure:"perPage,omitempty"`
}

func (c *GetRepositoryIssues) Name() string {
	return "github.getRepositoryIssues"
}

func (c *GetRepositoryIssues) Label() string {
	return "Get Repository Issues"
}

func (c *GetRepositoryIssues) Description() string {
	return "List issues from a GitHub repository"
}

func (c *GetRepositoryIssues) Documentation() string {
	return `The Get Repository Issues component retrieves a list of issues from a GitHub repository.

## Use Cases

- **Issue tracking**: Get all open issues for processing or reporting
- **Workflow automation**: List issues matching specific criteria
- **Dashboard integration**: Fetch issues to display in a status dashboard
- **Batch processing**: Process multiple issues in a workflow

## Configuration

- **Repository**: Select the GitHub repository to list issues from
- **State**: Filter by issue state (open, closed, or all)
- **Labels**: Filter by one or more labels (optional)
- **Sort**: Sort by created, updated, or comments
- **Direction**: Sort direction (ascending or descending)
- **Per Page**: Number of issues to return (max 100)

## Output

Returns a list of issues, each containing:
- Issue number, title, and body
- State (open/closed)
- Labels and assignees
- Created and updated timestamps
- Author information
- Comments count`
}

func (c *GetRepositoryIssues) Icon() string {
	return "github"
}

func (c *GetRepositoryIssues) Color() string {
	return "gray"
}

func (c *GetRepositoryIssues) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetRepositoryIssues) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:     "state",
			Label:    "State",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "open",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Open", Value: "open"},
						{Label: "Closed", Value: "closed"},
						{Label: "All", Value: "all"},
					},
				},
			},
			Description: "Filter issues by state",
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., bug, enhancement",
			Description: "Filter by labels (comma-separated)",
		},
		{
			Name:     "sort",
			Label:    "Sort By",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			Default:  "created",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Created", Value: "created"},
						{Label: "Updated", Value: "updated"},
						{Label: "Comments", Value: "comments"},
					},
				},
			},
			Description: "Sort issues by field",
		},
		{
			Name:     "direction",
			Label:    "Sort Direction",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			Default:  "desc",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Descending", Value: "desc"},
						{Label: "Ascending", Value: "asc"},
					},
				},
			},
			Description: "Sort order",
		},
		{
			Name:        "perPage",
			Label:       "Results Per Page",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     30,
			Placeholder: "30",
			Description: "Number of issues to return (max 100)",
		},
	}
}

func (c *GetRepositoryIssues) Setup(ctx core.SetupContext) error {
	return ensureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	)
}

func (c *GetRepositoryIssues) Execute(ctx core.ExecutionContext) error {
	var config GetRepositoryIssuesConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	// Build list options
	opts := &github.IssueListByRepoOptions{
		State:     config.State,
		Sort:      config.Sort,
		Direction: config.Direction,
		ListOptions: github.ListOptions{
			PerPage: 30,
		},
	}

	if config.Labels != nil && *config.Labels != "" {
		// Parse comma-separated labels, filtering empty strings
		rawLabels := strings.Split(*config.Labels, ",")
		labelList := make([]string, 0, len(rawLabels))
		for _, label := range rawLabels {
			trimmed := strings.TrimSpace(label)
			if trimmed != "" {
				labelList = append(labelList, trimmed)
			}
		}
		if len(labelList) > 0 {
			opts.Labels = labelList
		}
	}

	if config.PerPage != nil && *config.PerPage > 0 {
		if *config.PerPage > 100 {
			opts.ListOptions.PerPage = 100
		} else {
			opts.ListOptions.PerPage = *config.PerPage
		}
	}

	// Fetch issues
	issues, _, err := client.Issues.ListByRepo(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		opts,
	)
	if err != nil {
		return fmt.Errorf("failed to list issues: %w", err)
	}

	// Convert issues to output format
	issueList := make([]any, len(issues))
	for i, issue := range issues {
		issueList[i] = buildIssueData(issue)
	}

	// Wrap issueList in []any so the entire list is emitted as ONE output item
	// Emit() processes each element of payloads as a separate output
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.issues",
		[]any{issueList},
	)
}

func (c *GetRepositoryIssues) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetRepositoryIssues) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *GetRepositoryIssues) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetRepositoryIssues) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetRepositoryIssues) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetRepositoryIssues) Cleanup(ctx core.SetupContext) error {
	return nil
}

// buildIssueData converts a GitHub issue to a map for output emission
func buildIssueData(issue *github.Issue) map[string]any {
	data := map[string]any{
		"id":       issue.GetID(),
		"number":   issue.GetNumber(),
		"title":    issue.GetTitle(),
		"body":     issue.GetBody(),
		"state":    issue.GetState(),
		"html_url": issue.GetHTMLURL(),
		"comments": issue.GetComments(),
		"locked":   issue.GetLocked(),
	}

	if issue.CreatedAt != nil {
		data["created_at"] = issue.CreatedAt.Format("2006-01-02T15:04:05Z")
	}

	if issue.UpdatedAt != nil {
		data["updated_at"] = issue.UpdatedAt.Format("2006-01-02T15:04:05Z")
	}

	if issue.ClosedAt != nil {
		data["closed_at"] = issue.ClosedAt.Format("2006-01-02T15:04:05Z")
	}

	if issue.User != nil {
		data["user"] = map[string]any{
			"login":      issue.User.GetLogin(),
			"id":         issue.User.GetID(),
			"avatar_url": issue.User.GetAvatarURL(),
			"html_url":   issue.User.GetHTMLURL(),
		}
	}

	if len(issue.Labels) > 0 {
		labels := make([]map[string]any, len(issue.Labels))
		for i, label := range issue.Labels {
			labels[i] = map[string]any{
				"id":    label.GetID(),
				"name":  label.GetName(),
				"color": label.GetColor(),
			}
		}
		data["labels"] = labels
	}

	if len(issue.Assignees) > 0 {
		assignees := make([]map[string]any, len(issue.Assignees))
		for i, assignee := range issue.Assignees {
			assignees[i] = map[string]any{
				"login":      assignee.GetLogin(),
				"id":         assignee.GetID(),
				"avatar_url": assignee.GetAvatarURL(),
				"html_url":   assignee.GetHTMLURL(),
			}
		}
		data["assignees"] = assignees
	}

	if issue.Milestone != nil {
		data["milestone"] = map[string]any{
			"id":     issue.Milestone.GetID(),
			"number": issue.Milestone.GetNumber(),
			"title":  issue.Milestone.GetTitle(),
			"state":  issue.Milestone.GetState(),
		}
	}

	return data
}
