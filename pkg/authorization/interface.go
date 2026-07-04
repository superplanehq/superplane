package authorization

import (
	"context"

	"gorm.io/gorm"
)

type PermissionChecker interface {
	CheckOrganizationPermission(ctx context.Context, userID, orgID, resource, action string) (bool, error)
	IsValidPermission(domainType string, permission *Permission) bool
}

// Group management interface
type GroupManager interface {
	CreateGroup(domainID string, domainType string, groupName string, role string, displayName string, description string) error
	DeleteGroup(domainID string, domainType string, groupName string) error
	UpdateGroup(domainID string, domainType string, groupName string, newRole string, displayName string, description string) error
	AddUserToGroup(domainID string, domainType string, userID string, group string) error
	RemoveUserFromGroup(domainID string, domainType string, userID string, group string) error
	GetGroupUsers(ctx context.Context, domainID string, domainType string, group string) ([]string, error)
	GetUserGroups(ctx context.Context, domainID string, domainType string, userID string) ([]string, error)
	GetGroups(ctx context.Context, domainID string, domainType string) ([]string, error)
	GetGroupRole(ctx context.Context, domainID string, domainType string, group string) (string, error)
}

// Role management interface
type RoleManager interface {
	AssignRole(userID, role, domainID string, domainType string) error
	RemoveRole(userID, role, domainID string, domainType string) error
	GetOrgUsersForRole(ctx context.Context, role string, orgID string) ([]string, error)
}

// Setup and initialization interface
type AuthorizationSetup interface {
	SetupOrganization(tx *gorm.DB, orgID, ownerID string) error
	DestroyOrganization(tx *gorm.DB, orgID string) error
}

// User access and role query interface
type UserAccessQuery interface {
	GetUserRolesForOrg(ctx context.Context, userID string, orgID string) ([]*RoleDefinition, error)
}

// Role definition and hierarchy interface
type RoleDefinitionQuery interface {
	GetRoleDefinition(ctx context.Context, roleName string, domainType string, domainID string) (*RoleDefinition, error)
	GetAllRoleDefinitions(ctx context.Context, domainType string, domainID string) ([]*RoleDefinition, error)
	GetRolePermissions(ctx context.Context, roleName string, domainType string, domainID string) ([]*Permission, error)
	GetRoleHierarchy(ctx context.Context, roleName string, domainType string, domainID string) ([]string, error)
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
