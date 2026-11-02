package workers

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/encryption"
	sentryintegration "github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/sentryinbox"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

const sentryWorkerIssueBody = `{"action":"created","installation":{"uuid":"install-1"},"data":{"issue":{"id":"1"}}}`

func setupSentryListener(t *testing.T, r *support.ResourceRegistry) models.CanvasNode {
	t.Helper()

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "sentry", support.RandomName("sentry"), map[string]any{})
	require.NoError(t, err)
	integration.Metadata = datatypes.NewJSONType(map[string]any{"hostedApp": true, "installationUUID": "install-1"})
	require.NoError(t, database.Conn().Save(integration).Error)

	_, nodes := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID:        "listener",
		Name:          "listener",
		Type:          models.NodeTypeTrigger,
		Ref:           datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "sentry.onIssue"}}),
		Configuration: datatypes.NewJSONType(map[string]any{}),
	}}, nil)
	_, err = models.CreateIntegrationSubscription(&nodes[0], integration, map[string]any{"resources": []string{"issue"}})
	require.NoError(t, err)
	return nodes[0]
}

func acceptSentryDelivery(t *testing.T, encryptor crypto.Encryptor, resource, body string) uuid.UUID {
	t.Helper()
	accepted, err := sentryinbox.Accept(t.Context(), database.DB(t.Context()), encryptor, sentryinbox.Delivery{
		Source:   models.SentryWebhookSourceHostedApp,
		Resource: resource,
		Body:     []byte(body),
	})
	require.NoError(t, err)
	return accepted.InboxID
}

func quietSentryWorker(encryptor crypto.Encryptor, r *registry.Registry) *SentryWebhookWorker {
	processor := sentryinbox.NewProcessor(encryptor, r).WithEventPublisher(func(*models.CanvasEvent) error { return nil })
	return NewSentryWebhookWorkerWithProcessor(processor)
}

func countSentryEvents(t *testing.T, node models.CanvasNode) int64 {
	t.Helper()
	var count int64
	require.NoError(t, database.Conn().Model(&models.CanvasEvent{}).
		Where("workflow_id = ? AND node_id = ?", node.WorkflowID, node.NodeID).
		Count(&count).Error)
	return count
}

func Test__SentryWebhookWorker_concurrentWorkersProcessEachDeliveryOnce(t *testing.T) {
	r := support.Setup(t)
	node := setupSentryListener(t, r)
	for i := 0; i < 12; i++ {
		acceptSentryDelivery(t, r.Encryptor, "issue", sentryWorkerIssueBody)
	}

	var wg sync.WaitGroup
	for range 3 {
		worker := quietSentryWorker(r.Encryptor, r.Registry)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				claimed, err := worker.RunOnce(t.Context())
				if err != nil || claimed == 0 {
					return
				}
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(12), countSentryEvents(t, node))
	var unfinished int64
	require.NoError(t, database.Conn().Model(&models.SentryWebhookInbox{}).Where("completed_at IS NULL").Count(&unfinished).Error)
	assert.Zero(t, unfinished)
}

// blockingSentryAPI answers only when the request context ends.
type blockingSentryAPI struct {
	started  chan struct{}
	canceled chan error
	once     sync.Once
}

func (b *blockingSentryAPI) Do(request *http.Request) (*http.Response, error) {
	b.once.Do(func() { close(b.started) })
	<-request.Context().Done()
	select {
	case b.canceled <- request.Context().Err():
	default:
	}
	return nil, request.Context().Err()
}

