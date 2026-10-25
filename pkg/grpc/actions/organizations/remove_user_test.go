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
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/billing/seatrenewal"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/entitlements"
	"github.com/superplanehq/superplane/pkg/features"
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
			t.Setenv(models.BillingEnabledEnv, "yes")
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			t.Cleanup(cancel)
			ctx = authentication.SetUserIdInMetadata(ctx, r.User.String())
			db := database.DB(ctx)
			require.NoError(t, models.EnableExperimentalFeatureInTransaction(db, r.Organization.ID, features.FeatureNewBilling))
			setPurchasedSeatPlan(t, db, r.Organization.ID, seatQuantity(4))
			periodEnd := time.Now().Add(30 * 24 * time.Hour)
			require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Updates(map[string]any{
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
			_, err = models.AddPolarLLMCreditGrant(db, r.Organization.ID, models.CentsToMicros(2500), orderID)
			require.NoError(t, err)
			_, err = models.AddPolarLLMCreditRefund(db, r.Organization.ID, models.CentsToMicros(500), orderID, uuid.NewString())
			require.NoError(t, err)
			billing, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
			require.NoError(t, err)
			require.NotNil(t, billing)
			credits, err := models.ListOrganizationLLMCreditGrants(db, r.Organization.ID)
			require.NoError(t, err)

			// Removal and acceptance schedule the renewal seat count in the
			// background. Polar is down, so that schedule fails and changes
			// nothing. Seat reuse must not wait for it.
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
				seatrenewal.Wait()
				polar.Close()
				assert.Zero(t, seatUpdates.Load(), "seat reuse must not change the Polar subscription")
			})
			usePolarTestServer(t, polar)

			assertState := func(stage string, occupied, available, pending, invitable int) {
				t.Helper()
				seats, err := entitlements.ResolveSeats(db, r.Organization.ID)
				require.NoError(t, err, stage)
				assert.Equal(t, seatQuantity(4), seats.Purchased, stage)
				assert.Equal(t, occupied, seats.Occupied, stage)
				assert.Equal(t, seatQuantity(available), seats.Available, stage)
				assert.Equal(t, pending, seats.PendingInvitations, stage)
				assert.Equal(t, seatQuantity(invitable), seats.Invitable, stage)
				saved, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
				require.NoError(t, err, stage)
				require.NotNil(t, saved, stage)
				assert.Equal(t, billing.PurchasedSeats, saved.PurchasedSeats, stage)
				assert.Equal(t, billing.CurrentPeriodEnd, saved.CurrentPeriodEnd, stage)
				savedCredits, err := models.ListOrganizationLLMCreditGrants(db, r.Organization.ID)
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
			t.Setenv(models.BillingEnabledEnv, "no")
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

type failingRoleRemovalAuthorization struct {
	authorization.Authorization
}

func (a failingRoleRemovalAuthorization) RemoveRoleInTransaction(tx *gorm.DB, userID, role, domainID, domainType string) error {
	if err := a.Authorization.RemoveRoleInTransaction(tx, userID, role, domainID, domainType); err != nil {
		return err
	}
	return errors.New("role removal interrupted")
}

func TestRemoveUserRollsBackRolesMembershipAndInvitations(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(models.BillingEnabledEnv, "no")
	orgID := r.Organization.ID.String()
	s := InvitationService{AuthService: r.AuthService, BaseURL: "https://example.com"}
	member := createInvitationAdmin(t, r, "member@example.com")
	recipient, err := models.CreateAccount("Recipient", "recipient@example.com")
	require.NoError(t, err)
	created, err := s.Create(context.Background(), orgID, member.ID.String(), &pb.CreateInvitationRequest{Email: recipient.Email, Role: models.RoleOrgAdmin})
	require.NoError(t, err)
	rolesBefore, err := r.AuthService.GetUserRolesForOrg(context.Background(), member.ID.String(), orgID)
	require.NoError(t, err)
	require.NotEmpty(t, rolesBefore)

	_, err = RemoveUser(context.Background(), failingRoleRemovalAuthorization{r.AuthService}, orgID, member.ID.String())

	require.Equal(t, codes.Internal, grpcerrors.Code(err))
	_, err = models.FindActiveUserByID(orgID, member.ID.String())
	require.NoError(t, err)
	rolesAfter, err := r.AuthService.GetUserRolesForOrg(context.Background(), member.ID.String(), orgID)
	require.NoError(t, err)
	require.ElementsMatch(t, invitationRoleNames(rolesBefore), invitationRoleNames(rolesAfter))
	stored, err := models.FindOrganizationInvitation(database.Conn(), r.Organization.ID, uuid.MustParse(created.Invitation.Id))
	require.NoError(t, err)
	require.Equal(t, models.InvitationStatePending, stored.State)
}
