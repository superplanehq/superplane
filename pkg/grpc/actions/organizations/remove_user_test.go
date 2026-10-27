package organizations

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/ee/billing/entitlements"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	polartest "github.com/superplanehq/superplane/ee/billing/testsupport/polar"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/services"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

func Test_RemoveUser(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()

	t.Run("user not found -> error", func(t *testing.T) {
		_, err := RemoveUser(ctx, r.AuthService, orgID, uuid.NewString())
		require.Error(t, err)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
		assert.Equal(t, "user not found", msg)
	})

	t.Run("user found -> removes user from organization", func(t *testing.T) {
		db := database.DB(ctx)
		newUser := support.CreateUser(t, r, r.Organization.ID)
		plainToken, err := crypto.Base64String(64)
		require.NoError(t, err)
		require.NoError(t, newUser.UpdateTokenHash(crypto.HashToken(plainToken)))
		var personalTokens []*models.UserAPIToken
		for _, name := range []string{"Deploy", "Local development"} {
			token := models.NewUserAPIToken(newUser.ID, name, crypto.HashToken(uuid.NewString()))
			require.NoError(t, models.CreateUserAPIToken(db, token))
			personalTokens = append(personalTokens, token)
		}
		otherOrg, err := models.CreateOrganizationInTransaction(db, "Other organization", "")
		require.NoError(t, err)
		otherMember, err := models.CreateUserInTransaction(db, otherOrg.ID, *newUser.AccountID, newUser.GetEmail(), newUser.Name)
		require.NoError(t, err)
		otherOrgToken := models.NewUserAPIToken(otherMember.ID, "Other organization", crypto.HashToken(uuid.NewString()))
		require.NoError(t, models.CreateUserAPIToken(db, otherOrgToken))

		//
		// Remove the user from the organization
		//
		_, err = RemoveUser(ctx, r.AuthService, orgID, newUser.ID.String())
		require.NoError(t, err)

		//
		// Verify the user is soft deleted, and no longer active.
		//
		user, err := models.FindMaybeDeletedUserByID(orgID, newUser.ID.String())
		require.NoError(t, err)
		require.NotNil(t, user.DeletedAt)
		_, err = models.FindActiveUserByID(orgID, newUser.ID.String())
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		_, err = models.FindActiveUserByEmail(orgID, newUser.GetEmail())
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		_, err = models.FindActiveUserByTokenHash(newUser.TokenHash)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		require.Empty(t, user.TokenHash)
		for _, token := range personalTokens {
			_, err = models.FindUserAPITokenByHash(db, token.TokenHash)
			require.ErrorIs(t, err, gorm.ErrRecordNotFound, "removed members must not retain personal tokens that can revive after they rejoin")
		}
		retainedToken, err := models.FindUserAPITokenByHash(db, otherOrgToken.TokenHash)
		require.NoError(t, err)
		require.Equal(t, otherOrgToken.ID, retainedToken.ID)

		//
		// Verify no organization roles exist anymore for that user anymore
		//
		roles, err := r.AuthService.GetUserRolesForOrg(context.Background(), newUser.ID.String(), orgID)
		require.NoError(t, err)
		require.Len(t, roles, 0)
	})

	t.Run("cannot remove last owner", func(t *testing.T) {
		_, err := RemoveUser(ctx, r.AuthService, orgID, r.User.String())
		require.Error(t, err)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, "cannot remove the last organization owner", msg)
		owner, err := models.FindActiveUserByID(orgID, r.User.String())
		require.NoError(t, err)
		require.True(t, owner.IsOwner)
	})
}

