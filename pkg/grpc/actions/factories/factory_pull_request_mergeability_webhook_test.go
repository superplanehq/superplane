package factories

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestGitHubMergeabilityWebhookRef(t *testing.T) {
	t.Run("reads a pull request synchronize event", func(t *testing.T) {
		repository, numbers, sha := githubMergeabilityWebhookRef("pull_request", []byte(`{
			"action": "synchronize",
			"repository": {"full_name": "acme/app"},
			"pull_request": {"number": 42, "head": {"sha": "abc123"}}
		}`))
		assert.Equal(t, "acme/app", repository)
		assert.Equal(t, []int64{42}, numbers)
		assert.Equal(t, "abc123", sha)
	})

	t.Run("reads a completed check run", func(t *testing.T) {
		repository, numbers, sha := githubMergeabilityWebhookRef("check_run", []byte(`{
			"repository": {"full_name": "acme/app"},
			"check_run": {
				"head_sha": "def456",
				"pull_requests": [{"number": 7}, {"number": 7}]
			}
		}`))
		assert.Equal(t, "acme/app", repository)
		assert.Equal(t, []int64{7}, numbers)
		assert.Equal(t, "def456", sha)
	})

	t.Run("reads a commit status", func(t *testing.T) {
		repository, numbers, sha := githubMergeabilityWebhookRef("status", []byte(`{
			"sha": "fff999",
			"repository": {"full_name": "acme/app"}
		}`))
		assert.Equal(t, "acme/app", repository)
		assert.Empty(t, numbers)
		assert.Equal(t, "fff999", sha)
	})

	t.Run("ignores unrelated events", func(t *testing.T) {
		repository, numbers, sha := githubMergeabilityWebhookRef("issues", []byte(`{"repository":{"full_name":"acme/app"}}`))
		assert.Equal(t, "", repository)
		assert.Nil(t, numbers)
		assert.Equal(t, "", sha)
	})
}

func TestIsGitHubFactoryMergeabilityEvent(t *testing.T) {
	assert.True(t, IsGitHubFactoryMergeabilityEvent("check_run"))
	assert.True(t, IsGitHubFactoryMergeabilityEvent("ping"))
	assert.True(t, IsGitHubFactoryMergeabilityEvent("PULL_REQUEST"))
	assert.False(t, IsGitHubFactoryMergeabilityEvent("issues"))
	assert.False(t, IsGitHubFactoryMergeabilityEvent(""))
}

func TestEnsureGitHubFactoryMergeabilityWebhook(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		"github",
		support.RandomName("github"),
		map[string]any{},
	)
	require.NoError(t, err)
	require.NoError(t, db.Model(integration).Update("state", models.IntegrationStateReady).Error)

	require.NoError(t, ensureGitHubFactoryMergeabilityWebhook(t.Context(), db, r.Encryptor, integration, "acme/app"))

	webhooks, err := models.ListIntegrationWebhooks(db, integration.ID)
	require.NoError(t, err)
	require.Len(t, webhooks, 1)
	assert.Equal(t, models.WebhookStatePending, webhooks[0].State)
	assert.True(t, isFactoryMergeabilityWebhook(webhooks[0].Configuration.Data()))
	assert.Equal(t, "acme/app", factoryMergeabilityWebhookRepository(webhooks[0].Configuration.Data()))

	firstID := webhooks[0].ID
	require.NoError(t, ensureGitHubFactoryMergeabilityWebhook(t.Context(), db, r.Encryptor, integration, "acme/app"))
	webhooks, err = models.ListIntegrationWebhooks(db, integration.ID)
	require.NoError(t, err)
	require.Len(t, webhooks, 1)
	assert.Equal(t, firstID, webhooks[0].ID)

	require.NoError(t, db.Model(&webhooks[0]).Update("state", models.WebhookStateFailed).Error)
	require.NoError(t, ensureGitHubFactoryMergeabilityWebhook(t.Context(), db, r.Encryptor, integration, "acme/app"))
	webhooks, err = models.ListIntegrationWebhooks(db, integration.ID)
	require.NoError(t, err)
	require.Len(t, webhooks, 1)
	assert.Equal(t, firstID, webhooks[0].ID)
	assert.Equal(t, models.WebhookStatePending, webhooks[0].State)
	assert.Equal(t, "acme/app", factoryMergeabilityWebhookRepository(webhooks[0].Configuration.Data()))

	require.NoError(t, ensureGitHubFactoryMergeabilityWebhook(t.Context(), db, r.Encryptor, integration, "acme/other"))
	webhooks, err = models.ListIntegrationWebhooks(db, integration.ID)
	require.NoError(t, err)
	require.Len(t, webhooks, 2)
	repositories := map[string]struct{}{}
	for _, hook := range webhooks {
		assert.True(t, isFactoryMergeabilityWebhook(hook.Configuration.Data()))
		repositories[factoryMergeabilityWebhookRepository(hook.Configuration.Data())] = struct{}{}
	}
	assert.Equal(t, map[string]struct{}{"acme/app": {}, "acme/other": {}}, repositories)
}

func TestVerifyGitHubFactoryMergeabilitySignature(t *testing.T) {
	r := support.Setup(t)
	webhookID := uuid.New()
	secret := []byte("webhook-secret")
	encrypted, err := r.Encryptor.Encrypt(t.Context(), secret, []byte(webhookID.String()))
	require.NoError(t, err)
	webhook := &models.Webhook{ID: webhookID, Secret: encrypted}
	body := []byte(`{"ok":true}`)

	t.Run("accepts a matching GitHub signature", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+crypto.Sign(secret, body))
		code, err := VerifyGitHubFactoryMergeabilitySignature(t.Context(), r.Encryptor, webhook, headers, body)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
	})

	t.Run("rejects a missing signature", func(t *testing.T) {
		code, err := VerifyGitHubFactoryMergeabilitySignature(t.Context(), r.Encryptor, webhook, http.Header{}, body)
		require.Error(t, err)
		assert.Equal(t, http.StatusForbidden, code)
	})

	t.Run("rejects a mismatched signature", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256=deadbeef")
		code, err := VerifyGitHubFactoryMergeabilitySignature(t.Context(), r.Encryptor, webhook, headers, body)
		require.Error(t, err)
		assert.Equal(t, http.StatusForbidden, code)
	})
}
