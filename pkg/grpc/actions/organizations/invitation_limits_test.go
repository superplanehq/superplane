package organizations

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/services"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

const rateLimitedReason = "INVITATION_RATE_LIMITED"

type invitationLimitFixture struct {
	t       *testing.T
	r       *support.ResourceRegistry
	db      *gorm.DB
	mail    *services.NoopEmailService
	service InvitationService
}

func newInvitationLimitFixture(t *testing.T) *invitationLimitFixture {
	t.Helper()
	r := support.Setup(t)
	t.Cleanup(r.Close)
	t.Setenv(models.BillingEnabledEnv, "no")
	mail := services.NewNoopEmailService()
	return &invitationLimitFixture{
		t:       t,
		r:       r,
		db:      database.Conn(),
		mail:    mail,
		service: InvitationService{AuthService: r.AuthService, EmailService: mail, BaseURL: "https://example.com"},
	}
}

func (f *invitationLimitFixture) create(email string) (*pb.CreateInvitationResponse, error) {
	return f.service.Create(context.Background(), f.r.Organization.ID.String(), f.r.User.String(), &pb.CreateInvitationRequest{Email: email})
}

func (f *invitationLimitFixture) resend(invitationID string) error {
	_, err := f.service.Resend(context.Background(), f.r.Organization.ID.String(), f.r.User.String(), invitationID)
	return err
}

// seedSends records send rows that sent the given age ago.
func (f *invitationLimitFixture) seedSends(orgID uuid.UUID, email string, count int, age time.Duration) {
	f.t.Helper()
	for i := 0; i < count; i++ {
		address := email
		if address == "" {
			address = fmt.Sprintf("seeded-%d@example.com", i)
		}
		require.NoError(f.t, models.RecordOrganizationInvitationSend(f.db, orgID, address, time.Now().Add(-age)))
	}
}

func (f *invitationLimitFixture) seedPendingInvitations(count int) {
	f.t.Helper()
	for i := 0; i < count; i++ {
		invitation := &models.OrganizationInvitation{
			OrganizationID: f.r.Organization.ID,
			Email:          fmt.Sprintf("pending-%d@example.com", i),
			Role:           models.RoleOrgOperator,
			InvitedBy:      f.r.User,
			State:          models.InvitationStatePending,
			SentAt:         time.Now(),
			ExpiresAt:      time.Now().Add(time.Hour),
		}
		require.NoError(f.t, invitation.Save(f.db))
	}
}

func (f *invitationLimitFixture) sendCount() int64 {
	f.t.Helper()
	var count int64
	require.NoError(f.t, f.db.Model(&models.OrganizationInvitationSend{}).Where("organization_id = ?", f.r.Organization.ID).Count(&count).Error)
	return count
}

func (f *invitationLimitFixture) invitationCount() int64 {
	f.t.Helper()
	var count int64
	require.NoError(f.t, f.db.Model(&models.OrganizationInvitation{}).Where("organization_id = ?", f.r.Organization.ID).Count(&count).Error)
	return count
}

func TestInvitationOrganizationSendLimit(t *testing.T) {
	f := newInvitationLimitFixture(t)
	f.seedSends(f.r.Organization.ID, "", 50, 2*time.Hour)

	_, err := f.create("old-sends@example.com")
	require.NoError(t, err, "sends older than one hour must not count")
	require.Len(t, f.mail.SentOrganizationInvitationEmails(), 1)

	f.mail.Reset()
	f.seedSends(f.r.Organization.ID, "", 49, 10*time.Minute) // 49 recent plus the one above makes 50
	invitationsBefore, sendsBefore := f.invitationCount(), f.sendCount()

	_, err = f.create("blocked@example.com")
	requireInvitationErrorReason(t, err, codes.ResourceExhausted, rateLimitedReason)
	require.Empty(t, f.mail.SentOrganizationInvitationEmails())
	require.Equal(t, invitationsBefore, f.invitationCount())
	require.Equal(t, sendsBefore, f.sendCount())
}

func TestInvitationRecipientSendLimit(t *testing.T) {
	f := newInvitationLimitFixture(t)
	const address = "recipient@example.com"

	first, err := f.create(address)
	require.NoError(t, err)
	require.EqualValues(t, 1, f.sendCount(), "Create writes one send row")
	for i := 0; i < 4; i++ {
		require.NoError(t, f.resend(first.Invitation.Id))
		require.EqualValues(t, i+2, f.sendCount(), "Resend writes one send row")
	}
	require.Len(t, f.mail.SentOrganizationInvitationEmails(), 5)

	f.mail.Reset()
	requireInvitationErrorReason(t, f.resend(first.Invitation.Id), codes.ResourceExhausted, rateLimitedReason)
	require.Empty(t, f.mail.SentOrganizationInvitationEmails())
	require.EqualValues(t, 5, f.sendCount())

	_, err = f.create("other@example.com")
	require.NoError(t, err, "another address has its own limit")
}

