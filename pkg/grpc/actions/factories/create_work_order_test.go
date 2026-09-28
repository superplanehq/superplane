package factories

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
	"google.golang.org/grpc/metadata"
	"gorm.io/datatypes"
)

const githubIssueTestActor = "superplane-bot"

func init() {
	runManualTaskIssueBackgroundReconcile = false
}

func Test__CreateWorkOrder__AssignsTheCreator(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	other := support.CreateUser(t, r, r.Organization.ID)

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	resp, err := CreateWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
		FactoryId:   factoryModel.ID.String(),
		Title:       "Ship the refunds line",
		AssigneeIds: []string{other.ID.String()},
	})
	require.NoError(t, err)
	require.Len(t, resp.Order.Assignees, 1)
	assert.Equal(t, r.User.String(), resp.Order.Assignees[0].Id)
	assert.Equal(t, r.UserModel.Name, resp.Order.Assignees[0].Name)
	assert.Equal(t, r.User.String(), resp.Order.GetCreatedBy().GetUser().GetId())
}

func Test__CreateWorkOrder__ReparentsWorkspaceFiles(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
	t.Setenv("BASE_URL", "http://files.test")
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	file, err := models.CreatePendingFile(database.Conn(), models.CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		Filename:       "bug.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	require.NoError(t, storedfiles.CompleteUpload(ctx, database.Conn(), store, file, bytes.NewReader([]byte("png-bytes"))))

	resp, err := CreateWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
		FactoryId:   factoryModel.ID.String(),
		Title:       "Login screenshot",
		Description: "See ![bug](" + blob.FileRef(file.ID) + ")",
	})
	require.NoError(t, err)
	require.Len(t, resp.Order.Files, 1)
	assert.Equal(t, file.ID.String(), resp.Order.Files[0].Id)
	assert.NotEmpty(t, resp.Order.Files[0].DownloadUrl)
	assert.Contains(t, resp.Order.Description, blob.FileRef(file.ID))
	assert.NotContains(t, resp.Order.Description, "http://")

	reparented, err := models.FindFile(database.Conn(), file.ID)
	require.NoError(t, err)
	assert.Equal(t, blob.ScopeTask, reparented.Scope)
}

