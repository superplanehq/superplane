package models

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RunnerAcceptsModel checks the provider and funding source of the executing node.
func RunnerAcceptsModel(tx *gorm.DB, orgID uuid.UUID, factoryID *uuid.UUID, component string, configuration map[string]any, model string, ownerIDs ...uuid.UUID) (bool, error) {
	if component == SuperPlaneRunnerComponent {
		parsed, err := ParseHostedLLMModelKey(model)
		if err != nil {
			provider, _ := configuration["hostedProvider"].(string)
			parsed, err = NormalizeDefaultHostedLLMModel(provider, model)
		}
		if err != nil || !parsed.IsSet() {
			return false, nil
		}
		return ModelIsSelectable(tx, orgID, factoryID, parsed.Provider, UsageFundingSourceHosted, parsed.Model)
	}
	provider, ok := runnerLLMProvider(component, configuration)
	if !ok {
		return false, nil
	}
	credentials, _ := configuration["credentials"].(map[string]any)
	source, _ := credentials["source"].(string)
	if strings.TrimSpace(source) == "subscription" {
		if component != "runnerCodex" || factoryID == nil || *factoryID == uuid.Nil {
			return false, nil
		}
		if len(ownerIDs) != 1 {
			return false, nil
		}
		models, err := FactoryCodexSubscriptionModels(tx, orgID, *factoryID, ownerIDs[0])
		return slices.Contains(models, strings.TrimSpace(model)), err
	}
	funding := UsageFundingSourceHosted
	switch strings.TrimSpace(source) {
	case "secret", "integration":
		funding = UsageFundingSourceBYOK
	}
	return ModelIsSelectable(tx, orgID, factoryID, provider, funding, model)
}

// ResolveBYOKRunnerModel validates the final model after overrides and alias
// resolution. Blank models use the first permitted concrete model.
// Workspace model policy applies only to workspace automations. Standalone
// automations keep their configured model, as they did before workspace
// model policy existed.
func ResolveBYOKRunnerModel(tx *gorm.DB, orgID uuid.UUID, factoryID *uuid.UUID, component string, configuration map[string]any) (string, error) {
	model, _ := configuration["model"].(string)
	model = strings.TrimSpace(model)
	if factoryID == nil || *factoryID == uuid.Nil {
		return model, nil
	}
	provider, ok := runnerLLMProvider(component, configuration)
	if !ok {
		return model, nil
	}
	credentials, _ := configuration["credentials"].(map[string]any)
	if credentials["source"] != "integration" && credentials["source"] != "secret" {
		return model, nil
	}
	permitted, err := ResolveSelectableLLMModels(tx, orgID, factoryID, provider, UsageFundingSourceBYOK)
	if err != nil {
		return "", err
	}
	if model != "" {
		if !slices.Contains(permitted, model) {
			return "", ErrSelectableLLMModelNotAllowed
		}
		return model, nil
	}
	concrete := make([]string, 0, len(permitted))
	for _, candidate := range permitted {
		resolved := ConcreteClaudeModelID(candidate, permitted)
		if slices.Contains(permitted, resolved) {
			concrete = append(concrete, resolved)
		}
	}
	if len(concrete) == 0 {
		return "", fmt.Errorf("No models are enabled for this provider. Update Models in workspace settings: %w", ErrSelectableLLMModelNotAllowed)
	}
	slices.Sort(concrete)
	return concrete[0], nil
}

// RequireModelPolicyAdmission checks queued payloads without changing their
// snapshots. Running tasks do not pass through runner reservation again.
func (t *RunnerTask) RequireModelPolicyAdmission(tx *gorm.DB) error {
	execution, err := t.FindExecution(tx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	configuration := execution.Configuration.Data()
	credentials, _ := configuration["credentials"].(map[string]any)
	if credentials["source"] == "subscription" {
		return nil // Subscription admission also validates the owner and lease.
	}
	canvas, err := FindCanvasWithoutOrgScopeInTransaction(tx, execution.WorkflowID)
	if err != nil {
		return err
	}
	component := execution.FrozenComponentName()
	if component == "" {
		run, err := FindCanvasRunInTransaction(tx, execution.WorkflowID, execution.RunID)
		if err != nil {
			return err
		}
		version, err := FindCanvasVersionInTransaction(tx, execution.WorkflowID, run.VersionID)
		if err != nil {
			return err
		}
		for _, node := range version.Nodes {
			if node.ID == execution.NodeID {
				component = node.ComponentName()
				break
			}
		}
	}
	model, _ := configuration["model"].(string)
	if credentials["source"] == "integration" || credentials["source"] == "secret" {
		resolved, err := ResolveBYOKRunnerModel(tx, canvas.OrganizationID, canvas.FactoryID, component, configuration)
		if err != nil {
			return err
		}
		if strings.TrimSpace(model) != resolved {
			return ErrSelectableLLMModelNotAllowed
		}
		return nil
	}
	_, providerRunner := runnerLLMProvider(component, configuration)
	if component != SuperPlaneRunnerComponent && !providerRunner {
		return nil
	}
	allowed, err := RunnerAcceptsModel(tx, canvas.OrganizationID, canvas.FactoryID, component, configuration, model)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrSelectableLLMModelNotAllowed
	}
	return nil
}

func runnerLLMProvider(component string, configuration map[string]any) (string, bool) {
	switch component {
	case "runnerClaudeCode":
		return UsageProviderAnthropic, true
	case "runnerCodex":
		return UsageProviderOpenAI, true
	case "runnerOpenRouter":
		if configuration["llmProvider"] == UsageProviderCustom {
			return UsageProviderCustom, true
		}
		return UsageProviderOpenRouter, true
	default:
		return "", false
	}
}
