package factories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"

	_ "github.com/superplanehq/superplane/pkg/registryimports"
)

func Test__WorkOrderPRFeedbackRuns(t *testing.T) {
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
		appRepo := "acme/app"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			AppRepository: &appRepo,
		}))
		return factory
	}

	createHandler := func(t *testing.T, factory *models.Factory) *pb.FactoryPRFeedbackHandler {
		t.Helper()
		response, err := CreateFactoryPRFeedbackHandler(ctx, deps, orgID, &pb.CreateFactoryPRFeedbackHandlerRequest{
			FactoryId: factory.ID.String(),
		})
		require.NoError(t, err)
		return response.GetHandler()
	}

	createOrder := func(t *testing.T, factory *models.Factory, title string) *models.FactoryWorkOrder {
		t.Helper()
		order, err := factory.CreateWorkOrder(database.DB(t.Context()), title, "", &r.User, nil, nil)
		require.NoError(t, err)
		return order
	}

	attachPR := func(t *testing.T, order *models.FactoryWorkOrder, url string) {
		t.Helper()
		_, err := order.CreateArtifact(database.DB(t.Context()), models.FactoryWorkOrderArtifactParams{
			Type: models.FactoryWorkOrderArtifactTypePR,
			Data: map[string]any{"url": url},
			Key:  url,
		})
		require.NoError(t, err)
	}

	describe := func(t *testing.T, factory *models.Factory, order *models.FactoryWorkOrder) *pb.WorkOrder {
		t.Helper()
		response, err := DescribeWorkOrder(ctx, orgID, &pb.DescribeWorkOrderRequest{
			FactoryId: factory.ID.String(),
			OrderId:   order.ID.String(),
		})
		require.NoError(t, err)
		return response.GetOrder()
	}

	t.Run("returns no runs when the work order has no PR artifact", func(t *testing.T) {
		factory := newFactory(t)
		createHandler(t, factory)
		order := createOrder(t, factory, "No PR")

		assert.Empty(t, describe(t, factory, order).GetPrFeedbackRuns())
	})

	t.Run("returns the handler run whose root event matches the work order PR", func(t *testing.T) {
		factory := newFactory(t)
		handler := createHandler(t, factory)
		order := createOrder(t, factory, "Has PR")
		prURL := "https://github.com/acme/app/pull/99"
		attachPR(t, order, prURL)
		event := emitPRFeedbackComment(t, uuid.MustParse(handler.GetCanvasId()), prURL, 99)

		runs := describe(t, factory, order).GetPrFeedbackRuns()
		require.Len(t, runs, 1)
		item := runs[0]
		assert.Equal(t, handler.GetId(), item.GetHandlerId())
		assert.Equal(t, handler.GetName(), item.GetHandlerName())
		assert.Equal(t, handler.GetCanvasId(), item.GetCanvasId())
		require.NotNil(t, item.GetRun())
		assert.Equal(t, event.RunID.String(), item.GetRun().GetId())
		assert.Equal(t, order.ID.String(), item.GetRun().GetWorkOrderId())
		assert.Equal(t, prURL, item.GetRun().GetPullRequestUrl())
		assert.Equal(t, int64(99), item.GetRun().GetPullRequestNumber())
		assert.Equal(t, pb.FactoryPRFeedbackHandlerRun_STATUS_QUEUED, item.GetRun().GetStatus())
	})

	t.Run("does not return a run that belongs to another work order PR", func(t *testing.T) {
		factory := newFactory(t)
		handler := createHandler(t, factory)
		order := createOrder(t, factory, "This PR")
		other := createOrder(t, factory, "Other PR")
		attachPR(t, order, "https://github.com/acme/app/pull/1")
		attachPR(t, other, "https://github.com/acme/app/pull/2")
		emitPRFeedbackComment(t, uuid.MustParse(handler.GetCanvasId()), "https://github.com/acme/app/pull/2", 2)

		assert.Empty(t, describe(t, factory, order).GetPrFeedbackRuns())
	})

	t.Run("matches issue.pull_request.html_url payloads", func(t *testing.T) {
		factory := newFactory(t)
		handler := createHandler(t, factory)
		order := createOrder(t, factory, "Issue PR")
		prURL := "https://github.com/acme/app/pull/7"
		attachPR(t, order, prURL)
		event := support.EmitCanvasEventForNodeWithData(
			t,
			uuid.MustParse(handler.GetCanvasId()),
			prFeedbackCommentTriggerNodeID,
			"default",
			nil,
			map[string]any{
				"type": prFeedbackEventComment,
				"data": map[string]any{
					"issue": map[string]any{
						"number":       7,
						"pull_request": map[string]any{"html_url": prURL},
					},
					"comment": map[string]any{
						"html_url": prURL + "#issuecomment-1",
						"user":     map[string]any{"login": "alice"},
					},
				},
			},
		)

		runs := describe(t, factory, order).GetPrFeedbackRuns()
		require.Len(t, runs, 1)
		assert.Equal(t, event.RunID.String(), runs[0].GetRun().GetId())
	})

	t.Run("includes PR feedback runs on listed work orders", func(t *testing.T) {
		factory := newFactory(t)
		handler := createHandler(t, factory)
		order := createOrder(t, factory, "Active feedback")
		idle := createOrder(t, factory, "Idle")
		prURL := "https://github.com/acme/app/pull/11"
		attachPR(t, order, prURL)
		attachPR(t, idle, "https://github.com/acme/app/pull/12")
		event := emitPRFeedbackComment(t, uuid.MustParse(handler.GetCanvasId()), prURL, 11)

		listed, err := ListWorkOrders(ctx, orgID, &pb.ListWorkOrdersRequest{FactoryId: factory.ID.String()})
		require.NoError(t, err)
		byID := map[string]*pb.WorkOrder{}
		for _, item := range listed.GetOrders() {
			byID[item.GetId()] = item
		}
		require.Contains(t, byID, order.ID.String())
		require.Contains(t, byID, idle.ID.String())
		require.Len(t, byID[order.ID.String()].GetPrFeedbackRuns(), 1)
		assert.Equal(t, event.RunID.String(), byID[order.ID.String()].GetPrFeedbackRuns()[0].GetRun().GetId())
		assert.Empty(t, byID[idle.ID.String()].GetPrFeedbackRuns())
	})
}

func emitPRFeedbackComment(t *testing.T, canvasID uuid.UUID, prURL string, number int) *models.CanvasEvent {
	t.Helper()
	return support.EmitCanvasEventForNodeWithData(
		t,
		canvasID,
		prFeedbackCommentTriggerNodeID,
		"default",
		nil,
		map[string]any{
			"type": prFeedbackEventComment,
			"data": map[string]any{
				"repository": map[string]any{"full_name": "acme/app"},
				"pull_request": map[string]any{
					"html_url": prURL,
					"number":   number,
				},
				"comment": map[string]any{
					"html_url": prURL + "#issuecomment-1",
					"user":     map[string]any{"login": "alice"},
				},
			},
		},
	)
}
