package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

func Test__UseHostedApp(t *testing.T) {
	t.Setenv(common.EnvGitHubAppID, "")
	t.Setenv(common.EnvGitHubAppSlug, "")
	t.Setenv(common.EnvGitHubAppPrivateKey, "")
	t.Setenv(common.EnvGitHubAppWebhookSecret, "")

	t.Run("false when env is empty", func(t *testing.T) {
		restore := withFactoriesEnabledForTest(func(string) bool { return true })
		t.Cleanup(restore)
		assert.False(t, UseHostedApp("11111111-1111-1111-1111-111111111111"))
	})

	t.Run("false when factories is off", func(t *testing.T) {
		setHostedAppEnv(t)
		restore := withFactoriesEnabledForTest(func(string) bool { return false })
		t.Cleanup(restore)
		assert.False(t, UseHostedApp("11111111-1111-1111-1111-111111111111"))
	})

	t.Run("true when env is set and factories is on", func(t *testing.T) {
		setHostedAppEnv(t)
		restore := withFactoriesEnabledForTest(func(string) bool { return true })
		t.Cleanup(restore)
		assert.True(t, UseHostedApp("11111111-1111-1111-1111-111111111111"))
	})
}

func Test__UseHostedInstall(t *testing.T) {
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)

	orgID := "11111111-1111-1111-1111-111111111111"
	assert.True(t, UseHostedInstall(orgID, "github"))
	assert.False(t, UseHostedInstall(orgID, "slack"))
}

func Test__WantsPrivateApp(t *testing.T) {
	assert.False(t, WantsPrivateApp("github", nil))
	assert.False(t, WantsPrivateApp("github", map[string]any{}))
	assert.False(t, WantsPrivateApp("github", map[string]any{"privateApp": false}))
	assert.False(t, WantsPrivateApp("slack", map[string]any{"privateApp": true}))
	assert.True(t, WantsPrivateApp("github", map[string]any{"privateApp": true}))
}

func Test__PreferHostedInstall(t *testing.T) {
	setHostedAppEnv(t)
	restore := withFactoriesEnabledForTest(func(string) bool { return true })
	t.Cleanup(restore)

	orgID := "11111111-1111-1111-1111-111111111111"
	assert.True(t, PreferHostedInstall(orgID, "github", nil))
	assert.True(t, PreferHostedInstall(orgID, "github", map[string]any{}))
	assert.False(t, PreferHostedInstall(orgID, "github", map[string]any{"privateApp": true}))
	assert.False(t, PreferHostedInstall(orgID, "slack", nil))
}

func setHostedAppEnv(t *testing.T) {
	t.Helper()
	t.Setenv(common.EnvGitHubAppID, "99")
	t.Setenv(common.EnvGitHubAppSlug, "superplane")
	t.Setenv(common.EnvGitHubAppPrivateKey, "test-pem")
	t.Setenv(common.EnvGitHubAppWebhookSecret, "whsec")
	t.Setenv(common.EnvGitHubAppClientID, "")
	t.Setenv(common.EnvGitHubAppClientSecret, "")
}
