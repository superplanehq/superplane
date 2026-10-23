package organizations

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

func TestAcceptInvitationMembershipLifecycle(t *testing.T) {
	for _, restore := range []bool{false, true} {
		t.Run(map[bool]string{false: "new member", true: "removed member"}[restore], func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			t.Setenv(models.BillingEnabledEnv, "yes")
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
			require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
		})
	}
}

func TestAcceptInvitationRejectsInvalidRecipientOrState(t *testing.T) {
	for _, scenario := range []string{"wrong email", "expired", "canceled", "rotated", "capacity removed", "unknown token"} {
		t.Run(scenario, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			t.Setenv(models.BillingEnabledEnv, "yes")
			db := database.Conn()
			setPurchasedSeatPlan(t, db, r.Organization.ID, seatQuantity(2))
			account, err := models.CreateAccount("Invitee", "invitee@example.com")
			require.NoError(t, err)
			s := InvitationService{AuthService: r.AuthService, BaseURL: "https://example.com"}
			created, err := s.Create(context.Background(), r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: account.Email})
			require.NoError(t, err)
			token := strings.TrimPrefix(created.AcceptanceUrl, "https://example.com/invitations/")
			expected := codes.FailedPrecondition
			switch scenario {
			case "wrong email":
				account, err = models.CreateAccount("Other person", "other@example.com")
				require.NoError(t, err)
				expected = codes.PermissionDenied
			case "expired":
				require.NoError(t, db.Model(&models.OrganizationInvitation{}).Where("id = ?", created.Invitation.Id).Update("expires_at", time.Now().Add(-time.Hour)).Error)
			case "canceled":
				_, err := s.Cancel(context.Background(), r.Organization.ID.String(), created.Invitation.Id)
				require.NoError(t, err)
				expected = codes.NotFound
			case "rotated":
				_, err := s.Resend(context.Background(), r.Organization.ID.String(), r.User.String(), created.Invitation.Id)
				require.NoError(t, err)
				expected = codes.NotFound
			case "capacity removed":
				require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("purchased_seats", 1).Error)
			case "unknown token":
				token = "unknown"
				expected = codes.NotFound
			}
			_, err = s.Accept(context.Background(), account.ID.String(), token)
			require.Equal(t, expected, grpcerrors.Code(err))
			occupied, err := models.CountActiveHumanUsersByOrganization(db, r.Organization.ID.String())
			require.NoError(t, err)
			require.EqualValues(t, 1, occupied)
			stored, err := models.FindOrganizationInvitation(db, r.Organization.ID, uuid.MustParse(created.Invitation.Id))
			require.NoError(t, err)
			require.NotEqual(t, models.InvitationStateAccepted, stored.State)
		})
	}
}

func TestAcceptInvitationConcurrentRetries(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(models.BillingEnabledEnv, "yes")
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
	t.Setenv(models.BillingEnabledEnv, "no")
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
	t.Setenv(models.BillingEnabledEnv, "no")
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
	t.Setenv(models.BillingEnabledEnv, "yes")
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
	require.NoError(t, db.Model(&models.OrganizationBillingPlan{}).Where("organization_id = ?", r.Organization.ID).Update("purchased_seats", 2).Error)
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
		require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
	}
	require.Equal(t, 1, joined)
	occupied, err := models.CountActiveHumanUsersByOrganization(db, r.Organization.ID.String())
	require.NoError(t, err)
	require.EqualValues(t, 2, occupied)
	pending, err := models.CountPendingOrganizationInvitations(db, r.Organization.ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, pending)
}
