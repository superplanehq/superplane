package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/superplane/runner/shared/models"

	_ "modernc.org/sqlite"
)

// SQLiteStore implements Store with SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// OpenSQLite opens or creates a SQLite database and applies schema.
func OpenSQLite(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(time.Hour)

	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLiteStore) migrate() error {
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS tasks (
	id TEXT PRIMARY KEY,
	command_json TEXT NOT NULL,
	webhook_url TEXT NOT NULL,
	status TEXT NOT NULL,
	created_at INTEGER NOT NULL,
	claimed_at INTEGER,
	lease_until INTEGER,
	runner_id TEXT,
	execution_mode TEXT NOT NULL DEFAULT 'host',
	docker_image TEXT,
	exit_code INTEGER,
	output TEXT,
	error_message TEXT
);
CREATE INDEX IF NOT EXISTS tasks_status_created ON tasks(status, created_at);
`); err != nil {
		return err
	}
	if err := s.ensureCommandsJSONColumn(); err != nil {
		return err
	}
	return s.ensureCancelRequestedColumn()
}

func (s *SQLiteStore) ensureCommandsJSONColumn() error {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('tasks') WHERE name='commands_json'`).Scan(&n)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err = s.db.Exec(`ALTER TABLE tasks ADD COLUMN commands_json TEXT`)
	return err
}

func (s *SQLiteStore) ensureCancelRequestedColumn() error {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('tasks') WHERE name='cancel_requested'`).Scan(&n)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err = s.db.Exec(`ALTER TABLE tasks ADD COLUMN cancel_requested INTEGER NOT NULL DEFAULT 0`)
	return err
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) CreateTask(ctx context.Context, t *models.Task) error {
	cmd := t.Command
	if cmd == nil {
		cmd = []string{}
	}
	cmdJSON, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	var cmdsJSON any
	if len(t.Commands) > 0 {
		b, err := json.Marshal(t.Commands)
		if err != nil {
			return err
		}
		cmdsJSON = string(b)
	}
	if t.ExecutionMode == "" {
		t.ExecutionMode = models.ExecutionHost
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO tasks (id, command_json, commands_json, webhook_url, status, created_at, execution_mode, docker_image, cancel_requested)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0)`,
		t.ID, string(cmdJSON), cmdsJSON, t.WebhookURL, string(t.Status), t.CreatedAt.Unix(),
		string(t.ExecutionMode), nullString(t.DockerImage),
	)
	return err
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (s *SQLiteStore) GetTask(ctx context.Context, id string) (*models.Task, error) {
	t, err := s.GetByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return t, err
}

