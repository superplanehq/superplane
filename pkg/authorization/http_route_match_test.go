package authorization

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchHTTPRoute(t *testing.T) {
	rules := DefaultAuthorizationRules()

	tests := []struct {
		method string
		path   string
		want   HTTPRoute
		wantOK bool
	}{
		{
			method: http.MethodGet,
			path:   "/api/v1/canvases",
			want:   HTTPRoute{Method: http.MethodGet, Pattern: "/api/v1/canvases"},
			wantOK: true,
		},
		{
			method: http.MethodGet,
			path:   "/api/v1/canvases/canvas-123",
			want:   HTTPRoute{Method: http.MethodGet, Pattern: "/api/v1/canvases/{id}"},
			wantOK: true,
		},
		{
			method: http.MethodGet,
			path:   "/api/v1/canvases/canvas-123/runs",
			want:   HTTPRoute{Method: http.MethodGet, Pattern: "/api/v1/canvases/{canvas_id}/runs"},
			wantOK: true,
		},
		{
			method: http.MethodGet,
			path:   "/api/v1/me",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			route, ok := MatchHTTPRoute(tt.method, tt.path, rules)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.want, route)
			}
		})
	}
}

func TestGatewayAuthorizerRouteFromRequestWithoutHTTPPathPattern(t *testing.T) {
	authorizer := NewGatewayAuthorizer(allowingPermissionChecker{})
	r, err := http.NewRequest(http.MethodGet, "http://example.com/api/v1/canvases/canvas-123", nil)
	require.NoError(t, err)

	route, ok := authorizer.RouteFromRequest(r)
	require.True(t, ok)
	assert.Equal(t, HTTPRoute{Method: http.MethodGet, Pattern: "/api/v1/canvases/{id}"}, route)
}
