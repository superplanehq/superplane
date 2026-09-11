package factories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__StartPlanningSession__CreatesSessionAndPendingRun(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	setupPlanningStart(t, r.Organization.ID)
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	appRepo := "acme/payments"
	require.NoError(t, factoryModel.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		AppRepository: &appRepo,
	}))

	resp, err := StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
		FactoryId: factoryModel.ID.String(),
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Session)
	assert.Equal(t, "acme/payments", resp.Session.Repository)
	assert.Equal(t, models.PlanningSessionStateRunning, resp.Session.State)
	assert.NotEmpty(t, resp.Session.CanvasRunId)
	assert.Empty(t, resp.Session.ExecutionId)
	assert.Empty(t, resp.Session.Messages)

	canvas, err := models.FindPlanningCanvas(database.DB(t.Context()), r.Organization.ID, factoryModel.ID)
	require.NoError(t, err)
	nodes, err := models.FindCanvasNodesInTransaction(database.DB(t.Context()), canvas.ID)
	require.NoError(t, err)
	hasAgent := false
	for _, node := range nodes {
		if node.Type == models.NodeTypeComponent {
			hasAgent = true
			prompt := planningCanvasPromptFromConfig(node.Configuration.Data())
			assert.Contains(t, prompt, "greet the user in plain text")
			assert.Contains(t, prompt, "Use survey to ask one or more questions")
			assert.NotContains(t, prompt, "say:")
			assert.NotContains(t, prompt, "with say")
			assert.Contains(t, prompt, "Do not call wait_for_user")
			assert.Contains(t, prompt, "When the user creates or skips a draft")
			assert.Contains(t, prompt, "When the user starts a refine")
			assert.Contains(t, prompt, "planning_session.refine_key")
			assert.Contains(t, prompt, "If the refine key is not empty")
			assert.NotContains(t, prompt, "Start by calling wait_for_user")
		}
	}
	assert.True(t, hasAgent)

	apps, err := ListFactoryApps(ctx, r.Organization.ID.String(), &pb.ListFactoryAppsRequest{
		FactoryId: factoryModel.ID.String(),
	})
	require.NoError(t, err)
	require.Len(t, apps.Apps, 1)
	assert.Equal(t, canvas.ID.String(), apps.Apps[0].Id)
	assert.Equal(t, models.PlanningCanvasName, apps.Apps[0].Name)

	described, err := DescribePlanningSession(ctx, r.Organization.ID.String(), &pb.DescribePlanningSessionRequest{
		FactoryId: factoryModel.ID.String(),
		SessionId: resp.Session.Id,
	})
	require.NoError(t, err)
	assert.Equal(t, resp.Session.Id, described.Session.Id)
	assert.Empty(t, described.Session.ExecutionId)
}

func Test__DescribePlanningSession__IncludesPendingSurvey(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	setupPlanningStart(t, r.Organization.ID)
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	appRepo := "acme/payments"
	require.NoError(t, factoryModel.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		AppRepository: &appRepo,
	}))

	started, err := StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
		FactoryId: factoryModel.ID.String(),
	})
	require.NoError(t, err)
	session, err := models.FindPlanningSession(database.DB(t.Context()), r.Organization.ID, factoryModel.ID, uuid.MustParse(started.Session.Id))
	require.NoError(t, err)
	require.NoError(t, session.ProposeSurvey(database.DB(t.Context()), models.PlanningSessionSurvey{
		Questions: []models.PlanningSessionSurveyQuestion{
			{Prompt: "What is the priority?", Options: []string{"High", "Low"}},
		},
	}))

	described, err := DescribePlanningSession(ctx, r.Organization.ID.String(), &pb.DescribePlanningSessionRequest{
		FactoryId: factoryModel.ID.String(),
		SessionId: started.Session.Id,
	})
	require.NoError(t, err)
	require.Empty(t, described.Session.Messages)
	require.NotNil(t, described.Session.Survey)
	require.Len(t, described.Session.Survey.Questions, 1)
	assert.Equal(t, "What is the priority?", described.Session.Survey.Questions[0].Prompt)
}

