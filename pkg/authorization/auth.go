package authorization

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

const (
	RoleOrgOwner  = "org:owner"
	RoleOrgAdmin  = "org:admin"
	RoleOrgMember = "org:member"

	RoleCanvasOwner       = "canvas:owner"
	RoleCanvasAdmin       = "canvas:admin"
	RoleCanvasDeveloper   = "canvas:developer"
	RoleCanvasContributor = "canvas:contributor"
	RoleCanvasViewer      = "canvas:viewer"

	ActionRead   = "read"
	ActionWrite  = "write"
	ActionCreate = "create"
	ActionDelete = "delete"
	ActionAdmin  = "admin"
)
const (
	ResourceOrganization = "organization"
	ResourceCanvas       = "canvas"
	ResourceStage        = "stage"
	ResourceExecution    = "execution"
	ResourceEventSource  = "event_source"
	ResourceSecret       = "secret"
	ResourceUser         = "user"
)

// make sure AuthService implements AuthorizationService
var _ AuthorizationService = (*AuthService)(nil)

type AuthService struct {
	enforcer *casbin.Enforcer
	db       *gorm.DB
}

func NewAuthService() (*AuthService, error) {
	adapter, err := gormadapter.NewAdapterByDB(database.Conn())
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin adapter: %w", err)
	}

	enforcer, err := casbin.NewEnforcer("../config/rbac_model.conf", adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}
	enforcer.EnableAutoSave(true)

	if err := enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("failed to load policies: %w", err)
	}

	service := &AuthService{
		enforcer: enforcer,
		db:       database.Conn(),
	}

	if err := service.initializeDefaultPolicies(); err != nil {
		log.Warnf("Failed to initialize default policies: %v", err)
	}

	return service, nil
}

func (a *AuthService) CheckPermission(userID, resource, action string) (bool, error) {
	allowed, err := a.enforcer.Enforce(userID, resource, action)
	if err != nil {
		return false, fmt.Errorf("permission check failed: %w", err)
	}
	return allowed, nil
}

func (a *AuthService) CheckCanvasPermission(userID, canvasID, action string) (bool, error) {
	// Check if user has any canvas roles for this canvas and if those roles have the required permission
	canvasRoles := []string{
		RoleCanvasOwner,
		RoleCanvasAdmin,
		RoleCanvasDeveloper,
		RoleCanvasContributor,
		RoleCanvasViewer,
	}

	for _, role := range canvasRoles {
		resourceRole := fmt.Sprintf("%s:%s", role, canvasID)
		hasRole, err := a.enforcer.HasRoleForUser(userID, resourceRole)
		if err != nil {
			continue
		}
		if hasRole {
			// Check if this role has permission for the canvas resource
			// This will use Casbin's built-in role hierarchy through the base role
			roleAllowed, err := a.enforcer.Enforce(role, ResourceCanvas, action)
			if err != nil {
				continue
			}
			if roleAllowed {
				return true, nil
			}
		}
	}

	return false, nil
}

func (a *AuthService) CheckOrganizationPermission(userID, orgID, action string) (bool, error) {
	resource := fmt.Sprintf("organization:%s", orgID)
	return a.CheckPermission(userID, resource, action)
}

func (a *AuthService) AssignRole(userID, role, resourceID string) error {
	log.Infof("Assigning role %s to user %s for resource %s", role, userID, resourceID)

	if resourceID == "" {
		_, err := a.enforcer.AddRoleForUser(userID, role)
		if err != nil {
			return fmt.Errorf("failed to assign role: %w", err)
		}
	} else {
		resourceRole := fmt.Sprintf("%s:%s", role, resourceID)
		_, err := a.enforcer.AddRoleForUser(userID, resourceRole)
		if err != nil {
			return fmt.Errorf("failed to assign resource role: %w", err)
		}
	}

	return nil
}

func (a *AuthService) RemoveRole(userID, role, resourceID string) error {
	if resourceID == "" {
		_, err := a.enforcer.DeleteRoleForUser(userID, role)
		if err != nil {
			return fmt.Errorf("failed to remove role: %w", err)
		}
	} else {
		resourceRole := fmt.Sprintf("%s:%s", role, resourceID)
		_, err := a.enforcer.DeleteRoleForUser(userID, resourceRole)
		if err != nil {
			return fmt.Errorf("failed to remove resource role: %w", err)
		}
	}

	return nil
}

