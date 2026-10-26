package organizations

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func TestAcceptInvitationMembershipLifecycle(t *testing.T) {
	for _, restore := range []bool{false, true} {
		t.Run(map[bool]string{false: "new member", true: "removed member"}[restore], func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			t.Setenv(billingmodels.BillingEnabledEnv, "yes")
			db := database.Conn()
			setPurchasedSeatPlan(t, db, r.Organization.ID, seatQuantity(2))
			account, err := models.CreateAccount("Invitee", "invitee@example.com")
			require.NoError(t, err)
			var previousID uuid.UUID
			if restore {
				user, err := models.CreateUser(r.Organization.ID, account.ID, account.Email, account.Name)
				require.NoError(t, err)
				previousID = user.ID
				require.NoError(t, r.AuthService.AssignRole(user.ID.String(), models.RoleOrgAdmin, r.Organization.ID.String(), models.DomainTypeOrganization))
				require.NoError(t, db.Delete(user).Error)
			}
			s := InvitationService{AuthService: r.AuthService, BaseURL: "https://example.com"}
			created, err := s.Create(context.Background(), r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: account.Email, Role: models.RoleOrgMaintainer})
			require.NoError(t, err)
			token := strings.TrimPrefix(created.AcceptanceUrl, "https://example.com/invitations/")
			response, err := s.Accept(context.Background(), account.ID.String(), token)
			require.NoError(t, err)
			require.Equal(t, "joined", response.Fields["status"].GetStringValue())
			require.Equal(t, r.Organization.ID.String(), response.Fields["organization_id"].GetStringValue())
			user, err := models.FindActiveUserByEmail(r.Organization.ID.String(), account.Email)
			require.NoError(t, err)
			if restore {
				require.Equal(t, previousID, user.ID)
			}
			roles, err := r.AuthService.GetUserRolesForOrg(context.Background(), user.ID.String(), r.Organization.ID.String())
			require.NoError(t, err)
			require.ElementsMatch(t, []string{models.RoleOrgMaintainer, models.RoleOrgOperator}, invitationRoleNames(roles))
			stored, err := models.FindOrganizationInvitation(db, r.Organization.ID, uuid.MustParse(created.Invitation.Id))
			require.NoError(t, err)
			require.Equal(t, models.InvitationStateAccepted, stored.State)
			require.Equal(t, user.ID, *stored.AcceptedBy)
			require.WithinDuration(t, time.Now(), *stored.AcceptedAt, time.Minute)
			require.NoError(t, r.AuthService.AssignRole(user.ID.String(), models.RoleOrgOperator, r.Organization.ID.String(), models.DomainTypeOrganization))
			response, err = s.Accept(context.Background(), account.ID.String(), token)
			require.NoError(t, err)
			require.Equal(t, "already_member", response.Fields["status"].GetStringValue())
			roles, err = r.AuthService.GetUserRolesForOrg(context.Background(), user.ID.String(), r.Organization.ID.String())
			require.NoError(t, err)
			require.Equal(t, []string{models.RoleOrgOperator}, invitationRoleNames(roles))
			occupied, err := models.CountActiveHumanUsersByOrganization(db, r.Organization.ID.String())
			require.NoError(t, err)
			require.EqualValues(t, 2, occupied)
			// An accepted link must not restore access after the member is removed.
			require.NoError(t, db.Delete(user).Error)
			_, err = s.Accept(context.Background(), account.ID.String(), token)
			requireInvitationReason(t, err, "INVITATION_ALREADY_ACCEPTED")
			_, err = models.FindActiveUserByEmail(r.Organization.ID.String(), account.Email)
			require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		})
	}
}

