package workers

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/gorm"
)

type BadEncryptor struct{}

func (m *BadEncryptor) Encrypt(ctx context.Context, plaintext []byte, aad []byte) ([]byte, error) {
	return nil, fmt.Errorf("oops")
}

func (m *BadEncryptor) Decrypt(ctx context.Context, ciphertext []byte, aad []byte) ([]byte, error) {
	return nil, fmt.Errorf("oops")
}

func Test__WebhookProvisioner_WithoutAppInstallation(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	logger := logrus.NewEntry(logrus.New())
	provisioner := NewWebhookProvisioner("https://example.com", r.Encryptor, r.Registry)
	webhookID := uuid.New()
	webhook := models.Webhook{
		ID:         webhookID,
		State:      models.WebhookStatePending,
		Secret:     []byte("secret"),
		RetryCount: 0,
		MaxRetries: 3,
	}
	require.NoError(t, database.Conn().Create(&webhook).Error)

	err := provisioner.LockAndProcessWebhook(logger, webhook)
	require.NoError(t, err)

	updatedWebhook, err := models.FindWebhook(webhookID)
	require.NoError(t, err)
	assert.Equal(t, models.WebhookStateReady, updatedWebhook.State)
	assert.Equal(t, 0, updatedWebhook.RetryCount)
}

func Test__WebhookProvisioner_RetryOnError(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	logger := logrus.NewEntry(logrus.New())
	provisioner := NewWebhookProvisioner("https://example.com", &BadEncryptor{}, r.Registry)

	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
	r.Registry.WebhookHandlers["dummy"] = impl.NewDummyWebhookHandler(impl.DummyWebhookHandlerOptions{
		SetupFunc: func(ctx core.WebhookHandlerContext) (any, error) {
			return nil, errors.New("oops")
		},
	})

	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		"dummy",
		support.RandomName("integration"),
		nil,
	)

	require.NoError(t, err)

	webhookID := uuid.New()
	webhook := models.Webhook{
		ID:                webhookID,
		State:             models.WebhookStatePending,
		Secret:            []byte("encrypted-secret"),
		AppInstallationID: &integration.ID,
		RetryCount:        0,
		MaxRetries:        3,
	}
	require.NoError(t, database.Conn().Create(&webhook).Error)

	err = provisioner.LockAndProcessWebhook(logger, webhook)
	require.NoError(t, err)

	updatedWebhook, err := models.FindWebhook(webhookID)
	require.NoError(t, err)
	assert.Equal(t, models.WebhookStatePending, updatedWebhook.State)
	assert.Equal(t, 1, updatedWebhook.RetryCount)
}

func Test__WebhookProvisioner_MaxRetriesExceeded(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	logger := logrus.NewEntry(logrus.New())
	provisioner := NewWebhookProvisioner("https://example.com", &BadEncryptor{}, r.Registry)

	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
	r.Registry.WebhookHandlers["dummy"] = impl.NewDummyWebhookHandler(impl.DummyWebhookHandlerOptions{
		SetupFunc: func(ctx core.WebhookHandlerContext) (any, error) {
			return nil, errors.New("oops")
		},
	})

	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		"dummy",
		support.RandomName("integration"),
		nil,
	)
	require.NoError(t, err)

	webhookID := uuid.New()
	webhook := models.Webhook{
		ID:                webhookID,
		State:             models.WebhookStatePending,
		Secret:            []byte("encrypted-secret"),
		AppInstallationID: &integration.ID,
		RetryCount:        3,
		MaxRetries:        3,
	}
	require.NoError(t, database.Conn().Create(&webhook).Error)

	err = provisioner.LockAndProcessWebhook(logger, webhook)
	require.NoError(t, err)

	updatedWebhook, err := models.FindWebhook(webhookID)
	require.NoError(t, err)
	assert.Equal(t, models.WebhookStateFailed, updatedWebhook.State)
	assert.Equal(t, 3, updatedWebhook.RetryCount)
}

