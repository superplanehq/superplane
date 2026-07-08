package claude

import (
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// Batches typically finish within an hour but can take up to 24 hours. Poll
// with exponential backoff, capped well above the runAgent/runCodeAgent
// intervals, and allow enough attempts to comfortably cover a full day.
const (
	batchInitialPoll     = 30 * time.Second
	batchMaxPollInterval = 10 * time.Minute
	batchMaxPollAttempts = 200
	batchMaxPollErrors   = 5
)

func (c *CreateBatchMessage) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name == "poll" {
		return c.poll(ctx)
	}
	return fmt.Errorf("unknown hook: %s", ctx.Name)
}

func (c *CreateBatchMessage) poll(ctx core.ActionHookContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata := BatchExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}
	if metadata.BatchID == "" {
		return nil
	}

	attempt := 1
	errs := 0
	if a, ok := ctx.Parameters["attempt"].(float64); ok {
		attempt = int(a)
	}
	if e, ok := ctx.Parameters["errors"].(float64); ok {
		errs = int(e)
	}

	if attempt > batchMaxPollAttempts {
		ctx.Logger.Errorf("Message batch %s exceeded max poll attempts", metadata.BatchID)
		out := buildBatchOutput("timeout", &MessageBatch{ID: metadata.BatchID}, nil, false)
		return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, CreateBatchMessagePayloadType, []any{out})
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return c.scheduleNextPoll(ctx, attempt+1, errs)
	}

	batch, err := client.GetMessageBatch(metadata.BatchID)
	if err != nil {
		errs++
		if errs >= batchMaxPollErrors {
			ctx.Logger.Errorf("Message batch %s: polling failed repeatedly: %v", metadata.BatchID, err)
			out := buildBatchOutput("error", &MessageBatch{ID: metadata.BatchID}, nil, false)
			return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, CreateBatchMessagePayloadType, []any{out})
		}
		return c.scheduleNextPoll(ctx, attempt+1, errs)
	}

	if batch == nil || batch.ProcessingStatus != batchStatusEnded {
		if batch != nil {
			_ = ctx.Metadata.Set(BatchExecutionMetadata{
				BatchID:       metadata.BatchID,
				Status:        batch.ProcessingStatus,
				RequestCounts: &batch.RequestCounts,
			})
		}
		return c.scheduleNextPoll(ctx, attempt+1, 0)
	}

	spec, err := decodeBatchMessageSpec(ctx.Configuration)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	hasSchema := strings.TrimSpace(spec.OutputSchema) != ""

	results, err := client.GetMessageBatchResults(batch.ID)
	if err != nil {
		ctx.Logger.Warnf("Failed to fetch results for batch %s: %v. Retrying poll.", batch.ID, err)
		return c.scheduleNextPoll(ctx, attempt+1, errs+1)
	}

	out := buildBatchOutput(batchStatusEnded, batch, results, hasSchema)
	if err := ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, CreateBatchMessagePayloadType, []any{out}); err != nil {
		return err
	}

	_ = ctx.Metadata.Set(BatchExecutionMetadata{BatchID: batch.ID, Status: batch.ProcessingStatus, RequestCounts: &batch.RequestCounts})
	return nil
}

func (c *CreateBatchMessage) scheduleNextPoll(ctx core.ActionHookContext, nextAttempt, errors int) error {
	interval := batchInitialPoll * time.Duration(1<<uint(min(nextAttempt-1, 8)))
	if interval > batchMaxPollInterval {
		interval = batchMaxPollInterval
	}
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{
		"attempt": nextAttempt,
		"errors":  errors,
	}, interval)
}
