package jira

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const ChangeJiraTicketStatusPayloadType = "jira.issue.transition"

type ChangeJiraTicketStatus struct{}

type ChangeJiraTicketStatusSpec struct {
	TicketKey    string `json:"ticketKey"`
	TransitionID string `json:"transitionId"`
}

func (t *ChangeJiraTicketStatus) Name() string {
	return "jira.changeJiraTicketStatus"
}

func (t *ChangeJiraTicketStatus) Label() string {
	return "Change Ticket Status"
}

func (t *ChangeJiraTicketStatus) Description() string {
	return "Move a Jira ticket to a different workflow status"
}

func (t *ChangeJiraTicketStatus) Documentation() string {
	return `The Change Ticket Status component moves an existing Jira ticket to a new workflow state.

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

func (t *ChangeJiraTicketStatus) Icon() string {
	return "jira"
}

func (t *ChangeJiraTicketStatus) Color() string {
	return "blue"
}

func (t *ChangeJiraTicketStatus) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (t *ChangeJiraTicketStatus) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "ticketKey",
			Label:       "Ticket Key",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The Jira ticket key to transition (e.g. APP-42)",
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

func (t *ChangeJiraTicketStatus) Setup(ctx core.SetupContext) error {
	spec := ChangeJiraTicketStatusSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.TicketKey == "" {
		return fmt.Errorf("ticketKey is required")
	}

	if spec.TransitionID == "" {
		return fmt.Errorf("transitionId is required")
	}

	return nil
}

func (t *ChangeJiraTicketStatus) Execute(ctx core.ExecutionContext) error {
	spec := ChangeJiraTicketStatusSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	if err := client.TransitionIssue(spec.TicketKey, spec.TransitionID); err != nil {
		return fmt.Errorf("failed to transition issue: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		ChangeJiraTicketStatusPayloadType,
		[]any{map[string]string{
			"ticketKey":    spec.TicketKey,
			"transitionId": spec.TransitionID,
		}},
	)
}

func (t *ChangeJiraTicketStatus) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (t *ChangeJiraTicketStatus) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (t *ChangeJiraTicketStatus) Actions() []core.Action {
	return []core.Action{}
}

func (t *ChangeJiraTicketStatus) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (t *ChangeJiraTicketStatus) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (t *ChangeJiraTicketStatus) Cleanup(ctx core.SetupContext) error {
	return nil
}
