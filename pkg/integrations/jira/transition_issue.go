package jira

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const TransitionIssuePayloadType = "jira.issue.transition"

type TransitionIssue struct{}

type TransitionIssueSpec struct {
	IssueKey     string `json:"issueKey"`
	TransitionID string `json:"transitionId"`
}

func (t *TransitionIssue) Name() string {
	return "jira.transitionIssue"
}

func (t *TransitionIssue) Label() string {
	return "Transition Issue"
}

func (t *TransitionIssue) Description() string {
	return "Move a Jira issue to a different workflow state"
}

func (t *TransitionIssue) Documentation() string {
	return `The Transition Issue component moves an existing Jira issue to a new workflow state.

## Use Cases

- **Start work**: Move a ticket to In Progress when a build kicks off
- **Request review**: Move a ticket to In Review when AI review starts
- **Close ticket**: Move a ticket to Done after successful delivery

## Configuration

- **Issue Key**: The Jira issue key to transition (e.g. APP-42)
- **Transition ID**: The numeric ID of the target transition (board-specific)

To find transition IDs for your board, call:
  GET /rest/api/3/issue/{issueKey}/transitions

## Output

Returns the issue key and the transition ID that was applied.`
}

func (t *TransitionIssue) Icon() string {
	return "jira"
}

func (t *TransitionIssue) Color() string {
	return "blue"
}

func (t *TransitionIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (t *TransitionIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "issueKey",
			Label:       "Issue Key",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The Jira issue key to transition (e.g. APP-42)",
			Placeholder: "APP-42",
		},
		{
			Name:        "transitionId",
			Label:       "Transition ID",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The numeric transition ID for the target state (board-specific)",
			Placeholder: "21",
		},
	}
}

func (t *TransitionIssue) Setup(ctx core.SetupContext) error {
	spec := TransitionIssueSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.IssueKey == "" {
		return fmt.Errorf("issueKey is required")
	}

	if spec.TransitionID == "" {
		return fmt.Errorf("transitionId is required")
	}

	return nil
}

func (t *TransitionIssue) Execute(ctx core.ExecutionContext) error {
	spec := TransitionIssueSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	if err := client.TransitionIssue(spec.IssueKey, spec.TransitionID); err != nil {
		return fmt.Errorf("failed to transition issue: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		TransitionIssuePayloadType,
		[]any{map[string]string{
			"issueKey":     spec.IssueKey,
			"transitionId": spec.TransitionID,
		}},
	)
}

func (t *TransitionIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (t *TransitionIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (t *TransitionIssue) Actions() []core.Action {
	return []core.Action{}
}

func (t *TransitionIssue) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (t *TransitionIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (t *TransitionIssue) Cleanup(ctx core.SetupContext) error {
	return nil
}
