package route53

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	pollChangeActionName = "pollChange"
	pollInterval         = 5 * time.Second
)

// pollChangeUntilSynced runs when the pollChange action is invoked. It fetches the
// change status; if INSYNC it emits the result and finishes, otherwise schedules another poll.
func pollChangeUntilSynced(ctx core.ActionContext) error {
	var meta RecordChangePollMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &meta); err != nil {
		return fmt.Errorf("failed to decode poll metadata: %w", err)
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds)
	change, err := client.GetChange(meta.ChangeID)
	if err != nil {
		return fmt.Errorf("failed to get change status: %w", err)
	}

	submittedAt := change.SubmittedAt
	if submittedAt == "" {
		submittedAt = meta.SubmittedAt
	}

	output := map[string]any{
		"change": map[string]any{
			"id":          change.ID,
			"status":      change.Status,
			"submittedAt": submittedAt,
		},
		"record": map[string]any{
			"name": meta.RecordName,
			"type": meta.RecordType,
		},
	}

	if change.Status == "INSYNC" {
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"aws.route53.change",
			[]any{output},
		)
	}

	// Still PENDING; keep polling with same metadata
	return ctx.Requests.ScheduleActionCall(
		pollChangeActionName,
		map[string]any{},
		pollInterval,
	)
}
