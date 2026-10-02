package muse

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/components/runner"
)

type RunMuseSpec struct {
	MachineType             string                        `mapstructure:"machineType"`
	Steps                   []runner.AgentStep            `mapstructure:"steps"`
	Credentials             runner.AgentCredentials       `mapstructure:"credentials"`
	Model                   string                        `mapstructure:"model"`
	ThinkingLevel           string                        `mapstructure:"thinkingLevel"`
	WorkingDirectory        string                        `mapstructure:"workingDirectory"`
	EnvironmentFrom         []runner.EnvironmentFromEntry `mapstructure:"environmentFrom"`
	Environment             []runner.EnvironmentVariable  `mapstructure:"environment"`
	ExecutionTimeoutSeconds int                           `mapstructure:"executionTimeoutSeconds"`
	IncludeVisualEvidence   bool                          `mapstructure:"includeVisualEvidence"`
}

func decodeRunMuseSpec(raw any) (RunMuseSpec, error) {
	var spec RunMuseSpec
	dec, err := runner.NewSpecDecoder(&spec)
	if err != nil {
		return RunMuseSpec{}, fmt.Errorf("runnerMuse spec decoder: %w", err)
	}
	if err := dec.Decode(raw); err != nil {
		return RunMuseSpec{}, fmt.Errorf("decode runnerMuse configuration: %w", err)
	}
	if spec.ExecutionTimeoutSeconds <= 0 {
		spec.ExecutionTimeoutSeconds = runner.DefaultExecutionTimeoutSeconds
	}
	if thinking, err := runner.NormalizeThinkingLevel(spec.ThinkingLevel); err == nil {
		spec.ThinkingLevel = thinking
	}
	return spec, nil
}

func validateRunMuseSpec(spec RunMuseSpec) error {
	if strings.TrimSpace(spec.MachineType) == "" {
		return fmt.Errorf("machine type is required")
	}
	if err := runner.ValidateAgentSteps(spec.Steps); err != nil {
		return err
	}
	if err := runner.RejectHostedCredentials(spec.Credentials); err != nil {
		return err
	}
	if err := runner.ValidateAgentCredentials(spec.Credentials, true); err != nil {
		return err
	}
	if err := runner.ValidateEnvironmentFrom(spec.EnvironmentFrom); err != nil {
		return err
	}
	if err := runner.ValidateEnvironment(spec.Environment); err != nil {
		return err
	}
	if err := runner.ValidateReservedEnvironmentNames(spec.Environment, envMetaAPIKey, envCustomLLMAPIKey); err != nil {
		return err
	}
	if spec.ExecutionTimeoutSeconds != 0 {
		if spec.ExecutionTimeoutSeconds < 1 || spec.ExecutionTimeoutSeconds > runner.MaxExecutionTimeoutSecondsRequest {
			return fmt.Errorf("execution timeout must be between 1 and %d seconds, or 0 to use the default (%d seconds)", runner.MaxExecutionTimeoutSecondsRequest, runner.DefaultExecutionTimeoutSeconds)
		}
	}
	_, err := runner.NormalizeThinkingLevel(spec.ThinkingLevel)
	return err
}

type MuseBrokerTask struct {
	Commands []runner.BrokerCommand
	Files    []runner.BrokerTaskFile
}

func buildMuseBrokerTask(spec RunMuseSpec, usage string, setups []runner.IntegrationSetup, dispatched []runner.AgentStep, attachments []runner.TaskAttachment, inspectImages bool) MuseBrokerTask {
	commands, files := runner.BuildAgentBrokerTask(runner.AgentBrokerTaskInput{
		PrepareName:      "Prepare Muse Code",
		PrepareScript:    runner.NodePrepareScript("muse", "Muse Code CLI not found on PATH; install Muse Code on the runner", spec.WorkingDirectory),
		RunScriptName:    "run.js",
		RunScript:        runScript,
		WorkingDirectory: spec.WorkingDirectory,
		Steps:            spec.Steps,
		DispatchedSteps:  dispatched,
		Attachments:      attachments,
		InspectImages:    inspectImages,
		Usage:            usage,
		Setups:           setups,
		Model:            strings.TrimSpace(spec.Model),
		PromptCommand: func(promptName, model string) string {
			return runner.PromptNodeCommand(promptName, model, spec.ThinkingLevel)
		},
	})
	return MuseBrokerTask{Commands: commands, Files: files}
}

func BuildBrokerTask(spec RunMuseSpec, usage string, setups []runner.IntegrationSetup) MuseBrokerTask {
	return buildMuseBrokerTask(spec, usage, setups, nil, nil, false)
}

func applyPlanningFollowUp(task MuseBrokerTask, environment []runner.BrokerEnvironmentVariable, spec RunMuseSpec) MuseBrokerTask {
	if !runner.HasPlanningSessionToken(environment) {
		return task
	}
	task.Files = runner.AppendAttachmentSetupFiles(runner.AppendAttachmentLimitFile(append(task.Files, runner.FollowUpLoopFile())))
	task.Commands = append(task.Commands, planningFollowUpCommand(spec))
	return task
}

func planningFollowUpCommand(spec RunMuseSpec) runner.BrokerCommand {
	workdir := planningFollowUpWorkingDirectory(spec)
	model := strings.TrimSpace(spec.Model)
	return runner.BrokerCommand{
		Name: "Wait for the next message",
		Command: runner.WrapAgentStepCommand(
			runner.WrapPromptCommandInWorkingDirectory(
				workdir,
				runner.FollowUpLoopCommand(model, spec.ThinkingLevel),
			),
		),
		Kind:    runner.LiveLogKindPrompt,
		Preview: "Wait for the next user message",
	}
}

func planningFollowUpWorkingDirectory(spec RunMuseSpec) string {
	for i := len(spec.Steps) - 1; i >= 0; i-- {
		if runner.NormalizeAgentStepType(spec.Steps[i].Type) == runner.AgentStepPrompt {
			return runner.EffectiveWorkingDirectory(spec.WorkingDirectory, spec.Steps[i].WorkingDirectory)
		}
	}
	return strings.TrimSpace(spec.WorkingDirectory)
}