func Test__StartPlanningSession__KeepsExistingCanvasPrompt(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	setupPlanningStart(t, r.Organization.ID)
	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	_, err = StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
		FactoryId:  factoryModel.ID.String(),
		Repository: "acme/payments",
	})
	require.NoError(t, err)

	canvas, err := models.FindPlanningCanvas(database.DB(t.Context()), r.Organization.ID, factoryModel.ID)
	require.NoError(t, err)
	nodes, err := models.FindCanvasNodesInTransaction(database.DB(t.Context()), canvas.ID)
	require.NoError(t, err)
	rewrote := false
	for i := range nodes {
		if nodes[i].Type != models.NodeTypeComponent {
			continue
		}
		config := nodes[i].Configuration.Data()
		for _, step := range planningCanvasConfigSteps(config["steps"]) {
			if _, ok := step["prompt"]; ok {
				step["prompt"] = "Custom planning prompt."
				rewrote = true
			}
		}
		nodes[i].Configuration = datatypes.NewJSONType(config)
		require.NoError(t, database.DB(t.Context()).Model(&nodes[i]).Select("Configuration").Updates(&nodes[i]).Error)
	}
	require.True(t, rewrote)

	_, err = StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
		FactoryId:  factoryModel.ID.String(),
		Repository: "acme/payments",
	})
	require.NoError(t, err)
	assert.Equal(t, "Custom planning prompt.", planningAgentPrompt(t, r.Organization.ID, factoryModel.ID))
}

func Test__StartPlanningSession__AttachesDraftWorkOrder(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	setupPlanningStart(t, r.Organization.ID)
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &r.User, nil, nil)
	require.NoError(t, err)

	resp, err := StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
		FactoryId:   factoryModel.ID.String(),
		Repository:  "acme/payments",
		WorkOrderId: order.ID.String(),
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Session.Draft)
	assert.Equal(t, "Retry refunds", resp.Session.Draft.Title)
	assert.Equal(t, "Stop double charges.", resp.Session.Draft.Description)
	assert.Equal(t, order.ID.String(), resp.Session.Draft.WorkOrderId)
	require.Len(t, resp.Session.Created, 1)
	assert.Equal(t, order.ID.String(), resp.Session.Created[0].Id)

	session, err := models.FindPlanningSession(db, r.Organization.ID, factoryModel.ID, uuid.MustParse(resp.Session.Id))
	require.NoError(t, err)
	var run models.CanvasRun
	require.NoError(t, db.First(&run, "id = ?", session.CanvasRunID).Error)
	input, ok := run.Input.Data().(map[string]any)
	require.True(t, ok)
	planning, ok := input["planning_session"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, factoryModel.WorkOrderKey(order.Number), planning["refine_key"])
}

func Test__StartPlanningSession__SyncsStockGreetCloser(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	setupPlanningStart(t, r.Organization.ID)
	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	_, err = StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
		FactoryId:  factoryModel.ID.String(),
		Repository: "acme/payments",
	})
	require.NoError(t, err)

	canvas, err := models.FindPlanningCanvas(database.DB(t.Context()), r.Organization.ID, factoryModel.ID)
	require.NoError(t, err)
	nodes, err := models.FindCanvasNodesInTransaction(database.DB(t.Context()), canvas.ID)
	require.NoError(t, err)
	rewrote := false
	for i := range nodes {
		if nodes[i].Type != models.NodeTypeComponent {
			continue
		}
		config := nodes[i].Configuration.Data()
		for _, step := range planningCanvasConfigSteps(config["steps"]) {
			if _, ok := step["prompt"]; ok {
				step["prompt"] = "You are in a SuperPlane planning session.\n\nGreet the user in plain text. Then stop."
				rewrote = true
			}
		}
		nodes[i].Configuration = datatypes.NewJSONType(config)
		require.NoError(t, database.DB(t.Context()).Model(&nodes[i]).Select("Configuration").Updates(&nodes[i]).Error)
	}
	require.True(t, rewrote)
	require.NoError(t, rewritePlanningCanvasLivePrompt(t, canvas.ID, "You are in a SuperPlane planning session.\n\nGreet the user in plain text. Then stop."))

	_, err = StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
		FactoryId:  factoryModel.ID.String(),
		Repository: "acme/payments",
	})
	require.NoError(t, err)
	prompt := planningAgentPrompt(t, r.Organization.ID, factoryModel.ID)
	assert.Contains(t, prompt, "planning_session.refine_key")
	assert.Contains(t, prompt, "If the refine key is not empty")
	assert.Contains(t, planningLiveAgentPrompt(t, canvas.ID), "planning_session.refine_key")
}

