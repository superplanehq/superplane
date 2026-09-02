package claude

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func strPtr(v string) *string { return &v }

func secretRef(secret, key string) configuration.SecretKeyRef {
	return configuration.SecretKeyRef{Secret: secret, Key: key}
}

func TestDecodeRunClaudeCodeSpecAppliesDefaults(t *testing.T) {
	t.Parallel()

	spec, err := decodeRunClaudeCodeSpec(map[string]any{
		"machineType": testRunnerMachineType,
		"steps": []map[string]any{
			{"name": "Fix bug", "type": "prompt", "prompt": "fix the bug"},
		},
		"anthropicApiKey": map[string]any{
			"secret": "anthropic",
			"key":    "api_key",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, runner.DefaultExecutionTimeoutSeconds, spec.ExecutionTimeoutSeconds)
	require.Len(t, spec.Steps, 1)
	assert.Equal(t, "Fix bug", spec.Steps[0].Name)
	assert.Equal(t, runner.AgentStepPrompt, spec.Steps[0].Type)
}

func TestDecodeRunClaudeCodeSpecMigratesLegacyFields(t *testing.T) {
	t.Parallel()

	spec, err := decodeRunClaudeCodeSpec(map[string]any{
		"machineType":           testRunnerMachineType,
		"prompt":                "implement the issue",
		"enable_setup_commands": true,
		"setup_commands":        "git clone https://github.com/acme/widgets.git /tmp/repo",
		"enable_after_commands": true,
		"after_commands":        "git push",
		"anthropicApiKey": map[string]any{
			"secret": "anthropic",
			"key":    "api_key",
		},
	})
	require.NoError(t, err)
	require.Len(t, spec.Steps, 3)
	assert.Equal(t, "Setup", spec.Steps[0].Name)
	assert.Equal(t, runner.AgentStepBash, spec.Steps[0].Type)
	assert.Equal(t, "Prompt", spec.Steps[1].Name)
	assert.Equal(t, runner.AgentStepPrompt, spec.Steps[1].Type)
	assert.Equal(t, "After", spec.Steps[2].Name)
	assert.Equal(t, runner.AgentStepBash, spec.Steps[2].Type)
}

func TestValidateRunClaudeCodeSpec(t *testing.T) {
	t.Parallel()

	valid := RunClaudeCodeSpec{
		MachineType: testRunnerMachineType,
		Steps: []ClaudeCodeStep{
			{Name: "Do the thing", Type: runner.AgentStepPrompt, Prompt: strPtr("do the thing")},
		},
		Credentials: runner.AgentCredentials{
			Source: "secret",
			Secret: secretRef("anthropic", "api_key"),
		},
	}
	require.NoError(t, validateRunClaudeCodeSpec(valid))

	t.Run("requires step name", func(t *testing.T) {
		spec := valid
		spec.Steps = []ClaudeCodeStep{{Type: runner.AgentStepPrompt, Prompt: strPtr("go")}}
		require.Error(t, validateRunClaudeCodeSpec(spec))
	})

	t.Run("requires steps", func(t *testing.T) {
		spec := valid
		spec.Steps = nil
		require.Error(t, validateRunClaudeCodeSpec(spec))
	})

	t.Run("requires at least one prompt", func(t *testing.T) {
		spec := valid
		spec.Steps = []ClaudeCodeStep{{Name: "Echo", Type: runner.AgentStepBash, Command: strPtr("echo hi")}}
		require.Error(t, validateRunClaudeCodeSpec(spec))
	})

	t.Run("requires model for hosted credentials", func(t *testing.T) {
		spec := valid
		spec.Credentials = runner.AgentCredentials{Source: runner.CredentialsSourceHosted}
		err := validateRunClaudeCodeSpec(spec)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "model is required")
	})

	t.Run("accepts hosted credentials with model", func(t *testing.T) {
		spec := valid
		spec.Credentials = runner.AgentCredentials{Source: runner.CredentialsSourceHosted}
		spec.Model = "claude-sonnet-4-6"
		require.NoError(t, validateRunClaudeCodeSpec(spec))
	})

	t.Run("rejects hosted base URL environment override", func(t *testing.T) {
		spec := valid
		spec.Credentials = runner.AgentCredentials{Source: runner.CredentialsSourceHosted}
		spec.Model = "claude-sonnet-4-6"
		spec.Environment = []runner.EnvironmentVariable{{
			Name:        envAnthropicBaseURL,
			ValueSource: runner.EnvironmentValueSourceLiteral,
			Value:       strPtr("https://attacker.example"),
		}}
		err := validateRunClaudeCodeSpec(spec)
		require.Error(t, err)
		assert.Contains(t, err.Error(), envAnthropicBaseURL)
	})
}

func TestBuildClaudeCodeBrokerTaskRunsOrderedSteps(t *testing.T) {
	t.Parallel()

	spec := RunClaudeCodeSpec{
		Model:            "sonnet",
		WorkingDirectory: "/tmp/workspace",
		Steps: []ClaudeCodeStep{
			{Name: "Clone repo", Type: runner.AgentStepBash, Command: strPtr("git clone https://github.com/acme/widgets.git repo")},
			{Name: "Fix panic", Type: runner.AgentStepPrompt, Prompt: strPtr("Fix auth.py's nil panic")},
			{Name: "Fix tests", Type: runner.AgentStepPrompt, Prompt: strPtr("Run the tests and fix failures")},
			{Name: "Push", Type: runner.AgentStepBash, Command: strPtr("git push")},
		},
	}

	task := buildClaudeCodeBrokerTask(spec, "", nil)
	require.Len(t, task.Commands, 5)
	assert.Equal(t, "Prepare Claude Code", task.Commands[0].Name)
	assert.Equal(t, runner.LiveLogKindSetup, task.Commands[0].Kind)
	assert.Contains(t, task.Commands[0].Command, `source "$SUPERPLANE_TASK_DIR/prepare.sh"`)
	assert.Contains(t, task.Commands[0].Command, `export PATH="$SUPERPLANE_TASK_DIR/bin:$PATH"`)

	assert.Equal(t, "Clone repo", task.Commands[1].Name)
	assert.Equal(t, runner.LiveLogKindBash, task.Commands[1].Kind)
	assert.Equal(t, "git clone https://github.com/acme/widgets.git repo", task.Commands[1].Preview)
	assert.Contains(t, task.Commands[1].Command, `cd '/tmp/workspace'`)
	assert.Contains(t, task.Commands[1].Command, `source "$SUPERPLANE_TASK_DIR/steps/01-clone-repo.sh"`)
	assert.Contains(t, task.Commands[1].Command, `node "$SUPERPLANE_TASK_DIR/llm_usage.js" merge`)
	assert.Equal(t, "Fix panic", task.Commands[2].Name)
	assert.Equal(t, runner.LiveLogKindPrompt, task.Commands[2].Kind)
	assert.Equal(t, "Fix auth.py's nil panic", task.Commands[2].Preview)
	assert.Contains(t, task.Commands[2].Command, `cd '/tmp/workspace'`)
	assert.Contains(t, task.Commands[2].Command, `node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/02-fix-panic.txt" 'sonnet'`)
	assert.Contains(t, task.Commands[2].Command, `node "$SUPERPLANE_TASK_DIR/llm_usage.js" merge`)
	assert.Equal(t, "Fix tests", task.Commands[3].Name)
	assert.Contains(t, task.Commands[3].Command, `node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/03-fix-tests.txt" 'sonnet'`)
	assert.Equal(t, "Push", task.Commands[4].Name)
	assert.Contains(t, task.Commands[4].Command, `source "$SUPERPLANE_TASK_DIR/steps/04-push.sh"`)
	assert.Contains(t, task.Commands[4].Command, `node "$SUPERPLANE_TASK_DIR/llm_usage.js" merge`)

	require.Len(t, task.Files, 7)
	assert.Equal(t, runScript, requireTaskFile(t, task.Files, "run.js").Content)
	assert.Equal(t, runner.LLMUsageScript, requireTaskFile(t, task.Files, "llm_usage.js").Content)
	prepare := requireTaskFile(t, task.Files, "prepare.sh").Content
	assert.Contains(t, prepare, "claude CLI not found")
	assert.Contains(t, prepare, "node not found")
	assert.Contains(t, prepare, "cd '/tmp/workspace'")
	assert.Contains(t, prepare, `pwd -P >"$SUPERPLANE_TASK_DIR/task_cwd"`)
	assert.Contains(t, prepare, `echo "Claude Code ready"`)
	assert.Contains(t, prepare, "claude --version")
	assert.Contains(t, prepare, "node --version")

	assert.Equal(t, "git clone https://github.com/acme/widgets.git repo", requireTaskFile(t, task.Files, "steps/01-clone-repo.sh").Content)

	assert.Equal(t, "Fix auth.py's nil panic", requireTaskFile(t, task.Files, "prompts/02-fix-panic.txt").Content)
	assert.Equal(t, "Run the tests and fix failures", requireTaskFile(t, task.Files, "prompts/03-fix-tests.txt").Content)

	assert.Contains(t, runScript, "stream-json")
	assert.Contains(t, runScript, "--append-system-prompt")
	assert.Contains(t, runScript, "plain terminal text")
	assert.Contains(t, runScript, "--continue")
	assert.Contains(t, runScript, "SUPERPLANE_RESULT_FILE")
	assert.Contains(t, runScript, `"--add-dir"`)
	assert.Contains(t, runScript, `"--permission-mode"`)
	assert.Contains(t, runScript, `"acceptEdits"`)
	assert.Contains(t, runScript, `"bypassPermissions"`)
	assert.Contains(t, runScript, "--mcp-config")
	assert.Contains(t, runScript, "planning_session_mcp.js")
	assert.Contains(t, runScript, "mcp__superplane__propose_draft")
	assert.Contains(t, runScript, "mcp__superplane__survey")
	assert.NotContains(t, runScript, "mcp__superplane__say")
	assert.NotContains(t, runScript, "mcp__superplane__wait_for_user")
	assert.NotContains(t, runScript, "workdir")
}

func TestBuildClaudeCodeBrokerTaskAppliesIntegrationUsageAndSetup(t *testing.T) {
	t.Parallel()

	spec := RunClaudeCodeSpec{
		Model: "sonnet",
		Steps: []ClaudeCodeStep{
			{Name: "Fix tests", Type: runner.AgentStepPrompt, Prompt: strPtr("Fix the failing tests")},
		},
	}

	task := buildClaudeCodeBrokerTask(spec, "The gh CLI is already installed. Use GITHUB_TOKEN.", []runner.IntegrationSetup{
		{Name: "Set up Semaphore", Script: "echo install-sem-ai"},
	})
	require.Len(t, task.Commands, 3)
	assert.Equal(t, "Prepare Claude Code", task.Commands[0].Name)
	assert.Equal(t, "Set up Semaphore", task.Commands[1].Name)
	assert.Equal(t, runner.LiveLogKindSetup, task.Commands[1].Kind)
	assert.Equal(t, "Fix tests", task.Commands[2].Name)
	assert.Equal(t, "Fix the failing tests", task.Commands[2].Preview)
	assert.Equal(t, "echo install-sem-ai", requireTaskFile(t, task.Files, "setup/01-set-up-semaphore.sh").Content)
	assert.Equal(
		t,
		"The gh CLI is already installed. Use GITHUB_TOKEN.\n\nFix the failing tests",
		requireTaskFile(t, task.Files, "prompts/01-fix-tests.txt").Content,
	)
}

func TestClaudeStepSlug(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "01-clone-repo", runner.AgentStepSlug(1, "Clone repo"))
	assert.Equal(t, "02-step", runner.AgentStepSlug(2, "!!!"))
	assert.Equal(t, "03-step", runner.AgentStepSlug(3, "   "))
}

