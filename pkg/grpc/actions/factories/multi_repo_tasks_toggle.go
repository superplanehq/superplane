package factories

import (
	"context"
	"slices"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SetOrganizationMultiRepoTasks turns multi-repo tasks on or off for the
// organization and republishes its Implement automations with the matching
// template in the same transaction. When the republish fails, the flag stays
// unchanged.
func SetOrganizationMultiRepoTasks(
	ctx context.Context,
	organizationID uuid.UUID,
	enabled bool,
	deps IntakeDependencies,
) (BuiltInAutomationReset, error) {
	if features.IsReleased(features.FeatureFactoryMultiRepoTasks) {
		return BuiltInAutomationReset{}, nil
	}

	var result BuiltInAutomationReset
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		organization, err := models.FindOrganizationByIDInTransaction(
			tx.Clauses(clause.Locking{Strength: "UPDATE"}),
			organizationID.String(),
		)
		if err != nil {
			return err
		}
		if slices.Contains(organization.EnabledExperimentalFeatures, features.FeatureFactoryMultiRepoTasks) == enabled {
			return nil
		}

		if enabled {
			err = models.EnableExperimentalFeatureInTransaction(tx, organizationID, features.FeatureFactoryMultiRepoTasks)
		} else {
			err = models.DisableExperimentalFeatureInTransaction(tx, organizationID, features.FeatureFactoryMultiRepoTasks)
		}
		if err != nil {
			return err
		}

		result, err = resetOrganizationBuiltInAutomationsInTransaction(
			ctx, tx, organizationID, lineImplementationTemplateID, hasMultiRepoTemplate, deps,
		)
		return err
	})
	if err != nil {
		return BuiltInAutomationReset{}, err
	}
	return result, nil
}

// hasMultiRepoTemplate skips Implement automations whose provider has one
// template only, so the flag change leaves them unchanged.
func hasMultiRepoTemplate(version *models.CanvasVersion) bool {
	template, ok := resolveFactoryTemplate(version.Nodes)
	return ok && template.multiRepoCanvasFile != ""
}
