package organizations

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/ee/billing/entitlements"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/services"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func TestInvitationSeatReservations(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	db := database.Conn()
	setPurchasedSeatPlan(t, db, r.Organization.ID, seatQuantity(10))
	for i := 0; i < 8; i++ {
		support.CreateUser(t, r, r.Organization.ID)
	}
	mail := services.NewNoopEmailService()
	s := InvitationService{AuthService: r.AuthService, EmailService: mail, BaseURL: "https://example.com/"}
	var polarCalls atomic.Int32
	polar := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		polarCalls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer polar.Close()
	t.Setenv("POLAR_ACCESS_TOKEN", "test-token")
	t.Setenv("POLAR_API_BASE_URL", polar.URL)
	t.Cleanup(func() { require.Zero(t, polarCalls.Load()) })
	ctx := context.Background()
	create := func(email string) (*pb.CreateInvitationResponse, error) {
		return s.Create(ctx, r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: email})
	}
	before, err := entitlements.ResolveSeats(db, r.Organization.ID)
	require.NoError(t, err)
	first, err := create(" First@Example.com ")
	require.NoError(t, err)
	require.Equal(t, "first@example.com", first.Invitation.Email)
	require.Equal(t, models.RoleOrgOperator, first.Invitation.Role)
	require.WithinDuration(t, time.Now().Add(7*24*time.Hour), first.Invitation.ExpiresAt.AsTime(), time.Minute)
	stored, err := models.FindOrganizationInvitation(db, r.Organization.ID, uuid.MustParse(first.Invitation.Id))
	require.NoError(t, err)
	listedAfterCreate, err := s.List(ctx, r.Organization.ID.String())
	require.NoError(t, err)
	require.Len(t, listedAfterCreate.Invitations, 1)
	require.Equal(t, first.Invitation.SentAt.AsTime(), listedAfterCreate.Invitations[0].SentAt.AsTime())
	require.Equal(t, first.Invitation.ExpiresAt.AsTime(), listedAfterCreate.Invitations[0].ExpiresAt.AsTime())
	token := strings.TrimPrefix(first.AcceptanceUrl, "https://example.com/invitations/")
	require.NotEmpty(t, token)
	require.Equal(t, crypto.HashToken(token), *stored.TokenHash)
	require.Len(t, mail.SentOrganizationInvitationEmails(), 1)
	_, err = create("second@example.com")
	requireInvitationReason(t, err, "SEAT_PURCHASE_REQUIRED")
	after, err := entitlements.ResolveSeats(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, before.Available, after.Available)
	require.Equal(t, before.RenewalQuantity, after.RenewalQuantity)
	require.Equal(t, entitlements.JoinLimit(before), entitlements.JoinLimit(after))
	require.Equal(t, 1, after.PendingInvitations)
	require.Equal(t, 0, *after.Invitable)
	resent, err := s.Resend(ctx, r.Organization.ID.String(), r.User.String(), first.Invitation.Id)
	require.NoError(t, err)
	require.NotEqual(t, first.AcceptanceUrl, resent.AcceptanceUrl)
	stored, err = models.FindOrganizationInvitation(db, r.Organization.ID, stored.ID)
	require.NoError(t, err)
	require.NotEqual(t, crypto.HashToken(token), *stored.TokenHash)
	require.Equal(t, crypto.HashToken(strings.TrimPrefix(resent.AcceptanceUrl, "https://example.com/invitations/")), *stored.TokenHash)
	require.WithinDuration(t, time.Now().Add(7*24*time.Hour), stored.ExpiresAt, time.Minute)
	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("purchased_seats", 11).Error)
	_, err = create("second@example.com")
	require.NoError(t, err)
	_, err = s.Cancel(ctx, r.Organization.ID.String(), first.Invitation.Id)
	require.NoError(t, err)
	_, err = s.Cancel(ctx, r.Organization.ID.String(), first.Invitation.Id)
	require.NoError(t, err)
	stored, err = models.FindOrganizationInvitation(db, r.Organization.ID, stored.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.TokenHash)
	_, err = s.Resend(ctx, r.Organization.ID.String(), r.User.String(), first.Invitation.Id)
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	listed, err := s.List(ctx, r.Organization.ID.String())
	require.NoError(t, err)
	require.Equal(t, "second@example.com", listed.Invitations[0].Email)
	require.Equal(t, first.Invitation.Id, listed.Invitations[1].Id)
	require.Equal(t, "canceled", listed.Invitations[1].State)
	require.Equal(t, resent.Invitation.SentAt.AsTime(), listed.Invitations[1].SentAt.AsTime())
	require.Equal(t, resent.Invitation.ExpiresAt.AsTime(), listed.Invitations[1].ExpiresAt.AsTime())
	seats, err := entitlements.ResolveSeats(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, 1, *seats.Invitable)
}

