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
	"gorm.io/datatypes"
)

func Test__StartPlanningSession__CreatesSessionAndPendingRun(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	upsertHostedOnboardingProvider(t, db)
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
			assert.Contains(t, prompt, "Greet the user in plain text")
			assert.Contains(t, prompt, "Use survey to ask one or more questions")
			assert.NotContains(t, prompt, "say:")
			assert.NotContains(t, prompt, "with say")
			assert.Contains(t, prompt, "Do not call wait_for_user")
			assert.Contains(t, prompt, "When the user creates or skips a draft")
			assert.Contains(t, prompt, "When the user starts a refine")
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
}

func Test__DescribePlanningSession__IncludesPendingSurvey(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	upsertHostedOnboardingProvider(t, db)
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
	upsertHostedOnboardingProvider(t, database.DB(t.Context()))
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

func Test__StartPlanningSession__KeepsPreviousSessionForSameUser(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	upsertHostedOnboardingProvider(t, database.DB(t.Context()))
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
	upsertHostedOnboardingProvider(t, database.DB(t.Context()))
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
	upsertHostedOnboardingProvider(t, database.DB(t.Context()))
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

func Test__StartPlanningSession__UsesClaudeAndDefaultParallelism(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	upsertHostedOnboardingProvider(t, database.DB(t.Context()))
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
	foundClaude := false
	for _, node := range nodes {
		if node.Type == models.NodeTypeComponent {
			assert.Equal(t, "runnerClaudeCode", node.ComponentName())
			require.NotNil(t, node.ConcurrencyMax)
			assert.Equal(t, models.DefaultFactoryLineStepMaxParallelism, *node.ConcurrencyMax)
			foundClaude = true
		}
	}
	assert.True(t, foundClaude)
}

func Test__StartPlanningSession__KeepsClaudeWhenSetupIsCodex(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	upsertHostedOnboardingProvider(t, db)
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
			assert.Equal(t, "runnerClaudeCode", node.ComponentName())
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
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	_, err = StartPlanningSession(ctx, r.Organization.ID.String(), &pb.StartPlanningSessionRequest{
		FactoryId:  factoryModel.ID.String(),
		Repository: "acme/payments",
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "Connect Claude before you start Create with an Agent.")
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
	upsertHostedOnboardingProvider(t, database.DB(t.Context()))
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
