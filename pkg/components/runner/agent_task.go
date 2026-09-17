package runner

import (
	"embed"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/superplanehq/superplane/pkg/blob"
)

//go:embed fetch_task_attachments.sh process_video_attachments.sh
var attachmentSetupScripts embed.FS

type AgentPromptCommand func(promptName, model string) string

type AgentBrokerTaskInput struct {
	PrepareName      string
	PrepareScript    string
	RunScriptName    string
	RunScript        string
	WorkingDirectory string
	Steps            []AgentStep
	// DispatchedSteps, when set to the same length as Steps, supply minted
	// prompt/command text for task files and attachment fetches. Preview
	// text stays on Steps.
	DispatchedSteps []AgentStep
	Usage           string
	Setups          []IntegrationSetup
	Model           string
	PromptCommand   AgentPromptCommand
	Attachments     []TaskAttachment
}

type TaskAttachment struct {
	ID          string
	URL         string
	Filename    string
	ContentType string
	SizeBytes   int64
	Checksum    string
}

func BuildAgentBrokerTask(input AgentBrokerTaskInput) (commands []BrokerCommand, files []BrokerTaskFile) {
	files = []BrokerTaskFile{
		LLMUsageTaskFile(),
		TurnTelemetryTaskFile(),
		ActivityStreamTaskFile(),
		{Path: input.RunScriptName, Content: input.RunScript, Mode: "0644"},
		{Path: "prepare.sh", Content: input.PrepareScript, Mode: "0644"},
	}

	setupCommands, setupFiles := BuildIntegrationSetupCommands(input.Setups)
	files = append(files, setupFiles...)

	commands = make([]BrokerCommand, 0, len(input.Steps)+len(setupCommands)+1)
	commands = append(commands, BrokerCommand{
		Name:    input.PrepareName,
		Command: WithTaskBinOnPath(`source "$SUPERPLANE_TASK_DIR/prepare.sh"`),
		Kind:    LiveLogKindSetup,
	})
	attachmentFiles, attachmentCommands := AttachmentSetup(ResolveTaskAttachments(input))
	files = append(files, attachmentFiles...)
	commands = append(commands, attachmentCommands...)
	commands = append(commands, setupCommands...)

	for i, step := range input.Steps {
		file, command := buildAgentStep(i+1, step, AgentStepForDispatch(input.Steps, input.DispatchedSteps, i), input.WorkingDirectory, input.Usage, input.Model, input.PromptCommand)
		files = append(files, file)
		commands = append(commands, command)
	}
	return commands, files
}

func AgentStepsForDispatch(original, dispatched []AgentStep) []AgentStep {
	if len(dispatched) == len(original) {
		return dispatched
	}
	return original
}

func AgentStepForDispatch(original, dispatched []AgentStep, i int) AgentStep {
	if i >= 0 && i < len(dispatched) && len(dispatched) == len(original) {
		return dispatched[i]
	}
	return original[i]
}

func ApplyIntegrationUsage(prompt, usage string) string {
	usage = strings.TrimSpace(usage)
	if usage == "" {
		return prompt
	}
	if strings.TrimSpace(prompt) == "" {
		return usage
	}
	return usage + "\n\n" + prompt
}

func WithTaskBinOnPath(command string) string {
	return `export PATH="$SUPERPLANE_TASK_DIR/bin:$PATH"
` + command
}

func BuildIntegrationSetupCommands(setups []IntegrationSetup) (commands []BrokerCommand, files []BrokerTaskFile) {
	for i, setup := range setups {
		if strings.TrimSpace(setup.Script) == "" {
			continue
		}
		name := strings.TrimSpace(setup.Name)
		if name == "" {
			name = "Set up integration"
		}
		scriptName := AgentStepSlug(i+1, name) + ".sh"
		path := "setup/" + scriptName
		files = append(files, BrokerTaskFile{
			Path:    path,
			Content: setup.Script,
			Mode:    "0644",
		})
		commands = append(commands, BrokerCommand{
			Name:    name,
			Command: WrapAgentStepCommand(fmt.Sprintf(`source "$SUPERPLANE_TASK_DIR/%s"`, path)),
			Kind:    LiveLogKindSetup,
			Preview: LiveLogText(name),
		})
	}
	return commands, files
}

