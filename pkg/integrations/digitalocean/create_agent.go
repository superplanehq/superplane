package digitalocean

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	createAgentPollInterval = 10 * time.Second
	createAgentTimeout      = 10 * time.Minute
	agentPayloadType        = "digitalocean.gradientai.agent"
)

type CreateAgent struct{}

type CreateAgentSpec struct {
	// Basic
	Name        string `mapstructure:"name"`
	Instruction string `mapstructure:"instruction"`

	// Model
	ModelProvider string `mapstructure:"modelProvider"`
	ModelUUID     string `mapstructure:"modelUUID"`

	// Workspace
	WorkspaceSource string `mapstructure:"workspaceSource"`
	WorkspaceUUID   string `mapstructure:"workspaceUUID"`
	WorkspaceName   string `mapstructure:"workspaceName"`

	// Region & tags
	Region    string   `mapstructure:"region"`
	Tags      []string `mapstructure:"tags"`
	ProjectID string   `mapstructure:"projectID"`

	// Settings
	UseDefaultSettings bool    `mapstructure:"useDefaultSettings"`
	MaxTokens          int     `mapstructure:"maxTokens"`
	Temperature        float64 `mapstructure:"temperature"`
	TopP               float64 `mapstructure:"topP"`
	K                  int     `mapstructure:"k"`
	RetrievalMethod    string  `mapstructure:"retrievalMethod"`
	ProvideCitations   bool    `mapstructure:"provideCitations"`

	// Optional resources
	KnowledgeBases []string `mapstructure:"knowledgeBases"`
	Guardrails     []string `mapstructure:"guardrails"`
	AgentRoutes    []string `mapstructure:"agentRoutes"`
}

type CreateAgentExecutionMetadata struct {
	AgentUUID string `json:"agentUUID" mapstructure:"agentUUID"`
	StartedAt int64  `json:"startedAt" mapstructure:"startedAt"`
	APIKey    string `json:"apiKey" mapstructure:"apiKey"`
}

func (c *CreateAgent) Name() string {
	return "digitalocean.createAgent"
}

func (c *CreateAgent) Label() string {
	return "Create Agent"
}

func (c *CreateAgent) Description() string {
	return "Create and deploy a new GradientAI agent"
}

func (c *CreateAgent) Documentation() string {
	return `The Create Agent component creates and deploys a new GradientAI agent.

## Use Cases

- **Release agents**: Provision a dedicated AI assistant per release version
- **App agents**: Create an agent for each deployed application
- **Service agents**: Provision a support agent per microservice

## Configuration

- **Name**: The agent name
- **Instructions**: System prompt that defines the agent's behavior
- **Model Provider**: The provider of the AI model (Anthropic, OpenAI, etc.)
- **Model**: The AI model to use
- **Workspace**: Existing or new workspace to organize the agent
- **Knowledge Bases**: Attach existing knowledge bases (optional)
- **Guardrails**: Attach guardrails for content safety (optional)
- **Agent Routes**: Child agents to route requests to (optional)

## Provider API Keys

If the selected model requires a third-party provider API key (e.g. Anthropic or OpenAI),
set the corresponding key in the **DigitalOcean integration configuration** — not here.
The component will automatically use the key stored in the integration when creating the agent.

## Output

Returns the deployed agent including:
- **uuid**: Agent identifier
- **url**: Agent endpoint URL
- **deployment.status**: Deployment status`
}

func (c *CreateAgent) Icon() string {
	return "bot"
}

func (c *CreateAgent) Color() string {
	return "blue"
}

