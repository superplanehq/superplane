package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	pbActions "github.com/superplanehq/superplane/pkg/protos/actions"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubActionsServer struct {
	pbActions.UnimplementedActionsServer
}

func (stubActionsServer) ListActions(context.Context, *pbActions.ListActionsRequest) (*pbActions.ListActionsResponse, error) {
	return &pbActions.ListActionsResponse{}, nil
}

func TestUnaryChainRunsAuthorizationInterceptor(t *testing.T) {
	authService, err := authorization.NewAuthService()
	require.NoError(t, err)

	chain := NewUnaryChain(authService)
	server := WrapActionsServer(stubActionsServer{}, chain)

	_, err = server.ListActions(context.Background(), &pbActions.ListActionsRequest{})
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.NotFound, st.Code())
}
