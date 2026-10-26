package contexts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/encryption"
	"github.com/superplanehq/superplane/pkg/models"
	runnerapi "github.com/superplanehq/superplane/pkg/runners/api"
	"gorm.io/gorm"
)

type RunnerTaskContext struct {
	tx             *gorm.DB
	encryptor      crypto.Encryptor
	organizationID uuid.UUID
	runID          uuid.UUID
	secrets        *SecretsContext
	refreshes      []runnerapi.CredentialRefresh
	onChanged      func(organizationID uuid.UUID, runnerID *uuid.UUID)
}

func (c *RunnerTaskContext) WithRunScope(runID uuid.UUID) *RunnerTaskContext {
	c.runID = runID
	return c
}

// WithOnChanged records mutations for publication after the owning transaction commits.
func (c *RunnerTaskContext) WithOnChanged(fn func(organizationID uuid.UUID, runnerID *uuid.UUID)) *RunnerTaskContext {
	c.onChanged = fn
	return c
}

func (c *RunnerTaskContext) WithSecrets(secrets *SecretsContext) *RunnerTaskContext {
	c.secrets = secrets
	if secrets != nil {
		secrets.deferRunnerCredentials = true
	}
	return c
}

func (c *RunnerTaskContext) RecordCredentialRefresh(descriptor runnerapi.CredentialRefresh) {
	c.refreshes = append(c.refreshes, descriptor)
}

func NewRunnerTaskContext(
	tx *gorm.DB,
	encryptor crypto.Encryptor,
	organizationID uuid.UUID,
) *RunnerTaskContext {
	return &RunnerTaskContext{
		tx:             tx,
		encryptor:      encryption.ForDB(tx, encryptor),
		organizationID: organizationID,
	}
}

func (c *RunnerTaskContext) Create(rawID, fleetID string, payload []byte) error {
	if c.encryptor == nil {
		return errors.New("runner task encryptor is required")
	}
	id, err := uuid.Parse(rawID)
	if err != nil {
		return fmt.Errorf("invalid runner task ID: %w", err)
	}
	payload, err = withDefaultLogUploadPolicy(payload)
	if err != nil {
		return err
	}
	payload, err = c.withCredentialRefreshes(payload)
	if err != nil {
		return err
	}
	fleet, err := models.FindEnabledRunnerFleetForOrganization(c.tx, c.organizationID, fleetID)
	if err != nil {
		return err
	}
	encrypted, err := c.encryptor.Encrypt(
		context.Background(),
		payload,
		[]byte(id.String()),
	)
	if err != nil {
		return fmt.Errorf("encrypt runner task payload: %w", err)
	}

	now := time.Now()
	task := &models.RunnerTask{
		ID:                id,
		OrganizationID:    c.organizationID,
		FleetID:           fleet.ID,
		Backend:           models.RunnerTaskBackendIntegrated,
		State:             models.RunnerTaskStateQueued,
		PayloadCiphertext: encrypted,
		QueuedAt:          now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := c.tx.Create(task).Error; err != nil {
		return err
	}
	if c.onChanged != nil {
		c.onChanged(c.organizationID, nil)
	}
	return nil
}

func withDefaultLogUploadPolicy(payload []byte) ([]byte, error) {
	var task runnerapi.TaskPayload
	if err := json.Unmarshal(payload, &task); err != nil {
		return nil, fmt.Errorf("runner task payload must be valid JSON: %w", err)
	}
	policy := runnerapi.DefaultLogUploadPolicy()
	task.LogUploadPolicy = &policy
	payload, err := json.Marshal(task)
	if err != nil {
		return nil, fmt.Errorf("marshal runner task payload: %w", err)
	}
	return payload, nil
}

func (c *RunnerTaskContext) Find(rawID string) (*core.RunnerTask, error) {
	task, err := c.find(rawID)
	if err != nil {
		return nil, err
	}

	var exitCode *int
	if task.ExitCode != nil {
		value := int(*task.ExitCode)
		exitCode = &value
	}
	errorMessage := ""
	if task.ErrorMessage != nil {
		errorMessage = *task.ErrorMessage
	}
	result := json.RawMessage(nil)
	if len(task.Result) > 0 {
		result = append(result, task.Result...)
	}
	return &core.RunnerTask{
		ID:           task.ID.String(),
		State:        task.State,
		Result:       result,
		ExitCode:     exitCode,
		ErrorMessage: errorMessage,
		ReservedAt:   task.ReservedAt,
		StartedAt:    task.StartedAt,
		FinishedAt:   task.FinishedAt,
	}, nil
}

func (c *RunnerTaskContext) RequestCancel(rawID string) error {
	task, err := c.find(rawID)
	if err != nil {
		return err
	}
	if err := task.RequestCancel(c.tx, time.Now()); err != nil {
		return err
	}
	if c.onChanged != nil {
		c.onChanged(c.organizationID, task.RunnerID)
	}

	return nil
}

func (c *RunnerTaskContext) find(rawID string) (*models.RunnerTask, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, fmt.Errorf("invalid runner task ID: %w", err)
	}
	task, err := models.FindRunnerTask(c.tx, id)
	if err != nil {
		return nil, err
	}
	if task.OrganizationID != c.organizationID {
		return nil, models.ErrRunnerTaskNotFound
	}
	return task, nil
}

var _ core.RunnerTaskContext = (*RunnerTaskContext)(nil)
