package jira

import (
	"io"
	"net/http"
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
					Body:       io.NopCloser(strings.NewReader(`{"cloudId":"35273b54-3f06-40d2-880f-dd28cf6daafa"}`)),
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
		assert.Equal(t, "35273b54-3f06-40d2-880f-dd28cf6daafa", meta.CloudID)
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

	t.Run("missing site URL -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"email":    testEmail,
				"apiToken": testAPIToken,
			},
		}

		err := j.Sync(core.SyncContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: appCtx,
			Logger:      newLogger(),
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "site URL")
	})
}
