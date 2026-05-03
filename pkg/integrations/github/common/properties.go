package common

/*
 * An integration can be connected to:
 * - A user account
 * - An organization
 */
const PropertyOwnerType = "ownerType"
const PropertyOwner = "owner"
const OwnerTypeUser = "User Account"
const OwnerTypeOrganization = "Organization"

/*
 * When connecting to the owner with a GitHub App,
 * these are the properties we get back from GitHub,
 * through the app creation / installation flow.
 */
const PropertyAppID = "GitHub App ID"
const PropertyAppSlug = "GitHub App Slug"
const PropertyAppClientID = "GitHub App Client ID"
const PropertyAppInstallationID = "GitHub App Installation ID"
const PropertyAppState = "GitHub App State"

/*
 * Two authentication methods are supported:
 * - Personal Access Token (PAT)
 * - GitHub App
 */
const PropertyAuthMethod = "Authentication Method"
const AuthMethodPAT = "Personal Access Token"
const AuthMethodGitHubApp = "GitHub App"
