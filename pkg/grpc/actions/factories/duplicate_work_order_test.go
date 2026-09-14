package factories

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__DuplicateWorkOrder__CopiesDraftFromAnyState(t *testing.T) {
	r := support.Setup(t)
	enableTaskRefinement(t, r.Organization.ID)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	cases := []struct {
		name  string
		setup func(*models.FactoryWorkOrder)
	}{
		{name: "draft", setup: func(*models.FactoryWorkOrder) {}},
		{name: "open", setup: func(order *models.FactoryWorkOrder) {
			require.NoError(t, order.TransitionOnDispatch(db, &r.User))
		}},
		{name: "closed", setup: func(order *models.FactoryWorkOrder) {
			require.NoError(t, order.TransitionOnDispatch(db, &r.User))
			_, err := order.Close(db, models.FactoryWorkOrderResultCompleted, &r.User)
			require.NoError(t, err)
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source, err := factoryModel.CreateWorkOrder(db, "Ship refunds", "Fix the ledger.", &r.User, nil, nil)
			require.NoError(t, err)
			tc.setup(source)
			source, err = factoryModel.FindWorkOrder(db, source.ID)
			require.NoError(t, err)
			sourceState := source.State

			resp, err := DuplicateWorkOrder(ctx, r.Organization.ID.String(), &pb.DuplicateWorkOrderRequest{
				FactoryId: factoryModel.ID.String(),
				OrderId:   source.ID.String(),
			})
			require.NoError(t, err)
			require.NotNil(t, resp.Order)
			assert.Equal(t, pb.WorkOrder_STATE_DRAFT, resp.Order.State)
			assert.Equal(t, "Ship refunds", resp.Order.Title)
			assert.Equal(t, "Fix the ledger.", resp.Order.Description)
			assert.NotEqual(t, source.ID.String(), resp.Order.Id)
			require.NotEmpty(t, resp.Order.Assignees)
			assert.Equal(t, r.User.String(), resp.Order.Assignees[0].Id)

			reloaded, err := factoryModel.FindWorkOrder(db, source.ID)
			require.NoError(t, err)
			assert.Equal(t, sourceState, reloaded.State)
		})
	}
}

func Test__DuplicateWorkOrder__CopiesOriginAndRepository(t *testing.T) {
	r := support.Setup(t)
	enableTaskRefinement(t, r.Organization.ID)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	source, err := factoryModel.CreateWorkOrderWithOrigin(
		db,
		"From GitHub",
		"Body",
		&r.User,
		nil,
		nil,
		models.WorkOrderOrigin{URL: "https://github.com/acme/pay/issues/12", Label: "acme/pay#12"},
	)
	require.NoError(t, err)
	require.NoError(t, db.Model(source).Updates(map[string]any{
		"repository":     "acme/legacy",
		"default_branch": "release",
	}).Error)

	resp, err := DuplicateWorkOrder(ctx, r.Organization.ID.String(), &pb.DuplicateWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   source.ID.String(),
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Order.Origin)
	assert.Equal(t, "https://github.com/acme/pay/issues/12", resp.Order.Origin.Url)
	assert.Equal(t, "acme/pay#12", resp.Order.Origin.Label)

	copiedID, err := uuid.Parse(resp.Order.Id)
	require.NoError(t, err)
	copied, err := factoryModel.FindWorkOrder(db, copiedID)
	require.NoError(t, err)
	require.NotNil(t, copied.Repository)
	require.NotNil(t, copied.DefaultBranch)
	assert.Equal(t, "acme/legacy", *copied.Repository)
	assert.Equal(t, "release", *copied.DefaultBranch)
}

func Test__DuplicateWorkOrder__CopiesDescriptionFiles(t *testing.T) {
	r := support.Setup(t)
	enableTaskRefinement(t, r.Organization.ID)
	t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
	t.Setenv("BASE_URL", "http://files.test")
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	source, err := factoryModel.CreateWorkOrder(db, "Login screenshot", "", &r.User, nil, nil)
	require.NoError(t, err)

	file, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeTask,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    source.ID,
		Filename:       "bug.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	require.NoError(t, storedfiles.CompleteUpload(ctx, db, store, file, bytes.NewReader([]byte("png-bytes"))))
	description := "See ![bug](" + blob.FileRef(file.ID) + ")"
	require.NoError(t, source.UpdateContent(db, nil, &description))

	resp, err := DuplicateWorkOrder(ctx, r.Organization.ID.String(), &pb.DuplicateWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   source.ID.String(),
	})
	require.NoError(t, err)
	require.Len(t, resp.Order.Files, 1)
	assert.NotEqual(t, file.ID.String(), resp.Order.Files[0].Id)
	copiedFileID, err := uuid.Parse(resp.Order.Files[0].Id)
	require.NoError(t, err)
	assert.Contains(t, resp.Order.Description, blob.FileRef(copiedFileID))
	assert.NotContains(t, resp.Order.Description, blob.FileRef(file.ID))

	original, err := models.FindFile(db, file.ID)
	require.NoError(t, err)
	assert.Equal(t, source.ID, *original.WorkOrderID)
}

