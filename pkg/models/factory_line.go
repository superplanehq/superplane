package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	FactoryLineStepTypeRunApp = "runApp"

	// DefaultFactoryLineStepMaxParallelism is the number of in-flight runs a
	// step allows when maxParallelism is not set. 0 means unlimited.
	DefaultFactoryLineStepMaxParallelism = 10

	factoryLineNameUniqueConstraint = "factory_lines_factory_id_name_key"
)

var (
	ErrFactoryLineNotFound          = errors.New("factory line not found")
	ErrFactoryLineNameAlreadyExists = errors.New("factory line name already exists")
	ErrFactoryLineHasNoSteps        = errors.New("factory line has no steps")
	ErrFactoryLineStepNotOnRun      = errors.New("factory line step entrypoint must use the onRun trigger")
)

type FactoryLineStep struct {
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	AppID      uuid.UUID `json:"app_id"`
	Entrypoint string    `json:"entrypoint"`
	// MaxParallelism caps the step's in-flight runs. Unset means the
	// default of 10; 0 means unlimited.
	MaxParallelism *int `json:"max_parallelism,omitempty"`
}

func (s *FactoryLineStep) EffectiveMaxParallelism() int {
	if s.MaxParallelism == nil {
		return DefaultFactoryLineStepMaxParallelism
	}
	return *s.MaxParallelism
}

func (s *FactoryLineStep) Unlimited() bool {
	return s.MaxParallelism != nil && *s.MaxParallelism == 0
}

type FactoryLine struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	Name           string
	Steps          datatypes.JSONSlice[FactoryLineStep]
	CreatedAt      time.Time
	UpdatedAt      time.Time
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
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := tx.Clauses(clause.Returning{}).Create(line).Error; err != nil {
		return nil, MapFactoryLineNameUniqueConstraintError(err)
	}

	return line, nil
}

type FactoryLineStepResult struct {
	Run       *CanvasRun
	Execution *FactoryWorkOrderExecution
	// QueueItem is set instead of Run/Execution when the step was at its
	// maxParallelism and the work order was queued.
	QueueItem *FactoryWorkOrderQueueItem
}

const onRunTriggerName = "onRun"

// StartStep launches the given step against the run's inputs and records a
// `step.execution.created` event. It does not check step capacity; use
// EnqueueOrStartStep for admission-aware starts.
func (l *FactoryLine) StartStep(tx *gorm.DB, order *FactoryWorkOrder, stepIndex int) (*FactoryLineStepResult, error) {
	step, err := l.stepAt(stepIndex)
	if err != nil {
		return nil, err
	}

	run, err := l.createStepRun(tx, order, step)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	execution := &FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: l.OrganizationID,
		FactoryID:      l.FactoryID,
		WorkOrderID:    order.ID,
		LineID:         l.ID,
		StepIndex:      stepIndex,
		StepName:       step.Name,
		RunID:          run.ID,
		Status:         FactoryWorkOrderExecutionStatusPending,
		Result:         "",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := tx.Clauses(clause.Returning{}).Create(execution).Error; err != nil {
		return nil, err
	}

	if err := l.RecordStepExecutionCreated(tx, order, execution, step, run); err != nil {
		return nil, err
	}

	return &FactoryLineStepResult{
		Run:       run,
		Execution: execution,
	}, nil
}

// EnqueueOrStartStep starts the step when it is below its maxParallelism,
// and otherwise puts the work order in the step's queue (result.Run is
// nil and result.QueueItem is set). Admission is serialized on the line
// row lock so concurrent dispatches and completions cannot over-admit.
func (l *FactoryLine) EnqueueOrStartStep(tx *gorm.DB, order *FactoryWorkOrder, stepIndex int) (*FactoryLineStepResult, error) {
	if err := l.lockForAdmission(tx); err != nil {
		return nil, err
	}

	step, err := l.stepAt(stepIndex)
	if err != nil {
		return nil, err
	}

	hasCapacity, err := l.stepHasCapacity(tx, stepIndex, step)
	if err != nil {
		return nil, err
	}

	if hasCapacity {
		return l.StartStep(tx, order, stepIndex)
	}

	item := &FactoryWorkOrderQueueItem{
		ID:             uuid.New(),
		OrganizationID: l.OrganizationID,
		FactoryID:      l.FactoryID,
		WorkOrderID:    order.ID,
		LineID:         l.ID,
		StepIndex:      stepIndex,
		StepName:       step.Name,
		CreatedAt:      time.Now(),
	}

	if err := tx.Clauses(clause.Returning{}).Create(item).Error; err != nil {
		return nil, err
	}

	if err := l.recordStepExecutionQueued(tx, order, step); err != nil {
		return nil, err
	}

	return &FactoryLineStepResult{QueueItem: item}, nil
}

