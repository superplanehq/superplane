package models

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	RunnerFleetScopeInstallation = "installation"
	RunnerFleetScopeOrganization = "organization"

	RunnerFleetE1LargeAMD64 = "e1-large-amd64"
	RunnerFleetE1LargeARM64 = "e1-large-arm64"
	RunnerFleetE1TinyAMD64  = "e1-tiny-amd64"
	RunnerFleetE1TinyARM64  = "e1-tiny-arm64"

	DefaultRunnerVersion = "dev"
)

var (
	ErrRunnerFleetNotFound   = errors.New("runner fleet not found")
	ErrRunnerFleetIDConflict = errors.New("runner fleet ID conflicts with an existing fleet")
)

type RunnerFleetSpec struct {
	OperatingSystem            string   `json:"operating_system,omitempty"`
	Architecture               string   `json:"architecture,omitempty"`
	CPUMillicores              int32    `json:"cpu_millicores,omitempty"`
	MemoryMB                   int32    `json:"memory_mb,omitempty"`
	DiskGB                     int32    `json:"disk_gb,omitempty"`
	Capabilities               []string `json:"capabilities,omitempty"`
	MaxExecutionTimeoutSeconds int32    `json:"max_execution_timeout_seconds,omitempty"`
}

/*
 * RunnerFleet is SuperPlane's authoritative definition of a class of runner.
 * It identifies the installation or organization that can use the fleet and
 * defines the runner release and opaque provisioning specification that a
 * fleet manager must use. Spec remains JSON while fleet requirements evolve.
 * Promote stable, frequently queried values from Spec to columns when needed.
 *
 * ID is an internal relational key that keeps foreign keys narrow.
 * Slug is the stable, scope-local fleet ID exposed to users and used in
 * node and Fleet Manager configuration, for example "e1-large-amd64".
 */
type RunnerFleet struct {
	ID            uuid.UUID
	Slug          string
	ScopeType     string
	ScopeID       *uuid.UUID
	Enabled       bool
	Spec          datatypes.JSONType[RunnerFleetSpec]
	RunnerVersion string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt
}

// DefaultInstallationRunnerFleets is the initial fleet catalog for a new
// installation. Providers decide how to satisfy these specifications.
func DefaultInstallationRunnerFleets(runnerVersion string) []RunnerFleet {
	now := time.Now()
	return []RunnerFleet{
		defaultInstallationRunnerFleet(
			RunnerFleetE1LargeAMD64,
			"amd64",
			2000,
			8192,
			runnerVersion,
			now,
		),
		defaultInstallationRunnerFleet(
			RunnerFleetE1LargeARM64,
			"arm64",
			2000,
			8192,
			runnerVersion,
			now,
		),
		defaultInstallationRunnerFleet(
			RunnerFleetE1TinyAMD64,
			"amd64",
			2000,
			1024,
			runnerVersion,
			now,
		),
		defaultInstallationRunnerFleet(
			RunnerFleetE1TinyARM64,
			"arm64",
			2000,
			1024,
			runnerVersion,
			now,
		),
	}
}

func defaultInstallationRunnerFleet(
	slug, architecture string,
	cpuMillicores, memoryMB int32,
	runnerVersion string,
	now time.Time,
) RunnerFleet {
	return RunnerFleet{
		ID:            uuid.New(),
		Slug:          slug,
		ScopeType:     RunnerFleetScopeInstallation,
		Enabled:       true,
		RunnerVersion: runnerVersion,
		Spec: datatypes.NewJSONType(RunnerFleetSpec{
			OperatingSystem:            "linux",
			Architecture:               architecture,
			CPUMillicores:              cpuMillicores,
			MemoryMB:                   memoryMB,
			DiskGB:                     30,
			Capabilities:               []string{"docker"},
			MaxExecutionTimeoutSeconds: 3600,
		}),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// CreateDefaultInstallationRunnerFleets inserts missing defaults without
// changing fleets that an installation administrator already configured.
func CreateDefaultInstallationRunnerFleets(
	tx *gorm.DB,
	runnerVersion string,
) error {
	for _, fleet := range DefaultInstallationRunnerFleets(runnerVersion) {
		existing, err := FindInstallationRunnerFleet(tx, fleet.Slug)
		if err == nil && existing != nil {
			continue
		}
		if !errors.Is(err, ErrRunnerFleetNotFound) {
			return err
		}
		if err := fleet.Create(tx); err != nil {
			return err
		}
	}
	return nil
}

/*
 * The required rules for fleet slugs are:
 * - Installation fleet IDs are unique.
 * - Organization fleet IDs are unique within each organization.
 * - Different organizations can reuse the same fleet ID.
 * - Organization fleet IDs cannot match any installation fleet ID.
 *
 * Create serializes fleet creation by slug before checking scope conflicts.
 * The transaction-scoped advisory lock prevents concurrent installation
 * and organization creates from both passing the conflict check before either row exists.
 */
func (f *RunnerFleet) Create(tx *gorm.DB) error {
	f.Slug = strings.TrimSpace(f.Slug)
	if f.Slug == "" {
		return errors.New("runner fleet ID is required")
	}

	return tx.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			"SELECT pg_advisory_xact_lock(hashtextextended(?, 0))",
			f.Slug,
		).Error; err != nil {
			return err
		}

		query := tx.Model(&RunnerFleet{}).Where("slug = ?", f.Slug)
		if f.ScopeType == RunnerFleetScopeOrganization {
			query = query.Where(
				"scope_type = ? OR scope_id = ?",
				RunnerFleetScopeInstallation,
				f.ScopeID,
			)
		}

		var count int64
		if err := query.Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrRunnerFleetIDConflict
		}

		return tx.Create(f).Error
	})
}

