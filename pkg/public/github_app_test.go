package public

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gh "github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestHandleGitHubAppSetup(t *testing.T) {
	previous := enqueueGitHubAppReconciliation
	enqueueGitHubAppReconciliation = func(context.Context, time.Time) error { return nil }
	t.Cleanup(func() { enqueueGitHubAppReconciliation = previous })

	t.Run("approved installation returns to onboarding", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		(&Server{}).HandleGitHubAppSetup(recorder, httptest.NewRequest(
			http.MethodGet,
			"/api/v1/github/app/setup?installation_id=159131070&setup_action=install",
			nil,
		))

		assert.Equal(t, http.StatusFound, recorder.Code)
		assert.Equal(t, "/github/approved", recorder.Header().Get("Location"))
	})

	t.Run("repository update returns to the app", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		(&Server{}).HandleGitHubAppSetup(recorder, httptest.NewRequest(
			http.MethodGet,
			"/api/v1/github/app/setup?installation_id=159131070&setup_action=update",
			nil,
		))

		assert.Equal(t, http.StatusFound, recorder.Code)
		assert.Equal(t, "/", recorder.Header().Get("Location"))
	})

	t.Run("approval request returns to the app without an installation id", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		(&Server{}).HandleGitHubAppSetup(recorder, httptest.NewRequest(
			http.MethodGet,
			"/api/v1/github/app/setup?setup_action=request",
			nil,
		))

		assert.Equal(t, http.StatusFound, recorder.Code)
		assert.Equal(t, "/", recorder.Header().Get("Location"))
	})

	t.Run("installation id is required", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		(&Server{}).HandleGitHubAppSetup(recorder, httptest.NewRequest(
			http.MethodGet,
			"/api/v1/github/app/setup?setup_action=install",
			nil,
		))

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}

func TestGitHubInstallationID(t *testing.T) {
	id, ok := githubInstallationID([]byte(`{"installation":{"id":42}}`))
	require.True(t, ok)
	assert.Equal(t, int64(42), id)

	_, ok = githubInstallationID([]byte(`{"repository":{"id":7}}`))
	assert.False(t, ok)
}

func TestVCSProviderRepositoryEvent(t *testing.T) {
	assert.False(t, githubAppRepositoryEvent(&gh.InstallationEvent{}))
	assert.False(t, githubAppRepositoryEvent(&gh.InstallationRepositoriesEvent{}))
	assert.False(t, githubAppRepositoryEvent(&gh.MemberEvent{}))
	assert.True(t, githubAppRepositoryEvent(&gh.PushEvent{}))
	assert.True(t, githubAppRepositoryEvent(&gh.PullRequestEvent{}))
}