func (s *SQLiteStore) ClaimTask(ctx context.Context, runnerID string, lease time.Duration) (*models.Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().Unix()
	leaseUntil := now + int64(lease.Seconds())

	var id string
	err = tx.QueryRowContext(ctx, `
UPDATE tasks SET
	status = ?,
	claimed_at = ?,
	lease_until = ?,
	runner_id = ?
WHERE rowid = (
	SELECT rowid FROM tasks WHERE status = ? ORDER BY created_at ASC LIMIT 1
)
RETURNING id`,
		string(models.StatusClaimed), now, leaseUntil, runnerID,
		string(models.StatusQueued),
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

// GetByID returns a task by id.
func (s *SQLiteStore) GetByID(ctx context.Context, id string) (*models.Task, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, command_json, commands_json, webhook_url, status, created_at, claimed_at, lease_until, runner_id,
	execution_mode, docker_image, exit_code, output, error_message, cancel_requested
FROM tasks WHERE id = ?`, id)
	return scanTask(row)
}

func scanTask(row *sql.Row) (*models.Task, error) {
	var (
		id, cmdJSON, webhook, status          string
		commandsJSON                          sql.NullString
		createdAt, claimedAt, leaseUntil      sql.NullInt64
		runnerID, dockerImage, output, errMsg sql.NullString
		execMode                              string
		exitCode                              sql.NullInt64
		cancelReq                             int64
	)
	if err := row.Scan(&id, &cmdJSON, &commandsJSON, &webhook, &status, &createdAt, &claimedAt, &leaseUntil,
		&runnerID, &execMode, &dockerImage, &exitCode, &output, &errMsg, &cancelReq); err != nil {
		return nil, err
	}
	var cmd []string
	if err := json.Unmarshal([]byte(cmdJSON), &cmd); err != nil {
		return nil, fmt.Errorf("command_json: %w", err)
	}
	var cmds []string
	if commandsJSON.Valid && strings.TrimSpace(commandsJSON.String) != "" {
		if err := json.Unmarshal([]byte(commandsJSON.String), &cmds); err != nil {
			return nil, fmt.Errorf("commands_json: %w", err)
		}
	}
	t := &models.Task{
		ID:              id,
		Command:         cmd,
		Commands:        cmds,
		WebhookURL:      webhook,
		Status:          models.TaskStatus(status),
		CreatedAt:       time.Unix(createdAt.Int64, 0).UTC(),
		RunnerID:        runnerID.String,
		ExecutionMode:   models.ExecutionMode(execMode),
		DockerImage:     dockerImage.String,
		Output:          output.String,
		ErrorMessage:    errMsg.String,
		CancelRequested: cancelReq != 0,
	}
	if claimedAt.Valid {
		ct := time.Unix(claimedAt.Int64, 0).UTC()
		t.ClaimedAt = &ct
	}
	if leaseUntil.Valid {
		lt := time.Unix(leaseUntil.Int64, 0).UTC()
		t.LeaseUntil = &lt
	}
	if exitCode.Valid {
		ec := int(exitCode.Int64)
		t.ExitCode = &ec
	}
	return t, nil
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

// RequestCancelTask implements caller-initiated stop (see CancelOutcome).
func (s *SQLiteStore) RequestCancelTask(ctx context.Context, id string) (*models.Task, CancelOutcome, error) {
	res, err := s.db.ExecContext(ctx, `
UPDATE tasks SET
	status = ?,
	exit_code = ?,
	output = ?,
	error_message = NULL,
	claimed_at = NULL,
	lease_until = NULL,
	runner_id = NULL,
	cancel_requested = 0
WHERE id = ? AND status = ?`,
		string(models.StatusCanceled), exitCanceled, msgCanceledQueued, id, string(models.StatusQueued),
	)
	if err != nil {
		return nil, CancelOutcome(""), err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, CancelOutcome(""), err
	}
	if n == 1 {
		t, err := s.GetByID(ctx, id)
		if err != nil {
			return nil, CancelOutcome(""), err
		}
		return t, CancelOutcomeCanceledQueued, nil
	}

	res, err = s.db.ExecContext(ctx, `
UPDATE tasks SET cancel_requested = 1 WHERE id = ? AND status = ?`,
		id, string(models.StatusClaimed),
	)
	if err != nil {
		return nil, CancelOutcome(""), err
	}
	n, err = res.RowsAffected()
	if err != nil {
		return nil, CancelOutcome(""), err
	}
	if n == 1 {
		t, err := s.GetByID(ctx, id)
		if err != nil {
			return nil, CancelOutcome(""), err
		}
		return t, CancelOutcomeCancelRequested, nil
	}

	t, err := s.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, CancelOutcomeNotFound, nil
		}
		return nil, CancelOutcome(""), err
	}
	if terminalTaskStatus(t.Status) {
		return t, CancelOutcomeAlreadyTerminal, nil
	}
	return nil, CancelOutcome(""), fmt.Errorf("cancel: task %s in unexpected state %s", id, t.Status)
}

func (s *SQLiteStore) CompleteTask(ctx context.Context, id, runnerID string, exitCode int, output, errMsg string, canceled bool) (*models.Task, error) {
	var final models.TaskStatus
	if canceled {
		final = models.StatusCanceled
	} else {
		final = models.StatusSucceeded
		if exitCode != 0 || errMsg != "" {
			final = models.StatusFailed
		}
	}
	res, err := s.db.ExecContext(ctx, `
UPDATE tasks SET
	status = ?,
	exit_code = ?,
	output = ?,
	error_message = ?,
	cancel_requested = 0
WHERE id = ? AND runner_id = ? AND status = ?`,
		string(final), exitCode, output, nullStringErr(errMsg), id, runnerID, string(models.StatusClaimed),
	)
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, fmt.Errorf("task not found, wrong runner, or not claimed: %s", id)
	}
	return s.GetByID(ctx, id)
}

func nullStringErr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (s *SQLiteStore) ReapExpiredLeases(ctx context.Context) (int64, []*models.Task, error) {
	now := time.Now().Unix()

	rows, err := s.db.QueryContext(ctx, `
UPDATE tasks SET
	status = ?,
	cancel_requested = 0,
	claimed_at = NULL,
	lease_until = NULL,
	runner_id = NULL,
	exit_code = ?,
	output = ?,
	error_message = NULL
WHERE status = ? AND lease_until IS NOT NULL AND lease_until <= ? AND cancel_requested = 1
RETURNING id`,
		string(models.StatusCanceled), exitCanceled, msgCanceledLeaseReap,
		string(models.StatusClaimed), now,
	)
	if err != nil {
		return 0, nil, err
	}
	var canceledIDs []string
	for rows.Next() {
		var tid string
		if err := rows.Scan(&tid); err != nil {
			_ = rows.Close()
			return 0, nil, err
		}
		canceledIDs = append(canceledIDs, tid)
	}
	if err := rows.Close(); err != nil {
		return 0, nil, err
	}
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}

	var canceled []*models.Task
	for _, tid := range canceledIDs {
		t, err := s.GetByID(ctx, tid)
		if err != nil {
			return 0, nil, err
		}
		canceled = append(canceled, t)
	}

	res, err := s.db.ExecContext(ctx, `
UPDATE tasks SET
	status = ?,
	claimed_at = NULL,
	lease_until = NULL,
	runner_id = NULL
WHERE status = ? AND lease_until IS NOT NULL AND lease_until <= ? AND cancel_requested = 0`,
		string(models.StatusQueued), string(models.StatusClaimed), now,
	)
	if err != nil {
		return 0, canceled, err
	}
	requeued, err := res.RowsAffected()
	if err != nil {
		return 0, canceled, err
	}
	return requeued, canceled, nil
}
