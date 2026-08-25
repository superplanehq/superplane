package models

import (
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	FactoryIntakeSourceGitHubIssues       = "github-issues"
	FactoryIntakeSourceSentryExceptions   = "sentry-exceptions"
	FactoryIntakeSourcePagerDutyIncidents = "pagerduty-incidents"

	factoryIntakeCanvasUniqueConstraint = "idx_factory_intakes_canvas_id"
)

var (
	ErrFactoryIntakeNotFound       = errors.New("factory intake not found")
	ErrFactoryIntakeCanvasInUse    = errors.New("canvas already implements a factory intake")
	ErrFactoryIntakeSourceInvalid  = errors.New("factory intake source is not valid")
	ErrFactoryIntakeCanvasRequired = errors.New("factory intake canvas is required")
)

var factoryIntakeSources = []string{
	FactoryIntakeSourceGitHubIssues,
	FactoryIntakeSourceSentryExceptions,
	FactoryIntakeSourcePagerDutyIncidents,
}

// FactoryIntake declares that a factory canvas listens to an external source,
// scores what it receives, and creates backlog work orders. The row owns
// identity; the canvas graph owns behavior.
type FactoryIntake struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	CanvasID       uuid.UUID
	Source         string
	CreatedAt      time.Time
	UpdatedAt      time.Time

	Canvas *Canvas `gorm:"foreignKey:CanvasID"`
}

func ValidFactoryIntakeSource(source string) bool {
	return slices.Contains(factoryIntakeSources, source)
}

func (FactoryIntake) TableName() string {
	return "factory_intakes"
}

// Name is the intake's display name, which is the canvas name. Intakes are
// renamed by renaming their canvas, so the name is never stored twice.
func (i *FactoryIntake) Name() string {
	if i.Canvas == nil {
		return ""
	}
	return i.Canvas.Name
}

func MapFactoryIntakeCanvasUniqueConstraintError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == factoryIntakeCanvasUniqueConstraint {
		return ErrFactoryIntakeCanvasInUse
	}

	return err
}

func (f *Factory) CreateIntake(tx *gorm.DB, canvasID uuid.UUID, source string) (*FactoryIntake, error) {
	if canvasID == uuid.Nil {
		return nil, ErrFactoryIntakeCanvasRequired
	}
	if !ValidFactoryIntakeSource(source) {
		return nil, ErrFactoryIntakeSourceInvalid
	}

	now := time.Now()
	intake := &FactoryIntake{
		ID:             uuid.New(),
		OrganizationID: f.OrganizationID,
		FactoryID:      f.ID,
		CanvasID:       canvasID,
		Source:         source,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := tx.Clauses(clause.Returning{}).Create(intake).Error; err != nil {
		return nil, MapFactoryIntakeCanvasUniqueConstraintError(err)
	}

	return intake, nil
}

func (f *Factory) FindIntake(tx *gorm.DB, intakeID uuid.UUID) (*FactoryIntake, error) {
	var intake FactoryIntake
	err := liveCanvasIntakes(tx).
		Where("factory_intakes.organization_id = ? AND factory_intakes.factory_id = ? AND factory_intakes.id = ?", f.OrganizationID, f.ID, intakeID).
		First(&intake).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryIntakeNotFound
		}
		return nil, err
	}

	return &intake, nil
}

func (f *Factory) ListIntakes(tx *gorm.DB) ([]FactoryIntake, error) {
	var intakes []FactoryIntake
	err := liveCanvasIntakes(tx).
		Where("factory_intakes.organization_id = ? AND factory_intakes.factory_id = ?", f.OrganizationID, f.ID).
		Order("factory_intakes.created_at ASC").
		Order("factory_intakes.id ASC").
		Find(&intakes).
		Error
	if err != nil {
		return nil, err
	}

	return intakes, nil
}

func (i *FactoryIntake) Delete(tx *gorm.DB) error {
	return tx.Where("id = ?", i.ID).Delete(&FactoryIntake{}).Error
}

// DeleteFactoryIntakesByCanvas removes the intakes a canvas implements. The
// canvas reference is a RESTRICT foreign key, so canvas deletion has to call
// this before the canvas row goes away.
func DeleteFactoryIntakesByCanvas(tx *gorm.DB, canvasID uuid.UUID) error {
	return tx.Where("canvas_id = ?", canvasID).Delete(&FactoryIntake{}).Error
}

// liveCanvasIntakes scopes intakes to canvases that still exist. A soft-deleted
// canvas leaves its intake row behind until the cleanup worker hard-deletes the
// canvas, and a deleted intake must not surface in the meantime.
func liveCanvasIntakes(tx *gorm.DB) *gorm.DB {
	return tx.
		Joins("JOIN workflows ON workflows.id = factory_intakes.canvas_id AND workflows.deleted_at IS NULL").
		Preload("Canvas")
}
