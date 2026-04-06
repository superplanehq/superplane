package digitalocean

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type AttachKnowledgeBase struct{}

type AttachKnowledgeBaseSpec struct {
	AgentID         string `json:"agentId" mapstructure:"agentId"`
	KnowledgeBaseID string `json:"knowledgeBaseId" mapstructure:"knowledgeBaseId"`
}

func (a *AttachKnowledgeBase) Name() string {
	return "digitalocean.attachKnowledgeBase"
}

func (a *AttachKnowledgeBase) Label() string {
	return "Attach Knowledge Base"
}

func (a *AttachKnowledgeBase) Description() string {
	return "Attach a knowledge base to a DigitalOcean Gradient AI agent"
}

func (a *AttachKnowledgeBase) Documentation() string {
	return `The Attach Knowledge Base component connects a knowledge base to an existing Gradient AI agent, enabling the agent to use it for retrieval-augmented generation (RAG).

## Use Cases

- **Post-creation wiring**: After creating a new knowledge base, attach it to an agent to make it immediately available
- **Blue/green KB deployment**: Attach a newly indexed knowledge base to an agent as part of a promotion pipeline
- **Multi-KB agents**: Add additional knowledge bases to an agent that already has others attached

## Configuration

- **Agent**: The agent to attach the knowledge base to (required)
- **Knowledge Base**: The knowledge base to attach — only shows knowledge bases not already attached to the selected agent (required)

## Output

Returns confirmation of the attachment including:
- **agentId**: UUID of the agent
- **knowledgeBaseId**: UUID of the attached knowledge base`
}

func (a *AttachKnowledgeBase) Icon() string {
	return "bot"
}

func (a *AttachKnowledgeBase) Color() string {
	return "blue"
}

func (a *AttachKnowledgeBase) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (a *AttachKnowledgeBase) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "agentId",
			Label:       "Agent UUID",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select an agent",
			Description: "The agent to attach the knowledge base to",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "agent",
				},
			},
		},
		{
			Name:        "knowledgeBaseId",
			Label:       "Knowledge Base UUID",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select a knowledge base to attach",
			Description: "The knowledge base to attach. Only knowledge bases not already attached to the selected agent are shown.",
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

func (a *AttachKnowledgeBase) Setup(ctx core.SetupContext) error {
	spec := AttachKnowledgeBaseSpec{}
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

func (a *AttachKnowledgeBase) Execute(ctx core.ExecutionContext) error {
	spec := AttachKnowledgeBaseSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	if err := client.AttachKnowledgeBase(spec.AgentID, spec.KnowledgeBaseID); err != nil {
		return fmt.Errorf("failed to attach knowledge base: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.knowledge_base.attached",
		[]any{map[string]any{
			"agentId":         spec.AgentID,
			"knowledgeBaseId": spec.KnowledgeBaseID,
		}},
	)
}

func (a *AttachKnowledgeBase) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (a *AttachKnowledgeBase) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (a *AttachKnowledgeBase) Actions() []core.Action {
	return []core.Action{}
}

func (a *AttachKnowledgeBase) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions defined")
}

func (a *AttachKnowledgeBase) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (a *AttachKnowledgeBase) Cleanup(ctx core.SetupContext) error {
	return nil
}

// KBNodeMetadata stores metadata about an agent + knowledge base node for display in the UI
type KBNodeMetadata struct {
	AgentID           string `json:"agentId" mapstructure:"agentId"`
	AgentName         string `json:"agentName" mapstructure:"agentName"`
	KnowledgeBaseID   string `json:"knowledgeBaseId" mapstructure:"knowledgeBaseId"`
	KnowledgeBaseName string `json:"knowledgeBaseName" mapstructure:"knowledgeBaseName"`
}

// resolveKBNodeMetadata fetches the agent and knowledge base names from the API and stores them in metadata
func resolveKBNodeMetadata(ctx core.SetupContext, agentID, kbID string) error {
	meta := KBNodeMetadata{
		AgentID:         agentID,
		KnowledgeBaseID: kbID,
	}

	isAgentExpr := strings.Contains(agentID, "{{")
	isKbExpr := strings.Contains(kbID, "{{")

	if isAgentExpr {
		meta.AgentName = agentID
	}
	if isKbExpr {
		meta.KnowledgeBaseName = kbID
	}

	var existing KBNodeMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &existing)
	if err == nil &&
		existing.AgentID == agentID && existing.AgentName != "" &&
		existing.KnowledgeBaseID == kbID && existing.KnowledgeBaseName != "" {
		return nil
	}

	if isAgentExpr && isKbExpr {
		return ctx.Metadata.Set(meta)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	if !isAgentExpr {
		agent, err := client.GetAgent(agentID)
		if err != nil {
			return fmt.Errorf("failed to fetch agent %q: %w", agentID, err)
		}
		meta.AgentName = agent.Name
	}

	if !isKbExpr {
		if kb, err := client.GetKnowledgeBase(kbID); err == nil {
			meta.KnowledgeBaseName = kb.Name
		}
	}

	return ctx.Metadata.Set(meta)
}
