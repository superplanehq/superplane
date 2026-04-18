package claude

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const PromptAIPayloadType = "claude.promptAI"

type PromptAI struct{}

type PromptAISpec struct {
	Model        string `json:"model"`
	Prompt       string `json:"prompt"`
	AlertDetails string `json:"alertDetails"`
}

type PromptAIPayload struct {
	Text string `json:"text"`
}

func (c *PromptAI) Name() string {
	return "claude.promptAI"
}

func (c *PromptAI) Label() string {
	return "Prompt AI"
}

func (c *PromptAI) Description() string {
	return "Send a prompt and optional alert details to Claude and receive a concise response"
}

func (c *PromptAI) Documentation() string {
	return "Combines a static instruction prompt with dynamic alert data and sends it to Claude."
}

func (c *PromptAI) Icon() string {
	return "message-circle"
}

func (c *PromptAI) Color() string {
	return "purple"
}

func (c *PromptAI) ExampleOutput() map[string]any {
	return map[string]any{
		"text": "The alert indicates high CPU usage on the worker nodes...",
	}
}

func (c *PromptAI) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *PromptAI) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "model",
			Label:       "Model",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Default:     "claude-opus-4-6",
			Placeholder: "Select a Claude model",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "model",
				},
			},
		},
		{
			Name:        "prompt",
			Label:       "Prompt",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Placeholder: "e.g. Analyze this alert and suggest remediation steps",
			Description: "The instruction for Claude",
		},
		{
			Name:        "alertDetails",
			Label:       "Alert Details",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Placeholder: "Dynamic alert data from upstream nodes",
			Description: "Optional context (e.g. Grafana alert payload) appended to the prompt",
		},
	}
}

func (c *PromptAI) Setup(ctx core.SetupContext) error {
	spec := PromptAISpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}
	if spec.Model == "" {
		return fmt.Errorf("model is required")
	}
	if spec.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}
	return nil
}

func (c *PromptAI) Execute(ctx core.ExecutionContext) error {
	spec := PromptAISpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.Model == "" {
		return fmt.Errorf("model is required")
	}
	if spec.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	combined := spec.Prompt
	if spec.AlertDetails != "" {
		combined = fmt.Sprintf("%s\n\nContext:\n%s\n\nOdgovori koncizno.", spec.Prompt, spec.AlertDetails)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	resp, err := client.CreateMessage(CreateMessageRequest{
		Model:     spec.Model,
		MaxTokens: 4096,
		Messages:  []Message{{Role: "user", Content: combined}},
	})
	if err != nil {
		return fmt.Errorf("claude api error: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		PromptAIPayloadType,
		[]any{PromptAIPayload{Text: extractMessageText(resp)}},
	)
}

func (c *PromptAI) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *PromptAI) Actions() []core.Action {
	return []core.Action{}
}

func (c *PromptAI) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *PromptAI) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *PromptAI) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *PromptAI) Cleanup(ctx core.SetupContext) error {
	return nil
}
