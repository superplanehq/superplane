package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	FactoryVelocitySyncStateIdle = 0
	// Backfilling continues older history before BackfilledFrom.
	FactoryVelocitySyncStateBackfilling = -1
	// CatchUp reads newer merges after the older history is saved.
	FactoryVelocitySyncStateCatchUp = -2
	// CatchingUp continues a newer-window read before BackfilledFrom.
	FactoryVelocitySyncStateCatchingUp = -3

	factoryVelocitySyncCandidatePageSize = 200
)

// FactoryVelocitySync records how far a workspace's repository sync has
// reached. Repository is the repository the stored pull requests belong to, so
// a workspace that changes its app repository restarts the backfill.
type FactoryVelocitySync struct {
	FactoryID       uuid.UUID `gorm:"primaryKey"`
	ScopeRepository string    `gorm:"primaryKey"`
	Repository      string
	SyncedAt        *time.Time
	BackfilledFrom  *time.Time
	// SyncState records which batch read must continue.
	SyncState int
	Error     string
	CreatedAt time.Time
	UpdatedAt time.Time
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
	VCSProvider      string
	SyncedRepository string
	SyncedAt         *time.Time
	UpdatedAt        *time.Time
	BackfilledFrom   *time.Time
	SyncState        int
}

func (FactoryVelocitySync) TableName() string {
	return "factory_velocity_syncs"
}

// CoversRepository reports whether the stored rows belong to the repository the
// workspace reports on now.
func (s *FactoryVelocitySync) CoversRepository(repository string) bool {
	return strings.EqualFold(strings.TrimSpace(s.Repository), strings.TrimSpace(repository))
}

// SyncInProgress reports a bounded read that has not finished. SyncedAt may
// still hold the time the newest pages were read. That stamp is not a
// finished sync.
func (s *FactoryVelocitySync) SyncInProgress() bool {
	return s != nil && s.SyncState != FactoryVelocitySyncStateIdle
}

// RecordProgress saves a short slice without marking the sync finished.
// newestReadAt is when the newest pages were read. A zero time clears the
// stamp. A later slice keeps the earlier stamp so the finishing read can
// include merges made after the first slice. An active sync state
// stays unfinished even when the stamp is set.
func (s *FactoryVelocitySync) RecordProgress(tx *gorm.DB, repository string, backfilledFrom, newestReadAt time.Time) error {
	s.Repository = strings.TrimSpace(repository)
	s.BackfilledFrom = &backfilledFrom
	s.Error = ""

	var syncedAt any
	if newestReadAt.IsZero() {
		s.SyncedAt = nil
		syncedAt = gorm.Expr("NULL")
	} else {
		readAt := newestReadAt
		s.SyncedAt = &readAt
		syncedAt = s.SyncedAt
	}

	return s.recordUpdates(tx, map[string]any{
		"repository":      s.Repository,
		"synced_at":       syncedAt,
		"backfilled_from": s.BackfilledFrom,
		"sync_state":      s.SyncState,
		"error":           "",
	})
}

// RecordSuccess stamps a finished sync and clears any earlier failure.
func (s *FactoryVelocitySync) RecordSuccess(tx *gorm.DB, repository string, syncedAt, backfilledFrom time.Time) error {
	return s.RecordProgress(tx, repository, backfilledFrom, syncedAt)
}

// RecordError keeps synced_at untouched so the next tick retries the same
// window, and leaves the rows a previous sync stored in place.
func (s *FactoryVelocitySync) RecordError(tx *gorm.DB, message string) error {
	s.Error = message
	return s.recordUpdates(tx, map[string]any{"error": message})
}

