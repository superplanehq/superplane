package factories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
)

// grantIntakeCatalogAccess enables the intake feature flags, so tests about
// intake behavior do not depend on which flags the organization already has.
func grantIntakeCatalogAccess(t *testing.T, organizationID uuid.UUID) {
	t.Helper()

	db := database.DB(t.Context())
	for _, featureID := range []string{
		features.FeatureFactoryJiraIntake,
		features.FeatureFactoryProductiveIntake,
		features.FeatureFactoryDatadogIntake,
		features.FeatureFactoryPagerDutyIntake,
		features.FeatureFactoryLinearIntake,
	} {
		require.NoError(t, models.EnableExperimentalFeatureInTransaction(db, organizationID, featureID))
	}
}