func Test__StartPlanningSession__KeepsPreviousSessionForSameUser(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	setupPlanningStart(t, r.Organization.ID)
	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	first, err := StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
		FactoryId:  factoryModel.ID.String(),
		Repository: "acme/payments",
	})
	require.NoError(t, err)
	firstRunID := uuid.MustParse(first.Session.CanvasRunId)

	second, err := StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
		FactoryId:  factoryModel.ID.String(),
		Repository: "acme/payments",
	})
	require.NoError(t, err)
	assert.NotEqual(t, first.Session.Id, second.Session.Id)
	assert.Equal(t, models.PlanningSessionStateRunning, second.Session.State)

	db := database.DB(t.Context())
	previous, err := models.FindPlanningSession(db, r.Organization.ID, factoryModel.ID, uuid.MustParse(first.Session.Id))
	require.NoError(t, err)
	assert.Equal(t, models.PlanningSessionStateRunning, previous.State)

	canvas, err := models.FindPlanningCanvas(db, r.Organization.ID, factoryModel.ID)
	require.NoError(t, err)
	run, err := models.FindCanvasRunInTransaction(db, canvas.ID, firstRunID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStatePending, run.State)
}

func Test__StartPlanningSession__RejectsWhenParallelismCapIsReached(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	setupPlanningStart(t, r.Organization.ID)
	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	first, err := StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
		FactoryId:  factoryModel.ID.String(),
		Repository: "acme/payments",
	})
	require.NoError(t, err)
	db := database.DB(t.Context())
	canvas, err := models.FindPlanningCanvas(db, r.Organization.ID, factoryModel.ID)
	require.NoError(t, err)
	node, err := models.FindCanvasNode(db, canvas.ID, planningCanvasAgentNodeID)
	require.NoError(t, err)
	limit := 2
	node.SetConcurrencySpec(&models.ConcurrencySpec{Max: &limit})
	require.NoError(t, db.Model(node).Select("ConcurrencyKey", "ConcurrencyMax").Updates(node).Error)

	second, err := StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
		FactoryId:  factoryModel.ID.String(),
		Repository: "acme/payments",
	})
	require.NoError(t, err)
	assert.NotEqual(t, first.Session.Id, second.Session.Id)

	_, err = StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
		FactoryId:  factoryModel.ID.String(),
		Repository: "acme/payments",
	})
	require.Error(t, err)
}

func Test__StartPlanningSession__RejectsEleventhSessionAtDefaultCap(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	setupPlanningStart(t, r.Organization.ID)
	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	for i := 0; i < models.DefaultFactoryLineStepMaxParallelism; i++ {
		_, err = StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
			FactoryId:  factoryModel.ID.String(),
			Repository: "acme/payments",
		})
		require.NoError(t, err)
	}

	_, err = StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
		FactoryId:  factoryModel.ID.String(),
		Repository: "acme/payments",
	})
	require.Error(t, err)
}

func Test__StartPlanningSession__UsesSuperPlaneAndDefaultParallelism(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	setupPlanningStart(t, r.Organization.ID)
	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	_, err = StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
		FactoryId:  factoryModel.ID.String(),
		Repository: "acme/payments",
	})
	require.NoError(t, err)
	canvas, err := models.FindPlanningCanvas(database.DB(t.Context()), r.Organization.ID, factoryModel.ID)
	require.NoError(t, err)
	nodes, err := models.FindCanvasNodesInTransaction(database.DB(t.Context()), canvas.ID)
	require.NoError(t, err)
	foundAgent := false
	for _, node := range nodes {
		if node.Type == models.NodeTypeComponent {
			assert.Equal(t, models.SuperPlaneRunnerComponent, node.ComponentName())
			require.NotNil(t, node.ConcurrencyMax)
			assert.Equal(t, models.DefaultFactoryLineStepMaxParallelism, *node.ConcurrencyMax)
			foundAgent = true
		}
	}
	assert.True(t, foundAgent)
}

