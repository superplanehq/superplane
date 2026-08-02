package authorization

import (
	"context"
	"encoding/json"
	"slices"
)

// CheckCanvasAccess verifies organization permission for a canvas-scoped action and
// enforces canvas allowlists from scoped JWT scopes and/or API-key canvas IDs.
// Empty scopedScopes and empty apiKeyCanvasIDs mean the caller is not canvas-scoped.
func CheckCanvasAccess(
	ctx context.Context,
	auth organizationPermissionChecker,
	userID, orgID, canvasID, resource, action string,
	scopedScopes []string,
	apiKeyCanvasIDs []string,
) (bool, error) {
	allowed, err := checkOrganizationPermission(ctx, auth, userID, orgID, resource, action)
	if err != nil || !allowed {
		return allowed, err
	}

	if len(apiKeyCanvasIDs) > 0 && !slices.Contains(apiKeyCanvasIDs, canvasID) {
		return false, nil
	}

	if len(scopedScopes) == 0 {
		return true, nil
	}

	scopesJSON, err := json.Marshal(scopedScopes)
	if err != nil {
		return false, err
	}

	rule := AuthorizationRule{
		Resource:           resource,
		Action:             action,
		ResourcePathParams: []string{CanvasIDPathParam},
	}
	pathParams := map[string]string{CanvasIDPathParam: canvasID}
	if !hasRequiredScopedTokenPermissionForScopes(string(scopesJSON), pathParams, rule) {
		return false, nil
	}

	return true, nil
}
