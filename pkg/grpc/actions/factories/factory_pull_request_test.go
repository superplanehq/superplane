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
		resp, err := CreateFactoryPullRequest(ctx, IntakeDependencies{}, orgID, req)
		require.NoError(t, err)
		require.NotNil(t, resp.GetPullRequest())
		return resp.GetPullRequest()
	}

	t.Run("returns pull requests on listed work orders", func(t *testing.T) {
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

		resp, err := ListWorkOrders(ctx, orgID, &pb.ListWorkOrdersRequest{
			FactoryId: factory.ID.String(),
		})
		require.NoError(t, err)
		byID := workOrdersByID(resp.GetOrders())
		require.Len(t, byID[first.ID.String()].GetPullRequests(), 1)
		assert.Equal(t, later.GetId(), byID[first.ID.String()].GetPullRequests()[0].GetId())
		assert.Equal(t, first.Number, byID[first.ID.String()].GetPullRequests()[0].GetWorkOrderNumber())
		require.Len(t, byID[second.ID.String()].GetPullRequests(), 1)
		assert.Equal(t, earlierOnSecond.GetId(), byID[second.ID.String()].GetPullRequests()[0].GetId())
		assert.Equal(t, second.Number, byID[second.ID.String()].GetPullRequests()[0].GetWorkOrderNumber())
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

	t.Run("includes usage on pull request activities", func(t *testing.T) {
		factory := newFactory(t)
		order := createOrder(t, factory, "Tracked")
		created := createPR(t, factory, &pb.CreateFactoryPullRequestRequest{
			WorkOrderId: order.ID.String(),
			Url:         "https://github.com/acme/app/pull/88",
		})

		canvas, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "fix-checks", "start")
		event := support.EmitCanvasEventForNodeWithData(t, canvas.ID, entry, "default", nil, map[string]any{"key": "value"})
		run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, event)
		require.NoError(t, err)
		modelPR, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: parseUUID(t, created.GetId())})
		require.NoError(t, err)
		require.NoError(t, modelPR.LinkRun(db, run.ID, "Fixing failed checks on d8b80c2"))
		require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
			OrganizationID:  r.Organization.ID,
			CanvasRunID:     run.ID,
			NodeExecutionID: uuid.New(),
			NodeID:          "prompt",
			Provider:        models.UsageProviderAnthropic,
			Model:           "claude-sonnet-4-6",
			InputTokens:     1_000_000,
			TotalTokens:     1_000_000,
		}))

		resp, err := DescribeWorkOrder(ctx, orgID, &pb.DescribeWorkOrderRequest{
			FactoryId: factory.ID.String(),
			OrderId:   order.ID.String(),
		})
		require.NoError(t, err)
		require.Len(t, resp.GetOrder().GetPullRequests(), 1)
		require.Len(t, resp.GetOrder().GetPullRequests()[0].GetActivities(), 1)
		activity := resp.GetOrder().GetPullRequests()[0].GetActivities()[0]
		assert.Equal(t, int64(1_000_000), activity.GetTotalTokens())
		assert.Greater(t, activity.GetCostCents(), int64(0))
		assert.Equal(t, int64(1_000_000), resp.GetOrder().GetPullRequests()[0].GetRuns()[0].GetTotalTokens())
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

		resp, err := DescribeWorkOrder(ctx, orgID, &pb.DescribeWorkOrderRequest{
			FactoryId: factory.ID.String(),
			OrderId:   order.ID.String(),
		})
		require.NoError(t, err)
		require.Len(t, resp.GetOrder().GetPullRequests(), 1)
		require.Len(t, resp.GetOrder().GetPullRequests()[0].GetRuns(), 1)
		assert.EqualValues(t, 1_000_000, resp.GetOrder().GetPullRequests()[0].GetRuns()[0].GetTotalTokens())
		assert.Positive(t, resp.GetOrder().GetPullRequests()[0].GetRuns()[0].GetCostCents())

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

		degraded, err := loadSerializedPullRequestsByWorkOrderIDs(
			ctx,
			db,
			[]uuid.UUID{order.ID},
			map[uuid.UUID]int64{order.ID: order.Number},
		)
		require.NoError(t, err)
		require.Len(t, degraded[order.ID], 1)
		require.Len(t, degraded[order.ID][0].GetRuns(), 1)
		assert.EqualValues(t, 0, degraded[order.ID][0].GetRuns()[0].GetTotalTokens())
		assert.EqualValues(t, 0, degraded[order.ID][0].GetRuns()[0].GetCostCents())
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

		resp, err := ListWorkOrders(ctx, orgID, &pb.ListWorkOrdersRequest{
			FactoryId: other.ID.String(),
		})
		require.NoError(t, err)
		for _, listed := range resp.GetOrders() {
			assert.Empty(t, listed.GetPullRequests())
		}
	})
}

func workOrdersByID(orders []*pb.WorkOrderSummary) map[string]*pb.WorkOrderSummary {
	result := make(map[string]*pb.WorkOrderSummary, len(orders))
	for _, order := range orders {
		result[order.GetId()] = order
	}
	return result
}

func parseUUID(t *testing.T, raw string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(raw)
	require.NoError(t, err)
	return id
}
