package models

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	WorkOrderListSortUpdated    = "updated"
	WorkOrderListSortConfidence = "confidence"
	WorkOrderListSortSource     = "source"
	WorkOrderListSortCreated    = "created"

	WorkOrderListDirectionAsc  = "asc"
	WorkOrderListDirectionDesc = "desc"

	WorkOrderListAgeLast7Days       = "last_7_days"
	WorkOrderListAgeLast30Days      = "last_30_days"
	WorkOrderListAgeLast90Days      = "last_90_days"
	WorkOrderListAgeOlderThan90Days = "older_than_90_days"

	WorkOrderSourceGroupManual             = "manual"
	WorkOrderSourceGroupGitHubIssues       = "github-issues"
	WorkOrderSourceGroupJiraIssues         = "jira-issues"
	WorkOrderSourceGroupPagerDutyIncidents = "pagerduty-incidents"
	WorkOrderSourceGroupProductiveTasks    = "productive-tasks"
	WorkOrderSourceGroupSentryExceptions   = "sentry-exceptions"
	WorkOrderSourceGroupSlack              = "slack"
)

var workOrderSourceGroupRank = map[string]int{
	WorkOrderSourceGroupManual:             1,
	WorkOrderSourceGroupGitHubIssues:       2,
	WorkOrderSourceGroupJiraIssues:         3,
	WorkOrderSourceGroupPagerDutyIncidents: 4,
	WorkOrderSourceGroupProductiveTasks:    5,
	WorkOrderSourceGroupSentryExceptions:   6,
	WorkOrderSourceGroupSlack:              7,
}

var (
	workOrderListSQLOnce        sync.Once
	workOrderSourceGroupSQL     string
	workOrderSourceRankSQL      string
	workOrderConfidenceScoreSQL string
)

func ensureWorkOrderListSQL() {
	workOrderListSQLOnce.Do(func() {
		workOrderSourceGroupSQL = buildWorkOrderSourceGroupSQL()
		workOrderSourceRankSQL = buildWorkOrderSourceRankSQL()
		workOrderConfidenceScoreSQL = buildWorkOrderConfidenceScoreSQL()
	})
}

func normalizeWorkOrderSourceGroups(groups []string) ([]string, bool) {
	if len(groups) == 0 {
		return nil, true
	}

	known := make([]string, 0, len(groups))
	seen := map[string]struct{}{}
	for _, group := range groups {
		if _, ok := workOrderSourceGroupRank[group]; !ok {
			continue
		}
		if _, dup := seen[group]; dup {
			continue
		}
		seen[group] = struct{}{}
		known = append(known, group)
	}
	if len(known) == 0 {
		return nil, false
	}
	return known, true
}

func applyWorkOrderListFilters(query *gorm.DB, filters ListFactoryWorkOrdersFilters, now time.Time) *gorm.DB {
	ensureWorkOrderListSQL()
	if len(filters.SourceGroups) > 0 {
		query = query.Where(workOrderSourceGroupSQL+" IN ?", filters.SourceGroups)
	}
	if filters.MinConfidence != nil {
		query = query.Where(workOrderConfidenceScoreSQL+" >= ?", *filters.MinConfidence)
	}
	if filters.ConfidenceMissing {
		query = query.Where(workOrderConfidenceScoreSQL + " IS NULL")
	}
	return applyWorkOrderAgeFilter(query, filters.Age, now)
}

func applyWorkOrderAgeFilter(query *gorm.DB, age string, now time.Time) *gorm.DB {
	switch age {
	case WorkOrderListAgeLast7Days:
		return query.Where("factory_work_orders.created_at >= ?", now.Add(-7*24*time.Hour))
	case WorkOrderListAgeLast30Days:
		return query.Where("factory_work_orders.created_at >= ?", now.Add(-30*24*time.Hour))
	case WorkOrderListAgeLast90Days:
		return query.Where("factory_work_orders.created_at >= ?", now.Add(-90*24*time.Hour))
	case WorkOrderListAgeOlderThan90Days:
		return query.Where("factory_work_orders.created_at < ?", now.Add(-90*24*time.Hour))
	default:
		return query
	}
}