func TestDispatchGitHubAppWebhookToEveryMatchingIntegrationWebhook(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	const triggerName = "github-app-fanout"
	deliveries := 0
	r.Registry.Triggers[triggerName] = impl.NewDummyTrigger(impl.DummyTriggerOptions{
		Name: triggerName,
		HandleWebhookFunc: func(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
			deliveries++
			assert.JSONEq(t, `{"repository":{"id":201,"name":"api","full_name":"acme/api"}}`, string(ctx.Body))
			signature := strings.TrimPrefix(ctx.Headers.Get("X-Hub-Signature-256"), "sha256=")
			require.NoError(t, crypto.VerifySignature([]byte("local-webhook-secret"), ctx.Body, signature))
			return http.StatusOK, nil, nil
		},
	})

	server, err := NewServer(
		r.Encryptor,
		r.Registry,
		jwt.NewSigner("test"),
		support.NewOIDCProvider(),
		"",
		"http://localhost",
		"http://localhost",
		"test",
		"/app/templates",
		r.AuthService,
		false,
	)
	require.NoError(t, err)

	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		"github",
		"hosted",
		map[string]any{},
	)
	require.NoError(t, err)
	integration.State = models.IntegrationStateReady
	integration.Metadata = datatypes.NewJSONType(map[string]any{"hostedApp": true})
	require.NoError(t, database.Conn().Save(integration).Error)

	for i := range 3 {
		webhookID := uuid.New()
		repository := "acme/api"
		if i == 2 {
			repository = "acme/other"
		}
		encryptedSecret, encryptErr := r.Encryptor.Encrypt(
			t.Context(),
			[]byte("local-webhook-secret"),
			[]byte(webhookID.String()),
		)
		require.NoError(t, encryptErr)
		webhook := models.Webhook{
			ID:     webhookID,
			State:  models.WebhookStateReady,
			Secret: encryptedSecret,
			Configuration: datatypes.NewJSONType[any](map[string]any{
				"eventType":  "push",
				"repository": repository,
			}),
			AppInstallationID: &integration.ID,
		}
		require.NoError(t, database.Conn().Create(&webhook).Error)

		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
			NodeID: "trigger-" + string(rune('a'+i)),
			Type:   models.NodeTypeTrigger,
			Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: triggerName}}),
		}}, nil)
		require.NoError(t, database.Conn().Model(&models.CanvasNode{}).
			Where("workflow_id = ?", canvas.ID).
			Updates(map[string]any{
				"webhook_id":          webhook.ID,
				"app_installation_id": integration.ID,
			}).Error)
	}

	payload := []byte(`{"repository":{"id":201,"name":"api","full_name":"acme/api"}}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/github/app/webhook", bytes.NewReader(payload))
	request.Header.Set("X-GitHub-Event", "push")
	require.NoError(t, server.dispatchGitHubAppWebhook(request, payload, integration))
	assert.Equal(t, 2, deliveries)
}

func TestGitHubAppWebhookMatchesEventAndRepository(t *testing.T) {
	webhook := &models.Webhook{Configuration: datatypes.NewJSONType[any](map[string]any{
		"eventTypes": []string{"push", "create"},
		"repository": "acme/api",
	})}
	payload := []byte(`{"repository":{"id":201,"name":"api","full_name":"acme/api"}}`)

	assert.True(t, githubAppWebhookMatches(webhook, "push", payload))
	assert.False(t, githubAppWebhookMatches(webhook, "issues", payload))
	assert.False(t, githubAppWebhookMatches(webhook, "push", []byte(`{"repository":{"full_name":"acme/other"}}`)))
}

func TestApplyGitHubCatalogWebhook(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	const installationID = int64(101)
	const repositoryID = int64(201)
	const accountID = int64(301)
	const requesterID = int64(401)

	require.NoError(t, models.ReplaceVCSProviderInstallRequests(db, models.ProviderGitHub, []models.VCSProviderInstallRequest{{
		RequestID:      501,
		AccountID:      gh.Ptr(accountID),
		AccountLogin:   "acme",
		AccountType:    "Organization",
		RequesterID:    requesterID,
		RequesterLogin: "octocat",
		RequestedAt:    time.Now(),
	}}))

	installation := &gh.Installation{
		ID:                  gh.Ptr(installationID),
		Account:             &gh.User{ID: gh.Ptr(accountID), Login: gh.Ptr("acme")},
		HTMLURL:             gh.Ptr("https://github.com/organizations/acme/settings/installations/101"),
		TargetType:          gh.Ptr("Organization"),
		RepositorySelection: gh.Ptr("selected"),
	}
	repository := &gh.Repository{
		ID:            gh.Ptr(repositoryID),
		FullName:      gh.Ptr("acme/api"),
		Private:       gh.Ptr(true),
		DefaultBranch: gh.Ptr("main"),
	}
	created := &gh.InstallationEvent{
		Action:       gh.Ptr("created"),
		Installation: installation,
		Repositories: []*gh.Repository{repository},
	}

	// Duplicate deliveries are idempotent and remove a matching approval row
	// as soon as the installation becomes authoritative.
	require.NoError(t, applyGitHubCatalogWebhook(db, created, installationID))
	require.NoError(t, applyGitHubCatalogWebhook(db, created, installationID))
	requests, err := models.ListVCSProviderInstallRequests(db, models.ProviderGitHub, requesterID)
	require.NoError(t, err)
	assert.Empty(t, requests)
	stored, err := models.FindVCSProviderRepository(db, models.ProviderGitHub, repositoryID)
	require.NoError(t, err)
	assert.Equal(t, "acme/api", stored.FullName)

	var jobCount int64
	require.NoError(t, db.Model(&models.VCSProviderRepositorySyncJob{}).Where("repository_id = ?", repositoryID).Count(&jobCount).Error)
	assert.Equal(t, int64(1), jobCount)
	var queuedJob models.VCSProviderRepositorySyncJob
	require.NoError(t, db.First(&queuedJob, "provider = ? AND repository_id = ?", models.ProviderGitHub, repositoryID).Error)
	assert.Equal(t, models.VCSProviderRepositorySyncPriorityInteractive, queuedJob.Priority)
	assert.WithinDuration(t, time.Now(), queuedJob.RunAt, time.Second)

	// Member changes enqueue the repository again after the previous job is
	// complete so cached push access is refreshed.
	claimAt := time.Now().Add(time.Minute)
	job, err := models.ClaimVCSProviderRepositorySync(db, models.ProviderGitHub, claimAt, claimAt.Add(-time.Minute))
	require.NoError(t, err)
	require.NotNil(t, job)
	require.NoError(t, models.CompleteVCSProviderRepositorySync(db, models.ProviderGitHub, repositoryID, *job.LockedAt))
	require.NoError(t, applyGitHubCatalogWebhook(db, &gh.MemberEvent{
		Action:       gh.Ptr("edited"),
		Repo:         repository,
		Installation: installation,
	}, installationID))
	require.NoError(t, db.Model(&models.VCSProviderRepositorySyncJob{}).Where("repository_id = ?", repositoryID).Count(&jobCount).Error)
	assert.Equal(t, int64(1), jobCount)

	// Repository removal revokes catalog access immediately.
	require.NoError(t, applyGitHubCatalogWebhook(db, &gh.InstallationRepositoriesEvent{
		Action:              gh.Ptr("removed"),
		Installation:        installation,
		RepositoriesRemoved: []*gh.Repository{repository},
		RepositorySelection: gh.Ptr("selected"),
	}, installationID))
	_, err = models.FindVCSProviderRepository(db, models.ProviderGitHub, repositoryID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Uninstall removes the global installation and every local binding, but
	// deleting a binding alone never deletes the global installation.
	binding, err := models.FindOrCreateVCSProviderBinding(db, r.Organization.ID, models.ProviderGitHub, installationID, "acme")
	require.NoError(t, err)
	require.NoError(t, applyGitHubCatalogWebhook(db, &gh.InstallationEvent{
		Action:       gh.Ptr("deleted"),
		Installation: installation,
	}, installationID))
	_, err = models.FindVCSProviderInstallation(db, models.ProviderGitHub, installationID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	_, err = models.FindVCSProviderIntegrationBinding(db, binding.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
