package openrouter

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestCreateKeyUsesManagementBearerAndReturnsHashOnce(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		response(http.StatusOK, `{"key":"sk-or-v1-child","data":{"hash":"abc123"}}`),
	}}
	client := NewManagementClient(httpContext, "sk-or-mgmt")

	created, err := client.CreateKey(CreateKeyRequest{
		Name:      "superplane-run-1",
		ExpiresAt: "2026-09-09T12:41:00Z",
	})
	require.NoError(t, err)
	assert.Equal(t, "sk-or-v1-child", created.Key)
	assert.Equal(t, "abc123", created.Hash)

	require.Len(t, httpContext.Requests, 1)
	req := httpContext.Requests[0]
	assert.Equal(t, http.MethodPost, req.Method)
	assert.Equal(t, "https://openrouter.ai/api/v1/keys", req.URL.String())
	assert.Equal(t, "Bearer sk-or-mgmt", req.Header.Get("Authorization"))
	assert.Equal(t, attributionReferer, req.Header.Get("HTTP-Referer"))
	assert.Equal(t, attributionTitle, req.Header.Get("X-Title"))

	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	var payload map[string]string
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, "superplane-run-1", payload["name"])
	assert.Equal(t, "2026-09-09T12:41:00Z", payload["expires_at"])
}

func TestCreateKeyFailsOnProviderError(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		response(http.StatusForbidden, `{"error":{"code":403,"message":"Only management keys can perform this operation"}}`),
	}}
	client := NewManagementClient(httpContext, "sk-or-inference")

	_, err := client.CreateKey(CreateKeyRequest{Name: "superplane-run-1", ExpiresAt: "2026-09-09T12:41:00Z"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}

func TestDeleteKeyByHash(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		response(http.StatusOK, `{"deleted":true}`),
	}}
	client := NewManagementClient(httpContext, "sk-or-mgmt")

	require.NoError(t, client.DeleteKey("abc123"))
	require.Len(t, httpContext.Requests, 1)
	req := httpContext.Requests[0]
	assert.Equal(t, http.MethodDelete, req.Method)
	assert.Equal(t, "https://openrouter.ai/api/v1/keys/abc123", req.URL.String())
	assert.Equal(t, "Bearer sk-or-mgmt", req.Header.Get("Authorization"))
}

func TestDeleteKeyTreatsMissingKeyAsSuccess(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		response(http.StatusNotFound, `{"error":{"message":"Not found","code":404}}`),
	}}
	client := NewManagementClient(httpContext, "sk-or-mgmt")
	require.NoError(t, client.DeleteKey("already-gone"))
}
