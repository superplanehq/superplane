package postgres

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	api "github.com/superplanehq/superplane/pkg/runners/logs"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	schemaLockName = "superplane_runner_active_log_schema"
	schemaVersion  = 1
)

type Store struct {
	setupMu sync.Mutex
	db      *gorm.DB
	metrics storeMetrics
}

type activeLog struct {
	TaskID       uuid.UUID `gorm:"primaryKey"`
	NextSequence int64
	TotalBytes   int64
	Truncated    bool
	UpdatedAt    time.Time
}

func (activeLog) TableName() string {
	return "runner_active_logs"
}

type activeLogChunk struct {
	TaskID    uuid.UUID `gorm:"primaryKey"`
	Sequence  int64     `gorm:"primaryKey"`
	Content   []byte
	CreatedAt time.Time
}

func (activeLogChunk) TableName() string {
	return "runner_active_log_chunks"
}

func New() *Store {
	return &Store{}
}

func (s *Store) Name() string {
	return api.StorePostgres
}

func (s *Store) Setup(ctx api.SetupContext) error {
	s.setupMu.Lock()
	defer s.setupMu.Unlock()

	if ctx.Context == nil {
		return errors.New("PostgreSQL active log store requires a context")
	}
	if ctx.Database == nil {
		return errors.New("PostgreSQL active log store requires a database")
	}
	if ctx.MeterProvider == nil {
		return errors.New("PostgreSQL active log store requires a meter provider")
	}
	metrics, err := newStoreMetrics(ctx.MeterProvider)
	if err != nil {
		return fmt.Errorf("create PostgreSQL active log store metrics: %w", err)
	}
	err = ctx.Database.WithContext(ctx.Context).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			"SELECT pg_advisory_xact_lock(hashtextextended(?, 0))",
			schemaLockName,
		).Error; err != nil {
			return fmt.Errorf("lock active log schema: %w", err)
		}

		if err := tx.Exec(`
			CREATE TABLE IF NOT EXISTS runner_active_log_schema_migrations (
			  version INTEGER PRIMARY KEY,
			  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)
		`).Error; err != nil {
			return fmt.Errorf("create active log schema migrations: %w", err)
		}

		var current int
		if err := tx.Raw(
			"SELECT COALESCE(MAX(version), 0) FROM runner_active_log_schema_migrations",
		).Scan(&current).Error; err != nil {
			return fmt.Errorf("read active log schema version: %w", err)
		}
		if current >= schemaVersion {
			return nil
		}

		if err := tx.Exec(`
			CREATE TABLE runner_active_logs (
			  task_id UUID PRIMARY KEY,
			  next_sequence BIGINT NOT NULL DEFAULT 0,
			  total_bytes BIGINT NOT NULL DEFAULT 0,
			  truncated BOOLEAN NOT NULL DEFAULT FALSE,
			  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			  CONSTRAINT runner_active_logs_values_check
			    CHECK (next_sequence >= 0 AND total_bytes >= 0)
			);

			CREATE INDEX runner_active_logs_updated_at_idx
			  ON runner_active_logs (updated_at);

			CREATE TABLE runner_active_log_chunks (
			  task_id UUID NOT NULL
			    REFERENCES runner_active_logs(task_id) ON DELETE CASCADE,
			  sequence BIGINT NOT NULL,
			  content BYTEA NOT NULL,
			  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			  PRIMARY KEY (task_id, sequence),
			  CONSTRAINT runner_active_log_chunks_sequence_check
			    CHECK (sequence >= 0)
			);
		`).Error; err != nil {
			return fmt.Errorf("apply active log schema version 1: %w", err)
		}

		if err := tx.Exec(
			"INSERT INTO runner_active_log_schema_migrations (version) VALUES (?)",
			schemaVersion,
		).Error; err != nil {
			return fmt.Errorf("record active log schema version 1: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.db = ctx.Database
	s.metrics = metrics
	return nil
}

func (s *Store) Initialize(ctx context.Context, taskID uuid.UUID) (err error) {
	startedAt := time.Now()
	defer func() {
		s.metrics.recordOperation(ctx, startedAt, "initialize", err)
	}()

	result := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&activeLog{
			TaskID:    taskID,
			UpdatedAt: time.Now(),
		})
	if result.Error != nil {
		return fmt.Errorf("initialize active runner log: %w", result.Error)
	}
	return nil
}

