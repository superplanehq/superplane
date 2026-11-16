package public

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/llm"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestResolveHostedListModelsBaseURL(t *testing.T) {
	proxy := "https://proxy.example/v1"
	stored := "https://saved.example/v1"

	assert.Equal(t, proxy, resolveHostedListModelsBaseURL(&proxy, stored))
	empty := ""
	assert.Equal(t, "", resolveHostedListModelsBaseURL(&empty, stored))
	assert.Equal(t, stored, resolveHostedListModelsBaseURL(nil, stored))
	assert.Equal(t, "", resolveHostedListModelsBaseURL(nil, ""))
}

func TestAdminHostedAnthropicSaveRefreshesCapabilities(t *testing.T) {
	r := support.Setup(t)
	var unavailable atomic.Bool
	provider := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if unavailable.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		assert.Equal(t, "/models", req.URL.Path)
		assert.Equal(t, "hosted-key", req.Header.Get("x-api-key"))
		_, _ = io.WriteString(w, `{"data":[{"id":"claude-haiku-5-5","capabilities":{"effort":{"supported":true,"high":{"supported":true}}}}]}`)
	}))
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	require.NoError(t, provider.Listener.Close())
	provider.Listener = listener
	provider.Start()
	defer provider.Close()
	// Use the container hostname so the real HTTP context reaches this local
	// provider without changing the application's transport or network policy.
	hostname, err := os.Hostname()
	require.NoError(t, err)
	baseURL := fmt.Sprintf("http://%s:%d", hostname, listener.Addr().(*net.TCPAddr).Port)
	body, err := json.Marshal(map[string]any{"api_key": "hosted-key", "base_url": baseURL})
	require.NoError(t, err)
	request := mux.SetURLVars(httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(string(body))), map[string]string{"provider": models.UsageProviderAnthropic})
	response := httptest.NewRecorder()
	server := &Server{encryptor: r.Encryptor, registry: r.Registry}
	server.adminUpdateHostedLLMProvider(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	rows, err := models.ListLLMModelCapabilities(database.DB(t.Context()), models.UsageProviderAnthropic)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, baseURL, rows[0].BaseURL)
	assert.Equal(t, datatypes.JSONSlice[string]{"high"}, rows[0].ThinkingLevels)
	saved, err := models.FindHostedLLMProvider(database.DB(t.Context()), models.UsageProviderAnthropic)
	require.NoError(t, err)
	key, err := llm.DecryptAPIKey(t.Context(), r.Encryptor, saved.Provider, saved.APIKey)
	require.NoError(t, err)
	assert.Equal(t, "hosted-key", key)
	unavailable.Store(true)
	request = mux.SetURLVars(httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(string(body))), map[string]string{"provider": models.UsageProviderAnthropic})
	response = httptest.NewRecorder()
	server.adminUpdateHostedLLMProvider(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	after, err := models.ListLLMModelCapabilities(database.DB(t.Context()), models.UsageProviderAnthropic)
	require.NoError(t, err)
	assert.Equal(t, rows, after)
}
