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
)

func Test__StartPlanningSession__CreatesSessionAndPendingRun(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	appRepo := "acme/payments"
	require.NoError(t, factoryModel.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
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
	require.NotEmpty(t, resp.Session.Messages)
	assert.Equal(t, models.PlanningSessionGreeting, resp.Session.Messages[0].Text)

	canvas, err := models.FindPlanningCanvas(database.DB(t.Context()), r.Organization.ID, factoryModel.ID)
	require.NoError(t, err)
	nodes, err := models.FindCanvasNodesInTransaction(database.DB(t.Context()), canvas.ID)
	require.NoError(t, err)
	hasAgent := false
	for _, node := range nodes {
		if node.Type == models.NodeTypeComponent {
			hasAgent = true
		}
	}
	assert.True(t, hasAgent)

	apps, err := ListFactoryApps(ctx, r.Organization.ID.String(), &pb.ListFactoryAppsRequest{
		FactoryId: factoryModel.ID.String(),
	})
	require.NoError(t, err)
	require.NotEmpty(t, apps.Apps)
	listed := false
	for _, app := range apps.Apps {
		if app.Id == canvas.ID.String() {
			listed = true
			assert.Equal(t, models.PlanningCanvasName, app.Name)
			assert.Equal(t, models.PlanningCanvasDescription, app.Description)
		}
	}
	assert.True(t, listed)

	described, err := DescribePlanningSession(ctx, r.Organization.ID.String(), &pb.DescribePlanningSessionRequest{
		FactoryId: factoryModel.ID.String(),
		SessionId: resp.Session.Id,
	})
	require.NoError(t, err)
	assert.Equal(t, resp.Session.Id, described.Session.Id)
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
	require.GreaterOrEqual(t, len(sent.Session.Messages), 2)

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

	ended, err := EndPlanningSession(ctx, r.Organization.ID.String(), &pb.EndPlanningSessionRequest{
		FactoryId: factoryModel.ID.String(),
		SessionId: sessionID,
	})
	require.NoError(t, err)
	assert.Equal(t, models.PlanningSessionStateEnded, ended.Session.State)
}