func TestInvitationExpiryAndValidation(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "no")
	db := database.Conn()
	ctx := context.Background()
	settings, err := models.GetInstallationMetadata(db)
	require.NoError(t, err)
	settings.InvitationDurationDays = 14
	require.NoError(t, models.UpdateInstallationMetadata(db, settings))
	s := InvitationService{AuthService: r.AuthService, BaseURL: "https://example.com"}
	create := func(email, role string) (*pb.CreateInvitationResponse, error) {
		return s.Create(ctx, r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: email, Role: role})
	}
	for _, email := range []string{"", "bad", "Name <person@example.com>", "a@example.com\r\nBcc: x@y.com"} {
		_, err := create(email, "")
		require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	}
	_, err = create("unknown@example.com", "missing")
	require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	account, err := models.CreateAccount("Removed member", "removed@example.com")
	require.NoError(t, err)
	user, err := models.CreateUser(r.Organization.ID, account.ID, account.Email, account.Name)
	require.NoError(t, err)
	_, err = create(user.GetEmail(), "")
	require.Equal(t, codes.AlreadyExists, grpcerrors.Code(err))
	require.NoError(t, db.Delete(user).Error)
	_, err = create(user.GetEmail(), "")
	require.NoError(t, err)
	first, err := create("person@example.com", models.RoleOrgMaintainer)
	require.NoError(t, err)
	require.Equal(t, 14*24*time.Hour, first.Invitation.ExpiresAt.AsTime().Sub(first.Invitation.SentAt.AsTime()))
	settings.InvitationDurationDays = 3
	require.NoError(t, models.UpdateInstallationMetadata(db, settings))
	stored, err := models.FindOrganizationInvitation(db, r.Organization.ID, uuid.MustParse(first.Invitation.Id))
	require.NoError(t, err)
	require.Equal(t, 14*24*time.Hour, stored.ExpiresAt.Sub(stored.SentAt))
	_, err = create("PERSON@example.com", "")
	require.Equal(t, codes.AlreadyExists, grpcerrors.Code(err))
	require.NoError(t, db.Model(&models.OrganizationInvitation{}).Where("id = ?", first.Invitation.Id).Update("expires_at", time.Now().Add(-time.Hour)).Error)
	listed, err := s.List(ctx, r.Organization.ID.String())
	require.NoError(t, err)
	require.Equal(t, int32(3), listed.InvitationDurationDays)
	require.Equal(t, "expired", listed.Invitations[0].State)
	require.Zero(t, listed.Seats.PendingInvitations)
	require.Nil(t, listed.Seats.Invitable)
	next, err := create("PERSON@example.com", "")
	require.NoError(t, err)
	_, err = s.Resend(ctx, r.Organization.ID.String(), r.User.String(), first.Invitation.Id)
	require.Equal(t, codes.AlreadyExists, grpcerrors.Code(err))
	_, err = s.Cancel(ctx, r.Organization.ID.String(), next.Invitation.Id)
	require.NoError(t, err)
	revived, err := s.Resend(ctx, r.Organization.ID.String(), r.User.String(), first.Invitation.Id)
	require.NoError(t, err)
	require.NotEqual(t, first.AcceptanceUrl, revived.AcceptanceUrl)
	require.Equal(t, 3*24*time.Hour, revived.Invitation.ExpiresAt.AsTime().Sub(revived.Invitation.SentAt.AsTime()))
	require.NoError(t, db.Model(&models.OrganizationInvitation{}).Where("id = ?", first.Invitation.Id).Update("state", "accepted").Error)
	_, err = s.Cancel(ctx, r.Organization.ID.String(), first.Invitation.Id)
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	_, err = s.Resend(ctx, r.Organization.ID.String(), r.User.String(), first.Invitation.Id)
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	other := support.CreateOrganization(t, r, r.User)
	_, err = s.Cancel(ctx, other.ID.String(), first.Invitation.Id)
	require.Equal(t, codes.NotFound, grpcerrors.Code(err))
	_, err = s.Resend(ctx, other.ID.String(), r.User.String(), first.Invitation.Id)
	require.Equal(t, codes.NotFound, grpcerrors.Code(err))
	otherList, err := s.List(ctx, other.ID.String())
	require.NoError(t, err)
	require.Empty(t, otherList.Invitations)
}

