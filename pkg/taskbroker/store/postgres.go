package store

import (
	"context"
	"errors"

	brokermodels "github.com/superplanehq/superplane/pkg/taskbroker/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PostgresStore implements Store using PostgreSQL via GORM.
type PostgresStore struct {
	db *gorm.DB
}

// NewPostgresStore wraps an existing GORM connection. Schema is owned by
// SuperPlane SQL migrations — this does not AutoMigrate.
func NewPostgresStore(db *gorm.DB) *PostgresStore {
	return &PostgresStore{db: db}
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
