package jira

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const GetIssuePayloadType = "jira.issue.fetched"

type GetIssue struct{}

type GetIssueSpec struct {
	IssueKey string `json:"issueKey"`
}

type GetIssuePayload struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Self        string `json:"self"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Status      string `json:"status"`
	IssueType   string `json:"issueType"`
	Priority    string `json:"priority"`
	Assignee    string `json:"assignee"`
}

func (g *GetIssue) Name() string { return "jira.getIssue" }

func (g *GetIssue) Label() string { return "Get Issue" }

func (g *GetIssue) Description() string {
	return "Fetch a Jira issue and emit its fields for use in downstream nodes"
}

func (g *GetIssue) Documentation() string {
	return `The Get Issue component fetches a Jira issue by its key and emits its fields.

## Use Cases

- **Feed task to AI agent**: Pass the issue summary and description to a Claude Code Agent
- **Branch naming**: Use the issue key as the branch name
- **Conditional routing**: Route based on issue type or priority

## Configuration

- **Issue Key**: The Jira issue key to fetch (e.g. KAN-1). Supports expressions.

## Output

- **key**: Issue key (e.g. KAN-1)
- **summary**: Issue title
- **description**: Issue description (plain text)
- **status**: Current status name
- **issueType**: Issue type (Bug, Story, Task, etc.)
- **priority**: Priority name
- **assignee**: Assignee display name`
}

func (g *GetIssue) Icon() string  { return "jira" }
func (g *GetIssue) Color() string { return "blue" }

func (g *GetIssue) ExampleOutput() map[string]any {
	return map[string]any{
		"type": GetIssuePayloadType,
		"data": map[string]any{
			"id":          "10001",
			"key":         "KAN-1",
			"self":        "https://your-domain.atlassian.net/rest/api/3/issue/10001",
			"summary":     "Make landing page for our AI Delivery Orchestrator",
			"description": "Create a landing page that explains the AI Delivery Orchestrator project.",
			"status":      "In Progress",
			"issueType":   "Task",
			"priority":    "Medium",
			"assignee":    "Milan Popovic",
		},
	}
}

func (g *GetIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "issueKey",
			Label:       "Issue Key",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The Jira issue key to fetch (e.g. KAN-1)",
			Placeholder: "KAN-1",
		},
	}
}

func (g *GetIssue) Setup(ctx core.SetupContext) error {
	spec := GetIssueSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.IssueKey == "" {
		return fmt.Errorf("issueKey is required")
	}

	return nil
}

func (g *GetIssue) Execute(ctx core.ExecutionContext) error {
	spec := GetIssueSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	issue, err := client.GetIssue(spec.IssueKey)
	if err != nil {
		return fmt.Errorf("failed to get issue: %v", err)
	}

	payload := GetIssuePayload{
		ID:   issue.ID,
		Key:  issue.Key,
		Self: issue.Self,
	}

	if fields := issue.Fields; fields != nil {
		if v, ok := fields["summary"].(string); ok {
			payload.Summary = v
		}
		if v, ok := fields["description"].(string); ok {
			payload.Description = v
		}
		if status, ok := fields["status"].(map[string]any); ok {
			if name, ok := status["name"].(string); ok {
				payload.Status = name
			}
		}
		if issueType, ok := fields["issuetype"].(map[string]any); ok {
			if name, ok := issueType["name"].(string); ok {
				payload.IssueType = name
			}
		}
		if priority, ok := fields["priority"].(map[string]any); ok {
			if name, ok := priority["name"].(string); ok {
				payload.Priority = name
			}
		}
		if assignee, ok := fields["assignee"].(map[string]any); ok {
			if name, ok := assignee["displayName"].(string); ok {
				payload.Assignee = name
			}
		}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetIssuePayloadType,
		[]any{payload},
	)
}

func (g *GetIssue) Cancel(ctx core.ExecutionContext) error      { return nil }
func (g *GetIssue) Cleanup(ctx core.SetupContext) error          { return nil }
func (g *GetIssue) Actions() []core.Action                       { return []core.Action{} }
func (g *GetIssue) HandleAction(ctx core.ActionContext) error    { return nil }

func (g *GetIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