func TestInvitationFinalSeatConcurrent(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	setPurchasedSeatPlan(t, database.Conn(), r.Organization.ID, seatQuantity(2))
	s := InvitationService{AuthService: r.AuthService, BaseURL: "https://example.com"}
	start := make(chan struct{})
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			_, err := s.Create(context.Background(), r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: uuid.NewString() + "@example.com"})
			results <- err
		}()
	}
	close(start)
	success := 0
	for i := 0; i < 2; i++ {
		err := <-results
		if err == nil {
			success++
			continue
		}
		requireInvitationReason(t, err, "SEAT_PURCHASE_REQUIRED")
	}
	require.Equal(t, 1, success)
}

func requireInvitationReason(t *testing.T, err error, reason string) {
	t.Helper()
	requireInvitationErrorReason(t, err, codes.FailedPrecondition, reason)
}

func requireInvitationErrorReason(t *testing.T, err error, code codes.Code, reason string) {
	t.Helper()
	require.Equal(t, code, grpcerrors.Code(err))
	details := status.Convert(err).Details()
	require.Len(t, details, 1)
	require.Equal(t, reason, details[0].(*errdetails.ErrorInfo).Reason)
}

type committedInvitationEmail struct {
	services.EmailService
	t     *testing.T
	orgID uuid.UUID
	fail  bool
	calls int
}

func (e *committedInvitationEmail) SendOrganizationInvitationEmail(to string, data services.OrganizationInvitationTemplateData) error {
	rows, err := models.ListOrganizationInvitations(database.Conn(), e.orgID)
	require.NoError(e.t, err)
	require.Len(e.t, rows, 1)
	require.Equal(e.t, to, rows[0].Email)
	token := strings.TrimPrefix(data.AcceptanceURL, "https://example.com/invitations/")
	require.Equal(e.t, crypto.HashToken(token), *rows[0].TokenHash)
	e.calls++
	if e.fail {
		return errors.New("delivery failed")
	}
	return nil
}

func TestInvitationEmailAfterCommit(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "delivered", true: "failed"}[fail], func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			mail := &committedInvitationEmail{t: t, orgID: r.Organization.ID, fail: fail}
			s := InvitationService{AuthService: r.AuthService, EmailService: mail, BaseURL: "https://example.com"}
			resp, err := s.Create(context.Background(), r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: "person@example.com"})
			require.NoError(t, err)
			require.NotEmpty(t, resp.AcceptanceUrl)
			require.Equal(t, 1, mail.calls)
		})
	}
}

func TestInvitationCapacityPolicies(t *testing.T) {
	for _, tc := range []struct {
		name              string
		perSeat           bool
		purchased         *int
		maximum           *int
		override          *int
		unlimitedOverride bool
		disabled          bool
		reason            string
	}{
		{name: "missing purchased capacity", perSeat: true, maximum: seatQuantity(50), reason: "SEAT_PURCHASE_REQUIRED"},
		{name: "plan maximum", perSeat: true, purchased: seatQuantity(1), maximum: seatQuantity(1), reason: "PLAN_MAXIMUM_REACHED"},
		{name: "admin maximum", perSeat: true, purchased: seatQuantity(1), maximum: seatQuantity(50), override: seatQuantity(1), reason: "PLAN_MAXIMUM_REACHED"},
		{name: "admin expanded maximum", perSeat: true, purchased: seatQuantity(1), maximum: seatQuantity(1), override: seatQuantity(3), reason: "SEAT_PURCHASE_REQUIRED"},
		{name: "admin unlimited maximum", perSeat: true, purchased: seatQuantity(1), maximum: seatQuantity(1), unlimitedOverride: true, reason: "SEAT_PURCHASE_REQUIRED"},
		{name: "other plan full", maximum: seatQuantity(1), reason: "MEMBER_LIMIT_REACHED"},
		{name: "other plan free capacity", maximum: seatQuantity(2)},
		{name: "other plan unlimited"},
		{name: "billing disabled", perSeat: true, maximum: seatQuantity(1), disabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			t.Setenv(billingmodels.BillingEnabledEnv, "yes")
			if tc.disabled {
				t.Setenv(billingmodels.BillingEnabledEnv, "no")
			}
			db := database.Conn()
			plan := billingmodels.BillingPlan{Key: "invite_" + uuid.NewString()[:8], Name: "Invitation plan", Category: "paid", Visibility: "private", PerSeat: tc.perSeat, SeatLimit: tc.maximum, CreditFrequency: "none", CreditExpiration: "never"}
			if tc.perSeat {
				plan.PolarProductID = new("prod_" + uuid.NewString())
			}
			require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
			if tc.perSeat {
				_, err := billingmodels.ApplyPolarSubscription(db, r.Organization.ID, billingmodels.PolarSubscriptionApply{
					ID: "sub_" + uuid.NewString(), ProductID: *plan.PolarProductID,
					Status: billingmodels.PolarSubscriptionStatusActive, PurchasedSeats: new(1),
				}, nil)
				require.NoError(t, err)
			}
			if !tc.perSeat {
				_, err := billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, plan.Key, nil)
				require.NoError(t, err)
			}
			require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("purchased_seats", tc.purchased).Error)
			if tc.override != nil || tc.unlimitedOverride {
				require.NoError(t, billingmodels.SetOrganizationSeatLimitOverride(db, r.Organization.ID, tc.override))
			}
			s := InvitationService{AuthService: r.AuthService, BaseURL: "https://example.com"}
			for _, email := range []string{"first@example.com", "second@example.com"} {
				resp, err := s.Create(context.Background(), r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: email})
				if tc.reason != "" {
					requireInvitationReason(t, err, tc.reason)
					continue
				}
				require.NoError(t, err)
				_, err = s.Resend(context.Background(), r.Organization.ID.String(), r.User.String(), resp.Invitation.Id)
				require.NoError(t, err)
			}
			if tc.reason == "" {
				listed, err := s.List(context.Background(), r.Organization.ID.String())
				require.NoError(t, err)
				expectedPending := int64(2)
				if tc.disabled {
					expectedPending = 0
				}
				require.Equal(t, expectedPending, listed.Seats.PendingInvitations)
				if !tc.perSeat || tc.disabled {
					require.Nil(t, listed.Seats.Invitable)
				}
			}
			if !tc.perSeat && tc.maximum != nil && tc.reason == "" {
				support.CreateUser(t, r, r.Organization.ID)
				list, err := s.List(context.Background(), r.Organization.ID.String())
				require.NoError(t, err)
				_, err = s.Resend(context.Background(), r.Organization.ID.String(), r.User.String(), list.Invitations[0].Id)
				requireInvitationReason(t, err, "MEMBER_LIMIT_REACHED")
			}
		})
	}
}

