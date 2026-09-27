package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

const (
	RunnerStatePending    = "pending"
	RunnerStateIdle       = "idle"
	RunnerStateBusy       = "busy"
	RunnerStateTerminated = "terminated"

	RunnerTerminationRequested           = "requested"
	RunnerTerminationRegistrationExpired = "registration_timeout"
	RunnerTerminationProvisioningFailed  = "provisioning_failed"
	RunnerTerminationVersionMismatch     = "version_mismatch"
)

var (
	ErrRunnerNotFound            = errors.New("runner not found")
	ErrRunnerBusy                = errors.New("busy runner cannot be terminated")
	ErrRunnerIdempotencyConflict = errors.New("idempotency key was used with a different request")
)

const runnerCreationIdempotencyKeyConstraint = "runners_creation_idempotency_key"

/*
 * Runner is a durable record for one provisioned execution agent.
 * SuperPlane creates it in the pending state before infrastructure starts,
 * then uses it to track registration, connectivity, workload state, and termination.
 *
 * Ephemeral runners terminate after their first task. Reusable runners return
 * to idle. This is an instance property because one fleet can contain both.
 *
 * CreationIdempotencyKey is optional for generic capacity. Task-specific
 * runners require it so a lost create response can recover the runner and
 * registration token without leaving the task reserved until expiry.
 *
 * CreationRequestHash binds the idempotency key to its original fleet and
 * task inputs so a client cannot reuse the key for a different request.
 */
type Runner struct {
	ID                     uuid.UUID
	FleetID                uuid.UUID
	State                  string
	RunnerVersion          string
	Ephemeral              bool
	RegisteredAt           *time.Time
	LastSeenAt             *time.Time
	CurrentConnectionID    *uuid.UUID
	TerminationReason      *string
	CreatedAt              time.Time
	UpdatedAt              time.Time
	TerminatedAt           *time.Time
	CreationIdempotencyKey *string
	CreationRequestHash    *string
}

/*
 * RunnerCredential stores the hash of the bearer token that authenticates a
 * registered runner. The plaintext token is returned once at registration and
 * is never persisted. Revocation prevents a terminated runner from reconnecting.
 */
type RunnerCredential struct {
	RunnerID        uuid.UUID
	AccessTokenHash string
	CreatedAt       time.Time
	RevokedAt       *time.Time
}

/*
 * RunnerRegistration is a short-lived, single-use grant for converting a
 * pending runner into an authenticated runner. Its JTI is embedded in the
 * registration JWT so SuperPlane can reject replayed, expired, or revoked
 * grants without storing the JWT itself.
 */
type RunnerRegistration struct {
	JTI        uuid.UUID
	RunnerID   uuid.UUID
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
}

func FindRunnerRegistration(tx *gorm.DB, runnerID uuid.UUID) (*RunnerRegistration, error) {
	var registration RunnerRegistration
	err := tx.
		Where("runner_id = ?", runnerID).
		Order("created_at ASC").
		First(&registration).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRunnerNotFound
	}
	if err != nil {
		return nil, err
	}
	return &registration, nil
}

func FindRunner(tx *gorm.DB, id uuid.UUID) (*Runner, error) {
	var runner Runner
	err := tx.Where("id = ?", id).First(&runner).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRunnerNotFound
	}
	if err != nil {
		return nil, err
	}
	return &runner, nil
}

func FindInstallationRunnerByCreationIdempotencyKey(tx *gorm.DB, key string) (*Runner, error) {
	var runner Runner
	err := tx.Model(&Runner{}).
		Select("runners.*").
		Joins("JOIN runner_fleets ON runner_fleets.id = runners.fleet_id").
		Where("runners.creation_idempotency_key = ?", key).
		Where("runner_fleets.scope_type = ?", RunnerFleetScopeInstallation).
		Where("runner_fleets.deleted_at IS NULL").
		First(&runner).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRunnerNotFound
	}
	if err != nil {
		return nil, err
	}
	return &runner, nil
}

// IsRunnerCreationIdempotencyKeyConflict identifies the unique-key race that
// occurs when concurrent retries create the same logical runner. The caller
// can then load and return the runner committed by the winning transaction.
func IsRunnerCreationIdempotencyKeyConflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.ConstraintName == runnerCreationIdempotencyKeyConstraint
}

func (r *Runner) Terminate(tx *gorm.DB, reason string) error {
	if r.State == RunnerStateBusy {
		return ErrRunnerBusy
	}
	if r.State == RunnerStateTerminated {
		return nil
	}

	now := time.Now()
	result := tx.Model(r).
		Where("state IN ?", []string{RunnerStatePending, RunnerStateIdle}).
		Updates(map[string]any{
			"state":                 RunnerStateTerminated,
			"termination_reason":    reason,
			"terminated_at":         now,
			"updated_at":            now,
			"current_connection_id": nil,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		current, err := FindRunner(tx, r.ID)
		if err != nil {
			return err
		}
		if current.State == RunnerStateTerminated {
			*r = *current
			return nil
		}
		return ErrRunnerBusy
	}

	r.State = RunnerStateTerminated
	r.TerminationReason = &reason
	r.TerminatedAt = &now
	r.UpdatedAt = now
	r.CurrentConnectionID = nil
	return nil
}
