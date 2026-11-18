package authorization

import (
	"context"

	"github.com/superplanehq/superplane/pkg/telemetry"
)

type organizationPermissionChecker interface {
	CheckOrganizationPermission(ctx context.Context, userID, orgID, resource, action string) (bool, error)
}

func checkOrganizationPermission(ctx context.Context, auth organizationPermissionChecker, userID, orgID, resource, action string) (bool, error) {
	var err error
	ctx, done := telemetry.Span(ctx, "auth.check_organization_permission")
	defer done(&err)

	allowed, err := auth.CheckOrganizationPermission(ctx, userID, orgID, resource, action)
	if err != nil {
		return false, err
	}

	return allowed, nil
}
