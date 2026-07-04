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
const PropertyAppID = "appID"
const PropertyAppSlug = "appSlug"
const PropertyAppURL = "appURL"
const PropertyAppInstallationID = "appInstallationID"
const PropertyAppInstallationURL = "appInstallationURL"
const PropertyAppClientID = "appClientID"
const PropertyAppState = "appState"

/*
 * Two authentication methods are supported:
 * - Personal Access Token (PAT)
 * - GitHub App
 */
const PropertyAuthMethod = "authMethod"
const AuthMethodPAT = "pat"
const AuthMethodApp = "app"
