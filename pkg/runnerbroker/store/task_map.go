package store

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/runnerbroker/models"
	brokermodels "github.com/superplanehq/superplane/pkg/runnerbroker/storemodels"
)

func taskRowFromModel(t *models.Task) (*brokermodels.Task, error) {
	cmd := t.Command
	if cmd == nil {
		cmd = []string{}
	}
	cmdJSON, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}
	runMode := string(t.RunMode)
	if runMode == "" {
		runMode = string(models.InferRunMode(t.Commands, t.Command, t.Script))
	}
	if runMode == "" {
		runMode = string(models.RunModeCommandList)
	}
	row := &brokermodels.Task{
		ID:                      t.ID,
		FleetID:                 t.FleetID,
		RunMode:                 runMode,
		CommandJSON:             string(cmdJSON),
		WebhookURL:              t.WebhookURL,
		Status:                  string(t.Status),
		CreatedAt:               t.CreatedAt,
		RunnerID:                t.RunnerID,
		ExecutionMode:           string(t.ExecutionMode),
		DockerImage:             t.DockerImage,
		Output:                  t.Output,
		ResultJSON:              t.ResultJSON,
		ErrorMessage:            t.ErrorMessage,
		InfraRetryCount:         t.InfraRetryCount,
		CancelRequested:         t.CancelRequested,
		ClaimedAt:               t.ClaimedAt,
		LeaseUntil:              t.LeaseUntil,
		ExecutionTimeoutSeconds: t.ExecutionTimeoutSeconds,
		ExitCode:                t.ExitCode,
	}
	if strings.TrimSpace(t.Script) != "" {
		row.ScriptJSON = t.Script
	}
	if mc := strings.TrimSpace(t.MessageChainJSON); mc != "" {
		row.MessageChainJSON = mc
	}
	if len(t.Commands) > 0 {
		b, err := json.Marshal(t.Commands)
		if err != nil {
			return nil, err
		}
		row.CommandsJSON = string(b)
	}
	if len(t.SetupCommands) > 0 {
		b, err := json.Marshal(t.SetupCommands)
		if err != nil {
			return nil, err
		}
		row.SetupCommandsJSON = string(b)
	}
	if len(t.Environment) > 0 {
		b, err := json.Marshal(t.Environment)
		if err != nil {
			return nil, err
		}
		row.EnvironmentJSON = string(b)
	}
	if row.ExecutionMode == "" {
		row.ExecutionMode = string(models.ExecutionHost)
	}
	return row, nil
}

func taskModelFromRow(row *brokermodels.Task) (*models.Task, error) {
	var cmd []string
	if err := json.Unmarshal([]byte(row.CommandJSON), &cmd); err != nil {
		return nil, fmt.Errorf("command_json: %w", err)
	}
	var cmds []string
	if strings.TrimSpace(row.CommandsJSON) != "" {
		if err := json.Unmarshal([]byte(row.CommandsJSON), &cmds); err != nil {
			return nil, fmt.Errorf("commands_json: %w", err)
		}
	}
	var setupCmds []string
	if strings.TrimSpace(row.SetupCommandsJSON) != "" {
		if err := json.Unmarshal([]byte(row.SetupCommandsJSON), &setupCmds); err != nil {
			return nil, fmt.Errorf("setup_commands_json: %w", err)
		}
	}
	var env []models.EnvironmentVariable
	if strings.TrimSpace(row.EnvironmentJSON) != "" {
		if err := json.Unmarshal([]byte(row.EnvironmentJSON), &env); err != nil {
			return nil, fmt.Errorf("environment_json: %w", err)
		}
	}
	t := &models.Task{
		ID:                      row.ID,
		FleetID:                 row.FleetID,
		RunMode:                 models.RunMode(row.RunMode),
		Script:                  row.ScriptJSON,
		MessageChainJSON:        row.MessageChainJSON,
		Command:                 cmd,
		Commands:                cmds,
		SetupCommands:           setupCmds,
		Environment:             env,
		WebhookURL:              row.WebhookURL,
		Status:                  models.TaskStatus(row.Status),
		CreatedAt:               row.CreatedAt.UTC(),
		RunnerID:                row.RunnerID,
		ExecutionMode:           models.ExecutionMode(row.ExecutionMode),
		DockerImage:             row.DockerImage,
		Output:                  row.Output,
		ResultJSON:              row.ResultJSON,
		ErrorMessage:            row.ErrorMessage,
		InfraRetryCount:         row.InfraRetryCount,
		CancelRequested:         row.CancelRequested,
		ClaimedAt:               row.ClaimedAt,
		LeaseUntil:              row.LeaseUntil,
		ExecutionTimeoutSeconds: row.ExecutionTimeoutSeconds,
		ExitCode:                row.ExitCode,
	}
	if t.RunMode == "" {
		t.RunMode = models.InferRunMode(t.Commands, t.Command, t.Script)
	}
	if t.ClaimedAt != nil {
		ct := t.ClaimedAt.UTC()
		t.ClaimedAt = &ct
	}
	if t.LeaseUntil != nil {
		lt := t.LeaseUntil.UTC()
		t.LeaseUntil = &lt
	}
	return t, nil
}
