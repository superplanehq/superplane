package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	UsageKindModel = "model"

	UsageFundingSourceBYOK   = "byok"
	UsageFundingSourceHosted = "hosted"

	UsageProviderAnthropic  = "anthropic"
	UsageProviderOpenAI     = "openai"
	UsageProviderOpenRouter = "openrouter"
	UsageProviderPerplexity = "perplexity"
)

// LLMUsageEvent is one append-only LLM spend row. It is the source of truth
// for reports. Factory execution token/cost columns are cached rollups.
type LLMUsageEvent struct {
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
	CostMicros           int64
	ProviderCostMicros   int64
	Currency             string
	PriceBookVersion     string
	IdempotencyKey       string
	OccurredAt           time.Time
	CreatedAt            time.Time
}

func (LLMUsageEvent) TableName() string {
	return "llm_usage_events"
}

// LLMUsageEventInput is the call-site payload before factory scope is resolved.
type LLMUsageEventInput struct {
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
}

// RecordUsage inserts one factory-linked usage row and copies ledger totals
// into the step cache so in-progress work orders show spend. Non-factory
// runs are skipped. Each billed call gets its own row, including retries
// of the same node execution.
func RecordUsage(tx *gorm.DB, in LLMUsageEventInput) error {
	if in.Provider == "" || in.Model == "" || in.NodeExecutionID == uuid.Nil || in.CanvasRunID == uuid.Nil {
		return fmt.Errorf("llm usage event requires provider, model, node execution, and canvas run")
	}

	execution, err := FindWorkOrderExecutionForRun(tx, in.CanvasRunID)
	if err != nil {
		if errors.Is(err, ErrFactoryWorkOrderExecutionNotFound) {
			return nil
		}
		return err
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
		markupBPS, markupErr := ResolveOrganizationMarkupBPS(tx, execution.OrganizationID)
		if markupErr != nil {
			return markupErr
		}
		billedMicros = ApplyMarkupMicros(providerCostMicros, markupBPS)
	}

	now := time.Now()
	event := LLMUsageEvent{
		ID:                   uuid.New(),
		OrganizationID:       execution.OrganizationID,
		FactoryID:            &execution.FactoryID,
		WorkOrderID:          &execution.WorkOrderID,
		LineID:               &execution.LineID,
		LineDispatchID:       &execution.LineDispatchID,
		WorkOrderExecutionID: &execution.ID,
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
		IdempotencyKey:       "call:" + uuid.New().String(),
		OccurredAt:           now,
		CreatedAt:            now,
	}

	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "idempotency_key"}},
		DoNothing: true,
	}).Create(&event).Error
	if err != nil {
		return err
	}

	return execution.RollupUsage(tx)
}

// UsageReportFilter scopes ledger aggregates.
type UsageReportFilter struct {
	OrganizationID uuid.UUID
	FactoryID      *uuid.UUID
	WorkOrderID    *uuid.UUID
	Since          time.Time
	Until          time.Time
}

// UsageTotals is a token and cost sum.
type UsageTotals struct {
	TotalTokens int64
	CostMicros  int64
}

// UsageByModel is one model bucket in a spend report.
type UsageByModel struct {
	Provider    string
	Model       string
	TotalTokens int64
	CostMicros  int64
}

func (t UsageTotals) CostCents() int64 {
	return pricebook.MicrosToCents(t.CostMicros)
}

func (r UsageByModel) CostCents() int64 {
	return pricebook.MicrosToCents(r.CostMicros)
}

func usageReportQuery(tx *gorm.DB, filter UsageReportFilter) *gorm.DB {
	query := tx.Model(&LLMUsageEvent{}).Where("organization_id = ?", filter.OrganizationID)
	if filter.FactoryID != nil {
		query = query.Where("factory_id = ?", *filter.FactoryID)
	}
	if filter.WorkOrderID != nil {
		query = query.Where("work_order_id = ?", *filter.WorkOrderID)
	}
	if !filter.Since.IsZero() {
		query = query.Where("occurred_at >= ?", filter.Since)
	}
	if !filter.Until.IsZero() {
		query = query.Where("occurred_at < ?", filter.Until)
	}
	return query
}

// SummarizeUsage returns org or workspace totals and a per-model breakdown.
func SummarizeUsage(tx *gorm.DB, filter UsageReportFilter) (UsageTotals, []UsageByModel, error) {
	var totals UsageTotals
	err := usageReportQuery(tx, filter).
		Select("COALESCE(SUM(total_tokens), 0) AS total_tokens, COALESCE(SUM(cost_micros), 0) AS cost_micros").
		Scan(&totals).Error
	if err != nil {
		return UsageTotals{}, nil, err
	}

	var byModel []UsageByModel
	err = usageReportQuery(tx, filter).
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
		err = inner.Model(&LLMUsageEvent{}).
			Select("COALESCE(SUM(total_tokens), 0) AS total_tokens, COALESCE(SUM(cost_micros), 0) AS cost_micros").
			Where("work_order_execution_id = ?", e.ID).
			Scan(&totals).Error
		if err != nil {
			return err
		}

		now := time.Now()
		e.TotalTokens = totals.TotalTokens
		e.CostCents = totals.CostCents()
		e.UpdatedAt = now

		return inner.Model(e).Updates(map[string]any{
			"total_tokens": e.TotalTokens,
			"cost_cents":   e.CostCents,
			"updated_at":   now,
		}).Error
	})
}
