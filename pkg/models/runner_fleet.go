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
	OperatingSystem string   `json:"operating_system,omitempty"`
	Architecture    string   `json:"architecture,omitempty"`
	CPUMillicores   int32    `json:"cpu_millicores,omitempty"`
	MemoryMB        int32    `json:"memory_mb,omitempty"`
	DiskGB          int32    `json:"disk_gb,omitempty"`
	Capabilities    []string `json:"capabilities,omitempty"`
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

// MetricScope distinguishes installation fleets from organization fleets that share a slug.
func (f *RunnerFleet) MetricScope(tx *gorm.DB) (string, error) {
	if f.ScopeType != RunnerFleetScopeOrganization {
		return f.ScopeType, nil
	}
	if f.ScopeID == nil {
		return "", errors.New("organization fleet has no scope ID")
	}
	var organization Organization
	if err := tx.Unscoped().Select("slug").First(&organization, "id = ?", *f.ScopeID).Error; err != nil {
		return "", err
	}
	if organization.Slug == "" {
		return "", errors.New("organization fleet has no organization slug")
	}
	return RunnerFleetMetricScope(f.ScopeType, organization.Slug), nil
}

// RunnerFleetMetricScope identifies a fleet's installation or organization scope.
func RunnerFleetMetricScope(scopeType, organizationSlug string) string {
	if scopeType == RunnerFleetScopeOrganization {
		if organizationSlug == "" {
			return ""
		}
		return scopeType + "/" + organizationSlug
	}
	return scopeType
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
			OperatingSystem: "linux",
			Architecture:    architecture,
			CPUMillicores:   cpuMillicores,
			MemoryMB:        memoryMB,
			DiskGB:          30,
			Capabilities:    []string{"docker"},
		}),
		CreatedAt: now,
		UpdatedAt: now,
	}
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

type ListPage struct {
	States  []string
	Limit   int
	AfterID *uuid.UUID
}

type RunnerListPage struct {
	Runners     []Runner
	TotalCount  int64
	HasNextPage bool
}

type TaskListPage struct {
	Tasks       []RunnerTask
	TotalCount  int64
	HasNextPage bool
}

