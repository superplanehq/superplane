package factories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

func Test__CreateWorkOrder__AssignsTheCreator(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	other := support.CreateUser(t, r, r.Organization.ID)

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	resp, err := CreateWorkOrder(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
		FactoryId:   factoryModel.ID.String(),
		Title:       "Ship the refunds line",
		AssigneeIds: []string{other.ID.String()},
	})
	require.NoError(t, err)
	require.Len(t, resp.Order.Assignees, 1)
	assert.Equal(t, r.User.String(), resp.Order.Assignees[0].Id)
	assert.Equal(t, r.User.String(), resp.Order.GetCreatedBy().GetUser().GetId())
}
