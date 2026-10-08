package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	RunnerStatePending    = "pending"
	RunnerStateIdle       = "idle"
	RunnerStateBusy       = "busy"
	RunnerStateTerminated = "terminated"

	RunnerTerminationRequested           = "requested"
	RunnerTerminationInterrupted         = "interrupted"
	RunnerTerminationRegistrationExpired = "registration_timeout"
	RunnerTerminationProvisioningFailed  = "provisioning_failed"
	RunnerTerminationVersionMismatch     = "version_mismatch"
	RunnerTerminationTaskCompleted       = "task_completed"
	RunnerTerminationConnectionLost      = "connection_lost"
)

var (
	ErrRunnerNotFound            = errors.New("runner not found")
	ErrRunnerBusy                = errors.New("busy runner cannot be terminated")
	ErrRunnerIdempotencyConflict = errors.New("idempotency key was used with a different request")
	ErrRunnerRegistrationInvalid = errors.New("runner registration is invalid")
	ErrRunnerVersionMismatch     = errors.New("runner version does not match")
	ErrRunnerCredentialNotFound  = errors.New("runner credential not found")
	ErrRunnerConnectionReplaced  = errors.New("runner connection was replaced")
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
 * Termination releases the key because a retry after a failed provisioning
 * attempt must create a replacement runner for the same task.
 *
 * CreationRequestHash binds the idempotency key to its original fleet and
 * task inputs so a client cannot reuse the key for a different request.
 */
type Runner struct {
	ID                     uuid.UUID
	FleetID                uuid.UUID
	State                  string
	RunnerVersion          string
	OS                     string
	Arch                   string
	Hostname               string
	IP                     string
	Tags                   datatypes.JSONType[map[string]string]
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

type RunnerHost struct {
	OS       string
	Arch     string
	Hostname string
	IP       string
	Tags     map[string]string
}

/*
 * RunnerCredential stores the hash of the bearer token that authenticates a
 * registered runner. The plaintext token is returned once at registration and
 * is never persisted. A recently completed ephemeral runner can reconnect to
 * retry its idempotent completion until credential cleanup revokes the token.
 */
type RunnerCredential struct {
	RunnerID        uuid.UUID `gorm:"primaryKey"`
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
	JTI        uuid.UUID `gorm:"primaryKey"`
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

func LockRunner(tx *gorm.DB, id uuid.UUID) (*Runner, error) {
	return FindRunner(
		tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}),
		id,
	)
}

func ListExpiredPendingRunnerIDs(tx *gorm.DB, now time.Time, limit int) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := tx.Table("runners").
		Select("runners.id").
		Joins("JOIN runner_registrations ON runner_registrations.runner_id = runners.id").
		Where(
			"runners.state = ? AND runner_registrations.consumed_at IS NULL "+
				"AND runner_registrations.revoked_at IS NULL AND runner_registrations.expires_at <= ?",
			RunnerStatePending,
			now,
		).
		Order("runner_registrations.expires_at ASC").
		Limit(limit).
		Pluck("runners.id", &ids).
		Error
	return ids, err
}

func (r *Runner) Expire(tx *gorm.DB, now time.Time) error {
	if r.State != RunnerStatePending {
		return nil
	}

	result := tx.Model(&RunnerRegistration{}).
		Where(
			"runner_id = ? AND consumed_at IS NULL AND revoked_at IS NULL AND expires_at <= ?",
			r.ID,
			now,
		).
		Update("revoked_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return nil
	}

	return r.Terminate(tx, RunnerTerminationRegistrationExpired)
}

func RevokeTerminatedRunnerCredentials(tx *gorm.DB, terminatedBefore, now time.Time) error {
	return tx.Exec(
		`UPDATE runner_credentials
		 SET revoked_at = ?
		 FROM runners
		 WHERE runner_credentials.runner_id = runners.id
		   AND runner_credentials.revoked_at IS NULL
		   AND runners.state = ?
		   AND runners.terminated_at <= ?`,
		now,
		RunnerStateTerminated,
		terminatedBefore,
	).Error
}

func ListStaleRunnerIDs(tx *gorm.DB, lastSeenBefore time.Time, limit int) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := tx.Model(&Runner{}).
		Where("state IN ?", []string{RunnerStateIdle, RunnerStateBusy}).
		Where("last_seen_at IS NOT NULL AND last_seen_at <= ?", lastSeenBefore).
		Order("last_seen_at ASC").
		Limit(limit).
		Pluck("id", &ids).
		Error
	return ids, err
}

