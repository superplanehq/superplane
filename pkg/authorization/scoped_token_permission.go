package authorization

import (
	"encoding/json"
	"slices"

	"github.com/superplanehq/superplane/pkg/jwt"
)

func hasRequiredScopedTokenPermissionForScopes(
	tokenScopesJSON string,
	pathParams map[string]string,
	rule AuthorizationRule,
) bool {
	if tokenScopesJSON == "" {
		return true
	}

	var scopes []string
	if err := json.Unmarshal([]byte(tokenScopesJSON), &scopes); err != nil {
		return false
	}

	return HasScopedTokenPermission(scopes, pathParams, rule)
}

// HasScopedTokenPermission reports whether scopes allow rule's resource and
// action on the resources named by pathParams.
//
// AuthorizeHTTP applies this to every route served by the gRPC gateway, reading
// the scopes back out of the x-Token-Scopes header. Routes registered directly
// on the router never reach that middleware, so they call this instead. Callers
// must only reach here with a restricted credential: an empty scope list means
// "no permissions", not "unrestricted".
func HasScopedTokenPermission(scopes []string, pathParams map[string]string, rule AuthorizationRule) bool {
	permissions := jwt.PermissionsFromScopes(scopes)
	if len(permissions) == 0 {
		return false
	}

	actions := rule.AllowedActions()
	for _, permission := range permissions {
		if permission.ResourceType != rule.Resource || !slices.Contains(actions, permission.Action) {
			continue
		}

		if len(permission.Resources) == 0 {
			return true
		}

		resourceIDs := resourceIDsFromPathParams(pathParams, rule.ResourcePathParams)
		if len(resourceIDs) == 0 {
			continue
		}

		for _, resourceID := range resourceIDs {
			if slices.Contains(permission.Resources, resourceID) {
				return true
			}
		}
	}

	return false
}

func permissionsFromScopedTokenScopes(tokenScopesJSON string) ([]jwt.Permission, error) {
	var scopes []string
	if err := json.Unmarshal([]byte(tokenScopesJSON), &scopes); err != nil {
		return nil, err
	}

	return jwt.PermissionsFromScopes(scopes), nil
}
