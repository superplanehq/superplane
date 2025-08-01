package authorization

// Permission checking interface
type PermissionChecker interface {
	CheckCanvasPermission(userID, canvasID, resource, action string) (bool, error)
	CheckOrganizationPermission(userID, orgID, resource, action string) (bool, error)
}

// Group management interface
type GroupManager interface {
	CreateGroup(domainID string, domainType string, groupName string, role string, displayName string, description string) error
	DeleteGroup(domainID string, domainType string, groupName string) error
	UpdateGroup(domainID string, domainType string, groupName string, newRole string, displayName string, description string) error
	AddUserToGroup(domainID string, domainType string, userID string, group string) error
	RemoveUserFromGroup(domainID string, domainType string, userID string, group string) error
	GetGroupUsers(domainID string, domainType string, group string) ([]string, error)
	GetGroups(domainID string, domainType string) ([]string, error)
	GetGroupRole(domainID string, domainType string, group string) (string, error)
}

// Role management interface
type RoleManager interface {
	AssignRole(userID, role, domainID string, domainType string) error
	RemoveRole(userID, role, domainID string, domainType string) error
	GetOrgUsersForRole(role string, orgID string) ([]string, error)
	GetCanvasUsersForRole(role string, canvasID string) ([]string, error)
}

// Setup and initialization interface
type AuthorizationSetup interface {
	SetupOrganizationRoles(orgID string) error
	SetupCanvasRoles(canvasID string) error
	DestroyOrganizationRoles(orgID string) error
	DestroyCanvasRoles(canvasID string) error
	CreateOrganizationOwner(userID, orgID string) error
}

// User access and role query interface
type UserAccessQuery interface {
	GetAccessibleOrgsForUser(userID string) ([]string, error)
	GetAccessibleCanvasesForUser(userID string) ([]string, error)
	GetUserRolesForOrg(userID string, orgID string) ([]*RoleDefinition, error)
	GetUserRolesForCanvas(userID string, canvasID string) ([]*RoleDefinition, error)
}

// Role definition and hierarchy interface
type RoleDefinitionQuery interface {
	GetRoleDefinition(roleName string, domainType string, domainID string) (*RoleDefinition, error)
	GetAllRoleDefinitions(domainType string, domainID string) ([]*RoleDefinition, error)
	GetRolePermissions(roleName string, domainType string, domainID string) ([]*Permission, error)
	GetRoleHierarchy(roleName string, domainType string, domainID string) ([]string, error)
}

// Custom role management interface
type CustomRoleManager interface {
	CreateCustomRole(domainID string, roleDefinition *RoleDefinition) error
	UpdateCustomRole(domainID string, roleDefinition *RoleDefinition) error
	DeleteCustomRole(domainID string, domainType string, roleName string) error
	IsDefaultRole(roleName string, domainType string) bool
}

// Authorization interface
type Authorization interface {
	PermissionChecker
	GroupManager
	RoleManager
	AuthorizationSetup
	UserAccessQuery
	RoleDefinitionQuery
	CustomRoleManager
}

type RoleDefinition struct {
	Name         string
	DisplayName  string
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
