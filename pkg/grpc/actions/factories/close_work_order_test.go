package factories

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	bitbucketintegration "github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

func TestCloseWorkOrder_ClosesOpenAndDraftPullRequests(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factory := closeWorkOrderFactory(t, r)
	order := openWorkOrder(t, db, factory, r.User, "Reject with pull requests")
	createWorkOrderPullRequest(t, db, order, "https://github.com/acme/app/pull/42", models.FactoryPullRequestStateOpen)
	createWorkOrderPullRequest(t, db, order, "https://github.com/acme/app/pull/43", models.FactoryPullRequestStateDraft)
	createWorkOrderPullRequest(t, db, order, "https://github.com/acme/app/pull/44", models.FactoryPullRequestStateMerged)
	createWorkOrderPullRequest(t, db, order, "https://github.com/acme/app/pull/45", models.FactoryPullRequestStateClosed)

	api := &fakeFactoryGitHub{}
	useCloseWorkOrderGitHub(t, api, nil)

	resp, err := CloseWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.CloseWorkOrderRequest{
		FactoryId: factory.ID.String(),
		OrderId:   order.ID.String(),
		Result:    pb.WorkOrder_RESULT_REJECTED,
	})
	require.NoError(t, err)
	assert.Equal(t, pb.WorkOrder_RESULT_REJECTED, resp.GetOrder().GetResult())
	assert.Equal(t, 2, api.editCalls)

	reloaded, err := factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateClosed, reloaded.State)
	assert.Equal(t, models.FactoryWorkOrderResultManualRejected, reloaded.Result)
	assertPullRequestStates(t, db, order.ID, map[int64]string{
		42: models.FactoryPullRequestStateClosed,
		43: models.FactoryPullRequestStateClosed,
		44: models.FactoryPullRequestStateMerged,
		45: models.FactoryPullRequestStateClosed,
	})
}

func TestCloseWorkOrder_ArchiveClosesOpenPullRequest(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factory := closeWorkOrderFactory(t, r)
	order := openWorkOrder(t, db, factory, r.User, "Archive with pull request")
	createWorkOrderPullRequest(t, db, order, "https://github.com/acme/app/pull/42", models.FactoryPullRequestStateOpen)

	api := &fakeFactoryGitHub{}
	useCloseWorkOrderGitHub(t, api, nil)

	resp, err := CloseWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.CloseWorkOrderRequest{
		FactoryId: factory.ID.String(),
		OrderId:   order.ID.String(),
		Result:    pb.WorkOrder_RESULT_ARCHIVED,
	})
	require.NoError(t, err)
	assert.Equal(t, pb.WorkOrder_RESULT_ARCHIVED, resp.GetOrder().GetResult())
	assert.Equal(t, 1, api.editCalls)

	reloaded, err := factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderResultArchived, reloaded.Result)
	assertPullRequestStates(t, db, order.ID, map[int64]string{
		42: models.FactoryPullRequestStateClosed,
	})
}

func TestCloseWorkOrder_CompletedFailedAndPullRequestClosedDoNotCallGitHost(t *testing.T) {
	results := []struct {
		name   string
		result pb.WorkOrder_Result
		stored string
	}{
		{name: "completed", result: pb.WorkOrder_RESULT_COMPLETED, stored: models.FactoryWorkOrderResultCompleted},
		{name: "failed", result: pb.WorkOrder_RESULT_FAILED, stored: models.FactoryWorkOrderResultFailed},
		{name: "pull request closed", result: pb.WorkOrder_RESULT_PR_CLOSED, stored: models.FactoryWorkOrderResultPRClosed},
	}

	for _, tc := range results {
		t.Run(tc.name, func(t *testing.T) {
			r := support.Setup(t)
			ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
			db := database.DB(t.Context())
			factory := closeWorkOrderFactory(t, r)
			order := openWorkOrder(t, db, factory, r.User, tc.name)
			createWorkOrderPullRequest(t, db, order, "https://github.com/acme/app/pull/42", models.FactoryPullRequestStateOpen)

			api := &fakeFactoryGitHub{}
			useCloseWorkOrderGitHub(t, api, nil)

			_, err := CloseWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.CloseWorkOrderRequest{
				FactoryId: factory.ID.String(),
				OrderId:   order.ID.String(),
				Result:    tc.result,
			})
			require.NoError(t, err)
			assert.Equal(t, 0, api.editCalls)

			reloaded, err := factory.FindWorkOrder(db, order.ID)
			require.NoError(t, err)
			assert.Equal(t, tc.stored, reloaded.Result)
			assertPullRequestStates(t, db, order.ID, map[int64]string{
				42: models.FactoryPullRequestStateOpen,
			})
		})
	}
}

