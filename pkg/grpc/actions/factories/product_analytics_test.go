package factories

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/productanalytics"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

type analyticsRecorder struct {
	mu     sync.Mutex
	events []productanalytics.Event
}

func (r *analyticsRecorder) Capture(event productanalytics.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}

func (r *analyticsRecorder) IdentifyOrganizationGroup(productanalytics.OrganizationGroupIdentity) {}

func (r *analyticsRecorder) Close() error { return nil }

func (r *analyticsRecorder) Enabled() bool { return true }

func (r *analyticsRecorder) reset() {
	r.mu.Lock()
	r.events = nil
	r.mu.Unlock()
}

func (r *analyticsRecorder) snapshot() []productanalytics.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	copied := make([]productanalytics.Event, len(r.events))
	copy(copied, r.events)
	return copied
}

func useAnalytics(t *testing.T) *analyticsRecorder {
	t.Helper()
	recorder := &analyticsRecorder{}
	previous := productanalytics.SetCurrent(recorder)
	t.Cleanup(func() {
		productanalytics.SetCurrent(previous)
	})
	return recorder
}

func eventsNamed(recorder *analyticsRecorder, name productanalytics.EventName) []productanalytics.Event {
	matched := []productanalytics.Event{}
	for _, event := range recorder.snapshot() {
		if event.Name == name {
			matched = append(matched, event)
		}
	}
	return matched
}

func requireNamedEvent(t *testing.T, recorder *analyticsRecorder, name productanalytics.EventName) productanalytics.Event {
	t.Helper()
	matched := eventsNamed(recorder, name)
	require.Len(t, matched, 1)
	return matched[0]
}

func assertNoRestrictedContent(t *testing.T, event productanalytics.Event, forbidden ...string) {
	t.Helper()
	encoded, err := json.Marshal(event)
	require.NoError(t, err)
	for _, value := range forbidden {
		assert.NotContains(t, string(encoded), value)
	}
}

