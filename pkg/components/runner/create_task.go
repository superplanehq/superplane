package runner

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	runnerapi "github.com/superplanehq/superplane/pkg/runners/api"
	runnermodels "github.com/superplanehq/superplane/pkg/runners/models"
)

func CreateTask(tasks core.RunnerTaskContext, params CreateTaskParams) (string, error) {
	if tasks == nil {
		return "", fmt.Errorf("runner task context is unavailable")
	}
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

	limit := params.WebhookPayloadSizeLimit
	if limit <= 0 {
		limit = config.MaxWebhookPayloadSize
	}
	payload := runnerapi.TaskPayload{
		ID:                      id.String(),
		RunMode:                 strings.TrimSpace(params.RunMode),
		Script:                  strings.TrimSpace(params.Script),
		MessageChain:            params.MessageChain,
		Commands:                runnermodels.CommandList(params.Commands),
		SetupCommands:           params.SetupCommands,
		Environment:             params.Environment,
		Files:                   params.Files,
		ExecutionMode:           mode,
		DockerImage:             strings.TrimSpace(params.DockerImage),
		ExecutionTimeoutSeconds: &timeout,
		WebhookPayloadSizeLimit: limit,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal integrated runner task: %w", err)
	}
	if err := tasks.Create(id.String(), fleetID, encoded); err != nil {
		return "", fmt.Errorf("create integrated runner task: %w", err)
	}
	return id.String(), nil
}
