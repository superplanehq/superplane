package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestSubscriptionOwnerAdmissionDistinguishesLoginContention(t *testing.T) {
	for _, scenario := range []string{"ongoing login", "revoked login", "stale connection", "subscription unavailable", "removed member"} {
		t.Run(scenario, func(t *testing.T) {
			db, connection, loginTask, _ := newDisconnectedCodexLogin(t)
			factory, err := models.FindFactory(db, connection.OrganizationID, connection.FactoryID)
			require.NoError(t, err)
			order, err := factory.CreateWorkOrder(db, "Await sign-in", "", &connection.OwnerUserID, []uuid.UUID{connection.OwnerUserID}, nil)
			require.NoError(t, err)
			subscription := true
			require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &subscription}))
			switch scenario {
			case "revoked login":
				require.NoError(t, db.Model(&models.FactoryCodexLease{}).Where("task_id = ?", loginTask.ID).Update("revoked_at", time.Now()).Error)
			case "stale connection":
				require.NoError(t, models.MarkCodexConnectionUncertain(db, loginTask.ID, time.Now()))
			case "subscription unavailable":
				require.NoError(t, models.DisableExperimentalFeature(connection.OrganizationID, features.FeatureCodexSubscriptions))
			case "removed member":
				member, err := models.FindActiveUserByIDInTransaction(db, connection.OrganizationID.String(), connection.OwnerUserID.String())
				require.NoError(t, err)
				require.NoError(t, member.SoftDelete(db, time.Now(), ""))
			}

			ownerID, err := order.RequireSubscriptionOwner(db, true)
			require.Error(t, err)
			assert.Equal(t, uuid.Nil, ownerID)
			if scenario == "ongoing login" {
				var busy *models.CodexConnectionBusyError
				require.ErrorAs(t, err, &busy)
				assert.Equal(t, loginTask.ID, busy.TaskID)
				assert.False(t, models.IsPermanentSubscriptionAdmissionError(err))
				return
			}
			assert.True(t, models.IsPermanentSubscriptionAdmissionError(err), "a login lease cannot make an unavailable subscription retryable: %v", err)
		})
	}
}

func TestValidateSubscriptionCanvasModelsChecksLiteralChoicesOnly(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureCodexSubscriptions))
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	subscription := true
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &subscription}))
	connection, err := models.CreateFactoryCodexConnection(db, r.Organization.ID, factory.ID, r.User, time.Now())
	require.NoError(t, err)
	require.NoError(t, connection.StoreUsage(db, nil, []models.CodexModel{{Model: "gpt-5.4"}}, time.Now()))
	require.NoError(t, db.Model(connection).Updates(map[string]any{
		"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("bundle"),
	}).Error)
	order, err := factory.CreateWorkOrder(db, "Pick a model", "", &r.User, []uuid.UUID{r.User}, nil)
	require.NoError(t, err)

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{
			NodeID:        "codex",
			Type:          models.NodeTypeComponent,
			Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runnerCodex"}}),
			Configuration: datatypes.NewJSONType(map[string]any{"credentials": map[string]any{"source": "subscription"}}),
		},
	}, nil)

	for _, tc := range []struct {
		name     string
		model    string
		expected error
	}{
		{"available literal", "gpt-5.4", nil},
		{"unavailable literal", "gpt-4.1", models.ErrSubscriptionModelUnavailable},
		{"expression", "{{ 'gpt-4.1' }}", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := order.ValidateSubscriptionCanvasModels(db, canvas.ID, tc.model)
			if tc.expected == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.expected)
		})
	}
}