/*
 * Register consumes one registration grant and creates the long-lived
 * credential for this runner. The row locks make token consumption and runner
 * activation atomic when duplicate registration requests arrive.
 *
 * A task-bound grant must match the task reserved during runner creation.
 * Generic grants must not register a runner that gained an assignment through
 * another path.
 */
func (r *Runner) Register(
	tx *gorm.DB,
	jti uuid.UUID,
	fleetSlug string,
	taskID *uuid.UUID,
	version string,
	host RunnerHost,
	accessTokenHash string,
	now time.Time,
) error {
	var current Runner
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", r.ID).
		First(&current).
		Error; err != nil {
		return fmt.Errorf("%w: runner record is unavailable", ErrRunnerRegistrationInvalid)
	}
	*r = current

	if r.State != RunnerStatePending || r.RunnerVersion != version {
		if r.RunnerVersion != version {
			return ErrRunnerVersionMismatch
		}
		return fmt.Errorf("%w: runner is not pending", ErrRunnerRegistrationInvalid)
	}

	var registration RunnerRegistration
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("jti = ? AND runner_id = ?", jti, r.ID).
		First(&registration).
		Error
	if err != nil {
		return fmt.Errorf("%w: registration grant is unavailable", ErrRunnerRegistrationInvalid)
	}
	switch {
	case registration.ConsumedAt != nil:
		return fmt.Errorf("%w: registration grant is consumed", ErrRunnerRegistrationInvalid)
	case registration.RevokedAt != nil:
		return fmt.Errorf("%w: registration grant is revoked", ErrRunnerRegistrationInvalid)
	case !registration.ExpiresAt.After(now):
		return fmt.Errorf("%w: registration grant is expired", ErrRunnerRegistrationInvalid)
	}

	var fleet RunnerFleet
	err = tx.Where("id = ? AND slug = ? AND enabled = ?", r.FleetID, fleetSlug, true).
		First(&fleet).
		Error
	if err != nil {
		return fmt.Errorf("%w: fleet is unavailable", ErrRunnerRegistrationInvalid)
	}

	var task RunnerTask
	taskErr := tx.
		Where("runner_id = ? AND state IN ?", r.ID, []string{
			RunnerTaskStateReserved,
			RunnerTaskStateRunning,
		}).
		First(&task).
		Error
	switch {
	case taskID == nil && taskErr == nil:
		return fmt.Errorf("%w: generic runner has a reserved task", ErrRunnerRegistrationInvalid)
	case taskID != nil && (taskErr != nil || task.ID != *taskID):
		return fmt.Errorf("%w: reserved task does not match token", ErrRunnerRegistrationInvalid)
	case taskErr != nil && !errors.Is(taskErr, gorm.ErrRecordNotFound):
		return taskErr
	}

	credential := &RunnerCredential{
		RunnerID:        r.ID,
		AccessTokenHash: accessTokenHash,
		CreatedAt:       now,
	}
	if err := tx.Create(credential).Error; err != nil {
		return err
	}

	result := tx.Model(&registration).
		Where("consumed_at IS NULL AND revoked_at IS NULL").
		Update("consumed_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("%w: registration grant was consumed concurrently", ErrRunnerRegistrationInvalid)
	}

	result = tx.Model(r).
		Where("state = ?", RunnerStatePending).
		Updates(map[string]any{
			"state":         RunnerStateIdle,
			"os":            host.OS,
			"arch":          host.Arch,
			"hostname":      host.Hostname,
			"ip":            host.IP,
			"tags":          datatypes.NewJSONType(host.Tags),
			"registered_at": now,
			"last_seen_at":  now,
			"updated_at":    now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("%w: runner state changed concurrently", ErrRunnerRegistrationInvalid)
	}

	r.State = RunnerStateIdle
	r.OS = host.OS
	r.Arch = host.Arch
	r.Hostname = host.Hostname
	r.IP = host.IP
	r.Tags = datatypes.NewJSONType(host.Tags)
	r.RegisteredAt = &now
	r.LastSeenAt = &now
	r.UpdatedAt = now
	return nil
}

func FindRunnerByAccessTokenHash(tx *gorm.DB, accessTokenHash string) (*Runner, error) {
	var runner Runner
	err := tx.Model(&Runner{}).
		Select("runners.*").
		Joins("JOIN runner_credentials ON runner_credentials.runner_id = runners.id").
		Where("runner_credentials.access_token_hash = ?", accessTokenHash).
		Where("runner_credentials.revoked_at IS NULL").
		First(&runner).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRunnerCredentialNotFound
	}
	if err != nil {
		return nil, err
	}
	return &runner, nil
}

