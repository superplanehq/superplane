package telemetry

import (
	"context"
	"time"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

//
// Reports metrics at periodic intervals.
//

const activeWindow = 24 * time.Hour

type Periodic struct {
	ctx                  context.Context
	lastPoolWaitCount    int64
	lastPoolWaitDuration time.Duration
}

func NewPeriodic(ctx context.Context) *Periodic {
	return &Periodic{
		ctx: ctx,
	}
}

func (p *Periodic) Start() {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			p.report()
		}
	}()
}

func (p *Periodic) report() {
	p.reportDatabasePoolStats()
	p.reportDatabaseLocks()
	p.reportLongQueries()
	p.reportStuckQueueItems()
	p.reportPendingEvents()
	p.reportPendingExecutions()
	p.reportPendingIntegrationRequests()
	p.reportOrganizationsTotal()
	p.reportUsersTotal()
	p.reportWorkflowsTotal()
	p.reportWorkflowNodesTotal()
	p.reportDraftsTotal()
	p.reportIntegrationsTotal()
	p.reportIntegrationSecretsTotal()
	p.reportWorkflowsActive()
	p.reportWorkflowRunsDaily()
	p.reportWorkflowEventsDaily()
	p.reportWorkflowNodeExecutionsDaily()
}

func (p *Periodic) reportDatabasePoolStats() {
	stats, err := database.PoolStats()
	if err != nil {
		return
	}

	RecordDBPoolStats(
		p.ctx,
		int64(stats.MaxOpenConnections),
		int64(stats.OpenConnections),
		int64(stats.InUse),
		int64(stats.Idle),
	)

	waitCountDelta := stats.WaitCount - p.lastPoolWaitCount
	waitDurationDelta := stats.WaitDuration - p.lastPoolWaitDuration
	p.lastPoolWaitCount = stats.WaitCount
	p.lastPoolWaitDuration = stats.WaitDuration

	RecordDBPoolWaitCount(p.ctx, waitCountDelta)
	RecordDBPoolWaitDuration(p.ctx, waitDurationDelta)
}

func (p *Periodic) reportDatabaseLocks() {
	var count int64

	err := database.Conn().Raw("select count(*) from pg_locks").Scan(&count).Error
	if err != nil {
		return
	}

	RecordDBLocksCount(p.ctx, count)
}

func (p *Periodic) reportStuckQueueItems() {
	count, err := countStuckQueueNodes()
	if err != nil {
		return
	}

	RecordStuckQueueItemsCount(p.ctx, int(count))
}

func (p *Periodic) reportLongQueries() {
	var count int64

	err := database.Conn().Raw(`
		SELECT COUNT(*)
		FROM pg_stat_activity
		WHERE state = 'active'
		  AND now() - query_start > interval '1 minutes'
	`).Scan(&count).Error

	if err != nil {
		return
	}

	RecordDBLongQueriesCount(p.ctx, count)
}

func (p *Periodic) reportPendingEvents() {
	count, err := countPendingEvents()
	if err != nil {
		return
	}

	RecordPendingEventsCount(p.ctx, count)
}

func (p *Periodic) reportPendingExecutions() {
	count, err := countPendingExecutions()
	if err != nil {
		return
	}

	RecordPendingExecutionsCount(p.ctx, count)
}

func (p *Periodic) reportPendingIntegrationRequests() {
	count, err := countPendingIntegrationRequests()
	if err != nil {
		return
	}

	RecordPendingIntegrationRequestsCount(p.ctx, count)

	//
	// The max-per-installation count is the real early warning for a runaway
	// refresh loop: a single installation holding hundreds of pending requests
	// is invisible in the global total (#5386).
	//
	max, err := maxPendingIntegrationRequestsPerInstallation()
	if err != nil {
		return
	}

	RecordPendingIntegrationRequestsMaxPerInstallation(p.ctx, max)
}

func (p *Periodic) reportOrganizationsTotal() {
	count, err := countOrganizations()
	if err != nil {
		return
	}

	RecordOrganizationsTotal(p.ctx, count)
}

func (p *Periodic) reportUsersTotal() {
	count, err := countUsers()
	if err != nil {
		return
	}

	RecordUsersTotal(p.ctx, count)
}

func (p *Periodic) reportWorkflowsTotal() {
	count, err := countWorkflows()
	if err != nil {
		return
	}

	RecordWorkflowsTotal(p.ctx, count)
}

func (p *Periodic) reportWorkflowNodesTotal() {
	count, err := countWorkflowNodes()
	if err != nil {
		return
	}

	RecordWorkflowNodesTotal(p.ctx, count)
}

func (p *Periodic) reportDraftsTotal() {
	count, err := countDrafts()
	if err != nil {
		return
	}

	RecordDraftsTotal(p.ctx, count)
}

func (p *Periodic) reportIntegrationsTotal() {
	count, err := countIntegrations()
	if err != nil {
		return
	}

	RecordIntegrationsTotal(p.ctx, count)
}

func (p *Periodic) reportIntegrationSecretsTotal() {
	count, err := countIntegrationSecrets()
	if err != nil {
		return
	}

	RecordIntegrationSecretsTotal(p.ctx, count)
}

func (p *Periodic) reportWorkflowsActive() {
	count, err := countActiveWorkflows(activeWindow)
	if err != nil {
		return
	}

	RecordWorkflowsActiveCount(p.ctx, count)
}

