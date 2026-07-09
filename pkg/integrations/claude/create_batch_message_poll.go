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
		ctx.Logger.Errorf("Message batch poll invoked with undecodable execution metadata: %v", err)
		out := buildBatchOutput("error", nil, nil, false, 0)
		return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, CreateBatchMessagePayloadType, []any{out})
	}
	if metadata.BatchID == "" {
		ctx.Logger.Errorf("Message batch poll invoked without a batch id in execution metadata")
		out := buildBatchOutput("error", nil, nil, false, 0)
		return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, CreateBatchMessagePayloadType, []any{out})
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
		out := buildBatchOutput("timeout", stubBatch(metadata), nil, false, metadata.ItemCount)
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
			out := buildBatchOutput("error", stubBatch(metadata), nil, false, metadata.ItemCount)
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
				ItemCount:     metadata.ItemCount,
			})
		}
		return c.scheduleNextPoll(ctx, attempt+1, 0)
	}

	// The batch itself has ended even if fetching its results below fails;
	// refresh metadata with its terminal status/counts now so the UI doesn't
	// keep showing stale in-progress state while results are retried.
	_ = ctx.Metadata.Set(BatchExecutionMetadata{
		BatchID:       batch.ID,
		Status:        batch.ProcessingStatus,
		RequestCounts: &batch.RequestCounts,
		ItemCount:     metadata.ItemCount,
	})

	spec, err := decodeBatchMessageSpec(ctx.Configuration)
	if err != nil {
		ctx.Logger.Errorf("Message batch %s ended but failed to decode configuration: %v", batch.ID, err)
		out := buildBatchOutput("error", batch, nil, false, metadata.ItemCount)
		return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, CreateBatchMessagePayloadType, []any{out})
	}
	hasSchema := strings.TrimSpace(spec.OutputSchema) != ""

	results, err := client.GetMessageBatchResults(batch.ID)
	if err != nil {
		errs++
		if errs >= batchMaxPollErrors {
			ctx.Logger.Errorf("Message batch %s ended but fetching results failed repeatedly: %v", batch.ID, err)
			out := buildBatchOutput("error", batch, nil, false, metadata.ItemCount)
			return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, CreateBatchMessagePayloadType, []any{out})
		}
		ctx.Logger.Warnf("Failed to fetch results for batch %s: %v. Retrying poll.", batch.ID, err)
		return c.scheduleNextPoll(ctx, attempt+1, errs)
	}

	out := buildBatchOutput(batchStatusEnded, batch, results, hasSchema, metadata.ItemCount)
	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, CreateBatchMessagePayloadType, []any{out})
}

// stubBatch builds a placeholder MessageBatch carrying the last request counts
// seen in execution metadata, so a terminal timeout/error output doesn't zero
// out progress that was already reported to the user on an earlier poll.
func stubBatch(metadata BatchExecutionMetadata) *MessageBatch {
	batch := &MessageBatch{ID: metadata.BatchID}
	if metadata.RequestCounts != nil {
		batch.RequestCounts = *metadata.RequestCounts
	}
	return batch
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
