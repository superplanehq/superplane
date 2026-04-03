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

type DetachKnowledgeBase struct{}

type DetachKnowledgeBaseSpec struct {
	AgentID         string `json:"agentId" mapstructure:"agentId"`
	KnowledgeBaseID string `json:"knowledgeBaseId" mapstructure:"knowledgeBaseId"`
}

func (d *DetachKnowledgeBase) Name() string {
	return "digitalocean.detachKnowledgeBase"
}

func (d *DetachKnowledgeBase) Label() string {
	return "Detach Knowledge Base"
}

func (d *DetachKnowledgeBase) Description() string {
	return "Detach a knowledge base from a DigitalOcean Gradient AI agent"
}

func (d *DetachKnowledgeBase) Documentation() string {
	return `The Detach Knowledge Base component removes a knowledge base from an existing Gradient AI agent.

## Use Cases

- **Rollback**: Remove a knowledge base that is causing poor agent responses
- **Cleanup**: Detach an outdated knowledge base before attaching a freshly indexed one
- **Rotation**: As part of a blue/green pipeline, detach the old knowledge base after the new one is verified

## Configuration

- **Agent**: The agent to detach the knowledge base from (required)
- **Knowledge Base**: The knowledge base to detach — only shows knowledge bases currently attached to the selected agent (required)

## Output

Returns confirmation of the detachment including:
- **agentId**: UUID of the agent
- **knowledgeBaseId**: UUID of the detached knowledge base`
}

func (d *DetachKnowledgeBase) Icon() string {
	return "bot"
}

func (d *DetachKnowledgeBase) Color() string {
	return "blue"
}

func (d *DetachKnowledgeBase) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DetachKnowledgeBase) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "agentId",
			Label:       "Agent",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select an agent",
			Description: "The agent to detach the knowledge base from",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "agent",
				},
			},
		},
		{
			Name:        "knowledgeBaseId",
			Label:       "Knowledge Base",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select a knowledge base to detach",
			Description: "The knowledge base to detach. Only knowledge bases currently attached to the selected agent are shown.",
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
	}
}

func (d *DetachKnowledgeBase) Setup(ctx core.SetupContext) error {
	spec := DetachKnowledgeBaseSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.AgentID == "" {
		return errors.New("agentId is required")
	}

	if spec.KnowledgeBaseID == "" {
		return errors.New("knowledgeBaseId is required")
	}

	if err := resolveKBNodeMetadata(ctx, spec.AgentID, spec.KnowledgeBaseID); err != nil {
		return fmt.Errorf("error resolving metadata: %v", err)
	}

	return nil
}

func (d *DetachKnowledgeBase) Execute(ctx core.ExecutionContext) error {
	spec := DetachKnowledgeBaseSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	if err := client.DetachKnowledgeBase(spec.AgentID, spec.KnowledgeBaseID); err != nil {
		return fmt.Errorf("failed to detach knowledge base: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.knowledge_base.detached",
		[]any{map[string]any{
			"agentId":         spec.AgentID,
			"knowledgeBaseId": spec.KnowledgeBaseID,
		}},
	)
}

func (d *DetachKnowledgeBase) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DetachKnowledgeBase) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DetachKnowledgeBase) Actions() []core.Action {
	return []core.Action{}
}

func (d *DetachKnowledgeBase) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions defined")
}

func (d *DetachKnowledgeBase) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DetachKnowledgeBase) Cleanup(ctx core.SetupContext) error {
	return nil
}
