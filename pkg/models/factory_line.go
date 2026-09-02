package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	FactoryLineStepTypeRunApp = "runApp"

	// DefaultFactoryLineStepMaxParallelism is the number of in-flight runs
	// a step allows when maxParallelism is not set.
	DefaultFactoryLineStepMaxParallelism = 10

	factoryLineNameUniqueConstraint = "factory_lines_factory_id_name_key"
)

var (
	ErrFactoryLineNotFound          = errors.New("factory line not found")
	ErrFactoryLineNameAlreadyExists = errors.New("factory line name already exists")
	ErrFactoryLineHasNoSteps        = errors.New("factory line has no steps")
	ErrFactoryLineStepNotOnRun      = errors.New("factory line step entrypoint must use the onRun trigger")
	ErrFactoryLineStepOutOfRange    = errors.New("factory line step index is out of range")
)

type FactoryLineStep struct {
	Type       string    `json:"type"`
	AppID      uuid.UUID `json:"app_id"`
	Entrypoint string    `json:"entrypoint"`
	// MaxParallelism caps the step's in-flight runs across all work
	// orders on the line. Unset means the default of 10; there is no
	// unbounded setting.
	MaxParallelism *int `json:"max_parallelism,omitempty"`
}

func (s *FactoryLineStep) EffectiveMaxParallelism() int {
	if s.MaxParallelism == nil || *s.MaxParallelism < 1 {
		return DefaultFactoryLineStepMaxParallelism
	}
	return *s.MaxParallelism
}

type FactoryLine struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	Name           string
	Steps          datatypes.JSONSlice[FactoryLineStep]
	// ColumnColors holds the board's column colors, keyed by column key
	// ("backlog", "phase-<step index>", "verify", "done"). A missing key
	// means the column uses the default color.
	ColumnColors datatypes.JSONType[map[string]string]
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ColumnColorsValue returns the stored column colors, defaulting to an
// empty map when none have been saved yet.
func (l *FactoryLine) ColumnColorsValue() map[string]string {
	colors := l.ColumnColors.Data()
	if colors == nil {
		return map[string]string{}
	}
	return colors
}

func (FactoryLine) TableName() string {
	return "factory_lines"
}

func MapFactoryLineNameUniqueConstraintError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == factoryLineNameUniqueConstraint {
		return ErrFactoryLineNameAlreadyExists
	}

	return err
}

func (f *Factory) CreateLine(tx *gorm.DB, name string, steps []FactoryLineStep) (*FactoryLine, error) {
	now := time.Now()
	line := &FactoryLine{
		ID:             uuid.New(),
		OrganizationID: f.OrganizationID,
		FactoryID:      f.ID,
		Name:           name,
		Steps:          datatypes.JSONSlice[FactoryLineStep](steps),
		ColumnColors:   datatypes.NewJSONType(map[string]string{}),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := tx.Clauses(clause.Returning{}).Create(line).Error; err != nil {
		return nil, MapFactoryLineNameUniqueConstraintError(err)
	}

	return line, nil
}

// FactoryLineStepResult is the outcome of making a work order ready for a
// step (see FactoryLine.Dispatch and
// FactoryWorkOrderLineDispatch.EnqueueOrStartStep): either the run +
// execution created by starting it, or — when the step is at its
// maxParallelism — the queue item created instead (Run and Execution nil).
type FactoryLineStepResult struct {
	Run       *CanvasRun
	Execution *FactoryWorkOrderExecution
	QueueItem *FactoryWorkOrderQueueItem
}

const onRunTriggerName = "onRun"

func (f *Factory) FindLine(tx *gorm.DB, lineID uuid.UUID) (*FactoryLine, error) {
	var line FactoryLine
	err := tx.
		Where("organization_id = ? AND factory_id = ? AND id = ?", f.OrganizationID, f.ID, lineID).
		First(&line).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryLineNotFound
		}
		return nil, err
	}

	return &line, nil
}

func (f *Factory) FindLineByName(tx *gorm.DB, name string) (*FactoryLine, error) {
	var line FactoryLine
	err := tx.
		Where("organization_id = ? AND factory_id = ? AND name = ?", f.OrganizationID, f.ID, name).
		First(&line).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryLineNotFound
		}
		return nil, err
	}

	return &line, nil
}

func (f *Factory) ListLines(tx *gorm.DB) ([]FactoryLine, error) {
	var lines []FactoryLine
	err := tx.
		Where("organization_id = ? AND factory_id = ?", f.OrganizationID, f.ID).
		Order("name ASC").
		Order("id ASC").
		Find(&lines).
		Error
	if err != nil {
		return nil, err
	}

	return lines, nil
}

func ListFactoryLinesByFactoryIDs(tx *gorm.DB, organizationID uuid.UUID, factoryIDs []uuid.UUID) ([]FactoryLine, error) {
	if len(factoryIDs) == 0 {
		return nil, nil
	}

	var lines []FactoryLine
	err := tx.
		Where("organization_id = ? AND factory_id IN ?", organizationID, factoryIDs).
		Order("name ASC").
		Order("id ASC").
		Find(&lines).
		Error
	if err != nil {
		return nil, err
	}

	return lines, nil
}

// Update persists the given fields. A nil name, steps, or columnColors
// means "do not change" that field; an empty (non-nil) steps slice is not
// valid (callers must not pass one), while an empty (non-nil) columnColors
// map is a valid "clear all colors" request.
func (l *FactoryLine) Update(tx *gorm.DB, name *string, steps []FactoryLineStep, columnColors map[string]string) error {
	updates := map[string]any{
		"updated_at": time.Now(),
	}
	if name != nil {
		updates["name"] = *name
	}
	if steps != nil {
		updates["steps"] = datatypes.JSONSlice[FactoryLineStep](steps)
	}
	if columnColors != nil {
		updates["column_colors"] = datatypes.NewJSONType(columnColors)
	}

	err := tx.Model(l).Updates(updates).Error
	if err != nil {
		return MapFactoryLineNameUniqueConstraintError(err)
	}

	return tx.
		Where("id = ?", l.ID).
		First(l).
		Error
}