func Test__StartPlanningSession__UsesSuperPlaneWhenSetupIsCodex(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	setupPlanningStart(t, r.Organization.ID)
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	agentID := createReadyOnboardingIntegration(t, r.Organization.ID, "openai")
	require.NoError(t, factoryModel.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		AgentIntegrationID: &agentID,
	}))

	_, err = StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
		FactoryId:  factoryModel.ID.String(),
		Repository: "acme/payments",
	})
	require.NoError(t, err)
	canvas, err := models.FindPlanningCanvas(db, r.Organization.ID, factoryModel.ID)
	require.NoError(t, err)
	nodes, err := models.FindCanvasNodesInTransaction(db, canvas.ID)
	require.NoError(t, err)
	for _, node := range nodes {
		if node.Type == models.NodeTypeComponent {
			assert.Equal(t, models.SuperPlaneRunnerComponent, node.ComponentName())
			return
		}
	}
	t.Fatal("missing planning agent")
}

func Test__StartPlanningSession__RequiresClaude(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.Conn()
	require.NoError(t, db.Where("provider <> ?", "").Delete(&models.HostedLLMProvider{}).Error)
	createReadyOnboardingIntegration(t, r.Organization.ID, intakeGitHubAppName)
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	_, err = StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
		FactoryId:  factoryModel.ID.String(),
		Repository: "acme/payments",
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "Connect Claude before you start Create with an Agent.")
}

func Test__StartPlanningSession__RequiresGitHub(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	upsertHostedOnboardingProvider(t, database.DB(t.Context()))
	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	_, err = StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
		FactoryId:  factoryModel.ID.String(),
		Repository: "acme/payments",
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "Connect GitHub before you start Create with an Agent.")
}

func Test__StartPlanningSession__RequiresRepository(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	_, err = StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
		FactoryId: factoryModel.ID.String(),
	})
	require.Error(t, err)
}

func Test__PlanningSession__MessageDraftCreateAndEnd(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	setupPlanningStart(t, r.Organization.ID)
	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	started, err := StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
		FactoryId:  factoryModel.ID.String(),
		Repository: "acme/payments",
	})
	require.NoError(t, err)
	sessionID := started.Session.Id

	sent, err := SendPlanningSessionMessage(ctx, r.Organization.ID.String(), &pb.SendPlanningSessionMessageRequest{
		FactoryId: factoryModel.ID.String(),
		SessionId: sessionID,
		Text:      "Add refund retries",
	})
	require.NoError(t, err)
	require.Len(t, sent.Session.Messages, 1)
	assert.Equal(t, "Add refund retries", sent.Session.Messages[0].Text)
	require.NotNil(t, sent.Session.Messages[0].CreatedAt)
	assert.False(t, sent.Session.Messages[0].CreatedAt.AsTime().IsZero())

	db := database.DB(t.Context())
	session, err := models.FindPlanningSession(db, r.Organization.ID, factoryModel.ID, uuid.MustParse(sessionID))
	require.NoError(t, err)
	require.NoError(t, session.ProposeDraft(db, models.PlanningSessionDraft{
		Title:       "Retry refunds",
		Description: "Stop double charges.",
	}))

	created, err := CreatePlanningSessionWorkOrder(ctx, r.Organization.ID.String(), &pb.CreatePlanningSessionWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		SessionId: sessionID,
	})
	require.NoError(t, err)
	require.Len(t, created.Session.Created, 1)
	assert.Equal(t, "Retry refunds", created.Session.Created[0].Title)

	refined, err := SendPlanningSessionMessage(ctx, r.Organization.ID.String(), &pb.SendPlanningSessionMessageRequest{
		FactoryId: factoryModel.ID.String(),
		SessionId: sessionID,
		Text:      models.PlanningRefineNote(created.Session.Created[0].Key, created.Session.Created[0].Title),
	})
	require.NoError(t, err)
	require.NotNil(t, refined.Session.Draft)
	assert.Equal(t, "Retry refunds", refined.Session.Draft.Title)

	session, err = models.FindPlanningSession(db, r.Organization.ID, factoryModel.ID, uuid.MustParse(sessionID))
	require.NoError(t, err)
	require.NoError(t, session.UpdateDraft(db, models.PlanningSessionDraft{
		Title:       "Retry refunds once",
		Description: "One retry only.",
	}))

	updated, err := CreatePlanningSessionWorkOrder(ctx, r.Organization.ID.String(), &pb.CreatePlanningSessionWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		SessionId: sessionID,
	})
	require.NoError(t, err)
	require.Len(t, updated.Session.Created, 1)
	assert.Equal(t, created.Session.Created[0].Id, updated.Session.Created[0].Id)
	assert.Equal(t, "Retry refunds once", updated.Session.Created[0].Title)

	ended, err := EndPlanningSession(ctx, r.Organization.ID.String(), &pb.EndPlanningSessionRequest{
		FactoryId: factoryModel.ID.String(),
		SessionId: sessionID,
	})
	require.NoError(t, err)
	assert.Equal(t, models.PlanningSessionStateEnded, ended.Session.State)
}

