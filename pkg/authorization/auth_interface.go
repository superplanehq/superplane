package authorization

import (
	"net/http"

	"github.com/casbin/casbin/v2"
)

// AuthorizationService defines the interface for authorization operations
type AuthorizationService interface {
	// Permission checking methods
	CheckPermission(userID, resource, action string) (bool, error)
	CheckCanvasPermission(userID, canvasID, action string) (bool, error)
	CheckOrganizationPermission(userID, orgID, action string) (bool, error)

	// Role management methods
	AssignRole(userID, role, resourceID string) error
	RemoveRole(userID, role, resourceID string) error
	GetUserRoles(userID string) ([]string, error)
	GetUsersForRole(role string) ([]string, error)
	GetCanvasUsers(canvasID string) (map[string][]string, error)

	// Permission management methods
	AddPermission(role, resource, action string) error
	RemovePermission(role, resource, action string) error

	// Convenience methods
	CreateOrganizationOwner(userID, orgID string) error
	InviteUserToCanvas(userID, canvasID, role string) error

	// Middleware and utility methods
	Middleware() func(next http.Handler) http.Handler
	GetEnforcer() *casbin.Enforcer
}
