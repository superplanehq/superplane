package models

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	FactoryOnboardingIssuesSourceVCS    = "vcs"
	FactoryOnboardingIssuesSourceLinear = "linear"
	FactoryOnboardingIssuesSourceJira   = "jira"
	FactoryOnboardingIssuesSourceSkip   = "skip"

	FactoryOnboardingAgentHarnessClaudeCode = "claude-code"
	FactoryOnboardingAgentHarnessCursor     = "cursor"
	FactoryOnboardingAgentHarnessCodex      = "codex"

	FactoryOnboardingAgentProviderAnthropic  = "anthropic"
	FactoryOnboardingAgentProviderOpenAI     = "openai"
	FactoryOnboardingAgentProviderOpenRouter = "openrouter"
)

var (
	ErrFactoryOnboardingInvalidIssuesSource      = errors.New("invalid issues source")
	ErrFactoryOnboardingInvalidAgentHarness      = errors.New("invalid agent harness")
	ErrFactoryOnboardingInvalidAgentProvider     = errors.New("invalid agent provider")
	ErrFactoryOnboardingInvalidIntegrationID     = errors.New("invalid integration id")
	ErrFactoryOnboardingInvalidAppID             = errors.New("invalid provisioned app id")
	ErrFactoryOnboardingInvalidLineID            = errors.New("invalid provisioned line id")
	ErrFactoryOnboardingInvalidRepository        = errors.New("repository must use the owner/name format")
	ErrFactoryOnboardingVCSIntegrationRequired   = errors.New("version control integration id is required")
	ErrFactoryOnboardingAgentIntegrationRequired = errors.New("agent integration id is required")
	ErrFactoryOnboardingHostedAgentUnavailable   = errors.New("hosted agent credentials are not available")
	ErrFactoryOnboardingAppRepositoryRequired    = errors.New("app repository is required")
	ErrFactoryOnboardingBacklogRepoRequired      = errors.New("backlog repository is required")
	ErrFactoryOnboardingIssuesSourceRequired     = errors.New("issues source is required")
	ErrFactoryOnboardingAgentHarnessRequired     = errors.New("agent harness is required")
	ErrFactoryOnboardingAppIDRequired            = errors.New("provisioned app id is required")
	ErrFactoryOnboardingLineIDRequired           = errors.New("provisioned line id is required")
)

var factoryOnboardingRepositoryPattern = regexp.MustCompile(`^[^/\s]+/[^/\s]+$`)

// FactoryOnboardingConfig stores durable wizard choices and provisioned
// resource IDs. Empty strings mean the field has not been saved yet.
type FactoryOnboardingConfig struct {
	VCSIntegrationID   string `json:"vcs_integration_id,omitempty"`
	AgentIntegrationID string `json:"agent_integration_id,omitempty"`
	AppRepository      string `json:"app_repository,omitempty"`
	BacklogRepository  string `json:"backlog_repository,omitempty"`
	DefaultBranch      string `json:"default_branch,omitempty"`
	IssuesSource       string `json:"issues_source,omitempty"`
	AgentHarness       string `json:"agent_harness,omitempty"`
	AgentProvider      string `json:"agent_provider,omitempty"`
	AgentModel         string `json:"agent_model,omitempty"`
	AgentPlanningModel string `json:"agent_planning_model,omitempty"`
	ProvisionedAppID   string `json:"provisioned_app_id,omitempty"`
	ProvisionedLineID  string `json:"provisioned_line_id,omitempty"`
}

// FactoryOnboardingPatch carries optional field updates for a partial merge.
// A nil pointer means "leave unchanged"; a non-nil pointer replaces the value
// (including clearing when the pointed string is empty, or when an enum is
// cleared to the empty string).
type FactoryOnboardingPatch struct {
	VCSIntegrationID   *string
	AgentIntegrationID *string
	AppRepository      *string
	BacklogRepository  *string
	DefaultBranch      *string
	IssuesSource       *string
	AgentHarness       *string
	AgentProvider      *string
	AgentModel         *string
	AgentPlanningModel *string
	ProvisionedAppID   *string
	ProvisionedLineID  *string
}

