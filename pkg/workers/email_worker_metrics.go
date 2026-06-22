package workers

import (
	"context"
	"time"

	"github.com/superplanehq/superplane/pkg/telemetry"
)

const (
	emailTypeInvitation = "invitation"
	emailTypeMagicCode  = "magic_code"

	emailWorkerReasonInvalidMessage       = "invalid_message"
	emailWorkerReasonInvitationNotFound   = "invitation_not_found"
	emailWorkerReasonOrganizationNotFound = "organization_not_found"
	emailWorkerReasonInvitationNotPending = "invitation_not_pending"
	emailWorkerReasonInviterNotFound      = "inviter_not_found"
	emailWorkerReasonSendError            = "send_error"
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
