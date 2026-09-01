package factories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	canvasespb "github.com/superplanehq/superplane/pkg/protos/canvases"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__FactoryPullRequestActions(t *testing.T) {
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

	createOrder := func(t *testing.T, factory *models.Factory, title string) *models.FactoryWorkOrder {
		t.Helper()
		order, err := factory.CreateWorkOrder(db, title, "", &r.User, nil, nil)
		require.NoError(t, err)
		return order
	}

	createPR := func(t *testing.T, factory *models.Factory, req *pb.CreateFactoryPullRequestRequest) *pb.FactoryPullRequest {
		t.Helper()
		req.FactoryId = factory.ID.String()
		resp, err := CreateFactoryPullRequest(ctx, orgID, req)
		require.NoError(t, err)
		require.NotNil(t, resp.GetPullRequest())
		return resp.GetPullRequest()
	}

	t.Run("lists all factory pull requests sorted by work order number", func(t *testing.T) {
		factory := newFactory(t)
		first := createOrder(t, factory, "First")
		second := createOrder(t, factory, "Second")
		require.Less(t, first.Number, second.Number)

		later := createPR(t, factory, &pb.CreateFactoryPullRequestRequest{
			WorkOrderId: first.ID.String(),
			Url:         "https://github.com/acme/app/pull/2",
			Title:       "Later",
		})
		earlierOnSecond := createPR(t, factory, &pb.CreateFactoryPullRequestRequest{
			WorkOrderId: second.ID.String(),
			Url:         "https://github.com/acme/app/pull/1",
			Title:       "On later order",
		})

		resp, err := ListFactoryPullRequests(ctx, orgID, &pb.ListFactoryPullRequestsRequest{
			FactoryId: factory.ID.String(),
		})
		require.NoError(t, err)
		require.Len(t, resp.GetPullRequests(), 2)
		assert.Equal(t, later.GetId(), resp.GetPullRequests()[0].GetId())
		assert.Equal(t, first.Number, resp.GetPullRequests()[0].GetWorkOrderNumber())
		assert.Equal(t, earlierOnSecond.GetId(), resp.GetPullRequests()[1].GetId())
		assert.Equal(t, second.Number, resp.GetPullRequests()[1].GetWorkOrderNumber())
	})

	t.Run("filters by work order number", func(t *testing.T) {
		factory := newFactory(t)
		order := createOrder(t, factory, "Tracked")
		other := createOrder(t, factory, "Other")
		wanted := createPR(t, factory, &pb.CreateFactoryPullRequestRequest{
			WorkOrderId: order.ID.String(),
			Url:         "https://github.com/acme/app/pull/12",
		})
		_ = createPR(t, factory, &pb.CreateFactoryPullRequestRequest{
			WorkOrderId: other.ID.String(),
			Url:         "https://github.com/acme/app/pull/13",
		})

		orderNumber := order.Number
		resp, err := ListFactoryPullRequests(ctx, orgID, &pb.ListFactoryPullRequestsRequest{
			FactoryId: factory.ID.String(),
			Order:     &orderNumber,
		})
		require.NoError(t, err)
		require.Len(t, resp.GetPullRequests(), 1)
		assert.Equal(t, wanted.GetId(), resp.GetPullRequests()[0].GetId())
	})

	t.Run("filters by work order ids", func(t *testing.T) {
		factory := newFactory(t)
		first := createOrder(t, factory, "First")
		second := createOrder(t, factory, "Second")
		ignored := createOrder(t, factory, "Ignored")
		_ = createPR(t, factory, &pb.CreateFactoryPullRequestRequest{
			WorkOrderId: first.ID.String(),
			Url:         "https://github.com/acme/app/pull/21",
		})
		_ = createPR(t, factory, &pb.CreateFactoryPullRequestRequest{
			WorkOrderId: second.ID.String(),
			Url:         "https://github.com/acme/app/pull/22",
		})
		_ = createPR(t, factory, &pb.CreateFactoryPullRequestRequest{
			WorkOrderId: ignored.ID.String(),
			Url:         "https://github.com/acme/app/pull/23",
		})

		resp, err := ListFactoryPullRequests(ctx, orgID, &pb.ListFactoryPullRequestsRequest{
			FactoryId:    factory.ID.String(),
			WorkOrderIds: []string{second.ID.String(), first.ID.String()},
		})
		require.NoError(t, err)
		require.Len(t, resp.GetPullRequests(), 2)
		assert.Equal(t, first.Number, resp.GetPullRequests()[0].GetWorkOrderNumber())
		assert.Equal(t, second.Number, resp.GetPullRequests()[1].GetWorkOrderNumber())
	})

	t.Run("rejects conflicting filters", func(t *testing.T) {
		factory := newFactory(t)
		order := createOrder(t, factory, "Tracked")
		orderNumber := order.Number
		_, err := ListFactoryPullRequests(ctx, orgID, &pb.ListFactoryPullRequestsRequest{
			FactoryId:    factory.ID.String(),
			Order:        &orderNumber,
			WorkOrderIds: []string{order.ID.String()},
		})
		require.Error(t, err)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("describes a pull request with linked runs", func(t *testing.T) {
		factory := newFactory(t)
		order := createOrder(t, factory, "Tracked")
		created := createPR(t, factory, &pb.CreateFactoryPullRequestRequest{
			WorkOrderId: order.ID.String(),
			Provider:    pb.FactoryPullRequest_PROVIDER_GITHUB,
			ExternalId:  "99",
			Repository:  "acme/app",
			Number:      42,
			Url:         "https://github.com/acme/app/pull/42",
			Title:       "Fix retry",
			State:       pb.FactoryPullRequest_STATE_OPEN,
		})

		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
			nil,
		)
		rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
		run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, rootEvent)
		require.NoError(t, err)
		modelPR, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: parseUUID(t, created.GetId())})
		require.NoError(t, err)
		require.NoError(t, modelPR.LinkRun(db, run.ID, "Please add tests for the retry path."))

		resp, err := DescribeFactoryPullRequest(ctx, orgID, &pb.DescribeFactoryPullRequestRequest{
			FactoryId: factory.ID.String(),
			PrId:      created.GetId(),
		})
		require.NoError(t, err)
		described := resp.GetPullRequest()
		require.NotNil(t, described)
		assert.Equal(t, created.GetId(), described.GetId())
		assert.Equal(t, order.ID.String(), described.GetWorkOrderId())
		assert.Equal(t, order.Number, described.GetWorkOrderNumber())
		assert.Equal(t, "Fix retry", described.GetTitle())
		require.Len(t, described.GetRuns(), 1)
		linked := described.GetRuns()[0]
		require.NotNil(t, linked.GetRun())
		assert.Equal(t, run.ID.String(), linked.GetRun().GetId())
		assert.Equal(t, canvas.ID.String(), linked.GetRun().GetCanvasId())
		assert.Equal(t, canvasespb.CanvasRun_STATE_STARTED, linked.GetRun().GetState())
		assert.Equal(t, "Please add tests for the retry path.", linked.GetDescription())
	})

	t.Run("degrades to zero usage when usage rollup is unavailable", func(t *testing.T) {
		factory := newFactory(t)
		order := createOrder(t, factory, "Tracked")
		created := createPR(t, factory, &pb.CreateFactoryPullRequestRequest{
			WorkOrderId: order.ID.String(),
			Url:         "https://github.com/acme/app/pull/70",
			Title:       "Retry flaky test",
		})

		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
			nil,
		)
		rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
		run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, rootEvent)
		require.NoError(t, err)
		modelPR, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: parseUUID(t, created.GetId())})
		require.NoError(t, err)
		require.NoError(t, modelPR.LinkRun(db, run.ID, "Automated fix"))

		now := time.Now()
		require.NoError(t, db.Create(&models.WorkspaceUsageEvent{
			ID:               uuid.New(),
			OrganizationID:   r.Organization.ID,
			CanvasRunID:      run.ID,
			NodeExecutionID:  uuid.New(),
			NodeID:           "prompt",
			Provider:         models.UsageProviderAnthropic,
			Model:            "claude-sonnet-4-6",
			UsageKind:        models.UsageKindModel,
			FundingSource:    models.UsageFundingSourceBYOK,
			TotalTokens:      1_000_000,
			CostMicros:       300_000,
			Currency:         "usd",
			PriceBookVersion: "test",
			IdempotencyKey:   uuid.NewString(),
			OccurredAt:       now,
			CreatedAt:        now,
		}).Error)

		// Happy path: usage is recorded and returned as-is.
		resp, err := ListFactoryPullRequests(ctx, orgID, &pb.ListFactoryPullRequestsRequest{
			FactoryId: factory.ID.String(),
		})
		require.NoError(t, err)
		require.Len(t, resp.GetPullRequests(), 1)
		require.Len(t, resp.GetPullRequests()[0].GetRuns(), 1)
		assert.EqualValues(t, 1_000_000, resp.GetPullRequests()[0].GetRuns()[0].GetTotalTokens())
		assert.Positive(t, resp.GetPullRequests()[0].GetRuns()[0].GetCostCents())

		// Simulate the usage rollup table being unavailable mid-migration
		// (for example, during a table rename). The PR listing must still
		// succeed, degrading to zero usage instead of failing with a 500.
		require.NoError(t, db.Exec(
			"ALTER TABLE workspace_usage_events RENAME TO workspace_usage_events_test_missing",
		).Error)
		defer func() {
			require.NoError(t, db.Exec(
				"ALTER TABLE workspace_usage_events_test_missing RENAME TO workspace_usage_events",
			).Error)
		}()

		degraded, err := ListFactoryPullRequests(ctx, orgID, &pb.ListFactoryPullRequestsRequest{
			FactoryId: factory.ID.String(),
		})
		require.NoError(t, err)
		require.Len(t, degraded.GetPullRequests(), 1)
		require.Len(t, degraded.GetPullRequests()[0].GetRuns(), 1)
		assert.EqualValues(t, 0, degraded.GetPullRequests()[0].GetRuns()[0].GetTotalTokens())
		assert.EqualValues(t, 0, degraded.GetPullRequests()[0].GetRuns()[0].GetCostCents())
	})

	t.Run("updates a tracked pull request", func(t *testing.T) {
		factory := newFactory(t)
		order := createOrder(t, factory, "Tracked")
		created := createPR(t, factory, &pb.CreateFactoryPullRequestRequest{
			WorkOrderId: order.ID.String(),
			Url:         "https://github.com/acme/app/pull/50",
			Title:       "Draft",
		})

		title := "Ready"
		resp, err := UpdateFactoryPullRequest(ctx, orgID, &pb.UpdateFactoryPullRequestRequest{
			FactoryId: factory.ID.String(),
			PrId:      created.GetId(),
			Title:     &title,
			State:     pb.FactoryPullRequest_STATE_MERGED.Enum(),
		})
		require.NoError(t, err)
		assert.Equal(t, "Ready", resp.GetPullRequest().GetTitle())
		assert.Equal(t, pb.FactoryPullRequest_STATE_MERGED, resp.GetPullRequest().GetState())
		assert.NotNil(t, resp.GetPullRequest().GetMergedAt())
	})

	t.Run("does not leak pull requests across factories", func(t *testing.T) {
		factory := newFactory(t)
		other := newFactory(t)
		order := createOrder(t, factory, "Tracked")
		created := createPR(t, factory, &pb.CreateFactoryPullRequestRequest{
			WorkOrderId: order.ID.String(),
			Url:         "https://github.com/acme/app/pull/60",
		})

		_, err := DescribeFactoryPullRequest(ctx, orgID, &pb.DescribeFactoryPullRequestRequest{
			FactoryId: other.ID.String(),
			PrId:      created.GetId(),
		})
		require.Error(t, err)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)

		resp, err := ListFactoryPullRequests(ctx, orgID, &pb.ListFactoryPullRequestsRequest{
			FactoryId: other.ID.String(),
		})
		require.NoError(t, err)
		assert.Empty(t, resp.GetPullRequests())
	})
}

func parseUUID(t *testing.T, raw string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(raw)
	require.NoError(t, err)
	return id
}
