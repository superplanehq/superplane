package sentry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/config"
)

func Test__UseHostedInstall(t *testing.T) {
	t.Setenv(config.EnvSentryAppClientID, "client-id")
	t.Setenv(config.EnvSentryAppClientSecret, "client-secret")
	t.Setenv(config.EnvSentryAppSlug, "superplane")
	defer withSentryIntakeEnabledForTest(func(string) bool { return true })()

	assert.True(t, UseHostedInstall("org-1", "sentry"))
	assert.False(t, UseHostedInstall("org-1", "github"))
	assert.True(t, PreferHostedInstall("org-1", "sentry", nil))
}

func Test__UseHostedInstall_requiresFlagAndEnv(t *testing.T) {
	t.Run("missing env", func(t *testing.T) {
		t.Setenv(config.EnvSentryAppClientID, "")
		t.Setenv(config.EnvSentryAppClientSecret, "")
		t.Setenv(config.EnvSentryAppSlug, "")
		defer withSentryIntakeEnabledForTest(func(string) bool { return true })()

		assert.False(t, UseHostedInstall("org-1", "sentry"))
	})

	t.Run("flag off", func(t *testing.T) {
		t.Setenv(config.EnvSentryAppClientID, "client-id")
		t.Setenv(config.EnvSentryAppClientSecret, "client-secret")
		t.Setenv(config.EnvSentryAppSlug, "superplane")
		defer withSentryIntakeEnabledForTest(func(string) bool { return false })()

		assert.False(t, UseHostedInstall("org-1", "sentry"))
	})
}

func Test__ExternalInstallURL(t *testing.T) {
	assert.Equal(
		t,
		"https://sentry.io/sentry-apps/superplane/external-install/",
		ExternalInstallURL("https://sentry.io", "superplane"),
	)
}

func Test__MetadataAllowsStartedBy(t *testing.T) {
	assert.True(t, Metadata{}.AllowsStartedBy("user-1"))
	assert.True(t, Metadata{StartedByUserID: "user-1"}.AllowsStartedBy("user-1"))
	assert.False(t, Metadata{StartedByUserID: "user-1"}.AllowsStartedBy("user-2"))
}
