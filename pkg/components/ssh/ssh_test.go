package ssh

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

const testMachineType = runner.MachineTypeE1LargeAMD64

// brokerRequest captures the subset of the task-broker POST /v1/tasks body the
// SSH component is expected to send. It mirrors the broker's JSON contract
// without depending on the runner package's unexported request struct.
type brokerRequest struct {
	FleetID                 string `json:"fleet_id"`
	RunMode                 string `json:"run_mode"`
	Script                  string `json:"script"`
	ExecutionMode           string `json:"execution_mode"`
	ExecutionTimeoutSeconds *int   `json:"execution_timeout_seconds"`
	Environment             []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"environment"`
}

// fakeFilesContext is an in-memory core.RepositoryFilesContext used to drive
// file-mode SSH tests without spinning up a git provider.
type fakeFilesContext struct {
	files   map[string]string
	listErr error
	readErr error
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
	if f.readErr != nil {
		return nil, f.readErr
	}
	content, ok := f.files[path]
	if !ok {
		return nil, fmt.Errorf("file %q not found", path)
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

func keyRef(secret, key string) map[string]any {
	return map[string]any{"secret": secret, "key": key}
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

func setBrokerEnv(t *testing.T) {
	t.Helper()
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")
}

func createTaskResponse(id string) *contexts.HTTPContext {
	return &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"` + id + `"}`))},
		},
	}
}

func decodeBrokerRequest(t *testing.T, http *contexts.HTTPContext) brokerRequest {
	t.Helper()
	require.Len(t, http.Requests, 1)
	body, err := io.ReadAll(http.Requests[0].Body)
	require.NoError(t, err)
	var req brokerRequest
	require.NoError(t, json.Unmarshal(body, &req))
	return req
}

// Legacy SSH nodes were saved before the commandSource field existed, so their
// stored configuration omits it entirely. ValidateConfiguration does not apply
// Field.Default, so commandSource must stay optional at the schema level.
func TestSSHCommand_ValidateConfiguration_LegacyConfigWithoutCommandSource(t *testing.T) {
	c := &SSHCommand{}
	legacyConfig := map[string]any{
		"machineType": testMachineType,
		"host":        "example.com",
		"port":        22,
		"username":    "root",
		"authentication": map[string]any{
			"authMethod": AuthMethodPassword,
			"password":   keyRef("my-secret", "password"),
		},
		"commands": "echo hi\nls -la",
		"timeout":  60,
	}

	require.NoError(t, configuration.ValidateConfiguration(c.Configuration(), legacyConfig))
}

