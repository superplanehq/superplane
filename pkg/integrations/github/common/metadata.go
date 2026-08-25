package common

type Metadata struct {
	InstallationID string            `mapstructure:"installationId" json:"installationId"`
	State          string            `mapstructure:"state" json:"state"`
	Owner          string            `mapstructure:"owner" json:"owner"`
	Repositories   []Repository      `mapstructure:"repositories" json:"repositories"`
	GitHubApp      GitHubAppMetadata `mapstructure:"githubApp" json:"githubApp"`
	// HostedApp is true when this connection installs SuperPlane's public
	// GitHub App. Credentials stay on the process, not on the integration.
	HostedApp bool `mapstructure:"hostedApp" json:"hostedApp"`
}

type GitHubAppMetadata struct {
	ID       int64  `mapstructure:"id" json:"id"`
	Slug     string `mapstructure:"slug" json:"slug"`
	ClientID string `mapstructure:"clientId" json:"clientId"`
}
