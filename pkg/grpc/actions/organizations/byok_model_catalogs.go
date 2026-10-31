package organizations

import (
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

// refreshMissingBYOKModelCatalogs warms unknown catalogs only. Provider catalog
// endpoints refresh existing snapshots without making every picker wait on HTTP.
func refreshMissingBYOKModelCatalogs(tx *gorm.DB, reg *registry.Registry, orgID uuid.UUID) {
	for _, provider := range append(models.KnownHostedLLMProviders(), models.UsageProviderCustom) {
		logger := log.WithFields(log.Fields{"organization_id": orgID.String(), "provider": provider})

		allowed, err := models.OrganizationBYOKAccess(tx, orgID, provider)
		if err != nil {
			logger.WithError(err).Warn("check provider catalog access")
			continue
		}
		if !allowed {
			continue
		}

		integration, err := models.FindReadyBYOKIntegration(tx, orgID, provider)
		if err != nil {
			logger.WithError(err).Warn("find byok integration")
			continue
		}
		if integration == nil {
			continue
		}

		catalog, err := models.FindIntegrationModelCatalog(tx, integration.ID)
		if err != nil {
			logger.WithError(err).Warn("load provider model catalog")
			continue
		}
		if catalog != nil {
			continue
		}

		if _, err := refreshBYOKModelCatalog(tx, reg, integration); err != nil {
			logger.WithError(err).Warn("refresh provider model catalog")
		}
	}
}
