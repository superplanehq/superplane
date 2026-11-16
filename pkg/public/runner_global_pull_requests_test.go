package public

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	runner "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	factoryactions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestRunnerTaskPullRequests(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	db := database.DB(t.Context())
	server := newRunnerArtifactTestServer(t, r, jwt.NewSigner("test"))
	ctx, factory, order := mustTaskLinkedAgentRun(t, r)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryMultiRepoTasks))
	integrationID := uuid.NewString()
	repository := "acme/api"
	provider := models.ProviderGitHub
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSIntegrationID: &integrationID, AppRepository: &repository, VCSProvider: &provider,
	}))
	_, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		Provider: models.FactoryPullRequestProviderGitHub, ExternalID: "1", Repository: "acme/api", Number: 7,
		URL: "https://github.com/acme/api/pull/7", Title: "Change the API", State: models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)
	token := runnerTokenFrom(runner.AttachRunnerEnv(ctx, nil, 60))
	require.NotEmpty(t, token)

	rec := runnerRequest(t, server, http.MethodGet, token, "/api/v1/runner/task/pull-requests", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var listed struct {
		PullRequests []runnerPullRequestView `json:"pull_requests"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
	assert.Equal(t, []runnerPullRequestView{{
		Repository: "acme/api", Number: 7, URL: "https://github.com/acme/api/pull/7", Title: "Change the API", State: "open",
	}}, listed.PullRequests)

	rec = runnerRequest(t, server, http.MethodPost, token, "/api/v1/runner/task/pull-requests",
		`{"repository":"acme/other","head":"feature","base":"main","title":"Change other"}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "The pull request repository is not in this workspace.")

	readOnly := runnerTokenWithScopes(t, token, runner.ScopePullRequestsRead)
	rec = runnerRequest(t, server, http.MethodPost, readOnly, "/api/v1/runner/task/pull-requests",
		`{"repository":"acme/api","head":"feature","base":"main","title":"Change"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
}

func TestRunnerPullRequestErrorReportsAGitHubTimeout(t *testing.T) {
	rec := httptest.NewRecorder()

	writeRunnerPullRequestError(rec, fmt.Errorf("%w: %w", factoryactions.ErrTaskPullRequestGitHubTimeout, context.DeadlineExceeded))

	assert.Equal(t, http.StatusGatewayTimeout, rec.Code)
}

func TestRunnerPullRequestErrorReportsBitbucketFailures(t *testing.T) {
	rec := httptest.NewRecorder()
	writeRunnerPullRequestError(rec, fmt.Errorf("%w: %w", factoryactions.ErrTaskPullRequestBitbucketTimeout, context.DeadlineExceeded))
	assert.Equal(t, http.StatusGatewayTimeout, rec.Code)

	rec = httptest.NewRecorder()
	writeRunnerPullRequestError(rec, &bitbucket.APIError{StatusCode: http.StatusBadRequest, Message: "branch not found"})
	assert.Equal(t, http.StatusBadGateway, rec.Code)
	assert.Contains(t, rec.Body.String(), "branch not found")

	rec = httptest.NewRecorder()
	writeRunnerPullRequestError(rec, factoryactions.ErrTaskPullRequestBitbucketNotConnected)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestRunnerPublicationCredentialAuthorization(t *testing.T) {
	r := support.Setup(t)
	t.Setenv("JWT_SECRET", "test")
	db := database.DB(t.Context())
	server := newRunnerArtifactTestServer(t, r, jwt.NewSigner("test"))
	ctx, factory, _ := mustTaskLinkedAgentRun(t, r)
	require.NoError(t, db.Model(&models.CanvasNodeExecution{}).Where("id = ?", ctx.ID).Update("state", models.CanvasNodeExecutionStateStarted).Error)
	integration := models.NewIntegration(uuid.New(), r.Organization.ID, "github", support.RandomName("github"), nil)
	integration.State = models.IntegrationStateReady
	setup := datatypes.NewJSONType(models.SetupState{})
	integration.SetupState = &setup
	integration.Properties = datatypes.NewJSONSlice([]core.IntegrationPropertyDefinition{{Name: common.PropertyAuthMethod, Value: common.AuthMethodPAT}})
	require.NoError(t, db.Create(integration).Error)
	secrets := contexts.NewIntegrationSecretStorage(db, r.Encryptor, integration)
	require.NoError(t, secrets.Create(core.IntegrationSecretDefinition{Name: common.SecretPAT, Value: "publication-test-token"}))
	repository, integrationID := "acme/api", integration.ID.String()
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{AppRepository: &repository, VCSIntegrationID: &integrationID}))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryMultiRepoTasks))
	for _, resource := range []struct {
		repository, access string
		enabled            bool
	}{
		{"acme/read", models.FactoryContextRepoAccessRead, true},
		{"acme/disabled", models.FactoryContextRepoAccessWrite, false},
		{"acme/write", models.FactoryContextRepoAccessWrite, true},
	} {
		_, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", resource.enabled, models.FactoryAgentResourceConfig{Repository: resource.repository, Access: resource.access})
		require.NoError(t, err)
	}
	token := runnerTokenFrom(runner.AttachRunnerEnv(ctx, nil, 60))
	endpoint := "/api/v1/runner/task/publication-credential"
	for _, test := range []struct {
		name, body string
		status     int
	}{
		{"primary", `{"repository":"acme/api","host":"github.com"}`, http.StatusOK},
		{"writable context", `{"repository":"acme/write","host":"github.com"}`, http.StatusOK},
		{"foreign repository", `{"repository":"other/api","host":"github.com"}`, http.StatusForbidden},
		{"foreign host", `{"repository":"acme/api","host":"foreign.example"}`, http.StatusForbidden},
		{"read only", `{"repository":"acme/read","host":"github.com"}`, http.StatusForbidden},
		{"disabled", `{"repository":"acme/disabled","host":"github.com"}`, http.StatusForbidden},
		{"caller task", `{"repository":"acme/api","host":"github.com","task":"foreign"}`, http.StatusBadRequest},
		{"caller workspace", `{"repository":"acme/api","host":"github.com","workspace":"foreign"}`, http.StatusBadRequest},
		{"caller integration", `{"repository":"acme/api","host":"github.com","integration":"foreign"}`, http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			rec := runnerRequest(t, server, http.MethodPost, token, endpoint, test.body)
			require.Equal(t, test.status, rec.Code, rec.Body.String())
			assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
			if test.status != http.StatusOK {
				assert.NotContains(t, rec.Body.String(), "publication-test-token")
			}
		})
	}
	body := `{"repository":"acme/api","host":"github.com"}`
	readOnly := runnerTokenWithScopes(t, token, runner.ScopePullRequestsRead)
	assert.Equal(t, http.StatusForbidden, runnerRequest(t, server, http.MethodPost, readOnly, endpoint, body).Code)
	scope, err := runner.ParseRunnerToken(jwt.NewSigner("test"), token)
	require.NoError(t, err)
	for _, mutate := range []func(*runner.RunnerScope){
		func(s *runner.RunnerScope) { s.WorkOrderID = uuid.New() },
		func(s *runner.RunnerScope) { s.FactoryID = uuid.New() },
		func(s *runner.RunnerScope) { s.OrganizationID = uuid.New() },
		func(s *runner.RunnerScope) { s.CanvasRunID = uuid.New() },
		func(s *runner.RunnerScope) { s.NodeExecutionID = uuid.New() },
	} {
		foreign := *scope
		mutate(&foreign)
		foreignToken, err := runner.MintRunnerToken(jwt.NewSigner("test"), foreign, time.Hour)
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, runnerRequest(t, server, http.MethodPost, foreignToken, endpoint, body).Code)
	}
	require.NoError(t, models.DisableExperimentalFeature(r.Organization.ID, features.FeatureFactoryMultiRepoTasks))
	assert.Equal(t, http.StatusForbidden, runnerRequest(t, server, http.MethodPost, token, endpoint, `{"repository":"acme/write","host":"github.com"}`).Code)
	require.NoError(t, db.Model(integration).Update("state", models.IntegrationStatePending).Error)
	assert.Equal(t, http.StatusConflict, runnerRequest(t, server, http.MethodPost, token, endpoint, body).Code)
	require.NoError(t, db.Model(&models.CanvasNodeExecution{}).Where("id = ?", ctx.ID).Update("state", models.CanvasNodeExecutionStateFinished).Error)
	assert.Equal(t, http.StatusForbidden, runnerRequest(t, server, http.MethodPost, token, endpoint, body).Code)
}
