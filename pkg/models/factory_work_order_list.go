package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	workOrderOriginHostSQL   = `lower(split_part(split_part(coalesce(nullif(btrim(factory_work_orders.origin_url), ''), ''), '://', 2), '/', 1))`
	workOrderCreatorLabelSQL = `lower(coalesce(creator_canvas.name, '') || ' ' || coalesce(creator_canvas.id::text, ''))`

	confidenceCheckJoin = `
LEFT JOIN factory_work_order_checks AS confidence_check
  ON confidence_check.work_order_id = factory_work_orders.id
 AND confidence_check.key = ?`

	creatorCanvasJoin = `
LEFT JOIN workflow_runs AS creator_run
  ON creator_run.id = factory_work_orders.source_run_id
LEFT JOIN workflows AS creator_canvas
  ON creator_canvas.id = creator_run.workflow_id`
)

var errWorkOrderListCursorMissing = errors.New("work order list cursor not found")

type workOrderListSort struct {
	Key        string
	Descending bool
}

type workOrderListCursor struct {
	ID          uuid.UUID
	UpdatedAt   time.Time
	CreatedAt   time.Time
	Confidence  *float64
	SourceLabel string
}

func workOrderListSortFromFilters(filters ListFactoryWorkOrdersFilters) workOrderListSort {
	key := filters.Sort
	if key == "" {
		key = FactoryWorkOrderListSortUpdated
	}

	descending := filters.SortDirection != FactoryWorkOrderListSortDirectionAsc
	if filters.SortDirection == "" && key == FactoryWorkOrderListSortSource {
		descending = false
	}

	return workOrderListSort{Key: key, Descending: descending}
}

func joinWorkOrderListRelations(query *gorm.DB, filters ListFactoryWorkOrdersFilters) *gorm.DB {
	sort := workOrderListSortFromFilters(filters)
	joined := false

	if sort.Key == FactoryWorkOrderListSortConfidence || filters.ConfidenceMissing || filters.MinConfidence != nil {
		query = query.Joins(confidenceCheckJoin, PlanningConfidenceCheckKey)
		joined = true
	}
	if sort.Key == FactoryWorkOrderListSortSource || len(filters.Sources) > 0 {
		query = query.Joins(creatorCanvasJoin)
		joined = true
	}
	if joined {
		query = query.Select("factory_work_orders.*")
	}
	return query
}

func applyWorkOrderColumnFilters(query *gorm.DB, filters ListFactoryWorkOrdersFilters) *gorm.DB {
	if filters.ConfidenceMissing {
		query = query.Where("confidence_check.score IS NULL")
	} else if filters.MinConfidence != nil {
		query = query.Where("confidence_check.score >= ?", *filters.MinConfidence)
	}

	if len(filters.Sources) > 0 {
		query = query.Where(workOrderSourceGroupSQL()+" IN ?", filters.Sources)
	}

	return applyWorkOrderAgeFilter(query, filters.Age, time.Now())
}

func applyWorkOrderAgeFilter(query *gorm.DB, age string, now time.Time) *gorm.DB {
	switch age {
	case FactoryWorkOrderListAgeLast7Days:
		return query.Where("factory_work_orders.created_at >= ?", now.Add(-7*24*time.Hour))
	case FactoryWorkOrderListAgeLast30Days:
		return query.Where("factory_work_orders.created_at >= ?", now.Add(-30*24*time.Hour))
	case FactoryWorkOrderListAgeLast90Days:
		return query.Where("factory_work_orders.created_at >= ?", now.Add(-90*24*time.Hour))
	case FactoryWorkOrderListAgeOlderThan90Days:
		return query.Where("factory_work_orders.created_at < ?", now.Add(-90*24*time.Hour))
	default:
		return query
	}
}

func applyWorkOrderListCursor(
	tx *gorm.DB,
	factory *Factory,
	query *gorm.DB,
	filters ListFactoryWorkOrdersFilters,
) (*gorm.DB, error) {
	if filters.BeforeID == nil {
		return query, nil
	}

	sort := workOrderListSortFromFilters(filters)
	cursor, err := factory.loadWorkOrderListCursor(tx, *filters.BeforeID, sort)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errWorkOrderListCursorMissing
		}
		return nil, err
	}

	return applyWorkOrderListCursorPredicate(query, sort, cursor), nil
}

func applyWorkOrderListCursorPredicate(query *gorm.DB, sort workOrderListSort, cursor *workOrderListCursor) *gorm.DB {
	op := ">"
	if sort.Descending {
		op = "<"
	}

	switch sort.Key {
	case FactoryWorkOrderListSortConfidence:
		return applyConfidenceListCursor(query, op, cursor)
	case FactoryWorkOrderListSortSource:
		return query.Where(
			fmt.Sprintf("(%s, factory_work_orders.id) %s (?, ?)", workOrderSourceSortLabelSQL(), op),
			cursor.SourceLabel,
			cursor.ID,
		)
	case FactoryWorkOrderListSortCreated:
		return query.Where(
			fmt.Sprintf("(factory_work_orders.created_at, factory_work_orders.id) %s (?, ?)", op),
			cursor.CreatedAt,
			cursor.ID,
		)
	default:
		return query.Where(
			fmt.Sprintf("(factory_work_orders.updated_at, factory_work_orders.id) %s (?, ?)", op),
			cursor.UpdatedAt,
			cursor.ID,
		)
	}
}