func TestShellSingleQuote(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `'hello'`, runner.ShellSingleQuote("hello"))
	assert.Equal(t, `'it'\''s fine'`, runner.ShellSingleQuote("it's fine"))
}

func TestApplyPlanningFollowUpLeavesLineAutomationsUnchanged(t *testing.T) {
	t.Parallel()

	spec := RunClaudeCodeSpec{
		Model: "sonnet",
		Steps: []ClaudeCodeStep{
			{Name: "Fix tests", Type: runner.AgentStepPrompt, Prompt: strPtr("fix"), WorkingDirectory: "repo"},
		},
	}
	base := buildClaudeCodeBrokerTask(spec, "", nil)
	got := applyPlanningFollowUp(base, nil, spec)
	assert.Len(t, got.Commands, len(base.Commands))
	assert.Len(t, got.Files, len(base.Files))
}

func TestApplyPlanningFollowUpAppendsWaitLoopForPlanningToken(t *testing.T) {
	t.Parallel()

	spec := RunClaudeCodeSpec{
		Model: "opus",
		Steps: []ClaudeCodeStep{
			{Name: "Clone", Type: runner.AgentStepBash, Command: strPtr("git clone")},
			{Name: "Hello", Type: runner.AgentStepPrompt, Prompt: strPtr("greet"), WorkingDirectory: "repo"},
		},
	}
	base := buildClaudeCodeBrokerTask(spec, "", nil)
	got := applyPlanningFollowUp(base, []runner.BrokerEnvironmentVariable{{
		Name:  runner.EnvSuperplanePlanningID,
		Value: "session-1",
	}}, spec)

	require.Len(t, got.Commands, len(base.Commands)+1)
	last := got.Commands[len(got.Commands)-1]
	assert.Equal(t, "Wait for the next message", last.Name)
	assert.Equal(t, runner.LiveLogKindPrompt, last.Kind)
	assert.Contains(t, last.Command, `node "$SUPERPLANE_TASK_DIR/follow_up_loop.js" 'opus'`)
	assert.Contains(t, last.Command, `cd "$_sp_root"/'repo'`)
	assert.Equal(t, followUpLoopScript, requireTaskFile(t, got.Files, "follow_up_loop.js").Content)
}

func requireEnvironmentValue(t *testing.T, environment []runner.BrokerEnvironmentVariable, name string) string {
	t.Helper()
	for _, variable := range environment {
		if variable.Name == name {
			return variable.Value
		}
	}
	t.Fatalf("missing environment variable %q", name)
	return ""
}

func requireTaskFile(t *testing.T, files []runner.BrokerTaskFile, path string) runner.BrokerTaskFile {
	t.Helper()
	for _, file := range files {
		if file.Path == path {
			return file
		}
	}
	t.Fatalf("missing task file %q", path)
	return runner.BrokerTaskFile{}
}