func Test__SentryWebhookWorker_lostLeaseCancelsSentryRequests(t *testing.T) {
	t.Setenv(config.EnvSentryAppSlug, "superplane")
	t.Setenv(config.EnvSentryAppClientID, "cid")
	t.Setenv(config.EnvSentryAppClientSecret, "csecret")
	r := support.Setup(t)
	sentryintegration.EnableHostedInstallBind(r.Encryptor)
	inboxID := acceptSentryDelivery(t, r.Encryptor, "installation",
		`{"action":"created","installation":{"uuid":"install-slow"},"data":{"installation":{"uuid":"install-slow","code":"grant-code"}}}`)

	sentryAPI := &blockingSentryAPI{started: make(chan struct{}), canceled: make(chan error, 1)}
	processor := sentryinbox.NewProcessor(r.Encryptor, r.Registry).WithHTTP(sentryAPI)
	worker := NewSentryWebhookWorkerWithProcessor(processor).WithLease(time.Minute, 20*time.Millisecond)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = worker.RunOnce(context.Background())
	}()

	select {
	case <-sentryAPI.started:
	case <-time.After(5 * time.Second):
		t.Fatal("the worker did not call Sentry")
	}
	require.NoError(t, database.Conn().Model(&models.SentryWebhookInbox{}).
		Where("id = ?", inboxID).
		Update("lease_owner", uuid.New()).Error)

	select {
	case err := <-sentryAPI.canceled:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("the lost lease did not cancel the Sentry request")
	}
	<-done

	var entry models.SentryWebhookInbox
	require.NoError(t, database.Conn().Where("id = ?", inboxID).First(&entry).Error)
	assert.Nil(t, entry.CompletedAt)
	assert.NotNil(t, entry.PayloadCiphertext)
}

func Test__SentryWebhookWorker_keyRotationKeepsUnfinishedDeliveries(t *testing.T) {
	ctx := t.Context()
	r := support.Setup(t)
	db := database.DB(context.Background())
	clean := func() {
		require.NoError(t, db.Exec("DELETE FROM application_encryption_migrations").Error)
		require.NoError(t, db.Exec("DELETE FROM application_encryption_keys").Error)
	}
	clean()
	t.Cleanup(clean)

	masters, err := crypto.ParseLocalMasterKeys("local-v1:" + base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32)))
	require.NoError(t, err)
	store := encryption.NewStore(masters, "local-v1")
	require.NoError(t, store.Initialize(ctx))
	encryptor := encryption.NewApplicationEncryptor(crypto.NewVersionedEncryptor(store), store)

	node := setupSentryListener(t, r)
	inboxID := acceptSentryDelivery(t, encryptor, "issue", sentryWorkerIssueBody)

	_, err = store.CreateAndActivateDataKey(ctx)
	require.NoError(t, err)
	active, err := models.LoadActiveApplicationEncryptionKey(db)
	require.NoError(t, err)
	location, err := inboxStorageLocation()
	require.NoError(t, err)
	job := &models.ApplicationEncryptionMigration{
		ID: uuid.New(), TargetKeyID: active.ID, State: "pending",
		RegistryVersion: encryption.RegistryVersion, StorageLocation: location.Name,
	}
	require.NoError(t, db.Create(job).Error)

	migration := NewApplicationEncryptionMigrationWorker("", encryptor, r.Registry, []models.EncryptedStorageLocation{location})
	for range 4 {
		_, err := migration.RunBatch(ctx)
		require.NoError(t, err)
	}
	job, err = models.LoadApplicationEncryptionMigration(db, job.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", job.State, job.ErrorMessage)

	var entry models.SentryWebhookInbox
	require.NoError(t, db.Where("id = ?", inboxID).First(&entry).Error)
	keyID, versioned, err := crypto.EncryptedDataKeyID(entry.PayloadCiphertext)
	require.NoError(t, err)
	require.True(t, versioned)
	assert.Equal(t, active.ID, keyID)

	claimed, err := quietSentryWorker(encryptor, r.Registry).RunOnce(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, claimed)
	assert.Equal(t, int64(1), countSentryEvents(t, node))
}

func inboxStorageLocation() (models.EncryptedStorageLocation, error) {
	for _, location := range models.ApplicationEncryptedStorage() {
		if location.Name == "sentry_webhook_inbox.payload_ciphertext" {
			return location, nil
		}
	}
	return models.EncryptedStorageLocation{}, errors.New("the Sentry webhook inbox is not in the encrypted storage inventory")
}
