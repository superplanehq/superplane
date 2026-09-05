package models

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
	"gorm.io/gorm"
)

const (
	SpendingGroupByWorkspace = "workspace"
	SpendingGroupByUser      = "user"
	SpendingGroupByModel     = "model"
	SpendingGroupByMachine   = "machine"

	SpendingTimeGrainHour  = "hour"
	SpendingTimeGrainDay   = "day"
	SpendingTimeGrainMonth = "month"

	spendingOtherSeriesID = "other"
)

const spendingReportMaxSpan = 366 * 24 * time.Hour

// SpendingKPITotals is the org-wide spend summary for a time window.
type SpendingKPITotals struct {
	CostMicros       int64
	TotalTokens      int64
	DurationSeconds  int64
	HostedCostMicros int64
	BYOKCostMicros   int64
}

func (t SpendingKPITotals) CostCents() int64 {
	return pricebook.MicrosToCents(t.CostMicros)
}

func (t SpendingKPITotals) HostedCostCents() int64 {
	return pricebook.MicrosToCents(t.HostedCostMicros)
}

func (t SpendingKPITotals) BYOKCostCents() int64 {
	return pricebook.MicrosToCents(t.BYOKCostMicros)
}

// SpendingBreakdownRow is one grouped row in the spending explorer table.
type SpendingBreakdownRow struct {
	ID              string
	TotalTokens     int64
	DurationSeconds int64
	CostMicros      int64
}

func (r SpendingBreakdownRow) CostCents() int64 {
	return pricebook.MicrosToCents(r.CostMicros)
}

// SpendingSeriesKey identifies one stacked chart segment.
type SpendingSeriesKey struct {
	ID    string
	Label string
}

// SpendingSeriesPoint is one time bucket in the stacked spending chart.
type SpendingSeriesPoint struct {
	Key        string
	Label      string
	TotalCents int64
	Values     map[string]int64
}

// SpendingCatalogItem is one filter or label option in the spending explorer.
type SpendingCatalogItem struct {
	ID    string
	Label string
}

// SpendingCatalogs lists filter options for the spending explorer.
type SpendingCatalogs struct {
	Workspaces []SpendingCatalogItem
	Users      []SpendingCatalogItem
	Models     []SpendingCatalogItem
	Machines   []SpendingCatalogItem
}

// SpendingExplorerReport is the filtered chart and table payload for one usage kind.
type SpendingExplorerReport struct {
	Totals     UsageTotals
	Breakdown  []SpendingBreakdownRow
	SeriesKeys []SpendingSeriesKey
	Series     []SpendingSeriesPoint
}

// ValidateSpendingReportWindow rejects windows longer than one year.
func ValidateSpendingReportWindow(since, until time.Time) error {
	if until.Before(since) || until.Equal(since) {
		return fmt.Errorf("spending report end time must be after start time")
	}
	if until.Sub(since) > spendingReportMaxSpan {
		return fmt.Errorf("spending report window cannot exceed 366 days")
	}
	return nil
}

// SummarizeSpendingKPITotals returns org-wide totals for model and compute usage in a window.
func SummarizeSpendingKPITotals(tx *gorm.DB, filter UsageReportFilter) (SpendingKPITotals, error) {
	var row struct {
		CostMicros       int64
		TotalTokens      int64
		DurationSeconds  int64
		HostedCostMicros int64
		BYOKCostMicros   int64
	}
	err := spendingScopedQuery(tx, filter, false).
		Select(`
			COALESCE(SUM(cost_micros), 0) AS cost_micros,
			COALESCE(SUM(total_tokens), 0) AS total_tokens,
			COALESCE(SUM(duration_seconds), 0) AS duration_seconds,
			COALESCE(SUM(CASE WHEN funding_source = ? THEN cost_micros ELSE 0 END), 0) AS hosted_cost_micros,
			COALESCE(SUM(CASE WHEN funding_source != ? THEN cost_micros ELSE 0 END), 0) AS byok_cost_micros`,
			UsageFundingSourceHosted, UsageFundingSourceHosted).
		Scan(&row).Error
	if err != nil {
		return SpendingKPITotals{}, err
	}
	return SpendingKPITotals{
		CostMicros:       row.CostMicros,
		TotalTokens:      row.TotalTokens,
		DurationSeconds:  row.DurationSeconds,
		HostedCostMicros: row.HostedCostMicros,
		BYOKCostMicros:   row.BYOKCostMicros,
	}, nil
}

