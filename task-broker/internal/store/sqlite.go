package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	brokermodels "github.com/superplane/runner/task-broker/internal/models"

	_ "modernc.org/sqlite"
)

// SQLiteStore implements Store using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// OpenSQLite opens or creates broker.db-style storage.
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
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS fleets (
	id TEXT PRIMARY KEY,
	base_url TEXT NOT NULL,
	auth_token TEXT,
	labels_json TEXT NOT NULL DEFAULT '[]',
	created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS broker_tasks (
	id TEXT PRIMARY KEY,
	fleet_id TEXT NOT NULL,
	fleet_task_id TEXT,
	caller_webhook_url TEXT NOT NULL,
	created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS broker_tasks_fleet_task ON broker_tasks(fleet_task_id);
`)
	return err
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) CreateFleet(ctx context.Context, f *brokermodels.Fleet) error {
	l := NormalizeLabels(f.Labels)
	b, err := json.Marshal(l)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
REPLACE INTO fleets (id, base_url, auth_token, labels_json, created_at)
VALUES (?, ?, ?, ?, ?)`,
		f.ID, f.BaseURL, nullString(f.AuthToken), string(b), f.CreatedAt.Unix(),
	)
	return err
}

func (s *SQLiteStore) DeleteFleet(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM fleets WHERE id = ?`, id)
	return err
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (s *SQLiteStore) ListFleets(ctx context.Context) ([]brokermodels.Fleet, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, base_url, auth_token, labels_json, created_at FROM fleets ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []brokermodels.Fleet
	for rows.Next() {
		f, err := scanFleet(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

func scanFleet(s interface {
	Scan(dest ...any) error
}) (*brokermodels.Fleet, error) {
	var id, baseURL string
	var auth sql.NullString
	var labelsJSON string
	var createdUnix int64
	if err := s.Scan(&id, &baseURL, &auth, &labelsJSON, &createdUnix); err != nil {
		return nil, err
	}
	var labels []string
	if labelsJSON != "" {
		if err := json.Unmarshal([]byte(labelsJSON), &labels); err != nil {
			return nil, err
		}
	}
	return &brokermodels.Fleet{
		ID:        id,
		BaseURL:   baseURL,
		AuthToken: auth.String,
		Labels:    labels,
		CreatedAt: time.Unix(createdUnix, 0).UTC(),
	}, nil
}

func (s *SQLiteStore) GetFleet(ctx context.Context, id string) (*brokermodels.Fleet, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, base_url, auth_token, labels_json, created_at FROM fleets WHERE id = ?`, id)
	f, err := scanFleet(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return f, nil
}

// FindFleetByLabels returns any fleet whose label set contains all required labels.
// Tie-breaker: smallest fleet id alphabetically for stability.
func (s *SQLiteStore) FindFleetByLabels(ctx context.Context, required []string) (*brokermodels.Fleet, error) {
	req := NormalizeLabels(required)
	if len(req) == 0 {
		return nil, nil
	}
	fleets, err := s.ListFleets(ctx)
	if err != nil {
		return nil, err
	}
	var candidates []string
	for _, f := range fleets {
		if LabelsSubset(f.Labels, req) {
			candidates = append(candidates, f.ID)
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	sort.Strings(candidates)
	return s.GetFleet(ctx, candidates[0])
}

func (s *SQLiteStore) InsertBrokerTask(ctx context.Context, t *brokermodels.BrokerTask) error {
	var fleetTask any
	if t.FleetTaskID != "" {
		fleetTask = t.FleetTaskID
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO broker_tasks (id, fleet_id, fleet_task_id, caller_webhook_url, created_at)
VALUES (?, ?, ?, ?, ?)`,
		t.ID, t.FleetID, fleetTask, t.CallerWebhookURL, t.CreatedAt.Unix(),
	)
	return err
}

func (s *SQLiteStore) UpdateBrokerTaskFleetTaskID(ctx context.Context, brokerID, fleetTaskID string) error {
	res, err := s.db.ExecContext(ctx, `
UPDATE broker_tasks SET fleet_task_id = ? WHERE id = ?`, fleetTaskID, brokerID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("broker task not found: %s", brokerID)
	}
	return nil
}

func (s *SQLiteStore) GetBrokerTask(ctx context.Context, brokerID string) (*brokermodels.BrokerTask, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, fleet_id, fleet_task_id, caller_webhook_url, created_at
FROM broker_tasks WHERE id = ?`, brokerID)
	var id, fid, cw string
	var ft sql.NullString
	var ct int64
	if err := row.Scan(&id, &fid, &ft, &cw, &ct); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &brokermodels.BrokerTask{
		ID:               id,
		FleetID:          fid,
		FleetTaskID:      ft.String,
		CallerWebhookURL: cw,
		CreatedAt:        time.Unix(ct, 0).UTC(),
	}, nil
}

func (s *SQLiteStore) DeleteBrokerTask(ctx context.Context, brokerID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM broker_tasks WHERE id = ?`, brokerID)
	return err
}
