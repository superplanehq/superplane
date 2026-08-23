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
	commands, files := BuildAgentBrokerTask(
		"Prepare",
		NodePrepareScript("", "", ""),
		"run.js",
		"echo run",
		"",
		[]AgentStep{
			{Name: "Clone", Type: AgentStepBash, Command: strPtr("git clone dest repo")},
			{Name: "Implement", Type: AgentStepPrompt, Prompt: &prompt, WorkingDirectory: "repo"},
			{Name: "Push", Type: AgentStepBash, Command: &command, WorkingDirectory: "repo"},
		},
		"google/gemini-3.7-flash",
		func(promptName, model string) string {
			return "node run.js " + promptName + " " + model
		},
	)

	require.Len(t, commands, 4)
	assert.Equal(t, `source "$SUPERPLANE_TASK_DIR/prepare.sh"`, commands[0].Command)
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

func TestBuildAgentBrokerTaskAppliesNodeWorkingDirectoryWhenStepEmpty(t *testing.T) {
	t.Parallel()

	prompt := "implement the change"
	command := "git push"
	commands, _ := BuildAgentBrokerTask(
		"Prepare",
		NodePrepareScript("", "", "repo"),
		"run.js",
		"echo run",
		"repo",
		[]AgentStep{
			{Name: "Clone", Type: AgentStepBash, Command: strPtr("git clone dest repo")},
			{Name: "Implement", Type: AgentStepPrompt, Prompt: &prompt},
			{Name: "Push", Type: AgentStepBash, Command: &command, WorkingDirectory: "/tmp/override"},
		},
		"google/gemini-3.7-flash",
		func(promptName, model string) string {
			return "node run.js " + promptName + " " + model
		},
	)

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
