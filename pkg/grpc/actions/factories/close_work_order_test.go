package factories

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

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

func TestCloseWorkOrder_LockQueriesUseTheReservedConnection(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	err := withFactoryWorkOrderLock(ctx, id, func(locked *gorm.DB) error {
		var queryPID int
		if err := locked.Raw("SELECT pg_backend_pid()").Scan(&queryPID).Error; err != nil {
			return err
		}
		var lockPID int
		if err := locked.Raw(`
			SELECT l.pid
			FROM pg_locks l
			WHERE l.locktype = 'advisory'
			  AND l.granted
			  AND l.objsubid = 1
			  AND (
			    ('x' || lpad(to_hex(l.classid::bigint), 8, '0') || lpad(to_hex(l.objid::bigint), 8, '0'))::bit(64)::bigint
			  ) = hashtextextended(?, 0)
		`, models.FactoryWorkOrderLockKey(id)).Scan(&lockPID).Error; err != nil {
			return err
		}
		if queryPID == 0 || queryPID != lockPID {
			return fmt.Errorf("locked query pid %d does not match lock pid %d", queryPID, lockPID)
		}
		return nil
	})
	require.NoError(t, err)
}

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

func TestCloseWorkOrder_ReopenBeforePullRequestCloseLeavesPullRequestOpen(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factory := closeWorkOrderFactory(t, r)
	order := openWorkOrder(t, db, factory, r.User, "Reopen before pull request close")
	createWorkOrderPullRequest(t, db, order, "https://github.com/acme/app/pull/42", models.FactoryPullRequestStateOpen)

	stale := *order
	_, err := order.Close(db, models.FactoryWorkOrderResultManualRejected, &r.User)
	require.NoError(t, err)
	stale.State = models.FactoryWorkOrderStateClosed
	stale.Result = models.FactoryWorkOrderResultManualRejected
	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateOpen,
		Actor:   &r.User,
	})
	require.NoError(t, err)

	api := &fakeFactoryGitHub{}
	useCloseWorkOrderGitHub(t, api, nil)
	require.NoError(t, closePullRequestsForRejectedWorkOrder(ctx, IntakeDependencies{}, factory, &stale))
	assert.Equal(t, 0, api.editCalls)

	reloaded, err := factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateOpen, reloaded.State)
	assert.Empty(t, reloaded.Result)
	assertPullRequestStates(t, db, order.ID, map[int64]string{
		42: models.FactoryPullRequestStateOpen,
	})
}

func TestCloseWorkOrder_ReopenDuringPullRequestCloseStaysOpen(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factory := closeWorkOrderFactory(t, r)
	order := openWorkOrder(t, db, factory, r.User, "Reopen during pull request close")
	createWorkOrderPullRequest(t, db, order, "https://github.com/acme/app/pull/42", models.FactoryPullRequestStateOpen)
	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		"github",
		support.RandomName("github"),
		map[string]any{},
	)
	require.NoError(t, err)

	gitEntered := make(chan struct{})
	releaseGit := make(chan struct{})
	var once sync.Once
	api := &fakeFactoryGitHub{
		onEdit: func() {
			once.Do(func() { close(gitEntered) })
			<-releaseGit
		},
	}
	useCloseWorkOrderGitHub(t, api, nil)

	closeDone := make(chan error, 1)
	go func() {
		_, closeErr := CloseWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.CloseWorkOrderRequest{
			FactoryId: factory.ID.String(),
			OrderId:   order.ID.String(),
			Result:    pb.WorkOrder_RESULT_REJECTED,
		})
		closeDone <- closeErr
	}()

	select {
	case <-gitEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("pull request close did not start")
	}

	reopenDone := make(chan error, 1)
	go func() {
		_, reopenErr := UpdateWorkOrderStatus(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderStatusRequest{
			FactoryId: factory.ID.String(),
			OrderId:   order.ID.String(),
			State:     pb.WorkOrder_STATE_OPEN,
		})
		reopenDone <- reopenErr
	}()

	require.Eventually(t, func() bool {
		waiting, waitErr := factoryWorkOrderLockWaiting(db, order.ID)
		return waitErr == nil && waiting
	}, 2*time.Second, 20*time.Millisecond)

	close(releaseGit)
	require.NoError(t, <-closeDone)
	require.NoError(t, <-reopenDone)

	RefreshFactoryPullRequestMergeabilityFromGitHubEvent(
		t.Context(),
		IntakeDependencies{},
		&models.Webhook{AppInstallationID: &integration.ID},
		"pull_request",
		[]byte(fmt.Sprintf(`{
			"action": "closed",
			"repository": {"full_name": "acme/app"},
			"sender": {"login": "alex", "html_url": "https://github.com/alex", "name": "Alex Rivera"},
			"pull_request": {"number": 42, "merged": false, "merged_at": null, "closed_at": %q}
		}`, time.Now().UTC().Format(time.RFC3339))),
	)

	reloaded, err := factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateOpen, reloaded.State)
	assert.Empty(t, reloaded.Result)
	assert.Equal(t, 1, api.editCalls)
	assertPullRequestStates(t, db, order.ID, map[int64]string{
		42: models.FactoryPullRequestStateClosed,
	})
}

func factoryWorkOrderLockWaiting(db *gorm.DB, id uuid.UUID) (bool, error) {
	var waiting int
	err := db.Raw(`
		SELECT COUNT(*)
		FROM pg_locks
		WHERE locktype = 'advisory'
		  AND NOT granted
		  AND objsubid = 1
		  AND (
		    ('x' || lpad(to_hex(classid::bigint), 8, '0') || lpad(to_hex(objid::bigint), 8, '0'))::bit(64)::bigint
		  ) = hashtextextended(?, 0)
	`, models.FactoryWorkOrderLockKey(id)).Scan(&waiting).Error
	return waiting > 0, err
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