func applyWorkOrderListOrder(query *gorm.DB, filters ListFactoryWorkOrdersFilters) *gorm.DB {
	ensureWorkOrderListSQL()
	direction := listSortDirectionSQL(filters.SortDirection)
	switch filters.Sort {
	case WorkOrderListSortConfidence:
		return query.
			Order("(" + workOrderConfidenceScoreSQL + " IS NULL) ASC").
			Order(workOrderConfidenceScoreSQL + " " + direction).
			Order("factory_work_orders.id " + direction)
	case WorkOrderListSortSource:
		return query.
			Order(workOrderSourceRankSQL + " " + direction).
			Order("factory_work_orders.id " + direction)
	case WorkOrderListSortCreated:
		return query.
			Order("factory_work_orders.created_at " + direction).
			Order("factory_work_orders.id " + direction)
	default:
		return query.
			Order("factory_work_orders.updated_at " + direction).
			Order("factory_work_orders.id " + direction)
	}
}

func (f *Factory) applyWorkOrderListCursor(
	tx *gorm.DB,
	query *gorm.DB,
	filters ListFactoryWorkOrdersFilters,
) (*gorm.DB, bool, error) {
	if filters.BeforeID == nil {
		return query, true, nil
	}

	ensureWorkOrderListSQL()

	cursor, err := f.workOrderListCursor(tx, *filters.BeforeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return query, false, nil
		}
		return nil, false, err
	}

	descending := filters.SortDirection != WorkOrderListDirectionAsc
	switch filters.Sort {
	case WorkOrderListSortConfidence:
		score, err := workOrderConfidenceScore(tx, cursor.ID)
		if err != nil {
			return nil, false, err
		}
		return applyConfidenceListCursor(query, cursor.ID, score, descending), true, nil
	case WorkOrderListSortSource:
		rank, err := f.workOrderSourceRank(tx, cursor.ID)
		if err != nil {
			return nil, false, err
		}
		return applyKeyedListCursor(query, workOrderSourceRankSQL, rank, cursor.ID, descending), true, nil
	case WorkOrderListSortCreated:
		return applyTimeListCursor(query, "created_at", cursor.CreatedAt, cursor.ID, descending), true, nil
	default:
		return applyTimeListCursor(query, "updated_at", cursor.UpdatedAt, cursor.ID, descending), true, nil
	}
}

func (f *Factory) workOrderListCursor(tx *gorm.DB, beforeID uuid.UUID) (*FactoryWorkOrder, error) {
	var cursor FactoryWorkOrder
	err := tx.
		Select("id", "created_at", "updated_at").
		Where("factory_work_orders.organization_id = ?", f.OrganizationID).
		Where("factory_work_orders.factory_id = ?", f.ID).
		Where("factory_work_orders.id = ?", beforeID).
		Take(&cursor).Error
	if err != nil {
		return nil, err
	}
	return &cursor, nil
}

func workOrderConfidenceScore(tx *gorm.DB, orderID uuid.UUID) (*float64, error) {
	var scores []float64
	err := tx.
		Table("factory_work_order_checks").
		Where("work_order_id = ? AND key = ?", orderID, PlanningConfidenceCheckKey).
		Limit(1).
		Pluck("score", &scores).Error
	if err != nil {
		return nil, err
	}
	if len(scores) == 0 {
		return nil, nil
	}
	score := scores[0]
	return &score, nil
}

func (f *Factory) workOrderSourceRank(tx *gorm.DB, orderID uuid.UUID) (int, error) {
	var rank int
	err := tx.Raw(
		"SELECT "+workOrderSourceRankSQL+" FROM factory_work_orders WHERE organization_id = ? AND factory_id = ? AND id = ?",
		f.OrganizationID,
		f.ID,
		orderID,
	).Scan(&rank).Error
	if err != nil {
		return 0, err
	}
	return rank, nil
}

func applyTimeListCursor(query *gorm.DB, column string, at time.Time, id uuid.UUID, descending bool) *gorm.DB {
	operator := ">"
	if descending {
		operator = "<"
	}
	return query.Where(
		fmt.Sprintf("(factory_work_orders.%s, factory_work_orders.id) %s (?, ?)", column, operator),
		at,
		id,
	)
}

func applyKeyedListCursor(query *gorm.DB, keySQL string, key int, id uuid.UUID, descending bool) *gorm.DB {
	operator := ">"
	if descending {
		operator = "<"
	}
	return query.Where(
		fmt.Sprintf("(%s, factory_work_orders.id) %s (?, ?)", keySQL, operator),
		key,
		id,
	)
}

