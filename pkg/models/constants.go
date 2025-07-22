package models

import "fmt"

const (
	ProviderGitHub = "github"
	ScopeUser      = "user"

	DomainOrg    = "org"
	DomainCanvas = "canvas"

	DisplayNameOwner  = "Owner"
	DisplayNameAdmin  = "Admin"
	DisplayNameViewer = "Viewer"

	RoleOrgOwner  = "org_owner"
	RoleOrgAdmin  = "org_admin"
	RoleOrgViewer = "org_viewer"

	RoleCanvasOwner  = "canvas_owner"
	RoleCanvasAdmin  = "canvas_admin"
	RoleCanvasViewer = "canvas_viewer"

	// Role descriptions
	DescOrgOwner    = "Complete control over the organization including settings and deletion"
	DescOrgAdmin    = "Full management access to organization resources including canvases and users"
	DescOrgViewer   = "Read-only access to organization resources"
	DescCanvasOwner = "Complete control over the canvas including member management"
	DescCanvasAdmin = "Full management access to canvas resources including stages and events"
	DescCanvasViewer = "Read-only access to canvas resources"

	// Metadata descriptions
	MetaDescOrgOwner    = "Full control over organization settings, billing, and member management."
	MetaDescOrgAdmin    = "Can manage canvases, users, groups, and roles within the organization."
	MetaDescOrgViewer   = "Read-only access to organization resources and information."
	MetaDescCanvasOwner = "Full control over canvas settings, members, and deletion."
	MetaDescCanvasAdmin = "Can manage stages, events, connections, and secrets within the canvas."
	MetaDescCanvasViewer = "Read-only access to canvas resources and execution information."
)

func ValidateDomainType(domainType string) error {
	if domainType != DomainOrg && domainType != DomainCanvas {
		return fmt.Errorf("invalid domain type %s", domainType)
	}
	return nil
}

func FormatDomain(domainType, domainID string) string {
	return fmt.Sprintf("%s:%s", domainType, domainID)
}

func PrefixUser(userID string) string {
	return fmt.Sprintf("user:%s", userID)
}

func PrefixGroup(groupName string) string {
	return fmt.Sprintf("group:%s", groupName)
}

func PrefixRole(role string) string {
	return fmt.Sprintf("role:%s", role)
}