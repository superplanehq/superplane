package organizations

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/polar"
	polartest "github.com/superplanehq/superplane/ee/billing/testsupport/polar"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

func Test__DeleteOrganization(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("organization does not exist -> error", func(t *testing.T) {
		_, err := DeleteOrganization(ctx, r.AuthService, uuid.New().String())
		require.Error(t, err)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
		assert.Equal(t, "organization not found", msg)
	})

	t.Run("unauthenticated user -> error", func(t *testing.T) {
		_, err := DeleteOrganization(context.Background(), r.AuthService, r.Organization.ID.String())
		require.Error(t, err)
		assert.ErrorContains(t, err, "user not authenticated")
	})

	t.Run("organization is deleted", func(t *testing.T) {
		response, err := DeleteOrganization(ctx, r.AuthService, r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)

		_, err = models.FindOrganizationByID(r.Organization.ID.String())
		assert.Error(t, err)
	})
}

func TestDeleteOrganizationWaitsForSubscriptionActivation(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, billingmodels.BindLegacyBillingProduct(db, "prod_business"))
	activation := db.Begin()
	require.NoError(t, activation.Error)
	defer activation.Rollback()
	_, err := billingmodels.ApplyPolarSubscription(activation, r.Organization.ID, billingmodels.PolarSubscriptionApply{
		ID: "sub_concurrent_delete", ProductID: "prod_business", Status: billingmodels.PolarSubscriptionStatusActive,
	}, nil)
	require.NoError(t, err)
	var activationPID int
	require.NoError(t, activation.Raw("SELECT pg_backend_pid()").Scan(&activationPID).Error)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	ctx = authentication.SetUserIdInMetadata(ctx, r.User.String())
	deleted := make(chan error, 1)
	go func() {
		_, err := DeleteOrganization(ctx, r.AuthService, r.Organization.ID.String())
		deleted <- err
	}()
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.Raw("SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE ? = ANY(pg_blocking_pids(pid)))", activationPID).Scan(&waiting).Error
		return err == nil && waiting
	}, 5*time.Second, 10*time.Millisecond, "deletion must wait for subscription activation")
	require.NoError(t, activation.Commit().Error)
	requireInvitationReason(t, <-deleted, "ORGANIZATION_SUBSCRIPTION_LIVE")
	organization, err := models.FindOrganizationByID(r.Organization.ID.String())
	require.NoError(t, err)
	assert.False(t, organization.DeletedAt.Valid)
	plan, err := billingmodels.FindOrganizationBillingPlan(db, organization.ID)
	require.NoError(t, err)
	assert.True(t, plan.HasLivePolarSubscription(time.Now()))
}

type pausedOrganizationDeletion struct {
	authorization.Authorization
	ctx     context.Context
	locked  chan int
	release chan struct{}
}

func (a *pausedOrganizationDeletion) DestroyOrganization(tx *gorm.DB, orgID string) error {
	var pid int
	if err := tx.Raw("SELECT pg_backend_pid()").Scan(&pid).Error; err != nil {
		return err
	}
	a.locked <- pid
	select {
	case <-a.release:
		return a.Authorization.DestroyOrganization(tx, orgID)
	case <-a.ctx.Done():
		return a.ctx.Err()
	}
}

func TestSubscriptionActivationWaitsForOrganizationDeletion(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, billingmodels.BindLegacyBillingProduct(db, "prod_business"))
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	auth := &pausedOrganizationDeletion{
		Authorization: r.AuthService, ctx: ctx, locked: make(chan int, 1), release: make(chan struct{}),
	}
	deleted := make(chan error, 1)
	go func() {
		_, err := DeleteOrganization(authentication.SetUserIdInMetadata(ctx, r.User.String()), auth, r.Organization.ID.String())
		deleted <- err
	}()
	var deletionPID int
	select {
	case deletionPID = <-auth.locked:
	case <-ctx.Done():
		t.Fatal("deletion did not take the organization lock")
	}
	activated := make(chan error, 1)
	go func() {
		activated <- polar.ApplySubscription(ctx, db, polar.SubscriptionData{
			ID: "sub_deleted_concurrently", ProductID: "prod_business", Status: billingmodels.PolarSubscriptionStatusActive,
			ExternalCustomerID: r.Organization.ID.String(),
		})
	}()
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.Raw("SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE ? = ANY(pg_blocking_pids(pid)))", deletionPID).Scan(&waiting).Error
		return err == nil && waiting
	}, 5*time.Second, 10*time.Millisecond, "activation must wait for organization deletion")
	close(auth.release)
	require.NoError(t, <-deleted)
	err := <-activated
	require.ErrorIs(t, err, polar.ErrSubscriptionForDeletedOrganization)
	require.ErrorIs(t, err, polar.ErrPermanentApply)
	_, err = models.FindOrganizationByID(r.Organization.ID.String())
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	plan, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	assert.False(t, plan.HasLivePolarSubscription(time.Now()))
}