func Test__CreateWorkOrder__MirrorsManualTasksAsGitHubIssues(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	issueURL := "https://github.com/acme/payments/issues/42"

	t.Run("creates a GitHub issue and saves it as the origin", func(t *testing.T) {
		httpCtx := githubIssueHTTP(http.StatusCreated, `{
			"number": 42,
			"html_url": "https://github.com/acme/payments/issues/42",
			"title": "Ship the refunds line"
		}`)
		factoryModel := githubIssueFactory(t, r, true, false, "acme/payments")

		resp, err := CreateWorkOrder(ctx, githubIssueDeps(r, httpCtx), r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId:   factoryModel.ID.String(),
			Title:       "Ship the refunds line",
			Description: "Stop double charges.",
		})
		require.NoError(t, err)
		assert.Equal(t, issueURL, resp.Order.GetOrigin().GetUrl())
		assert.Equal(t, "acme/payments#42", resp.Order.GetOrigin().GetLabel())

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Equal(t, "/repos/acme/payments/issues", httpCtx.Requests[0].URL.Path)
		body, err := io.ReadAll(httpCtx.Requests[0].Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "Ship the refunds line")
		assert.Contains(t, string(body), "Stop double charges.")
		assert.Contains(t, string(body), "superplane-manual-task:")
		assert.Equal(t, r.User.String(), resp.Order.GetCreatedBy().GetUser().GetId())
	})

	t.Run("keeps attached files on the manual task", func(t *testing.T) {
		t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
		t.Setenv("BASE_URL", "http://files.test")
		store, err := filesystem.New(t.TempDir())
		require.NoError(t, err)
		blob.SetCurrent(store)
		t.Cleanup(func() { blob.SetCurrent(nil) })

		httpCtx := githubIssueHTTP(http.StatusCreated, `{
			"number": 42,
			"html_url": "https://github.com/acme/payments/issues/42"
		}`)
		factoryModel := githubIssueFactory(t, r, true, false, "acme/payments")
		file, err := models.CreatePendingFile(database.Conn(), models.CreateFileParams{
			Scope:          blob.ScopeWorkspace,
			OrganizationID: r.Organization.ID,
			FactoryID:      factoryModel.ID,
			Filename:       "bug.png",
			ContentType:    "image/png",
			CreatedByID:    r.User,
		})
		require.NoError(t, err)
		require.NoError(t, storedfiles.CompleteUpload(ctx, database.Conn(), store, file, bytes.NewReader([]byte("png-bytes"))))

		resp, err := CreateWorkOrder(ctx, githubIssueDeps(r, httpCtx), r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId:   factoryModel.ID.String(),
			Title:       "Login screenshot",
			Description: "See ![bug](" + blob.FileRef(file.ID) + ")",
		})
		require.NoError(t, err)
		require.Len(t, resp.Order.Files, 1)
		assert.Equal(t, file.ID.String(), resp.Order.Files[0].Id)
		assert.Equal(t, r.User.String(), resp.Order.GetCreatedBy().GetUser().GetId())
		assert.Equal(t, issueURL, resp.Order.GetOrigin().GetUrl())
	})

	t.Run("does not call GitHub when the setting is off", func(t *testing.T) {
		httpCtx := githubIssueHTTP(http.StatusCreated, `{}`)
		factoryModel := githubIssueFactory(t, r, false, false, "acme/payments")

		resp, err := CreateWorkOrder(ctx, githubIssueDeps(r, httpCtx), r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			Title:     "Ship the refunds line",
		})
		require.NoError(t, err)
		assert.Nil(t, resp.Order.GetOrigin())
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("does not call GitHub when the intake is paused", func(t *testing.T) {
		httpCtx := githubIssueHTTP(http.StatusCreated, `{}`)
		factoryModel := githubIssueFactory(t, r, true, true, "acme/payments")

		resp, err := CreateWorkOrder(ctx, githubIssueDeps(r, httpCtx), r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			Title:     "Ship the refunds line",
		})
		require.NoError(t, err)
		assert.Nil(t, resp.Order.GetOrigin())
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("creates the task without an origin when GitHub fails", func(t *testing.T) {
		httpCtx := githubIssueHTTP(http.StatusForbidden, `{"message":"Resource not accessible by integration"}`)
		factoryModel := githubIssueFactory(t, r, true, false, "acme/payments")

		resp, err := CreateWorkOrder(ctx, githubIssueDeps(r, httpCtx), r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			Title:     "Ship the refunds line",
		})
		require.NoError(t, err)
		assert.Equal(t, "Ship the refunds line", resp.Order.GetTitle())
		assert.Nil(t, resp.Order.GetOrigin())
		require.Len(t, httpCtx.Requests, 1)

		orders, err := factoryModel.ListWorkOrders(database.Conn(), models.ListFactoryWorkOrdersFilters{Limit: 10})
		require.NoError(t, err)
		require.Len(t, orders, 1)
		assert.Nil(t, orders[0].OriginURL)
		assert.Nil(t, orders[0].OriginLabel)
		require.NotNil(t, orders[0].CreatedByID)
		assert.Equal(t, r.User, *orders[0].CreatedByID)
	})

	t.Run("uses the oldest enabled intake", func(t *testing.T) {
		httpCtx := githubIssueHTTP(http.StatusCreated, `{
			"number": 7,
			"html_url": "https://github.com/acme/older/issues/7"
		}`)
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		older := time.Now().Add(-time.Hour)
		seedGitHubIssueIntake(t, r, factoryModel, true, false, "acme/older", &older)
		seedGitHubIssueIntake(t, r, factoryModel, true, false, "acme/newer", nil)

		resp, err := CreateWorkOrder(ctx, githubIssueDeps(r, httpCtx), r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			Title:     "Use the first intake",
		})
		require.NoError(t, err)
		assert.Equal(t, "https://github.com/acme/older/issues/7", resp.Order.GetOrigin().GetUrl())
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "/repos/acme/older/issues", httpCtx.Requests[0].URL.Path)
	})

	t.Run("keeps the manual creator when a late webhook already opened a task", func(t *testing.T) {
		httpCtx := githubIssueHTTP(http.StatusCreated, `{
			"number": 42,
			"html_url": "https://github.com/acme/payments/issues/42"
		}`)
		factoryModel := githubIssueFactory(t, r, true, false, "acme/payments")
		existing, err := factoryModel.CreateWorkOrderWithOrigin(
			database.Conn(),
			"Ship the refunds line",
			"",
			nil,
			nil,
			nil,
			models.WorkOrderOrigin{URL: issueURL, Label: "acme/payments#42"},
		)
		require.NoError(t, err)

		resp, err := CreateWorkOrder(ctx, githubIssueDeps(r, httpCtx), r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId:   factoryModel.ID.String(),
			Title:       "Ship the refunds line",
			Description: "Stop double charges.",
		})
		require.NoError(t, err)
		assert.NotEqual(t, existing.ID.String(), resp.Order.GetId())
		assert.Equal(t, r.User.String(), resp.Order.GetCreatedBy().GetUser().GetId())
		assert.Equal(t, issueURL, resp.Order.GetOrigin().GetUrl())

		saved, err := factoryModel.FindWorkOrder(database.Conn(), uuid.MustParse(resp.Order.GetId()))
		require.NoError(t, err)
		require.NotNil(t, saved.CreatedByID)
		assert.Equal(t, r.User, *saved.CreatedByID)
		require.NotNil(t, saved.OriginURL)
		assert.Equal(t, issueURL, *saved.OriginURL)
	})

	t.Run("does not open a GitHub issue when the task save fails", func(t *testing.T) {
		httpCtx := githubIssueHTTP(http.StatusCreated, `{
			"number": 42,
			"html_url": "https://github.com/acme/payments/issues/42"
		}`)
		factoryModel := githubIssueFactory(t, r, true, false, "acme/payments")

		_, err := CreateWorkOrder(ctx, githubIssueDeps(r, httpCtx), r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId:   factoryModel.ID.String(),
			Title:       "Ship the refunds line",
			Description: "See ![bug](" + blob.FileRef(uuid.New()) + ")",
		})
		require.Error(t, err)

		orders, err := factoryModel.ListWorkOrders(database.Conn(), models.ListFactoryWorkOrdersFilters{Limit: 10})
		require.NoError(t, err)
		assert.Empty(t, orders)
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("recovers the issue when the create response is late", func(t *testing.T) {
		httpCtx := &lateIssueHTTP{found: true}
		factoryModel := githubIssueFactory(t, r, true, false, "acme/payments")

		resp, err := CreateWorkOrder(ctx, githubIssueDeps(r, httpCtx), r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId:   factoryModel.ID.String(),
			Title:       "Ship the refunds line",
			Description: "Stop double charges.",
		})
		require.NoError(t, err)
		assert.Equal(t, issueURL, resp.Order.GetOrigin().GetUrl())
		assert.Equal(t, r.User.String(), resp.Order.GetCreatedBy().GetUser().GetId())
		require.Len(t, httpCtx.requests, 2)
		assert.Equal(t, http.MethodPost, httpCtx.requests[0].Method)
		assert.Equal(t, http.MethodGet, httpCtx.requests[1].Method)
		assert.Equal(t, githubIssueTestActor, httpCtx.requests[1].URL.Query().Get("creator"))
		assert.Equal(t, "100", httpCtx.requests[1].URL.Query().Get("per_page"))

		orders, err := factoryModel.ListWorkOrders(database.Conn(), models.ListFactoryWorkOrdersFilters{Limit: 10})
		require.NoError(t, err)
		require.Len(t, orders, 1)
	})

	t.Run("keeps a marker when a timed-out create cannot be confirmed", func(t *testing.T) {
		httpCtx := &lateIssueHTTP{found: false}
		factoryModel := githubIssueFactory(t, r, true, false, "acme/payments")

		resp, err := CreateWorkOrder(ctx, githubIssueDeps(r, httpCtx), r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			Title:     "Ship the refunds line",
		})
		require.NoError(t, err)
		assert.Nil(t, resp.Order.GetOrigin())
		assert.Equal(t, r.User.String(), resp.Order.GetCreatedBy().GetUser().GetId())

		orders, err := factoryModel.ListWorkOrders(database.Conn(), models.ListFactoryWorkOrdersFilters{Limit: 10})
		require.NoError(t, err)
		require.Len(t, orders, 1)
		assert.Nil(t, orders[0].OriginURL)
		require.NotNil(t, orders[0].OriginLabel)
		assert.Contains(t, *orders[0].OriginLabel, "superplane-manual-task:")
	})

	t.Run("bounds the GitHub call", func(t *testing.T) {
		httpCtx := githubIssueHTTP(http.StatusCreated, `{
			"number": 42,
			"html_url": "https://github.com/acme/payments/issues/42"
		}`)
		recorder := &deadlineHTTP{inner: httpCtx}
		factoryModel := githubIssueFactory(t, r, true, false, "acme/payments")

		_, err := CreateWorkOrder(ctx, githubIssueDeps(r, recorder), r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			Title:     "Ship the refunds line",
		})
		require.NoError(t, err)
		require.True(t, recorder.hasDeadline)
		remaining := time.Until(recorder.deadline)
		assert.LessOrEqual(t, remaining, manualTaskGitHubIssueTimeout)
		assert.Greater(t, remaining, manualTaskGitHubIssueTimeout-2*time.Second)
	})

	t.Run("links the issue when the first search misses", func(t *testing.T) {
		httpCtx := &lateIssueHTTP{found: true, missesBeforeFound: 1}
		factoryModel := githubIssueFactory(t, r, true, false, "acme/payments")

		resp, err := CreateWorkOrder(ctx, githubIssueDeps(r, httpCtx), r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			Title:     "Ship the refunds line",
		})
		require.NoError(t, err)
		assert.Equal(t, issueURL, resp.Order.GetOrigin().GetUrl())
		assert.GreaterOrEqual(t, httpCtx.listCalls, 2)
	})

	t.Run("links a late issue without the intake webhook", func(t *testing.T) {
		previousAttempts := manualTaskGitHubIssueReconcileAttempts
		previousDelay := manualTaskIssueBackgroundDelay
		previousRuns := runManualTaskIssueBackgroundReconcile
		manualTaskGitHubIssueReconcileAttempts = 1
		manualTaskIssueBackgroundDelay = 10 * time.Millisecond
		runManualTaskIssueBackgroundReconcile = true
		t.Cleanup(func() {
			manualTaskGitHubIssueReconcileAttempts = previousAttempts
			manualTaskIssueBackgroundDelay = previousDelay
			runManualTaskIssueBackgroundReconcile = previousRuns
		})

		httpCtx := &lateIssueHTTP{found: true, missesBeforeFound: 1}
		factoryModel := githubIssueFactory(t, r, true, false, "acme/payments")

		resp, err := CreateWorkOrder(ctx, githubIssueDeps(r, httpCtx), r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			Title:     "Ship the refunds line",
		})
		require.NoError(t, err)
		assert.Nil(t, resp.Order.GetOrigin())

		require.Eventually(t, func() bool {
			orders, listErr := factoryModel.ListWorkOrders(database.Conn(), models.ListFactoryWorkOrdersFilters{Limit: 10})
			if listErr != nil || len(orders) != 1 || orders[0].OriginURL == nil {
				return false
			}
			return *orders[0].OriginURL == issueURL
		}, 2*time.Second, 20*time.Millisecond)
	})

	t.Run("returns the saved task when the client cancels during GitHub issue creation", func(t *testing.T) {
		callCtx, cancel := context.WithCancel(ctx)
		httpCtx := &lateIssueHTTP{found: true, cancel: cancel}
		factoryModel := githubIssueFactory(t, r, true, false, "acme/payments")

		resp, err := CreateWorkOrder(callCtx, githubIssueDeps(r, httpCtx), r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			Title:     "Ship the refunds line",
		})
		require.NoError(t, err)
		require.NotEmpty(t, resp.Order.GetId())
		assert.Equal(t, issueURL, resp.Order.GetOrigin().GetUrl())

		orders, err := factoryModel.ListWorkOrders(database.Conn(), models.ListFactoryWorkOrdersFilters{Limit: 10})
		require.NoError(t, err)
		require.Len(t, orders, 1)
	})

	t.Run("creates another task when the same title is submitted again", func(t *testing.T) {
		httpCtx := githubIssueResponses(
			githubIssueResponse(http.StatusCreated, `{
				"number": 42,
				"html_url": "https://github.com/acme/payments/issues/42"
			}`),
			githubIssueResponse(http.StatusCreated, `{
				"number": 43,
				"html_url": "https://github.com/acme/payments/issues/43"
			}`),
		)
		factoryModel := githubIssueFactory(t, r, true, false, "acme/payments")
		request := &pb.CreateWorkOrderRequest{
			FactoryId:   factoryModel.ID.String(),
			Title:       "Ship the refunds line",
			Description: "Stop double charges.",
			AssigneeIds: []string{r.User.String()},
		}

		first, err := CreateWorkOrder(ctx, githubIssueDeps(r, httpCtx), r.Organization.ID.String(), request)
		require.NoError(t, err)
		secondRequest := &pb.CreateWorkOrderRequest{
			FactoryId:   request.FactoryId,
			Title:       request.Title,
			Description: request.Description,
			AssigneeIds: []string{uuid.NewString()},
		}
		second, err := CreateWorkOrder(ctx, githubIssueDeps(r, httpCtx), r.Organization.ID.String(), secondRequest)
		require.NoError(t, err)
		assert.NotEqual(t, first.Order.GetId(), second.Order.GetId())

		orders, err := factoryModel.ListWorkOrders(database.Conn(), models.ListFactoryWorkOrdersFilters{Limit: 10})
		require.NoError(t, err)
		require.Len(t, orders, 2)
		assert.Equal(t, 2, countGitHubIssueCreates(httpCtx.Requests))
	})

	t.Run("keeps the marker when the GitHub login cannot be read", func(t *testing.T) {
		httpCtx := githubIssueHTTP(http.StatusCreated, `{
			"number": 42,
			"html_url": "https://github.com/acme/payments/issues/42"
		}`)
		factoryModel := githubIssueFactory(t, r, true, false, "acme/payments")
		deps := githubIssueDeps(r, httpCtx)
		deps.HTTP = &flakyActorRouter{inner: httpCtx, login: githubIssueTestActor, failures: manualTaskGitHubActorAttempts}

		resp, err := CreateWorkOrder(ctx, deps, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			Title:     "Ship the refunds line",
		})
		require.NoError(t, err)
		assert.Nil(t, resp.Order.GetOrigin())
		assert.Empty(t, httpCtx.Requests)

		orders, err := factoryModel.ListWorkOrders(database.Conn(), models.ListFactoryWorkOrdersFilters{Limit: 10})
		require.NoError(t, err)
		require.Len(t, orders, 1)
		assert.Nil(t, orders[0].OriginURL)
		require.NotNil(t, orders[0].OriginLabel)
		assert.Contains(t, *orders[0].OriginLabel, "superplane-manual-task:")
	})

	t.Run("opens an issue after a later login lookup", func(t *testing.T) {
		previousDelay := manualTaskIssueBackgroundDelay
		previousAttempts := manualTaskIssueBackgroundAttempts
		previousRuns := runManualTaskIssueBackgroundReconcile
		manualTaskIssueBackgroundDelay = 10 * time.Millisecond
		manualTaskIssueBackgroundAttempts = 4
		runManualTaskIssueBackgroundReconcile = true
		t.Cleanup(func() {
			manualTaskIssueBackgroundDelay = previousDelay
			manualTaskIssueBackgroundAttempts = previousAttempts
			runManualTaskIssueBackgroundReconcile = previousRuns
		})

		httpCtx := githubIssueResponses(
			githubIssueResponse(http.StatusOK, `[]`),
			githubIssueResponse(http.StatusCreated, `{
				"number": 42,
				"html_url": "https://github.com/acme/payments/issues/42"
			}`),
		)
		factoryModel := githubIssueFactory(t, r, true, false, "acme/payments")
		deps := githubIssueDeps(r, httpCtx)
		deps.HTTP = &flakyActorRouter{inner: httpCtx, login: githubIssueTestActor, failures: manualTaskGitHubActorAttempts}

		resp, err := CreateWorkOrder(ctx, deps, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			Title:     "Ship the refunds line",
		})
		require.NoError(t, err)
		assert.Nil(t, resp.Order.GetOrigin())

		require.Eventually(t, func() bool {
			orders, listErr := factoryModel.ListWorkOrders(database.Conn(), models.ListFactoryWorkOrdersFilters{Limit: 10})
			if listErr != nil || len(orders) != 1 || orders[0].OriginURL == nil {
				return false
			}
			return *orders[0].OriginURL == issueURL
		}, 2*time.Second, 20*time.Millisecond)
		assert.Equal(t, 1, countGitHubIssueCreates(httpCtx.Requests))
	})

	t.Run("returns the saved task when the same request key is sent again", func(t *testing.T) {
		httpCtx := githubIssueHTTP(http.StatusCreated, `{
			"number": 42,
			"html_url": "https://github.com/acme/payments/issues/42",
			"user": {"login": "superplane-bot"}
		}`)
		factoryModel := githubIssueFactory(t, r, true, false, "acme/payments")
		request := &pb.CreateWorkOrderRequest{
			FactoryId:   factoryModel.ID.String(),
			Title:       "Ship the refunds line",
			Description: "Stop double charges.",
		}
		keyed := withCreateRequestKey(ctx, r.User.String(), uuid.NewString())

		first, err := CreateWorkOrder(keyed, githubIssueDeps(r, httpCtx), r.Organization.ID.String(), request)
		require.NoError(t, err)
		assert.Equal(t, issueURL, first.Order.GetOrigin().GetUrl())
		assert.Equal(t, "acme/payments#42", first.Order.GetOrigin().GetLabel())

		second, err := CreateWorkOrder(keyed, githubIssueDeps(r, httpCtx), r.Organization.ID.String(), request)
		require.NoError(t, err)
		assert.Equal(t, first.Order.GetId(), second.Order.GetId())
		assert.Equal(t, issueURL, second.Order.GetOrigin().GetUrl())
		assert.Equal(t, "acme/payments#42", second.Order.GetOrigin().GetLabel())

		orders, err := factoryModel.ListWorkOrders(database.Conn(), models.ListFactoryWorkOrdersFilters{Limit: 10})
		require.NoError(t, err)
		require.Len(t, orders, 1)
		assert.Equal(t, 1, countGitHubIssueCreates(httpCtx.Requests))
		assert.Contains(t, *orders[0].OriginLabel, "\x1ecreate:")
	})

	t.Run("opens an issue when a later login lookup succeeds", func(t *testing.T) {
		httpCtx := githubIssueHTTP(http.StatusCreated, `{
			"number": 42,
			"html_url": "https://github.com/acme/payments/issues/42"
		}`)
		factoryModel := githubIssueFactory(t, r, true, false, "acme/payments")
		deps := githubIssueDeps(r, httpCtx)
		deps.HTTP = &flakyActorRouter{inner: httpCtx, login: githubIssueTestActor, failures: 1}

		resp, err := CreateWorkOrder(ctx, deps, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			Title:     "Ship the refunds line",
		})
		require.NoError(t, err)
		assert.Equal(t, issueURL, resp.Order.GetOrigin().GetUrl())
		require.Len(t, httpCtx.Requests, 1)
	})
}

