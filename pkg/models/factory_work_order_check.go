package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	FactoryWorkOrderCheckLevelPositive = factory.CheckLevelPositive
	FactoryWorkOrderCheckLevelNeutral  = factory.CheckLevelNeutral
	FactoryWorkOrderCheckLevelCaution  = factory.CheckLevelCaution
	FactoryWorkOrderCheckLevelCritical = factory.CheckLevelCritical

	// FactoryWorkOrderCheckSourceAutomation marks a check from a plan,
	// a merge confidence run, or a canvas component. An agent cannot
	// replace or remove it.
	FactoryWorkOrderCheckSourceAutomation = "automation"
	// FactoryWorkOrderCheckSourceAgent marks a check from add_checks.
	FactoryWorkOrderCheckSourceAgent = "agent"

	FactoryWorkOrderCheckFormatFraction = factory.CheckFormatFraction
	FactoryWorkOrderCheckFormatPercent  = factory.CheckFormatPercent
	FactoryWorkOrderCheckFormatBoolean  = factory.CheckFormatBoolean

	// MaxFactoryWorkOrderCheckKeyBytes, name, and repository match the column widths.
	MaxFactoryWorkOrderCheckKeyBytes        = 255
	MaxFactoryWorkOrderCheckNameBytes       = 255
	MaxFactoryWorkOrderCheckRepositoryBytes = 255

	// MaxFactoryWorkOrderCheckAnalysisBytes caps the markdown analysis,
	// mirroring the artifact data cap.
	MaxFactoryWorkOrderCheckAnalysisBytes = 64 * 1024

	// MaxFactoryWorkOrderCheckRecentScores caps the per-check score
	// history kept on the row; older entries fall off the front. Full
	// history stays in `order.check.reported` timeline events.
	MaxFactoryWorkOrderCheckRecentScores = 10
)

var (
	ErrFactoryWorkOrderCheckInvalid = errors.New("invalid work order check")
	// ErrFactoryWorkOrderCheckOwned rejects an agent check whose key
	// belongs to a check that an agent did not add.
	ErrFactoryWorkOrderCheckOwned = errors.New("check key belongs to SuperPlane")
)

const factoryWorkOrderCheckKeyUniqueConstraint = "idx_factory_work_order_checks_order_repository_key_unique"

// mergeConfidenceCheckKeySuffix ends every merge confidence check key.
const mergeConfidenceCheckKeySuffix = "-review"

// IsReservedWorkOrderCheckKey reports whether a key belongs to a plan score
// or a merge confidence check. Planning reads those keys to decide whether
// a task is ready, so an agent check cannot use them, even before the first
// plan or merge confidence report.
func IsReservedWorkOrderCheckKey(key string) bool {
	key = strings.TrimSpace(key)
	return slices.Contains(planningReviewCheckKeys, key) || strings.HasSuffix(key, mergeConfidenceCheckKeySuffix)
}

var planningReviewCheckKeys = []string{
	PlanningClarityCheckKey,
	PlanningConfidenceCheckKey,
	PlanningComplexityCheckKey,
	PlanningVerifiabilityCheckKey,
}

// FactoryWorkOrderCheck is latest-only state: one row per (work order,
// repository, key), updated in place on every report. History lives in
// `order.check.reported` timeline events, not in extra rows.
type FactoryWorkOrderCheck struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	WorkOrderID    uuid.UUID
	Key            string
	// Repository is the pull request repository of a merge confidence
	// check. It is empty for a check of the whole task.
	Repository    string
	Name          string
	Score         float64
	MaxScore      float64
	Format        string
	Level         string
	PreviousScore *float64
	// RecentScores holds the latest report scores, oldest first and
	// ending with Score, capped at MaxFactoryWorkOrderCheckRecentScores.
	RecentScores datatypes.JSONSlice[float64]
	Summary      string
	Analysis     string
	Automation   datatypes.JSON
	RunID        *uuid.UUID
	Source       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (FactoryWorkOrderCheck) TableName() string {
	return "factory_work_order_checks"
}

