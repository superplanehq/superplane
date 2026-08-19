package core

// UsageRecorder stores one LLM call on the spend ledger.
// A nil recorder on ExecutionContext is a no-op (tests and non-factory runs).
type UsageRecorder interface {
	Record(record UsageRecord) error
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
	if err := ctx.RecordUsage(record); err != nil && ctx.Logger != nil {
		ctx.Logger.WithError(err).Error("failed to record LLM usage")
	}
}
