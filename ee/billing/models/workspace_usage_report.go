package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/ee/billing/pricebook"
	"github.com/superplanehq/superplane/pkg/billing"
	coremodels "github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type UsageTotals = billing.UsageTotals
type UsageByModel = billing.UsageByModel
type UsageByMachineType = billing.UsageByMachineType
type UsageSplit = billing.UsageSplit

const usageSumSelect = "COALESCE(SUM(total_tokens), 0) AS total_tokens, COALESCE(SUM(duration_seconds), 0) AS duration_seconds, COALESCE(SUM(cost_micros), 0) AS cost_micros"

type UsageReportFilter = billing.UsageReportFilter
type WorkOrderRunUsage = billing.WorkOrderRunUsage

func usageReportQuery(tx *gorm.DB, filter UsageReportFilter) *gorm.DB {
	joinWorkOrders := filter.TaskOwnerID != nil
	return spendingScopedQuery(tx, filter, joinWorkOrders)
}

func spendingScopedQuery(tx *gorm.DB, filter UsageReportFilter, joinWorkOrders bool) *gorm.DB {
	query := tx.Model(&coremodels.WorkspaceUsageEvent{})
	if joinWorkOrders {
		query = query.Joins("LEFT JOIN factory_work_orders ON factory_work_orders.id = workspace_usage_events.work_order_id")
		if filter.TaskOwnerID != nil {
			query = query.Where("factory_work_orders.created_by_id = ?", *filter.TaskOwnerID)
		}
	}
	if filter.OrganizationID != uuid.Nil {
		query = query.Where("workspace_usage_events.organization_id = ?", filter.OrganizationID)
	}
	if filter.FactoryID != nil {
		query = query.Where("workspace_usage_events.factory_id = ?", *filter.FactoryID)
	}
	if filter.WorkOrderID != nil {
		query = query.Where("workspace_usage_events.work_order_id = ?", *filter.WorkOrderID)
	}
	if filter.UsageKind != "" {
		query = query.Where("workspace_usage_events.usage_kind = ?", filter.UsageKind)
	}
	if filter.Provider != "" {
		query = query.Where("workspace_usage_events.provider = ?", filter.Provider)
	}
	if filter.Model != "" {
		query = query.Where("workspace_usage_events.model = ?", filter.Model)
	}
	if filter.MachineType != "" {
		query = query.Where("workspace_usage_events.machine_type = ?", filter.MachineType)
	}
	if filter.FundingSource != "" {
		if filter.FundingSource == coremodels.UsageFundingSourceHosted {
			query = query.Where("workspace_usage_events.funding_source = ?", coremodels.UsageFundingSourceHosted)
		} else {
			query = query.Where("workspace_usage_events.funding_source IS DISTINCT FROM ?", coremodels.UsageFundingSourceHosted)
		}
	}
	if !filter.Since.IsZero() {
		query = query.Where("workspace_usage_events.occurred_at >= ?", filter.Since)
	}
	if !filter.Until.IsZero() {
		query = query.Where("workspace_usage_events.occurred_at < ?", filter.Until)
	}
	return query
}

