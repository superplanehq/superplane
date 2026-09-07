package models

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FactoryResourceCleaner hard-deletes factory-owned rows in FK-safe order
// after all factory canvases are gone. Each Run deletes at most limit rows
// total so large factories finish across ticks without long transactions.
type FactoryResourceCleaner struct {
	tx      *gorm.DB
	factory *Factory
	limit   int
}

func NewFactoryResourceCleaner(tx *gorm.DB, factory *Factory) *FactoryResourceCleaner {
	return &FactoryResourceCleaner{
		tx:      tx,
		factory: factory,
		limit:   500,
	}
}

func (c *FactoryResourceCleaner) WithLimit(limit int) *FactoryResourceCleaner {
	c.limit = limit
	return c
}

// Run deletes up to limit rows. complete is true when the factory row itself
// was hard-deleted.
func (c *FactoryResourceCleaner) Run() (deleted int64, complete bool, err error) {
	if c.limit <= 0 {
		return 0, false, nil
	}

	remaining := c.limit
	factoryOrders := c.tx.Model(&FactoryWorkOrder{}).Select("id").Where("factory_id = ?", c.factory.ID)

	// Queue items FK-reference line dispatches, work orders, and lines
	// (all RESTRICT), so they must go before any of those.
	count, err := deleteRowsLimited(c.tx, &FactoryWorkOrderQueueItem{}, remaining, "factory_id = ?", c.factory.ID)
	if err != nil {
		return deleted, false, fmt.Errorf("delete factory work order queue items: %w", err)
	}
	deleted += count
	remaining -= int(count)
	if remaining <= 0 {
		return deleted, false, nil
	}

	count, err = deleteRowsLimited(c.tx, &FactoryWorkOrderExecution{}, remaining, "factory_id = ?", c.factory.ID)
	if err != nil {
		return deleted, false, fmt.Errorf("delete factory work order executions: %w", err)
	}
	deleted += count
	remaining -= int(count)
	if remaining <= 0 {
		return deleted, false, nil
	}

	// Line dispatches are the parent of executions (line_dispatch_id is a
	// RESTRICT FK), so they're only safe to delete once their executions
	// are gone.
	count, err = deleteRowsLimited(c.tx, &FactoryWorkOrderLineDispatch{}, remaining, "factory_id = ?", c.factory.ID)
	if err != nil {
		return deleted, false, fmt.Errorf("delete factory work order line dispatches: %w", err)
	}
	deleted += count
	remaining -= int(count)
	if remaining <= 0 {
		return deleted, false, nil
	}

	count, err = deleteRowsLimited(
		c.tx,
		&FactoryWorkOrderEvent{},
		remaining,
		"work_order_id IN (?)",
		factoryOrders,
	)
	if err != nil {
		return deleted, false, fmt.Errorf("delete factory work order events: %w", err)
	}
	deleted += count
	remaining -= int(count)
	if remaining <= 0 {
		return deleted, false, nil
	}

	count, err = deleteRowsLimited(c.tx, &FactoryWorkOrderCheck{}, remaining, "factory_id = ?", c.factory.ID)
	if err != nil {
		return deleted, false, fmt.Errorf("delete factory work order checks: %w", err)
	}
	deleted += count
	remaining -= int(count)
	if remaining <= 0 {
		return deleted, false, nil
	}

	count, err = deleteFactoryAssigneesLimited(c.tx, c.factory.ID, remaining)
	if err != nil {
		return deleted, false, fmt.Errorf("delete factory work order assignees: %w", err)
	}
	deleted += count
	remaining -= int(count)
	if remaining <= 0 {
		return deleted, false, nil
	}

	count, err = clearFactoryPullRequestCurrentRevisionsLimited(c.tx, c.factory.ID, remaining)
	if err != nil {
		return deleted, false, fmt.Errorf("clear factory pull request current revisions: %w", err)
	}
	deleted += count
	remaining -= int(count)
	if remaining <= 0 {
		return deleted, false, nil
	}

	count, err = deleteFactoryPullRequestRunsLimited(c.tx, c.factory.ID, remaining)
	if err != nil {
		return deleted, false, fmt.Errorf("delete factory pull request runs: %w", err)
	}
	deleted += count
	remaining -= int(count)
	if remaining <= 0 {
		return deleted, false, nil
	}

	count, err = deleteFactoryPullRequestRevisionsLimited(c.tx, c.factory.ID, remaining)
	if err != nil {
		return deleted, false, fmt.Errorf("delete factory pull request revisions: %w", err)
	}
	deleted += count
	remaining -= int(count)
	if remaining <= 0 {
		return deleted, false, nil
	}

	count, err = deleteRowsLimited(c.tx, &FactoryPullRequest{}, remaining, "factory_id = ?", c.factory.ID)
	if err != nil {
		return deleted, false, fmt.Errorf("delete factory pull requests: %w", err)
	}
	deleted += count
	remaining -= int(count)
	if remaining <= 0 {
		return deleted, false, nil
	}

	count, err = deleteOrphanFactoryWorkOrdersLimited(c.tx, c.factory.ID, remaining)
	if err != nil {
		return deleted, false, fmt.Errorf("delete factory work orders: %w", err)
	}
	deleted += count
	remaining -= int(count)
	if remaining <= 0 {
		return deleted, false, nil
	}

	count, err = deleteRowsLimited(c.tx, &FactoryLine{}, remaining, "factory_id = ?", c.factory.ID)
	if err != nil {
		return deleted, false, fmt.Errorf("delete factory lines: %w", err)
	}
	deleted += count
	remaining -= int(count)
	if remaining <= 0 {
		return deleted, false, nil
	}

	// Canvas cleanup normally removes these along with the intake canvas. This
	// is the backstop for a factory whose canvases went away another way.
	count, err = deleteRowsLimited(c.tx, &FactoryIntake{}, remaining, "factory_id = ?", c.factory.ID)
	if err != nil {
		return deleted, false, fmt.Errorf("delete factory intakes: %w", err)
	}
	deleted += count
	remaining -= int(count)
	if remaining <= 0 {
		return deleted, false, nil
	}

	count, err = deleteRowsLimited(c.tx, &FactoryPRFeedbackHandler{}, remaining, "factory_id = ?", c.factory.ID)
	if err != nil {
		return deleted, false, fmt.Errorf("delete factory PR feedback handlers: %w", err)
	}
	deleted += count

	empty, err := c.factoryDomainEmpty()
	if err != nil {
		return deleted, false, err
	}
	if !empty {
		return deleted, false, nil
	}

	result := c.tx.Unscoped().Where("id = ?", c.factory.ID).Delete(&Factory{})
	if result.Error != nil {
		return deleted, false, fmt.Errorf("hard delete factory: %w", result.Error)
	}
	deleted += result.RowsAffected

	return deleted, result.RowsAffected > 0, nil
}

