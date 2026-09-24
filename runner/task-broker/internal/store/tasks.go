package store

import (
	"context"
	"database/sql"
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

func (s *PostgresStore) ListActiveTasks(ctx context.Context) ([]*models.Task, error) {
	var rows []brokermodels.Task
	err := s.db.WithContext(ctx).
		Where("status IN ?", []string{string(models.StatusQueued), string(models.StatusClaimed)}).
		Order("created_at ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return tasksFromRows(rows)
}

const tasksByRunnerIDLimit = 5

func (s *PostgresStore) TasksByRunnerID(ctx context.Context, runnerID string) ([]*models.Task, error) {
	runnerID = strings.TrimSpace(runnerID)
	if runnerID == "" {
		return nil, fmt.Errorf("runner_id required")
	}
	var rows []brokermodels.Task
	err := s.db.WithContext(ctx).
		Where("runner_id = ?", runnerID).
		Order("created_at DESC, id DESC").
		Limit(tasksByRunnerIDLimit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return tasksFromRows(rows)
}

func (s *PostgresStore) CountTasksByFleet(ctx context.Context, fleetID string) (int, int, error) {
	fleetID = strings.TrimSpace(fleetID)
	if fleetID == "" {
		return 0, 0, fmt.Errorf("fleet_id required for count")
	}
	db := s.db.WithContext(ctx)
	var queued int64
	if err := db.Model(&brokermodels.Task{}).
		Where("fleet_id = ? AND status = ?", fleetID, string(models.StatusQueued)).
		Count(&queued).Error; err != nil {
		return 0, 0, err
	}
	var claimed int64
	if err := db.Model(&brokermodels.Task{}).
		Where("fleet_id = ? AND status = ?", fleetID, string(models.StatusClaimed)).
		Count(&claimed).Error; err != nil {
		return 0, 0, err
	}
	return int(queued), int(claimed), nil
}

func (s *PostgresStore) OldestQueuedTaskCreatedAt(ctx context.Context, fleetID string) (*time.Time, error) {
	fleetID = strings.TrimSpace(fleetID)
	if fleetID == "" {
		return nil, fmt.Errorf("fleet_id required for oldest queued task")
	}
	var oldest sql.NullTime
	err := s.db.WithContext(ctx).
		Model(&brokermodels.Task{}).
		Select("MIN(created_at)").
		Where("fleet_id = ? AND status = ?", fleetID, string(models.StatusQueued)).
		Scan(&oldest).Error
	if err != nil {
		return nil, err
	}
	if !oldest.Valid {
		return nil, nil
	}
	t := oldest.Time.UTC()
	return &t, nil
}

func (s *PostgresStore) ClaimedRunnerIDsByFleet(ctx context.Context, fleetID string) ([]string, error) {
	fleetID = strings.TrimSpace(fleetID)
	if fleetID == "" {
		return nil, fmt.Errorf("fleet_id required for claimed runner ids")
	}
	var runnerIDs []string
	err := s.db.WithContext(ctx).
		Model(&brokermodels.Task{}).
		Distinct("runner_id").
		Where("fleet_id = ? AND status = ? AND runner_id <> ''", fleetID, string(models.StatusClaimed)).
		Order("runner_id ASC").
		Pluck("runner_id", &runnerIDs).Error
	if err != nil {
		return nil, err
	}
	return runnerIDs, nil
}

func (s *PostgresStore) ClaimedTaskIDsByRunners(ctx context.Context, fleetID string, runnerIDs []string) (map[string]string, error) {
	fleetID = strings.TrimSpace(fleetID)
	if fleetID == "" {
		return nil, fmt.Errorf("fleet_id required for claimed task ids")
	}
	runnerIDs = compactRunnerIDs(runnerIDs)
	if len(runnerIDs) == 0 {
		return map[string]string{}, nil
	}

	var rows []struct {
		RunnerID string
		TaskID   string
	}
	err := s.db.WithContext(ctx).
		Model(&brokermodels.Task{}).
		Select("runner_id, id AS task_id").
		Where("fleet_id = ? AND status = ? AND runner_id IN ?", fleetID, string(models.StatusClaimed), runnerIDs).
		Order("runner_id ASC, created_at ASC, id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make(map[string]string, len(rows))
	for _, row := range rows {
		if _, ok := out[row.RunnerID]; ok {
			continue
		}
		out[row.RunnerID] = row.TaskID
	}
	return out, nil
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

func (s *PostgresStore) ClaimDispatchCandidates(ctx context.Context, provisioner string, staleAfter time.Duration, limit int) ([]DispatchCandidate, error) {
	provisioner = strings.TrimSpace(provisioner)
	if provisioner == "" {
		return nil, fmt.Errorf("provisioner required for dispatch candidates")
	}
	if limit <= 0 {
		limit = 25
	}
	now := time.Now().UTC()
	staleBefore := now.Add(-staleAfter)

	var rows []DispatchCandidate
	err := s.db.WithContext(ctx).Raw(`
WITH candidates AS (
	SELECT t.id, t.fleet_id, f.provisioner, f.dispatch_target
	FROM tasks t
	JOIN fleets f ON f.id = t.fleet_id
	WHERE t.status = ?
	  AND f.provisioner = ?
	  AND (t.dispatch_requested_at IS NULL OR t.dispatch_requested_at < ?)
	ORDER BY t.created_at ASC
	LIMIT ?
	FOR UPDATE OF t SKIP LOCKED
),
updated AS (
	UPDATE tasks t SET dispatch_requested_at = ?
	FROM candidates c
	WHERE t.id = c.id
	RETURNING t.id AS task_id, c.fleet_id, c.provisioner, c.dispatch_target
)
SELECT task_id, fleet_id, provisioner, dispatch_target FROM updated ORDER BY task_id ASC`,
		string(models.StatusQueued), provisioner, staleBefore, limit, now,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func compactRunnerIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// UnclaimTask re-queues a claimed task so another runner can pick it up.
// Returns unclaimed=false when the task is not claimed by runnerID.
func (s *PostgresStore) UnclaimTask(ctx context.Context, taskID, runnerID string) (bool, error) {
	res := s.db.WithContext(ctx).Exec(`
UPDATE tasks SET
	status      = ?,
	claimed_at  = NULL,
	lease_until = NULL,
	runner_id   = NULL
WHERE id = ? AND status = ? AND runner_id = ? AND runner_termination_requested_at IS NULL`,
		string(models.StatusQueued), taskID, string(models.StatusClaimed), runnerID,
	)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
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
	msgCanceledQueued     = "canceled before execution"
	msgCanceledLeaseReap  = "canceled (lease expired while stop pending)"
	msgCanceledRunnerLost = "canceled (runner lost while stop pending)"
	msgRunnerLost         = "runner lost before completion"
	exitCanceled          = 130
	maxInfraRetries       = 1
)

func (s *PostgresStore) RequestCancelTask(ctx context.Context, id string) (*models.Task, CancelOutcome, error) {
	now := time.Now().UTC()
	res := s.db.WithContext(ctx).Exec(`
UPDATE tasks SET
	status = ?,
	exit_code = ?,
	output = ?,
	error_message = NULL,
	claimed_at = NULL,
	lease_until = NULL,
	runner_id = NULL,
	cancel_requested = false,
	finished_at = ?
WHERE id = ? AND status = ?`,
		string(models.StatusCanceled), exitCanceled, msgCanceledQueued, now, id, string(models.StatusQueued),
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

func (s *PostgresStore) CompleteTask(ctx context.Context, req CompleteTaskRequest) (*CompleteTaskResult, error) {
	if isRetryableInfraFailure(req) {
		requeued, err := s.requeueInfraFailure(ctx, req)
		if err != nil {
			return nil, err
		}
		if requeued != nil {
			return &CompleteTaskResult{Task: requeued, Outcome: CompleteTaskOutcomeRequeued}, nil
		}
	}

	final := models.StatusSucceeded
	if req.Canceled {
		final = models.StatusCanceled
	} else if req.ExitCode != 0 || req.ErrorMessage != "" {
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
	environment_json = NULL,
	runner_termination_requested_at = NULL,
	finished_at = ?
WHERE id = ? AND runner_id = ? AND status = ?`,
		string(final), req.ExitCode, nullIfEmpty(req.ResultJSON), nullIfEmpty(req.ErrorMessage), time.Now().UTC(),
		req.ID, req.RunnerID, string(models.StatusClaimed),
	)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return s.completeTaskNoRows(ctx, req)
	}
	task, err := s.GetTask(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return &CompleteTaskResult{Task: task, Outcome: CompleteTaskOutcomeTerminal}, nil
}

func (s *PostgresStore) completeTaskNoRows(ctx context.Context, req CompleteTaskRequest) (*CompleteTaskResult, error) {
	task, err := s.GetTask(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("task not found: %s", req.ID)
	}
	if task.RunnerID != req.RunnerID {
		return nil, fmt.Errorf("wrong runner for task %s", req.ID)
	}
	if terminalTaskStatus(task.Status) {
		return &CompleteTaskResult{Task: task, Outcome: CompleteTaskOutcomeAlreadyTerminal}, nil
	}
	return nil, fmt.Errorf("task not claimed: %s", req.ID)
}

func isRetryableInfraFailure(req CompleteTaskRequest) bool {
	return strings.TrimSpace(req.FailureKind) == api.FailureKindRunnerInfra && !req.Canceled
}

func (s *PostgresStore) requeueInfraFailure(ctx context.Context, req CompleteTaskRequest) (*models.Task, error) {
	type idRow struct{ ID string }
	var rows []idRow
	err := s.db.WithContext(ctx).Raw(`
UPDATE tasks SET
	status = ?,
	claimed_at = NULL,
	lease_until = NULL,
	runner_id = NULL,
	exit_code = NULL,
	output = '',
	result_json = NULL,
	error_message = NULL,
	cancel_requested = false,
	runner_termination_requested_at = NULL,
	infra_retry_count = infra_retry_count + 1
WHERE id = ? AND runner_id = ? AND status = ? AND cancel_requested = false AND infra_retry_count < ?
RETURNING id`,
		string(models.StatusQueued), req.ID, req.RunnerID, string(models.StatusClaimed), maxInfraRetries,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return s.GetTask(ctx, rows[0].ID)
}

func (s *PostgresStore) MarkLostRunnerTasksTerminating(ctx context.Context, fleetID string, runnerIDs []string) ([]LostRunnerTaskTermination, error) {
	fleetID = strings.TrimSpace(fleetID)
	if fleetID == "" {
		return nil, fmt.Errorf("fleet_id required for lost runner termination")
	}
	runnerIDs = compactRunnerIDs(runnerIDs)
	if len(runnerIDs) == 0 {
		return []LostRunnerTaskTermination{}, nil
	}

	type row struct {
		ID       string
		FleetID  string
		RunnerID string
	}
	var rows []row
	now := time.Now().UTC()
	err := s.db.WithContext(ctx).Raw(`
WITH candidates AS (
	SELECT id, fleet_id, runner_id
	FROM tasks
	WHERE fleet_id = ? AND status = ? AND runner_id IN ?
	FOR UPDATE
),
updated AS (
	UPDATE tasks t SET
		runner_termination_requested_at = COALESCE(t.runner_termination_requested_at, ?)
	FROM candidates c
	WHERE t.id = c.id
	RETURNING t.id, t.fleet_id, c.runner_id
)
SELECT id, fleet_id, runner_id FROM updated ORDER BY runner_id ASC, id ASC`,
		fleetID, string(models.StatusClaimed), runnerIDs, now,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	tasks := make([]LostRunnerTaskTermination, 0, len(rows))
	for _, row := range rows {
		tasks = append(tasks, LostRunnerTaskTermination{
			ID:       row.ID,
			FleetID:  row.FleetID,
			RunnerID: row.RunnerID,
		})
	}
	return tasks, nil
}

func (s *PostgresStore) FinalizeTerminatedRunnerTasks(ctx context.Context, fleetID string, runnerIDs []string) ([]LostRunnerTaskRecovery, error) {
	fleetID = strings.TrimSpace(fleetID)
	if fleetID == "" {
		return nil, fmt.Errorf("fleet_id required for terminated runner finalization")
	}
	runnerIDs = compactRunnerIDs(runnerIDs)
	if len(runnerIDs) == 0 {
		return []LostRunnerTaskRecovery{}, nil
	}

	now := time.Now().UTC()
	var recoveries []LostRunnerTaskRecovery
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		canceled, err := recoverLostRunnerTasks(tx, `
WITH candidates AS (
	SELECT id, fleet_id, runner_id
	FROM tasks
	WHERE fleet_id = ? AND status = ? AND runner_id IN ? AND cancel_requested = true AND runner_termination_requested_at IS NOT NULL
	FOR UPDATE
),
updated AS (
	UPDATE tasks t SET
		status = ?,
		cancel_requested = false,
		claimed_at = NULL,
		lease_until = NULL,
		runner_id = NULL,
		runner_termination_requested_at = NULL,
		exit_code = ?,
		output = ?,
		result_json = NULL,
		error_message = NULL,
		environment_json = '',
		finished_at = ?
	FROM candidates c
	WHERE t.id = c.id
	RETURNING t.id, t.fleet_id, c.runner_id, t.status
)
SELECT id, fleet_id, runner_id, status FROM updated ORDER BY runner_id ASC, id ASC`,
			fleetID, string(models.StatusClaimed), runnerIDs,
			string(models.StatusCanceled), exitCanceled, msgCanceledRunnerLost, now,
		)
		if err != nil {
			return err
		}
		recoveries = append(recoveries, canceled...)

		failed, err := recoverLostRunnerTasks(tx, `
WITH candidates AS (
	SELECT id, fleet_id, runner_id
	FROM tasks
	WHERE fleet_id = ? AND status = ? AND runner_id IN ? AND cancel_requested = false AND runner_termination_requested_at IS NOT NULL
	FOR UPDATE
),
updated AS (
	UPDATE tasks t SET
		status = ?,
		claimed_at = NULL,
		lease_until = NULL,
		runner_id = NULL,
		runner_termination_requested_at = NULL,
		exit_code = 1,
		output = '',
		result_json = NULL,
		error_message = ?,
		cancel_requested = false,
		environment_json = '',
		finished_at = ?
	FROM candidates c
	WHERE t.id = c.id
	RETURNING t.id, t.fleet_id, c.runner_id, t.status
)
	SELECT id, fleet_id, runner_id, status FROM updated ORDER BY runner_id ASC, id ASC`,
			fleetID, string(models.StatusClaimed), runnerIDs,
			string(models.StatusFailed), msgRunnerLost, now,
		)
		if err != nil {
			return err
		}
		recoveries = append(recoveries, failed...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return recoveries, nil
}

func recoverLostRunnerTasks(tx *gorm.DB, query string, args ...any) ([]LostRunnerTaskRecovery, error) {
	var rows []struct {
		ID       string
		FleetID  string
		RunnerID string
		Status   string
	}
	if err := tx.Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	recoveries := make([]LostRunnerTaskRecovery, 0, len(rows))
	for _, row := range rows {
		recoveries = append(recoveries, LostRunnerTaskRecovery{
			ID:       row.ID,
			FleetID:  row.FleetID,
			RunnerID: row.RunnerID,
			Status:   models.TaskStatus(row.Status),
		})
	}
	return recoveries, nil
}

func nullIfEmpty(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func (s *PostgresStore) ReapExpiredLeases(ctx context.Context) ([]ReapedLease, []*models.Task, error) {
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
	error_message = NULL,
	finished_at = ?
WHERE status = ? AND lease_until IS NOT NULL AND lease_until <= ? AND cancel_requested = true AND runner_termination_requested_at IS NULL
RETURNING id`,
		string(models.StatusCanceled), exitCanceled, msgCanceledLeaseReap, now,
		string(models.StatusClaimed), now,
	).Scan(&canceledIDs).Error
	if err != nil {
		return nil, nil, err
	}

	var canceled []*models.Task
	for _, row := range canceledIDs {
		t, err := s.GetTask(ctx, row.ID)
		if err != nil {
			return nil, nil, err
		}
		canceled = append(canceled, t)
	}

	type reapRow struct {
		ID      string
		FleetID string
	}
	var requeuedRows []reapRow
	err = s.db.WithContext(ctx).Raw(`
UPDATE tasks SET
	status = ?,
	claimed_at = NULL,
	lease_until = NULL,
	runner_id = NULL
WHERE status = ? AND lease_until IS NOT NULL AND lease_until <= ? AND cancel_requested = false AND runner_termination_requested_at IS NULL
RETURNING id, fleet_id`,
		string(models.StatusQueued), string(models.StatusClaimed), now,
	).Scan(&requeuedRows).Error
	if err != nil {
		return nil, canceled, err
	}

	requeued := make([]ReapedLease, 0, len(requeuedRows))
	for _, row := range requeuedRows {
		requeued = append(requeued, ReapedLease{ID: row.ID, FleetID: row.FleetID})
	}
	return requeued, canceled, nil
}