// SummarizeUsageByLocalDay returns ledger spend for the filter, grouped into
// the local days between bounds. bounds is the sorted starts of those days
// plus the exclusive end of the last day. An event before the first start, or
// at or after the last end, is ignored.
//
// The database groups by those exact instants. It does not load each event,
// and it does not use a SQL time zone. time.Local has no IANA name, and a
// daylight-saving day is not 24 hours.
//
// A kind other than compute counts as model spend, so the two bands add up to
// the filtered ledger.
func SummarizeUsageByLocalDay(tx *gorm.DB, filter UsageReportFilter, bounds []time.Time) ([]UsageSplit, error) {
	days := len(bounds) - 1
	if days <= 0 {
		return []UsageSplit{}, nil
	}
	if err := validateUsageDayBounds(bounds); err != nil {
		return nil, err
	}

	scoped := filter
	if scoped.Since.IsZero() {
		scoped.Since = bounds[0]
	}
	if scoped.Until.IsZero() {
		scoped.Until = bounds[len(bounds)-1]
	}

	dayIndex := usageLocalDayIndexExpr(bounds)
	var rows []usageDaySumRow
	err := spendingScopedQuery(tx, scoped, false).
		Select(dayIndex + " AS day_index, usage_kind, " + usageSumSelect).
		Group("day_index, usage_kind").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]UsageSplit, days)
	for _, row := range rows {
		if row.DayIndex < 0 || row.DayIndex >= days {
			continue
		}
		totals := UsageTotals{
			TotalTokens:     row.TotalTokens,
			DurationSeconds: row.DurationSeconds,
			CostMicros:      row.CostMicros,
		}
		if row.UsageKind == coremodels.UsageKindCompute {
			result[row.DayIndex].Compute = result[row.DayIndex].Compute.Add(totals)
		} else {
			result[row.DayIndex].Model = result[row.DayIndex].Model.Add(totals)
		}
	}
	return result, nil
}

type usageDaySumRow struct {
	DayIndex        int
	UsageKind       string
	TotalTokens     int64
	DurationSeconds int64
	CostMicros      int64
}

// usageLocalDayIndexExpr assigns each event to a bound by instant. The last
// bound is the exclusive end, so an event there is -1 and is not a day.
func usageLocalDayIndexExpr(bounds []time.Time) string {
	var expr strings.Builder
	expr.WriteString("CASE")
	for i := 0; i < len(bounds)-1; i++ {
		fmt.Fprintf(
			&expr,
			" WHEN workspace_usage_events.occurred_at >= %s AND workspace_usage_events.occurred_at < %s THEN %d",
			sqlTimestamptz(bounds[i]),
			sqlTimestamptz(bounds[i+1]),
			i,
		)
	}
	expr.WriteString(" ELSE -1 END")
	return expr.String()
}

func sqlTimestamptz(at time.Time) string {
	return "'" + at.UTC().Format(time.RFC3339Nano) + "'::timestamptz"
}

func validateUsageDayBounds(bounds []time.Time) error {
	for i := 1; i < len(bounds); i++ {
		if !bounds[i].After(bounds[i-1]) {
			return fmt.Errorf("usage day bounds must be strictly increasing")
		}
	}
	return nil
}

func modelUsageReportQuery(tx *gorm.DB, filter UsageReportFilter) *gorm.DB {
	filter.UsageKind = coremodels.UsageKindModel
	return usageReportQuery(tx, filter)
}

func computeUsageReportQuery(tx *gorm.DB, filter UsageReportFilter) *gorm.DB {
	filter.UsageKind = coremodels.UsageKindCompute
	return usageReportQuery(tx, filter)
}

// SummarizeUsage returns org or workspace model totals and a per-model breakdown.
func SummarizeUsage(tx *gorm.DB, filter UsageReportFilter) (UsageTotals, []UsageByModel, error) {
	var totals UsageTotals
	err := modelUsageReportQuery(tx, filter).
		Select(usageSumSelect).
		Scan(&totals).Error
	if err != nil {
		return UsageTotals{}, nil, err
	}

	var byModel []UsageByModel
	err = modelUsageReportQuery(tx, filter).
		Select("provider, model, COALESCE(SUM(total_tokens), 0) AS total_tokens, COALESCE(SUM(cost_micros), 0) AS cost_micros").
		Group("provider, model").
		Order("cost_micros DESC").
		Order("total_tokens DESC").
		Order("provider ASC").
		Order("model ASC").
		Scan(&byModel).Error
	if err != nil {
		return UsageTotals{}, nil, err
	}

	return totals, byModel, nil
}