func Test__ReloadPlanningSessionAgent__KeepsSessionAndLiveCanvas(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	setupPlanningStart(t, r.Organization.ID)
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	started, err := StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
		FactoryId:  factoryModel.ID.String(),
		Repository: "acme/payments",
	})
	require.NoError(t, err)
	require.NotEmpty(t, started.Session.CanvasRunId)
	originalRunID := started.Session.CanvasRunId

	_, err = SendPlanningSessionMessage(ctx, r.Organization.ID.String(), &pb.SendPlanningSessionMessageRequest{
		FactoryId: factoryModel.ID.String(),
		SessionId: started.Session.Id,
		Text:      "Add a Size field.",
	})
	require.NoError(t, err)

	reloaded, err := ReloadPlanningSessionAgent(ctx, r.Organization.ID.String(), &pb.ReloadPlanningSessionAgentRequest{
		FactoryId:          factoryModel.ID.String(),
		SessionId:          started.Session.Id,
		SelectableModelKey: "hosted::anthropic::sonnet",
	})
	require.NoError(t, err)
	require.NotNil(t, reloaded.Session)
	assert.Equal(t, started.Session.Id, reloaded.Session.Id)
	assert.Equal(t, models.PlanningSessionStateRunning, reloaded.Session.State)
	assert.NotEqual(t, originalRunID, reloaded.Session.CanvasRunId)
	assert.Equal(t, "hosted::anthropic::sonnet", reloaded.Session.SelectableModelKey)
	require.Len(t, reloaded.Session.Messages, 1)
	assert.Equal(t, "Add a Size field.", reloaded.Session.Messages[0].Text)

	canvas, err := models.FindPlanningCanvas(db, r.Organization.ID, factoryModel.ID)
	require.NoError(t, err)
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(db, canvas.ID)
	require.NoError(t, err)
	for _, node := range []models.Node(liveVersion.Nodes) {
		if node.ID != planningCanvasAgentNodeID {
			continue
		}
		assert.NotEqual(t, "hosted::anthropic::sonnet", node.Configuration["model"])
		assert.NotContains(t, planningCanvasPromptFromConfig(node.Configuration), "Prior messages")
	}

	run, err := models.FindCanvasRunInTransaction(db, canvas.ID, uuid.MustParse(reloaded.Session.CanvasRunId))
	require.NoError(t, err)
	assert.NotEqual(t, liveVersion.ID, run.VersionID)
	input, ok := run.Input.Data().(map[string]any)
	require.True(t, ok)
	planning, ok := input["planning_session"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "hosted::anthropic::sonnet", planning["selectable_model_key"])
	version, err := models.FindCanvasVersionInTransaction(db, canvas.ID, run.VersionID)
	require.NoError(t, err)
	foundAgent := false
	for _, node := range []models.Node(version.Nodes) {
		if node.ID != planningCanvasAgentNodeID {
			continue
		}
		foundAgent = true
		assert.Equal(t, models.SuperPlaneRunnerComponent, node.ComponentName())
		assert.Equal(t, "hosted::anthropic::sonnet", node.Configuration["model"])
		assert.Equal(t, "anthropic", node.Configuration["hostedProvider"])
		assert.Contains(t, planningCanvasPromptFromConfig(node.Configuration), "Add a Size field.")
		assert.Contains(t, planningCanvasPromptFromConfig(node.Configuration), "Do not greet as if the session is new")
	}
	assert.True(t, foundAgent)

	require.NoError(t, models.EndPlanningSessionForFinishedRun(db, uuid.MustParse(originalRunID), models.CanvasRunResultCancelled))
	session, err := models.FindPlanningSession(db, r.Organization.ID, factoryModel.ID, uuid.MustParse(started.Session.Id))
	require.NoError(t, err)
	assert.Equal(t, models.PlanningSessionStateRunning, session.State)
}