func TestAcceptInvitationRejectsInvalidRecipientOrState(t *testing.T) {
	for _, scenario := range []string{"wrong email", "expired", "swept expiry", "canceled", "rotated", "capacity removed", "member limit", "unknown token"} {
		t.Run(scenario, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			t.Setenv(billingmodels.BillingEnabledEnv, "yes")
			db := database.Conn()
			setPurchasedSeatPlan(t, db, r.Organization.ID, seatQuantity(2))
			account, err := models.CreateAccount("Invitee", "invitee@example.com")
			require.NoError(t, err)
			s := InvitationService{AuthService: r.AuthService, BaseURL: "https://example.com"}
			created, err := s.Create(context.Background(), r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: account.Email})
			require.NoError(t, err)
			token := strings.TrimPrefix(created.AcceptanceUrl, "https://example.com/invitations/")
			expected := codes.FailedPrecondition
			reason := map[string]string{"wrong email": "INVITATION_EMAIL_MISMATCH", "expired": "INVITATION_EXPIRED", "swept expiry": "INVITATION_EXPIRED", "canceled": "INVITATION_CANCELED", "rotated": "INVITATION_REPLACED", "capacity removed": "NO_SEAT_AVAILABLE", "member limit": "MEMBER_LIMIT_REACHED"}[scenario]
			switch scenario {
			case "wrong email":
				account, err = models.CreateAccount("Other person", "other@example.com")
				require.NoError(t, err)
				expected = codes.PermissionDenied
			case "expired", "swept expiry":
				require.NoError(t, db.Model(&models.OrganizationInvitation{}).Where("id = ?", created.Invitation.Id).Update("expires_at", time.Now().Add(-time.Hour)).Error)
				if scenario == "swept expiry" {
					require.NoError(t, models.ExpireOrganizationInvitations(db, r.Organization.ID, time.Now()))
				}
			case "canceled":
				_, err := s.Cancel(context.Background(), r.Organization.ID.String(), created.Invitation.Id)
				require.NoError(t, err)
			case "rotated":
				_, err := s.Resend(context.Background(), r.Organization.ID.String(), r.User.String(), created.Invitation.Id)
				require.NoError(t, err)
			case "capacity removed":
				require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("purchased_seats", 1).Error)
			case "member limit":
				require.NoError(t, billingmodels.SetOrganizationSeatLimitOverride(db, r.Organization.ID, seatQuantity(1)))
				require.NoError(t, db.Model(&billingmodels.BillingPlan{}).Where("id = (SELECT plan_id FROM organization_billing_plans WHERE organization_id = ?)", r.Organization.ID).Update("per_seat", false).Error)
			case "unknown token":
				token = "unknown"
				expected = codes.NotFound
			}
			_, err = s.Accept(context.Background(), account.ID.String(), token)
			require.Equal(t, expected, grpcerrors.Code(err))
			if reason != "" {
				details := status.Convert(err).Details()
				require.Len(t, details, 1)
				info := details[0].(*errdetails.ErrorInfo)
				require.Equal(t, reason, info.Reason)
				require.Equal(t, "superplane.com", info.Domain)
				if scenario == "wrong email" {
					require.Equal(t, map[string]string{"invited_email": "i***@example.com"}, info.Metadata)
				}
			}
			occupied, err := models.CountActiveHumanUsersByOrganization(db, r.Organization.ID.String())
			require.NoError(t, err)
			require.EqualValues(t, 1, occupied)
			stored, err := models.FindOrganizationInvitation(db, r.Organization.ID, uuid.MustParse(created.Invitation.Id))
			require.NoError(t, err)
			require.NotEqual(t, models.InvitationStateAccepted, stored.State)
			if scenario == "capacity removed" || scenario == "member limit" {
				require.Equal(t, models.InvitationStatePending, stored.State)
				require.Nil(t, stored.AcceptedAt)
				require.Nil(t, stored.AcceptedBy)
			}
		})
	}
}

func TestAcceptInvitationForExistingMemberWhilePaymentBlocked(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	db := database.Conn()
	setPurchasedSeatPlan(t, db, r.Organization.ID, seatQuantity(2))
	account, err := models.CreateAccount("Invitee", "invitee@example.com")
	require.NoError(t, err)
	s := InvitationService{AuthService: r.AuthService, BaseURL: "https://example.com"}
	created, err := s.Create(context.Background(), r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: account.Email})
	require.NoError(t, err)
	// The invitee joins through another path before accepting this invitation.
	_, err = models.CreateUser(r.Organization.ID, account.ID, account.Email, account.Name)
	require.NoError(t, err)
	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("polar_subscription_status", billingmodels.PolarSubscriptionStatusPastDue).Error)
	token := strings.TrimPrefix(created.AcceptanceUrl, "https://example.com/invitations/")
	for range 2 {
		response, err := s.Accept(context.Background(), account.ID.String(), token)
		require.NoError(t, err)
		require.Equal(t, "already_member", response.Fields["status"].GetStringValue())
	}
	occupied, err := models.CountActiveHumanUsersByOrganization(db, r.Organization.ID.String())
	require.NoError(t, err)
	require.EqualValues(t, 2, occupied)
}

