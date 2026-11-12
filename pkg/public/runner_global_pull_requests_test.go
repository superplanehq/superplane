package public

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	runner "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	factoryactions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
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
