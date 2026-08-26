package factories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"

	_ "github.com/superplanehq/superplane/pkg/registryimports"
)

func Test__FactoryPRFeedbackHandlerActions(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	deps := IntakeDependencies{
		Registry:       r.Registry,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		GitProvider:    r.GitProvider,
		WebhookBaseURL: "http://localhost:8000",
	}

	newFactory := func(t *testing.T) *models.Factory {
		t.Helper()
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		return factory
	}

	create := func(t *testing.T, factory *models.Factory, req *pb.CreateFactoryPRFeedbackHandlerRequest) *pb.FactoryPRFeedbackHandler {
		t.Helper()
		req.FactoryId = factory.ID.String()
		response, err := CreateFactoryPRFeedbackHandler(ctx, deps, orgID, req)
		require.NoError(t, err)
		return response.GetHandler()
	}

	t.Run("creating a handler builds a live canvas that can run", func(t *testing.T) {
		factory := newFactory(t)
		appRepo := "acme/app"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			AppRepository: &appRepo,
		}))

		handler := create(t, factory, &pb.CreateFactoryPRFeedbackHandlerRequest{})

		assert.Equal(t, pb.FactoryPRFeedbackHandler_SOURCE_GITHUB_PULL_REQUESTS, handler.GetSource())
		assert.Equal(t, prFeedbackDefaultName, handler.GetName())
		assert.True(t, handler.GetHealthy())
		assert.Equal(t, "acme/app", handler.GetSettings().GetRepository())
		assert.Equal(t, prFeedbackDefaultMention, handler.GetSettings().GetMention())
		assert.True(t, handler.GetSettings().GetIgnoreBots())

		canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(handler.GetCanvasId()))
		require.NoError(t, err)
		require.NotNil(t, canvas.FactoryID)
		assert.Equal(t, factory.ID, *canvas.FactoryID)

		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
		require.NoError(t, err)
		assert.Len(t, liveVersion.Nodes, 6)
		assert.Len(t, liveVersion.Edges, 5)
	})

	t.Run("creation fails when no repository is available", func(t *testing.T) {
		factory := newFactory(t)
		_, err := CreateFactoryPRFeedbackHandler(ctx, deps, orgID, &pb.CreateFactoryPRFeedbackHandlerRequest{
			FactoryId: factory.ID.String(),
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("the handler listens with the workspace connection", func(t *testing.T) {
		factory := newFactory(t)
		integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
		appRepo := "acme/app"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			VCSIntegrationID: &integrationID,
			AppRepository:    &appRepo,
		}))

		handler := create(t, factory, &pb.CreateFactoryPRFeedbackHandlerRequest{})
		assert.True(t, handler.GetHealthy())

		canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(handler.GetCanvasId()))
		require.NoError(t, err)
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
		require.NoError(t, err)

		for _, nodeID := range prFeedbackTriggerNodeIDs {
			var found bool
			for _, node := range liveVersion.Nodes {
				if node.ID != nodeID {
					continue
				}
				found = true
				require.NotNil(t, node.IntegrationID)
				assert.Equal(t, integrationID, *node.IntegrationID)
				assert.Equal(t, "acme/app", node.Configuration["repository"])
			}
			assert.True(t, found, "missing trigger %s", nodeID)
		}
	})

	t.Run("settings updates reach every trigger", func(t *testing.T) {
		factory := newFactory(t)
		appRepo := "acme/app"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			AppRepository: &appRepo,
		}))
		handler := create(t, factory, &pb.CreateFactoryPRFeedbackHandlerRequest{})

		name := "Address review comments"
		response, err := UpdateFactoryPRFeedbackHandler(ctx, deps, orgID, &pb.UpdateFactoryPRFeedbackHandlerRequest{
			FactoryId: factory.ID.String(),
			HandlerId: handler.GetId(),
			Name:      &name,
			Settings: &pb.FactoryPRFeedbackHandler_Settings{
				Repository: "acme/other",
				Mention:    "@superplaneagent",
				IgnoreBots: true,
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "Address review comments", response.GetHandler().GetName())
		assert.Equal(t, "acme/other", response.GetHandler().GetSettings().GetRepository())
		assert.True(t, response.GetHandler().GetHealthy())

		canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(handler.GetCanvasId()))
		require.NoError(t, err)
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
		require.NoError(t, err)
		for _, node := range liveVersion.Nodes {
			if node.ID != prFeedbackCommentTriggerNodeID && node.ID != prFeedbackReviewTriggerNodeID && node.ID != prFeedbackReplyTriggerNodeID {
				continue
			}
			assert.Equal(t, "acme/other", node.Configuration["repository"])
			assert.Equal(t, "@superplaneagent", node.Configuration["contentFilter"])
		}
	})

	t.Run("deleting a handler retires its canvas", func(t *testing.T) {
		factory := newFactory(t)
		appRepo := "acme/app"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			AppRepository: &appRepo,
		}))
		handler := create(t, factory, &pb.CreateFactoryPRFeedbackHandlerRequest{})

		_, err := DeleteFactoryPRFeedbackHandler(ctx, orgID, &pb.DeleteFactoryPRFeedbackHandlerRequest{
			FactoryId: factory.ID.String(),
			HandlerId: handler.GetId(),
		})
		require.NoError(t, err)

		response, err := ListFactoryPRFeedbackHandlers(ctx, orgID, &pb.ListFactoryPRFeedbackHandlersRequest{FactoryId: factory.ID.String()})
		require.NoError(t, err)
		assert.Empty(t, response.GetHandlers())

		_, err = models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(handler.GetCanvasId()))
		assert.Error(t, err)
	})

	t.Run("a new handler has no runs yet", func(t *testing.T) {
		factory := newFactory(t)
		appRepo := "acme/app"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			AppRepository: &appRepo,
		}))
		handler := create(t, factory, &pb.CreateFactoryPRFeedbackHandlerRequest{})

		response, err := ListFactoryPRFeedbackHandlerRuns(ctx, orgID, &pb.ListFactoryPRFeedbackHandlerRunsRequest{
			FactoryId: factory.ID.String(),
			HandlerId: handler.GetId(),
		})
		require.NoError(t, err)
		assert.Empty(t, response.GetRuns())
	})
}
