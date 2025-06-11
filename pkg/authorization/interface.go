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
	GetUserRolesForOrg(userID string, orgID string) ([]*RoleDefinition, error)
	GetUserRolesForCanvas(userID string, canvasID string) ([]*RoleDefinition, error)

	GetRoleDefinition(roleName string, domainType string, domainID string) (*RoleDefinition, error)
	GetAllRoleDefinitions(domainType string, domainID string) ([]*RoleDefinition, error)
	GetRolePermissions(roleName string, domainType string, domainID string) ([]*Permission, error)
	GetRoleHierarchy(roleName string, domainType string, domainID string) ([]string, error)
}

type RoleDefinition struct {
	Name         string
	DomainType   string
	Description  string
	Permissions  []*Permission
	InheritsFrom *RoleDefinition
	Readonly     bool
}

type Permission struct {
	Resource   string
	Action     string
	DomainType string
}
