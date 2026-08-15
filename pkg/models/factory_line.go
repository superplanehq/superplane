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
}

const onRunTriggerName = "onRun"

// StartStep launches the given step against the run's inputs and records a
// `step.execution.created` event.
func (l *FactoryLine) StartStep(tx *gorm.DB, order *FactoryWorkOrder, stepIndex int) (*FactoryLineStepResult, error) {
	steps := []FactoryLineStep(l.Steps)
	if stepIndex < 0 || stepIndex >= len(steps) {
		return nil, fmt.Errorf("step index %d out of range", stepIndex)
	}

	step := steps[stepIndex]

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
		// Snapshot the full step sequence as it looks right now, so the
		// history rendered for this execution never changes even if the
		// line is edited afterwards. See the field comment on
		// FactoryWorkOrderExecution for details.
		LineSteps: datatypes.JSONSlice[FactoryLineStep](steps),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := tx.Clauses(clause.Returning{}).Create(execution).Error; err != nil {
		return nil, err
	}

	if err := l.RecordStepExecutionCreated(tx, order, execution, &step, run); err != nil {
		return nil, err
	}

	return &FactoryLineStepResult{
		Run:       run,
		Execution: execution,
	}, nil
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
