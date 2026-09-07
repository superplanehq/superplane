package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEffectiveWorkingDirectoryPrefersStepThenNode(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "repo", EffectiveWorkingDirectory("workspace", "repo"))
	assert.Equal(t, "workspace", EffectiveWorkingDirectory("workspace", ""))
	assert.Equal(t, "repo", EffectiveWorkingDirectory("", "repo"))
	assert.Equal(t, "", EffectiveWorkingDirectory("  ", "  "))
}

func TestWrapCommandInWorkingDirectoryEmptyIsNoOp(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "git push", WrapCommandInWorkingDirectory("", "git push"))
	assert.Equal(t, "git push", WrapCommandInWorkingDirectory("  ", "git push"))
}

func TestWrapCommandInWorkingDirectoryUsesTaskLaunchDir(t *testing.T) {
	t.Parallel()

	got := WrapCommandInWorkingDirectory("repo", `source "$SUPERPLANE_TASK_DIR/steps/05-dco.sh"`)
	assert.Contains(t, got, `cat "$SUPERPLANE_TASK_DIR/task_cwd"`)
	assert.Contains(t, got, `cd "$_sp_root"/'repo'`)
	assert.Contains(t, got, `source "$SUPERPLANE_TASK_DIR/steps/05-dco.sh"`)
}

func TestWrapCommandInWorkingDirectoryAllowsAbsolutePath(t *testing.T) {
	t.Parallel()

	got := WrapCommandInWorkingDirectory("/tmp/workspace", "node run.js")
	assert.Equal(t, `cd '/tmp/workspace' && node run.js`, got)
}

func TestBuildAgentBrokerTaskAppliesStepWorkingDirectory(t *testing.T) {
	t.Parallel()

	prompt := "implement the change"
	command := "git push"
	commands, files := BuildAgentBrokerTask(AgentBrokerTaskInput{
		PrepareName:   "Prepare",
		PrepareScript: NodePrepareScript("", "", ""),
		RunScriptName: "run.js",
		RunScript:     "echo run",
		Steps: []AgentStep{
			{Name: "Clone", Type: AgentStepBash, Command: strPtr("git clone dest repo")},
			{Name: "Implement", Type: AgentStepPrompt, Prompt: &prompt, WorkingDirectory: "repo"},
			{Name: "Push", Type: AgentStepBash, Command: &command, WorkingDirectory: "repo"},
		},
		Model: "google/gemini-3.7-flash",
		PromptCommand: func(promptName, model string) string {
			return "node run.js " + promptName + " " + model
		},
	})

	require.Len(t, commands, 4)
	assert.Equal(t, LiveLogKindSetup, commands[0].Kind)
	assert.Contains(t, commands[0].Command, `source "$SUPERPLANE_TASK_DIR/prepare.sh"`)
	assert.Contains(t, commands[0].Command, `export PATH="$SUPERPLANE_TASK_DIR/bin:$PATH"`)
	assert.Equal(t, LiveLogKindBash, commands[1].Kind)
	assert.Equal(t, "git clone dest repo", commands[1].Preview)
	assert.Equal(t, LiveLogKindPrompt, commands[2].Kind)
	assert.Equal(t, "implement the change", commands[2].Preview)
	assertAgentStepMergesUsage(t, commands[1].Command, `source "$SUPERPLANE_TASK_DIR/steps/01-clone.sh"`)
	assert.Contains(t, commands[2].Command, `cat "$SUPERPLANE_TASK_DIR/task_cwd"`)
	assert.Contains(t, commands[2].Command, `cd "$_sp_root"/'repo'`)
	assert.Contains(t, commands[2].Command, "node run.js 02-implement.txt")
	assertAgentStepMergesUsage(t, commands[2].Command, "node run.js 02-implement.txt")
	assert.Contains(t, commands[3].Command, `cat "$SUPERPLANE_TASK_DIR/task_cwd"`)
	assert.Contains(t, commands[3].Command, `cd "$_sp_root"/'repo'`)
	assertAgentStepMergesUsage(t, commands[3].Command, `source "$SUPERPLANE_TASK_DIR/steps/03-push.sh"`)

	prepare := requireBrokerFile(t, files, "prepare.sh").Content
	assert.Contains(t, prepare, `pwd -P >"$SUPERPLANE_TASK_DIR/task_cwd"`)
	assert.Equal(t, LLMUsageScript, requireBrokerFile(t, files, "llm_usage.js").Content)
}

func TestBuildAgentBrokerTaskPreviewKeepsFullMultilineBody(t *testing.T) {
	t.Parallel()

	prompt := "\nFirst line.\nSecond line.\n"
	command := "set -e\necho one\necho two"
	commands, _ := BuildAgentBrokerTask(AgentBrokerTaskInput{
		PrepareName:   "Prepare",
		PrepareScript: NodePrepareScript("", "", ""),
		RunScriptName: "run.js",
		RunScript:     "echo run",
		Steps: []AgentStep{
			{Name: "Run", Type: AgentStepBash, Command: &command},
			{Name: "Implement", Type: AgentStepPrompt, Prompt: &prompt},
		},
		Model: "google/gemini-3.7-flash",
		PromptCommand: func(promptName, model string) string {
			return "node run.js " + promptName + " " + model
		},
	})

	require.Len(t, commands, 3)
	assert.Equal(t, "set -e\necho one\necho two", commands[1].Preview)
	assert.Equal(t, "First line.\nSecond line.", commands[2].Preview)
}

