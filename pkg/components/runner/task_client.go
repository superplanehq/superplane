package runner

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	runnerapi "github.com/superplanehq/superplane/pkg/runners/api"
	runnermodels "github.com/superplanehq/superplane/pkg/runners/models"
)

type TaskClient interface {
	CreateTask(CreateTaskParams) (string, error)
	FetchTaskStatus(string) (*Task, error)
	CancelTask(string) error
}

func NewTaskClient(
	httpClient core.HTTPContext,
	runnerTasks core.RunnerTaskContext,
) (TaskClient, string, error) {
	if runnerTasks != nil {
		enabled, err := runnerTasks.IntegratedBackendEnabled()
		if err != nil {
			return nil, "", fmt.Errorf("check integrated runner feature: %w", err)
		}
		if enabled {
			return &integratedTaskClient{tasks: runnerTasks},
				core.RunnerTaskBackendIntegrated, nil
		}
	}
	client, err := NewBrokerClient(httpClient)
	return client, core.RunnerTaskBackendLegacy, err
}

func taskClientForBackend(
	backend string,
	httpClient core.HTTPContext,
	runnerTasks core.RunnerTaskContext,
) (TaskClient, error) {
	if backend == core.RunnerTaskBackendIntegrated {
		if runnerTasks == nil {
			return nil, fmt.Errorf("integrated runner task context is unavailable")
		}
		return &integratedTaskClient{tasks: runnerTasks}, nil
	}
	return NewBrokerClient(httpClient)
}

type integratedTaskClient struct {
	tasks core.RunnerTaskContext
}

func (c *integratedTaskClient) CreateTask(params CreateTaskParams) (string, error) {
	fleetID, err := requireMachineType(params.MachineType)
	if err != nil {
		return "", err
	}
	id := uuid.New()
	timeout := params.TimeoutSeconds
	if timeout <= 0 {
		timeout = DefaultExecutionTimeoutSeconds
	}
	mode := strings.ToLower(strings.TrimSpace(params.ExecutionMode))
	if mode == "" {
		mode = ExecutionModeHost
	}

	payload := runnerapi.TaskPayload{
		ID:                      id.String(),
		RunMode:                 strings.TrimSpace(params.RunMode),
		Script:                  strings.TrimSpace(params.Script),
		MessageChain:            params.MessageChain,
		Commands:                integratedCommands(params.Commands),
		SetupCommands:           params.SetupCommands,
		Environment:             integratedEnvironment(params.Environment),
		Files:                   integratedFiles(params.Files),
		ExecutionMode:           mode,
		DockerImage:             strings.TrimSpace(params.DockerImage),
		ExecutionTimeoutSeconds: &timeout,
		WebhookPayloadSizeLimit: params.WebhookPayloadSizeLimit,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal integrated runner task: %w", err)
	}
	if err := c.tasks.Create(id.String(), fleetID, encoded); err != nil {
		return "", fmt.Errorf("create integrated runner task: %w", err)
	}
	return id.String(), nil
}

func (c *integratedTaskClient) FetchTaskStatus(id string) (*Task, error) {
	task, err := c.tasks.Find(id)
	if err != nil {
		return nil, err
	}
	status := task.State
	if status == "lost" {
		status = "failed"
		if task.ErrorMessage == "" {
			task.ErrorMessage = "runner task was lost"
		}
	}
	claimedAt := task.StartedAt
	if claimedAt == nil {
		claimedAt = task.ReservedAt
	}
	return &Task{
		ID:         task.ID,
		Status:     status,
		ExitCode:   task.ExitCode,
		Error:      task.ErrorMessage,
		Result:     append(json.RawMessage(nil), task.Result...),
		ClaimedAt:  claimedAt,
		FinishedAt: task.FinishedAt,
	}, nil
}

func (c *integratedTaskClient) CancelTask(id string) error {
	return c.tasks.RequestCancel(id)
}

func integratedCommands(commands []BrokerCommand) runnermodels.CommandList {
	out := make(runnermodels.CommandList, 0, len(commands))
	for _, command := range commands {
		out = append(out, runnermodels.CommandSpec{
			Name:    command.Name,
			Command: command.Command,
			Kind:    command.Kind,
			Preview: command.Preview,
		})
	}
	return out
}

func integratedEnvironment(
	environment []BrokerEnvironmentVariable,
) []runnerapi.EnvironmentVariable {
	out := make([]runnerapi.EnvironmentVariable, 0, len(environment))
	for _, variable := range environment {
		out = append(out, runnerapi.EnvironmentVariable{
			Name:  variable.Name,
			Value: variable.Value,
		})
	}
	return out
}

func integratedFiles(files []BrokerTaskFile) []runnerapi.TaskFile {
	out := make([]runnerapi.TaskFile, 0, len(files))
	for _, file := range files {
		out = append(out, runnerapi.TaskFile{
			Path:    file.Path,
			Content: file.Content,
			Mode:    file.Mode,
		})
	}
	return out
}

var _ TaskClient = (*BrokerClient)(nil)
var _ TaskClient = (*integratedTaskClient)(nil)
