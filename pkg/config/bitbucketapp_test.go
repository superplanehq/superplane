package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadBitbucketForgeAppConfig(t *testing.T) {
	t.Setenv(EnvBitbucketForgeAppID, "")
	t.Setenv(EnvBitbucketForgeInstallURL, "")
	assert.False(t, LoadBitbucketForgeAppConfig().Enabled())

	t.Setenv(EnvBitbucketForgeAppID, "ari:cloud:ecosystem::app/example")
	t.Setenv(EnvBitbucketForgeInstallURL, "https://developer.atlassian.com/console/install/example")
	cfg := LoadBitbucketForgeAppConfig()
	assert.True(t, cfg.Enabled())
	assert.Equal(t, DefaultBitbucketForgeJWKSURL, cfg.JWKSURL)

	t.Setenv(EnvBitbucketForgeAppID, "  ")
	assert.False(t, LoadBitbucketForgeAppConfig().Enabled())
}
