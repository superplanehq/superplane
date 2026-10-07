package public

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gh "github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	githubcommon "github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestHandleGitHubAppSetup(t *testing.T) {
	setGitHubAppEnvironment(t)
	previousRequestRefresh := requestGitHubAppInstallRequestRefresh
	previousInstallationReconciliation := enqueueGitHubAppInstallationReconciliation
	previousHasInstallationRequest := hasGitHubAppInstallationRequest
	organizationID := uuid.New()
	state, err := githubcommon.SignHostedAppInstallState("test-webhook-secret", organizationID)
	require.NoError(t, err)
	var installationIDs []int64
	var organizationIDs []uuid.UUID
	installationRequested := true
	var refreshDeadlines []time.Time
	var requestRefreshError error
	requestGitHubAppInstallRequestRefresh = func(_ context.Context, until time.Time) error {
		refreshDeadlines = append(refreshDeadlines, until)
		return requestRefreshError
	}
	enqueueGitHubAppInstallationReconciliation = func(
		_ context.Context,
		installationID int64,
		requestedOrganizationID uuid.UUID,
		_ time.Time,
	) error {
		installationIDs = append(installationIDs, installationID)
		organizationIDs = append(organizationIDs, requestedOrganizationID)
		return nil
	}
	hasGitHubAppInstallationRequest = func(_ context.Context, _ int64) (bool, error) {
		return installationRequested, nil
	}
	t.Cleanup(func() {
		requestGitHubAppInstallRequestRefresh = previousRequestRefresh
		enqueueGitHubAppInstallationReconciliation = previousInstallationReconciliation
		hasGitHubAppInstallationRequest = previousHasInstallationRequest
	})

	t.Run("direct installation returns to the app", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		(&Server{}).HandleGitHubAppSetup(recorder, httptest.NewRequest(
			http.MethodGet,
			"/api/v1/github/app/setup?installation_id=159131070&setup_action=install&state="+state,
			nil,
		))

		assert.Equal(t, http.StatusFound, recorder.Code)
		assert.Equal(t, "/", recorder.Header().Get("Location"))
		assert.Equal(t, []int64{159131070}, installationIDs)
		assert.Equal(t, []uuid.UUID{organizationID}, organizationIDs)
		assert.Empty(t, refreshDeadlines)
	})

	t.Run("direct listing installation returns to the app", func(t *testing.T) {
		installationRequested = false
		recorder := httptest.NewRecorder()
		(&Server{}).HandleGitHubAppSetup(recorder, httptest.NewRequest(
			http.MethodGet,
			"/api/v1/github/app/setup?installation_id=159131070&setup_action=install",
			nil,
		))

		assert.Equal(t, http.StatusFound, recorder.Code)
		assert.Equal(t, "/", recorder.Header().Get("Location"))
		assert.Equal(t, []int64{159131070, 159131070}, installationIDs)
		assert.Equal(t, []uuid.UUID{organizationID, uuid.Nil}, organizationIDs)
		assert.Empty(t, refreshDeadlines)
	})

	t.Run("approved request opens the confirmation page", func(t *testing.T) {
		installationRequested = true
		recorder := httptest.NewRecorder()
		(&Server{}).HandleGitHubAppSetup(recorder, httptest.NewRequest(
			http.MethodGet,
			"/api/v1/github/app/setup?installation_id=159131070&setup_action=install",
			nil,
		))

		assert.Equal(t, http.StatusFound, recorder.Code)
		assert.Equal(t, "/github/approved", recorder.Header().Get("Location"))
		assert.Equal(t, []int64{159131070, 159131070, 159131070}, installationIDs)
		assert.Equal(t, []uuid.UUID{organizationID, uuid.Nil, uuid.Nil}, organizationIDs)
		assert.Empty(t, refreshDeadlines)
	})

	t.Run("repository update returns to the app", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		(&Server{}).HandleGitHubAppSetup(recorder, httptest.NewRequest(
			http.MethodGet,
			"/api/v1/github/app/setup?installation_id=159131070&setup_action=update&state="+state,
			nil,
		))

		assert.Equal(t, http.StatusFound, recorder.Code)
		assert.Equal(t, "/", recorder.Header().Get("Location"))
		assert.Equal(t, []int64{159131070, 159131070, 159131070, 159131070}, installationIDs)
		assert.Equal(t, []uuid.UUID{organizationID, uuid.Nil, uuid.Nil, organizationID}, organizationIDs)
		assert.Empty(t, refreshDeadlines)
	})

	t.Run("approval request asks for a short request refresh", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		(&Server{}).HandleGitHubAppSetup(recorder, httptest.NewRequest(
			http.MethodGet,
			"/api/v1/github/app/setup?setup_action=request",
			nil,
		))

		assert.Equal(t, http.StatusFound, recorder.Code)
		assert.Equal(t, "/", recorder.Header().Get("Location"))
		require.Len(t, refreshDeadlines, 1)
		assert.WithinDuration(t, time.Now().Add(githubInstallRequestRefreshWindow), refreshDeadlines[0], 5*time.Second)
	})

	t.Run("approval request fails when the refresh cannot be saved", func(t *testing.T) {
		requestRefreshError = errors.New("database is unavailable")
		t.Cleanup(func() { requestRefreshError = nil })
		recorder := httptest.NewRecorder()
		(&Server{}).HandleGitHubAppSetup(recorder, httptest.NewRequest(
			http.MethodGet,
			"/api/v1/github/app/setup?setup_action=request",
			nil,
		))

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
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

func TestGitHubAppSetupOrganizationID(t *testing.T) {
	setGitHubAppEnvironment(t)
	organizationID := uuid.New()
	state, err := githubcommon.SignHostedAppInstallState("test-webhook-secret", organizationID)
	require.NoError(t, err)
	assert.Equal(t, organizationID, githubAppSetupOrganizationID(state))
	assert.Equal(t, uuid.Nil, githubAppSetupOrganizationID(state+"tampered"))
	assert.Equal(t, uuid.Nil, githubAppSetupOrganizationID("o_"+organizationID.String()))
}

func setGitHubAppEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv(githubcommon.EnvGitHubAppID, "12345")
	t.Setenv(githubcommon.EnvGitHubAppSlug, "superplane")
	t.Setenv(
		githubcommon.EnvGitHubAppPrivateKey,
		"-----BEGIN RSA PRIVATE KEY-----\nabc\n-----END RSA PRIVATE KEY-----",
	)
	t.Setenv(githubcommon.EnvGitHubAppWebhookSecret, "test-webhook-secret")
}

func TestGitHubInstallationID(t *testing.T) {
	id, ok := githubInstallationID([]byte(`{"installation":{"id":42}}`))
	require.True(t, ok)
	assert.Equal(t, int64(42), id)

	_, ok = githubInstallationID([]byte(`{"repository":{"id":7}}`))
	assert.False(t, ok)
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

	// Duplicate deliveries are idempotent and preserve the approval row until
	// the browser callback can identify why the installation was created.
	require.NoError(t, applyGitHubCatalogWebhook(db, created, installationID))
	require.NoError(t, applyGitHubCatalogWebhook(db, created, installationID))
	approved, err := models.HasVCSProviderInstallRequestForAccount(db, models.ProviderGitHub, gh.Ptr(accountID), "acme")
	require.NoError(t, err)
	assert.True(t, approved)
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

	// A member event keeps its delayed refresh even when an immediate job is
	// already queued. GitHub can take time to publish collaborator changes.
	memberReceivedAt := time.Now()
	require.NoError(t, applyGitHubCatalogWebhook(db, &gh.MemberEvent{
		Action:       gh.Ptr("edited"),
		Repo:         repository,
		Installation: installation,
	}, installationID))
	require.NoError(t, db.First(&queuedJob, "provider = ? AND repository_id = ?", models.ProviderGitHub, repositoryID).Error)
	assert.WithinDuration(t, memberReceivedAt.Add(10*time.Second), queuedJob.RunAt, time.Second)

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
