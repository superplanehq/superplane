package ssh

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/core"
)

type testMetadataContext struct {
	value any
}

func (m *testMetadataContext) Get() any {
	return m.value
}

func (m *testMetadataContext) Set(value any) error {
	m.value = value
	return nil
}

func authConfig(method string, privateKey, password any) map[string]any {
	m := map[string]any{"authMethod": method}
	if privateKey != nil {
		m["privateKey"] = privateKey
	}
	if password != nil {
		m["password"] = password
	}
	return m
}

func TestSSHCommand_Setup_ValidatesRequiredFields(t *testing.T) {
	c := &SSHCommand{}
	authWithKey := authConfig(AuthMethodSSHKey, map[string]any{"secret": "my-secret", "key": "private_key"}, nil)
	authWithPass := authConfig(AuthMethodPassword, nil, map[string]any{"secret": "my-secret", "key": "password"})

	t.Run("missing host", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"username":       "root",
				"authentication": authWithKey,
				"commands":       "ls",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "host")
	})

	t.Run("missing username", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"host":           "example.com",
				"authentication": authWithKey,
				"commands":       "ls",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "username")
	})

	t.Run("missing commands", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"host":           "example.com",
				"username":       "root",
				"authentication": authWithKey,
				"timeout":        60,
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "commands")
	})

	t.Run("whitespace only commands", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"host":           "example.com",
				"username":       "root",
				"authentication": authWithKey,
				"commands":       "  \n  ",
				"timeout":        60,
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "commands")
	})

	t.Run("ssh_key auth without secret ref", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"host":           "example.com",
				"username":       "root",
				"authentication": authConfig(AuthMethodSSHKey, nil, nil),
				"commands":       "ls",
				"timeout":        60,
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "private key")
	})

	t.Run("password auth without secret ref", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"host":           "example.com",
				"username":       "root",
				"authentication": authConfig(AuthMethodPassword, nil, nil),
				"commands":       "ls",
				"timeout":        60,
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "password")
	})

	t.Run("valid ssh_key config", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"host":           "example.com",
				"username":       "root",
				"authentication": authWithKey,
				"commands":       "ls -la",
				"timeout":        60,
			},
		})
		require.NoError(t, err)
	})

	t.Run("valid password config", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"host":           "example.com",
				"username":       "root",
				"authentication": authWithPass,
				"commands":       "whoami",
				"timeout":        60,
			},
		})
		require.NoError(t, err)
	})

	t.Run("invalid environment variable name", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"host":           "example.com",
				"username":       "root",
				"authentication": authWithPass,
				"commands":       "whoami",
				"timeout":        60,
				"environment": []map[string]any{
					{
						"name":  "BAD-NAME",
						"value": "x",
					},
				},
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid environment variable name")
	})
}

func TestSSHCommand_Execute_DoesNotPanicWithoutConnectionRetry(t *testing.T) {
	c := &SSHCommand{}
	metadata := &testMetadataContext{}

	err := c.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"host":     "example.com",
			"port":     22,
			"username": "root",
			"commands": "ls -la",
			"timeout":  60,
			"authentication": map[string]any{
				"authMethod": "invalid",
			},
		},
		Metadata: metadata,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported authentication method")

	saved, ok := metadata.Get().(ExecutionMetadata)
	require.True(t, ok)
	assert.Nil(t, saved.ConnectionRetry)
	assert.Equal(t, 0, saved.MaxRetries)
	assert.Equal(t, 0, saved.IntervalSeconds)
}

func TestSSHCommand_BuildRemoteCommand(t *testing.T) {
	c := &SSHCommand{}

	t.Run("command only", func(t *testing.T) {
		command := c.buildRemoteCommand("", nil, "echo hello")
		assert.Equal(t, "echo hello", command)
	})

	t.Run("working directory is shell quoted", func(t *testing.T) {
		command := c.buildRemoteCommand("/opt/app's hardening", nil, "bash run.sh")
		assert.Equal(t, "cd '/opt/app'\"'\"'s hardening' && bash run.sh", command)
	})

	t.Run("environment values are shell quoted and command wrapped", func(t *testing.T) {
		command := c.buildRemoteCommand(
			"",
			[]EnvironmentVariable{
				{Name: "PLAIN", Value: "ok"},
				{Name: "SPECIAL", Value: "a'b;$PATH $(whoami)"},
			},
			"echo \"$SPECIAL\"",
		)
		assert.Equal(
			t,
			"env PLAIN='ok' SPECIAL='a'\"'\"'b;$PATH $(whoami)' sh -lc 'echo \"$SPECIAL\"'",
			command,
		)
	})
}

func TestBuildCombinedCommands(t *testing.T) {
	t.Run("joins non-empty lines with &&", func(t *testing.T) {
		combined := buildCombinedCommands("echo 1\n\n  echo 2  \n")
		assert.Equal(t, "echo 1 && echo 2", combined)
	})

	t.Run("empty input yields empty string", func(t *testing.T) {
		combined := buildCombinedCommands("\n \n")
		assert.Equal(t, "", combined)
	})
}