func (a *AuthService) GetUserRoles(userID string) ([]string, error) {
	roles, err := a.enforcer.GetRolesForUser(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	return roles, nil
}

func (a *AuthService) GetUsersForRole(role string) ([]string, error) {
	users, err := a.enforcer.GetUsersForRole(role)
	if err != nil {
		return nil, fmt.Errorf("failed to get users for role: %w", err)
	}
	return users, nil
}

func (a *AuthService) GetCanvasUsers(canvasID string) (map[string][]string, error) {
	canvasUsers := make(map[string][]string)

	canvasRoles := []string{
		RoleCanvasOwner,
		RoleCanvasAdmin,
		RoleCanvasDeveloper,
		RoleCanvasContributor,
		RoleCanvasViewer,
	}

	for _, role := range canvasRoles {
		resourceRole := fmt.Sprintf("%s:%s", role, canvasID)
		users, err := a.enforcer.GetUsersForRole(resourceRole)
		if err != nil {
			return nil, fmt.Errorf("failed to get users for canvas role %s: %w", role, err)
		}
		if len(users) > 0 {
			canvasUsers[role] = users
		}
	}

	return canvasUsers, nil
}

func (a *AuthService) AddPermission(role, resource, action string) error {
	_, err := a.enforcer.AddPolicy(role, resource, action)
	if err != nil {
		return fmt.Errorf("failed to add permission: %w", err)
	}
	return err
}

func (a *AuthService) RemovePermission(role, resource, action string) error {
	_, err := a.enforcer.RemovePolicy(role, resource, action)
	if err != nil {
		return fmt.Errorf("failed to remove permission: %w", err)
	}
	return err
}

func (a *AuthService) CreateOrganizationOwner(userID, orgID string) error {
	return a.AssignRole(userID, RoleOrgOwner, orgID)
}

func (a *AuthService) InviteUserToCanvas(userID, canvasID, role string) error {
	validRoles := []string{
		RoleCanvasOwner,
		RoleCanvasAdmin,
		RoleCanvasDeveloper,
		RoleCanvasContributor,
		RoleCanvasViewer,
	}

	roleValid := false
	for _, validRole := range validRoles {
		if role == validRole {
			roleValid = true
			break
		}
	}

	if !roleValid {
		return fmt.Errorf("invalid canvas role: %s", role)
	}

	return a.AssignRole(userID, role, canvasID)
}

func (a *AuthService) initializeDefaultPolicies() error {
	hasAnyPolicy, err := a.enforcer.HasPolicy("org:owner", "organization", "admin")
	if err != nil {
		return fmt.Errorf("failed to check for policies: %w", err)
	}

	if hasAnyPolicy {
		return nil // Skip initialization
	}

	orgPolicies := [][]string{
		{RoleOrgOwner, ResourceOrganization, ActionAdmin},
		{RoleOrgOwner, ResourceCanvas, ActionAdmin},
		{RoleOrgOwner, ResourceUser, ActionAdmin},
		{RoleOrgOwner, ResourceSecret, ActionAdmin},

		{RoleOrgAdmin, ResourceCanvas, ActionAdmin},
		{RoleOrgAdmin, ResourceUser, ActionWrite},
		{RoleOrgAdmin, ResourceSecret, ActionWrite},
		{RoleOrgAdmin, ResourceOrganization, ActionRead},

		{RoleOrgMember, ResourceOrganization, ActionRead},
		{RoleOrgMember, ResourceCanvas, ActionRead},
	}

	canvasPolicies := [][]string{
		// Canvas Owner - should have ALL permissions
		{RoleCanvasOwner, ResourceCanvas, ActionRead},
		{RoleCanvasOwner, ResourceCanvas, ActionWrite},
		{RoleCanvasOwner, ResourceCanvas, ActionCreate},
		{RoleCanvasOwner, ResourceCanvas, ActionDelete},
		{RoleCanvasOwner, ResourceCanvas, ActionAdmin},
		{RoleCanvasOwner, ResourceStage, ActionAdmin},
		{RoleCanvasOwner, ResourceExecution, ActionAdmin},
		{RoleCanvasOwner, ResourceEventSource, ActionAdmin},

		// Canvas Admin - inherits from developer + admin permissions
		{RoleCanvasAdmin, ResourceCanvas, ActionRead},
		{RoleCanvasAdmin, ResourceCanvas, ActionWrite},
		{RoleCanvasAdmin, ResourceCanvas, ActionAdmin},
		{RoleCanvasAdmin, ResourceStage, ActionWrite},
		{RoleCanvasAdmin, ResourceExecution, ActionWrite},
		{RoleCanvasAdmin, ResourceEventSource, ActionWrite},

		// Canvas Developer - can write stages and create executions
		{RoleCanvasDeveloper, ResourceCanvas, ActionRead},
		{RoleCanvasDeveloper, ResourceCanvas, ActionWrite},
		{RoleCanvasDeveloper, ResourceStage, ActionRead},
		{RoleCanvasDeveloper, ResourceStage, ActionWrite},
		{RoleCanvasDeveloper, ResourceExecution, ActionRead},
		{RoleCanvasDeveloper, ResourceExecution, ActionCreate},
		{RoleCanvasDeveloper, ResourceEventSource, ActionRead},

		// Canvas Contributor - can create executions
		{RoleCanvasContributor, ResourceCanvas, ActionRead},
		{RoleCanvasContributor, ResourceStage, ActionRead},
		{RoleCanvasContributor, ResourceExecution, ActionRead},
		{RoleCanvasContributor, ResourceExecution, ActionCreate},
		{RoleCanvasContributor, ResourceEventSource, ActionRead},

		// Canvas Viewer - read only
		{RoleCanvasViewer, ResourceCanvas, ActionRead},
		{RoleCanvasViewer, ResourceStage, ActionRead},
		{RoleCanvasViewer, ResourceExecution, ActionRead},
		{RoleCanvasViewer, ResourceEventSource, ActionRead},
	}

	roleHierarchy := [][]string{
		{RoleOrgOwner, RoleOrgAdmin},
		{RoleOrgAdmin, RoleOrgMember},

		{RoleCanvasOwner, RoleCanvasAdmin},
		{RoleCanvasAdmin, RoleCanvasDeveloper},
		{RoleCanvasDeveloper, RoleCanvasContributor},
		{RoleCanvasContributor, RoleCanvasViewer},
	}

	for _, policy := range orgPolicies {
		if _, err := a.enforcer.AddPolicy(policy[0], policy[1], policy[2]); err != nil {
			log.Warnf("Failed to add policy %v: %v", policy, err)
		}
	}

	for _, policy := range canvasPolicies {
		if _, err := a.enforcer.AddPolicy(policy[0], policy[1], policy[2]); err != nil {
			log.Warnf("Failed to add policy %v: %v", policy, err)
		}
	}

	for _, inheritance := range roleHierarchy {
		if _, err := a.enforcer.AddGroupingPolicy(inheritance[0], inheritance[1]); err != nil {
			log.Warnf("Failed to add role inheritance %v: %v", inheritance, err)
		}
	}

	return err
}

func (a *AuthService) GetEnforcer() *casbin.Enforcer {
	return a.enforcer
}

func (a *AuthService) Middleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := extractUserIDFromRequest(r)
			if userID == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			resource := determineResource(r.URL.Path)
			action := mapHTTPMethodToAction(r.Method)

			allowed, err := a.CheckPermission(userID, resource, action)
			if err != nil {
				log.Errorf("Authorization check failed: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			if !allowed {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func extractUserIDFromRequest(r *http.Request) string {
	return r.Header.Get("X-User-ID")
}

func determineResource(path string) string {
	if strings.Contains(path, "/stage") {
		return ResourceStage
	}
	if strings.Contains(path, "/execution") {
		return ResourceExecution
	}
	if strings.Contains(path, "/sources") {
		return ResourceEventSource
	}
	if strings.Contains(path, "/canvas") {
		return ResourceCanvas
	}
	return "unknown"
}

func mapHTTPMethodToAction(method string) string {
	switch method {
	case "GET":
		return ActionRead
	case "POST":
		return ActionCreate
	case "PUT", "PATCH":
		return ActionWrite
	case "DELETE":
		return ActionDelete
	default:
		return ActionRead
	}
}
