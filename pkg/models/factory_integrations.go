package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListIntegrationNodes includes broken nodes so their connections can be repaired.
func (f *Factory) ListIntegrationNodes(tx *gorm.DB) ([]CanvasNode, error) {
	var nodes []CanvasNode
	err := withActiveCanvas(tx.Model(&CanvasNode{}), "workflow_nodes.workflow_id").
		Where("workflows.organization_id = ? AND workflows.factory_id = ?", f.OrganizationID, f.ID).
		Find(&nodes).Error
	return nodes, err
}

// RepositoryMonitoringWebhookIDs finds subscriptions that have no canvas node.
func (f *Factory) RepositoryMonitoringWebhookIDs(tx *gorm.DB) ([]uuid.UUID, error) {
	config := f.OnboardingConfigValue()
	integrationID, err := uuid.Parse(config.VCSIntegrationID)
	if err != nil || config.EffectiveVCSProvider() != ProviderGitHub {
		return nil, nil
	}
	repositories, err := f.WorkspaceRepositories(tx)
	if err != nil {
		return nil, err
	}
	for i := range repositories {
		repositories[i] = CanonicalContextRepo(repositories[i])
	}
	// Open pull requests keep monitoring their saved repository after a move.
	trackedRepositories := tx.Model(&FactoryPullRequest{}).Select("lower(trim(repository))").
		Where("organization_id = ? AND factory_id = ? AND provider = ?", f.OrganizationID, f.ID, ProviderGitHub).
		Where("state IN ?", []string{FactoryPullRequestStateOpen, FactoryPullRequestStateDraft})
	var ids []uuid.UUID
	err = tx.Model(&Webhook{}).
		Where("app_installation_id = ? AND configuration->>'factoryMergeability' = ?", integrationID, "true").
		Where("lower(trim(configuration->>'repository')) IN ? OR lower(trim(configuration->>'repository')) IN (?)", repositories, trackedRepositories).
		Pluck("id", &ids).Error
	return ids, err
}