func applyConfidenceListCursor(query *gorm.DB, op string, cursor *workOrderListCursor) *gorm.DB {
	if cursor.Confidence == nil {
		return query.Where(
			fmt.Sprintf("confidence_check.score IS NULL AND factory_work_orders.id %s ?", op),
			cursor.ID,
		)
	}

	return query.Where(
		fmt.Sprintf(`(
			confidence_check.score IS NULL
			OR (confidence_check.score, factory_work_orders.id) %s (?, ?)
		)`, op),
		*cursor.Confidence,
		cursor.ID,
	)
}

func applyWorkOrderListOrder(query *gorm.DB, filters ListFactoryWorkOrdersFilters) *gorm.DB {
	sort := workOrderListSortFromFilters(filters)
	direction := "ASC"
	if sort.Descending {
		direction = "DESC"
	}

	switch sort.Key {
	case FactoryWorkOrderListSortConfidence:
		return query.
			Order("(confidence_check.score IS NULL) ASC").
			Order("confidence_check.score " + direction).
			Order("factory_work_orders.id " + direction)
	case FactoryWorkOrderListSortSource:
		return query.
			Order(workOrderSourceSortLabelSQL() + " " + direction).
			Order("factory_work_orders.id " + direction)
	case FactoryWorkOrderListSortCreated:
		return query.
			Order("factory_work_orders.created_at " + direction).
			Order("factory_work_orders.id " + direction)
	default:
		return query.
			Order("factory_work_orders.updated_at " + direction).
			Order("factory_work_orders.id " + direction)
	}
}

func (f *Factory) loadWorkOrderListCursor(
	tx *gorm.DB,
	beforeID uuid.UUID,
	sort workOrderListSort,
) (*workOrderListCursor, error) {
	selectSQL := "factory_work_orders.id, factory_work_orders.updated_at, factory_work_orders.created_at"
	query := tx.Table("factory_work_orders").
		Where("factory_work_orders.organization_id = ?", f.OrganizationID).
		Where("factory_work_orders.factory_id = ?", f.ID).
		Where("factory_work_orders.id = ?", beforeID)

	switch sort.Key {
	case FactoryWorkOrderListSortConfidence:
		query = query.Joins(confidenceCheckJoin, PlanningConfidenceCheckKey)
		selectSQL += ", confidence_check.score AS confidence"
	case FactoryWorkOrderListSortSource:
		query = query.Joins(creatorCanvasJoin)
		selectSQL += ", " + workOrderSourceSortLabelSQL() + " AS source_label"
	}

	var cursor workOrderListCursor
	err := query.Select(selectSQL).Take(&cursor).Error
	if err != nil {
		return nil, err
	}
	return &cursor, nil
}

func workOrderSourceGroupSQL() string {
	host := workOrderOriginHostSQL
	creator := workOrderCreatorLabelSQL
	return fmt.Sprintf(`CASE
		WHEN nullif(btrim(factory_work_orders.origin_url), '') IS NOT NULL THEN
			CASE
				WHEN %s LIKE '%%sentry.io%%' THEN '%s'
				WHEN %s LIKE '%%slack.com%%' THEN '%s'
				WHEN %s LIKE '%%atlassian.net%%' OR %s LIKE '%%jira.com%%' OR %s LIKE '%%jira%%' THEN '%s'
				WHEN %s LIKE '%%productive%%' THEN '%s'
				WHEN %s LIKE '%%pagerduty%%' THEN '%s'
				ELSE '%s'
			END
		WHEN creator_run.id IS NOT NULL THEN
			CASE
				WHEN %s LIKE '%%sentry%%' THEN '%s'
				WHEN %s LIKE '%%slack%%' THEN '%s'
				WHEN %s LIKE '%%jira%%' THEN '%s'
				WHEN %s LIKE '%%productive%%' THEN '%s'
				WHEN %s LIKE '%%pagerduty%%' THEN '%s'
				ELSE '%s'
			END
		ELSE '%s'
	END`,
		host, FactoryWorkOrderSourceSentryExceptions,
		host, FactoryWorkOrderSourceSlack,
		host, host, host, FactoryWorkOrderSourceJiraIssues,
		host, FactoryWorkOrderSourceProductiveTasks,
		host, FactoryWorkOrderSourcePagerDutyIncidents,
		FactoryWorkOrderSourceGitHubIssues,
		creator, FactoryWorkOrderSourceSentryExceptions,
		creator, FactoryWorkOrderSourceSlack,
		creator, FactoryWorkOrderSourceJiraIssues,
		creator, FactoryWorkOrderSourceProductiveTasks,
		creator, FactoryWorkOrderSourcePagerDutyIncidents,
		FactoryWorkOrderSourceGitHubIssues,
		FactoryWorkOrderSourceManual,
	)
}

func workOrderSourceSortLabelSQL() string {
	group := workOrderSourceGroupSQL()
	return fmt.Sprintf(`CASE %s
		WHEN '%s' THEN 'Created manually'
		WHEN '%s' THEN 'GitHub issues'
		WHEN '%s' THEN 'Jira issues'
		WHEN '%s' THEN 'PagerDuty incidents'
		WHEN '%s' THEN 'Productive tasks'
		WHEN '%s' THEN 'Sentry exceptions'
		WHEN '%s' THEN 'Slack'
		ELSE %s
	END`,
		group,
		FactoryWorkOrderSourceManual,
		FactoryWorkOrderSourceGitHubIssues,
		FactoryWorkOrderSourceJiraIssues,
		FactoryWorkOrderSourcePagerDutyIncidents,
		FactoryWorkOrderSourceProductiveTasks,
		FactoryWorkOrderSourceSentryExceptions,
		FactoryWorkOrderSourceSlack,
		group,
	)
}
