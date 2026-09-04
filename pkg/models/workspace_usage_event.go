package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
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

var ErrHostedUsageUnpriced = errors.New("hosted LLM usage has no price for this model")

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
			return fmt.Errorf("%w: %s %s", ErrHostedUsageUnpriced, in.Provider, in.Model)
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
// from the compute price book (zero until rates are published). Hosted
// markup and wallet debit do not apply. Org canvases are skipped.
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

type usageSumRow struct {
	ID              uuid.UUID
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