func (r *Runner) OpenConnection(tx *gorm.DB, connectionID uuid.UUID, now time.Time) error {
	result := tx.Model(r).
		Where("state IN ? AND registered_at IS NOT NULL", []string{
			RunnerStateIdle,
			RunnerStateBusy,
			RunnerStateTerminated,
		}).
		Updates(map[string]any{
			"current_connection_id": connectionID,
			"last_seen_at":          now,
			"updated_at":            now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrRunnerConnectionReplaced
	}
	r.CurrentConnectionID = &connectionID
	r.LastSeenAt = &now
	r.UpdatedAt = now
	return nil
}

func (r *Runner) TouchConnection(tx *gorm.DB, connectionID uuid.UUID, now time.Time) error {
	result := tx.Model(r).
		Where("current_connection_id = ?", connectionID).
		Updates(map[string]any{
			"last_seen_at": now,
			"updated_at":   now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrRunnerConnectionReplaced
	}
	r.LastSeenAt = &now
	r.UpdatedAt = now
	return nil
}

func (r *Runner) CloseConnection(tx *gorm.DB, connectionID uuid.UUID, now time.Time) error {
	result := tx.Model(r).
		Where("current_connection_id = ?", connectionID).
		Updates(map[string]any{
			"current_connection_id": nil,
			"updated_at":            now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRunnerConnectionReplaced
	}
	r.CurrentConnectionID = nil
	r.UpdatedAt = now
	return nil
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

// ReleaseTerminatedRunnerCreationIdempotencyKey lets a controller retry the
// same logical provisioning request after its previous runner terminated.
// Pending and active runners keep the key so concurrent retries continue to
// return one runner.
func ReleaseTerminatedRunnerCreationIdempotencyKey(tx *gorm.DB, key string) error {
	return tx.Model(&Runner{}).
		Where("creation_idempotency_key = ? AND state = ?", key, RunnerStateTerminated).
		Updates(map[string]any{
			"creation_idempotency_key": nil,
			"creation_request_hash":    nil,
		}).
		Error
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
		return releaseReservedRunnerTask(tx, r.ID, time.Now())
	}

	now := time.Now()
	result := tx.Model(r).
		Where("state IN ?", []string{RunnerStatePending, RunnerStateIdle}).
		Updates(map[string]any{
			"state":                    RunnerStateTerminated,
			"termination_reason":       reason,
			"terminated_at":            now,
			"updated_at":               now,
			"current_connection_id":    nil,
			"creation_idempotency_key": nil,
			"creation_request_hash":    nil,
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
			return releaseReservedRunnerTask(tx, r.ID, now)
		}
		return ErrRunnerBusy
	}

	r.State = RunnerStateTerminated
	r.TerminationReason = &reason
	r.TerminatedAt = &now
	r.UpdatedAt = now
	r.CurrentConnectionID = nil
	r.CreationIdempotencyKey = nil
	r.CreationRequestHash = nil
	if err := releaseReservedRunnerTask(tx, r.ID, now); err != nil {
		return err
	}
	if err := tx.Model(&RunnerCredential{}).
		Where("runner_id = ? AND revoked_at IS NULL", r.ID).
		Update("revoked_at", now).
		Error; err != nil {
		return err
	}
	return tx.Model(&RunnerRegistration{}).
		Where("runner_id = ? AND revoked_at IS NULL", r.ID).
		Update("revoked_at", now).
		Error
}

func releaseReservedRunnerTask(tx *gorm.DB, runnerID uuid.UUID, now time.Time) error {
	var task RunnerTask
	err := tx.Where("runner_id = ? AND state = ?", runnerID, RunnerTaskStateReserved).
		First(&task).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return tx.Model(&task).Updates(map[string]any{
		"runner_id":   nil,
		"state":       RunnerTaskStateQueued,
		"reserved_at": nil,
		"updated_at":  now,
	}).Error
}