type FleetStateCount struct {
	FleetID    uuid.UUID
	FleetSlug  string
	FleetScope string
	State      string
	Count      int64
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

func (f *RunnerFleet) ListRunners(tx *gorm.DB, page ListPage) (RunnerListPage, error) {
	total, err := countFleetRecords(tx, &Runner{}, f.ID, page.States)
	if err != nil {
		return RunnerListPage{}, err
	}

	query := tx.Where("fleet_id = ?", f.ID)
	if len(page.States) > 0 {
		query = query.Where("state IN ?", page.States)
	}
	if page.AfterID != nil {
		cursor, found, cursorErr := f.runnerListCursor(tx, *page.AfterID)
		if cursorErr != nil {
			return RunnerListPage{}, cursorErr
		}
		if !found {
			return RunnerListPage{Runners: []Runner{}, TotalCount: total}, nil
		}
		query = query.Where("(created_at, id) > (?, ?)", cursor.CreatedAt, cursor.ID)
	}

	query = query.Order("created_at ASC, id ASC")
	if page.Limit > 0 {
		query = query.Limit(page.Limit + 1)
	}

	var runners []Runner
	if err := query.Find(&runners).Error; err != nil {
		return RunnerListPage{}, err
	}
	runners, hasNext := trimListPage(runners, page.Limit)
	return RunnerListPage{Runners: runners, TotalCount: total, HasNextPage: hasNext}, nil
}

func (f *RunnerFleet) ListTasks(tx *gorm.DB, page ListPage) (TaskListPage, error) {
	total, err := countFleetRecords(tx, &RunnerTask{}, f.ID, page.States)
	if err != nil {
		return TaskListPage{}, err
	}

	query := tx.Where("fleet_id = ?", f.ID)
	if len(page.States) > 0 {
		query = query.Where("state IN ?", page.States)
	}
	if page.AfterID != nil {
		cursor, found, cursorErr := f.taskListCursor(tx, *page.AfterID)
		if cursorErr != nil {
			return TaskListPage{}, cursorErr
		}
		if !found {
			return TaskListPage{Tasks: []RunnerTask{}, TotalCount: total}, nil
		}
		query = query.Where("(queued_at, id) > (?, ?)", cursor.QueuedAt, cursor.ID)
	}

	query = query.Order("queued_at ASC, id ASC")
	if page.Limit > 0 {
		query = query.Limit(page.Limit + 1)
	}

	var tasks []RunnerTask
	if err := query.Find(&tasks).Error; err != nil {
		return TaskListPage{}, err
	}
	tasks, hasNext := trimListPage(tasks, page.Limit)
	return TaskListPage{Tasks: tasks, TotalCount: total, HasNextPage: hasNext}, nil
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

func ListRunnerCountsByFleetState(tx *gorm.DB) ([]FleetStateCount, error) {
	return listCountsByFleetState(tx, &Runner{}, []string{
		RunnerStatePending,
		RunnerStateIdle,
		RunnerStateBusy,
	})
}

func ListRunnerTaskCountsByFleetState(tx *gorm.DB) ([]FleetStateCount, error) {
	return listCountsByFleetState(tx, &RunnerTask{}, []string{
		RunnerTaskStateQueued,
		RunnerTaskStateReserved,
		RunnerTaskStateRunning,
	})
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

func (f *RunnerFleet) runnerListCursor(tx *gorm.DB, id uuid.UUID) (*Runner, bool, error) {
	var cursor Runner
	err := tx.Select("id", "created_at").Where("fleet_id = ? AND id = ?", f.ID, id).Take(&cursor).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &cursor, true, nil
}

func (f *RunnerFleet) taskListCursor(tx *gorm.DB, id uuid.UUID) (*RunnerTask, bool, error) {
	var cursor RunnerTask
	err := tx.Select("id", "queued_at").Where("fleet_id = ? AND id = ?", f.ID, id).Take(&cursor).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &cursor, true, nil
}

func countFleetRecords(tx *gorm.DB, model any, fleetID uuid.UUID, states []string) (int64, error) {
	query := tx.Model(model).Where("fleet_id = ?", fleetID)
	if len(states) > 0 {
		query = query.Where("state IN ?", states)
	}
	var count int64
	err := query.Count(&count).Error
	return count, err
}

func trimListPage[T any](rows []T, limit int) ([]T, bool) {
	if rows == nil {
		return []T{}, false
	}
	if limit > 0 && len(rows) > limit {
		return rows[:limit], true
	}
	return rows, false
}

func listCountsByFleetState(tx *gorm.DB, model any, states []string) ([]FleetStateCount, error) {
	type fleetRow struct {
		ID               uuid.UUID
		Slug             string
		ScopeType        string
		OrganizationSlug string
	}

	var fleets []fleetRow
	if err := tx.Model(&RunnerFleet{}).
		Select("runner_fleets.id, runner_fleets.slug, runner_fleets.scope_type, organizations.slug AS organization_slug").
		Joins("LEFT JOIN organizations ON organizations.id = runner_fleets.scope_id").
		Order("runner_fleets.slug ASC, runner_fleets.id ASC").
		Scan(&fleets).
		Error; err != nil {
		return nil, err
	}
	if len(fleets) == 0 {
		return []FleetStateCount{}, nil
	}

	fleetIDs := make([]uuid.UUID, 0, len(fleets))
	for _, fleet := range fleets {
		fleetIDs = append(fleetIDs, fleet.ID)
	}

	type countRow struct {
		FleetID uuid.UUID
		State   string
		Count   int64
	}

	var rows []countRow
	if err := tx.Model(model).
		Select("fleet_id, state, COUNT(*) AS count").
		Where("fleet_id IN ? AND state IN ?", fleetIDs, states).
		Group("fleet_id, state").
		Scan(&rows).
		Error; err != nil {
		return nil, err
	}

	type key struct {
		FleetID uuid.UUID
		State   string
	}
	counts := make(map[key]int64, len(rows))
	for _, row := range rows {
		counts[key{FleetID: row.FleetID, State: row.State}] = row.Count
	}

	result := make([]FleetStateCount, 0, len(fleets)*len(states))
	for _, fleet := range fleets {
		for _, state := range states {
			result = append(result, FleetStateCount{
				FleetID:    fleet.ID,
				FleetSlug:  fleet.Slug,
				FleetScope: RunnerFleetMetricScope(fleet.ScopeType, fleet.OrganizationSlug),
				State:      state,
				Count:      counts[key{FleetID: fleet.ID, State: state}],
			})
		}
	}
	return result, nil
}
