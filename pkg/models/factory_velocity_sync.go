package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FactoryVelocitySync records how far a workspace's repository sync has
// reached. Repository is the repository the stored pull requests belong to, so
// a workspace that changes its app repository restarts the backfill.
type FactoryVelocitySync struct {
	FactoryID      uuid.UUID `gorm:"primaryKey"`
	Repository     string
	SyncedAt       *time.Time
	BackfilledFrom *time.Time
	Error          string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// FactoryVelocitySyncTarget is a workspace the sync worker must visit, together
// with the integration and repository its setup selected. SyncedRepository is
// the repository the stored rows were collected for, which differs from
// Repository when the workspace changed its app repository.
type FactoryVelocitySyncTarget struct {
	FactoryID        uuid.UUID
	OrganizationID   uuid.UUID
	IntegrationID    uuid.UUID
	Repository       string
	SyncedRepository string
	SyncedAt         *time.Time
	BackfilledFrom   *time.Time
}

func (FactoryVelocitySync) TableName() string {
	return "factory_velocity_syncs"
}

// CoversRepository reports whether the stored rows belong to the repository the
// workspace reports on now.
func (s *FactoryVelocitySync) CoversRepository(repository string) bool {
	return strings.EqualFold(strings.TrimSpace(s.Repository), strings.TrimSpace(repository))
}

// RecordSuccess stamps a finished sync and clears any earlier failure.
func (s *FactoryVelocitySync) RecordSuccess(tx *gorm.DB, repository string, syncedAt, backfilledFrom time.Time) error {
	now := time.Now()
	s.Repository = strings.TrimSpace(repository)
	s.SyncedAt = &syncedAt
	s.BackfilledFrom = &backfilledFrom
	s.Error = ""
	s.UpdatedAt = now

	return tx.Model(s).
		Where("factory_id = ?", s.FactoryID).
		Updates(map[string]any{
			"repository":      s.Repository,
			"synced_at":       s.SyncedAt,
			"backfilled_from": s.BackfilledFrom,
			"error":           "",
			"updated_at":      now,
		}).Error
}

// RecordError keeps synced_at untouched so the next tick retries the same
// window, and leaves the rows a previous sync stored in place.
func (s *FactoryVelocitySync) RecordError(tx *gorm.DB, message string) error {
	now := time.Now()
	s.Error = message
	s.UpdatedAt = now

	return tx.Model(s).
		Where("factory_id = ?", s.FactoryID).
		Updates(map[string]any{"error": message, "updated_at": now}).Error
}

// FindFactoryVelocitySync returns the sync row of a workspace. A workspace that
// has never synced has no row, and the caller gets gorm.ErrRecordNotFound.
func FindFactoryVelocitySync(tx *gorm.DB, factoryID uuid.UUID) (*FactoryVelocitySync, error) {
	var sync FactoryVelocitySync
	err := tx.Where("factory_id = ?", factoryID).First(&sync).Error
	if err != nil {
		return nil, err
	}
	return &sync, nil
}

// ListFactoryVelocitySyncTargets returns the workspaces whose setup selected a
// GitHub integration and an app repository, whose last sync is older than
// staleBefore, and which no other worker is syncing now. A workspace that never
// synced comes first.
//
// claimableBefore is the lease horizon: a row touched more recently than that
// is either freshly synced or in flight, and either way must be left alone.
func ListFactoryVelocitySyncTargets(
	tx *gorm.DB,
	staleBefore, claimableBefore time.Time,
	limit int,
) ([]FactoryVelocitySyncTarget, error) {
	var targets []FactoryVelocitySyncTarget
	err := tx.Raw(listFactoryVelocitySyncTargetsSQL, staleBefore, claimableBefore, limit).
		Scan(&targets).Error
	if err != nil {
		return nil, err
	}
	return targets, nil
}

const listFactoryVelocitySyncTargetsSQL = `
SELECT` + factoryVelocitySyncTargetColumns + `
FROM factories f
LEFT JOIN factory_velocity_syncs s ON s.factory_id = f.id
WHERE f.deleted_at IS NULL
	AND COALESCE(f.onboarding_config->>'vcs_integration_id', '') <> ''
	AND COALESCE(f.onboarding_config->>'app_repository', '') <> ''
	AND (s.synced_at IS NULL OR s.synced_at < ?)
	AND (s.updated_at IS NULL OR s.updated_at < ?)
ORDER BY s.synced_at ASC NULLS FIRST
LIMIT ?
`

// FindFactoryVelocitySyncTarget returns one workspace's sync target, whatever
// its schedule says. A user asking for a refresh must not be turned away because
// the workspace is not due yet.
//
// It reports nil when the workspace has no repository or no version control
// integration to read, which is a workspace the sync has nothing to do for.
func FindFactoryVelocitySyncTarget(
	tx *gorm.DB,
	factoryID uuid.UUID,
) (*FactoryVelocitySyncTarget, error) {
	var targets []FactoryVelocitySyncTarget
	err := tx.Raw(findFactoryVelocitySyncTargetSQL, factoryID).Scan(&targets).Error
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return nil, nil
	}
	return &targets[0], nil
}

const findFactoryVelocitySyncTargetSQL = `
SELECT` + factoryVelocitySyncTargetColumns + `
FROM factories f
LEFT JOIN factory_velocity_syncs s ON s.factory_id = f.id
WHERE f.deleted_at IS NULL
	AND f.id = ?
	AND COALESCE(f.onboarding_config->>'vcs_integration_id', '') <> ''
	AND COALESCE(f.onboarding_config->>'app_repository', '') <> ''
`

const factoryVelocitySyncTargetColumns = `
	f.id AS factory_id,
	f.organization_id,
	(f.onboarding_config->>'vcs_integration_id')::uuid AS integration_id,
	f.onboarding_config->>'app_repository' AS repository,
	COALESCE(s.repository, '') AS synced_repository,
	s.synced_at,
	s.backfilled_from`

// ClaimFactoryVelocitySync takes ownership of a workspace's sync for the length
// of the lease, and returns nil when another worker holds it.
//
// The claim is a single conditional UPDATE that moves updated_at forward, so no
// transaction is held while the worker talks to GitHub. A worker that dies
// mid-sync releases the workspace when the lease expires.
func ClaimFactoryVelocitySync(
	tx *gorm.DB,
	factoryID uuid.UUID,
	claimableBefore time.Time,
) (*FactoryVelocitySync, error) {
	// Seeded with the epoch so the caller that creates the row can claim it at
	// once. Raw SQL keeps GORM from stamping updated_at with the current time.
	if err := tx.Exec(seedFactoryVelocitySyncSQL, factoryID).Error; err != nil {
		return nil, err
	}

	var claimed []FactoryVelocitySync
	err := tx.Raw(claimFactoryVelocitySyncSQL, time.Now(), factoryID, claimableBefore).
		Scan(&claimed).Error
	if err != nil {
		return nil, err
	}
	if len(claimed) == 0 {
		return nil, nil
	}
	return &claimed[0], nil
}

const seedFactoryVelocitySyncSQL = `
INSERT INTO factory_velocity_syncs (factory_id, created_at, updated_at)
VALUES (?, NOW(), 'epoch')
ON CONFLICT (factory_id) DO NOTHING
`

const claimFactoryVelocitySyncSQL = `
UPDATE factory_velocity_syncs
SET updated_at = ?
WHERE factory_id = ? AND updated_at < ?
RETURNING factory_id, repository, synced_at, backfilled_from, error, created_at, updated_at
`
