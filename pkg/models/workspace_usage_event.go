package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	UsageKindModel   = "model"
	UsageKindCompute = "compute"

	UsageFundingSourceBYOK   = "byok"
	UsageFundingSourceHosted = "hosted"

	UsageProviderAnthropic  = "anthropic"
	UsageProviderOpenAI     = "openai"
	UsageProviderOpenRouter = "openrouter"
	UsageProviderPerplexity = "perplexity"
	UsageProviderRunner     = "runner"

	UsageIdempotencyKeyRunner = "runner"
)

// WorkspaceUsageEvent is one append-only spend row (model tokens or VM
// seconds). It is the source of truth for reports. Factory execution
// token/cost/duration columns are cached rollups.
type WorkspaceUsageEvent struct {
	ID                   uuid.UUID
	OrganizationID       uuid.UUID
	FactoryID            *uuid.UUID
	WorkOrderID          *uuid.UUID
	LineID               *uuid.UUID
	LineDispatchID       *uuid.UUID
	WorkOrderExecutionID *uuid.UUID
	CanvasRunID          uuid.UUID
	NodeExecutionID      uuid.UUID
	NodeID               string
	Provider             string
	Model                string
	UsageKind            string
	FundingSource        string
	InputTokens          int64
	OutputTokens         int64
	CacheReadTokens      int64
	CacheWriteTokens     int64
	ReasoningTokens      int64
	TotalTokens          int64
	DurationSeconds      int64
	MachineType          string
	FleetID              string
	CostMicros           int64
	ProviderCostMicros   int64
	Currency             string
	PriceBookVersion     string
	IdempotencyKey       string
	OccurredAt           time.Time
	CreatedAt            time.Time
}

func (WorkspaceUsageEvent) TableName() string {
	return "workspace_usage_events"
}

// WorkspaceUsageEventInput is the call-site payload before factory scope is resolved.
type WorkspaceUsageEventInput struct {
	OrganizationID   uuid.UUID
	CanvasRunID      uuid.UUID
	NodeExecutionID  uuid.UUID
	NodeID           string
	Provider         string
	Model            string
	FundingSource    string
	InputTokens      int64
	OutputTokens     int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	ReasoningTokens  int64
	TotalTokens      int64
	CostMicros       *int64
	IdempotencyKey   string
}

// ComputeUsageEventInput is the call-site payload for one runner-fleet task.
type ComputeUsageEventInput struct {
	OrganizationID  uuid.UUID
	CanvasRunID     uuid.UUID
	NodeExecutionID uuid.UUID
	NodeID          string
	MachineType     string
	FleetID         string
	DurationSeconds int64
	IdempotencyKey  string
}

// RecordUsage inserts one factory-linked model-usage row and copies ledger
// totals into the line-step cache when the run belongs to a line execution.
// Factory canvases without a line step (Backlog analysis, PR feedback)
// still persist. Org canvases are skipped. Each billed call gets its own
// row, including retries of the same node execution.
func RecordUsage(tx *gorm.DB, in WorkspaceUsageEventInput) error {
	if in.Provider == "" || in.Model == "" || in.NodeExecutionID == uuid.Nil || in.CanvasRunID == uuid.Nil {
		return fmt.Errorf("workspace usage event requires provider, model, node execution, and canvas run")
	}

	scope, err := resolveUsageScope(tx, in.CanvasRunID)
	if err != nil {
		return err
	}
	if scope == nil {
		return nil
	}

	total := in.TotalTokens
	if total == 0 {
		total = in.InputTokens + in.OutputTokens + in.CacheReadTokens + in.CacheWriteTokens + in.ReasoningTokens
	}

	version := pricebook.Version
	providerCostMicros := int64(0)
	if in.CostMicros != nil {
		providerCostMicros = *in.CostMicros
		version = pricebook.Version + "+provider"
	} else {
		if fundingSourceIsHosted(in.FundingSource) && !pricebook.IsPriced(in.Model) {
			log.WithFields(log.Fields{
				"provider": in.Provider,
				"model":    in.Model,
			}).Warn("hosted LLM usage has no price for this model; recording tokens at zero cost")
		}
		providerCostMicros = pricebook.EstimateMicros(
			in.Provider,
			in.Model,
			in.InputTokens,
			in.OutputTokens,
			in.CacheReadTokens,
			in.CacheWriteTokens,
			in.ReasoningTokens,
		)
	}

	fundingSource := in.FundingSource
	if fundingSource == "" {
		fundingSource = UsageFundingSourceBYOK
	}

	billedMicros := providerCostMicros
	if fundingSource == UsageFundingSourceHosted {
		markupBPS, markupErr := ResolveOrganizationMarkupBPS(tx, scope.OrganizationID)
		if markupErr != nil {
			return markupErr
		}
		billedMicros = ApplyMarkupMicros(providerCostMicros, markupBPS)
	}

	now := time.Now()
	event := WorkspaceUsageEvent{
		ID:                   uuid.New(),
		OrganizationID:       scope.OrganizationID,
		FactoryID:            &scope.FactoryID,
		WorkOrderID:          scope.WorkOrderID,
		LineID:               scope.LineID,
		LineDispatchID:       scope.LineDispatchID,
		WorkOrderExecutionID: scope.WorkOrderExecutionID,
		CanvasRunID:          in.CanvasRunID,
		NodeExecutionID:      in.NodeExecutionID,
		NodeID:               in.NodeID,
		Provider:             in.Provider,
		Model:                in.Model,
		UsageKind:            UsageKindModel,
		FundingSource:        fundingSource,
		InputTokens:          in.InputTokens,
		OutputTokens:         in.OutputTokens,
		CacheReadTokens:      in.CacheReadTokens,
		CacheWriteTokens:     in.CacheWriteTokens,
		ReasoningTokens:      in.ReasoningTokens,
		TotalTokens:          total,
		CostMicros:           billedMicros,
		ProviderCostMicros:   providerCostMicros,
		Currency:             "usd",
		PriceBookVersion:     version,
		IdempotencyKey:       usageIdempotencyKey(in.IdempotencyKey),
		OccurredAt:           now,
		CreatedAt:            now,
	}

	return persistUsageEvent(tx, event, scope.execution)
}