func Test__DeleteOrganization_TransactionRollback(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("auth service failure rolls back organization soft-deletion", func(t *testing.T) {
		//
		// Verify organization exists and is not soft-deleted
		//
		foundOrg, err := models.FindOrganizationByID(r.Organization.ID.String())
		require.NoError(t, err)
		assert.False(t, foundOrg.DeletedAt.Valid)

		//
		// Use an authentication service that fails
		//
		mockAuth := &mockAuthService{
			Authorization: r.AuthService,
			Error:         errors.New("ooops"),
		}

		//
		// Try to delete organization
		// It should fail due to destroy organization error.
		//
		_, err = DeleteOrganization(ctx, mockAuth, r.Organization.ID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ooops")

		//
		// Verify organization is NOT soft-deleted after transaction rollback
		//
		foundOrg, err = models.FindOrganizationByID(r.Organization.ID.String())
		require.NoError(t, err)
		assert.False(t, foundOrg.DeletedAt.Valid)
	})
}

func Test__DeleteOrganization_LivePolarSubscription(t *testing.T) {
	const message = "Cancel the subscription in Billing before you delete the organization. You can delete it after the paid period ends."

	t.Run("live subscription blocks deletion until it ends", func(t *testing.T) {
		r := support.Setup(t)
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
		orgID := r.Organization.ID
		provider := useDeletionPolar(t, orgID)
		periodEnd := time.Now().AddDate(0, 1, 0)
		applyDeletionSubscription(t, orgID, billingmodels.PolarSubscriptionStatusActive, false, periodEnd)
		provider.respond(deletionProviderSubscription(orgID, billingmodels.PolarSubscriptionStatusActive, false, periodEnd))

		_, err := DeleteOrganization(ctx, r.AuthService, orgID.String())
		requireInvitationReason(t, err, "ORGANIZATION_SUBSCRIPTION_LIVE")
		assert.Equal(t, message, grpcerrors.StatusMessage(err))
		requireOrganizationKept(t, orgID)

		applyDeletionSubscription(t, orgID, billingmodels.PolarSubscriptionStatusCanceled, false, time.Now().AddDate(0, 1, 0))

		_, err = DeleteOrganization(ctx, r.AuthService, orgID.String())
		require.NoError(t, err)
		_, err = models.FindOrganizationByID(orgID.String())
		assert.Error(t, err)
	})

	t.Run("scheduled cancellation blocks deletion until Polar confirms termination", func(t *testing.T) {
		r := support.Setup(t)
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
		orgID := r.Organization.ID
		provider := useDeletionPolar(t, orgID)
		applyDeletionSubscription(t, orgID, billingmodels.PolarSubscriptionStatusActive, true, time.Now().AddDate(0, 0, 10))
		provider.respond(deletionProviderSubscription(orgID, billingmodels.PolarSubscriptionStatusActive, true, time.Now().AddDate(0, 0, 10)))

		_, err := DeleteOrganization(ctx, r.AuthService, orgID.String())
		requireInvitationReason(t, err, "ORGANIZATION_SUBSCRIPTION_LIVE")

		// The local period elapsed, but Polar resumed the subscription and
		// its event did not arrive yet.
		elapsed := time.Now().Add(-time.Hour)
		require.NoError(t, database.Conn().Model(&billingmodels.OrganizationBillingPlan{}).
			Where("organization_id = ?", orgID).
			Update("current_period_end", elapsed).Error)
		provider.respond(deletionProviderSubscription(orgID, billingmodels.PolarSubscriptionStatusActive, false, elapsed.AddDate(0, 1, 0)))

		_, err = DeleteOrganization(ctx, r.AuthService, orgID.String())
		requireInvitationReason(t, err, "ORGANIZATION_SUBSCRIPTION_LIVE")
		requireOrganizationKept(t, orgID)

		provider.respond(deletionProviderSubscription(orgID, billingmodels.PolarSubscriptionStatusCanceled, true, elapsed))

		_, err = DeleteOrganization(ctx, r.AuthService, orgID.String())
		require.NoError(t, err)
		_, err = models.FindOrganizationByID(orgID.String())
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("Polar failure keeps the organization", func(t *testing.T) {
		r := support.Setup(t)
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
		orgID := r.Organization.ID
		provider := useDeletionPolar(t, orgID)
		applyDeletionSubscription(t, orgID, billingmodels.PolarSubscriptionStatusActive, true, time.Now().Add(-time.Hour))
		provider.fail(http.StatusInternalServerError)

		_, err := DeleteOrganization(ctx, r.AuthService, orgID.String())

		requireInvitationErrorReason(t, err, codes.Unavailable, "ORGANIZATION_SUBSCRIPTION_UNVERIFIED")
		assert.Equal(t, billingmodels.ErrOrganizationSubscriptionUnverified.Error(), grpcerrors.StatusMessage(err))
		requireOrganizationKept(t, orgID)
	})

	t.Run("subscription change during verification stops deletion", func(t *testing.T) {
		r := support.Setup(t)
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
		orgID := r.Organization.ID
		provider := useDeletionPolar(t, orgID)
		elapsed := time.Now().Add(-time.Hour)
		applyDeletionSubscription(t, orgID, billingmodels.PolarSubscriptionStatusActive, true, elapsed)
		provider.respond(deletionProviderSubscription(orgID, billingmodels.PolarSubscriptionStatusCanceled, true, elapsed))
		// A webhook commits while Polar answers. The verification holds no
		// organization lock, so the webhook does not wait for it.
		provider.beforeResponse = func() {
			modifiedAt := time.Now()
			_, err := billingmodels.ApplyPolarSubscription(database.Conn(), orgID, billingmodels.PolarSubscriptionApply{
				ID: deletionSubscriptionID, ProductID: "prod_business", Status: billingmodels.PolarSubscriptionStatusActive,
				ModifiedAt: &modifiedAt,
			}, nil)
			assert.NoError(t, err)
		}

		_, err := DeleteOrganization(ctx, r.AuthService, orgID.String())

		require.Equal(t, codes.Aborted, grpcerrors.Code(err))
		requireOrganizationKept(t, orgID)
	})

	t.Run("admin-assigned plan is deleted", func(t *testing.T) {
		r := support.Setup(t)
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		require.NoError(t, billingmodels.BindLegacyBillingProduct(database.Conn(), "prod_business"))
		_, err := billingmodels.SetAdminOrganizationPlan(database.Conn(), r.Organization.ID, billingmodels.BillingPlanBusiness, nil)
		require.NoError(t, err)
		useDeletionPolar(t, r.Organization.ID).fail(http.StatusInternalServerError)

		_, err = DeleteOrganization(ctx, r.AuthService, r.Organization.ID.String())
		require.NoError(t, err)
	})

	t.Run("organization without a billing row is deleted", func(t *testing.T) {
		r := support.Setup(t)
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		require.NoError(t, database.Conn().Where("organization_id = ?", r.Organization.ID).Delete(&billingmodels.OrganizationBillingPlan{}).Error)
		useDeletionPolar(t, r.Organization.ID).fail(http.StatusInternalServerError)

		_, err := DeleteOrganization(ctx, r.AuthService, r.Organization.ID.String())
		require.NoError(t, err)
	})
}

const deletionSubscriptionID = "sub_delete_guard"

func applyDeletionSubscription(t *testing.T, orgID uuid.UUID, status string, cancelAtPeriodEnd bool, periodEnd time.Time) {
	t.Helper()
	start := periodEnd.AddDate(0, -1, 0)
	_, err := billingmodels.ApplyPolarSubscription(database.Conn(), orgID, billingmodels.PolarSubscriptionApply{
		ID: deletionSubscriptionID, ProductID: "prod_business", Status: status,
		PeriodStart: &start, PeriodEnd: &periodEnd, CancelAtPeriodEnd: cancelAtPeriodEnd,
	}, nil)
	require.NoError(t, err)
}

func requireOrganizationKept(t *testing.T, orgID uuid.UUID) {
	t.Helper()
	organization, err := models.FindOrganizationByID(orgID.String())
	require.NoError(t, err)
	assert.False(t, organization.DeletedAt.Valid)
}

func deletionProviderSubscription(orgID uuid.UUID, status string, cancelAtPeriodEnd bool, periodEnd time.Time) map[string]any {
	return map[string]any{
		"id": deletionSubscriptionID, "status": status, "cancel_at_period_end": cancelAtPeriodEnd,
		"current_period_end":   periodEnd.UTC().Format(time.RFC3339),
		"external_customer_id": orgID.String(),
	}
}

// deletionPolar answers the subscription list that deletion verification
// reads. A test without a response fails on any Polar request.
type deletionPolar struct {
	t              *testing.T
	mu             sync.Mutex
	items          []map[string]any
	failure        int
	beforeResponse func()
}

func useDeletionPolar(t *testing.T, orgID uuid.UUID) *deletionPolar {
	t.Helper()
	provider := &deletionPolar{t: t}
	polartest.UseServer(t, polartest.APIServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/subscriptions/", r.URL.Path)
		assert.Equal(t, orgID.String(), r.URL.Query().Get("external_customer_id"))
		provider.serve(w)
	}))
	return provider
}

func (p *deletionPolar) respond(items ...map[string]any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.items, p.failure = items, 0
}

func (p *deletionPolar) fail(status int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.items, p.failure = nil, status
}

func (p *deletionPolar) serve(w http.ResponseWriter) {
	p.mu.Lock()
	items, failure, before := p.items, p.failure, p.beforeResponse
	p.mu.Unlock()
	if before != nil {
		before()
	}
	if failure != 0 {
		w.WriteHeader(failure)
		return
	}
	if items == nil {
		p.t.Error("unexpected Polar request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	assert.NoError(p.t, json.NewEncoder(w).Encode(map[string]any{"items": items, "pagination": map[string]any{"max_page": 1}}))
}
