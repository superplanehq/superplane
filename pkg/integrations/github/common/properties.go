package common

const (
	//
	// An integration can be connected to:
	// - A user account
	// - An organization
	//
	PropertyOwnerType     = "ownerType"
	PropertyOwner         = "owner"
	OwnerTypeUser         = "User Account"
	OwnerTypeOrganization = "Organization"

	//
	// When connecting to the owner with a GitHub App,
	// these are the properties we get back from GitHub,
	// through the app creation / installation flow.
	//
	PropertyAppID             = "GitHub App ID"
	PropertyAppSlug           = "GitHub App Slug"
	PropertyAppClientID       = "GitHub App Client ID"
	PropertyAppInstallationID = "GitHub App Installation ID"
	PropertyAppState          = "GitHub App State"

	//
	// Two authentication methods are supported:
	// - Personal Access Token (PAT)
	// - GitHub App
	//
	PropertyAuthMethod  = "Authentication Method"
	AuthMethodPAT       = "Personal Access Token"
	AuthMethodGitHubApp = "GitHub App"
)
