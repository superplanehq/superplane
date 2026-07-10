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

	permissions, err := permissionsFromScopedTokenScopes(tokenScopesJSON)
	if err != nil || len(permissions) == 0 {
		return false
	}

	actions := allowedActions(rule)
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

func allowedActions(rule AuthorizationRule) []string {
	return append([]string{rule.Action}, rule.LegacyActions...)
}

func permissionsFromScopedTokenScopes(tokenScopesJSON string) ([]jwt.Permission, error) {
	var scopes []string
	if err := json.Unmarshal([]byte(tokenScopesJSON), &scopes); err != nil {
		return nil, err
	}

	return jwt.PermissionsFromScopes(scopes), nil
}
