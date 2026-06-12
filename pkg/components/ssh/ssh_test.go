package ssh

import (
	"errors"
	"fmt"
	"io"
	"strings"
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

// fakeFilesContext is an in-memory core.RepositoryFilesContext used to drive
// file-mode SSH tests without spinning up a git provider.
type fakeFilesContext struct {
	files    map[string]string
	listErr  error
	readErr  error
	readPath string
}

func (f *fakeFilesContext) List() ([]string, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	paths := make([]string, 0, len(f.files))
	for path := range f.files {
		paths = append(paths, path)
	}
	return paths, nil
}

func (f *fakeFilesContext) Read(path string) (io.ReadCloser, error) {
	f.readPath = path
	if f.readErr != nil {
		return nil, f.readErr
	}
	content, ok := f.files[path]
	if !ok {
		return nil, fmt.Errorf("file %q not found", path)
	}
	return io.NopCloser(strings.NewReader(content)), nil
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

	t.Run("execution retry invalid retries", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"host":           "example.com",
				"username":       "root",
				"authentication": authWithKey,
				"commands":       "ls",
				"timeout":        60,
				"executionRetry": map[string]any{
					"enabled":         true,
					"retries":         -1,
					"intervalSeconds": 15,
				},
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "execution retry")
	})

	t.Run("execution retry invalid interval", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"host":           "example.com",
				"username":       "root",
				"authentication": authWithKey,
				"commands":       "ls",
				"timeout":        60,
				"executionRetry": map[string]any{
					"enabled":         true,
					"retries":         3,
					"intervalSeconds": 0,
				},
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "execution retry")
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

func TestSSHCommand_Setup_FileMode(t *testing.T) {
	c := &SSHCommand{}
	authWithKey := authConfig(AuthMethodSSHKey, map[string]any{"secret": "my-secret", "key": "private_key"}, nil)
	baseConfig := func(extra map[string]any) map[string]any {
		cfg := map[string]any{
			"host":           "example.com",
			"username":       "root",
			"authentication": authWithKey,
			"commandSource":  CommandSourceFile,
			"timeout":        60,
		}
		for key, value := range extra {
			cfg[key] = value
		}
		return cfg
	}

	t.Run("missing command file", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: baseConfig(nil),
			Files:         &fakeFilesContext{files: map[string]string{"scripts/deploy.sh": "echo hi"}},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "command file is required")
	})

	t.Run("invalid command file path", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: baseConfig(map[string]any{"commandFile": "../escape.sh"}),
			Files:         &fakeFilesContext{files: map[string]string{}},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid command file")
	})

	t.Run("no files context available", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: baseConfig(map[string]any{"commandFile": "scripts/deploy.sh"}),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file access is not available")
	})

	t.Run("file not in repository", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: baseConfig(map[string]any{"commandFile": "scripts/missing.sh"}),
			Files:         &fakeFilesContext{files: map[string]string{"scripts/deploy.sh": "echo hi"}},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found in app repository")
	})

	t.Run("list error surfaced", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: baseConfig(map[string]any{"commandFile": "scripts/deploy.sh"}),
			Files:         &fakeFilesContext{listErr: errors.New("boom")},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list repository files")
	})

	t.Run("valid file with leading slash is normalized", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: baseConfig(map[string]any{"commandFile": "/scripts/deploy.sh"}),
			Files:         &fakeFilesContext{files: map[string]string{"scripts/deploy.sh": "echo hi"}},
		})
		require.NoError(t, err)
	})

	t.Run("valid file in file mode passes", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: baseConfig(map[string]any{"commandFile": "scripts/deploy.sh"}),
			Files:         &fakeFilesContext{files: map[string]string{"scripts/deploy.sh": "echo hi"}},
		})
		require.NoError(t, err)
	})

	t.Run("whitespace-only file content is rejected at setup", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: baseConfig(map[string]any{"commandFile": "scripts/deploy.sh"}),
			Files:         &fakeFilesContext{files: map[string]string{"scripts/deploy.sh": "  \n\t"}},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("file mode ignores stale inline commands field", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: baseConfig(map[string]any{
				"commandFile": "scripts/deploy.sh",
				"commands":    "",
			}),
			Files: &fakeFilesContext{files: map[string]string{"scripts/deploy.sh": "echo hi"}},
		})
		require.NoError(t, err)
	})
}

