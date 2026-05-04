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
	return s.ensureCommandsJSONColumn()
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
INSERT INTO tasks (id, command_json, commands_json, webhook_url, status, created_at, execution_mode, docker_image)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
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
	execution_mode, docker_image, exit_code, output, error_message
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
	)
	if err := row.Scan(&id, &cmdJSON, &commandsJSON, &webhook, &status, &createdAt, &claimedAt, &leaseUntil,
		&runnerID, &execMode, &dockerImage, &exitCode, &output, &errMsg); err != nil {
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
		ID:            id,
		Command:       cmd,
		Commands:      cmds,
		WebhookURL:    webhook,
		Status:        models.TaskStatus(status),
		CreatedAt:     time.Unix(createdAt.Int64, 0).UTC(),
		RunnerID:      runnerID.String,
		ExecutionMode: models.ExecutionMode(execMode),
		DockerImage:   dockerImage.String,
		Output:        output.String,
		ErrorMessage:  errMsg.String,
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

func (s *SQLiteStore) CompleteTask(ctx context.Context, id, runnerID string, exitCode int, output, errMsg string) (*models.Task, error) {
	final := models.StatusSucceeded
	if exitCode != 0 || errMsg != "" {
		final = models.StatusFailed
	}
	res, err := s.db.ExecContext(ctx, `
UPDATE tasks SET
	status = ?,
	exit_code = ?,
	output = ?,
	error_message = ?
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

func (s *SQLiteStore) ReapExpiredLeases(ctx context.Context) (int64, error) {
	now := time.Now().Unix()
	res, err := s.db.ExecContext(ctx, `
UPDATE tasks SET
	status = ?,
	claimed_at = NULL,
	lease_until = NULL,
	runner_id = NULL
WHERE status = ? AND lease_until IS NOT NULL AND lease_until <= ?`,
		string(models.StatusQueued), string(models.StatusClaimed), now,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
