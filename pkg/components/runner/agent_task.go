package runner

import (
	"fmt"
	"path/filepath"
	"strings"
)

type AgentPromptCommand func(promptName, model string) string

func BuildAgentBrokerTask(
	prepareName, prepareScript, runScriptName, runScript, workingDirectory string,
	steps []AgentStep,
	model string,
	promptCommand AgentPromptCommand,
) (commands []BrokerCommand, files []BrokerTaskFile) {
	files = []BrokerTaskFile{
		LLMUsageTaskFile(),
		{Path: runScriptName, Content: runScript, Mode: "0644"},
		{Path: "prepare.sh", Content: prepareScript, Mode: "0644"},
	}

	commands = make([]BrokerCommand, 0, len(steps)+1)
	commands = append(commands, BrokerCommand{
		Name:    prepareName,
		Command: `source "$SUPERPLANE_TASK_DIR/prepare.sh"`,
		Kind:    LiveLogKindSetup,
	})

	for i, step := range steps {
		file, command := buildAgentStep(i+1, step, workingDirectory, model, runScriptName, promptCommand)
		files = append(files, file)
		commands = append(commands, command)
	}
	return commands, files
}

func buildAgentStep(stepNumber int, step AgentStep, nodeWorkingDirectory, model, runScriptName string, promptCommand AgentPromptCommand) (BrokerTaskFile, BrokerCommand) {
	stepSlug := AgentStepSlug(stepNumber, step.Name)
	workingDirectory := EffectiveWorkingDirectory(nodeWorkingDirectory, step.WorkingDirectory)
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
				Command: WrapAgentStepCommand(WrapCommandInWorkingDirectory(workingDirectory, fmt.Sprintf(`source "$SUPERPLANE_TASK_DIR/steps/%s"`, scriptName))),
				Kind:    LiveLogKindBash,
				Preview: LiveLogText(command),
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
				Command: WrapAgentStepCommand(WrapCommandInWorkingDirectory(workingDirectory, promptCommand(promptName, model))),
				Kind:    LiveLogKindPrompt,
				Preview: LiveLogText(prompt),
			}
	}
}

// EffectiveWorkingDirectory returns the per-step directory when set,
// otherwise the node working directory. Each broker command starts a new
// shell, so steps must cd even when they do not set a per-step directory.
func EffectiveWorkingDirectory(nodeDir, stepDir string) string {
	if dir := strings.TrimSpace(stepDir); dir != "" {
		return dir
	}
	return strings.TrimSpace(nodeDir)
}

// WrapCommandInWorkingDirectory prefixes command so it runs in dir.
// Relative dirs are resolved from the task launch directory recorded in
// prepare.sh, so a later `cd` in another step cannot nest or miss the clone.
func WrapCommandInWorkingDirectory(dir, command string) string {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return command
	}
	if filepath.IsAbs(dir) {
		return "cd " + ShellSingleQuote(dir) + " && " + command
	}
	return `_sp_root=$(cat "$SUPERPLANE_TASK_DIR/task_cwd")
cd "$_sp_root"/` + ShellSingleQuote(dir) + ` && ` + command
}

// WrapAgentStepCommand runs command, then merges accumulated LLM usage into
// SUPERPLANE_RESULT_FILE even when command exits non-zero.
func WrapAgentStepCommand(command string) string {
	return `_sp_status=0
{
` + command + `
} || _sp_status=$?
node "$SUPERPLANE_TASK_DIR/llm_usage.js" merge || true
if [ "$_sp_status" -ne 0 ]; then
  return "$_sp_status" 2>/dev/null || exit "$_sp_status"
fi`
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
	prepare += "pwd -P >\"$SUPERPLANE_TASK_DIR/task_cwd\"\n"
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
