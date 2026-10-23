package organizations

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

func TestAcceptInviteSeatLimits(t *testing.T) {
	for _, tc := range []struct {
		name                                        string
		limit                                       int
		members                                     int
		restored, existing, apiKey, billingDisabled bool
		plan                                        string
		expired                                     bool
		rejected                                    bool
	}{
		{name: "available", limit: 2},
		{name: "finite plan available", limit: 2, plan: "finite"},
		{name: "finite plan full", limit: 1, plan: "finite", rejected: true},
		{name: "full", limit: 1, rejected: true},
		{name: "exceeded", limit: 1, members: 1, rejected: true},
		{name: "restore available", limit: 2, restored: true},
		{name: "restore full", limit: 1, restored: true, rejected: true},
		{name: "existing member", limit: 1, existing: true},
		{name: "API keys excluded", limit: 2, apiKey: true},
		{name: "none plan", plan: "none"},
		{name: "trial plan", plan: "trial"},
		{name: "expired trial", plan: "trial", expired: true},
		{name: "billing disabled", limit: 1, billingDisabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			t.Setenv(models.BillingEnabledEnv, "yes")
			if tc.billingDisabled {
				t.Setenv(models.BillingEnabledEnv, "no")
			}
			db := database.Conn()

			t.Setenv("POLAR_ACCESS_TOKEN", "")
			if tc.plan == "finite" {
				plan := models.BillingPlan{Key: "seat_" + uuid.NewString()[:8], Name: "Seats", Category: "paid", Visibility: "private", SeatLimit: &tc.limit, CreditFrequency: "none", CreditExpiration: "never"}
				require.NoError(t, models.SaveBillingPlan(db, &plan))
				_, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, plan.Key, nil)
				require.NoError(t, err)
			} else if tc.plan != "" {
				var trialEnd *time.Time
				if tc.plan == "trial" {
					end := time.Now().Add(time.Hour)
					if tc.expired {
						end = time.Now().Add(-time.Hour)
					}
					trialEnd = &end
				}
				_, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, tc.plan, trialEnd)
				require.NoError(t, err)
			}
			if tc.limit > 0 && tc.plan != "finite" {
				require.NoError(t, models.SetOrganizationSeatLimitOverride(db, r.Organization.ID, &tc.limit))
			}
			for i := 0; i < tc.members; i++ {
				support.CreateUser(t, r, r.Organization.ID)
			}
			if tc.apiKey {
				_, err := models.CreateAPIKey(db, r.Organization.ID, "bot", nil, r.User, nil, nil)
				require.NoError(t, err)
			}
			account, err := models.CreateAccount("Invitee", uuid.NewString()+"@example.com")
			require.NoError(t, err)
			if tc.restored || tc.existing {
				user, err := models.CreateUser(r.Organization.ID, account.ID, account.Email, account.Name)
				require.NoError(t, err)
				if tc.restored {
					require.NoError(t, db.Delete(user).Error)
				}
			}
			link, err := models.FindOrCreateInviteLink(db, r.Organization.ID)
			require.NoError(t, err)
			resp, err := AcceptInviteLink(context.Background(), r.AuthService, account.ID.String(), link.Token.String())
			if tc.rejected {
				code, message, ok := grpcerrors.HandlerStatus(err)
				require.True(t, ok)
				require.Equal(t, codes.FailedPrecondition, code)
				require.Equal(t, "This organization has reached its seat limit. Contact an organization admin.", message)
				_, err = models.FindActiveUserByEmail(r.Organization.ID.String(), account.Email)
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			expected := "joined"
			if tc.existing {
				expected = "already_member"
			}
			require.Equal(t, expected, resp.Fields["status"].GetStringValue())
			user, err := models.FindActiveUserByEmail(r.Organization.ID.String(), account.Email)
			require.NoError(t, err)
			require.NotNil(t, user)
			if !tc.existing {
				roles, err := r.AuthService.GetUserRolesForOrg(context.Background(), user.ID.String(), r.Organization.ID.String())
				require.NoError(t, err)
				require.True(t, slices.ContainsFunc(roles, func(role *authorization.RoleDefinition) bool { return role.Name == models.RoleOrgOperator }))
			}
		})
	}
}

