package factories

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"

	_ "github.com/superplanehq/superplane/pkg/registryimports"
)

func TestFixFactoryPullRequestConflicts(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	db := database.DB(t.Context())
	deps := IntakeDependencies{
		Registry:       r.Registry,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		WebhookBaseURL: "http://localhost:8000",
	}
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryPullRequestMerge))

	newFactory := func(t *testing.T) *models.Factory {
		t.Helper()
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		repository := "acme/app"
		require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{AppRepository: &repository}))
		return factory
	}

	useGitHub := func(t *testing.T, api *fakeFactoryGitHub) {
		t.Helper()
		original := newFactoryGitHubAPI
		newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) {
			return api, nil
		}
		t.Cleanup(func() { newFactoryGitHubAPI = original })
	}

	conflictingAPI := func() *fakeFactoryGitHub {
		return &fakeFactoryGitHub{
			pullRequest: &github.PullRequest{
				Mergeable:      github.Ptr(false),
				MergeableState: github.Ptr("dirty"),
				Draft:          github.Ptr(false),
				Head:           &github.PullRequestBranch{SHA: github.Ptr("abc123")},
			},
			repository: allMethodsRepository(),
		}
	}

	t.Run("starts the checks runner without a pull request comment", func(t *testing.T) {
		factory := newFactory(t)
		created, err := CreateFactoryPRFeedbackHandler(ctx, deps, orgID, &pb.CreateFactoryPRFeedbackHandlerRequest{
			FactoryId: factory.ID.String(),
			Source:    pb.FactoryPRFeedbackHandler_SOURCE_PULL_REQUEST_CHECKS,
			Settings: &pb.FactoryPRFeedbackHandler_Settings{
				Checks: &pb.FactoryPRFeedbackHandler_CheckSettings{Names: []string{"lint"}},
			},
		})
		require.NoError(t, err)
		order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
		require.NoError(t, err)
		pr := createMergeTestPullRequest(t, db, order)
		useGitHub(t, conflictingAPI())

		response, err := FixFactoryPullRequestConflicts(ctx, deps, orgID, &pb.FixFactoryPullRequestConflictsRequest{
			FactoryId: factory.ID.String(),
			PrId:      pr.GetId(),
		})
		require.NoError(t, err)
		assert.Equal(t, "CONFLICTING", response.GetPullRequest().GetMergeBlockedReason())
		assert.False(t, response.GetPullRequest().GetMergeable())

		canvasID := uuid.MustParse(created.GetHandler().GetCanvasId())
		var events []models.CanvasEvent
		require.NoError(t, db.Where("workflow_id = ?", canvasID).Find(&events).Error)
		require.Len(t, events, 1)
		assert.Nil(t, events[0].ExecutionID)

		payload := events[0].Data.Data().(map[string]any)
		assert.Equal(t, factoryPullRequestConflictEventType, payload["type"])
		assert.NotEqual(t, "github.prComment", payload["type"])
		data := payload["data"].(map[string]any)
		assert.Equal(t, true, data["conflict_fix"])
		assert.Equal(t, "acme/app", data["repository"].(map[string]any)["full_name"])
		assert.Equal(t, json.Number("42"), data["pull_request"].(map[string]any)["number"])
		assert.Equal(t, json.Number("42"), data["pullrequest"].(map[string]any)["id"])

		nodes, edges, err := models.FindLiveCanvasSpecInTransaction(db, canvasID)
		require.NoError(t, err)
		spec := models.LiveCanvasSpec{Nodes: nodes, Edges: edges}
		graph := resolvePRFeedbackGraph(spec)
		assert.Equal(t, graph.StartRepairNodeID, events[0].NodeID)
		assert.NotEqual(t, graph.WaitChecksNodeID, events[0].NodeID)

		var activities []models.FactoryPullRequestRun
		require.NoError(t, db.Where("pull_request_id = ?", pr.GetId()).Find(&activities).Error)
		require.Len(t, activities, 1)
		assert.Equal(t, models.FactoryPullRequestAccessExclusive, activities[0].Access)
		assert.Nil(t, activities[0].RevisionID)
		assert.Equal(t, factoryPullRequestConflictTitle, activities[0].Title)
		assert.Equal(t, events[0].RunID, activities[0].RunID)
		require.NotNil(t, activities[0].FeedbackHandlerID)
		assert.Equal(t, created.GetHandler().GetId(), activities[0].FeedbackHandlerID.String())

		stored, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: uuid.MustParse(pr.GetId())})
		require.NoError(t, err)
		require.NotNil(t, stored.ActiveMutationRunID)
		assert.Equal(t, events[0].RunID, *stored.ActiveMutationRunID)
	})

	t.Run("refuses when the pull request has no conflict", func(t *testing.T) {
		factory := newFactory(t)
		order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
		require.NoError(t, err)
		pr := createMergeTestPullRequest(t, db, order)
		combined, checks := successChecks()
		useGitHub(t, &fakeFactoryGitHub{
			pullRequest: mergeableGitHubPullRequest("abc123"),
			combined:    combined,
			checkRuns:   checks,
			repository:  allMethodsRepository(),
		})

		_, err = FixFactoryPullRequestConflicts(ctx, deps, orgID, &pb.FixFactoryPullRequestConflictsRequest{
			FactoryId: factory.ID.String(),
			PrId:      pr.GetId(),
		})
		code, message, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, "The pull request has no merge conflict.", message)
	})

	t.Run("refuses when Fix pull request checks is not set up", func(t *testing.T) {
		factory := newFactory(t)
		order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
		require.NoError(t, err)
		pr := createMergeTestPullRequest(t, db, order)
		useGitHub(t, conflictingAPI())

		_, err = FixFactoryPullRequestConflicts(ctx, deps, orgID, &pb.FixFactoryPullRequestConflictsRequest{
			FactoryId: factory.ID.String(),
			PrId:      pr.GetId(),
		})
		code, message, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, "Set up Fix pull request checks for this repository before you fix conflicts.", message)
	})

	t.Run("refuses when merge conflicts are turned off", func(t *testing.T) {
		factory := newFactory(t)
		off := false
		_, err := CreateFactoryPRFeedbackHandler(ctx, deps, orgID, &pb.CreateFactoryPRFeedbackHandlerRequest{
			FactoryId: factory.ID.String(),
			Source:    pb.FactoryPRFeedbackHandler_SOURCE_PULL_REQUEST_CHECKS,
			Settings: &pb.FactoryPRFeedbackHandler_Settings{
				Checks: &pb.FactoryPRFeedbackHandler_CheckSettings{
					Names:             []string{"lint"},
					FixMergeConflicts: &off,
				},
			},
		})
		require.NoError(t, err)
		order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
		require.NoError(t, err)
		pr := createMergeTestPullRequest(t, db, order)
		useGitHub(t, conflictingAPI())

		_, err = FixFactoryPullRequestConflicts(ctx, deps, orgID, &pb.FixFactoryPullRequestConflictsRequest{
			FactoryId: factory.ID.String(),
			PrId:      pr.GetId(),
		})
		code, message, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, "Turn on merge conflicts in Fix pull request checks before you fix this pull request.", message)
	})

	t.Run("refuses while automation is already running", func(t *testing.T) {
		factory := newFactory(t)
		_, err := CreateFactoryPRFeedbackHandler(ctx, deps, orgID, &pb.CreateFactoryPRFeedbackHandlerRequest{
			FactoryId: factory.ID.String(),
		})
		require.NoError(t, err)
		order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
		require.NoError(t, err)
		pr := createMergeTestPullRequest(t, db, order)
		grantExclusivePullRequestAccess(t, db, r, factory, pr)
		useGitHub(t, conflictingAPI())

		_, err = FixFactoryPullRequestConflicts(ctx, deps, orgID, &pb.FixFactoryPullRequestConflictsRequest{
			FactoryId: factory.ID.String(),
			PrId:      pr.GetId(),
		})
		code, message, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, mergeBlockedActiveRun, message)
	})

	t.Run("starts a conflict fix when the workspace base branch is pushed", func(t *testing.T) {
		factory := newFactory(t)
		branch := "develop"
		require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{DefaultBranch: &branch}))
		created, err := CreateFactoryPRFeedbackHandler(ctx, deps, orgID, &pb.CreateFactoryPRFeedbackHandlerRequest{
			FactoryId: factory.ID.String(),
			Source:    pb.FactoryPRFeedbackHandler_SOURCE_PULL_REQUEST_CHECKS,
			Settings: &pb.FactoryPRFeedbackHandler_Settings{
				Checks: &pb.FactoryPRFeedbackHandler_CheckSettings{Names: []string{"lint"}},
			},
		})
		require.NoError(t, err)
		order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
		require.NoError(t, err)
		_ = createMergeTestPullRequest(t, db, order)
		useGitHub(t, conflictingAPI())
		integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "github", support.RandomName("github"), nil)
		require.NoError(t, err)

		RefreshFactoryPullRequestMergeabilityFromGitHubEvent(
			t.Context(),
			deps,
			&models.Webhook{AppInstallationID: &integration.ID},
			"push",
			[]byte(`{"ref":"refs/heads/develop","repository":{"full_name":"acme/app"}}`),
		)

		canvasID := uuid.MustParse(created.GetHandler().GetCanvasId())
		var events []models.CanvasEvent
		require.NoError(t, db.Where("workflow_id = ?", canvasID).Find(&events).Error)
		require.Len(t, events, 1)
		payload := events[0].Data.Data().(map[string]any)
		assert.Equal(t, true, payload["data"].(map[string]any)["conflict_fix"])
	})

	t.Run("ignores a push to a branch other than the workspace base", func(t *testing.T) {
		factory := newFactory(t)
		branch := "develop"
		require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{DefaultBranch: &branch}))
		created, err := CreateFactoryPRFeedbackHandler(ctx, deps, orgID, &pb.CreateFactoryPRFeedbackHandlerRequest{
			FactoryId: factory.ID.String(),
			Source:    pb.FactoryPRFeedbackHandler_SOURCE_PULL_REQUEST_CHECKS,
			Settings: &pb.FactoryPRFeedbackHandler_Settings{
				Checks: &pb.FactoryPRFeedbackHandler_CheckSettings{Names: []string{"lint"}},
			},
		})
		require.NoError(t, err)
		order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
		require.NoError(t, err)
		_ = createMergeTestPullRequest(t, db, order)
		useGitHub(t, conflictingAPI())
		integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "github", support.RandomName("github"), nil)
		require.NoError(t, err)

		RefreshFactoryPullRequestMergeabilityFromGitHubEvent(
			t.Context(),
			deps,
			&models.Webhook{AppInstallationID: &integration.ID},
			"push",
			[]byte(`{"ref":"refs/heads/main","repository":{"full_name":"acme/app"}}`),
		)

		var events []models.CanvasEvent
		require.NoError(t, db.Where("workflow_id = ?", uuid.MustParse(created.GetHandler().GetCanvasId())).Find(&events).Error)
		assert.Empty(t, events)
	})

	t.Run("keeps the task button after the automatic window", func(t *testing.T) {
		factory := newFactory(t)
		branch := "develop"
		require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{DefaultBranch: &branch}))
		_, err := CreateFactoryPRFeedbackHandler(ctx, deps, orgID, &pb.CreateFactoryPRFeedbackHandlerRequest{
			FactoryId: factory.ID.String(),
			Source:    pb.FactoryPRFeedbackHandler_SOURCE_PULL_REQUEST_CHECKS,
			Settings: &pb.FactoryPRFeedbackHandler_Settings{
				Checks: &pb.FactoryPRFeedbackHandler_CheckSettings{Names: []string{"lint"}},
			},
		})
		require.NoError(t, err)
		order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
		require.NoError(t, err)
		pr := createMergeTestPullRequest(t, db, order)
		require.NoError(t, db.Model(&models.FactoryPullRequest{}).
			Where("id = ?", pr.GetId()).
			Update("created_at", time.Now().Add(-48*time.Hour)).Error)
		useGitHub(t, conflictingAPI())
		integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "github", support.RandomName("github"), nil)
		require.NoError(t, err)

		RefreshFactoryPullRequestMergeabilityFromGitHubEvent(
			t.Context(),
			deps,
			&models.Webhook{AppInstallationID: &integration.ID},
			"push",
			[]byte(`{"ref":"refs/heads/develop","repository":{"full_name":"acme/app"}}`),
		)
		stored, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: uuid.MustParse(pr.GetId())})
		require.NoError(t, err)
		assert.Empty(t, stored.MergeBlockedReason)

		response, err := FixFactoryPullRequestConflicts(ctx, deps, orgID, &pb.FixFactoryPullRequestConflictsRequest{
			FactoryId: factory.ID.String(),
			PrId:      pr.GetId(),
		})
		require.NoError(t, err)
		assert.Equal(t, "CONFLICTING", response.GetPullRequest().GetMergeBlockedReason())
	})
}
