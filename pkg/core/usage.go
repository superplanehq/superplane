package core

import (
	log "github.com/sirupsen/logrus"
)

// UsageRecorder stores spend on the workspace usage ledger.
// A nil recorder on ExecutionContext is a no-op (tests and non-factory runs).
type UsageRecorder interface {
	Record(record UsageRecord) error
	RecordCompute(record ComputeUsageRecord) error
}

// ComputeUsageRecord is the call-site payload for one runner-fleet task.
type ComputeUsageRecord struct {
	MachineType     string
	FleetID         string
	DurationSeconds int64
	IdempotencyKey  string
}

// UsageRecord is the call-site payload for one provider response.
// Do not include prompt or completion text.
type UsageRecord struct {
	Provider         string
	Model            string
	InputTokens      int64
	OutputTokens     int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	ReasoningTokens  int64
	TotalTokens      int64
	// CostMicros is provider-reported cost in millionths of a US dollar.
	// Leave nil to price the call from the SuperPlane price book.
	CostMicros *int64
	// FundingSource is "hosted" or "byok". Empty defaults to byok.
	FundingSource string
	// IdempotencyKey is a stable ledger key. Empty generates a unique key
	// per call. Runner finish records use a prefix so webhook and poll
	// cannot insert two rows for one node execution.
	IdempotencyKey string
}

// RecordUsage writes the record when a recorder is wired. Tests that omit
// Usage on ExecutionContext skip tracking.
func (ctx ExecutionContext) RecordUsage(record UsageRecord) error {
	if ctx.Usage == nil {
		return nil
	}
	return ctx.Usage.Record(record)
}

// RecordUsageBestEffort records spend without failing the provider call.
// A ledger error must not drop the model output the caller already paid for.
func (ctx ExecutionContext) RecordUsageBestEffort(record UsageRecord) {
	recordUsageBestEffort(ctx.Usage, ctx.Logger, record)
}

// RecordUsage writes the record when a recorder is wired on a hook.
func (ctx ActionHookContext) RecordUsage(record UsageRecord) error {
	if ctx.Usage == nil {
		return nil
	}
	return ctx.Usage.Record(record)
}

// RecordUsageBestEffort records spend without failing the hook.
func (ctx ActionHookContext) RecordUsageBestEffort(record UsageRecord) {
	recordUsageBestEffort(ctx.Usage, ctx.Logger, record)
}

func recordUsageBestEffort(usage UsageRecorder, logger *log.Entry, record UsageRecord) {
	if usage == nil {
		return
	}
	if err := usage.Record(record); err != nil && logger != nil {
		logger.WithError(err).Error("failed to record LLM usage")
	}
}

// RecordCompute writes runner-fleet seconds when a recorder is wired.
func (ctx ExecutionContext) RecordCompute(record ComputeUsageRecord) error {
	if ctx.Usage == nil {
		return nil
	}
	return ctx.Usage.RecordCompute(record)
}

func recordComputeBestEffort(usage UsageRecorder, logger *log.Entry, record ComputeUsageRecord) {
	if usage == nil {
		return
	}
	if err := usage.RecordCompute(record); err != nil && logger != nil {
		logger.WithError(err).Error("failed to record compute usage")
	}
}
