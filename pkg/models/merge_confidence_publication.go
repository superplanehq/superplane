package models

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
)

// ListActivityRuns returns newest-created runs first, including active runs.
// Callers hold the pull request row lock to serialize against new activity links.
func (p *FactoryPullRequest) ListActivityRuns(tx *gorm.DB) ([]CanvasRun, error) {
	var runs []CanvasRun
	err := tx.Table("workflow_runs").Select("workflow_runs.*").
		Joins("JOIN factory_pull_request_runs a ON a.run_id = workflow_runs.id").
		Where("a.pull_request_id = ?", p.ID).
		Order("workflow_runs.created_at DESC, workflow_runs.id DESC").Find(&runs).Error
	return runs, err
}

// ChecksForRun reads immutable reports, never the mutable latest-check rows.
func (o *FactoryWorkOrder) ChecksForRun(tx *gorm.DB, runID uuid.UUID) (map[string]factory.CheckRef, error) {
	var events []FactoryWorkOrderEvent
	err := tx.Where("work_order_id = ? AND type = ?", o.ID, factory.EventTypeOrderCheckReported).
		Where("data->'run'->>'id' = ?", runID.String()).Order("created_at ASC, id ASC").Find(&events).Error
	if err != nil {
		return nil, err
	}
	checks := map[string]factory.CheckRef{}
	for _, event := range events {
		var report factory.WorkOrderCheckReported
		if err := json.Unmarshal(event.Data, &report); err != nil {
			return nil, err
		}
		if report.Check != nil && report.Run != nil && report.Run.ID == runID {
			checks[report.Check.Key] = *report.Check
		}
	}
	return checks, nil
}

// RootEvent reads the run's source even when admission failed before execution.
func (r *CanvasRun) RootEvent(tx *gorm.DB) (*CanvasEvent, error) {
	var event CanvasEvent
	err := tx.Where("run_id = ? AND execution_id IS NULL", r.ID).Order("created_at ASC, id ASC").First(&event).Error
	return &event, err
}
