package secrets

import (
	"github.com/google/uuid"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
)

// ResolveSecretDomain determines the (domainType, domainID) pair that a
// secrets request should operate on.
//
// Secrets are always authorized against the caller's organization
// permissions (there's no per-canvas RBAC in this codebase), so the
// organization id must come from the trusted request context
// (x-organization-id), never from client input. That's why, for the
// organization domain, we ignore any domain id the client might have sent
// and always use organizationID.
//
// For the canvas domain, the client-supplied domain id is the canvas
// ("app") id. We look it up scoped to the caller's organization, so a
// canvas belonging to a different organization is treated as not found -
// this is what prevents a caller from reading/writing another org's
// canvas-scoped secrets.
//
// An unspecified domain type is treated as the organization domain, to stay
// backwards compatible with clients that don't set domain_type/domain_id.
func ResolveSecretDomain(organizationID string, domainType pbAuth.DomainType, domainID string) (string, string, error) {
	switch domainType {
	case pbAuth.DomainType_DOMAIN_TYPE_CANVAS:
		canvasID, err := uuid.Parse(domainID)
		if err != nil {
			return "", "", grpcerrors.InvalidArgument(err, "invalid app id")
		}

		orgID, err := uuid.Parse(organizationID)
		if err != nil {
			return "", "", grpcerrors.Internal(err, "invalid organization id")
		}

		if _, err := models.FindCanvas(orgID, canvasID); err != nil {
			return "", "", grpcerrors.NotFound(err, "app not found")
		}

		return models.DomainTypeCanvas, canvasID.String(), nil

	case pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION, pbAuth.DomainType_DOMAIN_TYPE_UNSPECIFIED:
		return models.DomainTypeOrganization, organizationID, nil

	default:
		return "", "", grpcerrors.InvalidArgument(nil, "invalid domain type")
	}
}
