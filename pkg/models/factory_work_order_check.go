package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
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

	FactoryWorkOrderCheckFormatFraction = factory.CheckFormatFraction
	FactoryWorkOrderCheckFormatPercent  = factory.CheckFormatPercent

	// MaxFactoryWorkOrderCheckKeyBytes and name match the column widths.
	MaxFactoryWorkOrderCheckKeyBytes  = 255
	MaxFactoryWorkOrderCheckNameBytes = 255

	// MaxFactoryWorkOrderCheckAnalysisBytes caps the markdown analysis,
	// mirroring the artifact data cap.
	MaxFactoryWorkOrderCheckAnalysisBytes = 64 * 1024
)

var ErrFactoryWorkOrderCheckInvalid = errors.New("invalid work order check")

const factoryWorkOrderCheckKeyUniqueConstraint = "idx_factory_work_order_checks_order_key_unique"

// FactoryWorkOrderCheck is latest-only state: one row per (work order,
// key), updated in place on every report. History lives in
// `order.check.reported` timeline events, not in extra rows.
type FactoryWorkOrderCheck struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	WorkOrderID    uuid.UUID
	Key            string
	Name           string
	Score          float64
	MaxScore       float64
	Format         string
	Level          string
	PreviousScore  *float64
	Summary        string
	Analysis       string
	Automation     datatypes.JSON
	RunID          *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (FactoryWorkOrderCheck) TableName() string {
	return "factory_work_order_checks"
}

type FactoryWorkOrderCheckParams struct {
	Key        string
	Name       string
	Score      float64
	MaxScore   float64
	Format     string
	Level      string
	Summary    string
	Analysis   string
	Automation *factory.AutomationRef
	Run        *factory.RunRef
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

// ReportCheck upserts the check row keyed by (work order, key) and
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

func (o *FactoryWorkOrder) ListChecks(tx *gorm.DB) ([]FactoryWorkOrderCheck, error) {
	var checks []FactoryWorkOrderCheck
	err := tx.
		Where("work_order_id = ?", o.ID).
		Order("created_at ASC").
		Order("id ASC").
		Find(&checks).
		Error
	if err != nil {
		return nil, err
	}

	return checks, nil
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
	case FactoryWorkOrderCheckFormatFraction, FactoryWorkOrderCheckFormatPercent:
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
		existing, findErr := o.findCheckByKey(tx, normalized.Key)
		if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}

		now := time.Now()
		if existing != nil {
			previousScore := existing.Score
			existing.Name = normalized.Name
			existing.Score = normalized.Score
			existing.MaxScore = normalized.MaxScore
			existing.Format = normalized.Format
			existing.Level = normalized.Level
			existing.PreviousScore = &previousScore
			existing.Summary = normalized.Summary
			existing.Analysis = normalized.Analysis
			existing.Automation = automationJSON
			existing.RunID = runID
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
				Name:           normalized.Name,
				Score:          normalized.Score,
				MaxScore:       normalized.MaxScore,
				Format:         normalized.Format,
				Level:          normalized.Level,
				Summary:        normalized.Summary,
				Analysis:       normalized.Analysis,
				Automation:     automationJSON,
				RunID:          runID,
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

func (o *FactoryWorkOrder) findCheckByKey(tx *gorm.DB, key string) (*FactoryWorkOrderCheck, error) {
	var check FactoryWorkOrderCheck
	err := tx.
		Where("work_order_id = ? AND key = ?", o.ID, key).
		First(&check).
		Error
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
			Name:          check.Name,
			Score:         check.Score,
			MaxScore:      check.MaxScore,
			Format:        check.Format,
			Level:         check.Level,
			PreviousScore: check.PreviousScore,
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
	params.Name = strings.TrimSpace(params.Name)
	params.Summary = strings.TrimSpace(params.Summary)
	params.Analysis = strings.TrimSpace(params.Analysis)

	if params.Key == "" {
		return params, fmt.Errorf("%w: key is required", ErrFactoryWorkOrderCheckInvalid)
	}
	if len(params.Key) > MaxFactoryWorkOrderCheckKeyBytes {
		return params, fmt.Errorf("%w: key exceeds %d bytes", ErrFactoryWorkOrderCheckInvalid, MaxFactoryWorkOrderCheckKeyBytes)
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
	if !IsValidWorkOrderCheckLevel(params.Level) {
		return params, fmt.Errorf("%w: unknown level %q", ErrFactoryWorkOrderCheckInvalid, params.Level)
	}

	return params, nil
}

func isFiniteCheckNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// isFactoryWorkOrderCheckKeyConflict reports whether err is a violation
// of the per-order unique `key` index, mirroring
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
