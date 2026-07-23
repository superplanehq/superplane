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

func (s *PostgresStore) CreateRunnerRegistration(ctx context.Context, registration *brokermodels.RunnerRegistration) error {
	return s.db.WithContext(ctx).Create(registration).Error
}

func (s *PostgresStore) ExchangeRunnerRegistration(
	ctx context.Context,
	registrationHash, runnerID, fleetID, accessTokenHash string,
	now time.Time,
) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var registration brokermodels.RunnerRegistration
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&registration, "token_hash = ?", registrationHash).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidRunnerRegistration
		}
		if err != nil {
			return err
		}
		if registration.ConsumedAt != nil || !registration.ExpiresAt.After(now) ||
			registration.FleetID != strings.TrimSpace(fleetID) {
			return ErrInvalidRunnerRegistration
		}

		credential := brokermodels.RunnerCredential{
			RunnerID:        strings.TrimSpace(runnerID),
			FleetID:         registration.FleetID,
			AccessTokenHash: accessTokenHash,
			CreatedAt:       now,
		}
		if err := tx.Create(&credential).Error; err != nil {
			return err
		}
		return tx.Model(&registration).Update("consumed_at", now).Error
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
