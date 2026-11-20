package authorization

import "context"

type allowingPermissionChecker struct{}

func (allowingPermissionChecker) CheckOrganizationPermission(_ context.Context, _, _, _, _ string) (bool, error) {
	return true, nil
}
