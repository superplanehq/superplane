package store

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	brokermodels "github.com/superplane/runner/task-broker/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	return s.db.AutoMigrate(&brokermodels.Fleet{}, &brokermodels.BrokerTask{})
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
	return s.db.WithContext(ctx).Exec("TRUNCATE TABLE fleets, broker_tasks RESTART IDENTITY CASCADE").Error
}

func (s *PostgresStore) CreateFleet(ctx context.Context, f *brokermodels.Fleet) error {
	row := *f
	row.Labels = NormalizeLabels(f.Labels)
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"base_url", "auth_token", "labels", "created_at"}),
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

// FindFleetByLabels returns any fleet whose label set contains all required labels.
// Tie-breaker: smallest fleet id alphabetically for stability.
func (s *PostgresStore) FindFleetByLabels(ctx context.Context, required []string) (*brokermodels.Fleet, error) {
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

func (s *PostgresStore) InsertBrokerTask(ctx context.Context, t *brokermodels.BrokerTask) error {
	return s.db.WithContext(ctx).Create(t).Error
}

func (s *PostgresStore) UpdateBrokerTaskFleetTaskID(ctx context.Context, brokerID, fleetTaskID string) error {
	res := s.db.WithContext(ctx).Model(&brokermodels.BrokerTask{}).
		Where("id = ?", brokerID).
		Update("fleet_task_id", fleetTaskID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("broker task not found: %s", brokerID)
	}
	return nil
}

func (s *PostgresStore) GetBrokerTask(ctx context.Context, brokerID string) (*brokermodels.BrokerTask, error) {
	var t brokermodels.BrokerTask
	err := s.db.WithContext(ctx).First(&t, "id = ?", brokerID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *PostgresStore) DeleteBrokerTask(ctx context.Context, brokerID string) error {
	return s.db.WithContext(ctx).Delete(&brokermodels.BrokerTask{}, "id = ?", brokerID).Error
}
