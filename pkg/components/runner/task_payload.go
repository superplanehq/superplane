package runner

import (
	"encoding/json"
	"strings"

	runnerapi "github.com/superplanehq/superplane/pkg/runners/api"
	runnermodels "github.com/superplanehq/superplane/pkg/runners/models"
)

type TaskCommand = runnermodels.CommandSpec
type TaskEnvironmentVariable = runnerapi.EnvironmentVariable
type TaskFile = runnerapi.TaskFile

// TaskCommandsFromLines builds command directives with live log labels.
func TaskCommandsFromLines(lines []string) []TaskCommand {
	out := make([]TaskCommand, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, TaskCommand{
			Command: line,
			Kind:    LiveLogKindBash,
			Preview: LiveLogText(line),
		})
	}
	return out
}

const (
	RunModeJavaScript = "javascript_script"
	RunModePython     = "python_script"
	RunModeBash       = "bash_script"
)

// CreateTaskParams describes an integrated runner task.
type CreateTaskParams struct {
	MachineType             string
	RunMode                 string
	Script                  string
	MessageChain            json.RawMessage
	Commands                []TaskCommand
	SetupCommands           []string
	WebhookPayloadSizeLimit int
	Environment             []TaskEnvironmentVariable
	Files                   []TaskFile
	ExecutionMode           string
	DockerImage             string
	TimeoutSeconds          int // 0 = DefaultExecutionTimeoutSeconds
}
