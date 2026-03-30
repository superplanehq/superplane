package jira

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const GetTicketDataPayloadType = "jira.issue.fetched"

type GetTicketData struct{}

type GetTicketDataSpec struct {
	IssueKey string `json:"issueKey"`
}

type GetTicketDataPayload struct {
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

func (g *GetTicketData) Name() string { return "jira.getTicketData" }

func (g *GetTicketData) Label() string { return "Get Ticket Data" }

func (g *GetTicketData) Description() string {
	return "Fetch a Jira ticket and emit its fields for use in downstream nodes"
}

func (g *GetTicketData) Documentation() string {
	return `The Get Ticket Data component fetches a Jira ticket by its key and emits its fields.

## Use Cases

- **Feed task to AI agent**: Pass the ticket summary and description to a Claude Code Agent
- **Branch naming**: Use the ticket key as the branch name
- **Conditional routing**: Route based on issue type or priority

## Configuration

- **Issue Key**: The Jira ticket key to fetch (e.g. KAN-1). Supports expressions.

## Output

- **key**: Ticket key (e.g. KAN-1)
- **summary**: Ticket title
- **description**: Ticket description (plain text)
- **status**: Current status name
- **issueType**: Issue type (Bug, Story, Task, etc.)
- **priority**: Priority name
- **assignee**: Assignee display name`
}

func (g *GetTicketData) Icon() string  { return "jira" }
func (g *GetTicketData) Color() string { return "blue" }

func (g *GetTicketData) ExampleOutput() map[string]any {
	return map[string]any{
		"type": GetTicketDataPayloadType,
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

func (g *GetTicketData) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetTicketData) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "issueKey",
			Label:       "Issue Key",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The Jira ticket key to fetch (e.g. KAN-1)",
			Placeholder: "KAN-1",
		},
	}
}

func (g *GetTicketData) Setup(ctx core.SetupContext) error {
	spec := GetTicketDataSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.IssueKey == "" {
		return fmt.Errorf("issueKey is required")
	}

	return nil
}

func (g *GetTicketData) Execute(ctx core.ExecutionContext) error {
	spec := GetTicketDataSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	issue, err := client.GetIssue(spec.IssueKey)
	if err != nil {
		return fmt.Errorf("failed to get ticket: %v", err)
	}

	payload := GetTicketDataPayload{
		ID:   issue.ID,
		Key:  issue.Key,
		Self: issue.Self,
	}

	if fields := issue.Fields; fields != nil {
		if v, ok := fields["summary"].(string); ok {
			payload.Summary = v
		}
		payload.Description = extractADFText(fields["description"])
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
		GetTicketDataPayloadType,
		[]any{payload},
	)
}

func (g *GetTicketData) Cancel(ctx core.ExecutionContext) error      { return nil }
func (g *GetTicketData) Cleanup(ctx core.SetupContext) error          { return nil }
func (g *GetTicketData) Actions() []core.Action                       { return []core.Action{} }
func (g *GetTicketData) HandleAction(ctx core.ActionContext) error    { return nil }

func (g *GetTicketData) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetTicketData) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

// extractADFText converts an Atlassian Document Format (ADF) object to plain text.
// Jira REST API v3 returns description as ADF, not a plain string.
func extractADFText(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	node, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	var sb strings.Builder
	walkADF(node, &sb)
	return strings.TrimSpace(sb.String())
}

func walkADF(node map[string]any, sb *strings.Builder) {
	if nodeType, _ := node["type"].(string); nodeType == "text" {
		if text, ok := node["text"].(string); ok {
			sb.WriteString(text)
		}
		return
	}
	content, ok := node["content"].([]any)
	if !ok {
		return
	}
	for _, child := range content {
		childNode, ok := child.(map[string]any)
		if !ok {
			continue
		}
		walkADF(childNode, sb)
		if t, _ := childNode["type"].(string); t == "paragraph" || t == "heading" || t == "bulletList" || t == "orderedList" || t == "listItem" {
			sb.WriteString("\n")
		}
	}
}
