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
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	runnercontrol "github.com/superplanehq/superplane/pkg/runners/control"
	"gorm.io/gorm"
)

type RunnerTaskContext struct {
	tx             *gorm.DB
	encryptor      crypto.Encryptor
	organizationID uuid.UUID
}

func NewRunnerTaskContext(
	tx *gorm.DB,
	encryptor crypto.Encryptor,
	organizationID uuid.UUID,
) *RunnerTaskContext {
	return &RunnerTaskContext{
		tx:             tx,
		encryptor:      encryptor,
		organizationID: organizationID,
	}
}

func (c *RunnerTaskContext) IntegratedBackendEnabled() (bool, error) {
	organization, err := models.FindOrganizationByIDInTransaction(
		c.tx,
		c.organizationID.String(),
	)
	if err != nil {
		return false, err
	}
	return organization.HasExperimentalFeature(features.FeatureNewRunners), nil
}

func (c *RunnerTaskContext) Create(rawID, fleetID string, payload []byte) error {
	if c.encryptor == nil {
		return errors.New("runner task encryptor is required")
	}
	id, err := uuid.Parse(rawID)
	if err != nil {
		return fmt.Errorf("invalid runner task ID: %w", err)
	}
	if !json.Valid(payload) {
		return errors.New("runner task payload must be valid JSON")
	}
	fleet, err := models.FindInstallationRunnerFleet(c.tx, fleetID)
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
	_ = runnercontrol.Publish(runnercontrol.Notification{FleetID: fleet.Slug})
	return nil
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
	if task.RunnerID != nil {
		_ = runnercontrol.Publish(runnercontrol.Notification{
			RunnerID: task.RunnerID.String(),
		})
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