func TestFactoryProductAnalytics(t *testing.T) {
	r := support.Setup(t)
	recorder := useAnalytics(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	orgID := r.Organization.ID.String()

	newFactory := func(t *testing.T) *models.Factory {
		t.Helper()
		factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		return factoryModel
	}

	t.Run("manual creation emits one user event", func(t *testing.T) {
		recorder.reset()
		factoryModel := newFactory(t)
		response, err := CreateWorkOrder(ctx, orgID, &pb.CreateWorkOrderRequest{
			FactoryId:   factoryModel.ID.String(),
			Title:       "Ship refund retries",
			Description: "Do not send this prompt",
		})
		require.NoError(t, err)

		event := requireNamedEvent(t, recorder, productanalytics.EventFactoryTaskCreated)
		assert.Equal(t, "manual", event.Properties["creation_source"])
		assert.Equal(t, r.Account.ID.String(), event.DistinctID)
		assert.Equal(t, productanalytics.ActorUser, event.ActorType)
		assert.Equal(t, productanalytics.SourceAPI, event.Source)
		assert.Equal(t, orgID, event.OrganizationID)
		assert.Equal(t, factoryModel.ID.String(), event.FactoryID)
		assert.Equal(t, response.GetOrder().GetId(), event.TaskID)
		assertNoRestrictedContent(t, event, "Ship refund retries", "Do not send this prompt")
	})

	t.Run("mcp creation uses the mcp source", func(t *testing.T) {
		recorder.reset()
		factoryModel := newFactory(t)
		mcpCtx := productanalytics.WithSource(ctx, productanalytics.SourceMCP)
		_, err := CreateWorkOrder(mcpCtx, orgID, &pb.CreateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			Title:     "MCP task",
		})
		require.NoError(t, err)

		event := requireNamedEvent(t, recorder, productanalytics.EventFactoryTaskCreated)
		assert.Equal(t, "mcp", event.Properties["creation_source"])
		assert.Equal(t, productanalytics.SourceMCP, event.Source)
		assert.Equal(t, r.Account.ID.String(), event.DistinctID)
	})

	t.Run("failed creation emits nothing", func(t *testing.T) {
		recorder.reset()
		factoryModel := newFactory(t)
		_, err := CreateWorkOrder(ctx, orgID, &pb.CreateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			Title:     "   ",
		})
		require.Error(t, err)
		assert.Empty(t, recorder.snapshot())
	})

	t.Run("imported task emits one import event", func(t *testing.T) {
		recorder.reset()
		factoryModel := newFactory(t)
		canvas := support.CreateFactoryCanvas(t, r, factoryModel.ID, "GitHub issues")
		intake, err := factoryModel.CreateIntake(db, canvas.ID, models.FactoryIntakeSourceGitHubIssues)
		require.NoError(t, err)
		deps := IntakeDependencies{
			NewItemSource: func(context.Context, *gorm.DB, *models.FactoryIntake) (intakeItemSource, error) {
				return stubIntakeItemSource{items: []IntakeItem{{
					ID:    "12",
					Key:   "#12",
					Title: "Handle duplicate refunds",
					Body:  "Retrying a refund posts twice.",
					URL:   "https://github.com/acme/payments/issues/12",
				}}}, nil
			},
		}

		response, err := ImportFactoryIntakeItem(ctx, deps, orgID, &pb.ImportFactoryIntakeItemRequest{
			FactoryId: factoryModel.ID.String(),
			IntakeId:  intake.ID.String(),
			ItemId:    "12",
		})
		require.NoError(t, err)

		event := requireNamedEvent(t, recorder, productanalytics.EventFactoryTaskCreated)
		assert.Equal(t, "issue_import", event.Properties["creation_source"])
		assert.Equal(t, models.FactoryIntakeSourceGitHubIssues, event.Properties["issue_source"])
		assert.Equal(t, response.GetOrder().GetId(), event.TaskID)
		assertNoRestrictedContent(t, event, "Handle duplicate refunds", "https://github.com/acme/payments/issues/12")
	})

	t.Run("handoff emits creation and initial start", func(t *testing.T) {
		recorder.reset()
		factoryModel := seedHandOffFactory(t, r, db)
		require.NoError(t, factoryModel.UpdatePlanning(db, models.FactoryPlanning{Enabled: true, Clarity: true, Confidence: true}))
		planApp, planEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "plan", "start-plan")
		implApp, implEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "implement", "start-impl")
		require.NoError(t, implApp.StampFactoryAppTemplate(db, implEntry, "line-implementation", 1))
		_, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
			{Type: models.FactoryLineStepTypeRunApp, AppID: planApp.ID, Entrypoint: planEntry},
			{Type: models.FactoryLineStepTypeRunApp, AppID: implApp.ID, Entrypoint: implEntry},
		})
		require.NoError(t, err)

		_, err = HandOffWorkOrder(ctx, IntakeDependencies{}, orgID, HandOffWorkOrderRequest{
			FactoryID:   factoryModel.ID.String(),
			Title:       "Ship refund retries",
			Description: "Stop double charges.",
			Plan:        "# Retry refunds",
			Column:      HandOffColumnImplement,
		})
		require.NoError(t, err)

		created := requireNamedEvent(t, recorder, productanalytics.EventFactoryTaskCreated)
		started := requireNamedEvent(t, recorder, productanalytics.EventFactoryTaskStarted)
		assert.Equal(t, "handoff", created.Properties["creation_source"])
		assert.Equal(t, "initial", started.Properties["run_kind"])
		assert.Equal(t, created.TaskID, started.TaskID)
		assertNoRestrictedContent(t, created, "Ship refund retries", "Stop double charges.")
	})

	t.Run("planning emits one refinement", func(t *testing.T) {
		recorder.reset()
		session := openAnalysisSession(t, r, db)
		request := &pb.SendPlanningSessionMessageRequest{
			FactoryId: session.FactoryID.String(),
			SessionId: session.ID.String(),
			Text:      "Use the current retry form.",
		}
		_, err := SendPlanningSessionMessage(ctx, orgID, request)
		require.NoError(t, err)
		_, err = SendPlanningSessionMessage(ctx, orgID, &pb.SendPlanningSessionMessageRequest{
			FactoryId: session.FactoryID.String(),
			SessionId: session.ID.String(),
			Text:      "Keep the dark theme.",
		})
		require.NoError(t, err)

		event := requireNamedEvent(t, recorder, productanalytics.EventFactoryTaskRefined)
		assert.Equal(t, "message", event.Properties["interaction"])
		assert.Equal(t, r.Account.ID.String(), event.DistinctID)
		assertNoRestrictedContent(t, event, "Use the current retry form.", "Keep the dark theme.")

		_, err = SendPlanningSessionMessage(ctx, orgID, &pb.SendPlanningSessionMessageRequest{
			FactoryId: session.FactoryID.String(),
			SessionId: session.ID.String(),
			Text:      "   ",
		})
		require.Error(t, err)
		assert.Len(t, eventsNamed(recorder, productanalytics.EventFactoryTaskRefined), 1)
	})

	t.Run("survey answer emits one refinement", func(t *testing.T) {
		recorder.reset()
		session := openAnalysisSession(t, r, db)
		_, err := AnswerPlanningSessionSurvey(ctx, orgID, &pb.AnswerPlanningSessionSurveyRequest{
			FactoryId: session.FactoryID.String(),
			SessionId: session.ID.String(),
			Text:      "What is the priority? High",
		})
		require.NoError(t, err)

		event := requireNamedEvent(t, recorder, productanalytics.EventFactoryTaskRefined)
		assert.Equal(t, "survey_answer", event.Properties["interaction"])
		assertNoRestrictedContent(t, event, "What is the priority? High")
	})

	t.Run("dispatch distinguishes initial and rerun", func(t *testing.T) {
		recorder.reset()
		factoryModel := newFactory(t)
		order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
		require.NoError(t, err)
		app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
		line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
			{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
		})
		require.NoError(t, err)

		_, err = DispatchWorkOrder(ctx, orgID, &pb.DispatchWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
		})
		require.Error(t, err)
		assert.Empty(t, recorder.snapshot())

		_, err = DispatchWorkOrder(ctx, orgID, &pb.DispatchWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			LineName:  line.Name,
		})
		require.NoError(t, err)
		initial := requireNamedEvent(t, recorder, productanalytics.EventFactoryTaskStarted)
		assert.Equal(t, "initial", initial.Properties["run_kind"])

		openOrder, err := factoryModel.CreateWorkOrder(db, "Open first", "", &r.User, nil, nil)
		require.NoError(t, err)
		_, err = UpdateWorkOrderStatus(ctx, orgID, &pb.UpdateWorkOrderStatusRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   openOrder.ID.String(),
			State:     pb.WorkOrder_STATE_OPEN,
		})
		require.NoError(t, err)
		recorder.reset()
		_, err = DispatchWorkOrder(ctx, orgID, &pb.DispatchWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   openOrder.ID.String(),
			LineName:  line.Name,
		})
		require.NoError(t, err)
		opened := requireNamedEvent(t, recorder, productanalytics.EventFactoryTaskStarted)
		assert.Equal(t, "initial", opened.Properties["run_kind"])

		stored, err := factoryModel.FindWorkOrder(db, order.ID)
		require.NoError(t, err)
		firstDispatch, err := stored.FindActiveLineDispatch(db)
		require.NoError(t, err)
		require.NoError(t, firstDispatch.Finish(db, models.CanvasRunResultFailed))
		_, err = UpdateWorkOrderStatus(ctx, orgID, &pb.UpdateWorkOrderStatusRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			State:     pb.WorkOrder_STATE_DRAFT,
		})
		require.NoError(t, err)
		recorder.reset()
		_, err = DispatchWorkOrder(ctx, orgID, &pb.DispatchWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			LineName:  line.Name,
		})
		require.NoError(t, err)
		rerun := requireNamedEvent(t, recorder, productanalytics.EventFactoryTaskStarted)
		assert.Equal(t, "rerun", rerun.Properties["run_kind"])
	})

	t.Run("close results emit once", func(t *testing.T) {
		recorder.reset()
		factoryModel := newFactory(t)
		rejected, err := factoryModel.CreateWorkOrder(db, "Reject me", "secret prompt", &r.User, nil, nil)
		require.NoError(t, err)
		_, err = CloseWorkOrder(context.Background(), orgID, &pb.CloseWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   rejected.ID.String(),
			Result:    pb.WorkOrder_RESULT_REJECTED,
		})
		require.Error(t, err)
		assert.Empty(t, recorder.snapshot())

		_, err = CloseWorkOrder(ctx, orgID, &pb.CloseWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   rejected.ID.String(),
			Result:    pb.WorkOrder_RESULT_REJECTED,
		})
		require.NoError(t, err)
		_, err = CloseWorkOrder(ctx, orgID, &pb.CloseWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   rejected.ID.String(),
			Result:    pb.WorkOrder_RESULT_REJECTED,
		})
		require.NoError(t, err)
		rejectedEvent := requireNamedEvent(t, recorder, productanalytics.EventFactoryTaskRejected)
		assert.Equal(t, "manual", rejectedEvent.Properties["rejection_source"])
		assertNoRestrictedContent(t, rejectedEvent, "Reject me", "secret prompt")

		archived, err := factoryModel.CreateWorkOrder(db, "Archive me", "", &r.User, nil, nil)
		require.NoError(t, err)
		recorder.reset()
		_, err = CloseWorkOrder(ctx, orgID, &pb.CloseWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   archived.ID.String(),
			Result:    pb.WorkOrder_RESULT_ARCHIVED,
		})
		require.NoError(t, err)
		archivedEvent := requireNamedEvent(t, recorder, productanalytics.EventFactoryTaskArchived)
		assert.Equal(t, "manual", archivedEvent.Properties["archive_source"])

		approved, err := factoryModel.CreateWorkOrder(db, "Approve me", "", &r.User, nil, nil)
		require.NoError(t, err)
		_, err = UpdateWorkOrderStatus(ctx, orgID, &pb.UpdateWorkOrderStatusRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   approved.ID.String(),
			State:     pb.WorkOrder_STATE_OPEN,
		})
		require.NoError(t, err)
		recorder.reset()
		_, err = UpdateWorkOrderStatus(ctx, orgID, &pb.UpdateWorkOrderStatusRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   approved.ID.String(),
			State:     pb.WorkOrder_STATE_CLOSED,
			Result:    pb.WorkOrder_RESULT_COMPLETED,
		})
		require.NoError(t, err)
		approvedEvent := requireNamedEvent(t, recorder, productanalytics.EventFactoryTaskApproved)
		assert.Equal(t, "manual", approvedEvent.Properties["close_source"])

		_, err = UpdateWorkOrderStatus(ctx, orgID, &pb.UpdateWorkOrderStatusRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   approved.ID.String(),
			State:     pb.WorkOrder_STATE_CLOSED,
			Result:    pb.WorkOrder_RESULT_COMPLETED,
		})
		require.Error(t, err)
		assert.Len(t, eventsNamed(recorder, productanalytics.EventFactoryTaskApproved), 1)
	})

	t.Run("onboarding completion emits once", func(t *testing.T) {
		support.AssignOpenSystemTrial(t, r.Organization.ID)
		recorder.reset()
		require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
		factoryModel := newFactory(t)
		_, err := UpdateFactoryOnboarding(ctx, IntakeDependencies{}, orgID, &pb.UpdateFactoryOnboardingRequest{
			Id:       factoryModel.ID.String(),
			Complete: github.Ptr(true),
		})
		require.Error(t, err)
		assert.Empty(t, recorder.snapshot())

		vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
		agentID := createReadyOnboardingIntegration(t, r.Organization.ID, "claude")
		appID, lineID := createOnboardingResources(t, r, factoryModel)
		request := readyOnboardingRequest(factoryModel.ID.String(), vcsID, agentID, appID, lineID)
		request.Complete = github.Ptr(true)
		_, err = UpdateFactoryOnboarding(ctx, onboardingDeps(r), orgID, request)
		require.NoError(t, err)
		event := requireNamedEvent(t, recorder, productanalytics.EventFactoryOnboardingCompleted)
		assert.NotEmpty(t, event.Properties["vcs_provider"])
		assert.NotEmpty(t, event.Properties["issues_source"])
		assert.NotEmpty(t, event.Properties["agent_harness"])
		assertNoRestrictedContent(t, event, "acme/api", "acme/backlog")

		_, err = UpdateFactoryOnboarding(ctx, onboardingDeps(r), orgID, request)
		require.NoError(t, err)
		assert.Len(t, eventsNamed(recorder, productanalytics.EventFactoryOnboardingCompleted), 1)
	})

	t.Run("webhook merge is not duplicated or attributed to the creator", func(t *testing.T) {
		recorder.reset()
		factoryModel := newFactory(t)
		order, err := factoryModel.CreateWorkOrder(db, "Merge me", "", &r.User, nil, nil)
		require.NoError(t, err)
		_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
		require.NoError(t, err)
		pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
			Provider:   models.FactoryPullRequestProviderGitHub,
			Repository: "acme/app",
			Number:     42,
			URL:        "https://github.com/acme/app/pull/42",
			Title:      "Ready",
			State:      models.FactoryPullRequestStateOpen,
		})
		require.NoError(t, err)

		integration := &models.Integration{OrganizationID: r.Organization.ID}
		closeFactoryWorkOrdersFromGitHubPullRequestClosed(db, integration, githubPullRequestPayload("acme/app", 42, true, "octocat"))
		closeFactoryWorkOrdersFromGitHubPullRequestClosed(db, integration, githubPullRequestPayload("acme/app", 42, true, "octocat"))

		event := requireNamedEvent(t, recorder, productanalytics.EventFactoryTaskMerged)
		assert.Equal(t, models.ProviderGitHub, event.Properties["provider"])
		assert.Equal(t, "vcs_webhook", event.Properties["merge_source"])
		assert.Equal(t, productanalytics.ActorExternal, event.ActorType)
		assert.Equal(t, productanalytics.SourceGitHubWebhook, event.Source)
		assert.NotEqual(t, r.Account.ID.String(), event.DistinctID)
		assert.NotEqual(t, r.User.String(), event.DistinctID)
		assert.Equal(t, pullRequest.WorkOrderID.String(), event.TaskID)
		assertNoRestrictedContent(t, event, "acme/app", "Ready", "https://github.com/acme/app/pull/42")
	})

	t.Run("webhook close emits one rejection", func(t *testing.T) {
		recorder.reset()
		factoryModel := newFactory(t)
		order, err := factoryModel.CreateWorkOrder(db, "Close me", "", &r.User, nil, nil)
		require.NoError(t, err)
		_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
		require.NoError(t, err)
		_, err = order.CreatePullRequest(db, models.FactoryPullRequestParams{
			Provider:   models.FactoryPullRequestProviderGitHub,
			Repository: "acme/app",
			Number:     44,
			URL:        "https://github.com/acme/app/pull/44",
			Title:      "Closed",
			State:      models.FactoryPullRequestStateOpen,
		})
		require.NoError(t, err)

		integration := &models.Integration{OrganizationID: r.Organization.ID}
		payload := githubPullRequestPayload("acme/app", 44, false, "octocat")
		closeFactoryWorkOrdersFromGitHubPullRequestClosed(db, integration, payload)
		closeFactoryWorkOrdersFromGitHubPullRequestClosed(db, integration, payload)

		event := requireNamedEvent(t, recorder, productanalytics.EventFactoryTaskRejected)
		assert.Equal(t, "pull_request_closed", event.Properties["rejection_source"])
		assert.Equal(t, productanalytics.ActorExternal, event.ActorType)
		assert.NotEqual(t, r.Account.ID.String(), event.DistinctID)
	})

	t.Run("webhook merge maps a known github user", func(t *testing.T) {
		recorder.reset()
		factoryModel := newFactory(t)
		order, err := factoryModel.CreateWorkOrder(db, "Mapped merge", "", &r.User, nil, nil)
		require.NoError(t, err)
		_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
		require.NoError(t, err)
		_, err = order.CreatePullRequest(db, models.FactoryPullRequestParams{
			Provider:   models.FactoryPullRequestProviderGitHub,
			Repository: "acme/app",
			Number:     43,
			URL:        "https://github.com/acme/app/pull/43",
			Title:      "Mapped",
			State:      models.FactoryPullRequestStateOpen,
		})
		require.NoError(t, err)

		closeFactoryWorkOrdersFromGitHubPullRequestClosed(
			db,
			&models.Integration{OrganizationID: r.Organization.ID},
			githubPullRequestPayload("acme/app", 43, true, "testuser"),
		)

		event := requireNamedEvent(t, recorder, productanalytics.EventFactoryTaskMerged)
		assert.Equal(t, productanalytics.ActorUser, event.ActorType)
		assert.Equal(t, r.Account.ID.String(), event.DistinctID)
	})

	t.Run("in-app merge emits once and a failed merge emits nothing", func(t *testing.T) {
		recorder.reset()
		require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryPullRequestMerge))
		factoryModel := newFactory(t)
		order, err := factoryModel.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
		require.NoError(t, err)
		pullRequest := createMergeTestPullRequest(t, db, order)
		const headSHA = "abc123def456"
		combined, checks := successChecks()
		original := newFactoryGitHubAPI
		newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) {
			return &fakeFactoryGitHub{
				pullRequest: mergeableGitHubPullRequest(headSHA),
				combined:    combined,
				checkRuns:   checks,
				repository:  allMethodsRepository(),
			}, nil
		}
		t.Cleanup(func() { newFactoryGitHubAPI = original })

		_, err = MergeFactoryPullRequest(ctx, IntakeDependencies{}, orgID, &pb.MergeFactoryPullRequestRequest{
			FactoryId:       factoryModel.ID.String(),
			PrId:            pullRequest.GetId(),
			MergeMethod:     pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH,
			ExpectedHeadSha: "stale",
		})
		require.Error(t, err)
		assert.Empty(t, recorder.snapshot())

		_, err = MergeFactoryPullRequest(ctx, IntakeDependencies{}, orgID, &pb.MergeFactoryPullRequestRequest{
			FactoryId:       factoryModel.ID.String(),
			PrId:            pullRequest.GetId(),
			MergeMethod:     pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH,
			ExpectedHeadSha: headSHA,
		})
		require.NoError(t, err)
		event := requireNamedEvent(t, recorder, productanalytics.EventFactoryTaskMerged)
		assert.Equal(t, "superplane", event.Properties["merge_source"])
		assert.Equal(t, models.ProviderGitHub, event.Properties["provider"])
		assert.Equal(t, r.Account.ID.String(), event.DistinctID)
		assertNoRestrictedContent(t, event, "acme/app", "Ready")
	})
}

func githubPullRequestPayload(repository string, number int64, merged bool, login string) githubMergeabilityWebhookPayload {
	mergedAt := ""
	closedAt := "2026-10-20T12:00:00Z"
	if merged {
		mergedAt = closedAt
	}
	body := map[string]any{
		"repository": map[string]any{"full_name": repository},
		"pull_request": map[string]any{
			"number":    number,
			"merged":    merged,
			"merged_at": mergedAt,
			"closed_at": closedAt,
			"merged_by": map[string]any{"login": login, "name": login},
		},
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	var payload githubMergeabilityWebhookPayload
	if err := json.Unmarshal(encoded, &payload); err != nil {
		panic(err)
	}
	return payload
}