func (s *Store) Append(ctx context.Context, taskID uuid.UUID, sequence int64, content []byte) (result api.AppendResult, err error) {
	startedAt := time.Now()
	defer func() {
		s.metrics.recordOperation(ctx, startedAt, "append", err)
	}()

	if sequence < 0 {
		return api.AppendResult{}, api.ErrSequenceConflict
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		log, err := lockActiveLog(tx, taskID)
		if err != nil {
			return err
		}

		if sequence < log.NextSequence {
			result = api.AppendResult{
				Truncated: log.Truncated,
			}
			return nil
		}
		if sequence > log.NextSequence {
			return api.ErrSequenceConflict
		}
		if log.Truncated {
			result = api.AppendResult{Truncated: true}
			return nil
		}

		stored, truncated := api.RetainContent(log.TotalBytes, content)
		if err := tx.Create(&activeLogChunk{
			TaskID:    taskID,
			Sequence:  sequence,
			Content:   stored,
			CreatedAt: now,
		}).Error; err != nil {
			return err
		}

		nextBytes := log.TotalBytes + int64(len(stored))
		if err := tx.Model(&activeLog{}).
			Where("task_id = ?", taskID).
			Updates(map[string]any{
				"next_sequence": sequence + 1,
				"total_bytes":   nextBytes,
				"truncated":     truncated,
				"updated_at":    now,
			}).Error; err != nil {
			return err
		}

		result = api.AppendResult{
			Truncated: truncated,
		}
		return nil
	})
	if err != nil {
		return api.AppendResult{}, fmt.Errorf("append active runner log: %w", err)
	}
	return result, nil
}

func lockActiveLog(tx *gorm.DB, taskID uuid.UUID) (activeLog, error) {
	var log activeLog
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("task_id = ?", taskID).
		First(&log).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return activeLog{}, api.ErrNotFound
	}
	return log, err
}

func (s *Store) ReadAfter(ctx context.Context, taskID uuid.UUID, cursor string) (result *api.ReadResult, err error) {
	startedAt := time.Now()
	defer func() {
		s.metrics.recordOperation(ctx, startedAt, "read", err)
	}()

	return s.readAfter(ctx, taskID, cursor)
}

func (s *Store) readAfter(ctx context.Context, taskID uuid.UUID, cursor string) (*api.ReadResult, error) {
	after, err := parseCursor(cursor)
	if err != nil {
		return nil, err
	}

	var log activeLog
	err = s.db.WithContext(ctx).Where("task_id = ?", taskID).First(&log).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, api.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read active runner log metadata: %w", err)
	}
	if after > log.NextSequence {
		return nil, api.ErrInvalidCursor
	}

	var chunks []activeLogChunk
	if err := s.db.WithContext(ctx).
		Where("task_id = ? AND sequence >= ? AND sequence < ?", taskID, after, log.NextSequence).
		Order("sequence ASC").
		Find(&chunks).Error; err != nil {
		return nil, fmt.Errorf("read active runner log chunks: %w", err)
	}

	var content bytes.Buffer
	for _, chunk := range chunks {
		_, _ = content.Write(chunk.Content)
	}
	return &api.ReadResult{
		Content:   io.NopCloser(bytes.NewReader(content.Bytes())),
		Cursor:    formatCursor(log.NextSequence),
		Truncated: log.Truncated,
	}, nil
}

func (s *Store) Delete(ctx context.Context, taskID uuid.UUID) (err error) {
	startedAt := time.Now()
	defer func() {
		s.metrics.recordOperation(ctx, startedAt, "delete", err)
	}()

	if err = s.db.WithContext(ctx).Where("task_id = ?", taskID).Delete(&activeLog{}).Error; err != nil {
		return fmt.Errorf("delete active runner log: %w", err)
	}
	return nil
}

func (s *Store) DeleteExpired(ctx context.Context, before time.Time) (deleted int64, err error) {
	startedAt := time.Now()
	defer func() {
		s.metrics.recordOperation(ctx, startedAt, "delete_expired", err)
	}()

	result := s.db.WithContext(ctx).
		Where("updated_at < ?", before).
		Delete(&activeLog{})
	if result.Error != nil {
		return 0, fmt.Errorf("delete expired active runner logs: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func parseCursor(cursor string) (int64, error) {
	if cursor == "" {
		return 0, nil
	}
	sequence, err := strconv.ParseInt(cursor, 10, 64)
	if err != nil || sequence < 0 {
		return 0, api.ErrInvalidCursor
	}
	return sequence, nil
}

func formatCursor(sequence int64) string {
	return strconv.FormatInt(sequence, 10)
}

var _ api.Store = (*Store)(nil)