func TestCloseWorkOrder_BitbucketFactoryDeclines(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factory := bitbucketFactory(t, r)
	order := openWorkOrder(t, db, factory, r.User, "Bitbucket reject")
	_, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		Provider: models.FactoryPullRequestProviderBitbucket,
		URL:      "https://bitbucket.org/acme/widgets/pull-requests/42",
		State:    models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":42,"state":"DECLINED"}`)),
			},
		},
	}
	stubClient, err := bitbucketintegration.NewClient(
		bitbucketintegration.AuthTypeWorkspaceAccessToken,
		httpCtx,
		&contexts.IntegrationContext{Configuration: map[string]any{"token": "token"}},
	)
	require.NoError(t, err)
	restore := newFactoryBitbucketAPI
	newFactoryBitbucketAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (*bitbucketintegration.Client, error) {
		return stubClient, nil
	}
	t.Cleanup(func() { newFactoryBitbucketAPI = restore })

	resp, err := CloseWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.CloseWorkOrderRequest{
		FactoryId: factory.ID.String(),
		OrderId:   order.ID.String(),
		Result:    pb.WorkOrder_RESULT_REJECTED,
	})
	require.NoError(t, err)
	assert.Equal(t, pb.WorkOrder_RESULT_REJECTED, resp.GetOrder().GetResult())

	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
	assert.Equal(t, "/2.0/repositories/acme/widgets/pullrequests/42/decline", httpCtx.Requests[0].URL.Path)

	reloaded, err := factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderResultManualRejected, reloaded.Result)
	assertPullRequestStates(t, db, order.ID, map[int64]string{
		42: models.FactoryPullRequestStateClosed,
	})
}

func TestCloseWorkOrder_GitHostErrorKeepsResultAndRetryCloses(t *testing.T) {
	cases := []struct {
		name   string
		result pb.WorkOrder_Result
		stored string
	}{
		{name: "rejected", result: pb.WorkOrder_RESULT_REJECTED, stored: models.FactoryWorkOrderResultManualRejected},
		{name: "archived", result: pb.WorkOrder_RESULT_ARCHIVED, stored: models.FactoryWorkOrderResultArchived},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := support.Setup(t)
			ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
			db := database.DB(t.Context())
			factory := closeWorkOrderFactory(t, r)
			order := openWorkOrder(t, db, factory, r.User, tc.name)
			createWorkOrderPullRequest(t, db, order, "https://github.com/acme/app/pull/42", models.FactoryPullRequestStateOpen)

			api := &fakeFactoryGitHub{editErr: errors.New("not allowed")}
			useCloseWorkOrderGitHub(t, api, nil)

			_, err := CloseWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.CloseWorkOrderRequest{
				FactoryId: factory.ID.String(),
				OrderId:   order.ID.String(),
				Result:    tc.result,
			})
			code, message, ok := grpcerrors.HandlerStatus(err)
			require.True(t, ok)
			assert.Equal(t, codes.FailedPrecondition, code)
			assert.Equal(t, "not allowed", message)
			assert.Equal(t, 1, api.editCalls)

			reloaded, err := factory.FindWorkOrder(db, order.ID)
			require.NoError(t, err)
			assert.Equal(t, models.FactoryWorkOrderStateClosed, reloaded.State)
			assert.Equal(t, tc.stored, reloaded.Result)
			assertPullRequestStates(t, db, order.ID, map[int64]string{
				42: models.FactoryPullRequestStateOpen,
			})

			api.editErr = nil
			_, err = CloseWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.CloseWorkOrderRequest{
				FactoryId: factory.ID.String(),
				OrderId:   order.ID.String(),
				Result:    tc.result,
			})
			require.NoError(t, err)
			assert.Equal(t, 2, api.editCalls)

			reloaded, err = factory.FindWorkOrder(db, order.ID)
			require.NoError(t, err)
			assert.Equal(t, tc.stored, reloaded.Result)
			assertPullRequestStates(t, db, order.ID, map[int64]string{
				42: models.FactoryPullRequestStateClosed,
			})
		})
	}
}