// LockClaim fences a write against cleanup or a newer lease. The caller must
// hold this row lock until both the merges and progress commit.
func (s *FactoryVelocitySync) LockClaim(tx *gorm.DB) error {
	var current FactoryVelocitySync
	return tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("factory_id = ? AND scope_repository = ? AND updated_at = ?", s.FactoryID, s.ScopeRepository, s.UpdatedAt).
		First(&current).Error
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

func FindFactoryVelocityRepositorySync(tx *gorm.DB, factoryID uuid.UUID, repository string) (*FactoryVelocitySync, error) {
	var sync FactoryVelocitySync
	err := tx.Where("factory_id = ? AND (scope_repository = ? OR (scope_repository = '' AND lower(trim(repository)) = ?))", factoryID, CanonicalContextRepo(repository), CanonicalContextRepo(repository)).Order("scope_repository DESC").First(&sync).Error
	return &sync, err
}

// FactoryVelocityRepositoryTargets resolves the collection set with workspace access rules.
func FactoryVelocityRepositoryTargets(tx *gorm.DB, target FactoryVelocitySyncTarget) ([]FactoryVelocitySyncTarget, error) {
	var factory Factory
	if err := tx.Where("id = ?", target.FactoryID).First(&factory).Error; err != nil {
		return nil, err
	}
	repositories, err := factory.WorkspaceRepositories(tx)
	if err != nil {
		return nil, err
	}
	if err := ReconcileFactoryVelocityRepositories(tx, factory.ID, repositories); err != nil {
		return nil, err
	}
	targets := make([]FactoryVelocitySyncTarget, 0, len(repositories))
	for _, repository := range repositories {
		next := target
		next.Repository = repository
		next.SyncedRepository, next.SyncedAt, next.BackfilledFrom, next.SyncState = "", nil, nil, 0
		next.UpdatedAt = nil
		sync, err := FindFactoryVelocityRepositorySync(tx, factory.ID, repository)
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, err
		}
		if err == nil {
			next.SyncedRepository, next.SyncedAt, next.BackfilledFrom, next.SyncState = sync.Repository, sync.SyncedAt, sync.BackfilledFrom, sync.SyncState
			next.UpdatedAt = &sync.UpdatedAt
		}
		targets = append(targets, next)
	}
	return targets, nil
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
	result := []FactoryVelocitySyncTarget{}
	for offset := 0; ; offset += factoryVelocitySyncCandidatePageSize {
		var targets []FactoryVelocitySyncTarget
		err := tx.Raw(listFactoryVelocitySyncTargetsSQL, factoryVelocitySyncCandidatePageSize, offset).
			Scan(&targets).Error
		if err != nil {
			return nil, err
		}
		for _, target := range targets {
			expanded, err := FactoryVelocityRepositoryTargets(tx, target)
			if err != nil {
				return nil, err
			}
			for _, candidate := range expanded {
				if (candidate.SyncedAt != nil && !candidate.SyncedAt.Before(staleBefore)) ||
					(candidate.UpdatedAt != nil && !candidate.UpdatedAt.Before(claimableBefore)) {
					continue
				}
				result = append(result, candidate)
				if limit > 0 && len(result) >= limit {
					return result, nil
				}
			}
		}
		if len(targets) < factoryVelocitySyncCandidatePageSize {
			return result, nil
		}
	}
}

const listFactoryVelocitySyncTargetsSQL = `
SELECT` + factoryVelocitySyncTargetColumns + `
FROM factories f
LEFT JOIN factory_velocity_syncs s ON s.factory_id = f.id AND s.scope_repository = lower(trim(f.onboarding_config->>'app_repository'))
WHERE f.deleted_at IS NULL
	AND COALESCE(f.onboarding_config->>'vcs_integration_id', '') <> ''
	AND COALESCE(f.onboarding_config->>'app_repository', '') <> ''
ORDER BY s.synced_at ASC NULLS FIRST, f.id
LIMIT ? OFFSET ?
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
LEFT JOIN factory_velocity_syncs s ON s.factory_id = f.id AND (s.scope_repository = lower(trim(f.onboarding_config->>'app_repository')) OR s.scope_repository = '')
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
	COALESCE(f.onboarding_config->>'vcs_provider', '') AS vcs_provider,
	COALESCE(s.repository, '') AS synced_repository,
	s.synced_at,
	s.updated_at,
	s.backfilled_from,
	COALESCE(s.sync_state, 0) AS sync_state`

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
	repository ...string,
) (*FactoryVelocitySync, error) {
	// Seeded with the epoch so the caller that creates the row can claim it at
	// once. Raw SQL keeps GORM from stamping updated_at with the current time.
	scope := ""
	if len(repository) > 0 {
		scope = CanonicalContextRepo(repository[0])
	}
	if err := tx.Exec(seedFactoryVelocitySyncSQL, factoryID, scope).Error; err != nil {
		return nil, err
	}

	var claimed []FactoryVelocitySync
	err := tx.Raw(claimFactoryVelocitySyncSQL, time.Now(), factoryID, scope, claimableBefore).
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
INSERT INTO factory_velocity_syncs (factory_id, scope_repository, created_at, updated_at)
VALUES (?, ?, NOW(), 'epoch')
ON CONFLICT (factory_id, scope_repository) DO NOTHING
`

const claimFactoryVelocitySyncSQL = `
UPDATE factory_velocity_syncs
SET updated_at = ?
WHERE factory_id = ? AND scope_repository = ? AND updated_at < ?
RETURNING factory_id, scope_repository, repository, synced_at, backfilled_from, sync_state, error, created_at, updated_at
`

func (s *FactoryVelocitySync) recordUpdates(tx *gorm.DB, updates map[string]any) error {
	// Match PostgreSQL timestamp precision so the next slice can use this token.
	now := time.Now().Truncate(time.Microsecond)
	updates["updated_at"] = now
	result := tx.Model(&FactoryVelocitySync{}).
		Where("factory_id = ? AND scope_repository = ? AND updated_at = ?", s.FactoryID, s.ScopeRepository, s.UpdatedAt).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	s.UpdatedAt = now
	return nil
}