// RecordComputeUsage inserts one factory-linked runner-fleet row. Cost comes
// from the compute price book (zero until rates are published) at the raw
// provider rate; hosted markup does not apply. The stored cost_micros still
// draws down org hosted credit and factory hosted budgets, same as model
// usage. Org canvases are skipped.
func RecordComputeUsage(tx *gorm.DB, in ComputeUsageEventInput) error {
	machineType := strings.TrimSpace(in.MachineType)
	if machineType == "" || in.NodeExecutionID == uuid.Nil || in.CanvasRunID == uuid.Nil {
		return fmt.Errorf("compute usage event requires machine type, node execution, and canvas run")
	}
	if in.DurationSeconds < 0 {
		return fmt.Errorf("compute usage event duration cannot be negative")
	}

	scope, err := resolveUsageScope(tx, in.CanvasRunID)
	if err != nil {
		return err
	}
	if scope == nil {
		return nil
	}

	fleetID := strings.TrimSpace(in.FleetID)
	providerCostMicros := pricebook.EstimateComputeMicros(machineType, fleetID, in.DurationSeconds)

	now := time.Now()
	event := WorkspaceUsageEvent{
		ID:                   uuid.New(),
		OrganizationID:       scope.OrganizationID,
		FactoryID:            &scope.FactoryID,
		WorkOrderID:          scope.WorkOrderID,
		LineID:               scope.LineID,
		LineDispatchID:       scope.LineDispatchID,
		WorkOrderExecutionID: scope.WorkOrderExecutionID,
		CanvasRunID:          in.CanvasRunID,
		NodeExecutionID:      in.NodeExecutionID,
		NodeID:               in.NodeID,
		Provider:             UsageProviderRunner,
		Model:                machineType,
		UsageKind:            UsageKindCompute,
		FundingSource:        UsageFundingSourceHosted,
		DurationSeconds:      in.DurationSeconds,
		MachineType:          machineType,
		FleetID:              fleetID,
		CostMicros:           providerCostMicros,
		ProviderCostMicros:   providerCostMicros,
		Currency:             "usd",
		PriceBookVersion:     pricebook.Version,
		IdempotencyKey:       usageIdempotencyKey(in.IdempotencyKey),
		OccurredAt:           now,
		CreatedAt:            now,
	}

	return persistUsageEvent(tx, event, scope.execution)
}

func persistUsageEvent(tx *gorm.DB, event WorkspaceUsageEvent, execution *FactoryWorkOrderExecution) error {
	err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "idempotency_key"}},
		DoNothing: true,
	}).Create(&event).Error
	if err != nil {
		return err
	}

	if execution != nil {
		return execution.RollupUsage(tx)
	}
	return nil
}

