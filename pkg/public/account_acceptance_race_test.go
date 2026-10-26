package public

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/ee/billing/entitlements"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/organizations"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/services"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// acceptanceRace is an invitation for a new account that the account
// deletion can race.
type acceptanceRace struct {
	t        *testing.T
	r        *support.ResourceRegistry
	server   *Server
	invitee  *models.Account
	token    string
	occupied int
}

func newAcceptanceRace(t *testing.T) *acceptanceRace {
	t.Helper()
	r := support.Setup(t)
	t.Cleanup(r.Close)
	server, _, _ := setupTestServer(r, t)
	invitee, err := models.CreateAccount("Invitee", uuid.NewString()+"@example.com")
	require.NoError(t, err)
	created, err := (&organizations.InvitationService{AuthService: r.AuthService, EmailService: services.NewNoopEmailService(), BaseURL: "https://example.com"}).
		Create(context.Background(), r.Organization.ID.String(), r.User.String(), &pb.CreateInvitationRequest{Email: invitee.Email})
	require.NoError(t, err)
	race := &acceptanceRace{t: t, r: r, server: server, invitee: invitee, token: strings.TrimPrefix(created.AcceptanceUrl, "https://example.com/invitations/")}
	race.occupied = race.seats().Occupied
	return race
}

func (a *acceptanceRace) accept() error {
	service := &organizations.InvitationService{AuthService: a.r.AuthService, EmailService: services.NewNoopEmailService(), BaseURL: "https://example.com"}
	_, err := service.Accept(context.Background(), a.invitee.ID.String(), a.token)
	return err
}

func (a *acceptanceRace) deleteAccount() error {
	account := *a.invitee
	_, err := a.server.softDeleteAccount(context.Background(), &account)
	return err
}

func (a *acceptanceRace) seats() *entitlements.SeatUsage {
	a.t.Helper()
	seats, err := entitlements.ResolveSeats(database.Conn(), a.r.Organization.ID)
	require.NoError(a.t, err)
	return seats
}

// holdRow locks one row in a transaction and returns its release.
func (a *acceptanceRace) holdRow(model any, id uuid.UUID) func() {
	a.t.Helper()
	tx := database.Conn().Begin()
	require.NoError(a.t, tx.Error)
	require.NoError(a.t, tx.Clauses(clause.Locking{Strength: "UPDATE"}).Unscoped().Where("id = ?", id).First(model).Error)
	released := false
	release := func() {
		if !released {
			released = true
			require.NoError(a.t, tx.Commit().Error)
		}
	}
	a.t.Cleanup(release)
	return release
}

// waitForLockWaiters waits until count other sessions wait for a row lock.
func (a *acceptanceRace) waitForLockWaiters(count int) {
	a.t.Helper()
	require.Eventually(a.t, func() bool {
		var waiting int
		require.NoError(a.t, database.Conn().Raw("SELECT count(*) FROM pg_stat_activity WHERE wait_event_type = 'Lock' AND datname = current_database()").Scan(&waiting).Error)
		return waiting >= count
	}, 10*time.Second, 10*time.Millisecond)
}

func run(f func() error) <-chan error {
	done := make(chan error, 1)
	go func() { done <- f() }()
	return done
}

func finished(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(15 * time.Second):
		t.Fatal("the race did not finish; a lock order can deadlock")
		return nil
	}
}

// assertNoMembership fails when the deleted account keeps a membership,
// an organization role, or an occupied seat.
func (a *acceptanceRace) assertNoMembership() {
	a.t.Helper()
	_, err := models.FindActiveHumanUserByAccountAndOrganization(database.Conn(), a.r.Organization.ID, a.invitee.ID)
	require.ErrorIs(a.t, err, gorm.ErrRecordNotFound, "a deleted account keeps no active membership")
	var users []models.User
	require.NoError(a.t, database.Conn().Unscoped().Where("organization_id = ? AND account_id = ?", a.r.Organization.ID, a.invitee.ID).Find(&users).Error)
	for _, user := range users {
		roles, err := a.r.AuthService.GetUserRolesForOrg(context.Background(), user.ID.String(), a.r.Organization.ID.String())
		require.NoError(a.t, err)
		assert.Empty(a.t, roles, "a deleted account keeps no organization role")
	}
	assert.Equal(a.t, a.occupied, a.seats().Occupied, "a deleted account occupies no seat")
}

func TestAcceptanceAndAccountDeletionRace_DeletionWins(t *testing.T) {
	a := newAcceptanceRace(t)
	release := a.holdRow(&models.Account{}, a.invitee.ID)
	deletion := run(a.deleteAccount)
	a.waitForLockWaiters(1)
	// The acceptance reads the account before the deletion commits.
	acceptance := run(a.accept)
	a.waitForLockWaiters(2)

	release()

	require.NoError(t, finished(t, deletion))
	err := finished(t, acceptance)
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, grpcerrors.Code(err))
	a.assertNoMembership()
	var users []models.User
	require.NoError(t, database.Conn().Unscoped().Where("organization_id = ? AND account_id = ?", a.r.Organization.ID, a.invitee.ID).Find(&users).Error)
	assert.Empty(t, users, "the losing acceptance creates no membership")
	pending, err := models.CountPendingOrganizationInvitations(database.Conn(), a.r.Organization.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, pending, "the invitation stays pending")
}

func TestAcceptanceAndAccountDeletionRace_AcceptanceWins(t *testing.T) {
	a := newAcceptanceRace(t)
	// The acceptance takes the account lock and waits for the organization.
	release := a.holdRow(&models.Organization{}, a.r.Organization.ID)
	acceptance := run(a.accept)
	a.waitForLockWaiters(1)
	deletion := run(a.deleteAccount)
	a.waitForLockWaiters(2)

	release()

	require.NoError(t, finished(t, acceptance))
	require.NoError(t, finished(t, deletion), "the deletion removes the committed membership")
	a.assertNoMembership()
}

func TestMemberRemovalAndAccountDeletionRace_RemovalWins(t *testing.T) {
	a := newAcceptanceRace(t)
	require.NoError(t, a.accept())
	user, err := models.FindActiveHumanUserByAccountAndOrganization(database.Conn(), a.r.Organization.ID, a.invitee.ID)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	paused := make(chan struct{})
	resume := make(chan struct{})
	var pauseOnce sync.Once
	var resumeOnce sync.Once
	release := func() { resumeOnce.Do(func() { close(resume) }) }
	defer release()
	callback := "pause_member_removal_" + uuid.NewString()
	queries := database.Conn().Callback().Query()
	// Pause the real RemoveUser request after it locks the organization,
	// before it reads the user or removes grants. Deletion must wait without
	// holding grants that RemoveUser still needs.
	require.NoError(t, queries.Before("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Context != ctx || tx.Statement.Table != "users" {
			return
		}
		pauseOnce.Do(func() {
			close(paused)
			select {
			case <-resume:
			case <-ctx.Done():
			}
		})
	}))
	defer func() { require.NoError(t, queries.Remove(callback)) }()
	removal := run(func() error {
		_, err := organizations.RemoveUser(ctx, a.r.AuthService, a.r.Organization.ID.String(), user.ID.String())
		return err
	})
	select {
	case <-paused:
	case <-ctx.Done():
		t.Fatal("member removal did not reach the locked organization")
	}
	deletion := run(a.deleteAccount)
	a.waitForLockWaiters(1)
	release()

	require.NoError(t, finished(t, removal))
	require.NoError(t, finished(t, deletion), "both requests must finish without a deadlock abort")
	a.assertNoMembership()
}