// SummarizeComputeUsage returns org or workspace VM totals and a per-machine-type breakdown.
func SummarizeComputeUsage(tx *gorm.DB, filter UsageReportFilter) (UsageTotals, []UsageByMachineType, error) {
	var totals UsageTotals
	err := computeUsageReportQuery(tx, filter).
		Select(usageSumSelect).
		Scan(&totals).Error
	if err != nil {
		return UsageTotals{}, nil, err
	}

	var byMachine []UsageByMachineType
	err = computeUsageReportQuery(tx, filter).
		Select("machine_type, COALESCE(SUM(duration_seconds), 0) AS duration_seconds, COALESCE(SUM(cost_micros), 0) AS cost_micros").
		Group("machine_type").
		Order("cost_micros DESC").
		Order("duration_seconds DESC").
		Order("machine_type ASC").
		Scan(&byMachine).Error
	if err != nil {
		return UsageTotals{}, nil, err
	}

	return totals, byMachine, nil
}

const workOrderRunUsageGroupKey = `COALESCE(workspace_usage_events.work_order_execution_id, workspace_usage_events.canvas_run_id)`

const workOrderRunUsageSelect = `
	workspace_usage_events.work_order_execution_id,
	factory_work_orders.id AS work_order_id,
	factory_work_orders.number AS work_order_number,
	factory_work_orders.title AS title,
	MAX(workspace_usage_events.occurred_at) AS last_occurred_at,
	first_assignee.user_id AS user_id,
	COALESCE(users.name, '') AS user_name,
	COALESCE(users.email, '') AS user_email,
	COALESCE(SUM(workspace_usage_events.total_tokens), 0) AS total_tokens,
	COALESCE(SUM(workspace_usage_events.duration_seconds), 0) AS duration_seconds,
	COALESCE(SUM(workspace_usage_events.cost_micros), 0) AS cost_micros,
	COALESCE(SUM(CASE WHEN workspace_usage_events.funding_source = '` + coremodels.UsageFundingSourceHosted + `' AND workspace_usage_events.usage_kind = '` + coremodels.UsageKindModel + `' THEN workspace_usage_events.cost_micros ELSE 0 END), 0) AS hosted_cost_micros,
	COALESCE(SUM(CASE WHEN workspace_usage_events.funding_source = '` + coremodels.UsageFundingSourceBYOK + `' THEN workspace_usage_events.cost_micros ELSE 0 END), 0) AS byok_cost_micros,
	COALESCE(STRING_AGG(DISTINCT CASE WHEN workspace_usage_events.usage_kind = '` + coremodels.UsageKindModel + `' AND workspace_usage_events.funding_source IS DISTINCT FROM '` + coremodels.UsageFundingSourceBYOK + `' THEN workspace_usage_events.model END, E'\n'), '') AS models,
	COALESCE(STRING_AGG(DISTINCT CASE WHEN workspace_usage_events.usage_kind = '` + coremodels.UsageKindModel + `' AND workspace_usage_events.funding_source = '` + coremodels.UsageFundingSourceBYOK + `' THEN workspace_usage_events.model END, E'\n'), '') AS byok_models,
	COALESCE(STRING_AGG(DISTINCT CASE WHEN workspace_usage_events.usage_kind = '` + coremodels.UsageKindCompute + `' AND workspace_usage_events.machine_type <> '' THEN workspace_usage_events.machine_type END, E'\n'), '') AS machine_types`

type workOrderRunUsageScanRow struct {
	WorkOrderExecutionID uuid.UUID
	WorkOrderID          uuid.UUID
	WorkOrderNumber      int64
	Title                string
	LastOccurredAt       time.Time
	UserID               *uuid.UUID
	UserName             string
	UserEmail            string
	TotalTokens          int64
	DurationSeconds      int64
	CostMicros           int64
	HostedCostMicros     int64
	BYOKCostMicros       int64
	Models               string
	BYOKModels           string
	MachineTypes         string
}

// First assignee by assignment time, then user id. Usage names the owner,
// not the creator. Analysis rows with no assignee stay empty.
const firstWorkOrderAssigneeJoin = `LEFT JOIN LATERAL (
	SELECT user_id
	FROM factory_work_order_assignees
	WHERE work_order_id = factory_work_orders.id
	ORDER BY created_at ASC, user_id ASC
	LIMIT 1
) first_assignee ON TRUE`