func TestAcceptInvitationConcurrentRetries(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	db := database.Conn()
	setPurchasedSeatPlan(t, db, r.Organization.ID, seatQuantity(2))
	account, err := models.CreateAccount("Invitee", "invitee@example.com")
	require.NoError(t, err)
	s := InvitationService{AuthService: r.AuthService, BaseURL: "https://example.com"}
	created, err := s.Create(context.Background(), r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: account.Email})
	require.NoError(t, err)
	token := strings.TrimPrefix(created.AcceptanceUrl, "https://example.com/invitations/")
	start := make(chan struct{})
	type result struct {
		status string
		err    error
	}
	results := make(chan result, 2)
	for range 2 {
		go func() {
			<-start
			response, err := s.Accept(context.Background(), account.ID.String(), token)
			if err != nil {
				results <- result{err: err}
				return
			}
			results <- result{status: response.Fields["status"].GetStringValue()}
		}()
	}
	close(start)
	statuses := []string{}
	for range 2 {
		result := <-results
		require.NoError(t, result.err)
		statuses = append(statuses, result.status)
	}
	require.ElementsMatch(t, []string{"joined", "already_member"}, statuses)
	occupied, err := models.CountActiveHumanUsersByOrganization(db, r.Organization.ID.String())
	require.NoError(t, err)
	require.EqualValues(t, 2, occupied)
}

type failingInvitationAuthorization struct {
	authorization.Authorization
}

func (a failingInvitationAuthorization) AssignRoleInTransaction(tx *gorm.DB, userID, role, domainID, domainType string) error {
	if err := a.Authorization.AssignRoleInTransaction(tx, userID, role, domainID, domainType); err != nil {
		return err
	}
	return errors.New("role assignment interrupted")
}

func TestAcceptInvitationRollsBackMembershipAndRole(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "no")
	db := database.Conn()
	account, err := models.CreateAccount("Removed member", "removed@example.com")
	require.NoError(t, err)
	user, err := models.CreateUser(r.Organization.ID, account.ID, account.Email, account.Name)
	require.NoError(t, err)
	require.NoError(t, r.AuthService.AssignRole(user.ID.String(), models.RoleOrgAdmin, r.Organization.ID.String(), models.DomainTypeOrganization))
	require.NoError(t, db.Delete(user).Error)
	s := InvitationService{AuthService: failingInvitationAuthorization{r.AuthService}, BaseURL: "https://example.com"}
	created, err := s.Create(context.Background(), r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: account.Email, Role: models.RoleOrgMaintainer})
	require.NoError(t, err)
	token := strings.TrimPrefix(created.AcceptanceUrl, "https://example.com/invitations/")
	_, err = s.Accept(context.Background(), account.ID.String(), token)
	require.Equal(t, codes.Internal, grpcerrors.Code(err))
	_, err = models.FindActiveUserByEmail(r.Organization.ID.String(), account.Email)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	stored, err := models.FindOrganizationInvitation(db, r.Organization.ID, uuid.MustParse(created.Invitation.Id))
	require.NoError(t, err)
	require.Equal(t, models.InvitationStatePending, stored.State)
	require.Nil(t, stored.AcceptedAt)
	roles, err := r.AuthService.GetUserRolesForOrg(context.Background(), user.ID.String(), r.Organization.ID.String())
	require.NoError(t, err)
	require.ElementsMatch(t, []string{models.RoleOrgAdmin, models.RoleOrgMaintainer, models.RoleOrgOperator}, invitationRoleNames(roles))
}

