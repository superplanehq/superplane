package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestAutomationSubscriptionUsesWorkspaceDefaultWithoutTask(t *testing.T) {
	db, connection, task, _ := newReadyCodexConnection(t)
	factory, err := models.FindFactory(db, connection.OrganizationID, support.CodexWorkspaceID(t, db, connection.OrganizationID))
	require.NoError(t, err)
	otherUser := &models.User{ID: uuid.New(), OrganizationID: factory.OrganizationID, Type: models.UserTypeHuman, Name: "Another member"}
	require.NoError(t, db.Create(otherUser).Error)
	otherConnection, err := models.CreateCodexConnection(db, factory.OrganizationID, otherUser.ID, time.Now())
	require.NoError(t, err)
	require.NoError(t, db.Model(otherConnection).Updates(map[string]any{"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("other-bundle")}).Error)
	owner := connection.OwnerUserID.String()
	enabled := true
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &enabled, CodexDefaultOwnerID: &owner}))
	canvas, _ := support.CreateCanvas(t, factory.OrganizationID, otherUser.ID, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "trigger", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	// Event input cannot select a different subscription owner.
	require.NoError(t, db.Model(run).Update("input", models.NewJSONValue(map[string]any{"defaultOwnerId": otherUser.ID.String()})).Error)
	event := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "trigger", event.ID, event.ID)
	require.NoError(t, db.Model(execution).Updates(map[string]any{
		"run_id":        run.ID,
		"configuration": datatypes.NewJSONType(map[string]any{"credentials": map[string]any{"source": "subscription"}}),
	}).Error)
	require.NoError(t, models.CreateNodeExecutionKVInTransaction(db, canvas.ID, "trigger", execution.ID, "task_id", task.ID.String()))
	require.NoError(t, db.Model(task).Updates(map[string]any{"state": models.RunnerTaskStateQueued, "runner_id": nil, "finished_at": nil}).Error)

	reserved, err := models.ReserveFactoryCodexConnection(db, factory.OrganizationID, factory.ID, run.ID, task.ID, time.Now(), crypto.NewNoOpEncryptor())
	require.NoError(t, err)
	assert.Equal(t, connection.ID, reserved.ID)
	require.NoError(t, task.RequireSubscriptionLeaseAdmission(db))
	require.NoError(t, task.RequireSubscriptionLeaseContinuation(db))

	// Changing the default affects new leases, while the existing lease keeps its owner.
	owner = otherUser.ID.String()
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexDefaultOwnerID: &owner}))
	require.NoError(t, task.RequireSubscriptionLeaseAdmission(db))
	require.NoError(t, task.RequireSubscriptionLeaseContinuation(db))
	reserved, err = models.ReserveFactoryCodexConnection(db, factory.OrganizationID, factory.ID, run.ID, uuid.New(), time.Now(), crypto.NewNoOpEncryptor())
	require.NoError(t, err)
	assert.Equal(t, otherConnection.ID, reserved.ID)

	require.NoError(t, db.Model(connection).Update("status", models.CodexConnectionStatusDisconnected).Error)
	require.ErrorIs(t, task.RequireSubscriptionLeaseContinuation(db), models.ErrCodexConnectionNotReady)
}

func TestAutomationSubscriptionRejectsInvalidWorkspaceDefault(t *testing.T) {
	for _, scenario := range []string{"missing default", "invalid default", "disconnected owner", "removed owner", "other workspace", "other organization", "missing run"} {
		t.Run(scenario, func(t *testing.T) {
			db, connection, _, _ := newReadyCodexConnection(t)
			factory, err := models.FindFactory(db, connection.OrganizationID, support.CodexWorkspaceID(t, db, connection.OrganizationID))
			require.NoError(t, err)
			owner := connection.OwnerUserID.String()
			enabled := true
			wantErr := models.ErrSubscriptionTaskAssociation
			switch scenario {
			case "missing default":
				owner = ""
			case "invalid default":
				owner = "invalid"
				wantErr = models.ErrSubscriptionOwnerInvalid
			case "disconnected owner":
				wantErr = models.ErrSubscriptionOwnerInvalid
				require.NoError(t, db.Model(connection).Update("status", models.CodexConnectionStatusDisconnected).Error)
			case "removed owner":
				wantErr = models.ErrSubscriptionOwnerInvalid
				require.NoError(t, db.Delete(&models.User{}, "id = ?", connection.OwnerUserID).Error)
			}
			require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &enabled, CodexDefaultOwnerID: &owner}))
			canvas, _ := support.CreateCanvas(t, factory.OrganizationID, connection.OwnerUserID, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}}, nil)
			workspace := factory.ID
			if scenario == "other workspace" {
				other, err := models.CreateFactory(db, factory.OrganizationID, support.RandomName("workspace"), "", "")
				require.NoError(t, err)
				workspace = other.ID
			}
			require.NoError(t, db.Model(canvas).Update("factory_id", workspace).Error)
			run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "trigger", models.CanvasRunStateStarted, "")
			require.NoError(t, err)
			organization := factory.OrganizationID
			if scenario == "other organization" {
				other, err := models.CreateOrganization("Other organization", "")
				require.NoError(t, err)
				require.NoError(t, models.EnableExperimentalFeature(other.ID, features.FeatureFactories))
				organization = other.ID
			}
			if scenario == "missing run" {
				run.ID = uuid.New()
			}
			_, err = models.ReserveFactoryCodexConnection(db, organization, factory.ID, run.ID, uuid.New(), time.Now(), crypto.NewNoOpEncryptor())
			require.ErrorIs(t, err, wantErr)
			assert.Empty(t, codexLeases(t, db, connection.ID))
		})
	}
}
