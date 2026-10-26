package organizations

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/organizationref"
)

func resolveOrganizationID(ctx context.Context, ref string) (uuid.UUID, error) {
	return organizationref.Resolve(ctx, ref)
}