func ValidateFactoryOnboardingIssuesSource(source string) error {
	switch source {
	case "",
		FactoryOnboardingIssuesSourceVCS,
		FactoryOnboardingIssuesSourceLinear,
		FactoryOnboardingIssuesSourceJira,
		FactoryOnboardingIssuesSourceSkip:
		return nil
	default:
		return ErrFactoryOnboardingInvalidIssuesSource
	}
}

func ValidateFactoryOnboardingAgentHarness(harness string) error {
	switch harness {
	case "",
		FactoryOnboardingAgentHarnessClaudeCode,
		FactoryOnboardingAgentHarnessCursor,
		FactoryOnboardingAgentHarnessCodex:
		return nil
	default:
		return ErrFactoryOnboardingInvalidAgentHarness
	}
}

func ValidateFactoryOnboardingAgentProvider(provider string) error {
	switch provider {
	case "",
		FactoryOnboardingAgentProviderAnthropic,
		FactoryOnboardingAgentProviderOpenAI,
		FactoryOnboardingAgentProviderOpenRouter:
		return nil
	default:
		return ErrFactoryOnboardingInvalidAgentProvider
	}
}

func (f *Factory) IsOnboardingComplete() bool {
	return f.OnboardingCompletedAt != nil
}

func (f *Factory) OnboardingConfigValue() FactoryOnboardingConfig {
	return f.OnboardingConfig.Data()
}

func (f *Factory) OnboardingConfigAfter(patch FactoryOnboardingPatch) (FactoryOnboardingConfig, error) {
	return mergeFactoryOnboardingConfig(f.OnboardingConfigValue(), patch)
}

// UpdateOnboarding merges a partial patch into the stored config. It does not
// change completion status.
func (f *Factory) UpdateOnboarding(tx *gorm.DB, patch FactoryOnboardingPatch) error {
	merged, err := f.OnboardingConfigAfter(patch)
	if err != nil {
		return err
	}

	return f.persistOnboarding(tx, merged, f.OnboardingCompletedAt)
}

// CompleteOnboarding merges an optional patch, validates readiness, and sets
// onboarding_completed_at. When already complete, the existing timestamp is
// preserved (idempotent).
func (f *Factory) CompleteOnboarding(tx *gorm.DB, patch FactoryOnboardingPatch) error {
	merged, err := f.OnboardingConfigAfter(patch)
	if err != nil {
		return err
	}
	if err := validateFactoryOnboardingReady(merged); err != nil {
		return err
	}

	completedAt := f.OnboardingCompletedAt
	if completedAt == nil {
		now := time.Now()
		completedAt = &now
	}

	return f.persistOnboarding(tx, merged, completedAt)
}

func (f *Factory) persistOnboarding(tx *gorm.DB, config FactoryOnboardingConfig, completedAt *time.Time) error {
	now := time.Now()
	updates := map[string]any{
		"onboarding_config": datatypes.NewJSONType(config),
		"updated_at":        now,
	}
	if completedAt != nil {
		updates["onboarding_completed_at"] = *completedAt
	} else {
		updates["onboarding_completed_at"] = nil
	}

	err := tx.Model(f).
		Where("organization_id = ? AND id = ?", f.OrganizationID, f.ID).
		Updates(updates).
		Error
	if err != nil {
		return err
	}

	f.OnboardingConfig = datatypes.NewJSONType(config)
	f.OnboardingCompletedAt = completedAt
	f.UpdatedAt = now
	return nil
}

