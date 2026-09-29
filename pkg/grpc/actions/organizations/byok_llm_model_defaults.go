package organizations

import (
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// enableAllBYOKModelsByDefault saves every candidate as selected when the
// organization has not saved a model list for the provider yet. A saved
// list, including an empty one, is the organization's choice and stays.
func enableAllBYOKModelsByDefault(
	tx *gorm.DB,
	orgID uuid.UUID,
	provider string,
	candidates []*pb.HostedLLMModel,
) error {
	if len(candidates) == 0 {
		return nil
	}

	ids := make(datatypes.JSONSlice[string], 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.GetId())
	}
	return models.CreateOrganizationBYOKModelAllowlistIfAbsent(tx, orgID, provider, ids)
}

// enableAllConnectedBYOKModelsByDefault applies the default for every
// connected provider without a saved list. Provider failures only skip the
// default, so the caller can still list the models that are known.
func enableAllConnectedBYOKModelsByDefault(tx *gorm.DB, reg *registry.Registry, orgID uuid.UUID) {
	for _, provider := range models.KnownHostedLLMProviders() {
		logger := log.WithFields(log.Fields{"organization_id": orgID.String(), "provider": provider})
		exists, err := models.OrganizationBYOKModelAllowlistExists(tx, orgID, provider)
		if err != nil {
			logger.WithError(err).Warn("check saved byok models")
			continue
		}
		if exists {
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

		candidates, err := listBYOKCandidateModels(tx, reg, integration)
		if err != nil {
			logger.WithError(err).Warn("list byok candidate models")
			continue
		}
		if err := enableAllBYOKModelsByDefault(tx, orgID, provider, candidates); err != nil {
			logger.WithError(err).Warn("enable byok models by default")
		}
	}
}
