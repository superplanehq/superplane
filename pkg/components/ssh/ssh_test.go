package ssh

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// Legacy SSH nodes were saved before the commandSource field existed, so their
// stored configuration omits it entirely. ValidateConfiguration does not apply
// Field.Default, so commandSource must stay optional at the schema level or
// re-validating/patching such a node would fail even though inline commands are
// valid (the worker defaults the source to inline via commandSourceOrDefault).
func TestSSHCommand_ValidateConfiguration_LegacyConfigWithoutCommandSource(t *testing.T) {
	c := &SSHCommand{}
	fields := c.Configuration()

	legacyConfig := map[string]any{
		"host":     "example.com",
		"port":     22,
		"username": "root",
		"authentication": map[string]any{
			"authMethod": AuthMethodPassword,
			"password":   map[string]any{"secret": "my-secret", "key": "password"},
		},
		"commands": "echo hi\nls -la",
		"timeout":  60,
	}

	require.NoError(t, configuration.ValidateConfiguration(fields, legacyConfig))
}

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

	t.Run("stores only the file path in metadata", func(t *testing.T) {
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
		// Execute reads the file once at start to validate it is reachable
		// and non-empty (matches Setup's publish-time guard).
		assert.Equal(t, "scripts/deploy.sh", files.readPath)

		saved, ok := metadata.Get().(ExecutionMetadata)
		require.True(t, ok)
		assert.Equal(t, CommandSourceFile, saved.CommandSource)
		assert.Equal(t, "scripts/deploy.sh", saved.CommandFile)
		// File contents must NOT be persisted in metadata — the worker
		// re-reads the file from the canvas repository on every attempt
		// (initial + retries) so the script content never lives in the
		// database.
		assert.Empty(t, saved.Commands)
		assert.NotContains(t, fmt.Sprintf("%v", metadata.Get()), fileContent)
	})

	t.Run("inline mode still stores commands in metadata", func(t *testing.T) {
		metadata := &testMetadataContext{}
		err := c.Execute(core.ExecutionContext{
			Configuration: baseExecConfig(map[string]any{
				"commandSource": CommandSourceInline,
				"commandFile":   "",
				"commands":      "echo hi",
			}),
			Metadata: metadata,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported authentication method")

		saved, ok := metadata.Get().(ExecutionMetadata)
		require.True(t, ok)
		assert.Equal(t, CommandSourceInline, saved.CommandSource)
		assert.Equal(t, "echo hi", saved.Commands)
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

func TestLoadCommandFile_NormalizesLineEndings(t *testing.T) {
	t.Run("normalizes Windows CRLF line endings", func(t *testing.T) {
		files := &fakeFilesContext{
			files: map[string]string{"scripts/deploy.sh": "set -eo pipefail\r\nprintf ok\r\n"},
		}

		body, err := loadCommandFile(files, "scripts/deploy.sh")
		require.NoError(t, err)
		assert.Equal(t, "set -eo pipefail\nprintf ok\n", body)
		assert.NotContains(t, body, "\r")
	})

	t.Run("normalizes lone carriage returns", func(t *testing.T) {
		files := &fakeFilesContext{
			files: map[string]string{"scripts/deploy.sh": "set -eo pipefail\rprintf ok\r"},
		}

		body, err := loadCommandFile(files, "scripts/deploy.sh")
		require.NoError(t, err)
		assert.Equal(t, "set -eo pipefail\nprintf ok\n", body)
		assert.NotContains(t, body, "\r")
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

	t.Run("normalizes CRLF input before joining lines", func(t *testing.T) {
		combined := buildCombinedCommands("set -eo pipefail\r\necho ok\r\n")
		assert.Equal(t, "set -eo pipefail && echo ok", combined)
		assert.NotContains(t, combined, "\r")
	})

	t.Run("normalizes lone carriage returns before joining lines", func(t *testing.T) {
		combined := buildCombinedCommands("set -eo pipefail\recho ok\r")
		assert.Equal(t, "set -eo pipefail && echo ok", combined)
		assert.NotContains(t, combined, "\r")
	})

	t.Run("empty input yields empty string", func(t *testing.T) {
		combined := buildCombinedCommands("\n \n")
		assert.Equal(t, "", combined)
	})
}

func TestSSHCommand_BuildScriptCommand(t *testing.T) {
	c := &SSHCommand{}

	t.Run("returns plain `bash -s` and the script as the stdin payload", func(t *testing.T) {
		script := "#!/usr/bin/env bash\n# comment\nset -eo pipefail\necho hi"
		command, payload := c.buildScriptCommand("", nil, script)
		assert.Equal(t, "bash -s", command)
		assert.Equal(t, script, payload)
	})

	t.Run("working directory is prepended on its own line so a leading comment cannot eat cd", func(t *testing.T) {
		script := "#!/usr/bin/env bash\necho hi"
		command, payload := c.buildScriptCommand("/opt/app", nil, script)
		assert.Equal(t, "bash -s", command)
		assert.Equal(t, "cd '/opt/app' || exit 1\n#!/usr/bin/env bash\necho hi", payload)
	})

	t.Run("environment values are exported via env before bash and the script is unchanged", func(t *testing.T) {
		script := "echo \"$NAME\""
		command, payload := c.buildScriptCommand(
			"",
			[]EnvironmentVariable{
				{Name: "NAME", Value: "world"},
				{Name: "TOKEN", Value: "a'b"},
			},
			script,
		)
		assert.Equal(t, "env NAME='world' TOKEN='a'\"'\"'b' bash -s", command)
		assert.Equal(t, script, payload)
	})

	t.Run("script with single quotes is streamed verbatim (no shell escaping needed)", func(t *testing.T) {
		script := "echo 'hello world'"
		command, payload := c.buildScriptCommand("", nil, script)
		assert.Equal(t, "bash -s", command)
		assert.Equal(t, script, payload)
	})

	t.Run("normalizes CRLF scripts before streaming", func(t *testing.T) {
		command, payload := c.buildScriptCommand("", nil, "set -eo pipefail\r\necho ok\r\n")
		assert.Equal(t, "bash -s", command)
		assert.Equal(t, "set -eo pipefail\necho ok\n", payload)
		assert.NotContains(t, payload, "\r")
	})

	// Regression: a bash script with embedded `{{ ... }}` template syntax
	// (Docker inspect, Helm, kubectl jsonpath) must reach the remote host
	// untouched so it runs the same way it would when pasted into an
	// interactive SSH session.
	t.Run("preserves embedded {{ }} and multi-line script structure", func(t *testing.T) {
		script := `#!/usr/bin/env bash
set -eo pipefail
image=$(docker inspect --format '{{ .Config.Image }}' my-container)
if [ -n "$image" ]; then
  echo "$image"
fi`
		command, payload := c.buildScriptCommand("", nil, script)
		assert.Equal(t, "bash -s", command)
		assert.Equal(t, script, payload)
		assert.Contains(t, payload, "{{ .Config.Image }}")
		assert.Contains(t, payload, "\nset -eo pipefail\n")
	})
}

func TestSSHCommand_BuildExecutionCommand(t *testing.T) {
	c := &SSHCommand{}

	t.Run("inline mode streams multi-line commands", func(t *testing.T) {
		meta := ExecutionMetadata{
			CommandSource: CommandSourceInline,
			Commands:      "echo 1\necho 2",
		}
		command, stdin, err := c.buildExecutionCommand(meta, "")
		require.NoError(t, err)
		assert.Equal(t, "bash -s", command)
		require.NotNil(t, stdin)

		body, readErr := io.ReadAll(stdin)
		require.NoError(t, readErr)
		assert.Equal(t, "echo 1\necho 2", string(body))
	})

	t.Run("inline mode keeps single-line commands on the command line", func(t *testing.T) {
		meta := ExecutionMetadata{
			CommandSource: CommandSourceInline,
			Commands:      "echo 1 && echo 2",
		}

		command, stdin, err := c.buildExecutionCommand(meta, "")
		require.NoError(t, err)
		assert.Equal(t, "echo 1 && echo 2", command)
		assert.Nil(t, stdin)
	})

	t.Run("inline mode streams customer bash scripts instead of flattening them", func(t *testing.T) {
		script := strings.Join([]string{
			"#!/bin/bash",
			"set -euo pipefail",
			"cd ~/preview-sso.teams.novp.com",
			`sed -i -E "s#([a-z0-9\-]+\.teams\.novp\.com)#${APP_HOST_PREFIX}-\1#g" .env`,
			"docker compose down --remove-orphans && docker compose up -d",
			"docker compose run --rm php php artisan migrate",
		}, "\r\n")
		meta := ExecutionMetadata{
			CommandSource: CommandSourceInline,
			Commands:      script,
		}

		command, stdin, err := c.buildExecutionCommand(meta, "")
		require.NoError(t, err)
		assert.Equal(t, "bash -s", command)
		require.NotNil(t, stdin)

		body, readErr := io.ReadAll(stdin)
		require.NoError(t, readErr)
		assert.Equal(t, strings.ReplaceAll(script, "\r\n", "\n"), string(body))
		assert.NotContains(t, string(body), "\r")
		assert.Contains(t, string(body), `\1`)
	})

	t.Run("inline mode streams shell blocks that require real newlines", func(t *testing.T) {
		script := strings.Join([]string{
			"if [ -n \"$APP_BRANCH_NAME\" ]; then",
			"  git switch \"$APP_BRANCH_NAME\"",
			"fi",
		}, "\n")
		meta := ExecutionMetadata{
			CommandSource: CommandSourceInline,
			Commands:      script,
		}

		command, stdin, err := c.buildExecutionCommand(meta, "")
		require.NoError(t, err)
		assert.Equal(t, "bash -s", command)
		require.NotNil(t, stdin)

		body, readErr := io.ReadAll(stdin)
		require.NoError(t, readErr)
		assert.Equal(t, script, string(body))
	})

	t.Run("inline mode rejects empty commands", func(t *testing.T) {
		meta := ExecutionMetadata{
			CommandSource: CommandSourceInline,
			Commands:      "  \n  ",
		}
		_, _, err := c.buildExecutionCommand(meta, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "commands is required")
	})

	t.Run("file mode streams the pre-loaded script as stdin", func(t *testing.T) {
		script := "#!/usr/bin/env bash\n# header\necho hi"
		meta := ExecutionMetadata{
			CommandSource: CommandSourceFile,
			CommandFile:   "scripts/deploy.sh",
		}

		command, stdin, err := c.buildExecutionCommand(meta, script)
		require.NoError(t, err)
		assert.Equal(t, "bash -s", command)
		require.NotNil(t, stdin)
		assert.NotContains(t, command, " && ")

		body, readErr := io.ReadAll(stdin)
		require.NoError(t, readErr)
		assert.Equal(t, script, string(body))
	})

	t.Run("file mode normalizes CRLF before streaming stdin", func(t *testing.T) {
		script := "set -eo pipefail\r\necho hi\r\n"
		meta := ExecutionMetadata{
			CommandSource: CommandSourceFile,
			CommandFile:   "scripts/deploy.sh",
		}

		command, stdin, err := c.buildExecutionCommand(meta, script)
		require.NoError(t, err)
		assert.Equal(t, "bash -s", command)
		require.NotNil(t, stdin)

		body, readErr := io.ReadAll(stdin)
		require.NoError(t, readErr)
		assert.Equal(t, "set -eo pipefail\necho hi\n", string(body))
		assert.NotContains(t, string(body), "\r")
	})

	t.Run("file mode prepends working directory to the streamed script", func(t *testing.T) {
		script := "echo hi"
		meta := ExecutionMetadata{
			CommandSource:    CommandSourceFile,
			CommandFile:      "scripts/deploy.sh",
			WorkingDirectory: "/opt/app",
			Environment: []EnvironmentVariable{
				{Name: "NAME", Value: "world"},
			},
		}

		command, stdin, err := c.buildExecutionCommand(meta, script)
		require.NoError(t, err)
		assert.Equal(t, "env NAME='world' bash -s", command)

		body, readErr := io.ReadAll(stdin)
		require.NoError(t, readErr)
		assert.Equal(t, "cd '/opt/app' || exit 1\necho hi", string(body))
	})

	t.Run("file mode rejects an empty script body", func(t *testing.T) {
		meta := ExecutionMetadata{
			CommandSource: CommandSourceFile,
			CommandFile:   "scripts/deploy.sh",
		}
		_, _, err := c.buildExecutionCommand(meta, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "command file body is required")
	})

	t.Run("legacy inline metadata (empty CommandSource) keeps old behavior", func(t *testing.T) {
		meta := ExecutionMetadata{
			Commands: "echo legacy",
		}
		command, stdin, err := c.buildExecutionCommand(meta, "")
		require.NoError(t, err)
		assert.Equal(t, "echo legacy", command)
		assert.Nil(t, stdin)
	})
}

// Regression for the metadata-leak fix: the retry hook must re-read the
// command file from the canvas repository on every retry, surface file
// errors before any SSH work, and never fall back to script content from
// stored metadata.
func TestSSHCommand_HandleHook_FileModeRetryRereadsFile(t *testing.T) {
	c := &SSHCommand{}
	files := &fakeFilesContext{
		files: map[string]string{"scripts/deploy.sh": "echo hi"},
	}
	metadata := &testMetadataContext{
		value: ExecutionMetadata{
			Host:          "example.com",
			Port:          22,
			User:          "root",
			Timeout:       60,
			CommandSource: CommandSourceFile,
			CommandFile:   "scripts/deploy.sh",
			Authentication: AuthSpec{
				Method: "invalid",
			},
		},
	}

	err := c.HandleHook(core.ActionHookContext{
		Name:           "executionRetry",
		Metadata:       metadata,
		Files:          files,
		ExecutionState: &fakeExecutionState{},
	})

	require.Error(t, err)
	// We get past the file load (which proves the retry re-read it) and
	// only fail on the auth setup, which is the next step.
	assert.Contains(t, err.Error(), "unsupported authentication method")
	assert.Equal(t, "scripts/deploy.sh", files.readPath)
}

func TestSSHCommand_HandleHook_FileModeRetryReportsMissingFile(t *testing.T) {
	c := &SSHCommand{}
	files := &fakeFilesContext{readErr: errors.New("file gone")}
	metadata := &testMetadataContext{
		value: ExecutionMetadata{
			CommandSource: CommandSourceFile,
			CommandFile:   "scripts/deploy.sh",
		},
	}

	err := c.HandleHook(core.ActionHookContext{
		Name:           "executionRetry",
		Metadata:       metadata,
		Files:          files,
		ExecutionState: &fakeExecutionState{},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "read command file")
}

type fakeExecutionState struct{ finished bool }

func (f *fakeExecutionState) IsFinished() bool                                       { return f.finished }
func (f *fakeExecutionState) SetKV(key, value string) error                          { return nil }
func (f *fakeExecutionState) GetKV(key string) (string, error)                       { return "", nil }
func (f *fakeExecutionState) Emit(channel, payloadType string, payloads []any) error { return nil }
func (f *fakeExecutionState) EmitAndContinue(channel, payloadType string, payloads []any) error {
	return nil
}
func (f *fakeExecutionState) Pass() error                       { return nil }
func (f *fakeExecutionState) Fail(reason, message string) error { return nil }

func TestSSHCommand_CommandSourceOrDefault(t *testing.T) {
	t.Run("empty defaults to inline", func(t *testing.T) {
		assert.Equal(t, CommandSourceInline, Spec{CommandSource: ""}.commandSourceOrDefault())
	})

	t.Run("whitespace-only defaults to inline", func(t *testing.T) {
		assert.Equal(t, CommandSourceInline, Spec{CommandSource: "  \t "}.commandSourceOrDefault())
	})

	t.Run("recognized values are returned unchanged", func(t *testing.T) {
		assert.Equal(t, CommandSourceFile, Spec{CommandSource: "file"}.commandSourceOrDefault())
		assert.Equal(t, CommandSourceInline, Spec{CommandSource: "inline"}.commandSourceOrDefault())
	})

	// Regression: a non-empty padded value must be returned verbatim (NOT
	// trimmed to a known source). The UI evaluates the commandFile/commands
	// visibility and required conditions with an exact string comparison, so
	// trimming "\tfile\n" to "file" here would run file mode on the worker
	// while the UI had hidden and dropped commandFile from the saved payload.
	// Returning it verbatim makes validateCommandSource/resolveCommands reject
	// it loudly as an invalid command source instead.
	t.Run("non-empty padded values are returned verbatim to match the UI's exact matching", func(t *testing.T) {
		assert.Equal(t, "file ", Spec{CommandSource: "file "}.commandSourceOrDefault())
		assert.Equal(t, " inline", Spec{CommandSource: " inline"}.commandSourceOrDefault())
		assert.Equal(t, "\tfile\n", Spec{CommandSource: "\tfile\n"}.commandSourceOrDefault())
	})
}

func TestSSHCommand_Setup_RejectsPaddedCommandSource(t *testing.T) {
	c := &SSHCommand{}
	authWithKey := authConfig(AuthMethodSSHKey, map[string]any{"secret": "my-secret", "key": "private_key"}, nil)

	// A padded "file" value must not silently run in file mode: the UI would
	// have hidden and dropped commandFile, so Setup must reject it instead of
	// letting the node publish in an inconsistent state.
	err := c.Setup(core.SetupContext{
		Configuration: map[string]any{
			"host":           "example.com",
			"username":       "root",
			"authentication": authWithKey,
			"commandSource":  "\tfile\n",
			"commandFile":    "scripts/deploy.sh",
			"timeout":        60,
		},
		Files: &fakeFilesContext{files: map[string]string{"scripts/deploy.sh": "echo hi"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid command source")
}