func TestInvitationRecipientSendLimitWindowAndScope(t *testing.T) {
	f := newInvitationLimitFixture(t)
	otherOrg := support.CreateOrganization(t, f.r, f.r.User)
	f.seedSends(otherOrg.ID, "recipient@example.com", 5, time.Minute)
	f.seedSends(f.r.Organization.ID, "recipient@example.com", 5, 25*time.Hour)

	_, err := f.create("recipient@example.com")
	require.NoError(t, err, "other organizations and sends older than one day must not count")

	f.seedSends(f.r.Organization.ID, "other@example.com", 5, 23*time.Hour)
	_, err = f.create("other@example.com")
	requireInvitationErrorReason(t, err, codes.ResourceExhausted, rateLimitedReason)
}

func TestInvitationLimitEnvironmentOverride(t *testing.T) {
	f := newInvitationLimitFixture(t)
	t.Setenv("INVITATION_ORG_HOURLY_LIMIT", "2")
	t.Setenv("PENDING_INVITATION_LIMIT", "3")

	for _, email := range []string{"a@example.com", "b@example.com"} {
		_, err := f.create(email)
		require.NoError(t, err)
	}
	_, err := f.create("c@example.com")
	requireInvitationErrorReason(t, err, codes.ResourceExhausted, rateLimitedReason)

	t.Setenv("INVITATION_ORG_HOURLY_LIMIT", "10")
	_, err = f.create("c@example.com")
	require.NoError(t, err)
	_, err = f.create("d@example.com")
	requireInvitationReason(t, err, "PENDING_INVITATION_LIMIT_REACHED")
}

func TestLoadInvitationLimitsIgnoresInvalidValues(t *testing.T) {
	for _, value := range []string{"0", "-5", "ten", "1.5"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("INVITATION_ORG_HOURLY_LIMIT", value)
			t.Setenv("INVITATION_RECIPIENT_DAILY_LIMIT", value)
			t.Setenv("PENDING_INVITATION_LIMIT", value)
			require.Equal(t, invitationLimits{orgHourly: 50, recipientDaily: 5, pending: 100}, loadInvitationLimits())
		})
	}
	t.Setenv("INVITATION_ORG_HOURLY_LIMIT", "7")
	require.EqualValues(t, 7, loadInvitationLimits().orgHourly)
}

func TestPendingInvitationLimit(t *testing.T) {
	for _, tc := range []struct{ name, plan, billing string }{
		{name: "billing off", billing: "no"},
		{name: "none plan", plan: models.BillingPlanNone, billing: "yes"},
		{name: "trial plan", plan: models.BillingPlanTrial, billing: "yes"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newInvitationLimitFixture(t)
			t.Setenv(models.BillingEnabledEnv, tc.billing)
			if tc.plan != "" {
				_, err := models.SetAdminOrganizationPlan(f.db, f.r.Organization.ID, tc.plan, nil)
				require.NoError(t, err)
			}
			first, err := f.create("first@example.com")
			require.NoError(t, err)
			f.seedPendingInvitations(defaultPendingInvitationLimit - 1)
			f.mail.Reset()

			_, err = f.create("one-too-many@example.com")
			requireInvitationReason(t, err, "PENDING_INVITATION_LIMIT_REACHED")
			require.Empty(t, f.mail.SentOrganizationInvitationEmails())

			require.NoError(t, f.resend(first.Invitation.Id), "a resend does not add a pending invitation")
			require.Len(t, f.mail.SentOrganizationInvitationEmails(), 1)
		})
	}
}

func TestPendingInvitationLimitCountsExpiredResend(t *testing.T) {
	f := newInvitationLimitFixture(t)
	first, err := f.create("first@example.com")
	require.NoError(t, err)
	require.NoError(t, f.db.Model(&models.OrganizationInvitation{}).Where("id = ?", first.Invitation.Id).Update("expires_at", time.Now().Add(-time.Hour)).Error)
	f.seedPendingInvitations(defaultPendingInvitationLimit)

	requireInvitationReason(t, f.resend(first.Invitation.Id), "PENDING_INVITATION_LIMIT_REACHED")
}
