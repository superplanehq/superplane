package factories

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/proto"
)

func TestSubscriptionConnectionsAndCatalogsFollowTheTaskOwner(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureCodexSubscriptions))
	f, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	subscription := true
	require.NoError(t, f.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &subscription}))
	owner := support.CreateUser(t, r, r.Organization.ID)
	connection, err := models.CreateFactoryCodexConnection(db, r.Organization.ID, f.ID, owner.ID, time.Now())
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
	created, err := CreateWorkOrder(caller, r.Organization.ID.String(), &pb.CreateWorkOrderRequest{FactoryId: f.ID.String(), Title: "Unassigned task", Model: "saved-choice"})
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
	require.Error(t, err)
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
	require.Error(t, err)
	persisted, err := f.FindWorkOrder(db, uuid.MustParse(created.Order.Id))
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{owner.ID}, persisted.AssigneeIDs())
	started, err := DispatchWorkOrder(caller, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{FactoryId: f.ID.String(), OrderId: created.Order.Id, LineName: "ship", Model: "owner-model"})
	require.NoError(t, err)
	assert.Equal(t, owner.ID.String(), started.Order.Assignees[0].Id)
}

func codexLineModelIDs(list []*pb.FactoryLLMModel) []string {
	var ids []string
	for _, item := range list {
		ids = append(ids, item.Id)
	}
	return ids
}