func fundingSourceIsHosted(source string) bool {
	return strings.TrimSpace(source) == UsageFundingSourceHosted
}

// ParseUsageFundingSource accepts hosted or byok. Empty input is an error.
func ParseUsageFundingSource(source string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(source))
	if normalized == UsageFundingSourceHosted || normalized == UsageFundingSourceBYOK {
		return normalized, nil
	}
	return "", fmt.Errorf("unsupported usage funding source: %s", source)
}

func usageIdempotencyKey(key string) string {
	if trimmed := strings.TrimSpace(key); trimmed != "" {
		return trimmed
	}
	return "call:" + uuid.New().String()
}

// UsageReportFilter scopes ledger aggregates.
type UsageReportFilter struct {
	OrganizationID uuid.UUID
	FactoryID      *uuid.UUID
	WorkOrderID    *uuid.UUID
	UsageKind      string
	Since          time.Time
	Until          time.Time
	Provider       string
	Model          string
	MachineType    string
	TaskOwnerID    *uuid.UUID
	FundingSource  string
}

// WorkOrderRunUsage is one task-run spend row grouped from the ledger.
type WorkOrderRunUsage struct {
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
	Models               []string
	BYOKModels           []string
	MachineTypes         []string
}

func (r WorkOrderRunUsage) CostCents() int64 {
	return pricebook.MicrosToCents(r.CostMicros)
}

func (r WorkOrderRunUsage) HostedCostCents() int64 {
	return pricebook.MicrosToCents(r.HostedCostMicros)
}

func (r WorkOrderRunUsage) BYOKCostCents() int64 {
	return pricebook.MicrosToCents(r.BYOKCostMicros)
}

// UsageTotals is a token, duration, and cost sum.
type UsageTotals struct {
	TotalTokens     int64
	DurationSeconds int64
	CostMicros      int64
}

// UsageByModel is one model bucket in a spend report.
type UsageByModel struct {
	Provider    string
	Model       string
	TotalTokens int64
	CostMicros  int64
}

// UsageByMachineType is one fleet machine-type bucket in a compute report.
type UsageByMachineType struct {
	MachineType     string
	DurationSeconds int64
	CostMicros      int64
}

// UsageSplit is the ledger of one subject, divided by what the spend paid for.
type UsageSplit struct {
	// Model is spend on model tokens.
	Model UsageTotals
	// Compute is spend on runner machine time.
	Compute UsageTotals
}

func (t UsageTotals) CostCents() int64 {
	return pricebook.MicrosToCents(t.CostMicros)
}

// Add returns the field-wise sum of two ledger totals.
func (t UsageTotals) Add(other UsageTotals) UsageTotals {
	return UsageTotals{
		TotalTokens:     t.TotalTokens + other.TotalTokens,
		DurationSeconds: t.DurationSeconds + other.DurationSeconds,
		CostMicros:      t.CostMicros + other.CostMicros,
	}
}

func (r UsageByModel) CostCents() int64 {
	return pricebook.MicrosToCents(r.CostMicros)
}

func (r UsageByMachineType) CostCents() int64 {
	return pricebook.MicrosToCents(r.CostMicros)
}

// Total is the whole ledger of the subject, both bands together.
func (s UsageSplit) Total() UsageTotals {
	return s.Model.Add(s.Compute)
}

type usageSumRow struct {
	ID              uuid.UUID
	TotalTokens     int64
	DurationSeconds int64
	CostMicros      int64
}

type usageKindSumRow struct {
	ID              uuid.UUID
	UsageKind       string
	TotalTokens     int64
	DurationSeconds int64
	CostMicros      int64
}

func scanUsageSums(rows []usageSumRow) map[uuid.UUID]UsageTotals {
	result := make(map[uuid.UUID]UsageTotals, len(rows))
	for _, row := range rows {
		result[row.ID] = UsageTotals{
			TotalTokens:     row.TotalTokens,
			DurationSeconds: row.DurationSeconds,
			CostMicros:      row.CostMicros,
		}
	}
	return result
}

const usageSumSelect = "COALESCE(SUM(total_tokens), 0) AS total_tokens, COALESCE(SUM(duration_seconds), 0) AS duration_seconds, COALESCE(SUM(cost_micros), 0) AS cost_micros"