func mergeFactoryOnboardingConfig(current FactoryOnboardingConfig, patch FactoryOnboardingPatch) (FactoryOnboardingConfig, error) {
	next := current

	if patch.VCSIntegrationID != nil {
		value := strings.TrimSpace(*patch.VCSIntegrationID)
		if err := validateOptionalUUID(value, ErrFactoryOnboardingInvalidIntegrationID); err != nil {
			return FactoryOnboardingConfig{}, err
		}
		next.VCSIntegrationID = value
	}
	if patch.AgentIntegrationID != nil {
		value := strings.TrimSpace(*patch.AgentIntegrationID)
		if err := validateOptionalUUID(value, ErrFactoryOnboardingInvalidIntegrationID); err != nil {
			return FactoryOnboardingConfig{}, err
		}
		next.AgentIntegrationID = value
	}
	if patch.AppRepository != nil {
		value := strings.TrimSpace(*patch.AppRepository)
		if err := validateOptionalFactoryRepository(value); err != nil {
			return FactoryOnboardingConfig{}, err
		}
		next.AppRepository = value
	}
	if patch.BacklogRepository != nil {
		value := strings.TrimSpace(*patch.BacklogRepository)
		if err := validateOptionalFactoryRepository(value); err != nil {
			return FactoryOnboardingConfig{}, err
		}
		next.BacklogRepository = value
	}
	if patch.DefaultBranch != nil {
		next.DefaultBranch = strings.TrimSpace(*patch.DefaultBranch)
	}
	if patch.IssuesSource != nil {
		value := strings.TrimSpace(*patch.IssuesSource)
		if err := ValidateFactoryOnboardingIssuesSource(value); err != nil {
			return FactoryOnboardingConfig{}, err
		}
		next.IssuesSource = value
	}
	if patch.AgentHarness != nil {
		value := strings.TrimSpace(*patch.AgentHarness)
		if err := ValidateFactoryOnboardingAgentHarness(value); err != nil {
			return FactoryOnboardingConfig{}, err
		}
		next.AgentHarness = value
	}
	if patch.AgentProvider != nil {
		value := strings.TrimSpace(*patch.AgentProvider)
		if err := ValidateFactoryOnboardingAgentProvider(value); err != nil {
			return FactoryOnboardingConfig{}, err
		}
		next.AgentProvider = value
	}
	if patch.AgentModel != nil {
		next.AgentModel = strings.TrimSpace(*patch.AgentModel)
	}
	if patch.AgentPlanningModel != nil {
		next.AgentPlanningModel = strings.TrimSpace(*patch.AgentPlanningModel)
	}
	if patch.ProvisionedAppID != nil {
		value := strings.TrimSpace(*patch.ProvisionedAppID)
		if err := validateOptionalUUID(value, ErrFactoryOnboardingInvalidAppID); err != nil {
			return FactoryOnboardingConfig{}, err
		}
		next.ProvisionedAppID = value
	}
	if patch.ProvisionedLineID != nil {
		value := strings.TrimSpace(*patch.ProvisionedLineID)
		if err := validateOptionalUUID(value, ErrFactoryOnboardingInvalidLineID); err != nil {
			return FactoryOnboardingConfig{}, err
		}
		next.ProvisionedLineID = value
	}

	return next, nil
}

func validateFactoryOnboardingReady(config FactoryOnboardingConfig) error {
	if strings.TrimSpace(config.AppRepository) == "" {
		return ErrFactoryOnboardingAppRepositoryRequired
	}
	if strings.TrimSpace(config.BacklogRepository) == "" {
		return ErrFactoryOnboardingBacklogRepoRequired
	}
	if strings.TrimSpace(config.VCSIntegrationID) == "" {
		return ErrFactoryOnboardingVCSIntegrationRequired
	}
	if config.IssuesSource == "" {
		return ErrFactoryOnboardingIssuesSourceRequired
	}
	if err := ValidateFactoryOnboardingIssuesSource(config.IssuesSource); err != nil {
		return err
	}
	if strings.TrimSpace(config.AgentHarness) == "" {
		return ErrFactoryOnboardingAgentHarnessRequired
	}
	if err := ValidateFactoryOnboardingAgentHarness(config.AgentHarness); err != nil {
		return err
	}
	if strings.TrimSpace(config.ProvisionedAppID) == "" {
		return ErrFactoryOnboardingAppIDRequired
	}
	if err := validateOptionalUUID(config.ProvisionedAppID, ErrFactoryOnboardingInvalidAppID); err != nil {
		return err
	}
	if strings.TrimSpace(config.ProvisionedLineID) == "" {
		return ErrFactoryOnboardingLineIDRequired
	}
	return validateOptionalUUID(config.ProvisionedLineID, ErrFactoryOnboardingInvalidLineID)
}

func validateOptionalFactoryRepository(repository string) error {
	if repository == "" {
		return nil
	}
	if !factoryOnboardingRepositoryPattern.MatchString(repository) {
		return ErrFactoryOnboardingInvalidRepository
	}
	return nil
}

func validateOptionalUUID(value string, invalid error) error {
	if value == "" {
		return nil
	}
	if _, err := uuid.Parse(value); err != nil {
		return invalid
	}
	return nil
}
