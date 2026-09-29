package factories

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

func Test__SendWorkOrderToBacklog(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	db := database.DB(t.Context())

	newFactory := func(t *testing.T) *models.Factory {
		t.Helper()
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		return factory
	}

	closedOrder := func(t *testing.T, factory *models.Factory, title string) *models.FactoryWorkOrder {
		t.Helper()
		order, err := factory.CreateWorkOrder(db, title, "", &r.User, nil, nil)
		require.NoError(t, err)
		_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
			ToState: models.FactoryWorkOrderStateOpen,
			Actor:   &r.User,
		})
		require.NoError(t, err)
		_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
			ToState: models.FactoryWorkOrderStateClosed,
			Result:  models.FactoryWorkOrderResultFailed,
			Actor:   &r.User,
		})
		require.NoError(t, err)
		return order
	}

	useGitHub := func(t *testing.T, api factoryGitHubAPI, err error) {
		t.Helper()
		original := newFactoryGitHubAPI
		newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) {
			return api, err
		}
		t.Cleanup(func() { newFactoryGitHubAPI = original })
	}

	t.Run("moves a closed task to draft", func(t *testing.T) {
		factory := newFactory(t)
		order := closedOrder(t, factory, "Failed ship")

		resp, err := SendWorkOrderToBacklog(ctx, IntakeDependencies{}, orgID, &pb.SendWorkOrderToBacklogRequest{
			FactoryId: factory.ID.String(),
			OrderId:   order.ID.String(),
		})
		require.NoError(t, err)
		require.NotNil(t, resp.GetOrder())
		assert.Equal(t, pb.WorkOrder_STATE_DRAFT, resp.GetOrder().GetState())
		assert.Equal(t, pb.WorkOrder_RESULT_UNSPECIFIED, resp.GetOrder().GetResult())

		reloaded, err := factory.FindWorkOrder(db, order.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryWorkOrderStateDraft, reloaded.State)
		assert.Equal(t, "", reloaded.Result)
	})

	t.Run("rejects an open task", func(t *testing.T) {
		factory := newFactory(t)
		order, err := factory.CreateWorkOrder(db, "Still open", "", &r.User, nil, nil)
		require.NoError(t, err)

		_, err = SendWorkOrderToBacklog(ctx, IntakeDependencies{}, orgID, &pb.SendWorkOrderToBacklogRequest{
			FactoryId: factory.ID.String(),
			OrderId:   order.ID.String(),
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
	})

	t.Run("closes a GitHub pull request then moves the task", func(t *testing.T) {
		factory := newFactory(t)
		order := closedOrder(t, factory, "With PR")
		_, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
			URL:   "https://github.com/acme/app/pull/42",
			State: models.FactoryPullRequestStateOpen,
		})
		require.NoError(t, err)

		api := &fakeFactoryGitHub{}
		useGitHub(t, api, nil)

		resp, err := SendWorkOrderToBacklog(ctx, IntakeDependencies{}, orgID, &pb.SendWorkOrderToBacklogRequest{
			FactoryId:         factory.ID.String(),
			OrderId:           order.ID.String(),
			ClosePullRequests: true,
		})
		require.NoError(t, err)
		assert.Equal(t, pb.WorkOrder_STATE_DRAFT, resp.GetOrder().GetState())
		assert.Equal(t, 1, api.editCalls)

		grouped, err := models.ListPullRequestsByWorkOrderIDs(db, []uuid.UUID{order.ID})
		require.NoError(t, err)
		require.Len(t, grouped[order.ID], 1)
		assert.Equal(t, models.FactoryPullRequestStateClosed, grouped[order.ID][0].State)
	})

	t.Run("leaves the task closed when GitHub refuses the close", func(t *testing.T) {
		factory := newFactory(t)
		order := closedOrder(t, factory, "Blocked PR")
		_, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
			URL:   "https://github.com/acme/app/pull/43",
			State: models.FactoryPullRequestStateOpen,
		})
		require.NoError(t, err)

		api := &fakeFactoryGitHub{editErr: errors.New("not allowed")}
		useGitHub(t, api, nil)

		_, err = SendWorkOrderToBacklog(ctx, IntakeDependencies{}, orgID, &pb.SendWorkOrderToBacklogRequest{
			FactoryId:         factory.ID.String(),
			OrderId:           order.ID.String(),
			ClosePullRequests: true,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)

		reloaded, err := factory.FindWorkOrder(db, order.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryWorkOrderStateClosed, reloaded.State)
	})

	t.Run("fails a Bitbucket pull request close without moving the task", func(t *testing.T) {
		factory := newFactory(t)
		order := closedOrder(t, factory, "Bitbucket PR")
		_, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
			Provider: models.FactoryPullRequestProviderBitbucket,
			URL:      "https://bitbucket.org/acme/app/pull-requests/7",
			State:    models.FactoryPullRequestStateOpen,
		})
		require.NoError(t, err)

		_, err = SendWorkOrderToBacklog(ctx, IntakeDependencies{}, orgID, &pb.SendWorkOrderToBacklogRequest{
			FactoryId:         factory.ID.String(),
			OrderId:           order.ID.String(),
			ClosePullRequests: true,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)

		reloaded, err := factory.FindWorkOrder(db, order.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryWorkOrderStateClosed, reloaded.State)
	})

	t.Run("clears artifacts when requested", func(t *testing.T) {
		factory := newFactory(t)
		order := closedOrder(t, factory, "With artifacts")
		_, err := order.CreateArtifact(db, models.FactoryWorkOrderArtifactParams{
			Type:      models.FactoryWorkOrderArtifactTypeMarkdown,
			Data:      map[string]any{"title": "plan.md", "body": "# Plan"},
			CreatedBy: &r.User,
		})
		require.NoError(t, err)

		resp, err := SendWorkOrderToBacklog(ctx, IntakeDependencies{}, orgID, &pb.SendWorkOrderToBacklogRequest{
			FactoryId:      factory.ID.String(),
			OrderId:        order.ID.String(),
			ClearArtifacts: true,
		})
		require.NoError(t, err)
		assert.Equal(t, pb.WorkOrder_STATE_DRAFT, resp.GetOrder().GetState())

		artifacts, err := order.ListArtifacts(db)
		require.NoError(t, err)
		assert.Empty(t, artifacts)

		events, err := order.ListEvents(db, 20, nil)
		require.NoError(t, err)
		found := false
		for _, event := range events {
			if event.Type == factoryevents.EventTypeOrderArtifactsCleared {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("stamps a closed GitHub pull request when a later close fails", func(t *testing.T) {
		factory := newFactory(t)
		order := closedOrder(t, factory, "Two PRs")
		_, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
			URL:   "https://github.com/acme/app/pull/45",
			State: models.FactoryPullRequestStateOpen,
		})
		require.NoError(t, err)
		_, err = order.CreatePullRequest(db, models.FactoryPullRequestParams{
			URL:   "https://github.com/acme/app/pull/46",
			State: models.FactoryPullRequestStateOpen,
		})
		require.NoError(t, err)

		api := &fakeFactoryGitHub{editErr: errors.New("second refused"), editFailAfter: 1}
		useGitHub(t, api, nil)

		_, err = SendWorkOrderToBacklog(ctx, IntakeDependencies{}, orgID, &pb.SendWorkOrderToBacklogRequest{
			FactoryId:         factory.ID.String(),
			OrderId:           order.ID.String(),
			ClosePullRequests: true,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, 2, api.editCalls)

		reloaded, err := factory.FindWorkOrder(db, order.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryWorkOrderStateClosed, reloaded.State)

		grouped, err := models.ListPullRequestsByWorkOrderIDs(db, []uuid.UUID{order.ID})
		require.NoError(t, err)
		require.Len(t, grouped[order.ID], 2)
		states := []string{grouped[order.ID][0].State, grouped[order.ID][1].State}
		assert.ElementsMatch(t, []string{
			models.FactoryPullRequestStateClosed,
			models.FactoryPullRequestStateOpen,
		}, states)
	})

	t.Run("does not undo a concurrent reopen", func(t *testing.T) {
		factory := newFactory(t)
		order := closedOrder(t, factory, "Reopened during close")
		_, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
			URL:   "https://github.com/acme/app/pull/47",
			State: models.FactoryPullRequestStateOpen,
		})
		require.NoError(t, err)

		api := &fakeFactoryGitHub{onEdit: func() {
			_, reopenErr := order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
				ToState: models.FactoryWorkOrderStateOpen,
				Actor:   &r.User,
			})
			require.NoError(t, reopenErr)
		}}
		useGitHub(t, api, nil)

		_, err = SendWorkOrderToBacklog(ctx, IntakeDependencies{}, orgID, &pb.SendWorkOrderToBacklogRequest{
			FactoryId:         factory.ID.String(),
			OrderId:           order.ID.String(),
			ClosePullRequests: true,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)

		reloaded, err := factory.FindWorkOrder(db, order.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryWorkOrderStateOpen, reloaded.State)

		grouped, err := models.ListPullRequestsByWorkOrderIDs(db, []uuid.UUID{order.ID})
		require.NoError(t, err)
		require.Len(t, grouped[order.ID], 1)
		assert.Equal(t, models.FactoryPullRequestStateClosed, grouped[order.ID][0].State)
	})

	t.Run("leaves pull requests and artifacts when flags are off", func(t *testing.T) {
		factory := newFactory(t)
		order := closedOrder(t, factory, "Keep data")
		_, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
			URL:   "https://github.com/acme/app/pull/44",
			State: models.FactoryPullRequestStateOpen,
		})
		require.NoError(t, err)
		_, err = order.CreateArtifact(db, models.FactoryWorkOrderArtifactParams{
			Type:      models.FactoryWorkOrderArtifactTypeLink,
			Data:      map[string]any{"url": "https://example.com", "title": "Preview"},
			CreatedBy: &r.User,
		})
		require.NoError(t, err)

		resp, err := SendWorkOrderToBacklog(ctx, IntakeDependencies{}, orgID, &pb.SendWorkOrderToBacklogRequest{
			FactoryId: factory.ID.String(),
			OrderId:   order.ID.String(),
		})
		require.NoError(t, err)
		assert.Equal(t, pb.WorkOrder_STATE_DRAFT, resp.GetOrder().GetState())

		grouped, err := models.ListPullRequestsByWorkOrderIDs(db, []uuid.UUID{order.ID})
		require.NoError(t, err)
		require.Len(t, grouped[order.ID], 1)
		assert.Equal(t, models.FactoryPullRequestStateOpen, grouped[order.ID][0].State)

		artifacts, err := order.ListArtifacts(db)
		require.NoError(t, err)
		require.Len(t, artifacts, 1)
	})
}