// SumUsageForWorkOrders returns ledger totals keyed by work order. Missing
// IDs are absent from the map (zero value).
func SumUsageForWorkOrders(tx *gorm.DB, workOrderIDs []uuid.UUID) (map[uuid.UUID]UsageTotals, error) {
	if len(workOrderIDs) == 0 {
		return map[uuid.UUID]UsageTotals{}, nil
	}

	var rows []usageSumRow
	err := tx.Model(&WorkspaceUsageEvent{}).
		Select("work_order_id AS id, "+usageSumSelect).
		Where("work_order_id IN ?", workOrderIDs).
		Group("work_order_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return scanUsageSums(rows), nil
}

// SumUsageForWorkOrdersByKind returns ledger totals keyed by work order, with
// model spend and compute spend reported apart. Missing IDs are absent from
// the map (zero value).
//
// Anything that is not compute counts as model spend, so the two bands always
// add up to what the work order cost.
func SumUsageForWorkOrdersByKind(tx *gorm.DB, workOrderIDs []uuid.UUID) (map[uuid.UUID]UsageSplit, error) {
	if len(workOrderIDs) == 0 {
		return map[uuid.UUID]UsageSplit{}, nil
	}

	var rows []usageKindSumRow
	err := tx.Model(&WorkspaceUsageEvent{}).
		Select("work_order_id AS id, usage_kind, "+usageSumSelect).
		Where("work_order_id IN ?", workOrderIDs).
		Group("work_order_id, usage_kind").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID]UsageSplit, len(rows))
	for _, row := range rows {
		totals := UsageTotals{
			TotalTokens:     row.TotalTokens,
			DurationSeconds: row.DurationSeconds,
			CostMicros:      row.CostMicros,
		}
		split := result[row.ID]
		if row.UsageKind == UsageKindCompute {
			split.Compute = split.Compute.Add(totals)
		} else {
			split.Model = split.Model.Add(totals)
		}
		result[row.ID] = split
	}
	return result, nil
}

// SumUsageForRunTrees returns ledger totals for each root run, including
// spend recorded on descendant runs.
func SumUsageForRunTrees(tx *gorm.DB, rootIDs []uuid.UUID) (map[uuid.UUID]UsageTotals, error) {
	result := make(map[uuid.UUID]UsageTotals, len(rootIDs))
	if len(rootIDs) == 0 {
		return result, nil
	}

	rootOf := make(map[uuid.UUID]uuid.UUID, len(rootIDs))
	treeIDs := make([]uuid.UUID, 0, len(rootIDs))
	for _, id := range rootIDs {
		rootOf[id] = id
		treeIDs = append(treeIDs, id)
	}

	frontier := append([]uuid.UUID{}, rootIDs...)
	for len(frontier) > 0 {
		var children []CanvasRun
		err := tx.Select("id", "parent_run_id").Where("parent_run_id IN ?", frontier).Find(&children).Error
		if err != nil {
			return nil, err
		}

		frontier = frontier[:0]
		for i := range children {
			child := children[i]
			if child.ParentRunID == nil {
				continue
			}
			parentRoot, ok := rootOf[*child.ParentRunID]
			if !ok {
				continue
			}
			if _, seen := rootOf[child.ID]; seen {
				continue
			}
			rootOf[child.ID] = parentRoot
			treeIDs = append(treeIDs, child.ID)
			frontier = append(frontier, child.ID)
		}
	}

	var rows []usageSumRow
	err := tx.Model(&WorkspaceUsageEvent{}).
		Select("canvas_run_id AS id, "+usageSumSelect).
		Where("canvas_run_id IN ?", treeIDs).
		Group("canvas_run_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		rootID, ok := rootOf[row.ID]
		if !ok {
			continue
		}
		totals := result[rootID]
		totals.TotalTokens += row.TotalTokens
		totals.DurationSeconds += row.DurationSeconds
		totals.CostMicros += row.CostMicros
		result[rootID] = totals
	}
	return result, nil
}

func usageReportQuery(tx *gorm.DB, filter UsageReportFilter) *gorm.DB {
	joinWorkOrders := filter.TaskOwnerID != nil
	return spendingScopedQuery(tx, filter, joinWorkOrders)
}