func Test__WebhookProvisioner_PersistsIntegrationMetadataWrittenBySetup(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	logger := logrus.NewEntry(logrus.New())
	provisioner := NewWebhookProvisioner("https://example.com", r.Encryptor, r.Registry)

	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
	r.Registry.WebhookHandlers["dummy"] = impl.NewDummyWebhookHandler(impl.DummyWebhookHandlerOptions{
		SetupFunc: func(ctx core.WebhookHandlerContext) (any, error) {
			ctx.Integration.SetMetadata(map[string]any{"webhookId": 34})
			return map[string]any{}, nil
		},
	})

	integration := createDummyIntegration(t, r)
	webhook := createPendingWebhook(t, integration.ID)

	require.NoError(t, provisioner.LockAndProcessWebhook(logger, webhook))

	assertIntegrationMetadata(t, integration.ID, float64(34))
}

func Test__WebhookProvisioner_RetriesWhenPersistingSetupMetadataFails(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	logger := logrus.NewEntry(logrus.New())
	provisioner := NewWebhookProvisioner("https://example.com", r.Encryptor, r.Registry)

	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
	r.Registry.WebhookHandlers["dummy"] = impl.NewDummyWebhookHandler(impl.DummyWebhookHandlerOptions{
		SetupFunc: func(ctx core.WebhookHandlerContext) (any, error) {
			ctx.Integration.SetMetadata(map[string]any{"webhookId": 34})
			return map[string]any{}, nil
		},
	})

	integration := createDummyIntegration(t, r)
	webhook := createPendingWebhook(t, integration.ID)

	db := database.Conn()
	callbackName := "test:fail-app-installation-metadata"
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		table := tx.Statement.Table
		if table == "" && tx.Statement.Schema != nil {
			table = tx.Statement.Schema.Table
		}
		if table != "app_installations" {
			return
		}
		_ = tx.AddError(errors.New("forced metadata write failure"))
	}))
	t.Cleanup(func() {
		db.Callback().Update().Remove(callbackName)
	})

	require.NoError(t, provisioner.LockAndProcessWebhook(logger, webhook))

	updatedWebhook, err := models.FindWebhook(webhook.ID)
	require.NoError(t, err)
	assert.Equal(t, models.WebhookStatePending, updatedWebhook.State)
	assert.Equal(t, 1, updatedWebhook.RetryCount)
	assertIntegrationMetadata(t, integration.ID, nil)
}

func Test__WebhookProvisioner_PersistsIntegrationMetadataWhenSetupFails(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	logger := logrus.NewEntry(logrus.New())
	provisioner := NewWebhookProvisioner("https://example.com", r.Encryptor, r.Registry)

	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
	r.Registry.WebhookHandlers["dummy"] = impl.NewDummyWebhookHandler(impl.DummyWebhookHandlerOptions{
		SetupFunc: func(ctx core.WebhookHandlerContext) (any, error) {
			ctx.Integration.SetMetadata(map[string]any{"webhookId": 35})
			return nil, errors.New("registration rejected after the remote change")
		},
	})

	integration := createDummyIntegration(t, r)
	webhook := createPendingWebhook(t, integration.ID)

	require.NoError(t, provisioner.LockAndProcessWebhook(logger, webhook))

	assertIntegrationMetadata(t, integration.ID, float64(35))
}

func createDummyIntegration(t *testing.T, r *support.ResourceRegistry) *models.Integration {
	t.Helper()

	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		"dummy",
		support.RandomName("integration"),
		nil,
	)
	require.NoError(t, err)
	return integration
}

func createPendingWebhook(t *testing.T, integrationID uuid.UUID) models.Webhook {
	t.Helper()

	webhook := models.Webhook{
		ID:                uuid.New(),
		State:             models.WebhookStatePending,
		Secret:            []byte("encrypted-secret"),
		AppInstallationID: &integrationID,
		RetryCount:        0,
		MaxRetries:        3,
	}
	require.NoError(t, database.Conn().Create(&webhook).Error)
	return webhook
}

