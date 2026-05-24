package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/models"
	brokermodels "github.com/superplane/runner/task-broker/internal/models"
	"gorm.io/gorm"
)

func (s *PostgresStore) CreateTask(ctx context.Context, t *models.Task) error {
	row, err := taskRowFromModel(t)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Create(row).Error
}

func (s *PostgresStore) GetTask(ctx context.Context, id string) (*models.Task, error) {
	var row brokermodels.Task
	err := s.db.WithContext(ctx).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return taskModelFromRow(&row)
}

func (s *PostgresStore) ClaimTask(ctx context.Context, runnerID, fleetID string, lease time.Duration) (*models.Task, error) {
	fleetID = strings.TrimSpace(fleetID)
	if fleetID == "" {
		return nil, fmt.Errorf("fleet_id required for claim")
	}
	now := time.Now().UTC()
	runnerLeaseEnd := now.Add(lease)
	defaultExec := api.DefaultExecutionTimeoutSeconds
	buf := api.LeaseBufferSeconds

	var id string
	err := s.db.WithContext(ctx).Raw(`
UPDATE tasks SET
	status = ?,
	claimed_at = ?,
	lease_until = GREATEST(?::timestamptz, ?::timestamptz + (COALESCE(NULLIF(execution_timeout_seconds, 0), ?) + ?) * interval '1 second'),
	runner_id = ?
WHERE id = (
	SELECT id FROM tasks
	WHERE status = ? AND fleet_id = ?
	ORDER BY created_at ASC
	LIMIT 1
	FOR UPDATE SKIP LOCKED
)
RETURNING id`,
		string(models.StatusClaimed), now, runnerLeaseEnd, now, defaultExec, buf, runnerID,
		string(models.StatusQueued), fleetID,
	).Scan(&id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if id == "" {
		return nil, nil
	}
	return s.GetTask(ctx, id)
}

func terminalTaskStatus(st models.TaskStatus) bool {
	switch st {
	case models.StatusSucceeded, models.StatusFailed, models.StatusCanceled:
		return true
	default:
		return false
	}
}

const (
	msgCanceledQueued    = "canceled before execution"
	msgCanceledLeaseReap = "canceled (lease expired while stop pending)"
	exitCanceled         = 130
)

func (s *PostgresStore) RequestCancelTask(ctx context.Context, id string) (*models.Task, CancelOutcome, error) {
	res := s.db.WithContext(ctx).Exec(`
UPDATE tasks SET
	status = ?,
	exit_code = ?,
	output = ?,
	error_message = NULL,
	claimed_at = NULL,
	lease_until = NULL,
	runner_id = NULL,
	cancel_requested = false
WHERE id = ? AND status = ?`,
		string(models.StatusCanceled), exitCanceled, msgCanceledQueued, id, string(models.StatusQueued),
	)
	if res.Error != nil {
		return nil, "", res.Error
	}
	if res.RowsAffected == 1 {
		t, err := s.GetTask(ctx, id)
		if err != nil {
			return nil, "", err
		}
		return t, CancelOutcomeCanceledQueued, nil
	}

	res = s.db.WithContext(ctx).Exec(`
UPDATE tasks SET cancel_requested = true WHERE id = ? AND status = ?`,
		id, string(models.StatusClaimed),
	)
	if res.Error != nil {
		return nil, "", res.Error
	}
	if res.RowsAffected == 1 {
		t, err := s.GetTask(ctx, id)
		if err != nil {
			return nil, "", err
		}
		return t, CancelOutcomeCancelRequested, nil
	}

	t, err := s.GetTask(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if t == nil {
		return nil, CancelOutcomeNotFound, nil
	}
	if terminalTaskStatus(t.Status) {
		return t, CancelOutcomeAlreadyTerminal, nil
	}
	return nil, "", fmt.Errorf("cancel: task %s in unexpected state %s", id, t.Status)
}

func (s *PostgresStore) CompleteTask(ctx context.Context, id, runnerID string, exitCode int, resultJSON, errMsg string, canceled bool) (*models.Task, error) {
	final := models.StatusSucceeded
	if canceled {
		final = models.StatusCanceled
	} else if exitCode != 0 || errMsg != "" {
		final = models.StatusFailed
	}
	res := s.db.WithContext(ctx).Exec(`
UPDATE tasks SET
	status = ?,
	exit_code = ?,
	output = '',
	result_json = ?,
	error_message = ?,
	cancel_requested = false,
	environment_json = NULL
WHERE id = ? AND runner_id = ? AND status = ?`,
		string(final), exitCode, nullIfEmpty(resultJSON), nullIfEmpty(errMsg),
		id, runnerID, string(models.StatusClaimed),
	)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, fmt.Errorf("task not found, wrong runner, or not claimed: %s", id)
	}
	return s.GetTask(ctx, id)
}

func nullIfEmpty(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func (s *PostgresStore) ReapExpiredLeases(ctx context.Context) (int64, []*models.Task, error) {
	now := time.Now().UTC()

	type idRow struct{ ID string }
	var canceledIDs []idRow
	err := s.db.WithContext(ctx).Raw(`
UPDATE tasks SET
	status = ?,
	cancel_requested = false,
	claimed_at = NULL,
	lease_until = NULL,
	runner_id = NULL,
	exit_code = ?,
	output = ?,
	result_json = NULL,
	error_message = NULL
WHERE status = ? AND lease_until IS NOT NULL AND lease_until <= ? AND cancel_requested = true
RETURNING id`,
		string(models.StatusCanceled), exitCanceled, msgCanceledLeaseReap,
		string(models.StatusClaimed), now,
	).Scan(&canceledIDs).Error
	if err != nil {
		return 0, nil, err
	}

	var canceled []*models.Task
	for _, row := range canceledIDs {
		t, err := s.GetTask(ctx, row.ID)
		if err != nil {
			return 0, nil, err
		}
		canceled = append(canceled, t)
	}

	res := s.db.WithContext(ctx).Exec(`
UPDATE tasks SET
	status = ?,
	claimed_at = NULL,
	lease_until = NULL,
	runner_id = NULL
WHERE status = ? AND lease_until IS NOT NULL AND lease_until <= ? AND cancel_requested = false`,
		string(models.StatusQueued), string(models.StatusClaimed), now,
	)
	if res.Error != nil {
		return 0, canceled, res.Error
	}
	return res.RowsAffected, canceled, nil
}