// SummarizeSpendingExplorer returns filtered totals, breakdown rows, and stacked series.
func SummarizeSpendingExplorer(
	tx *gorm.DB,
	filter UsageReportFilter,
	groupBy string,
	grain string,
) (SpendingExplorerReport, error) {
	groupBy = normalizeSpendingGroupBy(groupBy, filter.UsageKind)
	grain = normalizeSpendingTimeGrain(grain)
	joinWorkOrders := groupBy == SpendingGroupByUser || filter.TaskOwnerID != nil

	breakdown, err := summarizeSpendingBreakdown(tx, filter, groupBy, joinWorkOrders)
	if err != nil {
		return SpendingExplorerReport{}, err
	}

	var totals UsageTotals
	err = spendingScopedQuery(tx, filter, joinWorkOrders).
		Select(usageSumSelect).
		Scan(&totals).Error
	if err != nil {
		return SpendingExplorerReport{}, err
	}

	seriesKeys := seriesKeysFromBreakdown(breakdown, groupBy)
	series, err := summarizeSpendingSeries(tx, filter, groupBy, grain, joinWorkOrders, seriesKeys, sinceFromFilter(filter), untilFromFilter(filter))
	if err != nil {
		return SpendingExplorerReport{}, err
	}

	return SpendingExplorerReport{
		Totals:     totals,
		Breakdown:  breakdown,
		SeriesKeys: seriesKeys,
		Series:     series,
	}, nil
}

// ListSpendingFilterCatalogs returns workspace, user, model, and machine filter options.
func ListSpendingFilterCatalogs(tx *gorm.DB, filter UsageReportFilter) (SpendingCatalogs, error) {
	workspaces, err := listSpendingWorkspaceCatalog(tx, filter)
	if err != nil {
		return SpendingCatalogs{}, err
	}
	users, err := listSpendingUserCatalog(tx, filter)
	if err != nil {
		return SpendingCatalogs{}, err
	}
	modelsCatalog, err := listSpendingModelCatalog(tx, filter)
	if err != nil {
		return SpendingCatalogs{}, err
	}
	machines, err := listSpendingMachineCatalog(tx, filter)
	if err != nil {
		return SpendingCatalogs{}, err
	}
	return SpendingCatalogs{
		Workspaces: workspaces,
		Users:      users,
		Models:     modelsCatalog,
		Machines:   machines,
	}, nil
}