func TestExpiredInvitationRequiresSeatOnResend(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	db := database.Conn()
	setPurchasedSeatPlan(t, db, r.Organization.ID, seatQuantity(2))
	s := InvitationService{AuthService: r.AuthService, BaseURL: "https://example.com"}
	first, err := s.Create(context.Background(), r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: "first@example.com"})
	require.NoError(t, err)
	require.NoError(t, db.Model(&models.OrganizationInvitation{}).Where("id = ?", first.Invitation.Id).Update("expires_at", time.Now().Add(-time.Hour)).Error)
	_, err = s.Create(context.Background(), r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: "second@example.com"})
	require.NoError(t, err)
	_, err = s.Resend(context.Background(), r.Organization.ID.String(), r.User.String(), first.Invitation.Id)
	requireInvitationReason(t, err, "SEAT_PURCHASE_REQUIRED")
}

type invitationEntitlements bool

func (e invitationEntitlements) IsEntitled(feature licensing.Feature) bool {
	return bool(e) && feature == licensing.FeatureCustomRoles
}

func TestInvitationCustomRoleEntitlement(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "no")
	require.NoError(t, r.AuthService.CreateCustomRole(r.Organization.ID.String(), &authorization.RoleDefinition{Name: "support", DomainType: models.DomainTypeOrganization, InheritsFrom: &authorization.RoleDefinition{Name: models.RoleOrgOperator}}))
	s := InvitationService{AuthService: r.AuthService, BaseURL: "https://example.com"}
	req := &pb.CreateInvitationRequest{Email: "custom@example.com", Role: "support"}
	_, err := s.Create(context.Background(), r.Organization.ID.String(), r.User.String(), req)
	require.Equal(t, codes.PermissionDenied, grpcerrors.Code(err))
	s.Entitlements = invitationEntitlements(true)
	resp, err := s.Create(context.Background(), r.Organization.ID.String(), r.User.String(), req)
	require.NoError(t, err)
	require.Equal(t, "support", resp.Invitation.Role)
	s.Entitlements = invitationEntitlements(false)
	_, err = s.Resend(context.Background(), r.Organization.ID.String(), r.User.String(), resp.Invitation.Id)
	require.Equal(t, codes.PermissionDenied, grpcerrors.Code(err))
}