func Test__CreateWorkOrder__SkipsBrokenManualTaskIntakes(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	assertUsesNewerIntake := func(t *testing.T, factoryModel *models.Factory, httpCtx *contexts.HTTPContext) {
		t.Helper()
		resp, err := CreateWorkOrder(ctx, githubIssueDeps(r, httpCtx), r.Organization.ID.String(), &pb.CreateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			Title:     "Use the healthy intake",
		})
		require.NoError(t, err)
		assert.Equal(t, "https://github.com/acme/newer/issues/9", resp.Order.GetOrigin().GetUrl())
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "/repos/acme/newer/issues", httpCtx.Requests[0].URL.Path)
	}

	t.Run("skips an older intake with no repository", func(t *testing.T) {
		httpCtx := githubIssueHTTP(http.StatusCreated, `{
			"number": 9,
			"html_url": "https://github.com/acme/newer/issues/9"
		}`)
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		older := time.Now().Add(-time.Hour)
		seedGitHubIssueIntake(t, r, factoryModel, true, false, "", &older)
		seedGitHubIssueIntake(t, r, factoryModel, true, false, "acme/newer", nil)
		assertUsesNewerIntake(t, factoryModel, httpCtx)
	})

	t.Run("skips an older intake whose integration is not ready", func(t *testing.T) {
		httpCtx := githubIssueHTTP(http.StatusCreated, `{
			"number": 9,
			"html_url": "https://github.com/acme/newer/issues/9"
		}`)
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		older := time.Now().Add(-time.Hour)
		_, integration := seedGitHubIssueIntake(t, r, factoryModel, true, false, "acme/older", &older)
		integration.State = models.IntegrationStateError
		require.NoError(t, database.Conn().Save(integration).Error)
		seedGitHubIssueIntake(t, r, factoryModel, true, false, "acme/newer", nil)
		assertUsesNewerIntake(t, factoryModel, httpCtx)
	})
}

