package enterprise

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/models"
)

// DatabaseGrantLookup reads organization Enterprise grants from the database.
type DatabaseGrantLookup struct{}

func (DatabaseGrantLookup) HasGrant(ctx context.Context, orgID uuid.UUID, feature licensing.Feature) (bool, error) {
	return models.HasOrganizationEnterpriseFeature(database.DB(ctx), orgID, string(feature))
}
