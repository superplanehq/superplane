package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
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
