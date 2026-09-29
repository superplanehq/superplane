package store

import (
	"context"
	"errors"
	"strings"
	"time"

	brokermodels "github.com/superplane/runner/task-broker/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInvalidRunnerRegistration = errors.New("invalid runner registration")

// RegisterRunnerWithJTI consumes a registration JWT jti (single-use) and issues a runner credential.
func (s *PostgresStore) RegisterRunnerWithJTI(
	ctx context.Context,
	jti, runnerID, fleetID, accessTokenHash string,
	now time.Time,
) error {
	jti = strings.TrimSpace(jti)
	runnerID = strings.TrimSpace(runnerID)
	fleetID = strings.TrimSpace(fleetID)
	if jti == "" || runnerID == "" || fleetID == "" || accessTokenHash == "" {
		return ErrInvalidRunnerRegistration
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		used := brokermodels.UsedRegistrationJTI{
			JTI:        jti,
			FleetID:    fleetID,
			ConsumedAt: now,
		}
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&used)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrInvalidRunnerRegistration
		}

		credential := brokermodels.RunnerCredential{
			RunnerID:        runnerID,
			FleetID:         fleetID,
			AccessTokenHash: accessTokenHash,
			CreatedAt:       now,
		}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "runner_id"}, {Name: "fleet_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"access_token_hash", "created_at"}),
		}).Create(&credential).Error
	})
}

func (s *PostgresStore) GetRunnerByAccessTokenHash(ctx context.Context, accessTokenHash string) (*brokermodels.RunnerCredential, error) {
	var credential brokermodels.RunnerCredential
	err := s.db.WithContext(ctx).First(&credential, "access_token_hash = ?", accessTokenHash).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &credential, nil
}

func (s *PostgresStore) DeleteRunnerCredentials(ctx context.Context, fleetID string, runnerIDs []string) error {
	if len(runnerIDs) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).
		Delete(&brokermodels.RunnerCredential{}, "fleet_id = ? AND runner_id IN ?", fleetID, runnerIDs).
		Error
}
