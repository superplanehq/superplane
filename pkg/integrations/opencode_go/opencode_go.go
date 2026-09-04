package opencodego

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("opencodego", &OpenCodeGo{})
}

const ResourceTypeModel = "model"

// The models endpoint does not identify the protocol for each model. Keep the
// routing tables here so one action can select the correct endpoint.
var supportedChatModels = map[string]bool{
	"glm-5.3":                      true,
	"glm-5.2":                      true,
	"glm-5.1":                      true,
	"kimi-k3":                      true,
	"kimi-k2.7-code":               true,
	"kimi-k2.6":                    true,
	"deepseek-v4-pro":              true,
	"deepseek-v4-flash":            true,
	"deepseek-v4-flash-vision-exp": true,
	"mimo-v2.5":                    true,
	"mimo-v2.5-pro":                true,
	"hy3":                          true,
	"ox-alpha-free":                true,
}

var supportedMessagesModels = map[string]bool{
	"minimax-m3":   true,
	"minimax-m2.7": true,
	"minimax-m2.5": true,
	"qwen3.8-max":  true,
	"qwen3.7-max":  true,
	"qwen3.7-plus": true,
	"qwen3.6-plus": true,
}

var supportedResponsesModels = map[string]bool{
	"grok-4.5":                   true,
	"gpt-5.6-luna":               true,
	"muse-spark-1.2-contributor": true,
}

type OpenCodeGo struct{}

type Configuration struct {
	APIKey string `json:"apiKey"`
}

func (o *OpenCodeGo) Name() string {
	return "opencodego"
}

func (o *OpenCodeGo) Label() string {
	return "OpenCode Go"
}

func (o *OpenCodeGo) Icon() string {
	return "opencodego"
}

func (o *OpenCodeGo) Description() string {
	return "Run prompts on curated open coding models through OpenCode Go"
}

func (o *OpenCodeGo) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "API key from the OpenCode Zen console (opencode.ai/auth)",
		},
	}
}

func (o *OpenCodeGo) Actions() []core.Action {
	return []core.Action{
		&ChatCompletion{},
		&GetUsage{},
	}
}

func (o *OpenCodeGo) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (o *OpenCodeGo) Instructions() string {
	return `## OpenCode Go API Key

Subscribe to OpenCode Go at [opencode.ai/docs/go](https://opencode.ai/docs/go).

Sign in at [opencode.ai/auth](https://opencode.ai/auth) and copy your API key. Paste it below.

Usage limits apply: $12 per 5 hours, $30 per week, and $60 per month.

## Supported models

The Chat Completion component selects the correct OpenCode Go endpoint for the model. Chat Completions supports GLM, Kimi, DeepSeek, MiMo, Hy3, and Ox Alpha Free. The Messages API supports MiniMax and Qwen. The Responses API supports Grok 4.5, GPT 5.6 Luna, and Muse Spark 1.2 Contributor.

## Model opt-ins

Some models need an opt-in in your OpenCode Go workspace settings at [opencode.ai/auth](https://opencode.ai/auth):

- DeepSeek V4 Pro and DeepSeek V4 Flash: enable the China region.
- Muse Spark 1.2 Contributor: accept training data use. Availability is limited to permitted regions.

If a run fails with a region or data policy error, check these settings first.`
}

func (o *OpenCodeGo) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (o *OpenCodeGo) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("apiKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.Verify(); err != nil {
		return err
	}

	ctx.Integration.Ready()
	return nil
}

func (o *OpenCodeGo) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (o *OpenCodeGo) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != ResourceTypeModel {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	models, err := client.ListModels()
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, 0, len(models))
	for _, model := range models {
		if model.ID == "" || (!supportedChatModels[model.ID] && !supportedMessagesModels[model.ID] && !supportedResponsesModels[model.ID]) {
			continue
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: model.ID,
			ID:   model.ID,
		})
	}

	return resources, nil
}

func (o *OpenCodeGo) Hooks() []core.Hook {
	return []core.Hook{}
}

func (o *OpenCodeGo) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
