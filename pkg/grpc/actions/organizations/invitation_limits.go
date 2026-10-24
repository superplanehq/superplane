package organizations

import (
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	defaultInvitationOrgHourlyLimit            = 50
	defaultInvitationRecipientDailyLimit       = 5
	defaultInvitationSharedRecipientDailyLimit = 10
	defaultPendingInvitationLimit              = 100

	invitationOrgWindow       = time.Hour
	invitationRecipientWindow = 24 * time.Hour

	invitationOrgHourlyLimitEnv            = "INVITATION_ORG_HOURLY_LIMIT"
	invitationRecipientDailyLimitEnv       = "INVITATION_RECIPIENT_DAILY_LIMIT"
	invitationSharedRecipientDailyLimitEnv = "INVITATION_SHARED_RECIPIENT_DAILY_LIMIT"
	pendingInvitationLimitEnv              = "PENDING_INVITATION_LIMIT"
)

// invitationLimits holds the effective invitation limits of this installation.
type invitationLimits struct {
	orgHourly            int64
	recipientDaily       int64
	sharedRecipientDaily int64
	pending              int64
}

// loadInvitationLimits reads the optional environment overrides. A value that
// is not a positive integer is ignored, so a typo cannot turn a limit off.
func loadInvitationLimits() invitationLimits {
	return invitationLimits{
		orgHourly:            positiveIntFromEnv(invitationOrgHourlyLimitEnv, defaultInvitationOrgHourlyLimit),
		recipientDaily:       positiveIntFromEnv(invitationRecipientDailyLimitEnv, defaultInvitationRecipientDailyLimit),
		sharedRecipientDaily: positiveIntFromEnv(invitationSharedRecipientDailyLimitEnv, defaultInvitationSharedRecipientDailyLimit),
		pending:              positiveIntFromEnv(pendingInvitationLimitEnv, defaultPendingInvitationLimit),
	}
}

func positiveIntFromEnv(name string, fallback int64) int64 {
	raw, ok := os.LookupEnv(name)
	if !ok || raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		log.WithField("variable", name).WithField("value", raw).Warn("Ignoring invitation limit that is not a positive integer")
		return fallback
	}
	return value
}
