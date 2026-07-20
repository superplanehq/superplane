package store

import (
	"context"
	"errors"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	brokermodels "github.com/superplanehq/superplane/pkg/runnerbroker/storemodels"
)

// PostgresStore implements Store using PostgreSQL via GORM.
type PostgresStore struct {
	db *gorm.DB
}

// NewPostgresStore uses SuperPlane's existing PostgreSQL connection.
func NewPostgresStore(db *gorm.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// OpenPostgres connects to PostgreSQL. Schema is managed by SuperPlane migrations.
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

	return NewPostgresStore(db), nil
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
	return s.db.WithContext(ctx).Exec("TRUNCATE TABLE runner_broker_fleets, runner_broker_tasks RESTART IDENTITY CASCADE").Error
}

func (s *PostgresStore) CreateFleet(ctx context.Context, f *brokermodels.Fleet) error {
	row := *f
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"provisioner", "arch", "size", "created_at"}),
	}).Create(&row).Error
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