func ListInstallationRunnerFleets(tx *gorm.DB) ([]RunnerFleet, error) {
	var fleets []RunnerFleet
	err := tx.
		Where("scope_type = ? AND scope_id IS NULL", RunnerFleetScopeInstallation).
		Order("slug ASC").
		Find(&fleets).
		Error
	return fleets, err
}

func ListEnabledRunnerFleetsForOrganization(tx *gorm.DB, organizationID uuid.UUID) ([]RunnerFleet, error) {
	var fleets []RunnerFleet
	err := tx.
		Where("enabled = ?", true).
		Where(
			"(scope_type = ? AND scope_id IS NULL) OR (scope_type = ? AND scope_id = ?)",
			RunnerFleetScopeInstallation,
			RunnerFleetScopeOrganization,
			organizationID,
		).
		Order("slug ASC").
		Find(&fleets).
		Error
	return fleets, err
}

func FindInstallationRunnerFleet(tx *gorm.DB, slug string) (*RunnerFleet, error) {
	var fleet RunnerFleet
	err := tx.
		Where("slug = ? AND scope_type = ? AND scope_id IS NULL", slug, RunnerFleetScopeInstallation).
		First(&fleet).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRunnerFleetNotFound
	}
	if err != nil {
		return nil, err
	}
	return &fleet, nil
}

func FindEnabledRunnerFleetForOrganization(
	tx *gorm.DB,
	organizationID uuid.UUID,
	slug string,
) (*RunnerFleet, error) {
	var fleet RunnerFleet
	err := tx.
		Where("slug = ? AND enabled = ?", slug, true).
		Where(
			"(scope_type = ? AND scope_id IS NULL) OR (scope_type = ? AND scope_id = ?)",
			RunnerFleetScopeInstallation,
			RunnerFleetScopeOrganization,
			organizationID,
		).
		First(&fleet).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRunnerFleetNotFound
	}
	if err != nil {
		return nil, err
	}
	return &fleet, nil
}

func FindRunnerFleet(tx *gorm.DB, id uuid.UUID) (*RunnerFleet, error) {
	var fleet RunnerFleet
	err := tx.First(&fleet, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRunnerFleetNotFound
	}
	if err != nil {
		return nil, err
	}
	return &fleet, nil
}

func (f *RunnerFleet) FindRunner(tx *gorm.DB, id uuid.UUID) (*Runner, error) {
	var runner Runner
	err := tx.Where("id = ? AND fleet_id = ?", id, f.ID).First(&runner).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRunnerNotFound
	}
	if err != nil {
		return nil, err
	}
	return &runner, nil
}

func (f *RunnerFleet) ListRunners(tx *gorm.DB, states []string, limit int) ([]Runner, error) {
	query := tx.Where("fleet_id = ?", f.ID)
	if len(states) > 0 {
		query = query.Where("state IN ?", states)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}

	var runners []Runner
	err := query.Order("created_at ASC, id ASC").Find(&runners).Error
	return runners, err
}

func (f *RunnerFleet) ListTasks(tx *gorm.DB, states []string, limit int) ([]RunnerTask, error) {
	query := tx.Where("fleet_id = ?", f.ID)
	if len(states) > 0 {
		query = query.Where("state IN ?", states)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}

	var tasks []RunnerTask
	err := query.Order("queued_at ASC, id ASC").Find(&tasks).Error
	return tasks, err
}

func (f *RunnerFleet) CountTasks(tx *gorm.DB, state string) (int64, error) {
	var count int64
	err := tx.Model(&RunnerTask{}).
		Where("fleet_id = ? AND state = ?", f.ID, state).
		Count(&count).
		Error
	return count, err
}

func (f *RunnerFleet) CountRunnersByState(tx *gorm.DB) (map[string]int64, error) {
	type row struct {
		State string
		Count int64
	}

	var rows []row
	err := tx.Model(&Runner{}).
		Select("state, COUNT(*) AS count").
		Where("fleet_id = ?", f.ID).
		Group("state").
		Scan(&rows).
		Error
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64, len(rows))
	for _, item := range rows {
		counts[item.State] = item.Count
	}
	return counts, nil
}

func (f *RunnerFleet) PinRunnerVersion(tx *gorm.DB, version string) error {
	version = strings.TrimSpace(version)
	if version == "" {
		return errors.New("runner version is required")
	}

	now := time.Now()
	err := tx.Model(f).Updates(map[string]any{
		"runner_version": version,
		"updated_at":     now,
	}).Error
	if err != nil {
		return err
	}

	f.RunnerVersion = version
	f.UpdatedAt = now
	return nil
}

func (f *RunnerFleet) SetEnabled(tx *gorm.DB, enabled bool) error {
	now := time.Now()
	if err := tx.Model(f).Updates(map[string]any{
		"enabled":    enabled,
		"updated_at": now,
	}).Error; err != nil {
		return err
	}

	f.Enabled = enabled
	f.UpdatedAt = now
	return nil
}
