package runner

import "fmt"

type AgentPromptCommand func(promptName, model string) string

func BuildAgentBrokerTask(
	prepareName, prepareScript, runScriptName, runScript string,
	steps []AgentStep,
	model string,
	promptCommand AgentPromptCommand,
) (commands []BrokerCommand, files []BrokerTaskFile) {
	files = []BrokerTaskFile{
		{Path: runScriptName, Content: runScript, Mode: "0644"},
		{Path: "prepare.sh", Content: prepareScript, Mode: "0644"},
	}

	commands = make([]BrokerCommand, 0, len(steps)+1)
	commands = append(commands, BrokerCommand{
		Name:    prepareName,
		Command: `source "$SUPERPLANE_TASK_DIR/prepare.sh"`,
	})

	for i, step := range steps {
		file, command := buildAgentStep(i+1, step, model, runScriptName, promptCommand)
		files = append(files, file)
		commands = append(commands, command)
	}
	return commands, files
}

func buildAgentStep(stepNumber int, step AgentStep, model, runScriptName string, promptCommand AgentPromptCommand) (BrokerTaskFile, BrokerCommand) {
	stepSlug := AgentStepSlug(stepNumber, step.Name)
	switch NormalizeAgentStepType(step.Type) {
	case AgentStepBash:
		command := ""
		if step.Command != nil {
			command = *step.Command
		}
		scriptName := stepSlug + ".sh"
		return BrokerTaskFile{
				Path:    "steps/" + scriptName,
				Content: command,
				Mode:    "0644",
			}, BrokerCommand{
				Name:    AgentStepLabel(step.Name, scriptName),
				Command: fmt.Sprintf(`source "$SUPERPLANE_TASK_DIR/steps/%s"`, scriptName),
			}
	default:
		prompt := ""
		if step.Prompt != nil {
			prompt = *step.Prompt
		}
		promptName := stepSlug + ".txt"
		return BrokerTaskFile{
				Path:    "prompts/" + promptName,
				Content: prompt,
				Mode:    "0644",
			}, BrokerCommand{
				Name:    AgentStepLabel(step.Name, promptName),
				Command: promptCommand(promptName, model),
			}
	}
}

func NodePrepareScript(cliName, cliMissingMessage string, workdir string) string {
	var prepare string
	prepare += "set -euo pipefail\n"
	prepare += ": \"${SUPERPLANE_TASK_DIR:?SUPERPLANE_TASK_DIR is required}\"\n"
	if cliName != "" {
		prepare += "if ! command -v " + cliName + " >/dev/null 2>&1; then\n"
		prepare += "  echo " + ShellSingleQuote(cliMissingMessage) + " >&2\n"
		prepare += "  return 127\n"
		prepare += "fi\n"
	}
	prepare += "if ! command -v node >/dev/null 2>&1; then\n"
	prepare += "  echo \"node not found on PATH; required to run prompt steps\" >&2\n"
	prepare += "  return 127\n"
	prepare += "fi\n"
	prepare += "printf '0\\n' >\"$SUPERPLANE_TASK_DIR/prompt_count\"\n"
	if workdir != "" {
		prepare += "cd " + ShellSingleQuote(workdir) + "\n"
	}
	prepare += "echo \"Agent ready\"\n"
	if cliName != "" {
		prepare += "echo \"" + cliName + "=$(" + cliName + " --version 2>/dev/null | head -n1)\"\n"
	}
	prepare += "echo \"node=$(node --version 2>/dev/null)\"\n"
	prepare += "echo \"cwd=$(pwd -P)\"\n"
	return prepare
}
