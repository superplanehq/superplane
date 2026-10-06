package factories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

// grantIntakeCatalogAccess adds the organization to every implemented catalog
// entry, so tests about intake behavior do not depend on maturity statuses.
func grantIntakeCatalogAccess(t *testing.T, organizationID uuid.UUID) {
	t.Helper()

	db := database.DB(t.Context())
	entries, err := models.ListIntakeCatalogEntries(db)
	require.NoError(t, err)
	for i := range entries {
		if !entries[i].Implemented() {
			continue
		}
		require.NoError(t, entries[i].AddOrganization(db, organizationID))
	}
}