type FactoryWorkOrderCheckParams struct {
	Key        string
	Repository string
	Name       string
	Score      float64
	MaxScore   float64
	Format     string
	Level      string
	Summary    string
	Analysis   string
	Automation *factory.AutomationRef
	Run        *factory.RunRef
	// Source defaults to FactoryWorkOrderCheckSourceAutomation.
	Source string
}

// AutomationRef decodes the automation snapshot stored on the row.
// Returns nil when the check has no automation attribution.
func (c *FactoryWorkOrderCheck) AutomationRef() (*factory.AutomationRef, error) {
	if len(c.Automation) == 0 {
		return nil, nil
	}

	var ref factory.AutomationRef
	if err := json.Unmarshal(c.Automation, &ref); err != nil {
		return nil, err
	}

	return &ref, nil
}

// ReportCheck upserts the check row keyed by (work order, repository, key) and
// records an `order.check.reported` event in the same transaction. A
// re-report of an existing key keeps that report's score as
// PreviousScore so the UI can show the movement.
func (o *FactoryWorkOrder) ReportCheck(
	db *gorm.DB,
	params FactoryWorkOrderCheckParams,
) (*FactoryWorkOrderCheck, error) {
	normalized, err := normalizeCheckParams(params)
	if err != nil {
		return nil, err
	}

	automationJSON, err := encodeCheckAutomation(normalized.Automation)
	if err != nil {
		return nil, err
	}

	var runID *uuid.UUID
	if normalized.Run != nil {
		id := normalized.Run.ID
		runID = &id
	}

	check, err := o.reportCheck(db, normalized, automationJSON, runID)
	if isFactoryWorkOrderCheckKeyConflict(err) {
		// Two first reports of the same key raced on insert. Postgres
		// holds the losing INSERT until the winner commits, so by the
		// time the unique violation surfaces the row is visible — a
		// single retry lands as the in-place update it should have been.
		check, err = o.reportCheck(db, normalized, automationJSON, runID)
	}
	if err != nil {
		return nil, err
	}

	return check, nil
}