func workOrderRunUsageQuery(tx *gorm.DB, filter UsageReportFilter) *gorm.DB {
	return spendingScopedQuery(tx, filter, true).
		Joins(firstWorkOrderAssigneeJoin).
		Joins("LEFT JOIN users ON users.id = first_assignee.user_id").
		Where("workspace_usage_events.work_order_id IS NOT NULL")
}

const workOrderRunUsageGroupBy = workOrderRunUsageGroupKey + `, workspace_usage_events.work_order_execution_id, factory_work_orders.id, factory_work_orders.number, factory_work_orders.title, first_assignee.user_id, users.name, users.email`

// ListWorkOrderRunUsage returns paginated task-run spend from the ledger.
// It does not write usage or change remaining hosted credit.
func ListWorkOrderRunUsage(tx *gorm.DB, filter UsageReportFilter, limit, offset int) ([]WorkOrderRunUsage, int64, error) {
	var totalRow struct {
		Count int64
	}
	err := workOrderRunUsageQuery(tx, filter).
		Select("COUNT(DISTINCT " + workOrderRunUsageGroupKey + ") AS count").
		Scan(&totalRow).Error
	if err != nil {
		return nil, 0, err
	}
	total := totalRow.Count

	var rows []workOrderRunUsageScanRow
	err = workOrderRunUsageQuery(tx, filter).
		Select(workOrderRunUsageSelect).
		Group(workOrderRunUsageGroupBy).
		Order("MAX(workspace_usage_events.occurred_at) DESC").
		Order("workspace_usage_events.work_order_execution_id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	return workOrderRunUsageRowsFromScan(rows), total, nil
}

// ListAllWorkOrderRunUsage returns every task-run spend row in the ledger for
// the filter's scope, with no date window and no pagination. It backs the
// full-history CSV export; the paginated ListWorkOrderRunUsage stays bound to
// a reporting period and page size for the table.
func ListAllWorkOrderRunUsage(tx *gorm.DB, filter UsageReportFilter) ([]WorkOrderRunUsage, error) {
	filter.Since = time.Time{}
	filter.Until = time.Time{}

	var rows []workOrderRunUsageScanRow
	err := workOrderRunUsageQuery(tx, filter).
		Select(workOrderRunUsageSelect).
		Group(workOrderRunUsageGroupBy).
		Order("MAX(workspace_usage_events.occurred_at) DESC").
		Order("workspace_usage_events.work_order_execution_id DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	return workOrderRunUsageRowsFromScan(rows), nil
}

func workOrderRunUsageRowsFromScan(rows []workOrderRunUsageScanRow) []WorkOrderRunUsage {
	result := make([]WorkOrderRunUsage, 0, len(rows))
	for _, row := range rows {
		result = append(result, WorkOrderRunUsage{
			WorkOrderExecutionID: row.WorkOrderExecutionID,
			WorkOrderID:          row.WorkOrderID,
			WorkOrderNumber:      row.WorkOrderNumber,
			Title:                row.Title,
			LastOccurredAt:       row.LastOccurredAt,
			UserID:               row.UserID,
			UserName:             row.UserName,
			UserEmail:            row.UserEmail,
			TotalTokens:          row.TotalTokens,
			DurationSeconds:      row.DurationSeconds,
			CostMicros:           row.CostMicros,
			HostedCostMicros:     row.HostedCostMicros,
			BYOKCostMicros:       row.BYOKCostMicros,
			Models:               splitUsageModelAgg(row.Models),
			BYOKModels:           splitUsageModelAgg(row.BYOKModels),
			MachineTypes:         splitUsageAgg(row.MachineTypes),
		})
	}
	return result
}

func splitUsageAgg(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, "\n")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func splitUsageModelAgg(value string) []string {
	parts := splitUsageAgg(value)
	if len(parts) == 0 {
		return nil
	}
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		model := pricebook.CatalogModelID(part)
		if model == "" {
			continue
		}
		if _, dup := seen[model]; dup {
			continue
		}
		seen[model] = struct{}{}
		out = append(out, model)
	}
	return out
}
