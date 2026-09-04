package models

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const SuperPlaneRunnerComponent = "runnerSuperPlane"

const (
	SuperPlaneRunnerNoCreditMessage        = "This organization has no hosted credit."
	SuperPlaneRunnerNoFactoryBudgetMessage = "This workspace has no remaining hosted credit."
	SuperPlaneRunnerNoModelMessage         = "The instance has no SuperPlane agent model."
	SuperPlaneRunnerModelNotAllowedMessage = "This workspace does not allow the SuperPlane agent model."
)

var hostedProviderRunnerComponents = []string{
	"runnerClaudeCode",
	"runnerCodex",
	"runnerOpenRouter",
}

var (
	ErrSuperPlaneRunnerNoCredit         = errors.New("this organization has no hosted credit")
	ErrSuperPlaneRunnerNoFactoryBudget  = errors.New("this workspace has no remaining hosted credit")
	ErrSuperPlaneRunnerNoModel          = errors.New("the instance has no SuperPlane agent model")
	ErrSuperPlaneRunnerModelNotAllowed  = errors.New("this workspace does not allow the SuperPlane agent model")
	ErrDefaultHostedModelIncomplete     = errors.New("SuperPlane agent model requires a provider and a model")
	ErrDefaultHostedModelNotOnAllowlist = errors.New("SuperPlane agent model is not on a hosted allowlist")
	ErrDefaultHostedModelMustBeReplaced = errors.New("Pick a SuperPlane agent model before you remove the current default from the allowlist")
)

// DefaultHostedLLMModel is the instance model the SuperPlane agent runs.
type DefaultHostedLLMModel struct {
	Provider string
	Model    string
}

func (d DefaultHostedLLMModel) IsSet() bool {
	return strings.TrimSpace(d.Provider) != "" && strings.TrimSpace(d.Model) != ""
}

// RewriteHostedProviderRunnerToSuperPlane converts a leftover hosted
// Claude, Codex, or OpenRouter node into runnerSuperPlane. Integration and
// secret nodes stay unchanged.
func RewriteHostedProviderRunnerToSuperPlane(node *Node) bool {
	if node == nil {
		return false
	}
	if !isHostedProviderRunnerComponent(node.ComponentName()) {
		return false
	}
	if !nodeHasHostedCredentials(node.Configuration) {
		return false
	}

	if node.Ref.Component == nil {
		node.Ref.Component = &ComponentRef{}
	}
	node.Ref.Component.Name = SuperPlaneRunnerComponent
	if node.Configuration != nil {
		delete(node.Configuration, "credentials")
		delete(node.Configuration, "model")
		delete(node.Configuration, "maxTurns")
	}
	return true
}

func RewriteHostedProviderRunnerNodes(nodes []Node) {
	for i := range nodes {
		RewriteHostedProviderRunnerToSuperPlane(&nodes[i])
	}
}

func NormalizeDefaultHostedLLMModel(provider, model string) (DefaultHostedLLMModel, error) {
	normalizedProvider := strings.ToLower(strings.TrimSpace(provider))
	normalizedModel := strings.TrimSpace(model)
	if normalizedProvider == "" && normalizedModel == "" {
		return DefaultHostedLLMModel{}, nil
	}
	if normalizedProvider == "" || normalizedModel == "" {
		return DefaultHostedLLMModel{}, ErrDefaultHostedModelIncomplete
	}
	known, err := NormalizeHostedLLMProvider(normalizedProvider)
	if err != nil {
		return DefaultHostedLLMModel{}, err
	}
	return DefaultHostedLLMModel{Provider: known, Model: normalizedModel}, nil
}

func InstallationDefaultHostedLLMModel(settings *InstallationLLMSettings) DefaultHostedLLMModel {
	if settings == nil {
		return DefaultHostedLLMModel{}
	}
	return DefaultHostedLLMModel{
		Provider: strings.TrimSpace(stringValue(settings.DefaultHostedProvider)),
		Model:    strings.TrimSpace(stringValue(settings.DefaultHostedModel)),
	}
}

func GetInstallationDefaultHostedLLMModel(tx *gorm.DB) (DefaultHostedLLMModel, error) {
	settings, err := GetInstallationLLMSettings(tx)
	if err != nil {
		return DefaultHostedLLMModel{}, err
	}
	return InstallationDefaultHostedLLMModel(settings), nil
}

func AssertDefaultHostedLLMModelAllowed(tx *gorm.DB, defaultModel DefaultHostedLLMModel) error {
	if !defaultModel.IsSet() {
		return nil
	}
	row, err := FindHostedLLMProvider(tx, defaultModel.Provider)
	if err != nil {
		if errors.Is(err, ErrHostedLLMProviderNotFound) {
			return ErrDefaultHostedModelNotOnAllowlist
		}
		return err
	}
	if !row.OffersHostedModels() || !row.AllowsModel(defaultModel.Model) {
		return ErrDefaultHostedModelNotOnAllowlist
	}
	return nil
}