func TestAcceptInvitationRequiresCurrentCustomRoleLicense(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "no")
	account, err := models.CreateAccount("Invitee", "invitee@example.com")
	require.NoError(t, err)
	require.NoError(t, r.AuthService.CreateCustomRole(r.Organization.ID.String(), &authorization.RoleDefinition{Name: "support", DomainType: models.DomainTypeOrganization, InheritsFrom: &authorization.RoleDefinition{Name: models.RoleOrgOperator}}))
	s := InvitationService{AuthService: r.AuthService, Entitlements: invitationEntitlements(true), BaseURL: "https://example.com"}
	created, err := s.Create(context.Background(), r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: account.Email, Role: "support"})
	require.NoError(t, err)
	token := strings.TrimPrefix(created.AcceptanceUrl, "https://example.com/invitations/")
	s.Entitlements = invitationEntitlements(false)
	_, err = s.Accept(context.Background(), account.ID.String(), token)
	require.Equal(t, codes.PermissionDenied, grpcerrors.Code(err))
	_, err = models.FindActiveUserByEmail(r.Organization.ID.String(), account.Email)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	s.Entitlements = invitationEntitlements(true)
	_, err = s.Accept(context.Background(), account.ID.String(), token)
	require.NoError(t, err)
	user, err := models.FindActiveUserByEmail(r.Organization.ID.String(), account.Email)
	require.NoError(t, err)
	roles, err := r.AuthService.GetUserRolesForOrg(context.Background(), user.ID.String(), r.Organization.ID.String())
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"support", models.RoleOrgOperator}, invitationRoleNames(roles))
}

func invitationRoleNames(roles []*authorization.RoleDefinition) []string {
	names := make([]string, 0, len(roles))
	for _, role := range roles {
		names = append(names, role.Name)
	}
	return names
}

func TestAcceptInvitationsCompeteForLastSeat(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "yes")
	db := database.Conn()
	setPurchasedSeatPlan(t, db, r.Organization.ID, seatQuantity(3))
	s := InvitationService{AuthService: r.AuthService, BaseURL: "https://example.com"}
	accounts := make([]*models.Account, 0, 2)
	tokens := make([]string, 0, 2)
	for range 2 {
		account, err := models.CreateAccount("Invitee", uuid.NewString()+"@example.com")
		require.NoError(t, err)
		accounts = append(accounts, account)
		created, err := s.Create(context.Background(), r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: account.Email})
		require.NoError(t, err)
		tokens = append(tokens, strings.TrimPrefix(created.AcceptanceUrl, "https://example.com/invitations/"))
	}
	// Capacity can change after invitations have reserved it.
	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("purchased_seats", 2).Error)
	start := make(chan struct{})
	results := make(chan error, 2)
	for i := range accounts {
		go func() {
			<-start
			_, err := s.Accept(context.Background(), accounts[i].ID.String(), tokens[i])
			results <- err
		}()
	}
	close(start)
	joined := 0
	for range 2 {
		err := <-results
		if err == nil {
			joined++
			continue
		}
		requireInvitationReason(t, err, "NO_SEAT_AVAILABLE")
	}
	require.Equal(t, 1, joined)
	occupied, err := models.CountActiveHumanUsersByOrganization(db, r.Organization.ID.String())
	require.NoError(t, err)
	require.EqualValues(t, 2, occupied)
	pending, err := models.CountPendingOrganizationInvitations(db, r.Organization.ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, pending)
}

