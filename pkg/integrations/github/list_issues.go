package github

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const ListIssuesPayloadType = "github.issues"
const ListIssuesOutputChannel = "default"

type ListIssues struct{}

type ListIssuesConfiguration struct {
	Repository  string `json:"repository" mapstructure:"repository"`
	SearchQuery string `json:"searchQuery" mapstructure:"searchQuery"`
	State       string `json:"state" mapstructure:"state"`
	Labels      string `json:"labels" mapstructure:"labels"`
	Assignee    string `json:"assignee" mapstructure:"assignee"`
	Creator     string `json:"creator" mapstructure:"creator"`
	Mentioned   string `json:"mentioned" mapstructure:"mentioned"`
	Sort        string `json:"sort" mapstructure:"sort"`
	Direction   string `json:"direction" mapstructure:"direction"`
	Since       string `json:"since" mapstructure:"since"`
	PerPage     string `json:"perPage" mapstructure:"perPage"`
	Page        string `json:"page" mapstructure:"page"`
}

func (c *ListIssues) Name() string {
	return "github.listIssues"
}

func (c *ListIssues) Label() string {
	return "Get Repository Issues"
}

func (c *ListIssues) Description() string {
	return "List issues from a GitHub repository with search and filter options"
}

func (c *ListIssues) Documentation() string {
	return `The Get Repository Issues component lists issues from a GitHub repository with powerful search and filter capabilities.

## Use Cases

- **Issue reporting**: List open or closed issues for reporting or automation
- **Triage workflows**: Find issues by label, assignee, or author for triage
- **Export and sync**: Export issue list for sync with Jira or Slack
- **Status dashboards**: Get issues matching specific criteria for dashboards

## Configuration

### Search Query (Optional)
A GitHub issue search query string (e.g., ` + "`is:issue state:open label:bug`" + `). When provided, this takes precedence and other filters are ignored.

### Individual Filters
- **Repository** (required): The GitHub repository to list issues from
- **State**: Filter by state (open, closed, all). Default: open
- **Labels**: Comma-separated list of labels to filter by
- **Assignee**: Filter by assignee username
- **Creator**: Filter by issue creator username
- **Mentioned**: Filter by user mentioned in issue
- **Sort**: Sort by (created, updated, comments). Default: created
- **Direction**: Sort direction (asc, desc). Default: desc
- **Since**: Only issues updated after this date (ISO 8601 format)
- **Per Page**: Results per page (1-100). Default: 30
- **Page**: Page number for pagination. Default: 1

## Output

Emits a list of issues to the default channel. Each issue includes:
- number, title, body, state
- user (author), assignees, labels
- created_at, updated_at, closed_at
- comments count, milestone, html_url

## Notes

- When using Search Query, the query must include ` + "`repo:owner/name`" + ` or the repository field is used
- Maximum 100 issues per page (GitHub API limit)
- For large result sets, use pagination with Page parameter`
}

func (c *ListIssues) Icon() string {
	return "github"
}

func (c *ListIssues) Color() string {
	return "gray"
}

func (c *ListIssues) ExampleOutput() map[string]any {
	return map[string]any{
		"issues": []map[string]any{
			{
				"number":     42,
				"title":      "Bug: Something is broken",
				"state":      "open",
				"user":       map[string]any{"login": "octocat"},
				"labels":     []map[string]any{{"name": "bug"}},
				"created_at": "2026-02-08T10:00:00Z",
			},
		},
		"total_count": 1,
	}
}

func (c *ListIssues) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:        ListIssuesOutputChannel,
			Label:       "Default",
			Description: "Emits the list of issues",
		},
	}
}