func (c *CreateAgent) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateAgent) Configuration() []configuration.Field {
	minTokens := 256
	maxTokens := 64000

	return []configuration.Field{
		// Basic
		{
			Name:        "name",
			Label:       "Agent Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g. release-v2-assistant",
			Description: "The name of the agent",
		},
		{
			Name:        "instruction",
			Label:       "Instructions",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Placeholder: "You are an assistant that...",
			Description: "System prompt that defines the agent's behavior and capabilities",
		},
		// Model
		{
			Name:        "modelProvider",
			Label:       "Model Provider",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select a provider",
			Description: "The AI model provider",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "gradientai_model_provider",
				},
			},
		},
		{
			Name:        "modelUUID",
			Label:       "Model",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select a model",
			Description: "The AI model to use. Select a provider first.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "gradientai_model",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "provider",
							ValueFrom: &configuration.ParameterValueFrom{Field: "modelProvider"},
						},
					},
				},
			},
		},

		// Workspace
		{
			Name:        "workspaceSource",
			Label:       "Workspace",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "existing",
			Description: "Use an existing workspace or create a new one",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Use existing workspace", Value: "existing"},
						{Label: "Create new workspace", Value: "new"},
					},
				},
			},
		},
		{
			Name:        "workspaceUUID",
			Label:       "Select Workspace",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Placeholder: "Select a workspace",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "gradientai_workspace",
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "workspaceSource", Values: []string{"existing"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "workspaceSource", Values: []string{"existing"}},
			},
		},
		{
			Name:        "workspaceName",
			Label:       "Workspace Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g. release-v2",
			Description: "Name for the new workspace",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "workspaceSource", Values: []string{"new"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "workspaceSource", Values: []string{"new"}},
			},
		},

		{
			Name:        "region",
			Label:       "Region",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select a region",
			Description: "The region where the agent will be deployed",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "gradientai_region",
				},
			},
		},
		{
			Name:        "projectID",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select a project",
			Description: "The DigitalOcean project this agent will belong to",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "do_project",
				},
			},
		},
		{
			Name:      "tags",
			Label:     "Tags",
			Type:      configuration.FieldTypeList,
			Required:  false,
			Togglable: true,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Tag",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
			Description: "Tags to apply to the agent",
		},

		// Settings toggle
		{
			Name:        "useDefaultSettings",
			Label:       "Use Default Settings",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Use default model settings. Disable to configure temperature, max tokens and more.",
		},
		{
			Name:        "maxTokens",
			Label:       "Max Tokens",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     2048,
			Description: "Maximum number of tokens in the agent response",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: &minTokens,
					Max: &maxTokens,
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "useDefaultSettings", Values: []string{"false"}},
			},
		},
		{
			Name:        "temperature",
			Label:       "Temperature",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     0.7,
			Description: "Controls randomness in responses (0.0 – 1.0)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "useDefaultSettings", Values: []string{"false"}},
			},
		},
		{
			Name:        "topP",
			Label:       "Top P",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     0.9,
			Description: "Controls diversity of responses (0.1 – 1.0)",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "useDefaultSettings", Values: []string{"false"}},
			},
		},
		{
			Name:        "k",
			Label:       "Top K",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     10,
			Description: "Number of knowledge base results to consider",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "useDefaultSettings", Values: []string{"false"}},
			},
		},
		{
			Name:        "retrievalMethod",
			Label:       "Retrieval Method",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "RETRIEVAL_METHOD_NONE",
			Description: "How queries are processed before searching the knowledge base",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "None", Value: "RETRIEVAL_METHOD_NONE"},
						{Label: "Rewrite", Value: "RETRIEVAL_METHOD_REWRITE"},
						{Label: "Step Back", Value: "RETRIEVAL_METHOD_STEP_BACK"},
						{Label: "Sub Queries", Value: "RETRIEVAL_METHOD_SUB_QUERIES"},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "useDefaultSettings", Values: []string{"false"}},
			},
		},
		{
			Name:        "provideCitations",
			Label:       "Include Citations",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Include citations from referenced knowledge base content in responses",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "useDefaultSettings", Values: []string{"false"}},
			},
		},

		// Optional resources
		{
			Name:        "knowledgeBases",
			Label:       "Knowledge Bases",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Placeholder: "Select knowledge bases",
			Description: "Knowledge bases to attach to the agent",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "gradientai_knowledge_base",
					Multi: true,
				},
			},
		},
		{
			Name:        "guardrails",
			Label:       "Guardrails",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Placeholder: "Select guardrails",
			Description: "Guardrails to attach to the agent for content safety",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "gradientai_guardrail",
					Multi: true,
				},
			},
		},
		{
			Name:        "agentRoutes",
			Label:       "Agent Routes",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Togglable:   true,
			Placeholder: "Select agents",
			Description: "Child agents this agent can route requests to",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "gradientai_agent",
					Multi: true,
				},
			},
		},
	}
}

func (c *CreateAgent) Setup(ctx core.SetupContext) error {
	spec := CreateAgentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.Name == "" {
		return fmt.Errorf("name is required")
	}

	if spec.Instruction == "" {
		return fmt.Errorf("instruction is required")
	}

	if spec.ModelUUID == "" {
		return fmt.Errorf("model is required")
	}

	if spec.WorkspaceSource == "existing" && spec.WorkspaceUUID == "" {
		return fmt.Errorf("workspace is required")
	}

	if spec.WorkspaceSource == "new" && spec.WorkspaceName == "" {
		return fmt.Errorf("workspace name is required")
	}

	if spec.Region == "" {
		return fmt.Errorf("region is required")
	}

	return nil
}

