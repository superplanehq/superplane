package factories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
)

func TestCustomAutomationChatLifecycle(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))
	factory := seedHandOffFactory(t, r, db)
	integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &integrationID}))
	claudeID := createReadyOnboardingIntegration(t, r.Organization.ID, "claude")
	_, err := models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderAnthropic, datatypes.JSONSlice[string]{"claude-sonnet-4-6"})
	require.NoError(t, err)
	ctx, deps := onboardingContext(r), onboardingDeps(r)
	onTask := true
	created, err := CreateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.CreateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), Name: "Discuss this task", ColumnKey: models.CanvasColumnKeyBacklog,
		Settings: &pb.FactoryCustomAutomation_Settings{
			StartMode: pb.FactoryCustomAutomation_START_MODE_MANUAL, RunsOnTask: &onTask,
			Agent: &pb.FactoryAppTemplateAgent{
				Component: "runnerClaudeCode", Model: "claude-sonnet-4-6", CredentialSource: runner.CredentialsSourceIntegration,
				CredentialIntegrationName: integrationName(t, r.Organization.ID, claudeID),
			},
			Steps: []*pb.FactoryCustomAutomation_Step{
				{Name: "Check", Type: pb.FactoryCustomAutomation_Step_TYPE_BASH, Command: "pwd"},
				{Name: "Discuss", Type: pb.FactoryCustomAutomation_Step_TYPE_CHAT, Prompt: "Discuss the task without changing its plan."},
			},
		},
	})
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Investigate", "Keep this description.", &r.User, nil, nil)
	require.NoError(t, err)
	otherOrder, err := factory.CreateWorkOrder(db, "Another task", "", &r.User, nil, nil)
	require.NoError(t, err)
	start := func() *pb.RunFactoryCustomAutomationResponse {
		t.Helper()
		started, err := RunFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.RunFactoryCustomAutomationRequest{
			FactoryId: factory.ID.String(), AutomationId: created.CustomAutomation.Automation.Id, WorkOrderId: order.ID.String(),
		})
		require.NoError(t, err)
		return started
	}
	first, second := start(), start()
	lookup := func(runID string) *pb.PlanningSession {
		t.Helper()
		found, err := FindPlanningSessionByWorkOrder(ctx, r.Organization.ID.String(), &pb.FindPlanningSessionByWorkOrderRequest{
			FactoryId: factory.ID.String(), WorkOrderId: order.ID.String(), RunId: runID,
		})
		require.NoError(t, err)
		return found.Session
	}
	firstSession, secondSession := lookup(first.RunId), lookup(second.RunId)
	require.NotEqual(t, firstSession.Id, secondSession.Id, "each run owns its conversation")
	_, err = FindPlanningSessionByWorkOrder(ctx, r.Organization.ID.String(), &pb.FindPlanningSessionByWorkOrderRequest{
		FactoryId: factory.ID.String(), WorkOrderId: otherOrder.ID.String(), RunId: first.RunId,
	})
	code, _, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	require.Equal(t, codes.NotFound, code)
	otherOrg := support.CreateOrganization(t, r, r.User)
	_, err = FindPlanningSessionByWorkOrder(ctx, otherOrg.ID.String(), &pb.FindPlanningSessionByWorkOrderRequest{
		FactoryId: factory.ID.String(), WorkOrderId: order.ID.String(), RunId: first.RunId,
	})
	require.Error(t, err, "another organization cannot read the conversation")

	planningCanvas, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "backlog", "start")
	planningRun, err := models.CreateCanvasRunInTransaction(db, planningCanvas.ID, "start", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	planning, err := factory.AttachAnalysisSession(db, models.AttachAnalysisSessionParams{
		Repository: "acme/app", CanvasID: planningCanvas.ID, CanvasRunID: planningRun.ID, WorkOrderID: order.ID,
	})
	require.NoError(t, err)
	found, err := FindPlanningSessionByWorkOrder(ctx, r.Organization.ID.String(), &pb.FindPlanningSessionByWorkOrderRequest{
		FactoryId: factory.ID.String(), WorkOrderId: order.ID.String(),
	})
	require.NoError(t, err)
	require.Equal(t, planning.ID.String(), found.Session.Id, "the default lookup remains task planning")

	session, err := models.FindPlanningSessionByRun(db, uuid.MustParse(first.RunId))
	require.NoError(t, err)
	require.NoError(t, session.BeginWait(db))
	// This resembles the old task-creation command; chat must deliver it unchanged.
	text := "Refine TASK-123: Keep the task unchanged."
	sent, err := SendPlanningSessionMessage(ctx, r.Organization.ID.String(), &pb.SendPlanningSessionMessageRequest{
		FactoryId: factory.ID.String(), SessionId: session.ID.String(), Text: text,
	})
	require.NoError(t, err)
	require.Equal(t, first.RunId, sent.Session.CanvasRunId)
	require.Len(t, sent.Session.Messages, 1)
	require.Empty(t, lookup(second.RunId).Messages)
	wait, err := session.ConsumeWait(db)
	require.NoError(t, err)
	require.Equal(t, text, wait.Text)
	unchanged, err := factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	require.Equal(t, order.Description, unchanged.Description)
	checks, err := unchanged.ListChecks(db)
	require.NoError(t, err)
	require.Empty(t, checks)
	artifacts, err := unchanged.ListArtifacts(db)
	require.NoError(t, err)
	require.Empty(t, artifacts)

	original, err := models.FindCanvasRunInTransaction(db, uuid.MustParse(firstSession.CanvasId), uuid.MustParse(first.RunId))
	require.NoError(t, err)
	require.NoError(t, finishCanvasRun(db, original, models.CanvasRunResultPassed))
	require.NoError(t, models.EndPlanningSessionForFinishedRun(db, original.ID, models.CanvasRunResultPassed))
	sent, err = SendPlanningSessionMessage(ctx, r.Organization.ID.String(), &pb.SendPlanningSessionMessageRequest{
		FactoryId: factory.ID.String(), SessionId: session.ID.String(), Text: "Explain the result.",
	})
	require.NoError(t, err)
	require.Equal(t, session.ID.String(), sent.Session.Id)
	require.NotEqual(t, first.RunId, sent.Session.CanvasRunId)
	require.Equal(t, models.PlanningSessionStateRunning, sent.Session.State)
	require.Len(t, sent.Session.Messages, 2)
	require.Equal(t, sent.Session.CanvasRunId, lookup(first.RunId).CanvasRunId, "the original run still locates the conversation")
	resumed, err := models.FindCanvasRunInTransaction(db, original.WorkflowID, uuid.MustParse(sent.Session.CanvasRunId))
	require.NoError(t, err)
	require.Equal(t, original.VersionID, resumed.VersionID, "continuation retains the original prompt version")
}
