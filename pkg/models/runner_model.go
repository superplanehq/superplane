package models

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RunnerInvokesModel excludes command-only nodes from model policy checks.
// Missing steps retain the legacy prompt configuration behavior.
func RunnerInvokesModel(configuration map[string]any) bool {
	raw, exists := configuration["steps"]
	if !exists {
		return true
	}
	switch steps := raw.(type) {
	case []any:
		for _, value := range steps {
			step, ok := value.(map[string]any)
			if !ok || step["type"] != "bash" {
				return true
			}
		}
	case []map[string]any:
		for _, step := range steps {
			if step["type"] != "bash" {
				return true
			}
		}
	default:
		return true
	}
	return false
}

// RunnerAcceptsModel checks execution policy for the executing funding source.
// Shared overrides must first match the runner provider with RunnerMatchesModel.
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
	if funding == UsageFundingSourceBYOK {
		enabled, err := OrganizationBYOKAccess(tx, orgID, provider)
		if err != nil || !enabled {
			return false, err
		}
		policy, err := runnerWorkspacePolicy(tx, factoryID, provider)
		if err != nil {
			return false, err
		}
		return policy.InheritsParent() || slices.Contains(policy.AllowedModels, strings.TrimSpace(model)), nil
	}
	return ModelIsSelectable(tx, orgID, factoryID, provider, funding, model)
}

// ResolveBYOKRunnerModel applies explicit workspace execution policy. Provider
// discovery is optional; inherited policy preserves the CLI's existing default.
func ResolveBYOKRunnerModel(tx *gorm.DB, orgID uuid.UUID, factoryID *uuid.UUID, component string, configuration map[string]any) (string, error) {
	model, _ := configuration["model"].(string)
	model = strings.TrimSpace(model)
	if !RunnerInvokesModel(configuration) {
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
	if factoryID != nil && *factoryID != uuid.Nil {
		if err := RequireOrganizationBYOKAccess(tx, orgID, provider); err != nil {
			if errors.Is(err, ErrBYOKAccessDenied) {
				return "", fmt.Errorf("%w: %w", ErrSelectableLLMModelNotAllowed, err)
			}
			return "", err
		}
	}
	policy, err := runnerWorkspacePolicy(tx, factoryID, provider)
	if err != nil {
		return "", err
	}
	if policy.InheritsParent() {
		return model, nil
	}
	permitted := CompactModelIDs(policy.AllowedModels)
	if len(permitted) == 0 {
		return "", fmt.Errorf("No models are enabled for this provider. Update Workspace Models: %w", ErrSelectableLLMModelNotAllowed)
	}
	if model != "" {
		model = ConcreteClaudeModelID(model, permitted)
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
		return "", ErrSelectableLLMModelNotAllowed
	}
	slices.Sort(concrete)
	return concrete[0], nil
}

func runnerWorkspacePolicy(tx *gorm.DB, factoryID *uuid.UUID, provider string) (*FactoryLLMModelAllowlist, error) {
	if factoryID == nil || *factoryID == uuid.Nil {
		return nil, nil
	}
	return FindFactoryLLMModelAllowlist(tx, *factoryID, provider, UsageFundingSourceBYOK)
}

// RunnerMatchesModel routes shared overrides independently of workspace policy.
// Unknown native IDs need discovery or a configured match; catalog absence never
// prevents execution of the runner's existing configuration.
func RunnerMatchesModel(tx *gorm.DB, orgID uuid.UUID, component string, configuration map[string]any, model string) (bool, error) {
	model = strings.TrimSpace(model)
	if component == SuperPlaneRunnerComponent {
		_, err := ParseHostedLLMModelKey(model)
		return err == nil, nil
	}
	provider, ok := runnerLLMProvider(component, configuration)
	if !ok {
		return false, nil
	}
	credentials, _ := configuration["credentials"].(map[string]any)
	if credentials["source"] == "subscription" {
		return component == "runnerCodex" && !strings.Contains(model, "/") && !strings.Contains(model, "::") && !strings.HasPrefix(model, "claude-") && !IsClaudeFamilyAlias(model), nil
	}
	if strings.Contains(model, "::") {
		return false, nil
	}
	stored, _ := configuration["model"].(string)
	if model == strings.TrimSpace(stored) {
		return true, nil
	}
	switch provider {
	case UsageProviderAnthropic:
		if strings.HasPrefix(model, "claude-") || IsClaudeFamilyAlias(model) && !strings.Contains(model, "/") {
			return true, nil
		}
	case UsageProviderOpenAI:
		if strings.HasPrefix(model, "gpt-") || strings.HasPrefix(model, "codex-") || len(model) > 1 && model[0] == 'o' && model[1] >= '0' && model[1] <= '9' {
			return true, nil
		}
	case UsageProviderOpenRouter:
		if strings.Contains(model, "/") {
			return true, nil
		}
	}
	ids, err := parentSelectableLLMModels(tx, orgID, provider, llmRunnerFundingSource(configuration))
	return slices.Contains(ids, model), err
}

func llmRunnerFundingSource(configuration map[string]any) string {
	credentials, _ := configuration["credentials"].(map[string]any)
	if credentials["source"] == "integration" || credentials["source"] == "secret" {
		return UsageFundingSourceBYOK
	}
	return UsageFundingSourceHosted
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
	if !RunnerInvokesModel(configuration) {
		return nil
	}
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
		provider, ok := runnerLLMProvider(component, configuration)
		if !ok {
			return nil
		}
		if canvas.FactoryID != nil {
			if err := RequireOrganizationBYOKAccess(tx, canvas.OrganizationID, provider); err != nil {
				if errors.Is(err, ErrBYOKAccessDenied) {
					return fmt.Errorf("%w: %w", ErrSelectableLLMModelNotAllowed, err)
				}
				return err
			}
		}
		policy, err := runnerWorkspacePolicy(tx, canvas.FactoryID, provider)
		if err != nil {
			return err
		}
		if policy.InheritsParent() {
			return nil
		}
		if !slices.Contains(policy.AllowedModels, strings.TrimSpace(model)) {
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