// AdmitNextQueuedForStep starts the oldest queued work order for the
// step if the step has capacity. Queue items whose work order is no
// longer open are deleted and skipped. Returns nil when nothing was
// admitted.
func (l *FactoryLine) AdmitNextQueuedForStep(tx *gorm.DB, stepIndex int) (*FactoryLineStepResult, error) {
	//
	// Line steps can change after executions were created; a stale step
	// index means there is no step queue to admit from.
	//
	if stepIndex < 0 || stepIndex >= len(l.Steps) {
		return nil, nil
	}

	if err := l.lockForAdmission(tx); err != nil {
		return nil, err
	}

	step, err := l.stepAt(stepIndex)
	if err != nil {
		return nil, err
	}

	for {
		hasCapacity, err := l.stepHasCapacity(tx, stepIndex, step)
		if err != nil {
			return nil, err
		}
		if !hasCapacity {
			return nil, nil
		}

		item, err := findOldestFactoryWorkOrderQueueItem(tx, l.ID, stepIndex)
		if err != nil {
			return nil, err
		}
		if item == nil {
			return nil, nil
		}

		var order FactoryWorkOrder
		err = tx.
			Where("organization_id = ? AND factory_id = ? AND id = ?", l.OrganizationID, l.FactoryID, item.WorkOrderID).
			First(&order).
			Error
		if err != nil {
			return nil, err
		}

		if err := item.Delete(tx); err != nil {
			return nil, err
		}

		if !order.IsOpen() {
			continue
		}

		return l.StartStep(tx, &order, stepIndex)
	}
}

func (l *FactoryLine) stepAt(stepIndex int) (*FactoryLineStep, error) {
	steps := []FactoryLineStep(l.Steps)
	if stepIndex < 0 || stepIndex >= len(steps) {
		return nil, fmt.Errorf("step index %d out of range", stepIndex)
	}
	return &steps[stepIndex], nil
}

// lockForAdmission serializes step admission decisions on the line row.
func (l *FactoryLine) lockForAdmission(tx *gorm.DB) error {
	var locked FactoryLine
	return tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", l.ID).
		First(&locked).
		Error
}

func (l *FactoryLine) stepHasCapacity(tx *gorm.DB, stepIndex int, step *FactoryLineStep) (bool, error) {
	if step.Unlimited() {
		return true, nil
	}

	var inFlight int64
	err := tx.
		Model(&FactoryWorkOrderExecution{}).
		Where("line_id = ? AND step_index = ?", l.ID, stepIndex).
		Where("status IN ?", []string{
			FactoryWorkOrderExecutionStatusPending,
			FactoryWorkOrderExecutionStatusRunning,
		}).
		Count(&inFlight).
		Error
	if err != nil {
		return false, err
	}

	return inFlight < int64(step.EffectiveMaxParallelism()), nil
}

// createStepRun validates the step's entrypoint and creates the pending
// canvas run carrying the work order as input.
func (l *FactoryLine) createStepRun(tx *gorm.DB, order *FactoryWorkOrder, step *FactoryLineStep) (*CanvasRun, error) {
	node, err := FindCanvasNode(tx, step.AppID, step.Entrypoint)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("entrypoint %q not found", step.Entrypoint)
		}
		return nil, err
	}

	ref := node.Ref.Data()
	if ref.Trigger == nil || ref.Trigger.Name != onRunTriggerName {
		return nil, ErrFactoryLineStepNotOnRun
	}

	liveVersion, err := FindLiveCanvasVersionInTransaction(tx, step.AppID)
	if err != nil {
		return nil, err
	}

	runInput, err := factoryWorkOrderRunInput(tx, order)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	run := &CanvasRun{
		ID:         uuid.New(),
		WorkflowID: step.AppID,
		NodeID:     step.Entrypoint,
		VersionID:  liveVersion.ID,
		Callbacks: datatypes.JSONSlice[core.RunCallback]{
			{
				When: core.RunCallbackWhenPending,
				On:   core.RunCallbackOnEntry,
				Hook: "onMessage",
			},
		},
		Input:     NewJSONValue(runInput),
		State:     CanvasRunStatePending,
		CreatedAt: &now,
		UpdatedAt: &now,
	}

	if err := tx.Create(run).Error; err != nil {
		return nil, err
	}

	return run, nil
}

func (l *FactoryLine) RecordStepExecutionCreated(tx *gorm.DB, order *FactoryWorkOrder, execution *FactoryWorkOrderExecution, step *FactoryLineStep, run *CanvasRun) error {
	data := factory.LineStepExecutionCreated{
		StepName: step.Name,
		Order:    order.Ref(),
		Line:     &factory.LineRef{ID: l.ID, Name: l.Name},
		App:      &factory.AppRef{ID: run.WorkflowID},
		Run:      &factory.RunRef{ID: run.ID, State: run.State},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	event := &FactoryWorkOrderEvent{
		ID:          uuid.New(),
		WorkOrderID: order.ID,
		Type:        factory.EventTypeLineStepExecutionCreated,
		Data:        datatypes.JSON(jsonData),
		CreatedAt:   time.Now(),
	}

	return tx.Create(event).Error
}

func (l *FactoryLine) recordStepExecutionQueued(tx *gorm.DB, order *FactoryWorkOrder, step *FactoryLineStep) error {
	data := factory.LineStepExecutionQueued{
		StepName: step.Name,
		Order:    order.Ref(),
		Line:     &factory.LineRef{ID: l.ID, Name: l.Name},
		App:      &factory.AppRef{ID: step.AppID},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	event := &FactoryWorkOrderEvent{
		ID:          uuid.New(),
		WorkOrderID: order.ID,
		Type:        factory.EventTypeLineStepExecutionQueued,
		Data:        datatypes.JSON(jsonData),
		CreatedAt:   time.Now(),
	}

	return tx.Create(event).Error
}

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

func (l *FactoryLine) Update(tx *gorm.DB, name *string, steps []FactoryLineStep) error {
	updates := map[string]any{
		"updated_at": time.Now(),
	}
	if name != nil {
		updates["name"] = *name
	}
	if steps != nil {
		updates["steps"] = datatypes.JSONSlice[FactoryLineStep](steps)
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