func TestResendElevatedInvitationRequiresUpdatePermission(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "no")
	s := InvitationService{AuthService: r.AuthService, BaseURL: "https://example.com"}
	adminInvite, err := s.Create(context.Background(), r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: "admin-invite@example.com", Role: models.RoleOrgAdmin})
	require.NoError(t, err)
	operatorInvite, err := s.Create(context.Background(), r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: "operator-invite@example.com"})
	require.NoError(t, err)
	stored, err := models.FindOrganizationInvitation(database.Conn(), r.Organization.ID, uuid.MustParse(adminInvite.Invitation.Id))
	require.NoError(t, err)
	originalHash := *stored.TokenHash

	account, err := models.CreateAccount("Inviter", "inviter@example.com")
	require.NoError(t, err)
	member, err := models.CreateUser(r.Organization.ID, account.ID, account.Email, account.Name)
	require.NoError(t, err)
	require.NoError(t, r.AuthService.CreateCustomRole(r.Organization.ID.String(), &authorization.RoleDefinition{
		Name:        "inviter",
		DomainType:  models.DomainTypeOrganization,
		Permissions: []*authorization.Permission{{Resource: "members", Action: "create"}},
	}))
	require.NoError(t, r.AuthService.AssignRole(member.ID.String(), "inviter", r.Organization.ID.String(), models.DomainTypeOrganization))

	_, err = s.Resend(context.Background(), r.Organization.ID.String(), member.ID.String(), adminInvite.Invitation.Id)
	require.Equal(t, codes.PermissionDenied, grpcerrors.Code(err))
	stored, err = models.FindOrganizationInvitation(database.Conn(), r.Organization.ID, stored.ID)
	require.NoError(t, err)
	require.Equal(t, originalHash, *stored.TokenHash)

	_, err = s.Resend(context.Background(), r.Organization.ID.String(), member.ID.String(), operatorInvite.Invitation.Id)
	require.NoError(t, err)

	scopes, err := json.Marshal([]string{"members:create"})
	require.NoError(t, err)
	limited := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-token-scopes", string(scopes)))
	_, err = s.Resend(limited, r.Organization.ID.String(), r.User.String(), adminInvite.Invitation.Id)
	require.Equal(t, codes.PermissionDenied, grpcerrors.Code(err))
	stored, err = models.FindOrganizationInvitation(database.Conn(), r.Organization.ID, stored.ID)
	require.NoError(t, err)
	require.Equal(t, originalHash, *stored.TokenHash)
}

func TestInvitationEmailRoleLabels(t *testing.T) {
	for _, tc := range []struct {
		role, displayName, expected string
		missing, lookupFailure      bool
	}{
		{role: models.RoleOrgAdmin, expected: "Admin"},
		{role: models.RoleOrgMaintainer, expected: "Maintainer"},
		{role: models.RoleOrgOperator, expected: "Operator"},
		{role: "incident_responder", displayName: "Incident responder", expected: "Incident responder"},
		{role: "empty_label", expected: "empty_label"},
		{role: "missing_metadata", expected: "missing_metadata", missing: true},
		{role: "unavailable_metadata", displayName: "Unavailable metadata", expected: "unavailable_metadata", lookupFailure: true},
	} {
		t.Run(tc.role, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			t.Setenv(billingmodels.BillingEnabledEnv, "no")
			if !r.AuthService.IsDefaultRole(tc.role, models.DomainTypeOrganization) {
				require.NoError(t, r.AuthService.CreateCustomRole(r.Organization.ID.String(), &authorization.RoleDefinition{Name: tc.role, DisplayName: tc.displayName, DomainType: models.DomainTypeOrganization, InheritsFrom: &authorization.RoleDefinition{Name: models.RoleOrgOperator}}))
				if tc.missing {
					require.NoError(t, models.DeleteRoleMetadata(tc.role, models.DomainTypeOrganization, r.Organization.ID.String()))
				}
			}
			if tc.lookupFailure {
				db := database.Conn()
				callback := "test:fail_invitation_role_metadata"
				require.NoError(t, db.Callback().Query().Before("gorm:query").Register(callback, func(tx *gorm.DB) {
					if tx.Statement.Table == "role_metadata" {
						tx.AddError(errors.New("metadata lookup interrupted"))
					}
				}))
				defer db.Callback().Query().Remove(callback)
			}
			mail := services.NewNoopEmailService()
			s := InvitationService{AuthService: r.AuthService, EmailService: mail, Entitlements: invitationEntitlements(true), BaseURL: "https://example.com"}
			created, err := s.Create(context.Background(), r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: "invitee@example.com", Role: tc.role})
			require.NoError(t, err)
			_, err = s.Resend(context.Background(), r.Organization.ID.String(), r.User.String(), created.Invitation.Id)
			require.NoError(t, err)
			emails := mail.SentOrganizationInvitationEmails()
			require.Len(t, emails, 2)
			for _, email := range emails {
				require.Equal(t, tc.expected, email.Data.Role)
			}
		})
	}
}