func TestRemoveUserReusesPurchasedSeatWithoutPolar(t *testing.T) {
	for _, mode := range []string{"HTTP 500", "stalled response"} {
		t.Run(mode, func(t *testing.T) {
			r := support.Setup(t)
			t.Cleanup(r.Close)
			t.Setenv(billingmodels.BillingEnabledEnv, "yes")
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			t.Cleanup(cancel)
			ctx = authentication.SetUserIdInMetadata(ctx, r.User.String())
			db := database.DB(ctx)
			setPurchasedSeatPlan(t, db, r.Organization.ID, seatQuantity(4))
			periodEnd := time.Now().Add(30 * 24 * time.Hour)
			require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Updates(map[string]any{
				"polar_subscription_id": "sub_" + uuid.NewString(),
				"current_period_end":    periodEnd,
			}).Error)
			removed := support.CreateUser(t, r, r.Organization.ID)
			for i := 0; i < 2; i++ {
				support.CreateUser(t, r, r.Organization.ID)
			}
			replacement, err := models.CreateAccount("Replacement member", uuid.NewString()+"@example.com")
			require.NoError(t, err)
			orderID := uuid.NewString()
			_, err = billingmodels.AddPolarLLMCreditGrant(db, r.Organization.ID, billingmodels.CentsToMicros(2500), orderID)
			require.NoError(t, err)
			_, err = billingmodels.AddPolarLLMCreditRefund(db, r.Organization.ID, billingmodels.CentsToMicros(500), orderID, uuid.NewString())
			require.NoError(t, err)
			billing, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
			require.NoError(t, err)
			require.NotNil(t, billing)
			credits, err := billingmodels.ListOrganizationLLMCreditGrants(db, r.Organization.ID)
			require.NoError(t, err)

			// Polar is down. Removal and acceptance never call it, so seat
			// reuse neither waits for Polar nor changes the subscription.
			var seatUpdates atomic.Int32
			release := make(chan struct{})
			polar := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet {
					seatUpdates.Add(1)
				}
				if mode == "stalled response" {
					select {
					case <-req.Context().Done():
					case <-release:
					}
					return
				}
				w.WriteHeader(http.StatusInternalServerError)
			}))
			t.Cleanup(func() {
				close(release)
				polar.Close()
				assert.Zero(t, seatUpdates.Load(), "seat reuse must not change the Polar subscription")
			})
			polartest.UseServer(t, polar)

			assertState := func(stage string, occupied, available, pending, invitable int) {
				t.Helper()
				seats, err := entitlements.ResolveSeats(db, r.Organization.ID)
				require.NoError(t, err, stage)
				assert.Equal(t, seatQuantity(4), seats.Purchased, stage)
				assert.Equal(t, occupied, seats.Occupied, stage)
				assert.Equal(t, seatQuantity(available), seats.Available, stage)
				assert.Equal(t, pending, seats.PendingInvitations, stage)
				assert.Equal(t, seatQuantity(invitable), seats.Invitable, stage)
				saved, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
				require.NoError(t, err, stage)
				require.NotNil(t, saved, stage)
				assert.Equal(t, billing.PurchasedSeats, saved.PurchasedSeats, stage)
				assert.Equal(t, billing.CurrentPeriodEnd, saved.CurrentPeriodEnd, stage)
				savedCredits, err := billingmodels.ListOrganizationLLMCreditGrants(db, r.Organization.ID)
				require.NoError(t, err, stage)
				assert.Equal(t, credits, savedCredits, "credit and refund records must not change: %s", stage)
				assert.Zero(t, seatUpdates.Load(), stage)
			}

			assertState("before removal", 4, 0, 0, 0)
			_, err = RemoveUser(ctx, r.AuthService, r.Organization.ID.String(), removed.ID.String())
			require.NoError(t, err)
			assertState("after removal", 3, 1, 0, 1)

			invitations := InvitationService{AuthService: r.AuthService, EmailService: services.NewNoopEmailService(), BaseURL: "https://example.com"}
			invitation, err := invitations.Create(ctx, r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: replacement.Email})
			require.NoError(t, err)
			assertState("after invitation", 3, 1, 1, 0)

			token := strings.TrimPrefix(invitation.AcceptanceUrl, "https://example.com/invitations/")
			_, err = invitations.Accept(ctx, replacement.ID.String(), token)
			require.NoError(t, err)
			assertState("after acceptance", 4, 0, 0, 0)
			member, err := models.FindActiveUserByEmail(r.Organization.ID.String(), replacement.Email)
			require.NoError(t, err)
			require.NotEqual(t, removed.ID, member.ID)
			require.Equal(t, &replacement.ID, member.AccountID)
		})
	}
}

