package models

import "fmt"

const (
	ProviderGitHub   = "github"
	ProviderGoogle   = "google"
	ProviderPassword = "password"

	DomainTypeOrganization = "org"

	DisplayNameOwner      = "Owner"
	DisplayNameAdmin      = "Admin"
	DisplayNameMaintainer = "Maintainer"
	DisplayNameOperator   = "Operator"

	RoleOrgAdmin      = "org_admin"
	RoleOrgMaintainer = "org_maintainer"
	RoleOrgOperator   = "org_operator"

	// Role descriptions
	DescOrgAdmin      = "Manage members, billing, and organization settings"
	DescOrgMaintainer = "Create and edit automations, integrations, and models"
	DescOrgOperator   = "Create tasks and interact with them"

	// Metadata descriptions
	MetaDescOrgAdmin      = "Can manage members, billing, and organization settings."
	MetaDescOrgMaintainer = "Can create and edit automations, set integrations, and change models."
	MetaDescOrgOperator   = "Can create tasks and interact with them."

	// User types
	UserTypeHuman  = "human"
	UserTypeAPIKey = "api_key"
)

var DefaultOrganizationRoles = []string{
	RoleOrgAdmin,
	RoleOrgMaintainer,
	RoleOrgOperator,
}

var (
	ErrNameAlreadyUsed         = fmt.Errorf("name already used")
	ErrSlugAlreadyUsed         = fmt.Errorf("slug already used")
	ErrInvitationAlreadyExists = fmt.Errorf("invitation already exists")
)

func ValidateDomainType(domainType string) error {
	if domainType != DomainTypeOrganization {
		return fmt.Errorf("invalid domain type %s", domainType)
	}
	return nil
}

func FormatDomain(domainType, domainID string) string {
	return fmt.Sprintf("%s:%s", domainType, domainID)
}

func PrefixUser(userID string) string {
	return fmt.Sprintf("/users/%s", userID)
}

func PrefixGroup(groupName string) string {
	return fmt.Sprintf("/groups/%s", groupName)
}

func PrefixRole(role string) string {
	return fmt.Sprintf("/roles/%s", role)
}

func IsDefaultOrganizationRole(roleName string) bool {
	for _, role := range DefaultOrganizationRoles {
		if role == roleName {
			return true
		}
	}
	return false
}