func setupPlanningStart(t *testing.T, organizationID uuid.UUID) {
	t.Helper()
	upsertHostedOnboardingProvider(t, database.DB(t.Context()))
	createReadyOnboardingIntegration(t, organizationID, intakeGitHubAppName)
}

func planningAgentPrompt(t *testing.T, organizationID, factoryID uuid.UUID) string {
	t.Helper()
	canvas, err := models.FindPlanningCanvas(database.DB(t.Context()), organizationID, factoryID)
	require.NoError(t, err)
	nodes, err := models.FindCanvasNodesInTransaction(database.DB(t.Context()), canvas.ID)
	require.NoError(t, err)
	for _, node := range nodes {
		if node.Type == models.NodeTypeComponent {
			return planningCanvasPromptFromConfig(node.Configuration.Data())
		}
	}
	t.Fatal("missing planning agent")
	return ""
}

func planningLiveAgentPrompt(t *testing.T, canvasID uuid.UUID) string {
	t.Helper()
	live, err := models.FindLiveCanvasVersionInTransaction(database.DB(t.Context()), canvasID)
	require.NoError(t, err)
	for _, node := range live.Nodes {
		if node.Type == models.NodeTypeComponent {
			return planningCanvasPromptFromConfig(node.Configuration)
		}
	}
	t.Fatal("missing planning agent in live version")
	return ""
}

func rewritePlanningCanvasLivePrompt(t *testing.T, canvasID uuid.UUID, prompt string) error {
	t.Helper()
	live, err := models.FindLiveCanvasVersionInTransaction(database.DB(t.Context()), canvasID)
	if err != nil {
		return err
	}
	nodes := append([]models.Node(nil), live.Nodes...)
	for i := range nodes {
		for _, step := range planningCanvasConfigSteps(nodes[i].Configuration["steps"]) {
			if _, ok := step["prompt"]; ok {
				step["prompt"] = prompt
			}
		}
	}
	live.Nodes = datatypes.NewJSONSlice(nodes)
	return database.DB(t.Context()).Model(live).Select("Nodes").Updates(live).Error
}

func Test__FindPlanningSessionByWorkOrder__ReturnsAnalysisSession(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	setupPlanningStart(t, r.Organization.ID)
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &r.User, nil, nil)
	require.NoError(t, err)

	_, err = StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
		FactoryId:   factoryModel.ID.String(),
		Repository:  "acme/payments",
		WorkOrderId: order.ID.String(),
	})
	require.NoError(t, err)

	_, err = FindPlanningSessionByWorkOrder(ctx, r.Organization.ID.String(), &pb.FindPlanningSessionByWorkOrderRequest{
		FactoryId:   factoryModel.ID.String(),
		WorkOrderId: order.ID.String(),
	})
	require.Error(t, err)

	canvas, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "backlog", "start")
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "start", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, models.AttachAnalysisSessionParams{
		CreatedByUserID: r.User,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		CanvasRunID:     run.ID,
		WorkOrderID:     order.ID,
	})
	require.NoError(t, err)

	found, err := FindPlanningSessionByWorkOrder(ctx, r.Organization.ID.String(), &pb.FindPlanningSessionByWorkOrderRequest{
		FactoryId:   factoryModel.ID.String(),
		WorkOrderId: order.ID.String(),
	})
	require.NoError(t, err)
	require.NotNil(t, found.Session)
	assert.Equal(t, session.ID.String(), found.Session.Id)
	assert.Equal(t, order.ID.String(), found.Session.Draft.WorkOrderId)
	assert.Empty(t, found.Session.ExecutionId)

	event := support.EmitCanvasEventForNode(t, canvas.ID, "start", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, intakeAnalysisNodeID, event.ID, event.ID)
	require.NoError(t, db.Model(execution).Update("run_id", run.ID).Error)

	found, err = FindPlanningSessionByWorkOrder(ctx, r.Organization.ID.String(), &pb.FindPlanningSessionByWorkOrderRequest{
		FactoryId:   factoryModel.ID.String(),
		WorkOrderId: order.ID.String(),
	})
	require.NoError(t, err)
	require.NotNil(t, found.Session)
	assert.Equal(t, execution.ID.String(), found.Session.ExecutionId)
}