func TestBuildAgentBrokerTaskAppliesNodeWorkingDirectoryWhenStepEmpty(t *testing.T) {
	t.Parallel()

	prompt := "implement the change"
	command := "git push"
	commands, _ := BuildAgentBrokerTask(AgentBrokerTaskInput{
		PrepareName:      "Prepare",
		PrepareScript:    NodePrepareScript("", "", "repo"),
		RunScriptName:    "run.js",
		RunScript:        "echo run",
		WorkingDirectory: "repo",
		Steps: []AgentStep{
			{Name: "Clone", Type: AgentStepBash, Command: strPtr("git clone dest repo")},
			{Name: "Implement", Type: AgentStepPrompt, Prompt: &prompt},
			{Name: "Push", Type: AgentStepBash, Command: &command, WorkingDirectory: "/tmp/override"},
		},
		Model: "google/gemini-3.7-flash",
		PromptCommand: func(promptName, model string) string {
			return "node run.js " + promptName + " " + model
		},
	})

	require.Len(t, commands, 4)
	assert.Contains(t, commands[1].Command, `'repo'`)
	assert.Contains(t, commands[2].Command, `'repo'`)
	assert.Contains(t, commands[3].Command, `cd '/tmp/override'`)
	assert.NotContains(t, commands[3].Command, `'repo'`)
}

func TestValidateAgentStepsRejectsParentWorkingDirectory(t *testing.T) {
	t.Parallel()

	prompt := "go"
	err := ValidateAgentSteps([]AgentStep{
		{Name: "Prompt", Type: AgentStepPrompt, Prompt: &prompt, WorkingDirectory: "../outside"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workingDirectory")
}

func TestApplyIntegrationUsagePrependsUsage(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "user prompt", ApplyIntegrationUsage("user prompt", ""))
	assert.Equal(t, "Use gh.", ApplyIntegrationUsage("", "Use gh."))
	assert.Equal(t, "Use gh.\n\nuser prompt", ApplyIntegrationUsage("user prompt", "Use gh."))
}

func TestBuildAgentBrokerTaskAppliesIntegrationUsageAndSetup(t *testing.T) {
	t.Parallel()

	prompt := "implement the change"
	commands, files := BuildAgentBrokerTask(AgentBrokerTaskInput{
		PrepareName:   "Prepare",
		PrepareScript: NodePrepareScript("", "", ""),
		RunScriptName: "run.js",
		RunScript:     "echo run",
		Steps: []AgentStep{
			{Name: "Implement", Type: AgentStepPrompt, Prompt: &prompt},
		},
		Usage: "The gh CLI is already installed. Use GITHUB_TOKEN.",
		Setups: []IntegrationSetup{
			{Name: "Set up Semaphore", Script: "echo install-sem-ai"},
		},
		Model: "google/gemini-3.7-flash",
		PromptCommand: func(promptName, model string) string {
			return "node run.js " + promptName + " " + model
		},
	})

	require.Len(t, commands, 3)
	assert.Equal(t, "Prepare", commands[0].Name)
	assert.Equal(t, LiveLogKindSetup, commands[0].Kind)
	assert.Equal(t, "Set up Semaphore", commands[1].Name)
	assert.Equal(t, LiveLogKindSetup, commands[1].Kind)
	assert.Equal(t, "Set up Semaphore", commands[1].Preview)
	assert.Contains(t, commands[1].Command, `source "$SUPERPLANE_TASK_DIR/setup/01-set-up-semaphore.sh"`)
	assert.Equal(t, "Implement", commands[2].Name)
	assert.Equal(t, "implement the change", commands[2].Preview)
	assert.Contains(t, commands[2].Command, `export PATH="$SUPERPLANE_TASK_DIR/bin:$PATH"`)

	assert.Equal(t, "echo install-sem-ai", requireBrokerFile(t, files, "setup/01-set-up-semaphore.sh").Content)
	assert.Equal(
		t,
		"The gh CLI is already installed. Use GITHUB_TOKEN.\n\nimplement the change",
		requireBrokerFile(t, files, "prompts/01-implement.txt").Content,
	)
}

func assertAgentStepMergesUsage(t *testing.T, command, inner string) {
	t.Helper()
	assert.Contains(t, command, inner)
	assert.Contains(t, command, `node "$SUPERPLANE_TASK_DIR/llm_usage.js" merge`)
}

func requireBrokerFile(t *testing.T, files []BrokerTaskFile, path string) BrokerTaskFile {
	t.Helper()
	for _, file := range files {
		if file.Path == path {
			return file
		}
	}
	t.Fatalf("missing task file %q", path)
	return BrokerTaskFile{}
}
