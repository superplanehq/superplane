package grpc

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func TestFactoryPullRequestGatewayRoutes(t *testing.T) {
	mux := runtime.NewServeMux()
	require.NoError(t, pb.RegisterFactoriesHandlerServer(t.Context(), mux, &pb.UnimplementedFactoriesServer{}))

	for _, test := range []struct {
		method string
		path   string
		status int
	}{
		{http.MethodPost, "/api/v1/factories/factory-1/prs/pr-1/merge", http.StatusNotFound},
		{http.MethodGet, "/api/v1/factories/factory-1/prs/pr-1/mergeability", http.StatusNotImplemented},
	} {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(`{}`))
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			assert.Equal(t, test.status, response.Code)
		})
	}
}
