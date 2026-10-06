package factories

import (
	"context"
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

	linear, err := models.FindIntakeCatalogEntry(db, models.FactoryIntakeSourceLinearIssues)
	require.NoError(t, err)
	originalStatus := linear.Status
	t.Cleanup(func() {
		// The testing context is canceled when the test finishes.
		restoreDB := database.DB(context.Background())
		status := originalStatus
		require.NoError(t, linear.Update(restoreDB, models.IntakeCatalogPatch{Status: &status}, nil))
	})
	status := models.IntakeStatusBeta
	require.NoError(t, linear.Update(db, models.IntakeCatalogPatch{Status: &status}, nil))
}