func buildAgentStep(stepNumber int, original, dispatched AgentStep, nodeWorkingDirectory, usage, model string, promptCommand AgentPromptCommand) (BrokerTaskFile, BrokerCommand) {
	stepSlug := AgentStepSlug(stepNumber, original.Name)
	workingDirectory := EffectiveWorkingDirectory(nodeWorkingDirectory, original.WorkingDirectory)
	switch NormalizeAgentStepType(original.Type) {
	case AgentStepBash:
		command := stringPtrValue(original.Command)
		scriptName := stepSlug + ".sh"
		return BrokerTaskFile{
				Path:    "steps/" + scriptName,
				Content: stringPtrValue(dispatched.Command),
				Mode:    "0644",
			}, BrokerCommand{
				Name:    AgentStepLabel(original.Name, scriptName),
				Command: WrapAgentStepCommand(WrapCommandInWorkingDirectory(workingDirectory, fmt.Sprintf(`source "$SUPERPLANE_TASK_DIR/steps/%s"`, scriptName))),
				Kind:    LiveLogKindBash,
				Preview: LiveLogText(command),
			}
	default:
		prompt := stringPtrValue(original.Prompt)
		promptName := stepSlug + ".txt"
		return BrokerTaskFile{
				Path:    "prompts/" + promptName,
				Content: ApplyAttachmentInstructions(ApplyIntegrationUsage(stringPtrValue(dispatched.Prompt), usage)),
				Mode:    "0644",
			}, BrokerCommand{
				Name:    AgentStepLabel(original.Name, promptName),
				Command: WrapAgentStepCommand(WrapCommandInWorkingDirectory(workingDirectory, promptCommand(promptName, model))),
				Kind:    LiveLogKindPrompt,
				Preview: LiveLogText(prompt),
			}
	}
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
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
_sp_merge_llm_usage() {
  node "$SUPERPLANE_TASK_DIR/llm_usage.js" merge || true
}
trap '_sp_merge_llm_usage' EXIT
trap 'exit 143' TERM
trap 'exit 130' INT
{
` + WithTaskBinOnPath(command) + `
} || _sp_status=$?
_sp_merge_llm_usage
trap - EXIT TERM INT
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

func CollectTaskAttachmentsFromSteps(steps []AgentStep) []TaskAttachment {
	texts := make([]string, 0, len(steps)*2)
	for _, step := range steps {
		if step.Prompt != nil {
			texts = append(texts, *step.Prompt)
		}
		if step.Command != nil {
			texts = append(texts, *step.Command)
		}
	}
	return CollectTaskAttachments(texts...)
}

func CollectTaskAttachments(texts ...string) []TaskAttachment {
	seen := map[string]struct{}{}
	var attachments []TaskAttachment
	for _, text := range texts {
		for _, raw := range blob.SignedFileURLs(text) {
			if _, exists := seen[raw]; exists {
				continue
			}
			seen[raw] = struct{}{}
			attachments = append(attachments, TaskAttachment{
				URL:      raw,
				Filename: scrapeAttachmentFilename(raw),
			})
		}
	}
	return attachments
}

func AttachmentSetup(attachments []TaskAttachment) (files []BrokerTaskFile, commands []BrokerCommand) {
	if len(attachments) == 0 {
		return nil, nil
	}
	files = append(AttachmentSetupFiles(), BrokerTaskFile{
		Path:    AttachmentManifestPath,
		Content: AttachmentManifestJSON(attachments),
		Mode:    "0644",
	})
	if fetch := AttachmentFetchCommand(attachments); fetch != nil {
		commands = append(commands, *fetch)
	}
	if HasVideoAttachment(attachments) {
		commands = append(commands, VideoAttachmentCommand())
	}
	return files, commands
}

func AttachmentFetchCommand(attachments []TaskAttachment) *BrokerCommand {
	if len(attachments) == 0 {
		return nil
	}
	return &BrokerCommand{
		Name:    "Fetch task attachments",
		Command: WithTaskBinOnPath(`bash "$SUPERPLANE_TASK_DIR/fetch_task_attachments.sh"`),
		Kind:    LiveLogKindSetup,
		Preview: LiveLogText("Download task files"),
	}
}

func AttachmentSetupFiles() []BrokerTaskFile {
	fetch, err := attachmentSetupScripts.ReadFile("fetch_task_attachments.sh")
	if err != nil {
		panic(err)
	}
	process, err := attachmentSetupScripts.ReadFile("process_video_attachments.sh")
	if err != nil {
		panic(err)
	}
	return []BrokerTaskFile{
		{Path: AttachmentFetchScriptPath, Content: string(fetch), Mode: "0755"},
		{Path: AttachmentProcessScriptPath, Content: string(process), Mode: "0755"},
	}
}

func AppendAttachmentSetupFiles(files []BrokerTaskFile) []BrokerTaskFile {
	for _, file := range files {
		if file.Path == AttachmentFetchScriptPath {
			return files
		}
	}
	return append(files, AttachmentSetupFiles()...)
}

func VideoAttachmentCommand() BrokerCommand {
	return BrokerCommand{
		Name:    "Process video attachments",
		Command: WithTaskBinOnPath(`bash "$SUPERPLANE_TASK_DIR/process_video_attachments.sh"`),
		Kind:    LiveLogKindSetup,
		Preview: LiveLogText("Extract still frames and transcribe task videos"),
	}
}

func VideoAttachmentCommands() []BrokerCommand {
	return []BrokerCommand{VideoAttachmentCommand()}
}

func scrapeAttachmentFilename(raw string) string {
	parsed, err := url.Parse(raw)
	base := "file"
	if err == nil {
		if name := path.Base(parsed.Path); name != "" && name != "." && name != "/" {
			base = name
		}
	}
	return sanitizeAttachmentName(base)
}

func attachmentFilename(raw string, index int) string {
	return fmt.Sprintf("%02d-%s", index, scrapeAttachmentFilename(raw))
}
