package networkpolicy

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveHTTPPolicyForSetting(t *testing.T) {
	t.Run("defaults block private networks", func(t *testing.T) {
		unsetEnvForTest(t, "BLOCKED_HTTP_HOSTS")
		unsetEnvForTest(t, "BLOCKED_PRIVATE_IP_RANGES")

		policy := ResolveHTTPPolicyForSetting(false)

		assert.False(t, policy.AllowPrivateNetworkAccess)
		assert.Equal(t, DefaultBlockedHTTPHosts, policy.BlockedHosts)
		assert.Equal(t, DefaultBlockedPrivateIPRanges, policy.PrivateIPRanges)
		assert.False(t, policy.BlockedHostsOverridden)
		assert.False(t, policy.PrivateIPRangesOverridden)
	})

	t.Run("installation setting can unblock private networks when env vars are unset", func(t *testing.T) {
		unsetEnvForTest(t, "BLOCKED_HTTP_HOSTS")
		unsetEnvForTest(t, "BLOCKED_PRIVATE_IP_RANGES")

		policy := ResolveHTTPPolicyForSetting(true)

		assert.True(t, policy.AllowPrivateNetworkAccess)
		assert.Empty(t, policy.BlockedHosts)
		assert.Empty(t, policy.PrivateIPRanges)
		assert.False(t, policy.BlockedHostsOverridden)
		assert.False(t, policy.PrivateIPRangesOverridden)
	})

	t.Run("empty env vars disable each block list explicitly", func(t *testing.T) {
		t.Setenv("BLOCKED_HTTP_HOSTS", "")
		t.Setenv("BLOCKED_PRIVATE_IP_RANGES", "")

		policy := ResolveHTTPPolicyForSetting(false)

		assert.Empty(t, policy.BlockedHosts)
		assert.Empty(t, policy.PrivateIPRanges)
		assert.True(t, policy.BlockedHostsOverridden)
		assert.True(t, policy.PrivateIPRangesOverridden)
	})

	t.Run("env overrides take precedence over installation setting", func(t *testing.T) {
		t.Setenv("BLOCKED_HTTP_HOSTS", "localhost, internal.example")
		t.Setenv("BLOCKED_PRIVATE_IP_RANGES", "10.0.0.0/8, 192.168.0.0/16")

		policy := ResolveHTTPPolicyForSetting(true)

		require.Equal(t, []string{"localhost", "internal.example"}, policy.BlockedHosts)
		require.Equal(t, []string{"10.0.0.0/8", "192.168.0.0/16"}, policy.PrivateIPRanges)
		assert.True(t, policy.BlockedHostsOverridden)
		assert.True(t, policy.PrivateIPRangesOverridden)
	})
}

func unsetEnvForTest(t *testing.T, key string) {
	t.Helper()

	previousValue, hadValue := os.LookupEnv(key)
	require.NoError(t, os.Unsetenv(key))

	t.Cleanup(func() {
		if !hadValue {
			require.NoError(t, os.Unsetenv(key))
			return
		}

		require.NoError(t, os.Setenv(key, previousValue))
	})
}
