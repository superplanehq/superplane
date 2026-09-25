package common

import (
	"slices"
	"strings"
)

type Metadata struct {
	InstallationID string       `mapstructure:"installationId" json:"installationId"`
	State          string       `mapstructure:"state" json:"state"`
	Owner          string       `mapstructure:"owner" json:"owner"`
	Repositories   []Repository `mapstructure:"repositories" json:"repositories"`
	// SelectedRepositories preserves the user's repository selection when an
	// installation temporarily loses access to one of those repositories.
	SelectedRepositories []Repository `mapstructure:"selectedRepositories" json:"selectedRepositories,omitempty"`
	// RepositoryScoped distinguishes new hosted connections from legacy
	// installation-wide connections. Runtime tokens for scoped connections are
	// restricted to the repository IDs in Repositories.
	RepositoryScoped bool              `mapstructure:"repositoryScoped" json:"repositoryScoped,omitempty"`
	GitHubApp        GitHubAppMetadata `mapstructure:"githubApp" json:"githubApp"`
	// HostedApp is true when this connection installs SuperPlane's public
	// GitHub App. Credentials stay on the process, not on the integration.
	HostedApp bool `mapstructure:"hostedApp" json:"hostedApp"`
	// StartedByUserID is the SuperPlane user who started this hosted install.
	// Repository discovery and bind use this user's linked GitHub identity.
	StartedByUserID string `mapstructure:"startedByUserID" json:"startedByUserID,omitempty"`
	// PendingInstallations is the server-verified list of installations and
	// writable repositories available to the linked GitHub identity.
	PendingInstallations []PendingInstallation `mapstructure:"pendingInstallations" json:"pendingInstallations,omitempty"`
	// InstallationsRefreshedAt limits full App installation discovery while the
	// picker polls, without making the verified list permanent.
	InstallationsRefreshedAt string `mapstructure:"installationsRefreshedAt" json:"installationsRefreshedAt,omitempty"`
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
	// StartedByGitHubLogin is the current login for the linked GitHub identity.
	// The request callback does not name the requested organization, so Sync
	// uses this login to find the member's App install request.
	StartedByGitHubLogin string `mapstructure:"startedByGitHubLogin" json:"startedByGitHubLogin,omitempty"`
	// InstallRequestDiscoveryUntil keeps installation discovery active briefly
	// after a closed request leaves the waiting UI. This covers an approval
	// that becomes visible after GitHub removes it from the open request list.
	InstallRequestDiscoveryUntil string `mapstructure:"installRequestDiscoveryUntil" json:"installRequestDiscoveryUntil,omitempty"`
	// SetupReturnPath is the in-app path to open after GitHub setup. Callbacks
	// use it when the browser cookie is missing, for example localhost to ngrok.
	SetupReturnPath string `mapstructure:"setupReturnPath" json:"setupReturnPath,omitempty"`
}

type PendingInstallation struct {
	ID           string       `mapstructure:"id" json:"id"`
	AccountLogin string       `mapstructure:"accountLogin" json:"accountLogin"`
	AccountType  string       `mapstructure:"accountType" json:"accountType"`
	Repositories []Repository `mapstructure:"repositories" json:"repositories,omitempty"`
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

func (m Metadata) PendingRepositories(installationID string) ([]Repository, bool) {
	index := slices.IndexFunc(m.PendingInstallations, func(installation PendingInstallation) bool {
		return installation.ID == installationID
	})
	if index == -1 {
		return nil, false
	}

	return slices.Clone(m.PendingInstallations[index].Repositories), true
}

func (m Metadata) SelectPendingRepositories(installationID string, repositoryIDs []int64) ([]Repository, bool) {
	available, ok := m.PendingRepositories(installationID)
	if !ok || len(repositoryIDs) == 0 {
		return nil, false
	}

	selected := make([]Repository, 0, len(repositoryIDs))
	for _, repositoryID := range repositoryIDs {
		index := slices.IndexFunc(available, func(repository Repository) bool {
			return repository.ID == repositoryID
		})
		if index == -1 {
			return nil, false
		}
		selected = append(selected, available[index])
	}

	return selected, true
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