func TestInvitationResendTokenHistory(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "no")
	db := database.Conn()
	account, err := models.CreateAccount("Invitee", "invitee@example.com")
	require.NoError(t, err)
	s := InvitationService{AuthService: r.AuthService, BaseURL: "https://example.com"}
	created, err := s.Create(context.Background(), r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: account.Email})
	require.NoError(t, err)
	tokens := []string{strings.TrimPrefix(created.AcceptanceUrl, "https://example.com/invitations/")}
	for range 3 {
		resent, err := s.Resend(context.Background(), r.Organization.ID.String(), r.User.String(), created.Invitation.Id)
		require.NoError(t, err)
		tokens = append(tokens, strings.TrimPrefix(resent.AcceptanceUrl, "https://example.com/invitations/"))
	}
	for _, token := range tokens[:3] {
		_, err := s.Accept(context.Background(), account.ID.String(), token)
		requireInvitationReason(t, err, "INVITATION_REPLACED")
	}
	_, err = s.Accept(context.Background(), account.ID.String(), tokens[3])
	require.NoError(t, err)
	for _, token := range tokens[:3] {
		// History continues to identify replaced links after the invitation is accepted.
		_, err := s.Accept(context.Background(), account.ID.String(), token)
		requireInvitationReason(t, err, "INVITATION_REPLACED")
	}
	var hashes []string
	require.NoError(t, db.Table("organization_invitation_replaced_tokens").Where("invitation_id = ?", created.Invitation.Id).Pluck("token_hash", &hashes).Error)
	require.Len(t, hashes, 3)
	for _, token := range tokens[:3] {
		require.Contains(t, hashes, crypto.HashToken(token))
	}
	require.NoError(t, db.Where("id = ?", created.Invitation.Id).Delete(&models.OrganizationInvitation{}).Error)
	var count int64
	require.NoError(t, db.Table("organization_invitation_replaced_tokens").Where("invitation_id = ?", created.Invitation.Id).Count(&count).Error)
	require.Zero(t, count)
}

func TestInvitationFailedResendKeepsPreviousToken(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "no")
	db := database.Conn()
	account, err := models.CreateAccount("Invitee", "invitee@example.com")
	require.NoError(t, err)
	s := InvitationService{AuthService: r.AuthService, BaseURL: "https://example.com"}
	created, err := s.Create(context.Background(), r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: account.Email})
	require.NoError(t, err)
	archivedBeforeFailure := false
	callback := "test:fail_invitation_rotation"
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register(callback, func(tx *gorm.DB) {
		if _, saving := tx.Statement.Dest.(*models.OrganizationInvitation); saving {
			var count int64
			if err := tx.Session(&gorm.Session{NewDB: true}).Table("organization_invitation_replaced_tokens").Where("invitation_id = ?", created.Invitation.Id).Count(&count).Error; err != nil {
				tx.AddError(err)
				return
			}
			archivedBeforeFailure = count == 1
			tx.AddError(errors.New("invitation write failed"))
		}
	}))
	defer db.Callback().Update().Remove(callback)
	_, err = s.Resend(context.Background(), r.Organization.ID.String(), r.User.String(), created.Invitation.Id)
	require.Equal(t, codes.Internal, grpcerrors.Code(err))
	require.NoError(t, db.Callback().Update().Remove(callback))
	require.True(t, archivedBeforeFailure, "resend must fail after the old hash is archived")
	var count int64
	require.NoError(t, db.Table("organization_invitation_replaced_tokens").Where("invitation_id = ?", created.Invitation.Id).Count(&count).Error)
	require.Zero(t, count)
	token := strings.TrimPrefix(created.AcceptanceUrl, "https://example.com/invitations/")
	_, err = s.Accept(context.Background(), account.ID.String(), token)
	require.NoError(t, err)
}

type invitationOperationContextKey struct{}

