package workers

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
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

	err := provisioner.LockAndProcessWebhook(webhook)
	require.NoError(t, err)

	updatedWebhook, err := models.FindWebhook(webhookID)
	require.NoError(t, err)
	assert.Equal(t, models.WebhookStateReady, updatedWebhook.State)
	assert.Equal(t, 0, updatedWebhook.RetryCount)
}

func Test__WebhookProvisioner_RetryOnError(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	provisioner := NewWebhookProvisioner("https://example.com", &BadEncryptor{}, r.Registry)

	r.Registry.Applications["dummy"] = support.NewDummyApplicationWithSetupWebhook(
		nil,
		func(ctx core.SetupWebhookContext) (any, error) {
			return nil, errors.New("oops")
		},
	)

	installation, err := models.CreateAppInstallation(
		uuid.New(),
		r.Organization.ID,
		"dummy",
		support.RandomName("installation"),
		nil,
	)

	require.NoError(t, err)

	webhookID := uuid.New()
	webhook := models.Webhook{
		ID:                webhookID,
		State:             models.WebhookStatePending,
		Secret:            []byte("encrypted-secret"),
		AppInstallationID: &installation.ID,
		Resource: datatypes.NewJSONType(models.WebhookResource{
			Type: "repository",
			ID:   "123",
			Name: "test-repo",
		}),
		RetryCount: 0,
		MaxRetries: 3,
	}
	require.NoError(t, database.Conn().Create(&webhook).Error)

	err = provisioner.LockAndProcessWebhook(webhook)
	require.NoError(t, err)

	updatedWebhook, err := models.FindWebhook(webhookID)
	require.NoError(t, err)
	assert.Equal(t, models.WebhookStatePending, updatedWebhook.State)
	assert.Equal(t, 1, updatedWebhook.RetryCount)
}

func Test__WebhookProvisioner_MaxRetriesExceeded(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	provisioner := NewWebhookProvisioner("https://example.com", &BadEncryptor{}, r.Registry)

	r.Registry.Applications["dummy"] = support.NewDummyApplicationWithSetupWebhook(
		nil,
		func(ctx core.SetupWebhookContext) (any, error) {
			return nil, errors.New("oops")
		},
	)

	installation, err := models.CreateAppInstallation(
		uuid.New(),
		r.Organization.ID,
		"dummy",
		support.RandomName("installation"),
		nil,
	)
	require.NoError(t, err)

	webhookID := uuid.New()
	webhook := models.Webhook{
		ID:                webhookID,
		State:             models.WebhookStatePending,
		Secret:            []byte("encrypted-secret"),
		AppInstallationID: &installation.ID,
		Resource: datatypes.NewJSONType(models.WebhookResource{
			Type: "repository",
			ID:   "123",
			Name: "test-repo",
		}),
		RetryCount: 3,
		MaxRetries: 3,
	}
	require.NoError(t, database.Conn().Create(&webhook).Error)

	err = provisioner.LockAndProcessWebhook(webhook)
	require.NoError(t, err)

	updatedWebhook, err := models.FindWebhook(webhookID)
	require.NoError(t, err)
	assert.Equal(t, models.WebhookStateFailed, updatedWebhook.State)
	assert.Equal(t, 3, updatedWebhook.RetryCount)
}

func Test__WebhookProvisioner_ConcurrentProcessing(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

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
		results <- worker1.LockAndProcessWebhook(webhook)
	}()

	go func() {
		worker2 := NewWebhookProvisioner("https://example.com", r.Encryptor, r.Registry)
		results <- worker2.LockAndProcessWebhook(webhook)
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
