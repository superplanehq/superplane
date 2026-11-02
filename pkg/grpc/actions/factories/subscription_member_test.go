package factories

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestSubscriptionConnectionsAndCatalogsFollowTheTaskOwner(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	f, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	subscription := true
	require.NoError(t, f.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &subscription}))
	owner := support.CreateUser(t, r, r.Organization.ID)
	connection, err := models.CreateCodexConnection(db, r.Organization.ID, owner.ID, time.Now())
	require.NoError(t, err)
	require.NoError(t, db.Model(connection).Updates(map[string]any{"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("owner-bundle"), "account_label": "private-owner-account"}).Error)
	require.NoError(t, connection.StoreUsage(db, nil, []models.CodexModel{{Model: "owner-model"}}, time.Now()))
	caller := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
	ownerContext := authentication.SetUserIdInMetadata(t.Context(), owner.ID.String())
	view, err := DescribeFactoryCodexConnection(caller, r.Organization.ID.String(), &pb.DescribeFactoryCodexConnectionRequest{FactoryId: f.ID.String()})
	require.NoError(t, err)
	assert.Empty(t, view.Connection.Id)
	assert.Empty(t, view.Connection.AccountLabel)
	personal, err := DescribeFactoryCodexConnection(ownerContext, r.Organization.ID.String(), &pb.DescribeFactoryCodexConnectionRequest{FactoryId: f.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, connection.ID.String(), personal.Connection.Id)
	assert.Equal(t, "private-owner-account", personal.Connection.AccountLabel)
	_, err = DisconnectFactoryCodexConnection(caller, r.Organization.ID.String(), &pb.DisconnectFactoryCodexConnectionRequest{FactoryId: f.ID.String()})
	require.Error(t, err)
	_, err = CreateWorkOrder(caller, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{FactoryId: f.ID.String(), Title: "Requires my connection", Model: "saved-choice"})
	requireCodexConnectionRequired(t, err)
	created, err := CreateWorkOrder(caller, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{FactoryId: f.ID.String(), Title: "Unassigned task", Model: "saved-choice", Unassigned: true})
	require.NoError(t, err)
	assert.Empty(t, created.Order.Assignees)
	assert.Equal(t, "saved-choice", created.Order.RefinementModel)
	updated, err := UpdateWorkOrder(caller, r.Organization.ID.String(), &pb.UpdateWorkOrderRequest{FactoryId: f.ID.String(), OrderId: created.Order.Id, Title: proto.String("Keep the selected refinement model")})
	require.NoError(t, err)
	assert.Equal(t, "saved-choice", updated.Order.RefinementModel)
	updated, err = UpdateWorkOrder(caller, r.Organization.ID.String(), &pb.UpdateWorkOrderRequest{FactoryId: f.ID.String(), OrderId: created.Order.Id, RefinementModel: proto.String("")})
	require.NoError(t, err)
	assert.Empty(t, updated.Order.RefinementModel)
	explicit, err := CreateWorkOrder(caller, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{FactoryId: f.ID.String(), Title: "Selected owner", AssigneeIds: []string{owner.ID.String()}})
	require.NoError(t, err)
	require.Len(t, explicit.Order.Assignees, 1)
	assert.Equal(t, owner.ID.String(), explicit.Order.Assignees[0].Id)
	unassignedTask, err := CreateWorkOrder(ownerContext, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{FactoryId: f.ID.String(), Title: "Explicitly unassigned", Unassigned: true})
	require.NoError(t, err)
	assert.Empty(t, unassignedTask.Order.Assignees)
	implicit, err := CreateWorkOrder(ownerContext, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{FactoryId: f.ID.String(), Title: "Eligible creator"})
	require.NoError(t, err)
	require.Len(t, implicit.Order.Assignees, 1)
	assert.Equal(t, owner.ID.String(), implicit.Order.Assignees[0].Id)
	_, err = CreateWorkOrder(caller, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{FactoryId: f.ID.String(), Title: "Invalid selection", AssigneeIds: []string{r.User.String()}})
	requireCodexConnectionRequired(t, err)
	app := createLineAppWithRunner(t, r, f.ID, runnerCodex, "subscription", "owner-model")
	_, err = f.CreateLine(db, "ship", []models.FactoryLineStep{{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"}})
	require.NoError(t, err)
	unassigned, err := ListFactoryLineRunnerModels(caller, r.Organization.ID.String(), &pb.ListFactoryLineRunnerModelsRequest{FactoryId: f.ID.String(), LineName: "ship", OrderId: created.Order.Id})
	require.NoError(t, err)
	assert.Empty(t, unassigned.Models)
	_, err = UpdateWorkOrderAssignees(caller, r.Organization.ID.String(), &pb.UpdateWorkOrderAssigneesRequest{FactoryId: f.ID.String(), OrderId: created.Order.Id, AssigneeIds: []string{owner.ID.String()}})
	require.NoError(t, err)
	for _, context := range []struct {
		task     string
		expected []string
	}{{created.Order.Id, []string{"owner-model"}}, {"", nil}} {
		listed, err := ListFactoryLineRunnerModels(caller, r.Organization.ID.String(), &pb.ListFactoryLineRunnerModelsRequest{FactoryId: f.ID.String(), LineName: "ship", OrderId: context.task})
		require.NoError(t, err)
		assert.Equal(t, context.expected, codexLineModelIDs(listed.Models))
	}
	_, err = UpdateWorkOrderAssignees(caller, r.Organization.ID.String(), &pb.UpdateWorkOrderAssigneesRequest{FactoryId: f.ID.String(), OrderId: created.Order.Id, AssigneeIds: []string{r.User.String()}})
	requireCodexConnectionRequired(t, err)
	persisted, err := f.FindWorkOrder(db, uuid.MustParse(created.Order.Id))
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{owner.ID}, persisted.AssigneeIDs())
	_, err = DispatchWorkOrder(caller, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{FactoryId: f.ID.String(), OrderId: created.Order.Id, LineName: "ship", Model: "owner-model"})
	requireCodexConnectionRequired(t, err)
	callerConnection, err := models.CreateCodexConnection(db, r.Organization.ID, r.User, time.Now())
	require.NoError(t, err)
	require.NoError(t, db.Model(callerConnection).Updates(map[string]any{"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("caller-bundle")}).Error)
	require.NoError(t, callerConnection.StoreUsage(db, nil, []models.CodexModel{{Model: "starter-model"}}, time.Now()))
	_, err = DispatchWorkOrder(caller, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{FactoryId: f.ID.String(), OrderId: created.Order.Id, LineName: "ship", Model: "owner-model"})
	require.Error(t, err, "Start must validate the starter's model catalog")
	persisted, err = f.FindWorkOrder(db, uuid.MustParse(created.Order.Id))
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{owner.ID}, persisted.AssigneeIDs(), "a failed dispatch must roll back its ownership change")
	started, err := DispatchWorkOrder(caller, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{FactoryId: f.ID.String(), OrderId: created.Order.Id, LineName: "ship", Model: "starter-model"})
	require.NoError(t, err)
	require.Len(t, started.Order.Assignees, 1)
	assert.Equal(t, r.User.String(), started.Order.Assignees[0].Id)
	require.NoError(t, db.Model(connection).Update("status", models.CodexConnectionStatusDisconnected).Error)
	for _, action := range []struct {
		name string
		run  func() error
	}{
		{name: "Start", run: func() error {
			_, err := DispatchWorkOrder(ownerContext, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{FactoryId: f.ID.String(), OrderId: created.Order.Id, LineName: "ship", ReplaceActive: true})
			return err
		}},
		{name: "self-assignment", run: func() error {
			_, err := UpdateWorkOrderAssignees(ownerContext, r.Organization.ID.String(), &pb.UpdateWorkOrderAssigneesRequest{FactoryId: f.ID.String(), OrderId: created.Order.Id, AssigneeIds: []string{owner.ID.String()}})
			return err
		}},
	} {
		t.Run("active runners block "+action.name+" before connection setup", func(t *testing.T) {
			err := action.run()
			require.Error(t, err)
			assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
			assert.Equal(t, models.ErrSubscriptionOwnerBusy.Error(), grpcerrors.StatusMessage(err))
			assert.Empty(t, status.Convert(err).Details())
			persisted, err := f.FindWorkOrder(db, uuid.MustParse(created.Order.Id))
			require.NoError(t, err)
			assert.Equal(t, []uuid.UUID{r.User}, persisted.AssigneeIDs())
		})
	}
}

func requireCodexConnectionRequired(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	st := status.Convert(err)
	require.Equal(t, codes.FailedPrecondition, st.Code())
	require.Len(t, st.Details(), 1)
	info, ok := st.Details()[0].(*errdetails.ErrorInfo)
	require.True(t, ok)
	assert.Equal(t, "CODEX_CONNECTION_REQUIRED", info.Reason)
}

func TestSubscriptionStartReassignsOpenTasksAndStepRetries(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	f, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	subscription := true
	require.NoError(t, f.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &subscription}))
	previous := support.CreateUser(t, r, r.Organization.ID)
	for _, userID := range []uuid.UUID{previous.ID, r.User} {
		connection, err := models.CreateCodexConnection(db, r.Organization.ID, userID, time.Now())
		require.NoError(t, err)
		require.NoError(t, db.Model(connection).Updates(map[string]any{"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("bundle")}).Error)
		catalog := []models.CodexModel{{Model: "shared-model"}}
		if userID == previous.ID {
			catalog = append(catalog, models.CodexModel{Model: "previous-owner-model"})
		}
		require.NoError(t, connection.StoreUsage(db, nil, catalog, time.Now()))
	}
	app := createLineAppWithRunner(t, r, f.ID, runnerCodex, "subscription", "shared-model")
	line, err := f.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"},
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
	for _, scenario := range []struct {
		name            string
		index           int32
		retry           bool
		savedRetryModel string
	}{
		{name: "open task"},
		{name: "step retry", index: 1, retry: true},
		{name: "saved retry model is unavailable to the starter", index: 1, retry: true, savedRetryModel: "previous-owner-model"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			order, err := f.CreateWorkOrder(db, "Start as the new owner", "", &previous.ID, []uuid.UUID{previous.ID}, nil)
			require.NoError(t, err)
			require.NoError(t, order.TransitionOnDispatch(db, &previous.ID))
			var prior *models.FactoryWorkOrderLineDispatch
			if scenario.savedRetryModel != "" {
				prior = support.CreateFactoryLineDispatch(t, r.Organization.ID, f.ID, order.ID, line.ID, line.Name, line.Steps)
				require.NoError(t, db.Model(prior).Updates(map[string]any{"state": models.FactoryWorkOrderLineDispatchStateFinished, "model": scenario.savedRetryModel, "thinking_level": "low"}).Error)
			}
			response, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
				FactoryId: f.ID.String(), OrderId: order.ID.String(), LineName: line.Name, StartStepIndex: scenario.index, ReplaceActive: scenario.retry, Model: "shared-model",
			})
			if prior != nil {
				require.ErrorIs(t, err, models.ErrSubscriptionModelUnavailable)
				persisted, err := f.FindWorkOrder(db, order.ID)
				require.NoError(t, err)
				assert.Equal(t, []uuid.UUID{previous.ID}, persisted.AssigneeIDs())
				persistedDispatch, err := models.FindWorkOrderLineDispatch(db, prior.ID)
				require.NoError(t, err)
				assert.Equal(t, models.FactoryWorkOrderLineDispatchStateFinished, persistedDispatch.State)
				assert.Equal(t, scenario.savedRetryModel, persistedDispatch.Model)
				assert.Equal(t, "low", persistedDispatch.ThinkingLevel)
				return
			}
			require.NoError(t, err)
			require.Len(t, response.Order.Assignees, 1)
			assert.Equal(t, r.User.String(), response.Order.Assignees[0].Id)
			require.Len(t, response.Order.LineDispatches, 1)
			require.Len(t, response.Order.LineDispatches[0].StepExecutions, 1)
			assert.Equal(t, scenario.index, response.Order.LineDispatches[0].StepExecutions[0].StepIndex)
		})
	}
}

func TestSubscriptionStartTakesDraftWhilePlanningRunnerStops(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	f, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	subscription := true
	require.NoError(t, f.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &subscription}))
	previous := support.CreateUser(t, r, r.Organization.ID)
	connections := map[uuid.UUID]*models.FactoryCodexConnection{}
	for _, userID := range []uuid.UUID{previous.ID, r.User} {
		connection, err := models.CreateCodexConnection(db, r.Organization.ID, userID, time.Now())
		require.NoError(t, err)
		require.NoError(t, db.Model(connection).Updates(map[string]any{
			"status": models.CodexConnectionStatusReady, "credential_version": 1,
			"bundle_ciphertext": []byte("bundle"),
		}).Error)
		require.NoError(t, connection.StoreUsage(db, nil, []models.CodexModel{{Model: "shared-model"}}, time.Now()))
		connections[userID] = connection
	}
	order, err := f.CreateWorkOrder(db, "Draft", "", &previous.ID, []uuid.UUID{previous.ID}, nil)
	require.NoError(t, err)
	planningCanvas := createOnWorkOrderCanvas(t, r, f.ID)
	planningRun, err := models.CreateCanvasRunInTransaction(db, planningCanvas.ID, "start", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := f.AttachAnalysisSession(db, models.AttachAnalysisSessionParams{
		CanvasID: planningCanvas.ID, CanvasRunID: planningRun.ID, WorkOrderID: order.ID,
	})
	require.NoError(t, err)
	event := support.EmitCanvasEventForNode(t, planningCanvas.ID, "start", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, planningCanvas.ID, backlogRefinementNodeID, event.ID, event.ID)
	require.NoError(t, db.Model(execution).Updates(map[string]any{
		"run_id": planningRun.ID, "state": models.CanvasNodeExecutionStateStarted,
	}).Error)
	fleet := models.DefaultInstallationRunnerFleets("test")[0]
	fleet.Slug = support.RandomName("planning")
	require.NoError(t, fleet.Create(db))
	oldTask := &models.RunnerTask{ID: uuid.New(), OrganizationID: r.Organization.ID, FleetID: fleet.ID,
		Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateRunning,
		QueuedAt: time.Now(), PayloadCiphertext: []byte("encrypted")}
	require.NoError(t, db.Create(oldTask).Error)
	require.NoError(t, models.CreateNodeExecutionKVInTransaction(db, planningCanvas.ID, backlogRefinementNodeID, execution.ID, "task_id", oldTask.ID.String()))
	_, err = models.ReserveFactoryCodexConnection(db, f.OrganizationID, f.ID, planningRun.ID, oldTask.ID, time.Now(), crypto.NewNoOpEncryptor())
	require.NoError(t, err)

	app := createLineAppWithRunner(t, r, f.ID, runnerCodex, "subscription", "shared-model")
	line, err := f.CreateLine(db, "ship", []models.FactoryLineStep{{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"}})
	require.NoError(t, err)
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
	eligibility, err := ListFactoryOwnerEligibility(ctx, r.Organization.ID.String(), &pb.ListFactoryOwnerEligibilityRequest{FactoryId: f.ID.String(), OrderId: order.ID.String()})
	require.NoError(t, err)
	assert.False(t, eligibility.OwnershipChangeAllowed)
	_, err = UpdateWorkOrderAssignees(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderAssigneesRequest{FactoryId: f.ID.String(), OrderId: order.ID.String(), AssigneeIds: []string{r.User.String()}})
	require.Error(t, err)
	assert.Equal(t, models.ErrSubscriptionOwnerBusy.Error(), grpcerrors.StatusMessage(err))

	request := &pb.DispatchWorkOrderRequest{FactoryId: f.ID.String(), OrderId: order.ID.String(), LineName: line.Name}
	request.Model = "unavailable-model"
	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), request)
	require.Error(t, err)
	unchanged, err := f.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateDraft, unchanged.State)
	assert.Equal(t, []uuid.UUID{previous.ID}, unchanged.AssigneeIDs())
	stillPlanning, err := models.FindPlanningSession(db, f.OrganizationID, f.ID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.PlanningSessionStateRunning, stillPlanning.State)
	stillRunning, err := models.FindRunnerTask(db, oldTask.ID)
	require.NoError(t, err)
	assert.Nil(t, stillRunning.CancelRequestedAt)
	unrelatedRun, err := models.CreateCanvasRunInTransaction(db, planningCanvas.ID, "start", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	require.NoError(t, order.RecordRefinementRun(db, unrelatedRun.ID))
	request.Model = "shared-model"
	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), request)
	require.Error(t, err)
	assert.Equal(t, models.ErrSubscriptionOwnerBusy.Error(), grpcerrors.StatusMessage(err))
	stillPlanning, err = models.FindPlanningSession(db, f.OrganizationID, f.ID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.PlanningSessionStateRunning, stillPlanning.State)
	require.NoError(t, db.Delete(&models.FactoryWorkOrderRefinementRun{}, "run_id = ?", unrelatedRun.ID).Error)

	started, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), request)
	require.NoError(t, err)
	require.Len(t, started.Order.Assignees, 1)
	assert.Equal(t, r.User.String(), started.Order.Assignees[0].Id)
	completedPlanning, err := models.FindPlanningSession(db, f.OrganizationID, f.ID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.PlanningSessionStateEnded, completedPlanning.State)
	completedRun, err := models.FindUnscopedCanvasRun(db, planningRun.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateCancelling, completedRun.State)
	assert.Equal(t, models.CanvasRunResultPassed, completedRun.Result)
	stoppingTask, err := models.FindRunnerTask(db, oldTask.ID)
	require.NoError(t, err)
	assert.NotNil(t, stoppingTask.CancelRequestedAt)
	assert.ErrorIs(t, stoppingTask.RequireSubscriptionLeaseContinuation(db), models.ErrSubscriptionOwnerInvalid)

	implementationRun, err := models.CreateCanvasRunInTransaction(db, app.ID, "start", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	require.NoError(t, db.Model(&models.FactoryWorkOrderExecution{}).
		Where("work_order_id = ? AND line_dispatch_id = ?", order.ID, uuid.MustParse(started.Order.LineDispatches[0].Id)).
		Update("run_id", implementationRun.ID).Error)
	newTask := &models.RunnerTask{ID: uuid.New(), OrganizationID: r.Organization.ID, FleetID: fleet.ID,
		Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStatePending,
		QueuedAt: time.Now(), PayloadCiphertext: []byte("encrypted")}
	require.NoError(t, db.Create(newTask).Error)
	reserved, err := models.ReserveFactoryCodexConnection(db, f.OrganizationID, f.ID, implementationRun.ID, newTask.ID, time.Now(), crypto.NewNoOpEncryptor())
	require.NoError(t, err)
	assert.Equal(t, connections[r.User].ID, reserved.ID)
	assert.NotEqual(t, connections[previous.ID].ID, reserved.ID)
}

func codexLineModelIDs(list []*pb.FactoryLLMModel) []string {
	var ids []string
	for _, item := range list {
		ids = append(ids, item.Id)
	}
	return ids
}