func TestRemoveUserHandlesPendingInvitationsOfInviter(t *testing.T) {
	for _, tc := range []struct {
		name        string
		role        string
		wantState   string
		wantAcceptE string
	}{
		{name: "admin invitation is canceled", role: models.RoleOrgAdmin, wantState: models.InvitationStateCanceled, wantAcceptE: "INVITATION_CANCELED"},
		{name: "operator invitation stays valid", role: models.RoleOrgOperator, wantState: models.InvitationStatePending},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			t.Setenv(billingmodels.BillingEnabledEnv, "no")
			orgID := r.Organization.ID.String()
			s := InvitationService{AuthService: r.AuthService, BaseURL: "https://example.com"}
			inviter := createInvitationAdmin(t, r, "inviter@example.com")
			recipient, err := models.CreateAccount("Recipient", "recipient@example.com")
			require.NoError(t, err)
			created, err := s.Create(context.Background(), orgID, inviter.ID.String(), &pb.CreateInvitationRequest{Email: recipient.Email, Role: tc.role})
			require.NoError(t, err)
			token := strings.TrimPrefix(created.AcceptanceUrl, "https://example.com/invitations/")

			_, err = RemoveUser(context.Background(), r.AuthService, orgID, inviter.ID.String())
			require.NoError(t, err)

			stored, err := models.FindOrganizationInvitation(database.Conn(), r.Organization.ID, uuid.MustParse(created.Invitation.Id))
			require.NoError(t, err)
			require.Equal(t, tc.wantState, stored.State)

			_, err = s.Accept(context.Background(), recipient.ID.String(), token)
			if tc.wantAcceptE != "" {
				requireInvitationReason(t, err, tc.wantAcceptE)
				_, err = models.FindActiveUserByEmail(orgID, recipient.Email)
				require.ErrorIs(t, err, gorm.ErrRecordNotFound)
				return
			}
			require.NoError(t, err)
			member, err := models.FindActiveUserByEmail(orgID, recipient.Email)
			require.NoError(t, err)
			roles, err := r.AuthService.GetUserRolesForOrg(context.Background(), member.ID.String(), orgID)
			require.NoError(t, err)
			require.Equal(t, []string{models.RoleOrgOperator}, invitationRoleNames(roles))
		})
	}
}

func TestRemoveGroupAdminThenReinviteAsOperatorGrantsOnlyOperator(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "no")
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	account, err := models.CreateAccount("Group admin", "group-admin@example.com")
	require.NoError(t, err)
	member, err := models.CreateUser(r.Organization.ID, account.ID, account.Email, account.Name)
	require.NoError(t, err)
	require.NoError(t, r.AuthService.AssignRole(member.ID.String(), models.RoleOrgOperator, orgID, models.DomainTypeOrganization))
	otherMember := addToAdminGroup(t, r, "admins", member.ID.String())
	require.True(t, canManageOrganization(t, r, member.ID.String()))

	_, err = RemoveUser(ctx, r.AuthService, orgID, member.ID.String())
	require.NoError(t, err)

	s := InvitationService{AuthService: r.AuthService, EmailService: services.NewNoopEmailService(), BaseURL: "https://example.com"}
	created, err := s.Create(ctx, orgID, r.User.String(), &pb.CreateInvitationRequest{Email: account.Email, Role: models.RoleOrgOperator})
	require.NoError(t, err)
	_, err = s.Accept(ctx, account.ID.String(), strings.TrimPrefix(created.AcceptanceUrl, "https://example.com/invitations/"))
	require.NoError(t, err)

	restored, err := models.FindActiveUserByEmail(orgID, account.Email)
	require.NoError(t, err)
	require.Equal(t, member.ID, restored.ID)
	roles, err := r.AuthService.GetUserRolesForOrg(context.Background(), restored.ID.String(), orgID)
	require.NoError(t, err)
	require.Equal(t, []string{models.RoleOrgOperator}, invitationRoleNames(roles))
	assert.False(t, canManageOrganization(t, r, restored.ID.String()), "a reinvited Operator must not keep Admin authority from a retained group membership")
	users, err := r.AuthService.GetGroupUsers(context.Background(), orgID, models.DomainTypeOrganization, "admins")
	require.NoError(t, err)
	require.Equal(t, []string{otherMember}, users)
	groupRole, err := r.AuthService.GetGroupRole(context.Background(), orgID, models.DomainTypeOrganization, "admins")
	require.NoError(t, err)
	require.Equal(t, models.RoleOrgAdmin, groupRole)
}

