package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/llm/modelid"
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
	UsageProviderCustom     = "custom"
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
	in.Model = modelid.CatalogModelID(in.Model)
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

	fundingSource := in.FundingSource
	if fundingSource == "" {
		fundingSource = UsageFundingSourceBYOK
	}
	price, err := billing.PriceUsage(tx, scope.OrganizationID, billing.Usage{
		Provider: in.Provider, Model: in.Model, FundingSource: fundingSource,
		InputTokens: in.InputTokens, OutputTokens: in.OutputTokens,
		CacheReadTokens: in.CacheReadTokens, CacheWriteTokens: in.CacheWriteTokens,
		ReasoningTokens: in.ReasoningTokens, CostMicros: in.CostMicros,
	})
	if err != nil {
		return err
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
		CostMicros:           price.CostMicros,
		ProviderCostMicros:   price.ProviderCostMicros,
		Currency:             "usd",
		PriceBookVersion:     price.PriceBookVersion,
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
	price, err := billing.PriceCompute(billing.ComputeUsage{MachineType: machineType, FleetID: fleetID, DurationSeconds: in.DurationSeconds})
	if err != nil {
		return err
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
		Provider:             UsageProviderRunner,
		Model:                machineType,
		UsageKind:            UsageKindCompute,
		FundingSource:        UsageFundingSourceHosted,
		DurationSeconds:      in.DurationSeconds,
		MachineType:          machineType,
		FleetID:              fleetID,
		CostMicros:           price.CostMicros,
		ProviderCostMicros:   price.ProviderCostMicros,
		Currency:             "usd",
		PriceBookVersion:     price.PriceBookVersion,
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

type UsageTotals = billing.UsageTotals
type UsageByModel = billing.UsageByModel
type UsageByMachineType = billing.UsageByMachineType
type UsageSplit = billing.UsageSplit

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

// WorkOrderHasUsageEvents reports whether the ledger has any row for the task.
// A task with no rows is not the same as a recorded total of zero.
func WorkOrderHasUsageEvents(tx *gorm.DB, workOrderID uuid.UUID) (bool, error) {
	if workOrderID == uuid.Nil {
		return false, nil
	}
	var event WorkspaceUsageEvent
	err := tx.Model(&WorkspaceUsageEvent{}).
		Select("id").
		Where("work_order_id = ?", workOrderID).
		Take(&event).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// SumUsageForCanvasRuns returns ledger totals for each run id.
// Descendant runs are not included. Missing ids are absent from the map.
func SumUsageForCanvasRuns(tx *gorm.DB, runIDs []uuid.UUID) (map[uuid.UUID]UsageTotals, error) {
	if len(runIDs) == 0 {
		return map[uuid.UUID]UsageTotals{}, nil
	}

	var rows []usageSumRow
	err := tx.Model(&WorkspaceUsageEvent{}).
		Select("canvas_run_id AS id, "+usageSumSelect).
		Where("canvas_run_id IN ?", runIDs).
		Group("canvas_run_id").
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

// ListModelsForWorkOrderExecutions returns distinct ledger models keyed by
// step execution. Missing IDs are absent from the map.
func ListModelsForWorkOrderExecutions(tx *gorm.DB, executionIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	if len(executionIDs) == 0 {
		return map[uuid.UUID][]string{}, nil
	}

	var rows []usageModelNameRow
	err := tx.Model(&WorkspaceUsageEvent{}).
		Select("work_order_execution_id AS id, provider, model").
		Where("work_order_execution_id IN ?", executionIDs).
		Where("usage_kind = ?", UsageKindModel).
		Where("model <> ''").
		Group("work_order_execution_id, provider, model").
		Order("work_order_execution_id ASC").
		Order("provider ASC").
		Order("model ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return scanUsageModelNames(rows), nil
}

// ListUsageByModelForWorkOrders returns model spend rows keyed by work order.
// Missing IDs are absent from the map.
func ListUsageByModelForWorkOrders(tx *gorm.DB, workOrderIDs []uuid.UUID) (map[uuid.UUID][]UsageByModel, error) {
	if len(workOrderIDs) == 0 {
		return map[uuid.UUID][]UsageByModel{}, nil
	}

	var rows []usageByModelRow
	err := tx.Model(&WorkspaceUsageEvent{}).
		Select("work_order_id AS id, provider, model, COALESCE(SUM(total_tokens), 0) AS total_tokens, COALESCE(SUM(cost_micros), 0) AS cost_micros").
		Where("work_order_id IN ?", workOrderIDs).
		Where("usage_kind = ?", UsageKindModel).
		Group("work_order_id, provider, model").
		Order("work_order_id ASC").
		Order("cost_micros DESC").
		Order("total_tokens DESC").
		Order("provider ASC").
		Order("model ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID][]UsageByModel, len(rows))
	for _, row := range rows {
		result[row.ID] = append(result[row.ID], UsageByModel{
			Provider:    row.Provider,
			Model:       row.Model,
			TotalTokens: row.TotalTokens,
			CostMicros:  row.CostMicros,
		})
	}
	return result, nil
}

// ListUsageByMachineTypeForWorkOrders returns compute spend rows keyed by work
// order. Missing IDs are absent from the map.
func ListUsageByMachineTypeForWorkOrders(tx *gorm.DB, workOrderIDs []uuid.UUID) (map[uuid.UUID][]UsageByMachineType, error) {
	if len(workOrderIDs) == 0 {
		return map[uuid.UUID][]UsageByMachineType{}, nil
	}

	var rows []usageByMachineTypeRow
	err := tx.Model(&WorkspaceUsageEvent{}).
		Select("work_order_id AS id, machine_type, COALESCE(SUM(duration_seconds), 0) AS duration_seconds, COALESCE(SUM(cost_micros), 0) AS cost_micros").
		Where("work_order_id IN ?", workOrderIDs).
		Where("usage_kind = ?", UsageKindCompute).
		Group("work_order_id, machine_type").
		Order("work_order_id ASC").
		Order("cost_micros DESC").
		Order("duration_seconds DESC").
		Order("machine_type ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID][]UsageByMachineType, len(rows))
	for _, row := range rows {
		result[row.ID] = append(result[row.ID], UsageByMachineType{
			MachineType:     row.MachineType,
			DurationSeconds: row.DurationSeconds,
			CostMicros:      row.CostMicros,
		})
	}
	return result, nil
}

// SumUsageAndModelsForRunTrees returns ledger totals and distinct models for
// each root run, including spend recorded on descendant runs. One tree walk
// serves both aggregations.
func SumUsageAndModelsForRunTrees(tx *gorm.DB, rootIDs []uuid.UUID) (map[uuid.UUID]UsageTotals, map[uuid.UUID][]string, error) {
	totals := make(map[uuid.UUID]UsageTotals, len(rootIDs))
	models := make(map[uuid.UUID][]string, len(rootIDs))
	if len(rootIDs) == 0 {
		return totals, models, nil
	}

	rootOf, treeIDs, err := collectRunTreeIDs(tx, rootIDs)
	if err != nil {
		return nil, nil, err
	}

	totals, err = sumUsageForMappedRuns(tx, rootOf, treeIDs)
	if err != nil {
		return nil, nil, err
	}
	models, err = listModelsForMappedRuns(tx, rootOf, treeIDs)
	if err != nil {
		return nil, nil, err
	}
	return totals, models, nil
}

// ListModelsForRunTrees returns distinct ledger models for each root run,
// including spend recorded on descendant runs.
func ListModelsForRunTrees(tx *gorm.DB, rootIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	if len(rootIDs) == 0 {
		return map[uuid.UUID][]string{}, nil
	}

	rootOf, treeIDs, err := collectRunTreeIDs(tx, rootIDs)
	if err != nil {
		return nil, err
	}
	return listModelsForMappedRuns(tx, rootOf, treeIDs)
}

// SumUsageForRunTrees returns ledger totals for each root run, including
// spend recorded on descendant runs.
func SumUsageForRunTrees(tx *gorm.DB, rootIDs []uuid.UUID) (map[uuid.UUID]UsageTotals, error) {
	if len(rootIDs) == 0 {
		return map[uuid.UUID]UsageTotals{}, nil
	}

	rootOf, treeIDs, err := collectRunTreeIDs(tx, rootIDs)
	if err != nil {
		return nil, err
	}
	return sumUsageForMappedRuns(tx, rootOf, treeIDs)
}

type usageModelNameRow struct {
	ID       uuid.UUID
	Provider string
	Model    string
}

type usageByModelRow struct {
	ID          uuid.UUID
	Provider    string
	Model       string
	TotalTokens int64
	CostMicros  int64
}

type usageByMachineTypeRow struct {
	ID              uuid.UUID
	MachineType     string
	DurationSeconds int64
	CostMicros      int64
}

func formatUsageModelName(provider, model string) string {
	model = modelid.CatalogModelID(model)
	if model == "" {
		return ""
	}
	provider = strings.TrimSpace(provider)
	if provider == "" || strings.Contains(model, "/") {
		return model
	}
	return provider + "/" + model
}

func scanUsageModelNames(rows []usageModelNameRow) map[uuid.UUID][]string {
	result := make(map[uuid.UUID][]string)
	seen := make(map[uuid.UUID]map[string]struct{})
	for _, row := range rows {
		name := formatUsageModelName(row.Provider, row.Model)
		if name == "" {
			continue
		}
		if _, ok := seen[row.ID]; !ok {
			seen[row.ID] = map[string]struct{}{}
		}
		if _, dup := seen[row.ID][name]; dup {
			continue
		}
		seen[row.ID][name] = struct{}{}
		result[row.ID] = append(result[row.ID], name)
	}
	return result
}

func collectRunTreeIDs(tx *gorm.DB, rootIDs []uuid.UUID) (map[uuid.UUID]uuid.UUID, []uuid.UUID, error) {
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
			return nil, nil, err
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
	return rootOf, treeIDs, nil
}

func sumUsageForMappedRuns(tx *gorm.DB, rootOf map[uuid.UUID]uuid.UUID, treeIDs []uuid.UUID) (map[uuid.UUID]UsageTotals, error) {
	result := make(map[uuid.UUID]UsageTotals, len(rootOf))
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

func listModelsForMappedRuns(tx *gorm.DB, rootOf map[uuid.UUID]uuid.UUID, treeIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	result := make(map[uuid.UUID][]string, len(rootOf))
	var rows []usageModelNameRow
	err := tx.Model(&WorkspaceUsageEvent{}).
		Select("canvas_run_id AS id, provider, model").
		Where("canvas_run_id IN ?", treeIDs).
		Where("usage_kind = ?", UsageKindModel).
		Where("model <> ''").
		Group("canvas_run_id, provider, model").
		Order("provider ASC").
		Order("model ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	seen := make(map[uuid.UUID]map[string]struct{})
	for _, row := range rows {
		rootID, ok := rootOf[row.ID]
		if !ok {
			continue
		}
		name := formatUsageModelName(row.Provider, row.Model)
		if name == "" {
			continue
		}
		if _, ok := seen[rootID]; !ok {
			seen[rootID] = map[string]struct{}{}
		}
		if _, dup := seen[rootID][name]; dup {
			continue
		}
		seen[rootID][name] = struct{}{}
		result[rootID] = append(result[rootID], name)
	}
	return result, nil
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
