package factories

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/blob"
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

	t.Run("creates link artifact", func(t *testing.T) {
		data, err := structpb.NewStruct(map[string]any{
			"url":   "https://preview.example.com/pr-42",
			"title": "Preview",
		})
		require.NoError(t, err)

		resp, err := CreateWorkOrderArtifact(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderArtifactRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Type:      pb.WorkOrderArtifact_TYPE_LINK,
			Data:      data,
		})
		require.NoError(t, err)
		assert.Equal(t, pb.WorkOrderArtifact_TYPE_LINK, resp.Artifact.Type)
		assert.Equal(t, "https://preview.example.com/pr-42", resp.Artifact.Data.AsMap()["url"])
		assert.Equal(t, "Preview", resp.Artifact.Data.AsMap()["title"])
	})

	t.Run("rejects link without url", func(t *testing.T) {
		_, err := CreateWorkOrderArtifact(ctx, r.Organization.ID.String(), &pb.CreateWorkOrderArtifactRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Type:      pb.WorkOrderArtifact_TYPE_LINK,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
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
			Type:      pb.WorkOrderArtifact_TYPE_LINK,
			Data:      data,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("creates file artifact from an uploaded artifact file", func(t *testing.T) {
		file := createReadyArtifactFile(t, r.Organization.ID, factoryModel.ID, order.ID, r.User)

		resp, err := createFileArtifact(ctx, r.Organization.ID.String(), factoryModel.ID.String(), order.ID.String(), file.ID.String())

		require.NoError(t, err)
		assert.Equal(t, pb.WorkOrderArtifact_TYPE_FILE, resp.Artifact.Type)
		artifactData := resp.Artifact.Data.AsMap()
		assert.Equal(t, file.ID.String(), artifactData["fileId"])
		assert.Equal(t, "login.png", artifactData["title"])
		assert.Contains(t, artifactData["url"], "/api/v1/public/artifacts/"+file.PublicID.String()+"/login.png")

		_, err = createFileArtifact(ctx, r.Organization.ID.String(), factoryModel.ID.String(), order.ID.String(), file.ID.String())
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code, "one file cannot back two artifacts")
	})

	t.Run("rejects a file from another task", func(t *testing.T) {
		other, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Other", "", &r.User, nil, nil)
		require.NoError(t, err)
		file := createReadyArtifactFile(t, r.Organization.ID, factoryModel.ID, other.ID, r.User)

		_, err = createFileArtifact(ctx, r.Organization.ID.String(), factoryModel.ID.String(), order.ID.String(), file.ID.String())

		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("rejects a file from another user", func(t *testing.T) {
		otherUser := support.CreateUser(t, r, r.Organization.ID)
		file := createReadyArtifactFile(t, r.Organization.ID, factoryModel.ID, order.ID, otherUser.ID)

		_, err := createFileArtifact(ctx, r.Organization.ID.String(), factoryModel.ID.String(), order.ID.String(), file.ID.String())

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

func createFileArtifact(ctx context.Context, organizationID, factoryID, orderID, fileID string) (*pb.CreateWorkOrderArtifactResponse, error) {
	data, err := structpb.NewStruct(map[string]any{"fileId": fileID})
	if err != nil {
		return nil, err
	}
	return CreateWorkOrderArtifact(ctx, organizationID, &pb.CreateWorkOrderArtifactRequest{
		FactoryId: factoryID,
		OrderId:   orderID,
		Type:      pb.WorkOrderArtifact_TYPE_FILE,
		Data:      data,
	})
}

func createReadyArtifactFile(t *testing.T, organizationID, factoryID, orderID, userID uuid.UUID) *models.File {
	t.Helper()
	db := database.DB(t.Context())
	file, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeTask,
		OrganizationID: organizationID,
		FactoryID:      factoryID,
		WorkOrderID:    orderID,
		Filename:       "login.png",
		ContentType:    "image/png",
		CreatedByID:    userID,
		Purpose:        models.FilePurposeArtifact,
	})
	require.NoError(t, err)
	require.NoError(t, file.MarkReady(db, 9, ""))
	loaded, err := models.FindFile(db, file.ID)
	require.NoError(t, err)
	return loaded
}
