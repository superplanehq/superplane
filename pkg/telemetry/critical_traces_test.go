package telemetry

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsCriticalHTTPRoute(t *testing.T) {
	t.Run("critical canvas endpoints", func(t *testing.T) {
		assert.True(t, IsCriticalHTTPRoute("/api/v1/canvases/{canvas_id}/runs"))
		assert.True(t, IsCriticalHTTPRoute("/api/v1/canvases/{canvas_id}/versions"))
		assert.True(t, IsCriticalHTTPRoute("/api/v1/canvases/{canvas_id}/repository/file"))
		assert.True(t, IsCriticalHTTPRoute("/api/v1/canvases/{canvas_id}/memory"))
	})

	t.Run("critical auth and org endpoints", func(t *testing.T) {
		assert.True(t, IsCriticalHTTPRoute("/api/v1/me"))
		assert.True(t, IsCriticalHTTPRoute("/api/v1/organizations/{id}"))
		assert.True(t, IsCriticalHTTPRoute("/api/v1/organizations/{id}/usage"))
	})

	t.Run("non-critical endpoints", func(t *testing.T) {
		assert.False(t, IsCriticalHTTPRoute("/api/v1/canvases"))
		assert.False(t, IsCriticalHTTPRoute("/api/v1/canvases/{canvas_id}/nodes/{node_id}/events"))
		assert.False(t, IsCriticalHTTPRoute(""))
	})
}

func TestIsCriticalHTTPHandler(t *testing.T) {
	assert.True(t, IsCriticalHTTPHandler("GET", "/organizations"))
	assert.False(t, IsCriticalHTTPHandler("POST", "/organizations"))
	assert.False(t, IsCriticalHTTPHandler("GET", "/account"))
}

func TestIsCriticalGRPCMethod(t *testing.T) {
	assert.True(t, IsCriticalGRPCMethod("/Superplane.Me.Me/Me"))
	assert.True(t, IsCriticalGRPCMethod("/Superplane.Canvases.Canvases/ListRuns"))
	assert.False(t, IsCriticalGRPCMethod("/Superplane.Canvases.Canvases/ListCanvases"))
}

func TestMayTraceHTTPRequest(t *testing.T) {
	t.Run("matches critical paths before route template is known", func(t *testing.T) {
		assert.True(t, MayTraceHTTPRequest(&http.Request{
			Method: http.MethodGet,
			URL:    mustParseURL("/api/v1/me?include_permissions=true"),
		}))
		assert.True(t, MayTraceHTTPRequest(&http.Request{
			Method: http.MethodGet,
			URL:    mustParseURL("/api/v1/canvases/4fc1e729-3e55-4347-b15a-47048be5d9f4/runs?limit=25"),
		}))
		assert.True(t, MayTraceHTTPRequest(&http.Request{
			Method: http.MethodGet,
			URL:    mustParseURL("/organizations"),
		}))
	})

	t.Run("ignores non-critical paths", func(t *testing.T) {
		assert.False(t, MayTraceHTTPRequest(&http.Request{
			Method: http.MethodGet,
			URL:    mustParseURL("/api/v1/canvases"),
		}))
		assert.False(t, MayTraceHTTPRequest(&http.Request{
			Method: http.MethodGet,
			URL:    mustParseURL("/api/v1/canvases/abc/nodes/node-1/events"),
		}))
	})
}

func mustParseURL(raw string) *url.URL {
	parsed, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}

	return parsed
}