func (c *ListIssues) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "searchQuery",
			Label:       "Search Query",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "GitHub search query (e.g., is:issue state:open label:bug). Overrides other filters when set.",
			Placeholder: "is:issue state:open label:bug assignee:username",
		},
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
			Name:        "state",
			Label:       "State",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "open",
			Description: "Filter by issue state",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Open", Value: "open"},
						{Label: "Closed", Value: "closed"},
						{Label: "All", Value: "all"},
					},
				},
			},
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Comma-separated list of labels (e.g., bug,urgent)",
			Placeholder: "bug,enhancement",
		},
		{
			Name:        "assignee",
			Label:       "Assignee",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter by assignee username",
			Placeholder: "username",
		},
		{
			Name:        "creator",
			Label:       "Creator",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter by issue creator username",
			Placeholder: "username",
		},
		{
			Name:        "mentioned",
			Label:       "Mentioned",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Filter by user mentioned in issue",
			Placeholder: "username",
		},
		{
			Name:        "sort",
			Label:       "Sort By",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "created",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Created", Value: "created"},
						{Label: "Updated", Value: "updated"},
						{Label: "Comments", Value: "comments"},
					},
				},
			},
		},
		{
			Name:        "direction",
			Label:       "Sort Direction",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "desc",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Descending", Value: "desc"},
						{Label: "Ascending", Value: "asc"},
					},
				},
			},
		},
		{
			Name:        "since",
			Label:       "Since",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Only issues updated after this date (ISO 8601)",
			Placeholder: "2026-01-01T00:00:00Z",
		},
		{
			Name:        "perPage",
			Label:       "Per Page",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "30",
			Description: "Results per page (1-100)",
			Placeholder: "30",
		},
		{
			Name:        "page",
			Label:       "Page",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "1",
			Description: "Page number",
			Placeholder: "1",
		},
	}
}

func (c *ListIssues) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ListIssues) Setup(ctx core.SetupContext) error {
	return ensureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	)
}

func (c *ListIssues) Execute(ctx core.ExecutionContext) error {
	var config ListIssuesConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	// Initialize GitHub client
	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	// Parse pagination
	perPage := 30
	if config.PerPage != "" {
		if p, err := strconv.Atoi(config.PerPage); err == nil && p > 0 && p <= 100 {
			perPage = p
		}
	}

	page := 1
	if config.Page != "" {
		if p, err := strconv.Atoi(config.Page); err == nil && p > 0 {
			page = p
		}
	}

	var issues []*github.Issue
	var totalCount int

	// If search query is provided, use Search API
	if config.SearchQuery != "" {
		query := config.SearchQuery
		// Add repo qualifier if not present
		if !strings.Contains(query, "repo:") {
			query = fmt.Sprintf("repo:%s/%s %s", appMetadata.Owner, config.Repository, query)
		}
		// Ensure it's searching issues
		if !strings.Contains(query, "is:issue") && !strings.Contains(query, "type:issue") {
			query = "is:issue " + query
		}

		opts := &github.SearchOptions{
			ListOptions: github.ListOptions{
				PerPage: perPage,
				Page:    page,
			},
		}

		result, _, err := client.Search.Issues(context.Background(), query, opts)
		if err != nil {
			return ctx.ExecutionState.Fail("api_error", fmt.Sprintf("failed to search issues: %v", err))
		}

		issues = result.Issues
		totalCount = result.GetTotal()
	} else {
		// Use List API with filters
		opts := &github.IssueListByRepoOptions{
			State:     config.State,
			Sort:      config.Sort,
			Direction: config.Direction,
			ListOptions: github.ListOptions{
				PerPage: perPage,
				Page:    page,
			},
		}

		if config.Labels != "" {
			opts.Labels = strings.Split(config.Labels, ",")
		}
		if config.Assignee != "" {
			opts.Assignee = config.Assignee
		}
		if config.Creator != "" {
			opts.Creator = config.Creator
		}
		if config.Mentioned != "" {
			opts.Mentioned = config.Mentioned
		}

		result, _, err := client.Issues.ListByRepo(
			context.Background(),
			appMetadata.Owner,
			config.Repository,
			opts,
		)
		if err != nil {
			return ctx.ExecutionState.Fail("api_error", fmt.Sprintf("failed to list issues: %v", err))
		}

		issues = result
		totalCount = len(result)
	}

	// Filter out pull requests (GitHub API returns PRs as issues)
	filteredIssues := make([]*github.Issue, 0)
	for _, issue := range issues {
		if issue.PullRequestLinks == nil {
			filteredIssues = append(filteredIssues, issue)
		}
	}

	payload := map[string]any{
		"issues":      filteredIssues,
		"total_count": totalCount,
		"page":        page,
		"per_page":    perPage,
	}

	return ctx.ExecutionState.Emit(ListIssuesOutputChannel, ListIssuesPayloadType, []any{payload})
}

func (c *ListIssues) Actions() []core.Action {
	return []core.Action{}
}

func (c *ListIssues) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *ListIssues) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *ListIssues) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ListIssues) Cleanup(ctx core.SetupContext) error {
	return nil
}
