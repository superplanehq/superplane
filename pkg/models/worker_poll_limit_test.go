package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func Test__ListPendingWebhooks__RespectsLimit(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	now := time.Now()
	for i := 0; i < 5; i++ {
		createdAt := now.Add(time.Duration(i) * time.Minute)
		webhook := &Webhook{
			ID:        uuid.New(),
			State:     WebhookStatePending,
			Secret:    []byte("secret"),
			CreatedAt: &createdAt,
			UpdatedAt: &createdAt,
		}
		require.NoError(t, database.Conn().Create(webhook).Error)
	}

	listed, err := ListPendingWebhooks(2)
	require.NoError(t, err)
	require.Len(t, listed, 2, "poll must not load more rows than the batch limit")
	assert.True(t, listed[0].CreatedAt.Before(*listed[1].CreatedAt),
		"pending webhooks must be returned oldest-first so a deep backlog does not starve early work")
}

func Test__ListIntegrationRequests__RespectsLimit(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	organization, err := CreateOrganization("org-"+uuid.NewString(), "")
	require.NoError(t, err)

	now := time.Now()
	for i := 0; i < 5; i++ {
		integration, err := CreateIntegration(uuid.New(), organization.ID, "dummy", "integration-"+uuid.NewString(), nil)
		require.NoError(t, err)

		runAt := now.Add(time.Duration(i)*time.Minute - time.Hour)
		require.NoError(t, database.Conn().Create(&IntegrationRequest{
			ID:                uuid.New(),
			AppInstallationID: integration.ID,
			State:             IntegrationRequestStatePending,
			Type:              IntegrationRequestTypeSync,
			RunAt:             runAt,
			CreatedAt:         runAt,
			UpdatedAt:         runAt,
		}).Error)
	}

	listed, err := ListIntegrationRequests(2)
	require.NoError(t, err)
	require.Len(t, listed, 2, "poll must not load more rows than the batch limit")
	assert.True(t, listed[0].RunAt.Before(listed[1].RunAt),
		"due integration requests must be returned earliest-run-at first")
}

func Test__ListDeletedWebhooks__RespectsLimit(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	now := time.Now()
	for i := 0; i < 5; i++ {
		createdAt := now.Add(time.Duration(i) * time.Minute)
		webhook := &Webhook{
			ID:        uuid.New(),
			State:     WebhookStateReady,
			Secret:    []byte("secret"),
			CreatedAt: &createdAt,
			UpdatedAt: &createdAt,
		}
		require.NoError(t, database.Conn().Create(webhook).Error)
		require.NoError(t, database.Conn().Delete(webhook).Error)
	}

	listed, err := ListDeletedWebhooks(2)
	require.NoError(t, err)
	require.Len(t, listed, 2, "cleanup poll must not load more rows than the batch limit")
}

func Test__ListPendingRuns__RespectsLimit(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	organization, err := CreateOrganization("org-"+uuid.NewString(), "")
	require.NoError(t, err)

	now := time.Now()
	liveVersionID := uuid.New()
	canvas := &Canvas{
		OrganizationID: organization.ID,
		LiveVersionID:  &liveVersionID,
		Name:           "poll-limit-canvas",
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	require.NoError(t, database.Conn().Create(canvas).Error)
	require.NoError(t, database.Conn().Create(&CanvasVersion{
		ID:         liveVersionID,
		WorkflowID: canvas.ID,
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}).Error)

	for i := 0; i < 5; i++ {
		createdAt := now.Add(time.Duration(i) * time.Minute)
		require.NoError(t, database.Conn().Create(&CanvasRun{
			ID:         uuid.New(),
			WorkflowID: canvas.ID,
			NodeID:     "trigger",
			VersionID:  liveVersionID,
			State:      CanvasRunStatePending,
			CreatedAt:  &createdAt,
			UpdatedAt:  &createdAt,
		}).Error)
	}

	listed, err := ListPendingRuns(database.Conn(), 2)
	require.NoError(t, err)
	require.Len(t, listed, 2, "pending-run sweep must bound work in SQL, not after load")
	assert.True(t, listed[0].CreatedAt.Before(*listed[1].CreatedAt),
		"pending runs must be returned oldest-first")
}