func Test__SendPlanningSessionMessage__RestartsEndedAnalysis(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	setupPlanningStart(t, r.Organization.ID)
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &r.User, nil, nil)
	require.NoError(t, err)
	canvas := createOnWorkOrderCanvas(t, r, factoryModel.ID)
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "start", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, models.AttachAnalysisSessionParams{
		CreatedByUserID: r.User,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		CanvasRunID:     run.ID,
		WorkOrderID:     order.ID,
	})
	require.NoError(t, err)
	require.NoError(t, session.ProposeSpec(db, "# Retry refunds\n\n## Executive summary\n\nStop double charges.\n"))
	require.NoError(t, session.End(db))

	sent, err := SendPlanningSessionMessage(ctx, r.Organization.ID.String(), &pb.SendPlanningSessionMessageRequest{
		FactoryId: factoryModel.ID.String(),
		SessionId: session.ID.String(),
		Text:      "Keep the existing retry helper.",
	})
	require.NoError(t, err)
	require.NotNil(t, sent.Session)
	assert.Equal(t, models.PlanningSessionStateRunning, sent.Session.State)
	assert.Empty(t, sent.Session.CanvasRunId)
	require.GreaterOrEqual(t, len(sent.Session.Messages), 1)
	assert.Equal(t, "Keep the existing retry helper.", sent.Session.Messages[len(sent.Session.Messages)-1].Text)

	events, err := models.ListCanvasEvents(db, canvas.ID, "start", 10, nil)
	require.NoError(t, err)
	require.NotEmpty(t, events)
	payload, ok := events[0].Data.Data().(map[string]any)
	require.True(t, ok)
	assert.Equal(t, factory.OnWorkOrderPayloadType, payload["type"])
}

func Test__SendPlanningSessionMessage__RestartsCancelledAnalysisRun(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	setupPlanningStart(t, r.Organization.ID)
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &r.User, nil, nil)
	require.NoError(t, err)
	canvas := createOnWorkOrderCanvas(t, r, factoryModel.ID)
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "start", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, models.AttachAnalysisSessionParams{
		CreatedByUserID: r.User,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		CanvasRunID:     run.ID,
		WorkOrderID:     order.ID,
	})
	require.NoError(t, err)
	require.NoError(t, finishCanvasRun(db, run, models.CanvasRunResultCancelled))
	require.Equal(t, models.PlanningSessionStateRunning, session.State)

	before, err := models.ListCanvasEvents(db, canvas.ID, "start", 10, nil)
	require.NoError(t, err)

	sent, err := SendPlanningSessionMessage(ctx, r.Organization.ID.String(), &pb.SendPlanningSessionMessageRequest{
		FactoryId: factoryModel.ID.String(),
		SessionId: session.ID.String(),
		Text:      "Keep the existing retry helper.",
	})
	require.NoError(t, err)
	require.NotNil(t, sent.Session)
	assert.Equal(t, models.PlanningSessionStateRunning, sent.Session.State)
	assert.Empty(t, sent.Session.CanvasRunId)
	require.GreaterOrEqual(t, len(sent.Session.Messages), 1)
	assert.Equal(t, "Keep the existing retry helper.", sent.Session.Messages[len(sent.Session.Messages)-1].Text)

	after, err := models.ListCanvasEvents(db, canvas.ID, "start", 10, nil)
	require.NoError(t, err)
	require.Greater(t, len(after), len(before))
	payload, ok := after[0].Data.Data().(map[string]any)
	require.True(t, ok)
	assert.Equal(t, factory.OnWorkOrderPayloadType, payload["type"])
}

