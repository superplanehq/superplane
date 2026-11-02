package sentryinbox_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/sentryinbox"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const issueBody = `{"action":"created","installation":{"uuid":"install-1"},"data":{"issue":{"id":"1","project":{"slug":"web"}}}}`

// sealingEncryptor makes ciphertext differ from the plaintext and checks
// the associated data, so a test can see that the payload is encrypted for
// one inbox entry.
type sealingEncryptor struct{}

func (sealingEncryptor) Encrypt(_ context.Context, data, associatedData []byte) ([]byte, error) {
	return append(append([]byte("sealed:"), associatedData...), append([]byte("|"), reverse(data)...)...), nil
}

func (sealingEncryptor) Decrypt(_ context.Context, data, associatedData []byte) ([]byte, error) {
	prefix := append(append([]byte("sealed:"), associatedData...), '|')
	if !bytes.HasPrefix(data, prefix) {
		return nil, errors.New("associated data does not match")
	}
	return reverse(data[len(prefix):]), nil
}

type failingEncryptor struct{}

func (failingEncryptor) Encrypt(context.Context, []byte, []byte) ([]byte, error) {
	return nil, errors.New("key service is unavailable")
}

func (failingEncryptor) Decrypt(context.Context, []byte, []byte) ([]byte, error) {
	return nil, errors.New("key service is unavailable")
}

func reverse(data []byte) []byte {
	reversed := make([]byte, len(data))
	for i := range data {
		reversed[len(data)-1-i] = data[i]
	}
	return reversed
}

func setupHostedConnection(t *testing.T, installationUUID string) (*support.ResourceRegistry, *models.Integration) {
	t.Helper()

	r := support.Setup(t)
	return r, createHostedConnection(t, r, installationUUID)
}

func createHostedConnection(t *testing.T, r *support.ResourceRegistry, installationUUID string) *models.Integration {
	t.Helper()

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "sentry", support.RandomName("sentry"), map[string]any{})
	require.NoError(t, err)
	integration.Metadata = datatypes.NewJSONType(map[string]any{
		"hostedApp":        true,
		"installationUUID": installationUUID,
	})
	require.NoError(t, database.Conn().Save(integration).Error)
	return integration
}

func createListener(t *testing.T, r *support.ResourceRegistry, integration *models.Integration, configuration map[string]any) (*models.Canvas, models.CanvasNode) {
	t.Helper()

	canvas, nodes := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID:        "listener",
		Name:          "listener",
		Type:          models.NodeTypeTrigger,
		Ref:           datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "sentry.onIssue"}}),
		Configuration: datatypes.NewJSONType(configuration),
	}}, nil)

	_, err := models.CreateIntegrationSubscription(&nodes[0], integration, map[string]any{"resources": []string{"issue"}})
	require.NoError(t, err)
	return canvas, nodes[0]
}

func acceptHosted(t *testing.T, resource, body string) sentryinbox.Accepted {
	t.Helper()

	accepted, err := sentryinbox.Accept(t.Context(), database.DB(t.Context()), sealingEncryptor{}, sentryinbox.Delivery{
		Source:   models.SentryWebhookSourceHostedApp,
		Resource: resource,
		Body:     []byte(body),
	})
	require.NoError(t, err)
	return accepted
}

// claimEntry leases one entry like a worker would.
func claimEntry(t *testing.T, id uuid.UUID, lease time.Duration) *models.SentryWebhookInbox {
	t.Helper()

	var claimed []models.SentryWebhookInbox
	require.NoError(t, database.DB(t.Context()).Transaction(func(tx *gorm.DB) error {
		var err error
		claimed, err = models.ClaimSentryWebhookInbox(tx, uuid.New(), time.Now(), lease, 25)
		return err
	}))
	for i := range claimed {
		if claimed[i].ID == id {
			return &claimed[i]
		}
	}
	t.Fatalf("entry %s was not claimable", id)
	return nil
}

func makeDue(t *testing.T, id uuid.UUID) {
	t.Helper()
	require.NoError(t, database.DB(t.Context()).Model(&models.SentryWebhookInbox{}).
		Where("id = ?", id).
		Update("next_attempt_at", time.Now().Add(-time.Second)).Error)
}

func expireLease(t *testing.T, id uuid.UUID) {
	t.Helper()
	require.NoError(t, database.Conn().Model(&models.SentryWebhookInbox{}).
		Where("id = ?", id).
		Update("lease_expires_at", time.Now().Add(-time.Second)).Error)
}

func reload(t *testing.T, id uuid.UUID) models.SentryWebhookInbox {
	t.Helper()
	var entry models.SentryWebhookInbox
	require.NoError(t, database.DB(t.Context()).Where("id = ?", id).First(&entry).Error)
	return entry
}

func receiptOf(t *testing.T, entry models.SentryWebhookInbox) models.SentryWebhookReceipt {
	t.Helper()
	require.NotNil(t, entry.ReceiptID)
	var receipt models.SentryWebhookReceipt
	require.NoError(t, database.DB(t.Context()).Where("id = ?", *entry.ReceiptID).First(&receipt).Error)
	return receipt
}

func targetsOf(t *testing.T, entry models.SentryWebhookInbox) []models.SentryWebhookDeliveryTarget {
	t.Helper()
	targets, err := entry.ListTargets(database.DB(t.Context()))
	require.NoError(t, err)
	return targets
}

func eventsFor(t *testing.T, node models.CanvasNode) []models.CanvasEvent {
	t.Helper()
	var events []models.CanvasEvent
	require.NoError(t, database.DB(t.Context()).
		Where("workflow_id = ? AND node_id = ?", node.WorkflowID, node.NodeID).
		Find(&events).Error)
	return events
}

func newProcessor(r *support.ResourceRegistry) *sentryinbox.Processor {
	return sentryinbox.NewProcessor(sealingEncryptor{}, r.Registry).
		WithEventPublisher(func(*models.CanvasEvent) error { return nil })
}

func useHostedApp(t *testing.T) {
	t.Helper()
	t.Setenv(config.EnvSentryAppSlug, "superplane")
	t.Setenv(config.EnvSentryAppClientID, "cid")
	t.Setenv(config.EnvSentryAppClientSecret, "csecret")
}

func codeDigest(code string) string {
	digest := sha256.Sum256([]byte(code))
	return hex.EncodeToString(digest[:])
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}
