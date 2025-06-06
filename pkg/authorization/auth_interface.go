package authorization

type AuthorizationServiceInterface interface {
	// Permission checking
	CheckCanvasPermission(userID, canvasID, resource, action string) (bool, error)
	CheckOrganizationPermission(userID, orgID, resource, action string) (bool, error)

	// Group management
	CreateGroup(orgID string, groupName string, role string) error
	AddUserToGroup(orgID string, userID string, group string) error
	RemoveUserFromGroup(orgID string, userID string, group string) error
	GetGroupUsers(orgID string, group string) ([]string, error)
	GetGroups(orgID string) ([]string, error)
	GetGroupRoles(orgID string, group string) ([]string, error)

	// Role management
	AssignRole(userID, role, domainID string, domainType string) error
	RemoveRole(userID, role, domainID string, domainType string) error
	GetOrgUsersForRole(role string, orgID string) ([]string, error)
	GetCanvasUsersForRole(role string, canvasID string) ([]string, error)

	// Setup methods
	SetupOrganizationRoles(orgID string) error
	SetupCanvasRoles(canvasID string) error
	CreateOrganizationOwner(userID, orgID string) error

	// User access queries
	GetAccessibleOrgsForUser(userID string) ([]string, error)
	GetAccessibleCanvasesForUser(userID string) ([]string, error)
	GetUserRolesForOrg(userID string, orgID string) ([]string, error)
	GetUserRolesForCanvas(userID string, canvasID string) ([]string, error)
}