func TestCloseWorkOrder_AlreadyClosedWithAnotherResultLeavesPullRequests(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factory := closeWorkOrderFactory(t, r)
	order := openWorkOrder(t, db, factory, r.User, "Already failed")
	createWorkOrderPullRequest(t, db, order, "https://github.com/acme/app/pull/42", models.FactoryPullRequestStateOpen)
	_, err := order.Close(db, models.FactoryWorkOrderResultFailed, &r.User)
	require.NoError(t, err)

	api := &fakeFactoryGitHub{}
	useCloseWorkOrderGitHub(t, api, nil)

	_, err = CloseWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.CloseWorkOrderRequest{
		FactoryId: factory.ID.String(),
		OrderId:   order.ID.String(),
		Result:    pb.WorkOrder_RESULT_REJECTED,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, api.editCalls)

	reloaded, err := factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderResultFailed, reloaded.Result)
	assertPullRequestStates(t, db, order.ID, map[int64]string{
		42: models.FactoryPullRequestStateOpen,
	})
}

func TestCloseWorkOrder_NoOpenPullRequestClosesAsToday(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factory := closeWorkOrderFactory(t, r)
	order := openWorkOrder(t, db, factory, r.User, "No pull request")

	resp, err := CloseWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.CloseWorkOrderRequest{
		FactoryId: factory.ID.String(),
		OrderId:   order.ID.String(),
		Result:    pb.WorkOrder_RESULT_REJECTED,
	})
	require.NoError(t, err)
	assert.Equal(t, pb.WorkOrder_STATE_CLOSED, resp.GetOrder().GetState())
	assert.Equal(t, pb.WorkOrder_RESULT_REJECTED, resp.GetOrder().GetResult())

	reloaded, err := factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderResultManualRejected, reloaded.Result)
}

func closeWorkOrderFactory(t *testing.T, r *support.ResourceRegistry) *models.Factory {
	t.Helper()
	factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	return factory
}

func openWorkOrder(t *testing.T, db *gorm.DB, factory *models.Factory, user uuid.UUID, title string) *models.FactoryWorkOrder {
	t.Helper()
	order, err := factory.CreateWorkOrder(db, title, "", &user, nil, nil)
	require.NoError(t, err)
	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateOpen,
		Actor:   &user,
	})
	require.NoError(t, err)
	return order
}

func createWorkOrderPullRequest(t *testing.T, db *gorm.DB, order *models.FactoryWorkOrder, url, state string) {
	t.Helper()
	_, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL:   url,
		State: state,
	})
	require.NoError(t, err)
}

func useCloseWorkOrderGitHub(t *testing.T, api factoryGitHubAPI, err error) {
	t.Helper()
	original := newFactoryGitHubAPI
	newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) {
		return api, err
	}
	t.Cleanup(func() { newFactoryGitHubAPI = original })
}

func assertPullRequestStates(t *testing.T, db *gorm.DB, orderID uuid.UUID, want map[int64]string) {
	t.Helper()
	grouped, err := models.ListPullRequestsByWorkOrderIDs(db, []uuid.UUID{orderID})
	require.NoError(t, err)
	require.Len(t, grouped[orderID], len(want))
	got := map[int64]string{}
	for _, pullRequest := range grouped[orderID] {
		got[pullRequest.Number] = pullRequest.State
	}
	assert.Equal(t, want, got)
}
