package runner

import (
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
)

// ModelOrFactoryDefault keeps a step model. An empty step model uses the
// workspace default when that id matches this runner.
func ModelOrFactoryDefault(ctx core.ExecutionContext, component, model string) (string, error) {
	if strings.TrimSpace(model) != "" {
		return model, nil
	}
	return FactoryDefaultModelID(ctx, component, configurationMap(ctx.Configuration))
}

func configurationMap(value any) map[string]any {
	configuration, _ := value.(map[string]any)
	return configuration
}

func FactoryDefaultModelID(ctx core.ExecutionContext, component string, configuration map[string]any) (string, error) {
	if ctx.Factory == nil {
		return "", nil
	}
	saved, err := ctx.Factory.DefaultAgentModel()
	if err != nil || !saved.IsSet() || strings.TrimSpace(saved.Provider) != "" {
		return "", err
	}
	if ctx.HostedLLM == nil {
		return "", nil
	}
	provider := byokProviderForRunner(component, configuration)
	if provider == "" {
		return "", nil
	}
	if err := ctx.HostedLLM.AssertModelSelectable(provider, models.UsageFundingSourceBYOK, saved.Model); err != nil {
		return "", nil
	}
	return strings.TrimSpace(saved.Model), nil
}

// SelectableHostedFactoryDefault returns a hosted workspace default when the
// workspace allows that model. An empty result means the caller keeps the
// installation fallback.
func SelectableHostedFactoryDefault(ctx core.ExecutionContext) (core.DefaultHostedLLMModel, bool, error) {
	if ctx.Factory == nil || ctx.HostedLLM == nil {
		return core.DefaultHostedLLMModel{}, false, nil
	}
	saved, err := ctx.Factory.DefaultAgentModel()
	if err != nil || !saved.IsSet() || strings.TrimSpace(saved.Provider) == "" {
		return core.DefaultHostedLLMModel{}, false, err
	}
	provider := strings.TrimSpace(saved.Provider)
	model := strings.TrimSpace(saved.Model)
	if err := ctx.HostedLLM.AssertModelSelectable(provider, models.UsageFundingSourceHosted, model); err != nil {
		return core.DefaultHostedLLMModel{}, false, nil
	}
	return core.DefaultHostedLLMModel{Provider: provider, Model: model}, true, nil
}

func byokProviderForRunner(component string, configuration map[string]any) string {
	switch component {
	case "runnerClaudeCode":
		return models.UsageProviderAnthropic
	case "runnerCodex":
		return models.UsageProviderOpenAI
	case "runnerOpenRouter":
		if configuration["llmProvider"] == models.UsageProviderCustom {
			return models.UsageProviderCustom
		}
		return models.UsageProviderOpenRouter
	default:
		return ""
	}
}
