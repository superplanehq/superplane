package support

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// SaveProviderModelCatalog seeds the same integration-scoped discovery data as
// a successful provider catalog request. It does not configure model policy.
func SaveProviderModelCatalog(tx *gorm.DB, orgID uuid.UUID, provider string, ids []string) (*models.IntegrationModelCatalog, error) {
	integration, err := models.FindReadyBYOKIntegration(tx, orgID, provider)
	if err != nil {
		return nil, err
	}
	if integration == nil {
		app, err := models.BYOKIntegrationAppName(provider)
		if err != nil {
			return nil, err
		}
		integration = models.NewIntegration(uuid.New(), orgID, app, RandomName(app), map[string]any{})
		integration.State = models.IntegrationStateReady
		if err := tx.Create(integration).Error; err != nil {
			return nil, err
		}
	}
	catalog := make([]models.SelectableLLMNamedID, 0, len(ids))
	for _, id := range ids {
		catalog = append(catalog, models.SelectableLLMNamedID{ID: id, Name: id})
	}
	if err := models.StoreIntegrationModelCatalog(tx, integration.ID, catalog); err != nil {
		return nil, err
	}
	return models.FindIntegrationModelCatalog(tx, integration.ID)
}
