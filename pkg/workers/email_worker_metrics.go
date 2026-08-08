package workers

import (
	"context"
	"time"

	"github.com/superplanehq/superplane/pkg/telemetry"
)

const (
	emailTypeMagicCode                = "magic_code"
	emailTypeOrganizationMemberJoined = "organization_member_joined"

	emailWorkerReasonInvalidMessage = "invalid_message"
	emailWorkerReasonSendError      = "send_error"
)

func recordEmailWorkerProcessing(start time.Time, emailType, outcome, reason string) {
	telemetry.RecordEmailWorkerEmailProcessing(
		context.Background(),
		time.Since(start),
		emailType,
		outcome,
		reason,
	)
}