func (c *CreateAgent) Execute(ctx core.ExecutionContext) (err error) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			err = fmt.Errorf("panic: %v\nstack:\n%s", r, buf[:n])
		}
	}()
	spec := CreateAgentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	// 1. Create workspace if needed
	workspaceUUID := spec.WorkspaceUUID
	if spec.WorkspaceSource == "new" {
		workspace, err := client.CreateGradientAIWorkspace(spec.WorkspaceName)
		if err != nil {
			return fmt.Errorf("failed to create workspace: %v", err)
		}
		workspaceUUID = workspace.UUID
	}

	// 2. Register provider API key in GradientAI if needed, then build the create request
	req := CreateGradientAIAgentRequest{
		Name:           spec.Name,
		Instruction:    spec.Instruction,
		ModelUUID:      spec.ModelUUID,
		WorkspaceUUID:  workspaceUUID,
		KnowledgeBases: spec.KnowledgeBases,
		Region:         spec.Region,
		Tags:           spec.Tags,
		ProjectID:      spec.ProjectID,
	}

	// Register the provider API key from the integration configuration if present.
	// We use spec.ModelProvider (explicitly selected by the user in the UI) to
	// determine which key to use. Keys are stored at the integration level so that
	// users don't need to re-enter them for every agent component.
	//
	// Only the provider-specific field (anthropic_key_uuid / open_ai_key_uuid) is
	// set here. model_provider_key_uuid is a separate resource created via
	// /v2/gen-ai/model_provider_keys and must NOT be set to an anthropic/openai
	// key UUID — doing so causes a 404 because the API looks up a
	// model_provider_key resource that doesn't exist.
	provider := strings.ToLower(spec.ModelProvider)
	switch {
	case strings.Contains(provider, "anthropic"):
		anthropicKeyRaw, err := ctx.Integration.GetConfig("anthropicKey")
		if err == nil && len(anthropicKeyRaw) > 0 {
			anthropicKey, err := client.CreateGradientAIAnthropicKey(spec.Name, string(anthropicKeyRaw), spec.ProjectID)
			if err != nil {
				return fmt.Errorf("failed to register Anthropic API key: %v", err)
			}
			req.AnthropicKeyUUID = anthropicKey.UUID
		}
	case strings.Contains(provider, "openai"):
		openAIKeyRaw, err := ctx.Integration.GetConfig("openAIKey")
		if err == nil && len(openAIKeyRaw) > 0 {
			openaiKey, err := client.CreateGradientAIOpenAIKey(spec.Name, string(openAIKeyRaw), spec.ProjectID)
			if err != nil {
				return fmt.Errorf("failed to register OpenAI API key: %v", err)
			}
			req.OpenAIKeyUUID = openaiKey.UUID
		}
	}

	// 4. Create the agent
	agent, err := client.CreateGradientAIAgent(req)
	if err != nil {
		return fmt.Errorf("failed to create agent: %v", err)
	}

	// 5. Update settings if custom settings are requested
	if !spec.UseDefaultSettings {
		_, err = client.UpdateGradientAIAgent(agent.UUID, UpdateGradientAIAgentRequest{
			Name:             spec.Name,
			Instruction:      spec.Instruction,
			MaxTokens:        spec.MaxTokens,
			Temperature:      spec.Temperature,
			TopP:             spec.TopP,
			K:                spec.K,
			RetrievalMethod:  spec.RetrievalMethod,
			ProvideCitations: spec.ProvideCitations,
		})
		if err != nil {
			return fmt.Errorf("failed to update agent settings: %v", err)
		}
	}

	// 5. Attach guardrails
	for _, guardrailUUID := range spec.Guardrails {
		if err := client.AttachGradientAIGuardrail(agent.UUID, guardrailUUID); err != nil {
			return fmt.Errorf("failed to attach guardrail %s: %v", guardrailUUID, err)
		}
	}

	// 6. Add agent routes
	for _, childAgentUUID := range spec.AgentRoutes {
		if err := client.AddGradientAIAgentRoute(agent.UUID, childAgentUUID); err != nil {
			return fmt.Errorf("failed to add agent route %s: %v", childAgentUUID, err)
		}
	}

	// 7. Store metadata and start polling.
	// GradientAI agents auto-deploy on creation — there is no separate
	// deploy endpoint. We poll GET /v2/gen-ai/agents/{uuid} until the
	// deployment status becomes running.
	if err := ctx.Metadata.Set(CreateAgentExecutionMetadata{
		AgentUUID: agent.UUID,
		StartedAt: time.Now().UnixNano(),
	}); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, createAgentPollInterval)
}

func (c *CreateAgent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateAgent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateAgent) Actions() []core.Action {
	return []core.Action{
		{Name: "poll", UserAccessible: false},
	}
}

func (c *CreateAgent) HandleAction(ctx core.ActionContext) error {
	if ctx.Name == "poll" {
		return c.poll(ctx)
	}
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (c *CreateAgent) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata CreateAgentExecutionMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	if time.Since(time.Unix(0, metadata.StartedAt)) > createAgentTimeout {
		return fmt.Errorf("agent %s timed out waiting to deploy", metadata.AgentUUID)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	agent, err := client.GetGradientAIAgent(metadata.AgentUUID)
	if err != nil {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, createAgentPollInterval)
	}

	if agent.Deployment == nil {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, createAgentPollInterval)
	}

	status := strings.ToLower(agent.Deployment.Status)

	switch {
	case strings.Contains(status, "running") || strings.Contains(status, "active"):
		apiKey, err := client.CreateGradientAIAgentAPIKey(agent.UUID)
		if err != nil {
			return fmt.Errorf("failed to create agent API key: %v", err)
		}

		metadata.APIKey = apiKey.SecretKey
		if err := ctx.Metadata.Set(metadata); err != nil {
			return err
		}

		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			agentPayloadType,
			[]any{agent},
		)
	case strings.Contains(status, "error"):
		return fmt.Errorf("agent %s failed to deploy", metadata.AgentUUID)
	default:
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, createAgentPollInterval)
	}
}

func (c *CreateAgent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateAgent) Cleanup(ctx core.SetupContext) error {
	return nil
}
