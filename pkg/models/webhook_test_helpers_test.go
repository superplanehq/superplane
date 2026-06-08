package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
)

func newOrg(t *testing.T) *models.Organization {
	t.Helper()
	org, err := models.CreateOrganization(uuid.New().String(), "test org")
	require.NoError(t, err)
	return org
}

func newIntegration(t *testing.T, orgID uuid.UUID) *models.Integration {
	t.Helper()
	integration, err := models.CreateIntegration(uuid.New(), orgID, "dummy", uuid.New().String(), nil)
	require.NoError(t, err)
	return integration
}

func newCanvas(t *testing.T, orgID uuid.UUID) *models.Canvas {
	t.Helper()
	now := time.Now()
	c := &models.Canvas{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           uuid.New().String(),
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	require.NoError(t, database.Conn().Create(c).Error)
	return c
}

func newWebhook(t *testing.T, appInstallationID *uuid.UUID, state, mode string) *models.Webhook {
	t.Helper()
	now := time.Now()
	w := &models.Webhook{
		ID:                uuid.New(),
		State:             state,
		Secret:            []byte("test-secret"),
		Configuration:     datatypes.NewJSONType[any](map[string]any{"repo": "owner/repo"}),
		AppInstallationID: appInstallationID,
		ProvisioningMode:  mode,
		MaxRetries:        5,
		CreatedAt:         &now,
		UpdatedAt:         &now,
	}
	require.NoError(t, database.Conn().Create(w).Error)
	return w
}

func newWebhookWithScope(t *testing.T, appInstallationID *uuid.UUID, state, mode, scopeKey string) *models.Webhook {
	t.Helper()
	w := newWebhook(t, appInstallationID, state, mode)
	require.NoError(t, database.Conn().Model(w).Update("scope_key", scopeKey).Error)
	w.ScopeKey = &scopeKey
	return w
}

func newWebhookOp(t *testing.T, webhookID uuid.UUID, opType, state string, nextAttemptAt time.Time) *models.WebhookOperation {
	t.Helper()
	now := time.Now()
	op := &models.WebhookOperation{
		WebhookID:      webhookID,
		OperationType:  opType,
		IdempotencyKey: uuid.New().String(),
		State:          state,
		MaxAttempts:    5,
		NextAttemptAt:  nextAttemptAt,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	require.NoError(t, database.Conn().Create(op).Error)
	return op
}

func newBinding(t *testing.T, orgID, installID, workflowID uuid.UUID, nodeID, scopeKey string, webhookID *uuid.UUID, active bool) *models.WebhookSubscriptionBinding {
	t.Helper()
	now := time.Now()
	b := &models.WebhookSubscriptionBinding{
		OrganizationID:    orgID,
		AppInstallationID: installID,
		WorkflowID:        workflowID,
		NodeID:            nodeID,
		WebhookID:         webhookID,
		ScopeKey:          scopeKey,
		RequestedConfig:   datatypes.NewJSONType[any](map[string]any{"repo": "owner/repo"}),
		RequestedHash:     "test-hash",
		Active:            active,
		CreatedAt:         &now,
		UpdatedAt:         &now,
	}
	require.NoError(t, database.Conn().Create(b).Error)
	return b
}