func (c *FactoryResourceCleaner) factoryDomainEmpty() (bool, error) {
	var executions, dispatches, orders, lines, intakes, handlers, pullRequests int64
	if err := c.tx.Model(&FactoryWorkOrderExecution{}).Where("factory_id = ?", c.factory.ID).Limit(1).Count(&executions).Error; err != nil {
		return false, err
	}
	if err := c.tx.Model(&FactoryWorkOrderLineDispatch{}).Where("factory_id = ?", c.factory.ID).Limit(1).Count(&dispatches).Error; err != nil {
		return false, err
	}
	if err := c.tx.Model(&FactoryWorkOrder{}).Where("factory_id = ?", c.factory.ID).Limit(1).Count(&orders).Error; err != nil {
		return false, err
	}
	if err := c.tx.Model(&FactoryLine{}).Where("factory_id = ?", c.factory.ID).Limit(1).Count(&lines).Error; err != nil {
		return false, err
	}
	if err := c.tx.Model(&FactoryIntake{}).Where("factory_id = ?", c.factory.ID).Limit(1).Count(&intakes).Error; err != nil {
		return false, err
	}
	if err := c.tx.Model(&FactoryPRFeedbackHandler{}).Where("factory_id = ?", c.factory.ID).Limit(1).Count(&handlers).Error; err != nil {
		return false, err
	}
	if err := c.tx.Model(&FactoryPullRequest{}).Where("factory_id = ?", c.factory.ID).Limit(1).Count(&pullRequests).Error; err != nil {
		return false, err
	}
	return executions == 0 && dispatches == 0 && orders == 0 && lines == 0 && intakes == 0 && handlers == 0 && pullRequests == 0, nil
}

