package factories

import (
	"context"
	"strings"
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
	"google.golang.org/protobuf/types/known/structpb"
)

func Test__CreateWorkOrderArtifact(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	t.Run("creates markdown artifact", func(t *testing.T) {
		data, err := structpb.NewStruct(map[string]any{
			"title": "PLAN.md",
			"body":  "# Plan",
		})
		require.NoError(t, err)

		resp, err := CreateWorkOrderArtifact(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderArtifactRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Type:      pb.WorkOrderArtifact_TYPE_MARKDOWN,
			Data:      data,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Artifact)
		assert.Equal(t, pb.WorkOrderArtifact_TYPE_MARKDOWN, resp.Artifact.Type)
		assert.Equal(t, "PLAN.md", resp.Artifact.Data.AsMap()["title"])
		assert.Equal(t, "# Plan", resp.Artifact.Data.AsMap()["body"])
		require.NotNil(t, resp.Artifact.CreatedBy)
		assert.Equal(t, r.User.String(), resp.Artifact.CreatedBy.Id)
		assert.Equal(t, r.UserModel.Name, resp.Artifact.CreatedBy.Name)

		artifacts, err := order.ListArtifacts(database.DB(t.Context()))
		require.NoError(t, err)
		require.NotEmpty(t, artifacts)
	})

	t.Run("creates pr artifact", func(t *testing.T) {
		data, err := structpb.NewStruct(map[string]any{
			"url":    "https://github.com/org/repo/pull/7",
			"number": float64(7),
		})
		require.NoError(t, err)

		resp, err := CreateWorkOrderArtifact(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderArtifactRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Type:      pb.WorkOrderArtifact_TYPE_PR,
			Data:      data,
		})
		require.NoError(t, err)
		assert.Equal(t, pb.WorkOrderArtifact_TYPE_PR, resp.Artifact.Type)
		assert.Equal(t, "https://github.com/org/repo/pull/7", resp.Artifact.Data.AsMap()["url"])
	})

	t.Run("creates branch artifact", func(t *testing.T) {
		data, err := structpb.NewStruct(map[string]any{"name": "feature/login"})
		require.NoError(t, err)

		resp, err := CreateWorkOrderArtifact(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderArtifactRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Type:      pb.WorkOrderArtifact_TYPE_BRANCH,
			Data:      data,
		})
		require.NoError(t, err)
		assert.Equal(t, pb.WorkOrderArtifact_TYPE_BRANCH, resp.Artifact.Type)
		assert.Equal(t, "feature/login", resp.Artifact.Data.AsMap()["name"])
	})

	t.Run("rejects unspecified type", func(t *testing.T) {
		_, err := CreateWorkOrderArtifact(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderArtifactRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Type:      pb.WorkOrderArtifact_TYPE_UNSPECIFIED,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("rejects markdown without body", func(t *testing.T) {
		data, err := structpb.NewStruct(map[string]any{"title": "PLAN.md"})
		require.NoError(t, err)

		_, err = CreateWorkOrderArtifact(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderArtifactRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Type:      pb.WorkOrderArtifact_TYPE_MARKDOWN,
			Data:      data,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("rejects artifact data over 64KiB", func(t *testing.T) {
		data, err := structpb.NewStruct(map[string]any{
			"body": strings.Repeat("a", models.MaxFactoryWorkOrderArtifactDataBytes),
		})
		require.NoError(t, err)

		_, err = CreateWorkOrderArtifact(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderArtifactRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Type:      pb.WorkOrderArtifact_TYPE_MARKDOWN,
			Data:      data,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("rejects unsafe url", func(t *testing.T) {
		data, err := structpb.NewStruct(map[string]any{"url": "javascript:alert(1)"})
		require.NoError(t, err)

		_, err = CreateWorkOrderArtifact(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderArtifactRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Type:      pb.WorkOrderArtifact_TYPE_PR,
			Data:      data,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("not found factory", func(t *testing.T) {
		data, err := structpb.NewStruct(map[string]any{"body": "x"})
		require.NoError(t, err)

		_, err = CreateWorkOrderArtifact(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderArtifactRequest{
			FactoryId: "00000000-0000-0000-0000-000000000001",
			OrderId:   order.ID.String(),
			Type:      pb.WorkOrderArtifact_TYPE_MARKDOWN,
			Data:      data,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		data, err := structpb.NewStruct(map[string]any{"body": "x"})
		require.NoError(t, err)

		_, err = CreateWorkOrderArtifact(context.Background(), r.Organization.ID.String(), &pb.CreateWorkOrderArtifactRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Type:      pb.WorkOrderArtifact_TYPE_MARKDOWN,
			Data:      data,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, code)
	})
}