func applyConfidenceListCursor(query *gorm.DB, id uuid.UUID, score *float64, descending bool) *gorm.DB {
	scoreSQL := workOrderConfidenceScoreSQL
	operator := ">"
	if descending {
		operator = "<"
	}
	return query.Where(fmt.Sprintf(`(
		(?::double precision IS NOT NULL AND %[1]s IS NOT NULL AND (%[1]s, factory_work_orders.id) %[2]s (?::double precision, ?::uuid))
		OR (?::double precision IS NOT NULL AND %[1]s IS NULL)
		OR (?::double precision IS NULL AND %[1]s IS NULL AND factory_work_orders.id %[2]s ?::uuid)
	)`, scoreSQL, operator),
		score,
		score,
		id,
		score,
		score,
		id,
	)
}

func listSortDirectionSQL(direction string) string {
	if direction == WorkOrderListDirectionAsc {
		return "ASC"
	}
	return "DESC"
}

func buildWorkOrderConfidenceScoreSQL() string {
	return fmt.Sprintf(`(
		SELECT factory_work_order_checks.score
		FROM factory_work_order_checks
		WHERE factory_work_order_checks.work_order_id = factory_work_orders.id
			AND factory_work_order_checks.key = '%s'
		LIMIT 1
	)`, PlanningConfidenceCheckKey)
}

func buildWorkOrderSourceGroupSQL() string {
	return fmt.Sprintf(`COALESCE(
		CASE WHEN NULLIF(BTRIM(factory_work_orders.origin_url), '') IS NOT NULL THEN %s END,
		%s,
		'%s'
	)`, buildWorkOrderOriginGroupSQL(), buildWorkOrderAutomationGroupSQL(), WorkOrderSourceGroupManual)
}

func buildWorkOrderOriginGroupSQL() string {
	host := `lower(split_part(split_part(regexp_replace(btrim(factory_work_orders.origin_url), '^[a-zA-Z][a-zA-Z0-9+.-]*://', ''), '/', 1), ':', 1))`
	return fmt.Sprintf(`CASE
		WHEN %[1]s LIKE '%%sentry.io%%' THEN '%[2]s'
		WHEN %[1]s LIKE '%%slack.com%%' THEN '%[3]s'
		WHEN %[1]s LIKE '%%atlassian.net%%' OR %[1]s LIKE '%%jira%%' THEN '%[4]s'
		WHEN %[1]s LIKE '%%productive%%' THEN '%[5]s'
		WHEN %[1]s LIKE '%%pagerduty%%' THEN '%[6]s'
		ELSE '%[7]s'
	END`,
		host,
		WorkOrderSourceGroupSentryExceptions,
		WorkOrderSourceGroupSlack,
		WorkOrderSourceGroupJiraIssues,
		WorkOrderSourceGroupProductiveTasks,
		WorkOrderSourceGroupPagerDutyIncidents,
		WorkOrderSourceGroupGitHubIssues,
	)
}

func buildWorkOrderAutomationGroupSQL() string {
	label := `lower(workflow_runs.workflow_id::text || ' ' || COALESCE(workflows.name, ''))`
	return fmt.Sprintf(`(
		SELECT CASE
			WHEN %[1]s LIKE '%%sentry%%' THEN '%[2]s'
			WHEN %[1]s LIKE '%%slack%%' THEN '%[3]s'
			WHEN %[1]s LIKE '%%jira%%' THEN '%[4]s'
			WHEN %[1]s LIKE '%%productive%%' THEN '%[5]s'
			WHEN %[1]s LIKE '%%pagerduty%%' THEN '%[6]s'
			ELSE '%[7]s'
		END
		FROM workflow_runs
		LEFT JOIN workflows ON workflows.id = workflow_runs.workflow_id
		WHERE workflow_runs.id = factory_work_orders.source_run_id
		LIMIT 1
	)`,
		label,
		WorkOrderSourceGroupSentryExceptions,
		WorkOrderSourceGroupSlack,
		WorkOrderSourceGroupJiraIssues,
		WorkOrderSourceGroupProductiveTasks,
		WorkOrderSourceGroupPagerDutyIncidents,
		WorkOrderSourceGroupGitHubIssues,
	)
}

func buildWorkOrderSourceRankSQL() string {
	groupSQL := buildWorkOrderSourceGroupSQL()
	type rankedGroup struct {
		name string
		rank int
	}
	groups := make([]rankedGroup, 0, len(workOrderSourceGroupRank))
	for name, rank := range workOrderSourceGroupRank {
		groups = append(groups, rankedGroup{name: name, rank: rank})
	}
	slices.SortFunc(groups, func(a, b rankedGroup) int {
		return a.rank - b.rank
	})

	sql := "CASE " + groupSQL
	for _, group := range groups {
		sql += fmt.Sprintf(" WHEN '%s' THEN %d", group.name, group.rank)
	}
	return sql + " ELSE 99 END"
}
