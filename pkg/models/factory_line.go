package models

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
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
	StageKey   string    `json:"stage_key,omitempty"`
	StageName  string    `json:"stage_name,omitempty"`
	StageKind  string    `json:"stage_kind,omitempty"`
	// MaxParallelism caps the step's in-flight runs across all work
	// orders on the line. Unset means the default of 10; there is no
	// unbounded setting.
	MaxParallelism *int `json:"max_parallelism,omitempty"`
}

type FactoryLineStageAutomationError struct {
	AutomationName string
	Reason         string
}

func (e *FactoryLineStageAutomationError) Error() string {
	return fmt.Sprintf("Automation %q %s", e.AutomationName, e.Reason)
}

type FactoryLineStage struct {
	ID              uuid.UUID `json:"id"`
	Key             string    `json:"key"`
	Name            string    `json:"name"`
	Kind            string    `json:"kind"`
	StepIndex       *int      `json:"step_index,omitempty"`
	AgentDisplayKey string    `json:"agent_display_key,omitempty"`
	AutoAdvance     *bool     `json:"auto_advance,omitempty"`
}

func (s FactoryLineStage) AutoAdvanceEnabled() bool {
	if s.AutoAdvance != nil {
		return *s.AutoAdvance
	}
	return true
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

func (l *FactoryLine) executionStages(tx *gorm.DB) ([]FactoryLineStage, error) {
	stages := l.StagesValue()
	if len(stages) > 0 {
		return stages, nil
	}
	return defaultFactoryLineStages(tx, l.Steps)
}

func (l *FactoryLine) ExecutionPlan(tx *gorm.DB) ([]FactoryLineStep, error) {
	var canvases []Canvas
	if err := tx.
		Where("factory_id = ? AND column_key IS NOT NULL", l.FactoryID).
		Order("created_at ASC").
		Order("id ASC").
		Find(&canvases).Error; err != nil {
		return nil, err
	}

	canvasIDs := make([]uuid.UUID, 0, len(canvases))
	canvasesByStage := make(map[string][]Canvas)
	for _, canvas := range canvases {
		if canvas.ColumnKey == nil || !strings.HasPrefix(*canvas.ColumnKey, "custom-") {
			continue
		}
		canvasIDs = append(canvasIDs, canvas.ID)
		canvasesByStage[*canvas.ColumnKey] = append(canvasesByStage[*canvas.ColumnKey], canvas)
	}
	specs, err := FindLiveCanvasSpecsByCanvasIDs(tx, canvasIDs)
	if err != nil {
		return nil, err
	}

	stages, err := l.executionStages(tx)
	if err != nil {
		return nil, err
	}
	steps := make([]FactoryLineStep, 0, len(l.Steps)+len(canvases))
	for _, stage := range stages {
		if stage.Kind != FactoryLineStagePhase {
			continue
		}
		if stage.StepIndex != nil {
			if *stage.StepIndex < 0 || *stage.StepIndex >= len(l.Steps) {
				continue
			}
			step := l.Steps[*stage.StepIndex]
			step.StageKey = stage.Key
			step.StageName = stage.Name
			step.StageKind = stage.Kind
			steps = append(steps, step)
		}
		if stage.StepIndex == nil {
			for _, canvas := range canvasesByStage[stage.Key] {
				entrypoint, err := customStageEntrypoint(canvas, specs[canvas.ID])
				if err != nil {
					return nil, err
				}
				if entrypoint == "" {
					continue
				}
				steps = append(steps, FactoryLineStep{
					Type:       FactoryLineStepTypeRunApp,
					AppID:      canvas.ID,
					Entrypoint: entrypoint,
					StageKey:   stage.Key,
					StageName:  stage.Name,
					StageKind:  stage.Kind,
				})
			}
		}
	}

	if len(steps) == 0 {
		return nil, ErrFactoryLineHasNoSteps
	}
	return steps, nil
}

func customStageEntrypoint(canvas Canvas, spec LiveCanvasSpec) (string, error) {
	if canvas.LiveVersionID == nil {
		return "", &FactoryLineStageAutomationError{
			AutomationName: canvas.Name,
			Reason:         "must be published before this task can start.",
		}
	}
	entrypoint := ""
	hasWebhook := false
	for _, node := range spec.Nodes {
		if node.Ref.Trigger != nil && node.Ref.Trigger.Name == onWebhookTriggerName {
			hasWebhook = true
		}
		if node.Ref.Trigger == nil || node.Ref.Trigger.Name != onRunTriggerName {
			continue
		}
		if entrypoint != "" {
			return "", &FactoryLineStageAutomationError{
				AutomationName: canvas.Name,
				Reason:         "must have one On Run trigger before this task can start.",
			}
		}
		entrypoint = node.ID
	}
	// A webhook-only stage automation adds no step. Tasks wait in the stage
	// when auto advance is off, and a webhook run moves them.
	if entrypoint == "" && !hasWebhook {
		return "", &FactoryLineStageAutomationError{
			AutomationName: canvas.Name,
			Reason:         "must have one On Run trigger before this task can start.",
		}
	}
	return entrypoint, nil
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
const onWebhookTriggerName = "onWebhook"

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
	usedKeys := make(map[string]bool, len(existingStages))
	for _, stage := range existingStages {
		existingByKey[stage.Key] = stage
		usedKeys[stage.Key] = true
	}
	// A deleted phase compacts steps without changing retained stage keys.
	for i := range systemStages {
		next := &systemStages[i]
		if next.StepIndex == nil {
			continue
		}
		matched := false
		for _, existing := range existingStages {
			if existing.StepIndex != nil && *existing.StepIndex == *next.StepIndex {
				next.Key = existing.Key
				matched = true
				break
			}
		}
		if !matched && *next.StepIndex >= len(l.Steps) {
			for index := *next.StepIndex; ; index++ {
				next.Key = "phase-" + strconv.Itoa(index)
				if !usedKeys[next.Key] {
					break
				}
			}
			usedKeys[next.Key] = true
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
			if _, existed := existingByKey[stage.Key]; existed {
				continue
			}
			result = append(result, stage)
			delete(nextByKey, stage.Key)
		}
	}
	for _, stage := range existingStages {
		if stage.Kind == FactoryLineStagePhase && stage.StepIndex == nil && stage.Key != "verify" {
			result = append(result, stage)
			continue
		}
		next, ok := nextByKey[stage.Key]
		if !ok {
			continue
		}
		// New phases go before Verify so a task stops in Verify after its work stages.
		if stage.Key == "verify" || stage.Kind == FactoryLineStageDone {
			appendNewStages()
		}
		stage.StepIndex = next.StepIndex
		if stage.StepIndex != nil && *stage.StepIndex < len(l.Steps) && l.Steps[*stage.StepIndex].AppID != steps[*stage.StepIndex].AppID {
			stage.AgentDisplayKey = ""
		}
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

func (l *FactoryLine) ListRemovedStageAutomations(tx *gorm.DB, next []FactoryLineStage) ([]Canvas, error) {
	keys := l.removedStageKeys(next)
	if len(keys) == 0 {
		return nil, nil
	}
	removed := make(map[string]bool, len(keys))
	for _, key := range keys {
		removed[key] = true
	}
	bound := map[uuid.UUID]bool{}
	retainedRunners := map[uuid.UUID]bool{}
	for _, stage := range l.Stages {
		if stage.StepIndex == nil || *stage.StepIndex < 0 || *stage.StepIndex >= len(l.Steps) {
			continue
		}
		id := l.Steps[*stage.StepIndex].AppID
		if removed[stage.Key] {
			bound[id] = true
			continue
		}
		retainedRunners[id] = true
	}
	factory := &Factory{ID: l.FactoryID, OrganizationID: l.OrganizationID}
	canvases, err := factory.ListCanvases(tx)
	if err != nil {
		return nil, err
	}
	handlers, err := factory.ListPRFeedbackHandlers(tx)
	if err != nil {
		return nil, err
	}
	feedback := map[uuid.UUID]bool{}
	for _, handler := range handlers {
		feedback[handler.CanvasID] = true
	}
	ids := make([]uuid.UUID, 0, len(canvases))
	for _, canvas := range canvases {
		ids = append(ids, canvas.ID)
	}
	specs, err := FindLiveCanvasSpecsByCanvasIDs(tx, ids)
	if err != nil {
		return nil, err
	}
	selected := make([]Canvas, 0)
	for _, canvas := range canvases {
		key := ""
		if canvas.ColumnKey != nil {
			key = *canvas.ColumnKey
		}
		if key == CanvasColumnKeyImplement {
			key = "phase-0"
		}
		if feedback[canvas.ID] {
			key = "verify"
		}
		if key == "" {
			switch FactoryAppTemplateID(specs[canvas.ID].Nodes) {
			case "risk-score", "visual-evidence", "review-request", "pr-feedback:checks", "pr-feedback:discussion":
				key = "verify"
			}
		}
		if !retainedRunners[canvas.ID] && (bound[canvas.ID] || removed[key]) {
			selected = append(selected, canvas)
		}
	}
	return selected, nil
}

func (l *FactoryLine) ClearRemovedStageDisplayColumns(tx *gorm.DB, next []FactoryLineStage) error {
	keys := l.removedStageKeys(next)
	if len(keys) == 0 {
		return nil
	}
	return tx.Model(&Canvas{}).
		Where("organization_id = ? AND factory_id = ? AND display_column_key IN ?", l.OrganizationID, l.FactoryID, keys).
		Update("display_column_key", "").Error
}

func (l *FactoryLine) removedStageKeys(next []FactoryLineStage) []string {
	retained := make(map[uuid.UUID]bool, len(next))
	for _, stage := range next {
		retained[stage.ID] = true
	}
	removedKeys := make([]string, 0, len(l.Stages))
	for _, stage := range l.StagesValue() {
		if !retained[stage.ID] {
			removedKeys = append(removedKeys, stage.Key)
		}
	}
	return removedKeys
}

// Remove deleted phase bindings and deleted canvases; keep retained stage identities.
func (l *FactoryLine) RemoveDeletedAutomationBindings(tx *gorm.DB, next []FactoryLineStage, deleted []Canvas) ([]FactoryLineStep, error) {
	ids := map[uuid.UUID]bool{}
	for _, canvas := range deleted {
		ids[canvas.ID] = true
	}
	removedIndexes := map[int]bool{}
	removedKeys := l.removedStageKeys(next)
	for _, stage := range l.StagesValue() {
		if stage.StepIndex != nil && slices.Contains(removedKeys, stage.Key) {
			removedIndexes[*stage.StepIndex] = true
		}
	}
	if len(deleted) == 0 && len(removedIndexes) == 0 {
		return nil, nil
	}
	factory := &Factory{ID: l.FactoryID, OrganizationID: l.OrganizationID}
	lines, err := factory.ListLines(tx.Clauses(clause.Locking{Strength: "UPDATE"}))
	if err != nil {
		return nil, err
	}
	var currentSteps []FactoryLineStep
	for i := range lines {
		line := &lines[i]
		steps := make([]FactoryLineStep, 0, len(line.Steps))
		indexes := map[int]int{}
		for index, step := range line.Steps {
			if ids[step.AppID] || (line.ID == l.ID && removedIndexes[index]) {
				continue
			}
			indexes[index] = len(steps)
			steps = append(steps, step)
		}
		stages := line.StagesValue()
		if line.ID == l.ID {
			stages = next
		}
		for j := range stages {
			stage := &stages[j]
			if stage.StepIndex == nil {
				continue
			}
			index, retained := indexes[*stage.StepIndex]
			stage.StepIndex = nil
			if retained {
				stage.StepIndex = &index
				continue
			}
			stage.AgentDisplayKey = ""
		}
		if line.ID == l.ID {
			currentSteps = steps
			continue
		}
		if len(steps) != len(line.Steps) {
			if err := line.UpdateConfiguration(tx, nil, steps, nil, stages); err != nil {
				return nil, err
			}
		}
	}
	return currentSteps, nil
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
		newFactoryLineStage(FactoryLineStagePhase, "verify", "Verify", nil),
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
	autoAdvance := key != "verify"
	return FactoryLineStage{
		ID:          uuid.New(),
		Key:         key,
		Name:        name,
		Kind:        kind,
		StepIndex:   stepIndex,
		AutoAdvance: &autoAdvance,
	}
}
