package organizations

import (
	"context"
	"errors"
	"testing"
	"time"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
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

	applySubscription := func(t *testing.T, orgID uuid.UUID, status string, cancelAtPeriodEnd bool, periodEnd time.Time) {
		t.Helper()
		start := periodEnd.AddDate(0, -1, 0)
		_, _, err := models.ApplyPolarSubscription(database.Conn(), orgID, models.PolarSubscriptionApply{
			ID: "sub_delete_guard", ProductID: "prod_business", Status: status,
			PeriodStart: &start, PeriodEnd: &periodEnd, CancelAtPeriodEnd: cancelAtPeriodEnd,
		})
		require.NoError(t, err)
	}

	t.Run("live subscription blocks deletion until it ends", func(t *testing.T) {
		r := support.Setup(t)
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
		orgID := r.Organization.ID
		applySubscription(t, orgID, models.PolarSubscriptionStatusActive, false, time.Now().AddDate(0, 1, 0))

		_, err := DeleteOrganization(ctx, r.AuthService, orgID.String())
		requireInvitationReason(t, err, "ORGANIZATION_SUBSCRIPTION_LIVE")
		assert.Equal(t, message, grpcerrors.StatusMessage(err))
		organization, err := models.FindOrganizationByID(orgID.String())
		require.NoError(t, err)
		assert.False(t, organization.DeletedAt.Valid)

		applySubscription(t, orgID, models.PolarSubscriptionStatusCanceled, false, time.Now().AddDate(0, 1, 0))

		_, err = DeleteOrganization(ctx, r.AuthService, orgID.String())
		require.NoError(t, err)
		_, err = models.FindOrganizationByID(orgID.String())
		assert.Error(t, err)
	})

	t.Run("scheduled cancellation blocks deletion until the period ends", func(t *testing.T) {
		r := support.Setup(t)
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
		orgID := r.Organization.ID
		applySubscription(t, orgID, models.PolarSubscriptionStatusActive, true, time.Now().AddDate(0, 0, 10))

		_, err := DeleteOrganization(ctx, r.AuthService, orgID.String())
		requireInvitationReason(t, err, "ORGANIZATION_SUBSCRIPTION_LIVE")

		require.NoError(t, database.Conn().Model(&models.OrganizationBillingPlan{}).
			Where("organization_id = ?", orgID).
			Update("current_period_end", time.Now().Add(-time.Hour)).Error)

		_, err = DeleteOrganization(ctx, r.AuthService, orgID.String())
		require.NoError(t, err)
	})

	t.Run("admin-assigned plan is deleted", func(t *testing.T) {
		r := support.Setup(t)
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		require.NoError(t, models.BindLegacyBillingProduct(database.Conn(), "prod_business"))
		_, err := models.SetAdminOrganizationPlan(database.Conn(), r.Organization.ID, models.BillingPlanBusiness, nil)
		require.NoError(t, err)

		_, err = DeleteOrganization(ctx, r.AuthService, r.Organization.ID.String())
		require.NoError(t, err)
	})

	t.Run("organization without a billing row is deleted", func(t *testing.T) {
		r := support.Setup(t)
		ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
		require.NoError(t, database.Conn().Where("organization_id = ?", r.Organization.ID).Delete(&models.OrganizationBillingPlan{}).Error)

		_, err := DeleteOrganization(ctx, r.AuthService, r.Organization.ID.String())
		require.NoError(t, err)
	})
}