func summarizeSpendingBreakdown(
	tx *gorm.DB,
	filter UsageReportFilter,
	groupBy string,
	joinWorkOrders bool,
) ([]SpendingBreakdownRow, error) {
	idExpr, groupExpr := spendingGroupExpressions(groupBy)
	var rows []SpendingBreakdownRow
	err := spendingScopedQuery(tx, filter, joinWorkOrders).
		Select(fmt.Sprintf(`
			%s AS id,
			COALESCE(SUM(total_tokens), 0) AS total_tokens,
			COALESCE(SUM(duration_seconds), 0) AS duration_seconds,
			COALESCE(SUM(cost_micros), 0) AS cost_micros`, idExpr)).
		Group(groupExpr).
		Order("cost_micros DESC").
		Order("total_tokens DESC").
		Order("duration_seconds DESC").
		Order("id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

type spendingSeriesBucketRow struct {
	BucketKey  string
	SeriesID   string
	CostMicros int64
}

func summarizeSpendingSeries(
	tx *gorm.DB,
	filter UsageReportFilter,
	groupBy string,
	grain string,
	joinWorkOrders bool,
	seriesKeys []SpendingSeriesKey,
	since time.Time,
	until time.Time,
) ([]SpendingSeriesPoint, error) {
	knownIDs := make(map[string]struct{}, len(seriesKeys))
	hasOther := false
	for _, key := range seriesKeys {
		knownIDs[key.ID] = struct{}{}
		if key.ID == spendingOtherSeriesID {
			hasOther = true
		}
	}

	idExpr, groupExpr := spendingGroupExpressions(groupBy)
	bucketExpr := spendingTimeBucketExpression(grain)
	var rows []spendingSeriesBucketRow
	err := spendingScopedQuery(tx, filter, joinWorkOrders).
		Select(fmt.Sprintf(`
			%s AS bucket_key,
			%s AS series_id,
			COALESCE(SUM(cost_micros), 0) AS cost_micros`, bucketExpr, idExpr)).
		Group("bucket_key, " + groupExpr).
		Order("bucket_key ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	points := emptySpendingSeriesBuckets(since, until, grain)
	for _, row := range rows {
		point, ok := points[row.BucketKey]
		if !ok {
			continue
		}
		seriesID := row.SeriesID
		if _, known := knownIDs[seriesID]; !known {
			if !hasOther {
				continue
			}
			seriesID = spendingOtherSeriesID
		}
		costCents := pricebook.MicrosToCents(row.CostMicros)
		point.Values[seriesID] = point.Values[seriesID] + costCents
		point.TotalCents += costCents
		points[row.BucketKey] = point
	}

	out := make([]SpendingSeriesPoint, 0, len(points))
	for _, point := range sortedSeriesPoints(points) {
		out = append(out, point)
	}
	return out, nil
}

func seriesKeysFromBreakdown(rows []SpendingBreakdownRow, groupBy string) []SpendingSeriesKey {
	top := make([]SpendingSeriesKey, 0, 6)
	for i, row := range rows {
		if i >= 5 {
			break
		}
		top = append(top, SpendingSeriesKey{ID: row.ID, Label: row.ID})
	}
	if len(rows) > 5 {
		top = append(top, SpendingSeriesKey{ID: spendingOtherSeriesID, Label: otherSpendingSeriesLabel(groupBy)})
	}
	return top
}

func listSpendingWorkspaceCatalog(tx *gorm.DB, filter UsageReportFilter) ([]SpendingCatalogItem, error) {
	type row struct {
		ID   uuid.UUID
		Name string
	}
	var rows []row
	err := spendingScopedQuery(tx, filter, false).
		Select("factories.id AS id, factories.name AS name").
		Joins("JOIN factories ON factories.id = workspace_usage_events.factory_id").
		Where("factories.organization_id = ?", filter.OrganizationID).
		Group("factories.id, factories.name").
		Order("factories.name ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]SpendingCatalogItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, SpendingCatalogItem{ID: row.ID.String(), Label: row.Name})
	}
	return out, nil
}

func listSpendingUserCatalog(tx *gorm.DB, filter UsageReportFilter) ([]SpendingCatalogItem, error) {
	type row struct {
		ID   uuid.UUID
		Name string
	}
	var rows []row
	err := spendingScopedQuery(tx, filter, true).
		Select("users.id AS id, users.name AS name").
		Joins("JOIN users ON users.id = factory_work_orders.created_by_id").
		Where("users.organization_id = ?", filter.OrganizationID).
		Group("users.id, users.name").
		Order("users.name ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]SpendingCatalogItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, SpendingCatalogItem{ID: row.ID.String(), Label: row.Name})
	}
	return out, nil
}

func listSpendingModelCatalog(tx *gorm.DB, filter UsageReportFilter) ([]SpendingCatalogItem, error) {
	modelFilter := filter
	modelFilter.UsageKind = UsageKindModel
	var rows []spendingModelCatalogRow
	err := spendingScopedQuery(tx, modelFilter, false).
		Select("workspace_usage_events.provider AS provider, workspace_usage_events.model AS model").
		Group("workspace_usage_events.provider, workspace_usage_events.model").
		Order("workspace_usage_events.model ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	versionedIDs, err := spendingVersionedModelIDs(tx, rows)
	if err != nil {
		return nil, err
	}
	out := make([]SpendingCatalogItem, 0, len(rows))
	for _, row := range rows {
		id := row.Provider + "/" + row.Model
		out = append(out, SpendingCatalogItem{ID: id, Label: SpendingModelDisplayName(row.Model, versionedIDs)})
	}
	return out, nil
}

type spendingModelCatalogRow struct {
	Provider string
	Model    string
}

func spendingVersionedModelIDs(tx *gorm.DB, rows []spendingModelCatalogRow) ([]string, error) {
	providers, err := ListHostedLLMProviders(tx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0)
	for _, provider := range providers {
		ids = append(ids, CompactModelIDs(provider.AllowedModels)...)
	}
	for _, row := range rows {
		if spendingModelHasVersion(row.Model) {
			ids = append(ids, row.Model)
		}
	}
	return CompactModelIDs(ids), nil
}

// SpendingModelDisplayName returns a versioned model id for explorer labels.
// Filter ids stay on the stored ledger model so existing rows still match.
func SpendingModelDisplayName(storedModel string, versionedIDs []string) string {
	stored := strings.TrimSpace(storedModel)
	if stored == "" {
		return stored
	}
	alias := stored
	if _, rest, found := strings.Cut(stored, "/"); found && rest != "" && !strings.Contains(rest, "/") {
		alias = rest
	}
	if spendingModelHasVersion(alias) {
		return stored
	}
	if match := firstVersionedModelID(versionedIDs, alias); match != "" {
		return canonicalSpendingModelName(match)
	}
	return stored
}

func canonicalSpendingModelName(id string) string {
	name := strings.TrimSpace(id)
	if _, rest, found := strings.Cut(name, "/"); found && rest != "" && !strings.Contains(rest, "/") {
		return rest
	}
	return name
}

func spendingModelHasVersion(model string) bool {
	for _, r := range model {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func firstVersionedModelID(ids []string, alias string) string {
	needle := strings.ToLower(strings.TrimSpace(alias))
	if needle == "" {
		return ""
	}
	matches := make([]string, 0, len(ids))
	for _, id := range ids {
		normalized := strings.ToLower(strings.TrimSpace(id))
		if normalized == "" {
			continue
		}
		name := normalized
		if _, rest, found := strings.Cut(normalized, "/"); found && rest != "" {
			name = rest
		}
		if name == needle || !spendingModelHasVersion(name) {
			continue
		}
		if !modelNameContainsToken(name, needle) {
			continue
		}
		matches = append(matches, strings.TrimSpace(id))
	}
	if len(matches) == 0 {
		return ""
	}
	sort.Strings(matches)
	return matches[len(matches)-1]
}

func modelNameContainsToken(name, token string) bool {
	for _, part := range strings.Split(name, "-") {
		if part == token {
			return true
		}
	}
	return false
}

func listSpendingMachineCatalog(tx *gorm.DB, filter UsageReportFilter) ([]SpendingCatalogItem, error) {
	type row struct {
		MachineType string
	}
	computeFilter := filter
	computeFilter.UsageKind = UsageKindCompute
	var rows []row
	err := spendingScopedQuery(tx, computeFilter, false).
		Select("workspace_usage_events.machine_type AS machine_type").
		Group("workspace_usage_events.machine_type").
		Order("workspace_usage_events.machine_type ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]SpendingCatalogItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, SpendingCatalogItem{ID: row.MachineType, Label: row.MachineType})
	}
	return out, nil
}

func spendingGroupExpressions(groupBy string) (idExpr string, groupExpr string) {
	switch groupBy {
	case SpendingGroupByUser:
		return "COALESCE(factory_work_orders.created_by_id::text, '')", "COALESCE(factory_work_orders.created_by_id::text, '')"
	case SpendingGroupByModel:
		return "workspace_usage_events.provider || '/' || workspace_usage_events.model", "workspace_usage_events.provider, workspace_usage_events.model"
	case SpendingGroupByMachine:
		return "workspace_usage_events.machine_type", "workspace_usage_events.machine_type"
	default:
		return "COALESCE(workspace_usage_events.factory_id::text, '')", "COALESCE(workspace_usage_events.factory_id::text, '')"
	}
}

func spendingTimeBucketExpression(grain string) string {
	switch grain {
	case SpendingTimeGrainHour:
		return `to_char(date_trunc('hour', workspace_usage_events.occurred_at AT TIME ZONE 'UTC'), 'YYYY-MM-DD"T"HH24')`
	case SpendingTimeGrainMonth:
		return `to_char(date_trunc('month', workspace_usage_events.occurred_at AT TIME ZONE 'UTC'), 'YYYY-MM')`
	default:
		return `to_char(date_trunc('day', workspace_usage_events.occurred_at AT TIME ZONE 'UTC'), 'YYYY-MM-DD')`
	}
}

func normalizeSpendingGroupBy(groupBy, usageKind string) string {
	switch strings.TrimSpace(groupBy) {
	case SpendingGroupByUser:
		return SpendingGroupByUser
	case SpendingGroupByModel:
		if usageKind == UsageKindCompute {
			return SpendingGroupByWorkspace
		}
		return SpendingGroupByModel
	case SpendingGroupByMachine:
		if usageKind == UsageKindModel {
			return SpendingGroupByWorkspace
		}
		return SpendingGroupByMachine
	default:
		return SpendingGroupByWorkspace
	}
}

func normalizeSpendingTimeGrain(grain string) string {
	switch strings.TrimSpace(grain) {
	case SpendingTimeGrainHour:
		return SpendingTimeGrainHour
	case SpendingTimeGrainMonth:
		return SpendingTimeGrainMonth
	default:
		return SpendingTimeGrainDay
	}
}

func otherSpendingSeriesLabel(groupBy string) string {
	switch groupBy {
	case SpendingGroupByUser:
		return "Other users"
	case SpendingGroupByModel:
		return "Other models"
	case SpendingGroupByMachine:
		return "Other machines"
	default:
		return "Other workspaces"
	}
}

func sinceFromFilter(filter UsageReportFilter) time.Time {
	return filter.Since
}

func untilFromFilter(filter UsageReportFilter) time.Time {
	return filter.Until
}

func emptySpendingSeriesBuckets(since, until time.Time, grain string) map[string]SpendingSeriesPoint {
	points := make(map[string]SpendingSeriesPoint)
	switch grain {
	case SpendingTimeGrainHour:
		for timeCursor := since.UTC(); timeCursor.Before(until); timeCursor = timeCursor.Add(time.Hour) {
			key := hourSeriesKey(timeCursor)
			points[key] = SpendingSeriesPoint{Key: key, Label: hourSeriesLabel(timeCursor), Values: map[string]int64{}}
		}
	case SpendingTimeGrainMonth:
		cursor := time.Date(since.UTC().Year(), since.UTC().Month(), 1, 0, 0, 0, 0, time.UTC)
		for cursor.Before(until) {
			key := monthSeriesKey(cursor)
			points[key] = SpendingSeriesPoint{Key: key, Label: monthSeriesLabel(cursor), Values: map[string]int64{}}
			cursor = cursor.AddDate(0, 1, 0)
		}
	default:
		cursor := startOfUTCSpendingDay(since)
		for cursor.Before(until) {
			key := daySeriesKey(cursor)
			points[key] = SpendingSeriesPoint{Key: key, Label: daySeriesLabel(cursor), Values: map[string]int64{}}
			cursor = cursor.AddDate(0, 0, 1)
		}
	}
	return points
}

func sortedSeriesPoints(points map[string]SpendingSeriesPoint) []SpendingSeriesPoint {
	keys := make([]string, 0, len(points))
	for key := range points {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]SpendingSeriesPoint, 0, len(keys))
	for _, key := range keys {
		out = append(out, points[key])
	}
	return out
}

func startOfUTCSpendingDay(value time.Time) time.Time {
	utc := value.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

func hourSeriesKey(value time.Time) string {
	return fmt.Sprintf("%sT%02d", daySeriesKey(value), value.UTC().Hour())
}

func daySeriesKey(value time.Time) string {
	utc := value.UTC()
	return fmt.Sprintf("%04d-%02d-%02d", utc.Year(), utc.Month(), utc.Day())
}

func monthSeriesKey(value time.Time) string {
	utc := value.UTC()
	return fmt.Sprintf("%04d-%02d", utc.Year(), utc.Month())
}

func hourSeriesLabel(value time.Time) string {
	return fmt.Sprintf("%02d:00", value.UTC().Hour())
}

func daySeriesLabel(value time.Time) string {
	return value.UTC().Format("Jan 2")
}

func monthSeriesLabel(value time.Time) string {
	return value.UTC().Format("Jan 06")
}

func spendingLabelMap(catalogs SpendingCatalogs, groupBy string) map[string]string {
	items := catalogs.Workspaces
	switch groupBy {
	case SpendingGroupByUser:
		items = catalogs.Users
	case SpendingGroupByModel:
		items = catalogs.Models
	case SpendingGroupByMachine:
		items = catalogs.Machines
	}
	out := make(map[string]string, len(items))
	for _, item := range items {
		out[item.ID] = item.Label
	}
	return out
}

// SpendingBreakdownLabel resolves one breakdown row label from catalogs.
func SpendingBreakdownLabel(id string, catalogs SpendingCatalogs, groupBy string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "Unknown"
	}
	if label, ok := spendingLabelMap(catalogs, groupBy)[id]; ok {
		return label
	}
	if groupBy == SpendingGroupByModel && strings.Contains(id, "/") {
		return strings.SplitN(id, "/", 2)[1]
	}
	return id
}
