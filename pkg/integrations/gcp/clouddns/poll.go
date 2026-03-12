package clouddns

import (
	"context"
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	pollChangeActionName = "pollChange"
	pollInterval         = 5 * time.Second
)

// pollChangeUntilDone polls for a Cloud DNS change status.
// When "done" it emits the result and finishes, when "pending" it schedules
// another poll, and any other status fails the execution.
func pollChangeUntilDone(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var meta RecordSetPollMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &meta); err != nil {
		return fmt.Errorf("failed to decode poll metadata: %w", err)
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create GCP client: %w", err)
	}

	change, err := getChange(context.Background(), client, client.ProjectID(), meta.ManagedZone, meta.ChangeID)
	if err != nil {
		return fmt.Errorf("failed to get change status: %w", err)
	}

	startTime := change.StartTime
	if startTime == "" {
		startTime = meta.StartTime
	}

	output := map[string]any{
		"change": map[string]any{
			"id":        change.ID,
			"status":    change.Status,
			"startTime": startTime,
		},
		"record": map[string]any{
			"name": meta.RecordName,
			"type": meta.RecordType,
		},
	}

	if change.Status == "done" {
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"gcp.clouddns.change",
			[]any{output},
		)
	}

	if change.Status != "pending" {
		return ctx.ExecutionState.Fail(
			"error",
			fmt.Sprintf("unexpected Cloud DNS change status %q for change %q", change.Status, change.ID),
		)
	}

	return ctx.Requests.ScheduleActionCall(pollChangeActionName, map[string]any{}, pollInterval)
}