func TestSSHCommand_Execute_FileMode(t *testing.T) {
	c := &SSHCommand{}

	baseExecConfig := func(extra map[string]any) map[string]any {
		cfg := map[string]any{
			"host":          "example.com",
			"port":          22,
			"username":      "root",
			"timeout":       60,
			"commandSource": CommandSourceFile,
			"commandFile":   "scripts/deploy.sh",
			"authentication": map[string]any{
				"authMethod": "invalid",
			},
		}
		for key, value := range extra {
			cfg[key] = value
		}
		return cfg
	}

	t.Run("reads file content verbatim into execution metadata", func(t *testing.T) {
		fileContent := "echo hello\nls -la"
		files := &fakeFilesContext{
			files: map[string]string{"scripts/deploy.sh": fileContent},
		}
		metadata := &testMetadataContext{}

		err := c.Execute(core.ExecutionContext{
			Configuration: baseExecConfig(nil),
			Metadata:      metadata,
			Files:         files,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported authentication method")
		assert.Equal(t, "scripts/deploy.sh", files.readPath)

		saved, ok := metadata.Get().(ExecutionMetadata)
		require.True(t, ok)
		assert.Equal(t, CommandSourceFile, saved.CommandSource)
		assert.Equal(t, "scripts/deploy.sh", saved.CommandFile)
		assert.Equal(t, fileContent, saved.Commands)
	})

	// Shell scripts frequently embed their own {{ ... }} syntax (Docker
	// inspect format strings, Helm/Go templates, kubectl jsonpath). The file
	// content must be passed through to the remote host untouched so these
	// scripts run the same way they do when pasted into an SSH session.
	t.Run("preserves embedded {{ }} syntax in file content", func(t *testing.T) {
		fileContent := `docker inspect --format '{{ .Config.Image }}' my-container
helm template chart --set image.tag={{ .Values.tag }}`
		metadata := &testMetadataContext{}

		err := c.Execute(core.ExecutionContext{
			Configuration: baseExecConfig(nil),
			Metadata:      metadata,
			Files:         &fakeFilesContext{files: map[string]string{"scripts/deploy.sh": fileContent}},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported authentication method")

		saved, ok := metadata.Get().(ExecutionMetadata)
		require.True(t, ok)
		assert.Equal(t, fileContent, saved.Commands)
	})

	t.Run("missing file context is reported", func(t *testing.T) {
		metadata := &testMetadataContext{}
		err := c.Execute(core.ExecutionContext{
			Configuration: baseExecConfig(nil),
			Metadata:      metadata,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file access is not available")
	})

	t.Run("missing command file path is reported", func(t *testing.T) {
		metadata := &testMetadataContext{}
		err := c.Execute(core.ExecutionContext{
			Configuration: baseExecConfig(map[string]any{"commandFile": "  "}),
			Metadata:      metadata,
			Files:         &fakeFilesContext{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "command file is required")
	})

	t.Run("empty file content is rejected", func(t *testing.T) {
		metadata := &testMetadataContext{}
		err := c.Execute(core.ExecutionContext{
			Configuration: baseExecConfig(nil),
			Metadata:      metadata,
			Files:         &fakeFilesContext{files: map[string]string{"scripts/deploy.sh": "   \n  "}},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("read error is reported", func(t *testing.T) {
		metadata := &testMetadataContext{}
		err := c.Execute(core.ExecutionContext{
			Configuration: baseExecConfig(nil),
			Metadata:      metadata,
			Files:         &fakeFilesContext{readErr: errors.New("disk gone")},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "read command file")
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
	assert.Nil(t, saved.ExecutionRetry)
	assert.Equal(t, 0, saved.MaxRetries)
	assert.Equal(t, 0, saved.IntervalSeconds)
	assert.Equal(t, 0, saved.ExecutionAttempt)
}

func TestSSHCommand_MetadataHelpersSupportStructMetadata(t *testing.T) {
	c := &SSHCommand{}
	metadata := &testMetadataContext{
		value: ExecutionMetadata{
			Host:     "example.com",
			User:     "root",
			Commands: "exit 1",
			Attempt:  0,
		},
	}

	err := c.incrementRetryCount(metadata)
	require.NoError(t, err)

	current, ok := metadata.Get().(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "example.com", current["host"])
	assert.Equal(t, "root", current["user"])
	assert.Equal(t, "exit 1", current["commands"])
	assert.Equal(t, 1, c.getRetryAttempt(metadata))

	err = c.incrementExecutionRetryCount(metadata)
	require.NoError(t, err)
	assert.Equal(t, 1, c.getExecutionAttempt(metadata))

	err = c.setResultMetadata(metadata, &CommandResult{
		Stdout:   "",
		Stderr:   "command failed",
		ExitCode: 1,
	})
	require.NoError(t, err)

	current, ok = metadata.Get().(map[string]any)
	require.True(t, ok)

	result, ok := current["result"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 1, result["exitCode"])
	assert.Equal(t, "command failed", result["stderr"])
}

func TestSSHCommand_ShouldRetry(t *testing.T) {
	c := &SSHCommand{}
	enabled := &RetrySpec{Enabled: true, Retries: 3}

	assert.False(t, c.shouldRetry(nil, 0))
	assert.False(t, c.shouldRetry(&RetrySpec{Enabled: false, Retries: 3}, 0))
	assert.True(t, c.shouldRetry(enabled, 0))
	assert.True(t, c.shouldRetry(enabled, 2))
	assert.False(t, c.shouldRetry(enabled, 3))
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

func TestSSHCommand_BuildScriptCommand(t *testing.T) {
	c := &SSHCommand{}

	t.Run("wraps script in bash -lc preserving newlines and comments", func(t *testing.T) {
		script := "#!/usr/bin/env bash\n# comment\nset -eo pipefail\necho hi"
		command := c.buildScriptCommand("", nil, script)
		assert.Equal(t, "bash -lc '#!/usr/bin/env bash\n# comment\nset -eo pipefail\necho hi'", command)
	})

	t.Run("working directory uses newline + exit guard so leading comments do not eat cd", func(t *testing.T) {
		script := "#!/usr/bin/env bash\necho hi"
		command := c.buildScriptCommand("/opt/app", nil, script)
		assert.Equal(t, "bash -lc 'cd '\"'\"'/opt/app'\"'\"' || exit 1\n#!/usr/bin/env bash\necho hi'", command)
	})

	t.Run("environment values are exported via env before bash", func(t *testing.T) {
		script := "echo \"$NAME\""
		command := c.buildScriptCommand(
			"",
			[]EnvironmentVariable{
				{Name: "NAME", Value: "world"},
				{Name: "TOKEN", Value: "a'b"},
			},
			script,
		)
		assert.Equal(t, "env NAME='world' TOKEN='a'\"'\"'b' bash -lc 'echo \"$NAME\"'", command)
	})

	t.Run("script with single quotes is properly escaped", func(t *testing.T) {
		script := "echo 'hello world'"
		command := c.buildScriptCommand("", nil, script)
		assert.Equal(t, "bash -lc 'echo '\"'\"'hello world'\"'\"''", command)
	})

	// Regression: a bash script that starts with a shebang and contains
	// multi-line constructs must NOT be collapsed to a single &&-joined
	// line, because the leading `#!` would turn the rest of the script
	// into a shell comment and nothing would execute.
	t.Run("preserves embedded {{ }} and multi-line script structure", func(t *testing.T) {
		script := `#!/usr/bin/env bash
set -eo pipefail
image=$(docker inspect --format '{{ .Config.Image }}' my-container)
if [ -n "$image" ]; then
  echo "$image"
fi`
		command := c.buildScriptCommand("", nil, script)
		expected := "bash -lc '" + strings.ReplaceAll(script, "'", `'"'"'`) + "'"
		assert.Equal(t, expected, command)
		assert.Contains(t, command, "{{ .Config.Image }}")
		assert.Contains(t, command, "\nset -eo pipefail\n")
	})
}

func TestSSHCommand_BuildExecutionCommand(t *testing.T) {
	c := &SSHCommand{}

	t.Run("inline mode joins lines and uses sh", func(t *testing.T) {
		meta := ExecutionMetadata{
			CommandSource: CommandSourceInline,
			Commands:      "echo 1\necho 2",
		}
		command, err := c.buildExecutionCommand(meta)
		require.NoError(t, err)
		assert.Equal(t, "echo 1 && echo 2", command)
	})

	t.Run("inline mode rejects empty commands", func(t *testing.T) {
		meta := ExecutionMetadata{
			CommandSource: CommandSourceInline,
			Commands:      "  \n  ",
		}
		_, err := c.buildExecutionCommand(meta)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "commands is required")
	})

	t.Run("file mode wraps multi-line script in bash -lc without &&-joining", func(t *testing.T) {
		meta := ExecutionMetadata{
			CommandSource: CommandSourceFile,
			Commands:      "#!/usr/bin/env bash\n# header\necho hi",
		}
		command, err := c.buildExecutionCommand(meta)
		require.NoError(t, err)
		assert.Equal(t, "bash -lc '#!/usr/bin/env bash\n# header\necho hi'", command)
		assert.NotContains(t, command, " && ")
	})

	t.Run("legacy inline metadata (empty CommandSource) keeps old behavior", func(t *testing.T) {
		meta := ExecutionMetadata{
			Commands: "echo legacy",
		}
		command, err := c.buildExecutionCommand(meta)
		require.NoError(t, err)
		assert.Equal(t, "echo legacy", command)
	})
}

func TestSSHCommand_CommandSourceOrDefault(t *testing.T) {
	t.Run("empty defaults to inline", func(t *testing.T) {
		assert.Equal(t, CommandSourceInline, Spec{CommandSource: ""}.commandSourceOrDefault())
	})

	t.Run("whitespace-only defaults to inline", func(t *testing.T) {
		assert.Equal(t, CommandSourceInline, Spec{CommandSource: "  \t "}.commandSourceOrDefault())
	})

	// Regression: surrounding whitespace must be trimmed so the returned
	// value matches the inline/file switch cases instead of being reported
	// as an invalid command source.
	t.Run("trims surrounding whitespace from recognized values", func(t *testing.T) {
		assert.Equal(t, CommandSourceFile, Spec{CommandSource: "file "}.commandSourceOrDefault())
		assert.Equal(t, CommandSourceInline, Spec{CommandSource: " inline"}.commandSourceOrDefault())
		assert.Equal(t, CommandSourceFile, Spec{CommandSource: "\tfile\n"}.commandSourceOrDefault())
	})
}
