package organizations

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListOrganizationCreditGrants(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	t.Run("invalid organization id", func(t *testing.T) {
		_, err := ListOrganizationCreditGrants(context.Background(), "not-a-uuid", &pb.ListOrganizationCreditGrantsRequest{})
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})

	t.Run("lists grants newest first with actor name and signed refund", func(t *testing.T) {
		actor := r.Account.ID
		_, err := models.AddAdminLLMCreditGrant(db, r.Organization.ID, models.CentsToMicros(1500), "Support grant", &actor)
		require.NoError(t, err)

		orderID := uuid.NewString()
		_, err = models.AddPolarLLMCreditGrant(db, r.Organization.ID, models.CentsToMicros(2500), orderID)
		require.NoError(t, err)
		_, err = models.AddPolarLLMCreditRefund(db, r.Organization.ID, models.CentsToMicros(500), orderID, orderID+":partial")
		require.NoError(t, err)

		resp, err := ListOrganizationCreditGrants(context.Background(), r.Organization.ID.String(), &pb.ListOrganizationCreditGrantsRequest{})
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(resp.Grants), 4)

		assert.Equal(t, models.LLMCreditGrantKindPolarRefund, resp.Grants[0].Kind)
		assert.Equal(t, int64(-500), resp.Grants[0].AmountCents)
		assert.Equal(t, orderID, resp.Grants[0].PolarOrderId)

		assert.Equal(t, models.LLMCreditGrantKindPolar, resp.Grants[1].Kind)
		assert.Equal(t, int64(2500), resp.Grants[1].AmountCents)

		assert.Equal(t, models.LLMCreditGrantKindAdmin, resp.Grants[2].Kind)
		assert.Equal(t, int64(1500), resp.Grants[2].AmountCents)
		assert.Equal(t, "Support grant", resp.Grants[2].Note)
		assert.Equal(t, r.Account.Name, resp.Grants[2].ActorName)

		assert.Equal(t, models.LLMCreditGrantKindWelcome, resp.Grants[3].Kind)
		assert.Equal(t, int64(models.DefaultWelcomeGrantCents), resp.Grants[3].AmountCents)
		assert.Empty(t, resp.Grants[3].ActorName)
	})
}