type flakyActorRouter struct {
	inner    core.HTTPContext
	login    string
	failures int
	calls    int
}

func (r *flakyActorRouter) Do(request *http.Request) (*http.Response, error) {
	if request.URL.Path == "/user" {
		r.calls++
		if r.calls <= r.failures {
			return githubIssueResponse(http.StatusInternalServerError, `{"message":"unavailable"}`), nil
		}
		return githubIssueResponse(http.StatusOK, `{"login":"`+r.login+`"}`), nil
	}
	return r.inner.Do(request)
}

func githubIssueDeps(r *support.ResourceRegistry, httpCtx core.HTTPContext) IntakeDependencies {
	return IntakeDependencies{
		Registry:       r.Registry,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		WebhookBaseURL: "http://localhost:8000",
		HTTP: &githubActorRouter{
			inner: httpCtx,
			login: githubIssueTestActor,
		},
	}
}

type githubActorRouter struct {
	inner core.HTTPContext
	login string
}

func (r *githubActorRouter) Do(request *http.Request) (*http.Response, error) {
	if request.URL.Path == "/user" {
		return githubIssueResponse(http.StatusOK, `{"login":"`+r.login+`"}`), nil
	}
	return r.inner.Do(request)
}

func withCreateRequestKey(ctx context.Context, userID, key string) context.Context {
	return metadata.NewIncomingContext(ctx, metadata.Pairs(
		"x-user-id", userID,
		"idempotency-key", key,
	))
}

