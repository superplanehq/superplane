package workers

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
)

// scopedDummyWebhookHandler wraps DummyWebhookHandler and satisfies core.ScopeKeyer.
// Tests that need a handler reporting a fixed scope key use this type; tests that need
// a handler WITHOUT ScopeKeyer use *impl.DummyWebhookHandler directly.
type scopedDummyWebhookHandler struct {
	*impl.DummyWebhookHandler
	key string
}

func (h *scopedDummyWebhookHandler) ScopeKey(_ any) (string, error) {
	return h.key, nil
}

func newScopedHandler(key string, opts impl.DummyWebhookHandlerOptions) *scopedDummyWebhookHandler {
	return &scopedDummyWebhookHandler{
		DummyWebhookHandler: impl.NewDummyWebhookHandler(opts),
		key:                 key,
	}
}

func wMakeWebhook(t *testing.T, appInstallationID *uuid.UUID, state, mode string) *models.Webhook {
	t.Helper()
	now := time.Now()
	cfg := datatypes.NewJSONType[any](map[string]any{"repo": "owner/repo"})
	w := &models.Webhook{
		ID:                uuid.New(),
		State:             state,
		Secret:            []byte("test-secret"),
		Configuration:     cfg,
		AppInstallationID: appInstallationID,
		ProvisioningMode:  mode,
		MaxRetries:        5,
		CreatedAt:         &now,
		UpdatedAt:         &now,
	}
	require.NoError(t, database.Conn().Create(w).Error)
	return w
}

func wMakeWebhookWithScope(t *testing.T, appInstallationID *uuid.UUID, state, mode, scopeKey string) *models.Webhook {
	t.Helper()
	w := wMakeWebhook(t, appInstallationID, state, mode)
	require.NoError(t, database.Conn().Model(w).Update("scope_key", scopeKey).Error)
	w.ScopeKey = &scopeKey
	return w
}

func wMakeOp(t *testing.T, webhookID uuid.UUID, opType, state string, nextAttemptAt time.Time) *models.WebhookOperation {
	t.Helper()
	now := time.Now()
	op := &models.WebhookOperation{
		WebhookID:      webhookID,
		OperationType:  opType,
		IdempotencyKey: fmt.Sprintf("test-%s-%s", opType, uuid.New().String()),
		State:          state,
		MaxAttempts:    5,
		NextAttemptAt:  nextAttemptAt,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	require.NoError(t, database.Conn().Create(op).Error)
	return op
}

func wMakeCanvas(t *testing.T, orgID uuid.UUID) *models.Canvas {
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

func wMakeNode(t *testing.T, canvasID uuid.UUID, nodeID string, webhookID, installID *uuid.UUID) *models.CanvasNode {
	t.Helper()
	now := time.Now()
	n := &models.CanvasNode{
		WorkflowID:        canvasID,
		NodeID:            nodeID,
		Type:              models.NodeTypeTrigger,
		State:             models.CanvasNodeStateReady,
		WebhookID:         webhookID,
		AppInstallationID: installID,
		Ref:               datatypes.NewJSONType(models.NodeRef{}),
		Configuration:     datatypes.NewJSONType(map[string]any{}),
		Metadata:          datatypes.NewJSONType(map[string]any{}),
		Position:          datatypes.NewJSONType(models.Position{}),
		CreatedAt:         &now,
		UpdatedAt:         &now,
	}
	require.NoError(t, database.Conn().Create(n).Error)
	return n
}

func wMakeBinding(t *testing.T, orgID, installID, canvasID uuid.UUID, nodeID, scopeKey string, webhookID *uuid.UUID) *models.WebhookSubscriptionBinding {
	t.Helper()
	now := time.Now()
	b := &models.WebhookSubscriptionBinding{
		OrganizationID:    orgID,
		AppInstallationID: installID,
		WorkflowID:        canvasID,
		NodeID:            nodeID,
		WebhookID:         webhookID,
		ScopeKey:          scopeKey,
		RequestedConfig:   datatypes.NewJSONType[any](map[string]any{"repo": "owner/repo"}),
		RequestedHash:     "test-hash",
		Active:            true,
		CreatedAt:         &now,
		UpdatedAt:         &now,
	}
	require.NoError(t, database.Conn().Create(b).Error)
	return b
}