func spendingScopedQuery(tx *gorm.DB, filter UsageReportFilter, joinWorkOrders bool) *gorm.DB {
	query := tx.Model(&WorkspaceUsageEvent{})
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
		if filter.FundingSource == UsageFundingSourceHosted {
			query = query.Where("workspace_usage_events.funding_source = ?", UsageFundingSourceHosted)
		} else {
			query = query.Where("workspace_usage_events.funding_source IS DISTINCT FROM ?", UsageFundingSourceHosted)
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

func modelUsageReportQuery(tx *gorm.DB, filter UsageReportFilter) *gorm.DB {
	filter.UsageKind = UsageKindModel
	return usageReportQuery(tx, filter)
}

func computeUsageReportQuery(tx *gorm.DB, filter UsageReportFilter) *gorm.DB {
	filter.UsageKind = UsageKindCompute
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
	COALESCE(SUM(CASE WHEN workspace_usage_events.funding_source = '` + UsageFundingSourceHosted + `' AND workspace_usage_events.usage_kind = '` + UsageKindModel + `' THEN workspace_usage_events.cost_micros ELSE 0 END), 0) AS hosted_cost_micros,
	COALESCE(SUM(CASE WHEN workspace_usage_events.funding_source = '` + UsageFundingSourceBYOK + `' THEN workspace_usage_events.cost_micros ELSE 0 END), 0) AS byok_cost_micros,
	COALESCE(STRING_AGG(DISTINCT CASE WHEN workspace_usage_events.usage_kind = '` + UsageKindModel + `' AND workspace_usage_events.funding_source IS DISTINCT FROM '` + UsageFundingSourceBYOK + `' THEN workspace_usage_events.provider || '/' || workspace_usage_events.model END, E'\n'), '') AS models,
	COALESCE(STRING_AGG(DISTINCT CASE WHEN workspace_usage_events.usage_kind = '` + UsageKindModel + `' AND workspace_usage_events.funding_source = '` + UsageFundingSourceBYOK + `' THEN workspace_usage_events.provider || '/' || workspace_usage_events.model END, E'\n'), '') AS byok_models,
	COALESCE(STRING_AGG(DISTINCT CASE WHEN workspace_usage_events.usage_kind = '` + UsageKindCompute + `' AND workspace_usage_events.machine_type <> '' THEN workspace_usage_events.machine_type END, E'\n'), '') AS machine_types`

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
			Models:               splitUsageAgg(row.Models),
			BYOKModels:           splitUsageAgg(row.BYOKModels),
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

// RollupUsage copies ledger totals into the cached execution columns.
// It locks the step row so concurrent RecordUsage calls cannot write a
// stale sum over a newer one.
func (e *FactoryWorkOrderExecution) RollupUsage(tx *gorm.DB) error {
	return tx.Transaction(func(inner *gorm.DB) error {
		var locked FactoryWorkOrderExecution
		err := inner.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", e.ID).
			First(&locked).Error
		if err != nil {
			return err
		}

		var totals UsageTotals
		err = inner.Model(&WorkspaceUsageEvent{}).
			Select(usageSumSelect).
			Where("work_order_execution_id = ?", e.ID).
			Scan(&totals).Error
		if err != nil {
			return err
		}

		now := time.Now()
		e.TotalTokens = totals.TotalTokens
		e.DurationSeconds = totals.DurationSeconds
		e.CostCents = totals.CostCents()
		e.UpdatedAt = now

		return inner.Model(e).Updates(map[string]any{
			"total_tokens":     e.TotalTokens,
			"duration_seconds": e.DurationSeconds,
			"cost_cents":       e.CostCents,
			"updated_at":       now,
		}).Error
	})
}

func attachUsageEventsToWorkOrder(tx *gorm.DB, factoryID, workOrderID, sourceRunID uuid.UUID) error {
	runIDs, err := canvasRunIDsInTree(tx, sourceRunID)
	if err != nil || len(runIDs) == 0 {
		return err
	}
	return tx.Model(&WorkspaceUsageEvent{}).
		Where("factory_id = ? AND canvas_run_id IN ? AND work_order_id IS NULL AND work_order_execution_id IS NULL", factoryID, runIDs).
		Update("work_order_id", workOrderID).Error
}

func canvasRunIDsInTree(tx *gorm.DB, rootID uuid.UUID) ([]uuid.UUID, error) {
	ids := []uuid.UUID{rootID}
	seen := map[uuid.UUID]struct{}{rootID: {}}
	frontier := []uuid.UUID{rootID}
	for len(frontier) > 0 {
		var children []CanvasRun
		err := tx.Select("id").Where("parent_run_id IN ?", frontier).Find(&children).Error
		if err != nil {
			return nil, err
		}
		frontier = frontier[:0]
		for _, child := range children {
			if _, exists := seen[child.ID]; exists {
				continue
			}
			seen[child.ID] = struct{}{}
			ids = append(ids, child.ID)
			frontier = append(frontier, child.ID)
		}
	}
	return ids, nil
}