func countGitHubIssueCreates(requests []*http.Request) int {
	count := 0
	for _, request := range requests {
		if request.Method == http.MethodPost && strings.Contains(request.URL.Path, "/issues") {
			count++
		}
	}
	return count
}

func githubIssueHTTP(status int, body string) *contexts.HTTPContext {
	return githubIssueResponses(githubIssueResponse(status, body))
}

func githubIssueResponses(responses ...*http.Response) *contexts.HTTPContext {
	return &contexts.HTTPContext{Responses: responses}
}

func githubIssueResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

type deadlineHTTP struct {
	inner       *contexts.HTTPContext
	deadline    time.Time
	hasDeadline bool
}

func (d *deadlineHTTP) Do(request *http.Request) (*http.Response, error) {
	d.deadline, d.hasDeadline = request.Context().Deadline()
	return d.inner.Do(request)
}

type lateIssueHTTP struct {
	found             bool
	missesBeforeFound int
	listCalls         int
	cancel            context.CancelFunc
	requests          []*http.Request
	marker            string
	mu                sync.Mutex
}

func (h *lateIssueHTTP) Do(request *http.Request) (*http.Response, error) {
	var body []byte
	if request.Body != nil {
		var err error
		body, err = io.ReadAll(request.Body)
		if err != nil {
			return nil, err
		}
		request.Body = io.NopCloser(bytes.NewReader(body))
	}
	h.mu.Lock()
	h.requests = append(h.requests, request)
	h.mu.Unlock()
	if request.Method == http.MethodPost {
		h.mu.Lock()
		h.marker = string(body)
		h.mu.Unlock()
		if h.cancel != nil {
			h.cancel()
		}
		return nil, context.DeadlineExceeded
	}

	h.mu.Lock()
	h.listCalls++
	listCalls := h.listCalls
	marker := h.marker
	h.mu.Unlock()
	if !h.found || listCalls <= h.missesBeforeFound {
		return githubIssueResponse(http.StatusOK, `[]`), nil
	}
	payload := `[{
		"number": 99,
		"html_url": "https://github.com/acme/payments/issues/99",
		"body": ` + strconv.Quote(marker) + `,
		"user": {"login": "attacker"}
	},{
		"number": 42,
		"html_url": "https://github.com/acme/payments/issues/42",
		"body": ` + strconv.Quote(marker) + `,
		"user": {"login": "` + githubIssueTestActor + `"}
	}]`
	return githubIssueResponse(http.StatusOK, payload), nil
}

