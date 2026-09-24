package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

const factoryCanvasRuntimeSweepLimit = 10_000

// FactoryRuntimeCleaner deletes factory runtime rows and keeps the workspace
// itself: apps, lines, intakes, PR feedback handlers, and integrations.
type FactoryRuntimeCleaner struct {
	tx      *gorm.DB
	factory *Factory
}

func NewFactoryRuntimeCleaner(tx *gorm.DB, factory *Factory) *FactoryRuntimeCleaner {
	return &FactoryRuntimeCleaner{tx: tx, factory: factory}
}

// Run wipes work orders, runs, pull requests, planning, files, and related
// runtime rows, then sets next_work_order_number back to 1.
func (c *FactoryRuntimeCleaner) Run() error {
	canvasIDs := c.tx.Model(&Canvas{}).Select("id").Where("factory_id = ?", c.factory.ID)
	factoryOrders := c.tx.Model(&FactoryWorkOrder{}).Select("id").Where("factory_id = ?", c.factory.ID)

	steps := []struct {
		name string
		run  func() error
	}{
		{"delete workspace usage events", func() error {
			return c.delete(&WorkspaceUsageEvent{}, "factory_id = ?", c.factory.ID)
		}},
		{"delete planning sessions", func() error {
			return c.delete(&FactoryPlanningSession{}, "factory_id = ?", c.factory.ID)
		}},
		{"delete agent sessions", func() error {
			return c.delete(&AgentSession{}, "canvas_id IN (?)", canvasIDs)
		}},
		{"delete canvas memories", func() error {
			return c.delete(&CanvasMemory{}, "canvas_id IN (?)", canvasIDs)
		}},
		{"delete work order comments", func() error {
			return c.delete(&FactoryWorkOrderComment{}, "factory_id = ?", c.factory.ID)
		}},
		{"delete work order artifacts", func() error {
			return c.delete(&FactoryWorkOrderArtifact{}, "factory_id = ?", c.factory.ID)
		}},
		{"delete files", func() error {
			return c.delete(&File{}, "factory_id = ?", c.factory.ID)
		}},
		{"delete work order queue items", func() error {
			return c.delete(&FactoryWorkOrderQueueItem{}, "factory_id = ?", c.factory.ID)
		}},
		{"delete work order executions", func() error {
			return c.delete(&FactoryWorkOrderExecution{}, "factory_id = ?", c.factory.ID)
		}},
		{"delete line dispatches", func() error {
			return c.delete(&FactoryWorkOrderLineDispatch{}, "factory_id = ?", c.factory.ID)
		}},
		{"delete work order events", func() error {
			return c.delete(&FactoryWorkOrderEvent{}, "work_order_id IN (?)", factoryOrders)
		}},
		{"delete work order checks", func() error {
			return c.delete(&FactoryWorkOrderCheck{}, "factory_id = ?", c.factory.ID)
		}},
		{"delete work order assignees", c.deleteAssignees},
		{"clear pull request current revisions", c.clearPullRequestCurrentRevisions},
		{"delete pull request runs", c.deletePullRequestRuns},
		{"delete pull request revisions", c.deletePullRequestRevisions},
		{"delete pull requests", func() error {
			return c.delete(&FactoryPullRequest{}, "factory_id = ?", c.factory.ID)
		}},
		{"delete velocity merges", func() error {
			return DeleteFactoryVelocityRepositoryMerges(c.tx, c.factory.ID)
		}},
		{"delete velocity syncs", func() error {
			return c.delete(&FactoryVelocitySync{}, "factory_id = ?", c.factory.ID)
		}},
		{"delete canvas runs", c.deleteCanvasRuntime},
		{"delete work orders", func() error {
			return c.delete(&FactoryWorkOrder{}, "factory_id = ?", c.factory.ID)
		}},
		{"reset work order numbers", c.resetWorkOrderNumbers},
	}

	for _, step := range steps {
		if err := step.run(); err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
	}

	return nil
}

func (c *FactoryRuntimeCleaner) delete(model any, query string, args ...any) error {
	_, err := deleteRows(c.tx, model, query, args...)
	return err
}

func (c *FactoryRuntimeCleaner) deleteAssignees() error {
	result := c.tx.Exec(`
		DELETE FROM factory_work_order_assignees AS assignees
		USING factory_work_orders
		WHERE assignees.work_order_id = factory_work_orders.id
		  AND factory_work_orders.factory_id = ?
	`, c.factory.ID)
	return result.Error
}

func (c *FactoryRuntimeCleaner) clearPullRequestCurrentRevisions() error {
	result := c.tx.Model(&FactoryPullRequest{}).
		Where("factory_id = ? AND current_revision_id IS NOT NULL", c.factory.ID).
		Update("current_revision_id", nil)
	return result.Error
}

func (c *FactoryRuntimeCleaner) deletePullRequestRuns() error {
	result := c.tx.Exec(`
		DELETE FROM factory_pull_request_runs AS links
		USING factory_pull_requests
		WHERE links.pull_request_id = factory_pull_requests.id
		  AND factory_pull_requests.factory_id = ?
	`, c.factory.ID)
	return result.Error
}

func (c *FactoryRuntimeCleaner) deletePullRequestRevisions() error {
	result := c.tx.Exec(`
		DELETE FROM factory_pull_request_revisions AS revisions
		USING factory_pull_requests
		WHERE revisions.pull_request_id = factory_pull_requests.id
		  AND factory_pull_requests.factory_id = ?
	`, c.factory.ID)
	return result.Error
}

func (c *FactoryRuntimeCleaner) deleteCanvasRuntime() error {
	result := c.tx.Exec(`
		UPDATE workflow_runs
		SET parent_run_id = NULL, parent_execution_id = NULL
		WHERE workflow_id IN (
			SELECT id FROM workflows WHERE factory_id = ?
		)
	`, c.factory.ID)
	if result.Error != nil {
		return result.Error
	}

	canvases, err := c.factory.ListCanvases(c.tx)
	if err != nil {
		return err
	}

	for i := range canvases {
		if err := sweepCanvasRuntime(c.tx, &canvases[i]); err != nil {
			return fmt.Errorf("canvas %s: %w", canvases[i].ID, err)
		}
	}

	return nil
}

func sweepCanvasRuntime(tx *gorm.DB, canvas *Canvas) error {
	for {
		_, complete, err := canvas.DeleteRemainingResources(tx, factoryCanvasRuntimeSweepLimit)
		if err != nil {
			return err
		}
		if complete {
			return nil
		}
	}
}

func (c *FactoryRuntimeCleaner) resetWorkOrderNumbers() error {
	now := time.Now()
	result := c.tx.Model(c.factory).
		Where("id = ?", c.factory.ID).
		Updates(map[string]any{
			"next_work_order_number": int64(1),
			"updated_at":             now,
		})
	if result.Error != nil {
		return result.Error
	}
	c.factory.NextWorkOrderNumber = 1
	c.factory.UpdatedAt = now
	return nil
}