// ReportAgentChecks saves the checks of one add_checks call. All checks
// save, or none do. A reserved key or the key of a non-agent check returns
// ErrFactoryWorkOrderCheckOwned.
func (o *FactoryWorkOrder) ReportAgentChecks(
	db *gorm.DB,
	params []FactoryWorkOrderCheckParams,
) ([]FactoryWorkOrderCheck, error) {
	checks := make([]FactoryWorkOrderCheck, 0, len(params))
	err := db.Transaction(func(tx *gorm.DB) error {
		for _, param := range params {
			if IsReservedWorkOrderCheckKey(param.Key) {
				return fmt.Errorf("%w: %s", ErrFactoryWorkOrderCheckOwned, strings.TrimSpace(param.Key))
			}
			if isPlanningScoreCheckName(param.Name) {
				return fmt.Errorf("%w: name %q belongs to SuperPlane; use another name", ErrFactoryWorkOrderCheckInvalid, strings.TrimSpace(param.Name))
			}
			param.Source = FactoryWorkOrderCheckSourceAgent
			check, err := o.ReportCheck(tx, param)
			if err != nil {
				return err
			}
			checks = append(checks, *check)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return checks, nil
}

// RemoveAgentChecks deletes the agent checks with the given keys and
// returns the removed keys. Other checks stay.
func (o *FactoryWorkOrder) RemoveAgentChecks(db *gorm.DB, keys []string) ([]string, error) {
	removed := []string{}
	if len(keys) == 0 {
		return removed, nil
	}

	var deleted []FactoryWorkOrderCheck
	err := db.
		Clauses(clause.Returning{Columns: []clause.Column{{Name: "key"}}}).
		Where("work_order_id = ? AND source = ? AND key IN ?", o.ID, FactoryWorkOrderCheckSourceAgent, keys).
		Delete(&deleted).
		Error
	if err != nil {
		return nil, err
	}

	for _, check := range deleted {
		removed = append(removed, check.Key)
	}
	return removed, nil
}

func (o *FactoryWorkOrder) ListChecks(tx *gorm.DB) ([]FactoryWorkOrderCheck, error) {
	grouped, err := ListChecksForWorkOrders(tx, []uuid.UUID{o.ID})
	if err != nil {
		return nil, err
	}
	return grouped[o.ID], nil
}

// ListChecksForWorkOrders loads the latest check row per key for each work
// order. Order within an order matches ListChecks: first report first.
func ListChecksForWorkOrders(tx *gorm.DB, workOrderIDs []uuid.UUID) (map[uuid.UUID][]FactoryWorkOrderCheck, error) {
	grouped := make(map[uuid.UUID][]FactoryWorkOrderCheck, len(workOrderIDs))
	if len(workOrderIDs) == 0 {
		return grouped, nil
	}

	var checks []FactoryWorkOrderCheck
	err := tx.
		Where("work_order_id IN ?", workOrderIDs).
		Order("created_at ASC").
		Order("id ASC").
		Find(&checks).
		Error
	if err != nil {
		return nil, err
	}

	for i := range checks {
		workOrderID := checks[i].WorkOrderID
		grouped[workOrderID] = append(grouped[workOrderID], checks[i])
	}
	return grouped, nil
}

// IsValidWorkOrderCheckLevel reports whether ReportCheck accepts the level.
func IsValidWorkOrderCheckLevel(level string) bool {
	switch level {
	case FactoryWorkOrderCheckLevelPositive,
		FactoryWorkOrderCheckLevelNeutral,
		FactoryWorkOrderCheckLevelCaution,
		FactoryWorkOrderCheckLevelCritical:
		return true
	}
	return false
}

// IsValidWorkOrderCheckFormat reports whether ReportCheck accepts the format.
func IsValidWorkOrderCheckFormat(format string) bool {
	switch format {
	case FactoryWorkOrderCheckFormatFraction,
		FactoryWorkOrderCheckFormatPercent,
		FactoryWorkOrderCheckFormatBoolean:
		return true
	}
	return false
}

func (o *FactoryWorkOrder) reportCheck(
	db *gorm.DB,
	normalized FactoryWorkOrderCheckParams,
	automationJSON datatypes.JSON,
	runID *uuid.UUID,
) (*FactoryWorkOrderCheck, error) {
	var check *FactoryWorkOrderCheck
	err := db.Transaction(func(tx *gorm.DB) error {
		existing, findErr := o.findCheck(tx, normalized.Repository, normalized.Key)
		if findErr != nil {
			return findErr
		}

		if existing != nil && normalized.Source == FactoryWorkOrderCheckSourceAgent && existing.Source != FactoryWorkOrderCheckSourceAgent {
			return fmt.Errorf("%w: %s", ErrFactoryWorkOrderCheckOwned, normalized.Key)
		}

		now := time.Now()
		if existing != nil {
			previousScore := existing.Score
			existing.Repository = normalized.Repository
			existing.Name = normalized.Name
			existing.Score = normalized.Score
			existing.MaxScore = normalized.MaxScore
			existing.Format = normalized.Format
			existing.Level = normalized.Level
			existing.PreviousScore = &previousScore
			existing.RecentScores = appendRecentCheckScore(existing.RecentScores, previousScore, normalized.Score)
			existing.Summary = normalized.Summary
			existing.Analysis = normalized.Analysis
			existing.Automation = automationJSON
			existing.RunID = runID
			existing.Source = normalized.Source
			existing.UpdatedAt = now
			if saveErr := tx.Save(existing).Error; saveErr != nil {
				return saveErr
			}
			check = existing
		} else {
			created := &FactoryWorkOrderCheck{
				ID:             uuid.New(),
				OrganizationID: o.OrganizationID,
				FactoryID:      o.FactoryID,
				WorkOrderID:    o.ID,
				Key:            normalized.Key,
				Repository:     normalized.Repository,
				Name:           normalized.Name,
				Score:          normalized.Score,
				MaxScore:       normalized.MaxScore,
				Format:         normalized.Format,
				Level:          normalized.Level,
				RecentScores:   datatypes.NewJSONSlice([]float64{normalized.Score}),
				Summary:        normalized.Summary,
				Analysis:       normalized.Analysis,
				Automation:     automationJSON,
				RunID:          runID,
				Source:         normalized.Source,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if createErr := tx.Clauses(clause.Returning{}).Create(created).Error; createErr != nil {
				return createErr
			}
			check = created
		}

		return o.recordCheckReported(tx, check, normalized.Automation, normalized.Run)
	})
	if err != nil {
		return nil, err
	}

	return check, nil
}

// findCheck returns the row a report updates, or nil for a first report.
// A row without a repository is from before checks had one. The first
// report with a repository takes it over, so the old score does not stay
// beside the new one.
func (o *FactoryWorkOrder) findCheck(tx *gorm.DB, repository, key string) (*FactoryWorkOrderCheck, error) {
	check, err := o.lockCheck(tx, repository, key)
	if err != nil || check != nil || repository == "" {
		return check, err
	}

	legacy, err := o.lockCheck(tx, "", key)
	if err != nil || legacy == nil || legacy.Source != FactoryWorkOrderCheckSourceAutomation {
		return nil, err
	}
	return legacy, nil
}

func (o *FactoryWorkOrder) lockCheck(tx *gorm.DB, repository, key string) (*FactoryWorkOrderCheck, error) {
	var check FactoryWorkOrderCheck
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("work_order_id = ? AND repository = ? AND key = ?", o.ID, repository, key).
		First(&check).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &check, nil
}

func (o *FactoryWorkOrder) recordCheckReported(
	tx *gorm.DB,
	check *FactoryWorkOrderCheck,
	automation *factory.AutomationRef,
	run *factory.RunRef,
) error {
	data := factory.WorkOrderCheckReported{
		Order: o.Ref(),
		Check: &factory.CheckRef{
			ID:            check.ID,
			Key:           check.Key,
			Repository:    check.Repository,
			Name:          check.Name,
			Score:         check.Score,
			MaxScore:      check.MaxScore,
			Format:        check.Format,
			Level:         check.Level,
			PreviousScore: check.PreviousScore,
			Summary:       check.Summary,
		},
		Automation: automation,
		Run:        run,
	}

	return o.recordEvent(tx, factory.EventTypeOrderCheckReported, data)
}

// normalizeCheckParams trims, defaults, and validates the report before
// it touches the database, so a bad component payload fails loudly
// instead of landing a half-formed check.
func normalizeCheckParams(params FactoryWorkOrderCheckParams) (FactoryWorkOrderCheckParams, error) {
	params.Key = strings.TrimSpace(params.Key)
	params.Repository = strings.TrimSpace(params.Repository)
	params.Name = strings.TrimSpace(params.Name)
	params.Summary = strings.TrimSpace(params.Summary)
	params.Analysis = strings.TrimSpace(params.Analysis)

	if params.Key == "" {
		return params, fmt.Errorf("%w: key is required", ErrFactoryWorkOrderCheckInvalid)
	}
	if len(params.Key) > MaxFactoryWorkOrderCheckKeyBytes {
		return params, fmt.Errorf("%w: key exceeds %d bytes", ErrFactoryWorkOrderCheckInvalid, MaxFactoryWorkOrderCheckKeyBytes)
	}
	if len(params.Repository) > MaxFactoryWorkOrderCheckRepositoryBytes {
		return params, fmt.Errorf("%w: repository exceeds %d bytes", ErrFactoryWorkOrderCheckInvalid, MaxFactoryWorkOrderCheckRepositoryBytes)
	}
	if params.Name == "" {
		return params, fmt.Errorf("%w: name is required", ErrFactoryWorkOrderCheckInvalid)
	}
	if len(params.Name) > MaxFactoryWorkOrderCheckNameBytes {
		return params, fmt.Errorf("%w: name exceeds %d bytes", ErrFactoryWorkOrderCheckInvalid, MaxFactoryWorkOrderCheckNameBytes)
	}
	if len(params.Analysis) > MaxFactoryWorkOrderCheckAnalysisBytes {
		return params, fmt.Errorf(
			"%w: analysis exceeds %d bytes",
			ErrFactoryWorkOrderCheckInvalid,
			MaxFactoryWorkOrderCheckAnalysisBytes,
		)
	}

	if !isFiniteCheckNumber(params.Score) || params.Score < 0 {
		return params, fmt.Errorf("%w: score must be a non-negative number", ErrFactoryWorkOrderCheckInvalid)
	}
	if !isFiniteCheckNumber(params.MaxScore) || params.MaxScore <= 0 {
		return params, fmt.Errorf("%w: maxScore must be a positive number", ErrFactoryWorkOrderCheckInvalid)
	}

	if params.Format == "" {
		params.Format = FactoryWorkOrderCheckFormatFraction
	}
	if !IsValidWorkOrderCheckFormat(params.Format) {
		return params, fmt.Errorf("%w: unknown format %q", ErrFactoryWorkOrderCheckInvalid, params.Format)
	}

	if params.Level == "" {
		params.Level = FactoryWorkOrderCheckLevelNeutral
	}
	if params.Source == "" {
		params.Source = FactoryWorkOrderCheckSourceAutomation
	}
	if params.Source != FactoryWorkOrderCheckSourceAutomation && params.Source != FactoryWorkOrderCheckSourceAgent {
		return params, fmt.Errorf("%w: unknown source %q", ErrFactoryWorkOrderCheckInvalid, params.Source)
	}
	if !IsValidWorkOrderCheckLevel(params.Level) {
		return params, fmt.Errorf("%w: unknown level %q", ErrFactoryWorkOrderCheckInvalid, params.Level)
	}

	// A boolean check is a verdict, not a scale — pin it to 0/1 so the
	// data cannot drift into "0.7 out of 1".
	if params.Format == FactoryWorkOrderCheckFormatBoolean {
		if params.Score != 0 && params.Score != 1 {
			return params, fmt.Errorf("%w: boolean score must be 0 or 1", ErrFactoryWorkOrderCheckInvalid)
		}
		if params.MaxScore != 1 {
			return params, fmt.Errorf("%w: boolean maxScore must be 1", ErrFactoryWorkOrderCheckInvalid)
		}
	}

	return params, nil
}

// appendRecentCheckScore extends the capped score history with the new
// report. Rows created before the history column existed have an empty
// slice; seed it with the score they held so the history does not start
// mid-stream.
func appendRecentCheckScore(
	history datatypes.JSONSlice[float64],
	previousScore float64,
	score float64,
) datatypes.JSONSlice[float64] {
	scores := []float64(history)
	if len(scores) == 0 {
		scores = []float64{previousScore}
	}

	scores = append(scores, score)
	if overflow := len(scores) - MaxFactoryWorkOrderCheckRecentScores; overflow > 0 {
		scores = scores[overflow:]
	}

	return datatypes.NewJSONSlice(scores)
}

func isFiniteCheckNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// isFactoryWorkOrderCheckKeyConflict reports whether err is a violation
// of the per-order unique (repository, key) index, mirroring
// MapFactoryWorkOrderArtifactKeyUniqueConstraintError.
func isFactoryWorkOrderCheckKeyConflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.ConstraintName == factoryWorkOrderCheckKeyUniqueConstraint
}

func encodeCheckAutomation(ref *factory.AutomationRef) (datatypes.JSON, error) {
	if ref == nil {
		return nil, nil
	}

	encoded, err := json.Marshal(ref)
	if err != nil {
		return nil, err
	}

	return datatypes.JSON(encoded), nil
}

// isPlanningScoreCheckName reports whether a name is a plan score name. The
// UI finds plan scores by name, so an agent check cannot use these names.
func isPlanningScoreCheckName(name string) bool {
	name = strings.TrimSpace(name)
	return strings.EqualFold(name, PlanningClarityCheckName) || strings.EqualFold(name, PlanningConfidenceCheckName)
}
