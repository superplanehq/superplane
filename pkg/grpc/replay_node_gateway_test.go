package grpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/require"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

type replayNodeGatewayStubServer struct {
	pb.UnimplementedCanvasesServer
	replayNodeCalled bool
}

func (s *replayNodeGatewayStubServer) ReplayNode(context.Context, *pb.ReplayNodeRequest) (*pb.ReplayNodeResponse, error) {
	s.replayNodeCalled = true
	return &pb.ReplayNodeResponse{}, nil
}

func TestGRPCGatewayRejectsMalformedJSONForReplayNode(t *testing.T) {
	mux := runtime.NewServeMux()

	server := &replayNodeGatewayStubServer{}
	err := pb.RegisterCanvasesHandlerServer(context.Background(), mux, server)
	require.NoError(t, err)

	requestBody := `{"payload": {"foo": }`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/canvases/11111111-1111-4111-8111-111111111111/nodes/some-node/replay",
		strings.NewReader(requestBody),
	)
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	require.Equal(t, http.StatusBadRequest, response.Code)
	require.False(t, server.replayNodeCalled)
}
