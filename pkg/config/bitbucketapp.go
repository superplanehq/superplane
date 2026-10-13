package config

import (
	"os"
	"strings"
)

const (
	EnvBitbucketClientID         = "BITBUCKET_CLIENT_ID"
	EnvBitbucketClientSecret     = "BITBUCKET_CLIENT_SECRET"
	EnvBitbucketForgeAppID       = "SUPERPLANE_BITBUCKET_FORGE_APP_ID"
	EnvBitbucketForgeInstallURL  = "SUPERPLANE_BITBUCKET_FORGE_INSTALL_URL"
	EnvBitbucketForgeJWKSURL     = "SUPERPLANE_BITBUCKET_FORGE_JWKS_URL"
	DefaultBitbucketForgeJWKSURL = "https://forge.cdn.prod.atlassian-dev.net/.well-known/jwks.json"
)

// BitbucketForgeAppConfig is SuperPlane Cloud's public Bitbucket Forge app.
// The process holds the app id and the install link. Tokens arrive from Forge
// and are not part of this config.
type BitbucketForgeAppConfig struct {
	AppID      string
	InstallURL string
	JWKSURL    string
}

// LoadBitbucketForgeAppConfig reads the public Forge app from the process
// environment. Self-hosted leaves these empty. Enabled() is false unless the
// app id and the install link are both set.
func LoadBitbucketForgeAppConfig() BitbucketForgeAppConfig {
	appID := strings.TrimSpace(os.Getenv(EnvBitbucketForgeAppID))
	installURL := strings.TrimSpace(os.Getenv(EnvBitbucketForgeInstallURL))
	jwksURL := strings.TrimSpace(os.Getenv(EnvBitbucketForgeJWKSURL))
	if jwksURL == "" {
		jwksURL = DefaultBitbucketForgeJWKSURL
	}
	if appID == "" || installURL == "" {
		return BitbucketForgeAppConfig{}
	}
	return BitbucketForgeAppConfig{
		AppID:      appID,
		InstallURL: installURL,
		JWKSURL:    jwksURL,
	}
}

// Enabled reports whether Cloud holds a complete public Bitbucket Forge app.
func (c BitbucketForgeAppConfig) Enabled() bool {
	return c.AppID != "" && c.InstallURL != ""
}