func (p *Periodic) reportWorkflowRunsDaily() {
	count, err := countWorkflowRunsCreated(activeWindow)
	if err != nil {
		return
	}

	RecordWorkflowRunsDailyCount(p.ctx, count)
}

func (p *Periodic) reportWorkflowEventsDaily() {
	count, err := countWorkflowEventsCreated(activeWindow)
	if err != nil {
		return
	}

	RecordWorkflowEventsDailyCount(p.ctx, count)
}

func (p *Periodic) reportWorkflowNodeExecutionsDaily() {
	count, err := countWorkflowNodeExecutionsCreated(activeWindow)
	if err != nil {
		return
	}

	RecordWorkflowNodeExecutionsDailyCount(p.ctx, count)
}

func countStuckQueueNodes() (int64, error) {
	db := database.Conn()

	var count int64

	if err := db.
		Raw(`
			SELECT COUNT(*)
			FROM workflow_nodes n
			WHERE EXISTS (
				SELECT 1
				FROM workflow_node_queue_items q
				WHERE q.workflow_id = n.workflow_id
				  AND q.node_id = n.node_id
			)
			AND NOT EXISTS (
				SELECT 1
				FROM workflow_node_executions e
				WHERE e.workflow_id = n.workflow_id
				  AND e.node_id = n.node_id
				  AND e.state <> 'finished'
			)
		`).
		Scan(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func countPendingEvents() (int64, error) {
	var count int64

	err := database.Conn().
		Table("workflow_events AS we").
		Joins("JOIN workflows AS w ON we.workflow_id = w.id").
		Where("we.state = ?", "pending").
		Where("w.deleted_at IS NULL").
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func countPendingExecutions() (int64, error) {
	var count int64

	err := database.Conn().
		Table("workflow_node_executions AS wne").
		Joins("JOIN workflows AS w ON wne.workflow_id = w.id").
		Where("wne.state = ?", "pending").
		Where("w.deleted_at IS NULL").
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func countPendingIntegrationRequests() (int64, error) {
	var count int64

	err := database.Conn().
		Table("app_installation_requests").
		Where("state = ?", "pending").
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func countOrganizations() (int64, error) {
	var count int64

	err := database.Conn().Model(&models.Organization{}).Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func countUsers() (int64, error) {
	var count int64

	err := database.Conn().Model(&models.User{}).Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func countWorkflows() (int64, error) {
	var count int64

	err := database.Conn().Model(&models.Canvas{}).Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func countWorkflowNodes() (int64, error) {
	var count int64

	err := database.Conn().
		Model(&models.CanvasNode{}).
		Joins("JOIN workflows ON workflows.id = workflow_nodes.workflow_id").
		Where("workflows.deleted_at IS NULL").
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func maxPendingIntegrationRequestsPerInstallation() (int64, error) {
	var max int64

	err := database.Conn().
		Raw(`
			SELECT COALESCE(MAX(per_installation), 0)
			FROM (
				SELECT COUNT(*) AS per_installation
				FROM app_installation_requests
				WHERE state = 'pending'
				GROUP BY app_installation_id
			) counts
		`).
		Scan(&max).
		Error
	if err != nil {
		return 0, err
	}

	return max, nil
}

func countDrafts() (int64, error) {
	var count int64

	err := database.Conn().
		Model(&models.CanvasVersion{}).
		Joins("JOIN workflows ON workflows.id = workflow_versions.workflow_id").
		Where("workflow_versions.state = ?", models.CanvasVersionStateDraft).
		Where("workflows.deleted_at IS NULL").
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func countIntegrations() (int64, error) {
	var count int64

	err := database.Conn().Model(&models.Integration{}).Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func countActiveWorkflows(window time.Duration) (int64, error) {
	var count int64

	since := time.Now().Add(-window)
	err := database.Conn().
		Table("workflow_runs AS wr").
		Joins("JOIN workflows AS w ON w.id = wr.workflow_id").
		Where("wr.created_at >= ?", since).
		Where("w.deleted_at IS NULL").
		Distinct("wr.workflow_id").
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func countWorkflowRunsCreated(window time.Duration) (int64, error) {
	var count int64

	since := time.Now().Add(-window)
	err := database.Conn().
		Table("workflow_runs AS wr").
		Joins("JOIN workflows AS w ON w.id = wr.workflow_id").
		Where("wr.created_at >= ?", since).
		Where("w.deleted_at IS NULL").
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func countWorkflowEventsCreated(window time.Duration) (int64, error) {
	var count int64

	since := time.Now().Add(-window)
	err := database.Conn().
		Table("workflow_events AS we").
		Joins("JOIN workflows AS w ON w.id = we.workflow_id").
		Where("we.created_at >= ?", since).
		Where("w.deleted_at IS NULL").
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func countWorkflowNodeExecutionsCreated(window time.Duration) (int64, error) {
	var count int64

	since := time.Now().Add(-window)
	err := database.Conn().
		Table("workflow_node_executions AS wne").
		Joins("JOIN workflows AS w ON w.id = wne.workflow_id").
		Where("wne.created_at >= ?", since).
		Where("w.deleted_at IS NULL").
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func countIntegrationSecrets() (int64, error) {
	var count int64

	err := database.Conn().
		Model(&models.IntegrationSecret{}).
		Joins("JOIN app_installations ON app_installations.id = app_installation_secrets.installation_id").
		Where("app_installations.deleted_at IS NULL").
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}
