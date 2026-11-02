package models_test

import (
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestScheduledSubscriptionReservationUsesPersistedVersionBeforeTaskExists(t *testing.T) {
	db, connection, _, _ := newReadyCodexConnection(t)
	factory, err := models.FindFactory(db, connection.OrganizationID, support.CodexWorkspaceID(t, db, connection.OrganizationID))
	require.NoError(t, err)
	enabled := true
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &enabled}))
	canvas, _ := support.CreateCanvas(t, connection.OrganizationID, connection.OwnerUserID, scheduledSubscriptionNodes(connection.OwnerUserID), nil)
	require.NoError(t, db.Model(canvas).Updates(map[string]any{"factory_id": factory.ID, "column_key": models.CanvasColumnKeyBacklog}).Error)
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "schedule", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	// A newer live default and untrusted event input must not redirect this run.
	live, err := models.FindLiveCanvasVersionInTransaction(db, canvas.ID)
	require.NoError(t, err)
	nodes := slices.Clone(live.Nodes)
	nodes[2].Configuration = maps.Clone(nodes[2].Configuration)
	nodes[2].Configuration["defaultOwnerId"] = uuid.NewString()
	version, err := models.CreateCommitVersionWithSpecInTransaction(db, canvas.ID, connection.OwnerUserID, "Change intake default", nodes, nil)
	require.NoError(t, err)
	require.NoError(t, models.PromoteToLiveInTransaction(db, version, nodes, live.Edges))
	require.NoError(t, db.Model(run).Update("input", models.NewJSONValue(map[string]any{"defaultOwnerId": uuid.NewString()})).Error)
	reserved, err := models.ReserveFactoryCodexConnection(db, connection.OrganizationID, factory.ID, run.ID, uuid.New(), time.Now(), crypto.NewNoOpEncryptor())
	require.NoError(t, err)
	assert.Equal(t, connection.ID, reserved.ID)
	require.NoError(t, db.Model(connection).Update("status", models.CodexConnectionStatusDisconnected).Error)
	_, err = models.ReserveFactoryCodexConnection(db, connection.OrganizationID, factory.ID, run.ID, uuid.New(), time.Now(), crypto.NewNoOpEncryptor())
	require.ErrorIs(t, err, models.ErrSubscriptionOwnerInvalid)
}

func TestScheduledSubscriptionRejectsUntrustedOrAmbiguousRunAssociations(t *testing.T) {
	for _, scenario := range []struct {
		name          string
		column        string
		entrypoint    string
		mutateNodes   func([]models.CanvasNode)
		multipleTasks bool
		childRun      bool
	}{
		{name: "wrong board column", column: models.CanvasColumnKeyVerify},
		{name: "wrong root entrypoint", entrypoint: "scheduled-agent"},
		{name: "missing server template marker", mutateNodes: func(nodes []models.CanvasNode) { nodes[0].Metadata = datatypes.NewJSONType(map[string]any{}) }},
		{name: "conflicting template marker", mutateNodes: func(nodes []models.CanvasNode) {
			nodes[2].Metadata = datatypes.NewJSONType(models.FactoryAppTemplateMetadata("custom-verification", 1))
		}},
		{name: "wrong proposal agent", mutateNodes: func(nodes []models.CanvasNode) {
			nodes[1].Ref = datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}})
		}},
		{name: "multiple associated tasks cannot use the intake fallback", multipleTasks: true},
		{name: "multiple parent tasks cannot use the intake fallback", multipleTasks: true, childRun: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			db, connection, _, _ := newReadyCodexConnection(t)
			factory, err := models.FindFactory(db, connection.OrganizationID, support.CodexWorkspaceID(t, db, connection.OrganizationID))
			require.NoError(t, err)
			enabled := true
			require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &enabled}))
			nodes := scheduledSubscriptionNodes(connection.OwnerUserID)
			if scenario.mutateNodes != nil {
				scenario.mutateNodes(nodes)
			}
			canvas, _ := support.CreateCanvas(t, connection.OrganizationID, connection.OwnerUserID, nodes, nil)
			column := scenario.column
			if column == "" {
				column = models.CanvasColumnKeyBacklog
			}
			require.NoError(t, db.Model(canvas).Updates(map[string]any{"factory_id": factory.ID, "column_key": column}).Error)
			entrypoint := scenario.entrypoint
			if entrypoint == "" {
				entrypoint = "schedule"
			}
			run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, entrypoint, models.CanvasRunStateStarted, "")
			require.NoError(t, err)
			require.NoError(t, db.Model(run).Update("input", models.NewJSONValue(map[string]any{"defaultOwnerId": connection.OwnerUserID.String()})).Error)
			if scenario.multipleTasks {
				for range 2 {
					_, err := factory.CreateWorkOrder(db, "An associated task", "", nil, []uuid.UUID{connection.OwnerUserID}, &run.ID)
					require.NoError(t, err)
				}
			}
			if scenario.childRun {
				child, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "schedule", models.CanvasRunStateStarted, "")
				require.NoError(t, err)
				require.NoError(t, db.Model(child).Update("parent_run_id", run.ID).Error)
				run = child
			}
			_, err = models.ReserveFactoryCodexConnection(db, connection.OrganizationID, factory.ID, run.ID, uuid.New(), time.Now(), crypto.NewNoOpEncryptor())
			require.ErrorIs(t, err, models.ErrSubscriptionTaskAssociation)
		})
	}
}

func scheduledSubscriptionNodes(ownerID uuid.UUID) []models.CanvasNode {
	return []models.CanvasNode{
		{NodeID: "schedule", Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}), Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata("scheduled-intake", 1))},
		{NodeID: "scheduled-agent", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runnerCodex"}}), Configuration: datatypes.NewJSONType(map[string]any{"credentials": map[string]any{"source": "subscription"}})},
		{NodeID: "create-task", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "createWorkOrder"}}), Configuration: datatypes.NewJSONType(map[string]any{"defaultOwnerId": ownerID.String()})},
	}
}
