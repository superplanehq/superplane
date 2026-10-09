package runnerapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/jwt"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
	runnerlogsfs "github.com/superplanehq/superplane/pkg/runners/logs/fs"
	"go.opentelemetry.io/otel"
)

func TestReadActiveTaskLogsRequiresReadToken(t *testing.T) {
	const secret = "runner-log-read-secret"
	signer := jwt.NewSigner(secret)
	server, err := NewServer(signer, crypto.NewNoOpEncryptor(), runnerlogs.StoreFS)
	require.NoError(t, err)

	store, err := runnerlogsfs.New(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, store.Setup(runnerlogs.SetupContext{
		Context:       t.Context(),
		MeterProvider: otel.GetMeterProvider(),
	}))
	previousStore := runnerlogs.Current()
	runnerlogs.SetCurrent(store)
	t.Cleanup(func() { runnerlogs.SetCurrent(previousStore) })

	taskID := uuid.New()
	require.NoError(t, store.Initialize(t.Context(), taskID))
	_, err = store.Append(t.Context(), taskID, 0, []byte("secret-line\n"))
	require.NoError(t, err)

	path := "/internal/v1/tasks/" + taskID.String() + "/logs"
	missing := httptest.NewRecorder()
	server.Handler().ServeHTTP(missing, httptest.NewRequest(http.MethodGet, path, nil))
	require.Equal(t, http.StatusUnauthorized, missing.Code)
	assert.NotContains(t, missing.Body.String(), "secret-line")

	expired, err := runnerlogs.SignActiveLogRead(secret, taskID, "", time.Now().Add(-time.Minute))
	require.NoError(t, err)
	expiredResponse := readActiveLogs(t, server, path, expired)
	require.Equal(t, http.StatusUnauthorized, expiredResponse.Code)
	assert.NotContains(t, expiredResponse.Body.String(), "secret-line")

	token, err := runnerlogs.SignActiveLogRead(secret, taskID, "", time.Now())
	require.NoError(t, err)
	ok := readActiveLogs(t, server, path, token)
	require.Equal(t, http.StatusOK, ok.Code)
	assert.Equal(t, "secret-line\n", ok.Body.String())
	assert.NotEmpty(t, ok.Header().Get(runnerlogs.HeaderCursor))
}

func readActiveLogs(t *testing.T, server *Server, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	return rec
}
