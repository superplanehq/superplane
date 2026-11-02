package factories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func TestRemoveWorkOrderArtifactLeavesTheOtherArtifact(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = order.CreateArtifact(db, models.FactoryWorkOrderArtifactParams{
		Type: models.FactoryWorkOrderArtifactTypeMarkdown,
		Data: map[string]any{"title": "PLAN.md", "body": "plan"},
	})
	require.NoError(t, err)
	_, err = order.CreateArtifact(db, models.FactoryWorkOrderArtifactParams{
		Type: models.FactoryWorkOrderArtifactTypeMarkdown,
		Key:  "notes",
		Data: map[string]any{"title": "NOTES.md", "body": "notes"},
	})
	require.NoError(t, err)

	response, err := RemoveWorkOrderArtifact(ctx, r.Organization.ID.String(), &pb.RemoveWorkOrderArtifactRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		Title:     "PLAN.md",
	})
	require.NoError(t, err)
	assert.Equal(t, "PLAN.md", response.GetTitle())

	remaining, err := order.ListArtifacts(db)
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	assert.Equal(t, "notes", *remaining[0].Key)

	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
	require.NoError(t, err)
	_, err = order.Close(db, models.FactoryWorkOrderResultCompleted, &r.User)
	require.NoError(t, err)

	_, err = RemoveWorkOrderArtifact(ctx, r.Organization.ID.String(), &pb.RemoveWorkOrderArtifactRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		Key:       "notes",
	})
	assert.Equal(t, codes.Aborted, grpcerrors.Code(err))
	remaining, err = order.ListArtifacts(db)
	require.NoError(t, err)
	assert.Len(t, remaining, 1)
}