// addToAdminGroup creates an Admin group with the user and one other member.
// It returns the ID of the other member.
func addToAdminGroup(t *testing.T, r *support.ResourceRegistry, group, userID string) string {
	t.Helper()
	orgID := r.Organization.ID.String()
	otherMember := uuid.NewString()
	require.NoError(t, r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, group, models.RoleOrgAdmin, "Admins", "Admins"))
	require.NoError(t, r.AuthService.AddUserToGroup(orgID, models.DomainTypeOrganization, userID, group))
	require.NoError(t, r.AuthService.AddUserToGroup(orgID, models.DomainTypeOrganization, otherMember, group))
	return otherMember
}

// canManageOrganization reports whether the user can update organization
// settings, which include billing, or manage members.
func canManageOrganization(t *testing.T, r *support.ResourceRegistry, userID string) bool {
	t.Helper()
	orgID := r.Organization.ID.String()
	updateOrg, err := r.AuthService.CheckOrganizationPermission(context.Background(), userID, orgID, "org", "update")
	require.NoError(t, err)
	updateMembers, err := r.AuthService.CheckOrganizationPermission(context.Background(), userID, orgID, "members", "update")
	require.NoError(t, err)
	return updateOrg || updateMembers
}

type failingGrantRemovalAuthorization struct {
	authorization.Authorization
}

func (a failingGrantRemovalAuthorization) RemoveUserFromOrganizationInTransaction(tx *gorm.DB, userID, orgID string) error {
	if err := a.Authorization.RemoveUserFromOrganizationInTransaction(tx, userID, orgID); err != nil {
		return err
	}
	return errors.New("grant removal interrupted")
}

func TestRemoveUserRollsBackRolesMembershipAndInvitations(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(billingmodels.BillingEnabledEnv, "no")
	orgID := r.Organization.ID.String()
	s := InvitationService{AuthService: r.AuthService, BaseURL: "https://example.com"}
	member := createInvitationAdmin(t, r, "member@example.com")
	otherMember := addToAdminGroup(t, r, "admins", member.ID.String())
	recipient, err := models.CreateAccount("Recipient", "recipient@example.com")
	require.NoError(t, err)
	created, err := s.Create(context.Background(), orgID, member.ID.String(), &pb.CreateInvitationRequest{Email: recipient.Email, Role: models.RoleOrgAdmin})
	require.NoError(t, err)
	rolesBefore, err := r.AuthService.GetUserRolesForOrg(context.Background(), member.ID.String(), orgID)
	require.NoError(t, err)
	require.NotEmpty(t, rolesBefore)

	_, err = RemoveUser(context.Background(), failingGrantRemovalAuthorization{r.AuthService}, orgID, member.ID.String())

	require.Equal(t, codes.Internal, grpcerrors.Code(err))
	_, err = models.FindActiveUserByID(orgID, member.ID.String())
	require.NoError(t, err)
	rolesAfter, err := r.AuthService.GetUserRolesForOrg(context.Background(), member.ID.String(), orgID)
	require.NoError(t, err)
	require.ElementsMatch(t, invitationRoleNames(rolesBefore), invitationRoleNames(rolesAfter))
	directAdmins, err := r.AuthService.GetOrgUsersForRole(context.Background(), models.RoleOrgAdmin, orgID)
	require.NoError(t, err)
	require.Contains(t, directAdmins, member.ID.String())
	groups, err := r.AuthService.GetUserGroups(context.Background(), orgID, models.DomainTypeOrganization, member.ID.String())
	require.NoError(t, err)
	require.Equal(t, []string{"admins"}, groups)
	users, err := r.AuthService.GetGroupUsers(context.Background(), orgID, models.DomainTypeOrganization, "admins")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{member.ID.String(), otherMember}, users)
	stored, err := models.FindOrganizationInvitation(database.Conn(), r.Organization.ID, uuid.MustParse(created.Invitation.Id))
	require.NoError(t, err)
	require.Equal(t, models.InvitationStatePending, stored.State)
}