func Test__DuplicateWorkOrder__CopiesSpecAndSkipsOtherArtifacts(t *testing.T) {
	r := support.Setup(t)
	enableTaskRefinement(t, r.Organization.ID)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	source, err := factoryModel.CreateWorkOrder(db, "With spec", "Body", &r.User, nil, nil)
	require.NoError(t, err)

	_, err = source.CreateArtifact(db, models.FactoryWorkOrderArtifactParams{
		Type: models.FactoryWorkOrderArtifactTypeMarkdown,
		Key:  models.PlanningSpecArtifactKey + ":" + source.ID.String(),
		Data: map[string]any{
			"name":  models.PlanningSpecArtifactTitle,
			"title": models.PlanningSpecArtifactTitle,
			"body":  "# Spec\nDo the thing.",
		},
	})
	require.NoError(t, err)
	_, err = source.CreateArtifact(db, models.FactoryWorkOrderArtifactParams{
		Type: models.FactoryWorkOrderArtifactTypeLink,
		Data: map[string]any{"url": "https://preview.example.com/pr-1", "title": "Preview"},
	})
	require.NoError(t, err)

	resp, err := DuplicateWorkOrder(ctx, r.Organization.ID.String(), &pb.DuplicateWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   source.ID.String(),
	})
	require.NoError(t, err)
	copiedID, err := uuid.Parse(resp.Order.Id)
	require.NoError(t, err)
	copied, err := factoryModel.FindWorkOrder(db, copiedID)
	require.NoError(t, err)

	spec, err := copied.FindArtifactByKey(db, models.PlanningSpecArtifactKey+":"+copied.ID.String())
	require.NoError(t, err)
	assert.Contains(t, string(spec.Data), "Do the thing.")

	artifacts, err := copied.ListArtifacts(db)
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
}

func Test__DuplicateWorkOrder__SkipsSpecWhenSourceHasNone(t *testing.T) {
	r := support.Setup(t)
	enableTaskRefinement(t, r.Organization.ID)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	source, err := factoryModel.CreateWorkOrder(db, "No spec", "Body", &r.User, nil, nil)
	require.NoError(t, err)

	resp, err := DuplicateWorkOrder(ctx, r.Organization.ID.String(), &pb.DuplicateWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   source.ID.String(),
	})
	require.NoError(t, err)
	copiedID, err := uuid.Parse(resp.Order.Id)
	require.NoError(t, err)
	copied, err := factoryModel.FindWorkOrder(db, copiedID)
	require.NoError(t, err)
	artifacts, err := copied.ListArtifacts(db)
	require.NoError(t, err)
	assert.Empty(t, artifacts)
}

func Test__DuplicateWorkOrder__DoesNotCopyRunsOrComments(t *testing.T) {
	r := support.Setup(t)
	enableTaskRefinement(t, r.Organization.ID)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	source, err := factoryModel.CreateWorkOrder(db, "Commented", "Body", &r.User, nil, nil)
	require.NoError(t, err)

	userID := r.User.String()
	_, err = source.RecordCommentAdded(db, models.FactoryWorkOrderCommentParams{
		Body: "Keep this on the original",
		Author: factory.WorkOrderCommentAuthor{
			Kind:   factory.CommentAuthorKindUser,
			UserID: &userID,
		},
	})
	require.NoError(t, err)

	resp, err := DuplicateWorkOrder(ctx, r.Organization.ID.String(), &pb.DuplicateWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   source.ID.String(),
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Order.SourceRunId)
	assert.Empty(t, resp.Order.LineDispatches)

	copiedID, err := uuid.Parse(resp.Order.Id)
	require.NoError(t, err)
	copied, err := factoryModel.FindWorkOrder(db, copiedID)
	require.NoError(t, err)
	comments, err := copied.ListComments(db)
	require.NoError(t, err)
	assert.Empty(t, comments)
	assert.Nil(t, copied.SourceRunID)
}

func Test__DuplicateWorkOrder__PermissionDeniedWithoutTaskRefinement(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	source, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	_, err = DuplicateWorkOrder(ctx, r.Organization.ID.String(), &pb.DuplicateWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   source.ID.String(),
	})
	code, _, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, code)
}

func enableTaskRefinement(t *testing.T, orgID uuid.UUID) {
	t.Helper()
	require.NoError(t, models.EnableExperimentalFeature(orgID, features.FeatureFactoryCreateWithAgent))
}
