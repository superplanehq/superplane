package models

import "fmt"

const (
	ProviderGitHub = "github"
	ProviderGoogle = "google"

	DomainTypeOrganization = "org"
	// DomainTypeCanvas scopes a resource to a single canvas (a.k.a. "app" in
	// user-facing CLI/UI copy). Currently only used by secrets.
	DomainTypeCanvas = "canvas"

	DisplayNameOwner  = "Owner"
	DisplayNameAdmin  = "Admin"
	DisplayNameViewer = "Viewer"

	RoleOrgOwner  = "org_owner"
	RoleOrgAdmin  = "org_admin"
	RoleOrgViewer = "org_viewer"

	// Role descriptions
	DescOrgOwner  = "Complete control over the organization including settings and deletion"
	DescOrgAdmin  = "Full management access to organization resources including canvases and users"
	DescOrgViewer = "Read-only access to organization resources"

	// Metadata descriptions
	MetaDescOrgOwner  = "Full control over organization settings, billing, and member management."
	MetaDescOrgAdmin  = "Can manage canvases, users, groups, and roles within the organization."
	MetaDescOrgViewer = "Read-only access to organization resources and information."

	// User types
	UserTypeHuman  = "human"
	UserTypeAPIKey = "api_key"
)

var (
	ErrNameAlreadyUsed         = fmt.Errorf("name already used")
	ErrInvitationAlreadyExists = fmt.Errorf("invitation already exists")
)

// ValidateDomainType validates domain types used for RBAC (roles/groups),
// which are only ever scoped to an organization. Do not loosen this to
// accept other domain types - use ValidateSecretDomainType for secrets.
func ValidateDomainType(domainType string) error {
	if domainType != DomainTypeOrganization {
		return fmt.Errorf("invalid domain type %s", domainType)
	}
	return nil
}

// ValidateSecretDomainType validates domain types accepted for secrets,
// which can be scoped to either an organization or a single canvas ("app").
func ValidateSecretDomainType(domainType string) error {
	switch domainType {
	case DomainTypeOrganization, DomainTypeCanvas:
		return nil
	default:
		return fmt.Errorf("invalid domain type %s", domainType)
	}
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
