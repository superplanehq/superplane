package organizations

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// resolveOrganizationID resolves ref -- a client-supplied organization
// identifier that may be a UUID or a slug -- to the organization's UUID.
// Every org-scoped action that receives its organization identifier from a
// client request path parameter or header goes through this helper, so that
// slug and UUID URLs both keep working against the same UUID-keyed data.
func resolveOrganizationID(ctx context.Context, ref string) (uuid.UUID, error) {
	organization, err := models.FindOrganizationByIDOrSlug(database.DB(ctx), ref)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, grpcerrors.InvalidArgument(err, "invalid organization id")
		}

		return uuid.Nil, grpcerrors.Internal(err, "failed to resolve organization")
	}

	return organization.ID, nil
}
