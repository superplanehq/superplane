package githubapp

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

func TestPublicManifestJSON(t *testing.T) {
	raw, err := PublicManifestJSON("https://app.example", "https://hooks.example")
	require.NoError(t, err)

	var manifest map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &manifest))
	assert.Equal(t, false, manifest["public"])
	assert.Equal(t, "SuperPlane", manifest["name"])
	assert.Equal(t, "https://app.example/api/v1/github/app/setup", manifest["setup_url"])
	assert.Equal(t, "https://app.example/api/v1/github/app/created", manifest["redirect_url"])
	hooks, ok := manifest["hook_attributes"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "https://hooks.example/api/v1/github/app/webhook", hooks["url"])
	events, ok := manifest["default_events"].([]any)
	require.True(t, ok)
	assert.Equal(t, []any{"member"}, events)
	permissions, ok := manifest["default_permissions"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "write", permissions["contents"])
	assert.Equal(t, "read", permissions["members"])
}

func TestConvertManifest(t *testing.T) {
	httpCtx := &stubHTTP{
		status: http.StatusCreated,
		body: `{
			"id": 44,
			"slug": "superplane-self",
			"webhook_secret": "whsec",
			"pem": "-----BEGIN RSA PRIVATE KEY-----\nabc\n-----END RSA PRIVATE KEY-----"
		}`,
	}

	cfg, err := ConvertManifest(httpCtx, "manifest-code")
	require.NoError(t, err)
	assert.Equal(t, int64(44), cfg.ID)
	assert.Equal(t, "superplane-self", cfg.Slug)
	assert.Equal(t, "whsec", cfg.WebhookSecret)
	assert.Contains(t, cfg.PrivateKey, "BEGIN RSA PRIVATE KEY")
	require.NotNil(t, httpCtx.request)
	assert.Equal(t, http.MethodPost, httpCtx.request.Method)
	assert.Equal(t, "https://api.github.com/app-manifests/manifest-code/conversions", httpCtx.request.URL.String())
}

func TestConvertManifestRejectsIncompleteApp(t *testing.T) {
	_, err := ConvertManifest(&stubHTTP{status: http.StatusOK, body: `{"id":1,"slug":"x"}`}, "code")
	require.Error(t, err)
}

func TestSafeReturnPath(t *testing.T) {
	assert.Equal(t, "/org/workspaces/new/setup?step=vcs", SafeReturnPath("/org/workspaces/new/setup?step=vcs"))
	assert.Equal(t, "/", SafeReturnPath(""))
	assert.Equal(t, "/", SafeReturnPath("https://evil.example"))
	assert.Equal(t, "/", SafeReturnPath("//evil.example"))
}

type stubHTTP struct {
	status  int
	body    string
	request *http.Request
}

func (s *stubHTTP) Do(request *http.Request) (*http.Response, error) {
	s.request = request
	return &http.Response{
		StatusCode: s.status,
		Body:       io.NopCloser(strings.NewReader(s.body)),
		Header:     make(http.Header),
		Request:    request,
	}, nil
}

var _ core.HTTPContext = (*stubHTTP)(nil)