func TestSSHCommand_Setup_ValidatesRequiredFields(t *testing.T) {
	c := &SSHCommand{}
	authWithKey := authConfig(AuthMethodSSHKey, keyRef("my-secret", "private_key"), nil)

	base := func() map[string]any {
		return map[string]any{
			"machineType":    testMachineType,
			"host":           "example.com",
			"username":       "root",
			"authentication": authWithKey,
			"commands":       "ls",
		}
	}

	t.Run("missing machine type", func(t *testing.T) {
		config := base()
		delete(config, "machineType")
		err := c.Setup(core.SetupContext{Configuration: config, Webhook: &contexts.NodeWebhookContext{}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "machine type")
	})

	t.Run("missing host", func(t *testing.T) {
		config := base()
		delete(config, "host")
		err := c.Setup(core.SetupContext{Configuration: config, Webhook: &contexts.NodeWebhookContext{}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "host")
	})

	t.Run("missing username", func(t *testing.T) {
		config := base()
		delete(config, "username")
		err := c.Setup(core.SetupContext{Configuration: config, Webhook: &contexts.NodeWebhookContext{}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "username")
	})

	t.Run("missing commands", func(t *testing.T) {
		config := base()
		delete(config, "commands")
		err := c.Setup(core.SetupContext{Configuration: config, Webhook: &contexts.NodeWebhookContext{}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "commands")
	})

	t.Run("ssh_key auth without secret ref", func(t *testing.T) {
		config := base()
		config["authentication"] = authConfig(AuthMethodSSHKey, nil, nil)
		err := c.Setup(core.SetupContext{Configuration: config, Webhook: &contexts.NodeWebhookContext{}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "private key")
	})

	t.Run("password auth without secret ref", func(t *testing.T) {
		config := base()
		config["authentication"] = authConfig(AuthMethodPassword, nil, nil)
		err := c.Setup(core.SetupContext{Configuration: config, Webhook: &contexts.NodeWebhookContext{}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "password")
	})

	t.Run("timeout above maximum", func(t *testing.T) {
		config := base()
		config["timeout"] = maxTimeoutSeconds + 1
		err := c.Setup(core.SetupContext{Configuration: config, Webhook: &contexts.NodeWebhookContext{}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
	})

	t.Run("valid config", func(t *testing.T) {
		require.NoError(t, c.Setup(core.SetupContext{Configuration: base(), Webhook: &contexts.NodeWebhookContext{}}))
	})
}

func TestSSHCommand_Setup_FileMode(t *testing.T) {
	c := &SSHCommand{}
	config := func(path string) map[string]any {
		return map[string]any{
			"machineType":    testMachineType,
			"host":           "example.com",
			"username":       "root",
			"authentication": authConfig(AuthMethodSSHKey, keyRef("s", "k"), nil),
			"commandSource":  CommandSourceFile,
			"commandFile":    path,
		}
	}

	t.Run("file exists", func(t *testing.T) {
		files := &fakeFilesContext{files: map[string]string{"scripts/deploy.sh": "echo deploy\n"}}
		require.NoError(t, c.Setup(core.SetupContext{Configuration: config("scripts/deploy.sh"), Files: files, Webhook: &contexts.NodeWebhookContext{}}))
	})

	t.Run("file missing", func(t *testing.T) {
		files := &fakeFilesContext{files: map[string]string{"scripts/other.sh": "echo x\n"}}
		err := c.Setup(core.SetupContext{Configuration: config("scripts/deploy.sh"), Files: files, Webhook: &contexts.NodeWebhookContext{}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("empty file", func(t *testing.T) {
		files := &fakeFilesContext{files: map[string]string{"scripts/deploy.sh": "   \n"}}
		err := c.Setup(core.SetupContext{Configuration: config("scripts/deploy.sh"), Files: files, Webhook: &contexts.NodeWebhookContext{}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})
}

func TestSSHCommand_Setup_RejectsPaddedCommandSource(t *testing.T) {
	c := &SSHCommand{}
	err := c.Setup(core.SetupContext{
		Configuration: map[string]any{
			"machineType":    testMachineType,
			"host":           "example.com",
			"username":       "root",
			"authentication": authConfig(AuthMethodSSHKey, keyRef("s", "k"), nil),
			"commandSource":  "\tinline\n",
			"commands":       "echo hi",
		},
		Webhook: &contexts.NodeWebhookContext{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid command source")
}

func TestSSHCommand_CommandSourceOrDefault(t *testing.T) {
	assert.Equal(t, CommandSourceInline, Spec{}.commandSourceOrDefault())
	assert.Equal(t, CommandSourceInline, Spec{CommandSource: "   "}.commandSourceOrDefault())
	assert.Equal(t, CommandSourceFile, Spec{CommandSource: CommandSourceFile}.commandSourceOrDefault())
	assert.Equal(t, "\tfile\n", Spec{CommandSource: "\tfile\n"}.commandSourceOrDefault())
}

func TestSSHCommand_Execute_SubmitsRunnerTask(t *testing.T) {
	setBrokerEnv(t)
	httpContext := createTaskResponse("task-1")
	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	requests := &contexts.RequestContext{}

	err := (&SSHCommand{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machineType":    testMachineType,
			"host":           "example.com",
			"port":           2222,
			"username":       "deploy",
			"authentication": authConfig(AuthMethodSSHKey, keyRef("ssh", "key"), nil),
			"commands":       "echo hi\nuptime",
			"environment":    []map[string]any{{"name": "STAGE", "value": "prod"}},
			"timeout":        120,
		},
		HTTP:           httpContext,
		Secrets:        &contexts.SecretsContext{Values: map[string][]byte{"ssh/key": []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n-----END OPENSSH PRIVATE KEY-----")}},
		Webhook:        &contexts.NodeWebhookContext{},
		Metadata:       &contexts.MetadataContext{},
		ExecutionState: state,
		Requests:       requests,
	})
	require.NoError(t, err)

	req := decodeBrokerRequest(t, httpContext)
	assert.Equal(t, testMachineType, req.FleetID)
	assert.Equal(t, runner.RunModeBash, req.RunMode)
	assert.Equal(t, runner.ExecutionModeHost, req.ExecutionMode)
	require.NotNil(t, req.ExecutionTimeoutSeconds)
	assert.Equal(t, 120, *req.ExecutionTimeoutSeconds)

	// The private key travels as a reserved env var, never in the script body.
	require.Len(t, req.Environment, 1)
	assert.Equal(t, envPrivateKey, req.Environment[0].Name)
	assert.Contains(t, req.Environment[0].Value, "BEGIN OPENSSH PRIVATE KEY")
	assert.NotContains(t, req.Script, "BEGIN OPENSSH PRIVATE KEY")

	// The runner script opens an SSH connection and carries the remote script
	// (with env + commands) as a base64 payload.
	assert.Contains(t, req.Script, "ssh -i \"$key_file\"")
	assert.Contains(t, req.Script, "-p 2222")
	assert.Contains(t, req.Script, "'deploy'@'example.com'")

	remote := decodeRemoteScript(t, req.Script)
	assert.Contains(t, remote, "export STAGE='prod'")
	assert.Contains(t, remote, "echo hi\nuptime")

	// Lifecycle: task id recorded and a poll scheduled.
	assert.Equal(t, "task-1", state.KVs["task_id"])
	assert.Equal(t, runner.HookActionPoll, requests.Action)
}

func TestSSHCommand_Execute_PasswordAuthSendsPasswordEnv(t *testing.T) {
	setBrokerEnv(t)
	httpContext := createTaskResponse("task-2")

	err := (&SSHCommand{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machineType":    testMachineType,
			"host":           "example.com",
			"username":       "root",
			"authentication": authConfig(AuthMethodPassword, nil, keyRef("ssh", "password")),
			"commands":       "whoami",
		},
		HTTP:           httpContext,
		Secrets:        &contexts.SecretsContext{Values: map[string][]byte{"ssh/password": []byte("s3cret")}},
		Webhook:        &contexts.NodeWebhookContext{},
		Metadata:       &contexts.MetadataContext{},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
	})
	require.NoError(t, err)

	req := decodeBrokerRequest(t, httpContext)
	require.Len(t, req.Environment, 1)
	assert.Equal(t, envPassword, req.Environment[0].Name)
	assert.Equal(t, "s3cret", req.Environment[0].Value)
	assert.Contains(t, req.Script, "sshpass -e ssh")
	assert.Contains(t, req.Script, "PubkeyAuthentication=no")
}

func TestSSHCommand_Execute_FileMode(t *testing.T) {
	setBrokerEnv(t)
	httpContext := createTaskResponse("task-3")
	files := &fakeFilesContext{files: map[string]string{"scripts/deploy.sh": "#!/usr/bin/env bash\necho deploy\n"}}

	err := (&SSHCommand{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machineType":    testMachineType,
			"host":           "example.com",
			"username":       "root",
			"authentication": authConfig(AuthMethodSSHKey, keyRef("ssh", "key"), nil),
			"commandSource":  CommandSourceFile,
			"commandFile":    "scripts/deploy.sh",
		},
		HTTP:           httpContext,
		Files:          files,
		Secrets:        &contexts.SecretsContext{Values: map[string][]byte{"ssh/key": []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n-----END OPENSSH PRIVATE KEY-----")}},
		Webhook:        &contexts.NodeWebhookContext{},
		Metadata:       &contexts.MetadataContext{},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
	})
	require.NoError(t, err)

	req := decodeBrokerRequest(t, httpContext)
	remote := decodeRemoteScript(t, req.Script)
	assert.Contains(t, remote, "echo deploy")
}

func TestSSHCommand_Execute_MissingSecretFails(t *testing.T) {
	setBrokerEnv(t)

	err := (&SSHCommand{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machineType":    testMachineType,
			"host":           "example.com",
			"username":       "root",
			"authentication": authConfig(AuthMethodSSHKey, keyRef("ssh", "key"), nil),
			"commands":       "echo hi",
		},
		HTTP:           &contexts.HTTPContext{},
		Secrets:        &contexts.SecretsContext{Values: map[string][]byte{}},
		Webhook:        &contexts.NodeWebhookContext{},
		Metadata:       &contexts.MetadataContext{},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private key")
}

func TestBuildRunnerScript_ConnectionAndExecutionRetries(t *testing.T) {
	spec := Spec{
		Host:            "h",
		Port:            22,
		User:            "u",
		Authentication:  AuthSpec{Method: AuthMethodSSHKey},
		ConnectionRetry: &RetrySpec{Enabled: true, Retries: 3, IntervalSeconds: 10},
		ExecutionRetry:  &RetrySpec{Enabled: true, Retries: 2, IntervalSeconds: 5},
	}

	script := buildRunnerScript(spec, "echo hi\n")
	assert.Contains(t, script, "connect_retries=3")
	assert.Contains(t, script, "connect_interval=10")
	assert.Contains(t, script, "exec_retries=2")
	assert.Contains(t, script, "exec_interval=5")
	assert.Contains(t, script, `[ "$code" -eq 255 ]`)
	assert.Contains(t, script, "key_file=")
}

func TestBuildRunnerScript_NoRetriesByDefault(t *testing.T) {
	spec := Spec{Host: "h", Port: 22, User: "u", Authentication: AuthSpec{Method: AuthMethodSSHKey}}
	script := buildRunnerScript(spec, "echo hi\n")
	assert.Contains(t, script, "connect_retries=0")
	assert.Contains(t, script, "exec_retries=0")
}

func TestSSHInvocation_PassphraseUsesSshpass(t *testing.T) {
	spec := Spec{
		Host:           "h",
		Port:           22,
		User:           "u",
		Authentication: AuthSpec{Method: AuthMethodSSHKey, Passphrase: configuration.SecretKeyRef{Secret: "s", Key: "k"}},
	}
	inv := sshInvocation(spec)
	assert.Contains(t, inv, "sshpass -P assphrase -e ssh -i \"$key_file\"")
	assert.Contains(t, inv, envPassphrase)
}

func TestBuildRemoteScript_WorkingDirectoryAndEnv(t *testing.T) {
	remote := buildRemoteScript(
		[]EnvironmentVariable{{Name: "FOO", Value: "b'ar"}},
		"/srv/app",
		"echo hi",
	)
	assert.Contains(t, remote, "set -e\n")
	assert.Contains(t, remote, `export FOO='b'"'"'ar'`)
	assert.Contains(t, remote, "cd '/srv/app' || exit 1")
	assert.True(t, strings.HasSuffix(remote, "echo hi\n"))
}

func TestNormalizePrivateKey(t *testing.T) {
	t.Run("escaped newlines restored", func(t *testing.T) {
		got := string(normalizePrivateKey([]byte(`"-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n-----END OPENSSH PRIVATE KEY-----"`)))
		assert.Equal(t, "-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n-----END OPENSSH PRIVATE KEY-----\n", got)
	})

	t.Run("base64 blob decoded", func(t *testing.T) {
		raw := "-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n-----END OPENSSH PRIVATE KEY-----"
		encoded := base64.StdEncoding.EncodeToString([]byte(raw))
		got := string(normalizePrivateKey([]byte(encoded)))
		assert.Equal(t, raw+"\n", got)
	})

	t.Run("trailing newline added once", func(t *testing.T) {
		got := string(normalizePrivateKey([]byte("-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n-----END OPENSSH PRIVATE KEY-----\n")))
		assert.Equal(t, "-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n-----END OPENSSH PRIVATE KEY-----\n", got)
	})
}

// decodeRemoteScript extracts and base64-decodes the remote script embedded in
// the generated runner script.
func decodeRemoteScript(t *testing.T, runnerScript string) string {
	t.Helper()
	const begin = "base64 -d <<'SUPERPLANE_SSH_SCRIPT_EOF'\n"
	const end = "\nSUPERPLANE_SSH_SCRIPT_EOF"
	start := strings.Index(runnerScript, begin)
	require.GreaterOrEqual(t, start, 0)
	rest := runnerScript[start+len(begin):]
	stop := strings.Index(rest, end)
	require.GreaterOrEqual(t, stop, 0)
	decoded, err := base64.StdEncoding.DecodeString(rest[:stop])
	require.NoError(t, err)
	return string(decoded)
}