func deleteFactoryAssigneesLimited(tx *gorm.DB, factoryID uuid.UUID, limit int) (int64, error) {
	if limit <= 0 {
		return 0, nil
	}

	// Composite PK — one DELETE with a bounded subquery keeps large factories predictable.
	result := tx.Exec(`
		DELETE FROM factory_work_order_assignees AS assignees
		USING (
			SELECT candidate.work_order_id, candidate.user_id
			FROM factory_work_order_assignees AS candidate
			JOIN factory_work_orders ON factory_work_orders.id = candidate.work_order_id
			WHERE factory_work_orders.factory_id = ?
			LIMIT ?
		) AS doomed
		WHERE assignees.work_order_id = doomed.work_order_id
		  AND assignees.user_id = doomed.user_id
	`, factoryID, limit)
	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

func clearFactoryPullRequestCurrentRevisionsLimited(tx *gorm.DB, factoryID uuid.UUID, limit int) (int64, error) {
	if limit <= 0 {
		return 0, nil
	}

	subQuery := tx.Model(&FactoryPullRequest{}).
		Select("id").
		Where("factory_id = ? AND current_revision_id IS NOT NULL", factoryID).
		Limit(limit)
	result := tx.Model(&FactoryPullRequest{}).
		Where("id IN (?)", subQuery).
		Update("current_revision_id", nil)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func deleteFactoryPullRequestRevisionsLimited(tx *gorm.DB, factoryID uuid.UUID, limit int) (int64, error) {
	if limit <= 0 {
		return 0, nil
	}

	subQuery := tx.Model(&FactoryPullRequestRevision{}).
		Select("factory_pull_request_revisions.id").
		Joins("JOIN factory_pull_requests ON factory_pull_requests.id = factory_pull_request_revisions.pull_request_id").
		Where("factory_pull_requests.factory_id = ?", factoryID).
		Limit(limit)
	result := tx.Where("id IN (?)", subQuery).Delete(&FactoryPullRequestRevision{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func deleteFactoryPullRequestRunsLimited(tx *gorm.DB, factoryID uuid.UUID, limit int) (int64, error) {
	if limit <= 0 {
		return 0, nil
	}

	result := tx.Exec(`
		DELETE FROM factory_pull_request_runs AS links
		USING (
			SELECT candidate.pull_request_id, candidate.run_id
			FROM factory_pull_request_runs AS candidate
			JOIN factory_pull_requests ON factory_pull_requests.id = candidate.pull_request_id
			WHERE factory_pull_requests.factory_id = ?
			LIMIT ?
		) AS doomed
		WHERE links.pull_request_id = doomed.pull_request_id
		  AND links.run_id = doomed.run_id
	`, factoryID, limit)
	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

func deleteOrphanFactoryWorkOrdersLimited(tx *gorm.DB, factoryID uuid.UUID, limit int) (int64, error) {
	if limit <= 0 {
		return 0, nil
	}

	// Only delete orders whose children are already gone so each tick stays within budget.
	subQuery := tx.Model(&FactoryWorkOrder{}).
		Select("factory_work_orders.id").
		Where("factory_work_orders.factory_id = ?", factoryID).
		Where(`
			NOT EXISTS (
				SELECT 1 FROM factory_work_order_events
				WHERE factory_work_order_events.work_order_id = factory_work_orders.id
			)`).
		Where(`
			NOT EXISTS (
				SELECT 1 FROM factory_work_order_assignees
				WHERE factory_work_order_assignees.work_order_id = factory_work_orders.id
			)`).
		Where(`
			NOT EXISTS (
				SELECT 1 FROM factory_work_order_executions
				WHERE factory_work_order_executions.work_order_id = factory_work_orders.id
			)`).
		Where(`
			NOT EXISTS (
				SELECT 1 FROM factory_work_order_line_dispatches
				WHERE factory_work_order_line_dispatches.work_order_id = factory_work_orders.id
			)`).
		Where(`
			NOT EXISTS (
				SELECT 1 FROM factory_work_order_queue_items
				WHERE factory_work_order_queue_items.work_order_id = factory_work_orders.id
			)`).
		Where(`
			NOT EXISTS (
				SELECT 1 FROM factory_work_order_checks
				WHERE factory_work_order_checks.work_order_id = factory_work_orders.id
			)`).
		Where(`
			NOT EXISTS (
				SELECT 1 FROM factory_pull_requests
				WHERE factory_pull_requests.work_order_id = factory_work_orders.id
			)`).
		Limit(limit)

	result := tx.Where("id IN (?)", subQuery).Delete(&FactoryWorkOrder{})
	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}