func TestAcceptInviteFinalSeatConcurrent(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(models.BillingEnabledEnv, "yes")
	db := database.Conn()
	limit := 2
	require.NoError(t, models.SetOrganizationSeatLimitOverride(db, r.Organization.ID, &limit))
	link, err := models.FindOrCreateInviteLink(db, r.Organization.ID)
	require.NoError(t, err)
	accounts := []*models.Account{}
	for i := 0; i < 2; i++ {
		a, err := models.CreateAccount("Invitee", uuid.NewString()+"@example.com")
		require.NoError(t, err)
		accounts = append(accounts, a)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, a := range accounts {
		go func(a *models.Account) {
			<-start
			_, err := AcceptInviteLink(context.Background(), r.AuthService, a.ID.String(), link.Token.String())
			results <- err
		}(a)
	}
	close(start)
	joined, rejected := 0, 0
	for i := 0; i < 2; i++ {
		err := <-results
		if err == nil {
			joined++
			continue
		}
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		require.Equal(t, codes.FailedPrecondition, code)
		rejected++
	}
	require.Equal(t, 1, joined)
	require.Equal(t, 1, rejected)
	used, err := models.CountActiveHumanUsersByOrganization(r.Organization.ID.String())
	require.NoError(t, err)
	require.EqualValues(t, 2, used)
}

type pausedInviteAuthorization struct {
	authorization.Authorization
	entered chan struct{}
	release chan struct{}
}

func (a *pausedInviteAuthorization) AssignRole(userID, role, domainID, domainType string) error {
	close(a.entered)
	<-a.release
	return a.Authorization.AssignRole(userID, role, domainID, domainType)
}

func TestSeatLimitReductionRacesWithJoin(t *testing.T) {
	for _, joinFirst := range []bool{false, true} {
		t.Run(fmt.Sprintf("join first %t", joinFirst), func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			t.Setenv(models.BillingEnabledEnv, "yes")
			db := database.Conn()
			limit := 2
			require.NoError(t, models.SetOrganizationSeatLimitOverride(db, r.Organization.ID, &limit))
			account, err := models.CreateAccount("Invitee", uuid.NewString()+"@example.com")
			require.NoError(t, err)
			link, err := models.FindOrCreateInviteLink(db, r.Organization.ID)
			require.NoError(t, err)
			joined := make(chan error, 1)
			reduced := make(chan error, 1)
			join := func(auth authorization.Authorization) {
				_, err := AcceptInviteLink(context.Background(), auth, account.ID.String(), link.Token.String())
				joined <- err
			}
			limit = 1
			if !joinFirst {
				tx := db.Begin()
				require.NoError(t, tx.Error)
				defer tx.Rollback()
				_, err := models.LockOrganization(tx, r.Organization.ID)
				require.NoError(t, err)
				go join(r.AuthService)
				waitForOrganizationLockWait(t, db)
				require.NoError(t, models.SetOrganizationSeatLimitOverride(tx, r.Organization.ID, &limit))
				require.NoError(t, tx.Commit().Error)
				code, _, ok := grpcerrors.HandlerStatus(<-joined)
				require.True(t, ok)
				require.Equal(t, codes.FailedPrecondition, code)
			} else {
				auth := &pausedInviteAuthorization{Authorization: r.AuthService, entered: make(chan struct{}), release: make(chan struct{})}
				go join(auth)
				<-auth.entered
				go func() { reduced <- models.SetOrganizationSeatLimitOverride(db, r.Organization.ID, &limit) }()
				waitForOrganizationLockWait(t, db)
				close(auth.release)
				require.NoError(t, <-joined)
				require.NoError(t, <-reduced)
				_, err = models.FindActiveUserByEmail(r.Organization.ID.String(), account.Email)
				require.NoError(t, err)
			}
			used, err := models.CountActiveHumanUsersByOrganization(r.Organization.ID.String())
			require.NoError(t, err)
			expected := int64(1)
			if joinFirst {
				expected = 2
			}
			require.Equal(t, expected, used)
		})
	}
}

func waitForOrganizationLockWait(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.Raw(`SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock' AND query LIKE '%"organizations"%' AND query LIKE '%FOR UPDATE%')`).Scan(&waiting).Error
		return err == nil && waiting
	}, 5*time.Second, 10*time.Millisecond, "a competing transaction must wait for the organization lock")
}