func githubIssueFactory(
	t *testing.T,
	r *support.ResourceRegistry,
	createIssue bool,
	paused bool,
	repository string,
) *models.Factory {
	t.Helper()

	factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	seedGitHubIssueIntake(t, r, factoryModel, createIssue, paused, repository, nil)
	return factoryModel
}

func seedGitHubIssueIntake(
	t *testing.T,
	r *support.ResourceRegistry,
	factoryModel *models.Factory,
	createIssue bool,
	paused bool,
	repository string,
	createdAt *time.Time,
) (*models.FactoryIntake, *models.Integration) {
	t.Helper()

	db := database.Conn()
	integration := createReadyGitHubIssueIntegration(t, r)
	const triggerNodeID = "trigger"
	integrationID := integration.ID.String()
	metadata := map[string]any{intakeMetadataGitHubCreateIssueForManualTasks: createIssue}

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{
			NodeID: triggerNodeID,
			Type:   models.NodeTypeTrigger,
			Name:   "On Issue",
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "github.onIssue"},
			}),
			Configuration: datatypes.NewJSONType(map[string]any{
				"repository": repository,
				"actions":    []any{"opened"},
			}),
			Metadata: datatypes.NewJSONType(metadata),
		}},
		nil,
	)
	require.NoError(t, db.Model(canvas).Update("factory_id", factoryModel.ID).Error)

	liveVersion, err := models.FindLiveCanvasVersionInTransaction(db, canvas.ID)
	require.NoError(t, err)
	nodes := append([]models.Node(nil), liveVersion.Nodes...)
	for i := range nodes {
		if nodes[i].ID != triggerNodeID {
			continue
		}
		nodes[i].IntegrationID = &integrationID
		nodes[i].Ref = models.NodeRef{Trigger: &models.TriggerRef{Name: "github.onIssue"}}
		nodes[i].Configuration = map[string]any{"repository": repository, "actions": []any{"opened"}}
		nodes[i].Metadata = metadata
	}
	require.NoError(t, db.Model(liveVersion).Update("nodes", datatypes.NewJSONSlice(nodes)).Error)

	intake, err := factoryModel.CreateIntake(db, canvas.ID, models.FactoryIntakeSourceGitHubIssues)
	require.NoError(t, err)
	if paused {
		require.NoError(t, intake.SetPaused(db, true))
	}
	if createdAt != nil {
		require.NoError(t, db.Model(intake).Update("created_at", *createdAt).Error)
	}
	return intake, integration
}

func createReadyGitHubIssueIntegration(t *testing.T, r *support.ResourceRegistry) *models.Integration {
	t.Helper()

	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		"github",
		support.RandomName("github"),
		map[string]any{},
	)
	require.NoError(t, err)

	setup := datatypes.NewJSONType(models.SetupState{})
	integration.State = models.IntegrationStateReady
	integration.SetupState = &setup
	integration.Properties = datatypes.NewJSONSlice([]core.IntegrationPropertyDefinition{
		{Name: common.PropertyAuthMethod, Value: common.AuthMethodPAT},
		{Name: common.PropertyOwner, Value: "acme"},
		{Name: common.PropertyOwnerType, Value: common.OwnerTypeOrganization},
	})
	require.NoError(t, database.Conn().Save(integration).Error)

	encrypted, err := r.Encryptor.Encrypt(t.Context(), []byte("ghp_test"), []byte(integration.ID.String()))
	require.NoError(t, err)
	now := time.Now()
	require.NoError(t, database.Conn().Create(&models.IntegrationSecret{
		OrganizationID: integration.OrganizationID,
		InstallationID: integration.ID,
		Name:           common.SecretPAT,
		Value:          encrypted,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}).Error)

	return integration
}