// SyncDefaultHostedLLMModelAfterProviderChange clears an invalid default when
// no hosted models remain. It rejects the change when other models still exist.
func SyncDefaultHostedLLMModelAfterProviderChange(tx *gorm.DB) error {
	defaultModel, err := GetInstallationDefaultHostedLLMModel(tx)
	if err != nil {
		return err
	}
	if !defaultModel.IsSet() {
		return nil
	}
	if err := AssertDefaultHostedLLMModelAllowed(tx, defaultModel); err == nil {
		return nil
	}

	offered, err := HasOfferedHostedLLMProvider(tx)
	if err != nil {
		return err
	}
	if offered {
		return ErrDefaultHostedModelMustBeReplaced
	}
	return clearInstallationDefaultHostedLLMModel(tx)
}

func SuperPlaneRunnerReadinessError(tx *gorm.DB, orgID uuid.UUID, factoryID *uuid.UUID) error {
	if orgID == uuid.Nil {
		return fmt.Errorf("organization is required for hosted LLM credit")
	}

	if err := AssertHostedRunAllowed(tx, orgID, factoryID); err != nil {
		if errors.Is(err, ErrHostedCreditEmpty) {
			return ErrSuperPlaneRunnerNoCredit
		}
		if errors.Is(err, ErrFactoryHostedBudgetEmpty) {
			return ErrSuperPlaneRunnerNoFactoryBudget
		}
		return err
	}

	defaultModel, err := GetInstallationDefaultHostedLLMModel(tx)
	if err != nil {
		return err
	}
	if !defaultModel.IsSet() {
		return ErrSuperPlaneRunnerNoModel
	}
	if err := AssertDefaultHostedLLMModelAllowed(tx, defaultModel); err != nil {
		return ErrSuperPlaneRunnerNoModel
	}

	allowed, err := ModelIsSelectable(tx, orgID, factoryID, defaultModel.Provider, UsageFundingSourceHosted, defaultModel.Model)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrSuperPlaneRunnerModelNotAllowed
	}
	return nil
}

func AnnotateSuperPlaneRunnerNodes(tx *gorm.DB, orgID uuid.UUID, factoryID *uuid.UUID, nodes []Node) error {
	var readiness error
	checked := false
	for i := range nodes {
		RewriteHostedProviderRunnerToSuperPlane(&nodes[i])
		if nodes[i].ComponentName() != SuperPlaneRunnerComponent {
			continue
		}
		if !checked {
			readiness = SuperPlaneRunnerReadinessError(tx, orgID, factoryID)
			checked = true
		}
		if readiness != nil {
			message := SuperPlaneRunnerReadinessMessage(readiness)
			nodes[i].ErrorMessage = &message
			continue
		}
		if nodes[i].ErrorMessage != nil && isSuperPlaneReadinessMessage(*nodes[i].ErrorMessage) {
			nodes[i].ErrorMessage = nil
		}
	}
	return nil
}

func SuperPlaneRunnerReadinessMessage(err error) string {
	switch {
	case errors.Is(err, ErrSuperPlaneRunnerNoCredit):
		return SuperPlaneRunnerNoCreditMessage
	case errors.Is(err, ErrSuperPlaneRunnerNoFactoryBudget), errors.Is(err, ErrFactoryHostedBudgetEmpty):
		return SuperPlaneRunnerNoFactoryBudgetMessage
	case errors.Is(err, ErrSuperPlaneRunnerNoModel):
		return SuperPlaneRunnerNoModelMessage
	case errors.Is(err, ErrSuperPlaneRunnerModelNotAllowed):
		return SuperPlaneRunnerModelNotAllowedMessage
	default:
		return err.Error()
	}
}

func isSuperPlaneReadinessMessage(message string) bool {
	switch strings.TrimSpace(message) {
	case SuperPlaneRunnerNoCreditMessage,
		SuperPlaneRunnerNoFactoryBudgetMessage,
		SuperPlaneRunnerNoModelMessage,
		SuperPlaneRunnerModelNotAllowedMessage:
		return true
	default:
		return false
	}
}

func isHostedProviderRunnerComponent(name string) bool {
	for _, component := range hostedProviderRunnerComponents {
		if name == component {
			return true
		}
	}
	return false
}

func nodeHasHostedCredentials(configuration map[string]any) bool {
	credentials, ok := configuration["credentials"].(map[string]any)
	if !ok {
		return false
	}
	source, _ := credentials["source"].(string)
	return strings.TrimSpace(source) == "hosted"
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func stringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func clearInstallationDefaultHostedLLMModel(tx *gorm.DB) error {
	return tx.Model(&InstallationLLMSettings{}).
		Where("id = ?", installationLLMSettingsID).
		Updates(map[string]any{
			"default_hosted_provider": nil,
			"default_hosted_model":    nil,
		}).Error
}