func assertIntegrationMetadata(t *testing.T, integrationID uuid.UUID, webhookID any) {
	t.Helper()

	var stored models.Integration
	require.NoError(t, database.Conn().Where("id = ?", integrationID).First(&stored).Error)
	data := stored.Metadata.Data()
	if webhookID == nil {
		if data == nil {
			return
		}
		assert.Nil(t, data["webhookId"])
		return
	}
	require.NotNil(t, data)
	assert.Equal(t, webhookID, data["webhookId"])
}

func Test__WebhookProvisioner_ConcurrentProcessing(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	logger := logrus.NewEntry(logrus.New())
	webhookID := uuid.New()
	webhook := models.Webhook{
		ID:         webhookID,
		State:      models.WebhookStatePending,
		Secret:     []byte("secret"),
		RetryCount: 0,
		MaxRetries: 3,
	}
	require.NoError(t, database.Conn().Create(&webhook).Error)

	results := make(chan error, 2)

	go func() {
		worker1 := NewWebhookProvisioner("https://example.com", r.Encryptor, r.Registry)
		results <- worker1.LockAndProcessWebhook(logger, webhook)
	}()

	go func() {
		worker2 := NewWebhookProvisioner("https://example.com", r.Encryptor, r.Registry)
		results <- worker2.LockAndProcessWebhook(logger, webhook)
	}()

	result1 := <-results
	result2 := <-results
	assert.NoError(t, result1)
	assert.NoError(t, result2)

	updatedWebhook, err := models.FindWebhook(webhookID)
	require.NoError(t, err)
	assert.Equal(t, models.WebhookStateReady, updatedWebhook.State)
}

func Test__WebhookProvisioner_HasExceededRetries(t *testing.T) {
	tests := []struct {
		name       string
		retryCount int
		maxRetries int
		expected   bool
	}{
		{
			name:       "not exceeded",
			retryCount: 2,
			maxRetries: 3,
			expected:   false,
		},
		{
			name:       "exactly at max",
			retryCount: 3,
			maxRetries: 3,
			expected:   true,
		},
		{
			name:       "exceeded",
			retryCount: 4,
			maxRetries: 3,
			expected:   true,
		},
		{
			name:       "zero retries",
			retryCount: 0,
			maxRetries: 3,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			webhook := &models.Webhook{
				RetryCount: tt.retryCount,
				MaxRetries: tt.maxRetries,
			}
			assert.Equal(t, tt.expected, webhook.HasExceededRetries())
		})
	}
}

func Test__WebhookProvisioner_IncrementRetry(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	webhookID := uuid.New()
	webhook := models.Webhook{
		ID:         webhookID,
		State:      models.WebhookStatePending,
		Secret:     []byte("secret"),
		RetryCount: 1,
		MaxRetries: 3,
	}
	require.NoError(t, database.Conn().Create(&webhook).Error)

	err := webhook.IncrementRetry(database.Conn())
	require.NoError(t, err)

	assert.Equal(t, 2, webhook.RetryCount)

	updatedWebhook, err := models.FindWebhook(webhookID)
	require.NoError(t, err)
	assert.Equal(t, 2, updatedWebhook.RetryCount)
}

func Test__WebhookProvisioner_MarkFailed(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	webhookID := uuid.New()
	webhook := models.Webhook{
		ID:         webhookID,
		State:      models.WebhookStatePending,
		Secret:     []byte("secret"),
		RetryCount: 3,
		MaxRetries: 3,
	}
	require.NoError(t, database.Conn().Create(&webhook).Error)

	err := webhook.MarkFailed(database.Conn())
	require.NoError(t, err)

	updatedWebhook, err := models.FindWebhook(webhookID)
	require.NoError(t, err)
	assert.Equal(t, models.WebhookStateFailed, updatedWebhook.State)
}
