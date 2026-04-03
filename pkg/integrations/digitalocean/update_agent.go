package digitalocean

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateAgent struct{}

type UpdateAgentSpec struct {
	AgentID            string `json:"agentId" mapstructure:"agentId"`
	OldKnowledgeBaseID string `json:"oldKnowledgeBaseId" mapstructure:"oldKnowledgeBaseId"`
	NewKnowledgeBaseID string `json:"newKnowledgeBaseId" mapstructure:"newKnowledgeBaseId"`
}

func (u *UpdateAgent) Name() string {
	return "digitalocean.updateAgent"
}

func (u *UpdateAgent) Label() string {
	return "Update Agent"
}

func (u *UpdateAgent) Description() string {
	return "Swap the knowledge base attached to a DigitalOcean Gradient AI agent"
}

func (u *UpdateAgent) Documentation() string {
	return `The Update Agent component swaps the knowledge base attached to an existing Gradient AI agent.

## Use Cases

- **Blue/green KB deployment**: Point the agent at a newly indexed knowledge base after validation passes
- **Rollback**: Revert an agent back to a previous knowledge base if evaluation fails
- **Environment promotion**: Promote a staging agent to use the production knowledge base

## Configuration

- **Agent**: The agent to update (required)
- **Current Knowledge Base**: The knowledge base currently attached to the agent that will be detached (required)
- **New Knowledge Base**: The knowledge base to attach to the agent (required)

## Output

Returns confirmation of the swap including:
- **agentId**: The ID of the updated agent
- **previousKnowledgeBaseId**: The knowledge base that was detached
- **newKnowledgeBaseId**: The knowledge base that was attached

## Notes

- If the current knowledge base is already detached, the component proceeds and attaches the new one
- The agent and both knowledge bases must exist in the same DigitalOcean account`
}

func (u *UpdateAgent) Icon() string {
	return "bot"
}

func (u *UpdateAgent) Color() string {
	return "blue"
}

func (u *UpdateAgent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (u *UpdateAgent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "agentId",
			Label:       "Agent",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select an agent",
			Description: "The agent to update",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "agent",
				},
			},
		},
		{
			Name:        "oldKnowledgeBaseId",
			Label:       "Current Knowledge Base",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select the knowledge base to detach",
			Description: "The knowledge base currently attached to the agent that will be replaced",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "agent_knowledge_base",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "agentId",
							ValueFrom: &configuration.ParameterValueFrom{Field: "agentId"},
						},
					},
				},
			},
		},
		{
			Name:        "newKnowledgeBaseId",
			Label:       "New Knowledge Base",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select the knowledge base to attach",
			Description: "The knowledge base to attach to the agent",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "agent_available_knowledge_base",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "agentId",
							ValueFrom: &configuration.ParameterValueFrom{Field: "agentId"},
						},
					},
				},
			},
		},
	}
}

func (u *UpdateAgent) Setup(ctx core.SetupContext) error {
	spec := UpdateAgentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.AgentID == "" {
		return errors.New("agentId is required")
	}

	if spec.OldKnowledgeBaseID == "" {
		return errors.New("oldKnowledgeBaseId is required")
	}

	if spec.NewKnowledgeBaseID == "" {
		return errors.New("newKnowledgeBaseId is required")
	}

	if spec.OldKnowledgeBaseID == spec.NewKnowledgeBaseID {
		return errors.New("oldKnowledgeBaseId and newKnowledgeBaseId must be different")
	}

	if err := resolveAgentMetadata(ctx, spec.AgentID, spec.OldKnowledgeBaseID, spec.NewKnowledgeBaseID); err != nil {
		return fmt.Errorf("error resolving agent metadata: %v", err)
	}

	return nil
}

func (u *UpdateAgent) Execute(ctx core.ExecutionContext) error {
	spec := UpdateAgentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.DetachKnowledgeBase(spec.AgentID, spec.OldKnowledgeBaseID)
	if err != nil {
		if doErr, ok := err.(*DOAPIError); !ok || doErr.StatusCode != http.StatusNotFound {
			return fmt.Errorf("failed to detach knowledge base: %v", err)
		}
	}

	if err := client.AttachKnowledgeBase(spec.AgentID, spec.NewKnowledgeBaseID); err != nil {
		return fmt.Errorf("failed to attach knowledge base: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.agent.updated",
		[]any{map[string]any{
			"agentId":                 spec.AgentID,
			"previousKnowledgeBaseId": spec.OldKnowledgeBaseID,
			"newKnowledgeBaseId":      spec.NewKnowledgeBaseID,
		}},
	)
}

func (u *UpdateAgent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (u *UpdateAgent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (u *UpdateAgent) Actions() []core.Action {
	return []core.Action{}
}

func (u *UpdateAgent) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions defined")
}

func (u *UpdateAgent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (u *UpdateAgent) Cleanup(ctx core.SetupContext) error {
	return nil
}