func TestAcceptInvitationRechecksHistoryAfterOrganizationLock(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "no")
	db := database.Conn()
	account, err := models.CreateAccount("Invitee", "invitee@example.com")
	require.NoError(t, err)
	s := InvitationService{AuthService: r.AuthService, BaseURL: "https://example.com"}
	created, err := s.Create(context.Background(), r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: account.Email})
	require.NoError(t, err)
	token := strings.TrimPrefix(created.AcceptanceUrl, "https://example.com/invitations/")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	lockHeld := make(chan struct{})
	rotate := make(chan struct{})
	var release sync.Once
	defer release.Do(func() { close(rotate) })
	var paused atomic.Bool
	callback := "test:pause_resend_after_lock"
	require.NoError(t, db.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table != "organizations" || tx.Statement.Context.Value(invitationOperationContextKey{}) != "resend" || !paused.CompareAndSwap(false, true) {
			return
		}
		close(lockHeld)
		select {
		case <-rotate:
		case <-ctx.Done():
			tx.AddError(ctx.Err())
		}
	}))
	defer db.Callback().Query().Remove(callback)
	resent := make(chan error, 1)
	go func() {
		_, err := s.Resend(context.WithValue(ctx, invitationOperationContextKey{}, "resend"), r.Organization.ID.String(), r.User.String(), created.Invitation.Id)
		resent <- err
	}()
	select {
	case <-lockHeld:
	case <-ctx.Done():
		t.Fatal("resend did not acquire the organization lock")
	}
	accepted := make(chan error, 1)
	go func() { _, err := s.Accept(ctx, account.ID.String(), token); accepted <- err }()
	require.Eventually(t, func() bool {
		var waiting int64
		err := db.Raw("SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock' AND query LIKE '%organizations%'").Scan(&waiting).Error
		return err == nil && waiting > 0
	}, 5*time.Second, 10*time.Millisecond, "acceptance must wait for the resend transaction")
	release.Do(func() { close(rotate) })
	require.NoError(t, <-resent)
	requireInvitationReason(t, <-accepted, "INVITATION_REPLACED")
}

func createInvitationAdmin(t *testing.T, r *support.ResourceRegistry, email string) *models.User {
	t.Helper()
	account, err := models.CreateAccount("Second Admin", email)
	require.NoError(t, err)
	admin, err := models.CreateUser(r.Organization.ID, account.ID, account.Email, account.Name)
	require.NoError(t, err)
	require.NoError(t, r.AuthService.AssignRole(admin.ID.String(), models.RoleOrgAdmin, r.Organization.ID.String(), models.DomainTypeOrganization))
	return admin
}

func TestAcceptInvitationRequiresInviterToManageMembers(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "no")
	db := database.Conn()
	orgID := r.Organization.ID.String()
	s := InvitationService{AuthService: r.AuthService, BaseURL: "https://example.com"}
	inviter := createInvitationAdmin(t, r, "inviter@example.com")
	recipient, err := models.CreateAccount("Recipient", "recipient@example.com")
	require.NoError(t, err)
	created, err := s.Create(context.Background(), orgID, inviter.ID.String(), &pb.CreateInvitationRequest{Email: recipient.Email, Role: models.RoleOrgAdmin})
	require.NoError(t, err)
	token := strings.TrimPrefix(created.AcceptanceUrl, "https://example.com/invitations/")
	invitationID := uuid.MustParse(created.Invitation.Id)

	require.NoError(t, r.AuthService.RemoveRole(inviter.ID.String(), models.RoleOrgAdmin, orgID, models.DomainTypeOrganization))
	require.NoError(t, r.AuthService.AssignRole(inviter.ID.String(), models.RoleOrgOperator, orgID, models.DomainTypeOrganization))

	_, err = s.Accept(context.Background(), recipient.ID.String(), token)
	requireInvitationReason(t, err, "INVITATION_INVITER_REVOKED")
	stored, err := models.FindOrganizationInvitation(db, r.Organization.ID, invitationID)
	require.NoError(t, err)
	require.Equal(t, models.InvitationStatePending, stored.State)
	_, err = models.FindActiveUserByEmail(orgID, recipient.Email)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	resent, err := s.Resend(context.Background(), orgID, r.User.String(), created.Invitation.Id)
	require.NoError(t, err)
	require.Equal(t, r.User.String(), resent.Invitation.InvitedBy)
	require.NotEmpty(t, resent.Invitation.InviterName)
	require.NotEqual(t, inviter.Name, resent.Invitation.InviterName)
	token = strings.TrimPrefix(resent.AcceptanceUrl, "https://example.com/invitations/")

	_, err = s.Accept(context.Background(), recipient.ID.String(), token)
	require.NoError(t, err)
	member, err := models.FindActiveUserByEmail(orgID, recipient.Email)
	require.NoError(t, err)
	roles, err := r.AuthService.GetUserRolesForOrg(context.Background(), member.ID.String(), orgID)
	require.NoError(t, err)
	require.Contains(t, invitationRoleNames(roles), models.RoleOrgAdmin)
}
