package factories

import (
	"context"
	"fmt"
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
		assert.Equal(t, []string{"anthropic/claude-sonnet-4-6"}, activity.GetModels())
		linkedRun := resp.GetOrder().GetPullRequests()[0].GetRuns()[0]
		assert.Equal(t, int64(1_000_000), linkedRun.GetTotalTokens())
		assert.Equal(t, []string{"anthropic/claude-sonnet-4-6"}, linkedRun.GetModels())
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
		assert.Empty(t, degraded[order.ID][0].GetRuns()[0].GetModels())
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

func Test__FactoryPullRequestActivityAuthor(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Feedback", "", &r.User, nil, nil)
	require.NoError(t, err)

	bot := true
	person := false
	cases := []struct {
		name   string
		source string
		event  map[string]any
		want   *bool
	}{
		{
			name:   "github bot comment",
			source: models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion,
			event:  githubDiscussionEvent("github.prComment", "comment", "user", "Bot"),
			want:   &bot,
		},
		{
			name:   "github person review",
			source: models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion,
			event:  githubDiscussionEvent("github.prReview", "review", "user", "User"),
			want:   &person,
		},
		{
			name:   "missing author",
			source: models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion,
			event: map[string]any{
				"type": "github.prComment",
				"data": map[string]any{"comment": map[string]any{"body": "hello"}},
			},
		},
		{
			name:   "bitbucket bot comment",
			source: models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion,
			event:  githubDiscussionEvent("bitbucket.pullRequestComment", "comment", "author", "app_user"),
			want:   &bot,
		},
		{
			name:   "check activity",
			source: models.FactoryPRFeedbackHandlerSourcePullRequestChecks,
			event:  githubDiscussionEvent("github.prComment", "comment", "user", "Bot"),
		},
	}

	for i, tc := range cases {
		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
			nil,
		)
		require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
		_, err := factory.CreatePRFeedbackHandler(
			db,
			canvas.ID,
			models.FactoryPRFeedbackHandlerSubjectGitHubPullRequest,
			tc.source,
		)
		require.NoError(t, err)
		handler, err := models.FindPRFeedbackHandlerByCanvasID(db, canvas.ID)
		require.NoError(t, err)
		require.NotNil(t, handler)

		created, err := CreateFactoryPullRequest(ctx, IntakeDependencies{}, orgID, &pb.CreateFactoryPullRequestRequest{
			FactoryId:   factory.ID.String(),
			WorkOrderId: order.ID.String(),
			Url:         fmt.Sprintf("https://github.com/acme/app/pull/%d", 80+i),
			Title:       tc.name,
		})
		require.NoError(t, err)

		rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "trigger", "default", nil, tc.event)
		run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, rootEvent)
		require.NoError(t, err)
		support.CreateCanvasNodeExecution(t, canvas.ID, "trigger", rootEvent.ID, rootEvent.ID)

		modelPR, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: parseUUID(t, created.GetPullRequest().GetId())})
		require.NoError(t, err)
		_, err = modelPR.CreateActivity(db, models.FactoryPullRequestActivityParams{
			RunID:             run.ID,
			Description:       tc.name,
			Access:            models.FactoryPullRequestAccessExclusive,
			FeedbackHandlerID: &handler.ID,
		})
		require.NoError(t, err)
	}

	resp, err := DescribeWorkOrder(ctx, orgID, &pb.DescribeWorkOrderRequest{
		FactoryId: factory.ID.String(),
		OrderId:   order.ID.String(),
	})
	require.NoError(t, err)

	byDescription := map[string]*pb.FactoryPullRequestActivity{}
	for _, pullRequest := range resp.GetOrder().GetPullRequests() {
		for _, activity := range pullRequest.GetActivities() {
			byDescription[activity.GetDescription()] = activity
		}
	}

	for _, tc := range cases {
		activity := byDescription[tc.name]
		require.NotNil(t, activity, tc.name)
		if tc.want == nil {
			assert.Nil(t, activity.AuthorIsBot, tc.name)
		} else {
			require.NotNil(t, activity.AuthorIsBot, tc.name)
			assert.Equal(t, *tc.want, activity.GetAuthorIsBot(), tc.name)
		}

		runID := parseUUID(t, activity.GetRun().GetId())
		stored, err := models.FindPullRequestActivityByRunID(db, runID)
		require.NoError(t, err)
		if tc.want == nil {
			assert.Nil(t, stored.AuthorIsBot, tc.name)
			if tc.name == "missing author" {
				assert.NotNil(t, stored.AuthorFlagCheckedAt, tc.name)
			}
			if tc.name == "check activity" {
				assert.Nil(t, stored.AuthorFlagCheckedAt, tc.name)
			}
			continue
		}
		require.NotNil(t, stored.AuthorIsBot, tc.name)
		assert.Equal(t, *tc.want, *stored.AuthorIsBot, tc.name)
		assert.NotNil(t, stored.AuthorFlagCheckedAt, tc.name)
	}

	missing := byDescription["missing author"]
	require.NotNil(t, missing)
	missingRunID := parseUUID(t, missing.GetRun().GetId())
	rootEvent, err := models.FindRootEventForRun(db, missingRunID)
	require.NoError(t, err)
	require.NoError(t, db.Model(rootEvent).Update("data", models.NewJSONValue(githubDiscussionEvent("github.prComment", "comment", "user", "Bot"))).Error)
	updatedEvent, err := models.FindRootEventForRun(db, missingRunID)
	require.NoError(t, err)
	updatedFlag := models.DiscussionAuthorIsBot(updatedEvent.Data.Data())
	require.NotNil(t, updatedFlag)
	assert.True(t, *updatedFlag)

	again, err := DescribeWorkOrder(ctx, orgID, &pb.DescribeWorkOrderRequest{
		FactoryId: factory.ID.String(),
		OrderId:   order.ID.String(),
	})
	require.NoError(t, err)
	var reread *pb.FactoryPullRequestActivity
	for _, pullRequest := range again.GetOrder().GetPullRequests() {
		for _, activity := range pullRequest.GetActivities() {
			if activity.GetDescription() == "missing author" {
				reread = activity
			}
		}
	}
	require.NotNil(t, reread)
	assert.Nil(t, reread.AuthorIsBot)
	rereadStored, err := models.FindPullRequestActivityByRunID(db, missingRunID)
	require.NoError(t, err)
	assert.Nil(t, rereadStored.AuthorIsBot)
	assert.NotNil(t, rereadStored.AuthorFlagCheckedAt)
}

func githubDiscussionEvent(eventType, parent, authorKey, authorType string) map[string]any {
	return map[string]any{
		"type": eventType,
		"data": map[string]any{
			parent: map[string]any{
				authorKey: map[string]any{"login": "reviewer", "type": authorType},
			},
		},
	}
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
