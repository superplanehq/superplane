package jira

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func newLogger() *logrus.Entry {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logrus.NewEntry(logger)
}

// fakeIntegrationSetupContext is a minimal core.IntegrationSetupContext fake that records the last step set.
type fakeIntegrationSetupContext struct {
	LastStep *core.SetupStep
}

func (f *fakeIntegrationSetupContext) SetStep(step core.SetupStep) error {
	f.LastStep = &step
	return nil
}

func Test__Jira__Sync(t *testing.T) {
	j := &Jira{}

	t.Run("valid credentials -> ready + populated projects", func(t *testing.T) {
		appCtx := newAuthorizedIntegration()

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"accountId":"acct-1","displayName":"Alice"}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"id":"10000","key":"TEST","name":"Test Project"}]`)),
				},
			},
		}

		err := j.Sync(core.SyncContext{
			HTTP:        httpContext,
			Integration: appCtx,
			Logger:      newLogger(),
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)

		meta, ok := appCtx.Metadata.(Metadata)
		require.True(t, ok)
		require.NotNil(t, meta.User)
		assert.Equal(t, "acct-1", meta.User.AccountID)
		assert.Equal(t, testCloudID, meta.CloudID)
		require.Len(t, meta.Projects, 1)
		assert.Equal(t, "TEST", meta.Projects[0].Key)
	})

	t.Run("invalid credentials -> error", func(t *testing.T) {
		appCtx := newAuthorizedIntegration()

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"message":"unauthorized"}`)),
				},
			},
		}

		err := j.Sync(core.SyncContext{
			HTTP:        httpContext,
			Integration: appCtx,
			Logger:      newLogger(),
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "verifying Jira credentials")
	})

	t.Run("missing OAuth connection -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{}

		err := j.Sync(core.SyncContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: appCtx,
			Logger:      newLogger(),
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cloud id")
	})
}

func Test__Jira__HandleRequest_OAuthRedirect(t *testing.T) {
	j := &Jira{}

	newIntegrationWithState := func(state string) *contexts.IntegrationContext {
		appCtx := &contexts.IntegrationContext{}
		require.NoError(t, appCtx.Properties().Create(core.IntegrationPropertyDefinition{Name: PropertyOAuthState, Value: state}))
		require.NoError(t, appCtx.Properties().Create(core.IntegrationPropertyDefinition{Name: PropertyClientID, Value: "client-1"}))
		require.NoError(t, appCtx.Secrets().Create(core.IntegrationSecretDefinition{Name: SecretOAuthClientSecret, Value: "secret-1"}))
		return appCtx
	}

	t.Run("completes the OAuth connection", func(t *testing.T) {
		appCtx := newIntegrationWithState("state-1")
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`{"access_token":"access-1","refresh_token":"refresh-1","expires_in":3600}`,
				))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`[{"id":"cloud-1","name":"Test Site","url":"https://test.atlassian.net"}]`,
				))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"accountId":"acct-1","displayName":"Alice"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[]`))},
			},
		}
		setupCtx := &fakeIntegrationSetupContext{}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/int-1/redirect?code=code-1&state=state-1", nil)
		rec := httptest.NewRecorder()

		j.HandleRequest(core.HTTPRequestContext{
			Logger:           newLogger(),
			Request:          req,
			Response:         rec,
			BaseURL:          "https://superplane.example.com",
			HTTP:             httpCtx,
			Integration:      appCtx,
			IntegrationSetup: setupCtx,
		})

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "https://superplane.example.com", rec.Header().Get("Location"))

		cloudID, err := appCtx.Properties().GetString(PropertyCloudID)
		require.NoError(t, err)
		assert.Equal(t, "cloud-1", cloudID)

		accessToken, err := appCtx.Secrets().Get(SecretOAuthAccessToken)
		require.NoError(t, err)
		assert.Equal(t, "access-1", accessToken)

		require.NotNil(t, setupCtx.LastStep)
		assert.Equal(t, core.SetupStepTypeDone, setupCtx.LastStep.Type)
		assert.Equal(t, "ready", appCtx.State)
	})

	t.Run("rejects a state mismatch", func(t *testing.T) {
		appCtx := newIntegrationWithState("expected-state")
		req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/int-1/redirect?code=code-1&state=wrong-state", nil)
		rec := httptest.NewRecorder()

		j.HandleRequest(core.HTTPRequestContext{
			Logger:      newLogger(),
			Request:     req,
			Response:    rec,
			Integration: appCtx,
		})

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("rejects a missing code", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/int-1/redirect?state=state-1", nil)
		rec := httptest.NewRecorder()

		j.HandleRequest(core.HTTPRequestContext{
			Logger:      newLogger(),
			Request:     req,
			Response:    rec,
			Integration: &contexts.IntegrationContext{},
		})

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
