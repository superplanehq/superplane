package common

import (
	"slices"
	"strings"
)

type Metadata struct {
	InstallationID string            `mapstructure:"installationId" json:"installationId"`
	State          string            `mapstructure:"state" json:"state"`
	Owner          string            `mapstructure:"owner" json:"owner"`
	Repositories   []Repository      `mapstructure:"repositories" json:"repositories"`
	GitHubApp      GitHubAppMetadata `mapstructure:"githubApp" json:"githubApp"`
	// HostedApp is true when this connection installs SuperPlane's public
	// GitHub App. Credentials stay on the process, not on the integration.
	HostedApp bool `mapstructure:"hostedApp" json:"hostedApp"`
	// StartedByUserID is the SuperPlane user who started this hosted install.
	// Setup, OAuth, and bind must run as this user when the field is set.
	StartedByUserID string `mapstructure:"startedByUserID" json:"startedByUserID,omitempty"`
	// PendingInstallations is the user-scoped allowlist written after GitHub
	// App user OAuth. Picker bind accepts only these installation ids.
	PendingInstallations []PendingInstallation `mapstructure:"pendingInstallations" json:"pendingInstallations,omitempty"`
	// InstallRequested is true when a non-admin asked a GitHub org admin to
	// install the app. Setup then returned setup_action=request.
	InstallRequested bool `mapstructure:"installRequested" json:"installRequested,omitempty"`
	// InstallRequestedAccount is the GitHub organization (or user) login the
	// member asked an admin to approve.
	InstallRequestedAccount string `mapstructure:"installRequestedAccount" json:"installRequestedAccount,omitempty"`
	// InstallRequests contains every open GitHub App installation request for
	// the member who started this connection. The legacy scalar fields above
	// mirror this collection for compatibility with older clients.
	InstallRequests []InstallRequest `mapstructure:"installRequests" json:"installRequests,omitempty"`
	// StartedByGitHubLogin is the GitHub login of the member who authorized
	// the connect OAuth. The request callback from GitHub does not name the
	// requested organization, so Sync finds that member's App install request
	// through this login.
	StartedByGitHubLogin string `mapstructure:"startedByGitHubLogin" json:"startedByGitHubLogin,omitempty"`
	// SetupReturnPath is the in-app path to open after GitHub setup. Callbacks
	// use it when the browser cookie is missing, for example localhost to ngrok.
	SetupReturnPath string `mapstructure:"setupReturnPath" json:"setupReturnPath,omitempty"`
	// AuthorizeURL is the GitHub user OAuth authorize URL for this connect.
	// The OAuth callback removes the browser action, so the connect screen
	// uses this URL to ask again which GitHub account to use.
	AuthorizeURL string `mapstructure:"authorizeURL" json:"authorizeURL,omitempty"`
}

type PendingInstallation struct {
	ID           string `mapstructure:"id" json:"id"`
	AccountLogin string `mapstructure:"accountLogin" json:"accountLogin"`
	AccountType  string `mapstructure:"accountType" json:"accountType"`
}

type InstallRequest struct {
	ID             string `mapstructure:"id" json:"id,omitempty"`
	AccountLogin   string `mapstructure:"accountLogin" json:"accountLogin,omitempty"`
	RequesterLogin string `mapstructure:"requesterLogin" json:"requesterLogin,omitempty"`
	CreatedAt      string `mapstructure:"createdAt" json:"createdAt,omitempty"`
}

type GitHubAppMetadata struct {
	ID       int64  `mapstructure:"id" json:"id"`
	Slug     string `mapstructure:"slug" json:"slug"`
	ClientID string `mapstructure:"clientId" json:"clientId"`
}

func (m Metadata) AllowsPendingInstallation(installationID string) bool {
	if installationID == "" {
		return false
	}

	return slices.ContainsFunc(m.PendingInstallations, func(pending PendingInstallation) bool {
		return pending.ID == installationID
	})
}

func (m Metadata) AllowsStartedBy(userID string) bool {
	if m.StartedByUserID == "" {
		return true
	}

	return userID != "" && m.StartedByUserID == userID
}

func (m Metadata) HasInstallRequests() bool {
	return len(m.InstallRequests) > 0 || m.InstallRequested
}

func (m Metadata) CurrentInstallRequests() []InstallRequest {
	if len(m.InstallRequests) > 0 {
		return slices.Clone(m.InstallRequests)
	}
	if !m.InstallRequested {
		return nil
	}
	return []InstallRequest{{
		AccountLogin:   strings.TrimSpace(m.InstallRequestedAccount),
		RequesterLogin: strings.TrimSpace(m.StartedByGitHubLogin),
	}}
}

func (m *Metadata) SetInstallRequests(requests []InstallRequest) {
	m.InstallRequests = uniqueInstallRequests(requests)
	m.InstallRequested = len(m.InstallRequests) > 0
	m.InstallRequestedAccount = singleInstallRequestAccount(m.InstallRequests)
}

func (m *Metadata) SetPendingInstallations(installations []PendingInstallation) {
	unique := make([]PendingInstallation, 0, len(installations))
	for _, installation := range installations {
		installation.ID = strings.TrimSpace(installation.ID)
		installation.AccountLogin = strings.TrimSpace(installation.AccountLogin)
		if installation.ID == "" || slices.ContainsFunc(unique, func(existing PendingInstallation) bool {
			return existing.ID == installation.ID
		}) {
			continue
		}
		unique = append(unique, installation)
	}
	m.PendingInstallations = unique
}

func uniqueInstallRequests(requests []InstallRequest) []InstallRequest {
	unique := make([]InstallRequest, 0, len(requests))
	hasKnownRequest := slices.ContainsFunc(requests, func(request InstallRequest) bool {
		return strings.TrimSpace(request.ID) != "" || strings.TrimSpace(request.AccountLogin) != ""
	})
	for _, request := range requests {
		request.ID = strings.TrimSpace(request.ID)
		request.AccountLogin = strings.TrimSpace(request.AccountLogin)
		request.RequesterLogin = strings.TrimSpace(request.RequesterLogin)
		if request.ID == "" && request.AccountLogin == "" && hasKnownRequest {
			continue
		}
		if slices.ContainsFunc(unique, func(existing InstallRequest) bool {
			if request.ID == "" && request.AccountLogin == "" {
				return existing.ID == "" && existing.AccountLogin == ""
			}
			if request.ID != "" && existing.ID != "" {
				return request.ID == existing.ID
			}
			return request.AccountLogin != "" && strings.EqualFold(request.AccountLogin, existing.AccountLogin)
		}) {
			continue
		}
		unique = append(unique, request)
	}
	return unique
}

func singleInstallRequestAccount(requests []InstallRequest) string {
	accounts := []string{}
	for _, request := range requests {
		if request.AccountLogin == "" || slices.ContainsFunc(accounts, func(account string) bool {
			return strings.EqualFold(account, request.AccountLogin)
		}) {
			continue
		}
		accounts = append(accounts, request.AccountLogin)
	}
	if len(accounts) != 1 {
		return ""
	}
	return accounts[0]
}
