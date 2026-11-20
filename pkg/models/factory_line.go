package models

import (
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	FactoryLineStepTypeRunApp = "runApp"
	FactoryLineStageBacklog   = "backlog"
	FactoryLineStagePhase     = "phase"
	FactoryLineStageCustom    = "custom"
	FactoryLineStageVerify    = "verify"
	FactoryLineStageDone      = "done"

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

type FactoryLineStage struct {
	ID        uuid.UUID `json:"id"`
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	StepIndex *int      `json:"step_index,omitempty"`
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
	Stages         datatypes.JSONSlice[FactoryLineStage]
	// ColumnColors holds the board's column colors, keyed by column key
	// ("backlog", "phase-<step index>", "verify", "done"). A missing key
	// means the column uses the default color.
	ColumnColors datatypes.JSONType[map[string]string]
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (l *FactoryLine) StagesValue() []FactoryLineStage {
	return []FactoryLineStage(l.Stages)
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

// defaultColumnColors returns the board colors a new line starts with,
// matching the colors shown in onboarding: backlog stays unset (neutral),
// the first phase column is sky, verify is yellow, and done is lime. Any
// additional phase columns are left unset.
func defaultColumnColors() map[string]string {
	return map[string]string{
		"phase-0": "sky",
		"verify":  "yellow",
		"done":    "lime",
	}
}

func (f *Factory) CreateLine(tx *gorm.DB, name string, steps []FactoryLineStep) (*FactoryLine, error) {
	stages, err := defaultFactoryLineStages(tx, steps)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	line := &FactoryLine{
		ID:             uuid.New(),
		OrganizationID: f.OrganizationID,
		FactoryID:      f.ID,
		Name:           name,
		Steps:          datatypes.JSONSlice[FactoryLineStep](steps),
		Stages:         datatypes.JSONSlice[FactoryLineStage](stages),
		ColumnColors:   datatypes.NewJSONType(defaultColumnColors()),
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
// execution created by starting it, a failed execution without a run when
// subscription admission is rejected, or a queue item when at capacity.
type FactoryLineStepResult struct {
	Run       *CanvasRun
	Execution *FactoryWorkOrderExecution
	QueueItem *FactoryWorkOrderQueueItem
}

const onRunTriggerName = "onRun"

func (f *Factory) FindLine(tx *gorm.DB, lineID uuid.UUID) (*FactoryLine, error) {
	return f.findLine(tx, lineID)
}

func (f *Factory) FindLineForUpdate(tx *gorm.DB, lineID uuid.UUID) (*FactoryLine, error) {
	return f.findLine(tx.Clauses(clause.Locking{Strength: "UPDATE"}), lineID)
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
	return l.update(tx, name, steps, columnColors, nil)
}

func (l *FactoryLine) UpdateConfiguration(
	tx *gorm.DB,
	name *string,
	steps []FactoryLineStep,
	columnColors map[string]string,
	stages []FactoryLineStage,
) error {
	return l.update(tx, name, steps, columnColors, stages)
}

func (l *FactoryLine) update(
	tx *gorm.DB,
	name *string,
	steps []FactoryLineStep,
	columnColors map[string]string,
	stages []FactoryLineStage,
) error {
	updates := map[string]any{
		"updated_at": time.Now(),
	}
	if name != nil {
		updates["name"] = *name
	}
	if steps != nil {
		updates["steps"] = datatypes.JSONSlice[FactoryLineStep](steps)
		if stages == nil {
			reconciled, err := l.ReconcileStages(tx, steps)
			if err != nil {
				return err
			}
			stages = reconciled
		}
	}
	if columnColors != nil {
		updates["column_colors"] = datatypes.NewJSONType(columnColors)
	}
	if stages != nil {
		if columnColors == nil {
			columnColors = l.ColumnColorsValue()
		}
		columnColors = RetainFactoryLineStageColors(columnColors, stages)
		updates["column_colors"] = datatypes.NewJSONType(columnColors)
		updates["stages"] = datatypes.JSONSlice[FactoryLineStage](stages)
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

func RetainFactoryLineStageColors(colors map[string]string, stages []FactoryLineStage) map[string]string {
	keys := make(map[string]bool, len(stages))
	for _, stage := range stages {
		keys[stage.Key] = true
	}
	result := make(map[string]string, len(colors))
	for key, color := range colors {
		if keys[key] {
			result[key] = color
		}
	}
	return result
}

func (l *FactoryLine) ReconcileStages(tx *gorm.DB, steps []FactoryLineStep) ([]FactoryLineStage, error) {
	systemStages, err := defaultFactoryLineStages(tx, steps)
	if err != nil {
		return nil, err
	}

	existingStages := l.StagesValue()
	if len(existingStages) == 0 {
		return systemStages, nil
	}
	existingByKey := make(map[string]FactoryLineStage, len(existingStages))
	for _, stage := range existingStages {
		if stage.Kind != FactoryLineStageCustom {
			existingByKey[stage.Key] = stage
		}
	}
	systemStages = retainConfiguredOptionalStages(systemStages, existingByKey, len(l.Steps))
	nextByKey := make(map[string]FactoryLineStage, len(systemStages))
	for _, stage := range systemStages {
		nextByKey[stage.Key] = stage
	}
	result := make([]FactoryLineStage, 0, len(systemStages)+len(l.Stages))
	appendNewStages := func() {
		for _, stage := range systemStages {
			if _, ok := nextByKey[stage.Key]; !ok || stage.Kind == FactoryLineStageBacklog || stage.Kind == FactoryLineStageDone {
				continue
			}
			result = append(result, stage)
			delete(nextByKey, stage.Key)
		}
	}
	for _, stage := range existingStages {
		if stage.Kind == FactoryLineStageCustom {
			result = append(result, stage)
			continue
		}
		next, ok := nextByKey[stage.Key]
		if !ok {
			continue
		}
		if stage.Kind == FactoryLineStageDone {
			appendNewStages()
		}
		stage.StepIndex = next.StepIndex
		result = append(result, stage)
		delete(nextByKey, stage.Key)
	}
	appendNewStages()
	return result, nil
}

func retainConfiguredOptionalStages(
	stages []FactoryLineStage,
	existingByKey map[string]FactoryLineStage,
	oldStepCount int,
) []FactoryLineStage {
	result := make([]FactoryLineStage, 0, len(stages))
	for _, stage := range stages {
		if stage.Kind == FactoryLineStageBacklog || stage.Kind == FactoryLineStageDone {
			result = append(result, stage)
			continue
		}
		if _, ok := existingByKey[stage.Key]; ok {
			result = append(result, stage)
			continue
		}
		if stage.Kind == FactoryLineStagePhase && stage.StepIndex != nil && *stage.StepIndex >= oldStepCount {
			result = append(result, stage)
		}
	}
	return result
}

func (l *FactoryLine) DetachRemovedCustomStageAutomations(tx *gorm.DB, next []FactoryLineStage) error {
	retained := make(map[uuid.UUID]bool, len(next))
	for _, stage := range next {
		retained[stage.ID] = true
	}
	removedKeys := make([]string, 0, len(l.Stages))
	for _, stage := range l.StagesValue() {
		if stage.Kind == FactoryLineStageCustom && !retained[stage.ID] {
			removedKeys = append(removedKeys, stage.Key)
		}
	}
	if len(removedKeys) == 0 {
		return nil
	}
	return tx.Model(&Canvas{}).
		Where("factory_id = ? AND column_key IN ?", l.FactoryID, removedKeys).
		Update("column_key", nil).
		Error
}

func (f *Factory) findLine(tx *gorm.DB, lineID uuid.UUID) (*FactoryLine, error) {
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

func defaultFactoryLineStages(tx *gorm.DB, steps []FactoryLineStep) ([]FactoryLineStage, error) {
	names, err := factoryLineStepNames(tx, steps)
	if err != nil {
		return nil, err
	}

	stages := make([]FactoryLineStage, 0, len(steps)+3)
	stages = append(stages, newFactoryLineStage(FactoryLineStageBacklog, "backlog", "Backlog", nil))
	for index, step := range steps {
		stepIndex := index
		name := names[step.AppID]
		if name == "" {
			name = "Step"
		}
		stages = append(stages, newFactoryLineStage(FactoryLineStagePhase, "phase-"+strconv.Itoa(index), name, &stepIndex))
	}
	stages = append(stages,
		newFactoryLineStage(FactoryLineStageVerify, "verify", "Verify", nil),
		newFactoryLineStage(FactoryLineStageDone, "done", "Done", nil),
	)
	return stages, nil
}

func factoryLineStepNames(tx *gorm.DB, steps []FactoryLineStep) (map[uuid.UUID]string, error) {
	ids := make([]uuid.UUID, 0, len(steps))
	for _, step := range steps {
		ids = append(ids, step.AppID)
	}
	if len(ids) == 0 {
		return map[uuid.UUID]string{}, nil
	}

	var canvases []Canvas
	if err := tx.Select("id", "name").Where("id IN ?", ids).Find(&canvases).Error; err != nil {
		return nil, err
	}
	names := make(map[uuid.UUID]string, len(canvases))
	for _, canvas := range canvases {
		names[canvas.ID] = canvas.Name
	}
	return names, nil
}

func newFactoryLineStage(kind, key, name string, stepIndex *int) FactoryLineStage {
	return FactoryLineStage{ID: uuid.New(), Key: key, Name: name, Kind: kind, StepIndex: stepIndex}
}