func Test__SendPlanningSessionMessage__KeepsLiveAnalysisOnTheCurrentRun(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	setupPlanningStart(t, r.Organization.ID)
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &r.User, nil, nil)
	require.NoError(t, err)
	canvas := createOnWorkOrderCanvas(t, r, factoryModel.ID)
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "start", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, models.AttachAnalysisSessionParams{
		CreatedByUserID: r.User,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		CanvasRunID:     run.ID,
		WorkOrderID:     order.ID,
	})
	require.NoError(t, err)
	before, err := models.ListCanvasEvents(db, canvas.ID, "start", 10, nil)
	require.NoError(t, err)

	sent, err := SendPlanningSessionMessage(ctx, r.Organization.ID.String(), &pb.SendPlanningSessionMessageRequest{
		FactoryId: factoryModel.ID.String(),
		SessionId: session.ID.String(),
		Text:      "Keep the existing retry helper.",
	})
	require.NoError(t, err)
	require.NotNil(t, sent.Session)
	assert.Equal(t, run.ID.String(), sent.Session.CanvasRunId)

	after, err := models.ListCanvasEvents(db, canvas.ID, "start", 10, nil)
	require.NoError(t, err)
	assert.Equal(t, len(before), len(after))
}

func Test__SendPlanningSessionMessage__RejectsEndedPlanningSession(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	setupPlanningStart(t, r.Organization.ID)
	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	started, err := StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
		FactoryId:  factoryModel.ID.String(),
		Repository: "acme/payments",
	})
	require.NoError(t, err)
	_, err = EndPlanningSession(ctx, r.Organization.ID.String(), &pb.EndPlanningSessionRequest{
		FactoryId: factoryModel.ID.String(),
		SessionId: started.Session.Id,
	})
	require.NoError(t, err)

	_, err = SendPlanningSessionMessage(ctx, r.Organization.ID.String(), &pb.SendPlanningSessionMessageRequest{
		FactoryId: factoryModel.ID.String(),
		SessionId: started.Session.Id,
		Text:      "Add refund retries",
	})
	require.Error(t, err)
}

func createOnWorkOrderCanvas(t *testing.T, r *support.ResourceRegistry, factoryID uuid.UUID) *models.Canvas {
	t.Helper()
	now := time.Now()
	liveVersionID := uuid.New()
	canvas := &models.Canvas{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		LiveVersionID:  &liveVersionID,
		FactoryID:      &factoryID,
		Name:           "Backlog",
		CreatedBy:      &r.User,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	require.NoError(t, database.DB(t.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(canvas).Error; err != nil {
			return err
		}
		node := models.CanvasNode{
			WorkflowID: canvas.ID,
			NodeID:     "start",
			Name:       "On Task",
			Type:       models.NodeTypeTrigger,
			State:      models.CanvasNodeStateReady,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: factory.OnWorkOrderTriggerName},
			}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}
		if err := tx.Create(&node).Error; err != nil {
			return err
		}
		version := models.CanvasVersion{
			ID:         liveVersionID,
			WorkflowID: canvas.ID,
			OwnerID:    &r.User,
			Nodes: datatypes.NewJSONSlice([]models.Node{{
				ID:   "start",
				Name: "On Task",
				Type: models.NodeTypeTrigger,
				Ref:  models.NodeRef{Trigger: &models.TriggerRef{Name: factory.OnWorkOrderTriggerName}},
			}}),
			Edges:     datatypes.NewJSONSlice([]models.Edge{}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}
		return tx.Create(&version).Error
	}))
	return canvas
}

func finishCanvasRun(db *gorm.DB, run *models.CanvasRun, result string) error {
	now := time.Now()
	run.State = models.CanvasRunStateFinished
	run.Result = result
	run.FinishedAt = &now
	run.UpdatedAt = &now
	return db.Model(run).Updates(map[string]any{
		"state":       run.State,
		"result":      run.Result,
		"finished_at": run.FinishedAt,
		"updated_at":  run.UpdatedAt,
	}).Error
}
