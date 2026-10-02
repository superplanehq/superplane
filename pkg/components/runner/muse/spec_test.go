package muse

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func museString(v string) *string { return &v }

func TestValidateRunMuseSpecRejectsReservedCredentials(t *testing.T) {
	t.Parallel()
	prompt := "fix tests"
	value := "do-not-store-here"
	spec := RunMuseSpec{
		MachineType: "e1-large-amd64",
		Steps:       []runner.AgentStep{{Name: "Prompt", Type: runner.AgentStepPrompt, Prompt: &prompt}},
		Credentials: runner.AgentCredentials{
			Source: runner.CredentialsSourceSecret,
			Secret: configuration.SecretKeyRef{Secret: "meta", Key: "api_key"},
		},
		Environment: []runner.EnvironmentVariable{{Name: envMetaAPIKey, ValueSource: runner.EnvironmentValueSourceLiteral, Value: &value}},
	}
	require.ErrorContains(t, validateRunMuseSpec(spec), envMetaAPIKey)
}

func TestBuildMuseBrokerTaskRunsOrderedSteps(t *testing.T) {
	t.Parallel()
	spec := RunMuseSpec{
		Model:            "muse-spark-1.3",
		ThinkingLevel:    "medium",
		WorkingDirectory: "/tmp/workspace",
		Steps: []runner.AgentStep{
			{Name: "Clone repo", Type: runner.AgentStepBash, Command: museString("git clone https://github.com/acme/widgets.git repo")},
			{Name: "Fix panic", Type: runner.AgentStepPrompt, Prompt: museString("Fix auth.py's nil panic")},
		},
	}
	task := buildMuseBrokerTask(spec, "", nil, nil, nil, false)
	require.Len(t, task.Commands, 3)
	assert.Equal(t, "Prepare Muse Code", task.Commands[0].Name)
	assert.Equal(t, "Clone repo", task.Commands[1].Name)
	assert.Equal(t, "Fix panic", task.Commands[2].Name)
	assert.Contains(t, task.Commands[2].Command, `node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/02-fix-panic.txt" 'muse-spark-1.3' 'medium'`)
	assert.Equal(t, runScript, museTaskFile(t, task.Files, "run.js").Content)
	assert.Equal(t, "Fix auth.py's nil panic", museTaskFile(t, task.Files, "prompts/02-fix-panic.txt").Content)
}

func museTaskFile(t *testing.T, files []runner.BrokerTaskFile, path string) runner.BrokerTaskFile {
	t.Helper()
	for _, file := range files {
		if file.Path == path {
			return file
		}
	}
	t.Fatalf("missing task file %q", path)
	return runner.BrokerTaskFile{}
}
