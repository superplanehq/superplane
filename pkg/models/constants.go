package models

import "fmt"

const (
	ProviderGitHub = "github"
	ProviderGoogle = "google"

	// ProviderPassword and ProviderMagicCode identify the non-OAuth login
	// methods in the account session's provider claim. They are not OAuth
	// SSO providers and are therefore never gated by an organization's
	// allowed_providers policy (see IsOAuthProvider).
	ProviderPassword  = "password"
	ProviderMagicCode = "magic_code"

	DomainTypeOrganization = "org"

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

// IsOAuthProvider reports whether the given provider is an OAuth SSO
// provider. Only OAuth providers are subject to an organization's
// allowed_providers policy; password and magic-code logins are governed by
// installation-level settings instead.
func IsOAuthProvider(provider string) bool {
	return provider == ProviderGitHub || provider == ProviderGoogle
}

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
