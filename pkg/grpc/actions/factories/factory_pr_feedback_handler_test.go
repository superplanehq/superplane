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

		assert.Equal(t, pb.FactoryPRFeedbackHandler_SUBJECT_GITHUB_PULL_REQUEST, handler.GetSubject())
		assert.Equal(t, pb.FactoryPRFeedbackHandler_SOURCE_PULL_REQUEST_DISCUSSION, handler.GetSource())
		assert.Equal(t, prFeedbackDefaultName, handler.GetName())
		assert.True(t, handler.GetHealthy())
		assert.Equal(t, "acme/app", handler.GetSettings().GetSubject().GetRepository())
		assert.Equal(t, prFeedbackDefaultMention, handler.GetSettings().GetDiscussion().GetMention())
		assert.True(t, handler.GetSettings().GetDiscussion().GetIgnoreBots())

		canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(handler.GetCanvasId()))
		require.NoError(t, err)
		require.NotNil(t, canvas.FactoryID)
		assert.Equal(t, factory.ID, *canvas.FactoryID)

		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
		require.NoError(t, err)
		assert.Len(t, liveVersion.Nodes, 6)
		assert.Len(t, liveVersion.Edges, 5)
		for _, node := range liveVersion.Nodes {
			assert.NotEqual(t, "noop", node.ComponentName())
		}
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
				Subject: &pb.FactoryPRFeedbackHandler_SubjectSettings{
					Repository: "acme/other",
				},
				Discussion: &pb.FactoryPRFeedbackHandler_DiscussionSettings{
					Mention:     "@superplaneagent",
					IgnoreBots:  true,
					AllowedBots: []string{"coderabbitai", "bugbot"},
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "Address review comments", response.GetHandler().GetName())
		assert.Equal(t, "acme/other", response.GetHandler().GetSettings().GetSubject().GetRepository())
		assert.True(t, response.GetHandler().GetHealthy())
		assert.Equal(t, []string{"coderabbitai", "bugbot"}, response.GetHandler().GetSettings().GetDiscussion().GetAllowedBots())

		canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(handler.GetCanvasId()))
		require.NoError(t, err)
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
		require.NoError(t, err)
		for _, node := range liveVersion.Nodes {
			if node.ID == prFeedbackActivityNodeID {
				assert.Equal(t, prFeedbackActivityDescriptionExpression(), node.Configuration["description"])
			}
			if node.ID != prFeedbackCommentTriggerNodeID && node.ID != prFeedbackReviewTriggerNodeID && node.ID != prFeedbackReplyTriggerNodeID {
				continue
			}
			assert.Equal(t, "acme/other", node.Configuration["repository"])
			assert.Equal(t, "@superplaneagent", node.Configuration["contentFilter"])
			assert.Equal(t, []any{"coderabbitai", "bugbot"}, node.Configuration["allowedBots"])
		}
	})

	t.Run("creating a checks handler waits for pull request checks", func(t *testing.T) {
		factory := newFactory(t)
		appRepo := "acme/app"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			AppRepository: &appRepo,
		}))

		handler := create(t, factory, &pb.CreateFactoryPRFeedbackHandlerRequest{
			Source: pb.FactoryPRFeedbackHandler_SOURCE_PULL_REQUEST_CHECKS,
		})

		assert.Equal(t, pb.FactoryPRFeedbackHandler_SOURCE_PULL_REQUEST_CHECKS, handler.GetSource())
		assert.Equal(t, prFeedbackChecksDefaultName, handler.GetName())
		assert.True(t, handler.GetHealthy())
		assert.Equal(t, "acme/app", handler.GetSettings().GetSubject().GetRepository())
		assert.Empty(t, handler.GetSettings().GetChecks().GetNames())
		assert.Equal(t, int32(prFeedbackDefaultMaximumAttempts), handler.GetSettings().GetChecks().GetMaximumAttempts())

		canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(handler.GetCanvasId()))
		require.NoError(t, err)
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
		require.NoError(t, err)
		assert.Len(t, liveVersion.Nodes, 11)
		var foundWait, foundAnnounce bool
		for _, node := range liveVersion.Nodes {
			if node.ID == prFeedbackWaitChecksNodeID {
				foundWait = true
				assert.Equal(t, prFeedbackWaitChecksComponent, node.ComponentName())
			}
			if node.ID == prFeedbackAnnounceLimitNodeID {
				foundAnnounce = true
				assert.Equal(t, prFeedbackSetStatusNoteComponent, node.ComponentName())
			}
		}
		assert.True(t, foundWait)
		assert.True(t, foundAnnounce)
	})

	t.Run("checks creation rejects an invalid attempt limit", func(t *testing.T) {
		factory := newFactory(t)
		appRepo := "acme/app"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			AppRepository: &appRepo,
		}))
		zero := int32(0)
		_, err := CreateFactoryPRFeedbackHandler(ctx, deps, orgID, &pb.CreateFactoryPRFeedbackHandlerRequest{
			FactoryId: factory.ID.String(),
			Source:    pb.FactoryPRFeedbackHandler_SOURCE_PULL_REQUEST_CHECKS,
			Settings: &pb.FactoryPRFeedbackHandler_Settings{
				Checks: &pb.FactoryPRFeedbackHandler_CheckSettings{MaximumAttempts: &zero},
			},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("discussion creation rejects runner integrations", func(t *testing.T) {
		factory := newFactory(t)
		appRepo := "acme/app"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			AppRepository: &appRepo,
		}))
		_, err := CreateFactoryPRFeedbackHandler(ctx, deps, orgID, &pb.CreateFactoryPRFeedbackHandlerRequest{
			FactoryId: factory.ID.String(),
			Settings: &pb.FactoryPRFeedbackHandler_Settings{
				Checks: &pb.FactoryPRFeedbackHandler_CheckSettings{
					RunnerIntegrationIds: []string{uuid.NewString()},
				},
			},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("checks settings updates reach the wait and pause nodes", func(t *testing.T) {
		factory := newFactory(t)
		appRepo := "acme/app"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			AppRepository: &appRepo,
		}))
		handler := create(t, factory, &pb.CreateFactoryPRFeedbackHandlerRequest{
			Source: pb.FactoryPRFeedbackHandler_SOURCE_PULL_REQUEST_CHECKS,
		})

		name := "Fix selected checks"
		maximumAttempts := int32(5)
		response, err := UpdateFactoryPRFeedbackHandler(ctx, deps, orgID, &pb.UpdateFactoryPRFeedbackHandlerRequest{
			FactoryId: factory.ID.String(),
			HandlerId: handler.GetId(),
			Name:      &name,
			Settings: &pb.FactoryPRFeedbackHandler_Settings{
				Subject: &pb.FactoryPRFeedbackHandler_SubjectSettings{
					Repository: "acme/other",
				},
				Checks: &pb.FactoryPRFeedbackHandler_CheckSettings{
					Names:           []string{"lint", "unit"},
					MaximumAttempts: &maximumAttempts,
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "Fix selected checks", response.GetHandler().GetName())
		assert.Equal(t, []string{"lint", "unit"}, response.GetHandler().GetSettings().GetChecks().GetNames())
		assert.Equal(t, int32(5), response.GetHandler().GetSettings().GetChecks().GetMaximumAttempts())

		canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(handler.GetCanvasId()))
		require.NoError(t, err)
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
		require.NoError(t, err)
		for _, node := range liveVersion.Nodes {
			if node.ID == prFeedbackPullRequestTriggerNodeID || node.ID == prFeedbackWaitChecksNodeID {
				assert.Equal(t, "acme/other", node.Configuration["repository"])
			}
			if node.ID == prFeedbackWaitChecksNodeID {
				assert.Equal(t, []any{"lint", "unit"}, node.Configuration["checkNames"])
			}
			if node.ID == prFeedbackPauseFixesNodeID {
				assert.Equal(t, "Automatic fixes paused after 5 attempts", node.Configuration["description"])
			}
			if node.ID == prFeedbackAnnounceLimitNodeID {
				assert.Equal(t, prFeedbackChecksLimitStatusNoteBody(5), node.Configuration["body"])
				assert.Equal(t, "Automatic fixes did not succeed", node.Configuration["headline"])
			}
		}
	})

	t.Run("creating a conflicts handler waits for merge conflicts", func(t *testing.T) {
		factory := newFactory(t)
		appRepo := "acme/app"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			AppRepository: &appRepo,
		}))

		handler := create(t, factory, &pb.CreateFactoryPRFeedbackHandlerRequest{
			Source: pb.FactoryPRFeedbackHandler_SOURCE_PULL_REQUEST_CONFLICTS,
		})

		assert.Equal(t, pb.FactoryPRFeedbackHandler_SOURCE_PULL_REQUEST_CONFLICTS, handler.GetSource())
		assert.Equal(t, prFeedbackConflictsDefaultName, handler.GetName())
		assert.True(t, handler.GetHealthy())
		assert.Equal(t, "acme/app", handler.GetSettings().GetSubject().GetRepository())
		assert.Equal(t, "main", handler.GetSettings().GetConflicts().GetBaseBranch())
		assert.Equal(t, int32(prFeedbackDefaultMaximumAttempts), handler.GetSettings().GetConflicts().GetMaximumAttempts())

		canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(handler.GetCanvasId()))
		require.NoError(t, err)
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
		require.NoError(t, err)
		assert.Len(t, liveVersion.Nodes, 10)
		var foundWait, foundList, foundForEach, foundRepair bool
		for _, node := range liveVersion.Nodes {
			if node.ID == prFeedbackWaitMergeableNodeID {
				foundWait = true
				assert.Equal(t, prFeedbackWaitMergeableComponent, node.ComponentName())
			}
			if node.ID == prFeedbackListNodeID {
				foundList = true
				assert.Equal(t, prFeedbackListComponent, node.ComponentName())
			}
			if node.ID == prFeedbackForEachNodeID {
				foundForEach = true
			}
			if node.ID == prFeedbackStartConflictRepairNodeID {
				foundRepair = true
				assert.Equal(t, prFeedbackActivityComponent, node.ComponentName())
				assert.Equal(t, "exclusive", node.Configuration["access"])
			}
		}
		assert.True(t, foundWait)
		assert.True(t, foundList)
		assert.True(t, foundForEach)
		assert.True(t, foundRepair)
	})

	t.Run("a conflicts handler uses the onboarding default branch", func(t *testing.T) {
		factory := newFactory(t)
		appRepo := "acme/app"
		defaultBranch := "develop"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			AppRepository: &appRepo,
			DefaultBranch: &defaultBranch,
		}))

		handler := create(t, factory, &pb.CreateFactoryPRFeedbackHandlerRequest{
			Source: pb.FactoryPRFeedbackHandler_SOURCE_PULL_REQUEST_CONFLICTS,
		})

		assert.Equal(t, "develop", handler.GetSettings().GetConflicts().GetBaseBranch())

		canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(handler.GetCanvasId()))
		require.NoError(t, err)
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
		require.NoError(t, err)
		for _, node := range liveVersion.Nodes {
			if node.ID != prFeedbackPushTriggerNodeID {
				continue
			}
			assert.Equal(t, "develop", conflictsBaseBranchFromPushNode(&node))
		}
	})

	t.Run("conflicts creation rejects an invalid attempt limit", func(t *testing.T) {
		factory := newFactory(t)
		appRepo := "acme/app"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			AppRepository: &appRepo,
		}))
		zero := int32(0)
		_, err := CreateFactoryPRFeedbackHandler(ctx, deps, orgID, &pb.CreateFactoryPRFeedbackHandlerRequest{
			FactoryId: factory.ID.String(),
			Source:    pb.FactoryPRFeedbackHandler_SOURCE_PULL_REQUEST_CONFLICTS,
			Settings: &pb.FactoryPRFeedbackHandler_Settings{
				Conflicts: &pb.FactoryPRFeedbackHandler_ConflictSettings{MaximumAttempts: &zero},
			},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("conflicts creation rejects a mention", func(t *testing.T) {
		factory := newFactory(t)
		appRepo := "acme/app"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			AppRepository: &appRepo,
		}))
		_, err := CreateFactoryPRFeedbackHandler(ctx, deps, orgID, &pb.CreateFactoryPRFeedbackHandlerRequest{
			FactoryId: factory.ID.String(),
			Source:    pb.FactoryPRFeedbackHandler_SOURCE_PULL_REQUEST_CONFLICTS,
			Settings: &pb.FactoryPRFeedbackHandler_Settings{
				Discussion: &pb.FactoryPRFeedbackHandler_DiscussionSettings{Mention: "@superplaneagent"},
			},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("conflicts settings updates reach both triggers and the pause node", func(t *testing.T) {
		factory := newFactory(t)
		appRepo := "acme/app"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			AppRepository: &appRepo,
		}))
		handler := create(t, factory, &pb.CreateFactoryPRFeedbackHandlerRequest{
			Source: pb.FactoryPRFeedbackHandler_SOURCE_PULL_REQUEST_CONFLICTS,
		})

		name := "Resolve conflicts fast"
		maximumAttempts := int32(4)
		response, err := UpdateFactoryPRFeedbackHandler(ctx, deps, orgID, &pb.UpdateFactoryPRFeedbackHandlerRequest{
			FactoryId: factory.ID.String(),
			HandlerId: handler.GetId(),
			Name:      &name,
			Settings: &pb.FactoryPRFeedbackHandler_Settings{
				Subject: &pb.FactoryPRFeedbackHandler_SubjectSettings{
					Repository: "acme/other",
				},
				Conflicts: &pb.FactoryPRFeedbackHandler_ConflictSettings{
					BaseBranch:      "release",
					MaximumAttempts: &maximumAttempts,
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "Resolve conflicts fast", response.GetHandler().GetName())
		assert.Equal(t, "release", response.GetHandler().GetSettings().GetConflicts().GetBaseBranch())
		assert.Equal(t, int32(4), response.GetHandler().GetSettings().GetConflicts().GetMaximumAttempts())
		assert.True(t, response.GetHandler().GetHealthy())

		canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(handler.GetCanvasId()))
		require.NoError(t, err)
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
		require.NoError(t, err)
		for _, node := range liveVersion.Nodes {
			if node.ID == prFeedbackPullRequestTriggerNodeID || node.ID == prFeedbackWaitMergeableNodeID || node.ID == prFeedbackListNodeID {
				assert.Equal(t, "acme/other", node.Configuration["repository"])
			}
			if node.ID == prFeedbackPushTriggerNodeID {
				assert.Equal(t, "acme/other", node.Configuration["repository"])
				assert.Equal(t, "release", conflictsBaseBranchFromPushNode(&node))
			}
			if node.ID == prFeedbackPauseFixesNodeID {
				assert.Equal(t, "Automatic conflict fixes paused after 4 attempts", node.Configuration["description"])
			}
			if node.ID == prFeedbackAnnounceLimitNodeID {
				assert.Equal(t, prFeedbackConflictsLimitStatusNoteBody(4), node.Configuration["body"])
				assert.Equal(t, "Automatic conflict fixes did not succeed", node.Configuration["headline"])
			}
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
}
