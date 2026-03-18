package perplexity

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const AgentPayloadType = "perplexity.agent.response"

type runAgent struct{}

type runAgentSpec struct {
	ModelSource  string `mapstructure:"modelSource"`
	Preset       string `mapstructure:"preset"`
	Model        string `mapstructure:"model"`
	Input        string `mapstructure:"input"`
	Instructions string `mapstructure:"instructions"`
	WebSearch    bool   `mapstructure:"webSearch"`
	FetchURL     bool   `mapstructure:"fetchUrl"`
}

type agentPayload struct {
	ID        string         `json:"id"`
	Model     string         `json:"model"`
	Status    string         `json:"status"`
	Text      string         `json:"text"`
	Citations []citation     `json:"citations"`
	Usage     *AgentUsage    `json:"usage,omitempty"`
	Response  *AgentResponse `json:"response"`
}

type citation struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

func (c *runAgent) Name() string {
	return "perplexity.runAgent"
}

func (c *runAgent) Label() string {
	return "Run Agent"
}

func (c *runAgent) Description() string {
	return "Run a Perplexity AI agent with web search and URL fetching capabilities"
}

func (c *runAgent) Documentation() string {
	return `The Run Agent component uses Perplexity's Agent API to run AI agents that can search the web and fetch URLs.

## Use Cases

- **Research and synthesis**: Ask complex questions that require gathering and synthesizing information from multiple sources
- **Automated analysis**: Run AI-powered analysis on web content
- **Content generation with citations**: Generate text grounded in real-time web sources

## Configuration

- **Preset**: Agent preset to use (fast-search, pro-search, deep-research, advanced-deep-research). When set, model is ignored.
- **Model**: Model to use when no preset is specified
- **Input**: The prompt or question for the agent (supports expressions)
- **Instructions**: Optional system-level instructions
- **Web Search**: Enable the web_search tool (default: true)
- **Fetch URL**: Enable the fetch_url tool (default: true)

## Output

Returns the agent response including:
- **text**: The generated text response
- **citations**: Source citations from web results
- **model**: The model used
- **usage**: Token and cost usage information`
}

func (c *runAgent) Icon() string {
	return "bot"
}

func (c *runAgent) Color() string {
	return "teal"
}

func (c *runAgent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *runAgent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "modelSource",
			Label:       "Model Source",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "preset",
			Description: "Choose between a preset or a specific model",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Preset", Value: "preset"},
						{Label: "Custom Model", Value: "model"},
					},
				},
			},
		},
		{
			Name:        "preset",
			Label:       "Preset",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Default:     "pro-search",
			Description: "Agent preset to use",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "agent-preset",
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "modelSource", Values: []string{"preset"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "modelSource", Values: []string{"preset"}},
			},
		},
		{
			Name:        "model",
			Label:       "Model",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Default:     "sonar-pro",
			Placeholder: "Select a model",
			Description: "Model to use",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "agent-model",
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "modelSource", Values: []string{"model"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "modelSource", Values: []string{"model"}},
			},
		},
		{
			Name:        "input",
			Label:       "Input",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Placeholder: "Enter the prompt or question",
			Description: "The prompt or question for the agent (supports expressions)",
		},
		{
			Name:        "instructions",
			Label:       "Instructions",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Placeholder: "Optional system-level instructions",
			Description: "System-level instructions for the agent",
		},
		{
			Name:        "webSearch",
			Label:       "Web Search",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Enable the web_search tool",
		},
		{
			Name:        "fetchUrl",
			Label:       "Fetch URL",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Enable the fetch_url tool",
		},
	}
}

func (c *runAgent) Setup(ctx core.SetupContext) error {
	spec := runAgentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.ModelSource == "preset" && spec.Preset == "" {
		return fmt.Errorf("preset is required")
	}

	if spec.ModelSource == "model" && spec.Model == "" {
		return fmt.Errorf("model is required")
	}

	if spec.Input == "" {
		return fmt.Errorf("input is required")
	}

	return nil
}

func (c *runAgent) Execute(ctx core.ExecutionContext) error {
	spec := runAgentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.ModelSource == "preset" && spec.Preset == "" {
		return fmt.Errorf("preset is required")
	}

	if spec.ModelSource == "model" && spec.Model == "" {
		return fmt.Errorf("model is required")
	}

	if spec.Input == "" {
		return fmt.Errorf("input is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	req := AgentRequest{
		Input: spec.Input,
	}

	if spec.ModelSource == "preset" {
		req.Preset = spec.Preset
	} else {
		req.Model = spec.Model
	}

	if spec.Instructions != "" {
		req.Instructions = spec.Instructions
	}

	tools := make([]AgentTool, 0, 2)
	if spec.WebSearch {
		tools = append(tools, AgentTool{Type: "web_search"})
	}
	if spec.FetchURL {
		tools = append(tools, AgentTool{Type: "fetch_url"})
	}
	if len(tools) > 0 {
		req.Tools = tools
	}

	response, err := client.CreateAgentResponse(req)
	if err != nil {
		return err
	}

	text := extractAgentText(response)
	citations := extractCitations(response)

	payload := agentPayload{
		ID:        response.ID,
		Model:     response.Model,
		Status:    response.Status,
		Text:      text,
		Citations: citations,
		Usage:     response.Usage,
		Response:  response,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		AgentPayloadType,
		[]any{payload},
	)
}

func (c *runAgent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *runAgent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *runAgent) Actions() []core.Action {
	return []core.Action{}
}

func (c *runAgent) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *runAgent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *runAgent) Cleanup(ctx core.SetupContext) error {
	return nil
}

func extractAgentText(response *AgentResponse) string {
	if response == nil {
		return ""
	}

	var builder strings.Builder
	for _, output := range response.Output {
		for _, content := range output.Content {
			if content.Type != "" && content.Type != "output_text" && content.Type != "text" {
				continue
			}

			if content.Text == "" {
				continue
			}

			if builder.Len() > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString(content.Text)
		}
	}

	return builder.String()
}

func extractCitations(response *AgentResponse) []citation {
	if response == nil {
		return nil
	}

	var citations []citation
	for _, output := range response.Output {
		for _, content := range output.Content {
			for _, annotation := range content.Annotations {
				if annotation.URL != "" {
					citations = append(citations, citation{
						Type: annotation.Type,
						URL:  annotation.URL,
					})
				}
			}
		}
	}

	return citations
}
