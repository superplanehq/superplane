package ssh

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestSSHCommand_Setup_ValidatesRequiredFields(t *testing.T) {
	c := &SSHCommand{}

	t.Run("missing host", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"username":   "root",
				"authMethod": AuthMethodSSHKey,
				"privateKeySecretRef": "my-secret",
				"privateKeyKeyName":   "private_key",
				"command":    "ls",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "host")
	})

	t.Run("missing username", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"host":       "example.com",
				"authMethod": AuthMethodSSHKey,
				"privateKeySecretRef": "my-secret",
				"privateKeyKeyName":   "private_key",
				"command":    "ls",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "username")
	})

	t.Run("missing command", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"host":       "example.com",
				"username":   "root",
				"authMethod": AuthMethodSSHKey,
				"privateKeySecretRef": "my-secret",
				"privateKeyKeyName":   "private_key",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "command")
	})

	t.Run("ssh_key auth without secret ref", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"host":       "example.com",
				"username":   "root",
				"authMethod": AuthMethodSSHKey,
				"command":    "ls",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "private key")
	})

	t.Run("password auth without secret ref", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"host":       "example.com",
				"username":   "root",
				"authMethod": AuthMethodPassword,
				"command":    "ls",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "password")
	})

	t.Run("valid ssh_key config", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"host":                "example.com",
				"username":            "root",
				"authMethod":          AuthMethodSSHKey,
				"privateKeySecretRef": "my-secret",
				"privateKeyKeyName":   "private_key",
				"command":             "ls -la",
			},
		})
		require.NoError(t, err)
	})

	t.Run("valid password config", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"host":               "example.com",
				"username":           "root",
				"authMethod":         AuthMethodPassword,
				"passwordSecretRef":  "my-secret",
				"passwordKeyName":    "password",
				"command":            "whoami",
			},
		})
		require.NoError(t, err)
	})
}

func TestSSHCommand_Execute_RequiresSecretsContext(t *testing.T) {
	c := &SSHCommand{}
	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"host":                "example.com",
			"username":            "root",
			"authMethod":          AuthMethodSSHKey,
			"privateKeySecretRef": "secret",
			"privateKeyKeyName":   "key",
			"command":             "ls",
		},
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		Secrets:        nil, // not set
	}

	_ = c.Execute(ctx)
	assert.True(t, stateCtx.Finished)
	assert.False(t, stateCtx.Passed)
	assert.Contains(t, stateCtx.FailureMessage, "secrets")
}
