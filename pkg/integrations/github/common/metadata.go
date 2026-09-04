package common

import "slices"

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
}

type PendingInstallation struct {
	ID           string `mapstructure:"id" json:"id"`
	AccountLogin string `mapstructure:"accountLogin" json:"accountLogin"`
	AccountType  string `mapstructure:"accountType" json:"accountType"`
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
