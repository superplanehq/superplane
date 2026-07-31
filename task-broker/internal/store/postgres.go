package store

import (
	"context"
	"errors"
	"sync"
	"time"

	brokermodels "github.com/superplane/runner/task-broker/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// PostgresStore implements Store using PostgreSQL via GORM.
type PostgresStore struct {
	db *gorm.DB
}

var migrateMu sync.Mutex

// OpenPostgres connects to PostgreSQL and applies schema migrations.
func OpenPostgres(dsn string) (*PostgresStore, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	s := &PostgresStore{db: db}
	if err := s.migrate(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return s, nil
}

func (s *PostgresStore) migrate() error {
	migrateMu.Lock()
	defer migrateMu.Unlock()
	if err := s.db.AutoMigrate(&brokermodels.Fleet{}, &brokermodels.Task{}); err != nil {
		return err
	}
	// Drop legacy proxy columns from earlier broker versions.
	for _, stmt := range []string{
		`ALTER TABLE fleets DROP COLUMN IF EXISTS base_url`,
		`ALTER TABLE fleets DROP COLUMN IF EXISTS auth_token`,
		`ALTER TABLE fleets DROP COLUMN IF EXISTS labels`,
		`ALTER TABLE fleets DROP COLUMN IF EXISTS type`,
		`DROP TABLE IF EXISTS broker_tasks`,
	} {
		if err := s.db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *PostgresStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Truncate removes all rows (for tests).
func (s *PostgresStore) Truncate(ctx context.Context) error {
	return s.db.WithContext(ctx).Exec("TRUNCATE TABLE fleets, tasks RESTART IDENTITY CASCADE").Error
}

// CreateFleet upserts a fleet. provisioner/arch/size/created_at are always
// overwritten by the caller's values, since they're expected to be fully
// resent on every registration. lambda_function_name, max_execution_timeout_seconds,
// and supports_docker are merged atomically in SQL instead: an empty/nil value
// in f means "not provided" and keeps whatever the fleet already had, so a
// caller that doesn't know about these fields can't clobber them on conflict.
func (s *PostgresStore) CreateFleet(ctx context.Context, f *brokermodels.Fleet) error {
	return s.db.WithContext(ctx).Raw(`
INSERT INTO fleets (id, provisioner, arch, size, created_at, lambda_function_name, max_execution_timeout_seconds, supports_docker)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (id) DO UPDATE SET
	provisioner = EXCLUDED.provisioner,
	arch = EXCLUDED.arch,
	size = EXCLUDED.size,
	created_at = EXCLUDED.created_at,
	lambda_function_name = COALESCE(NULLIF(EXCLUDED.lambda_function_name, ''), fleets.lambda_function_name),
	max_execution_timeout_seconds = COALESCE(EXCLUDED.max_execution_timeout_seconds, fleets.max_execution_timeout_seconds),
	supports_docker = COALESCE(EXCLUDED.supports_docker, fleets.supports_docker)
RETURNING id, provisioner, arch, size, created_at, lambda_function_name, max_execution_timeout_seconds, supports_docker`,
		f.ID, f.Provisioner, f.Arch, f.Size, f.CreatedAt,
		f.LambdaFunctionName, f.MaxExecutionTimeoutSeconds, f.SupportsDocker,
	).Scan(f).Error
}

func (s *PostgresStore) DeleteFleet(ctx context.Context, id string) error {
	res := s.db.WithContext(ctx).Delete(&brokermodels.Fleet{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (s *PostgresStore) ListFleets(ctx context.Context) ([]brokermodels.Fleet, error) {
	var out []brokermodels.Fleet
	err := s.db.WithContext(ctx).Order("id ASC").Find(&out).Error
	return out, err
}

func (s *PostgresStore) GetFleet(ctx context.Context, id string) (*brokermodels.Fleet, error) {
	var f brokermodels.Fleet
	err := s.db.WithContext(ctx).First(&f, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}
