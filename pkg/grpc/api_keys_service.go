package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/apikeys"
	pb "github.com/superplanehq/superplane/pkg/protos/api_keys"
)

type APIKeysService struct {
	pb.UnimplementedApiKeysServer
	authService authorization.Authorization
}

func NewAPIKeysService(authService authorization.Authorization) *APIKeysService {
	return &APIKeysService{
		authService: authService,
	}
}

func (s *APIKeysService) CreateAPIKey(ctx context.Context, req *pb.CreateAPIKeyRequest) (response *pb.CreateAPIKeyResponse, err error) {
	defer func() { securityChange(ctx, "api_key.create", "api_key", req, response, err) }()
	return apikeys.CreateAPIKey(ctx, req, s.authService)
}

func (s *APIKeysService) ListAPIKeys(ctx context.Context, req *pb.ListAPIKeysRequest) (*pb.ListAPIKeysResponse, error) {
	return apikeys.ListAPIKeys(ctx)
}

func (s *APIKeysService) DescribeAPIKey(ctx context.Context, req *pb.DescribeAPIKeyRequest) (*pb.DescribeAPIKeyResponse, error) {
	return apikeys.DescribeAPIKey(ctx, req)
}

func (s *APIKeysService) UpdateAPIKey(ctx context.Context, req *pb.UpdateAPIKeyRequest) (response *pb.UpdateAPIKeyResponse, err error) {
	defer func() { securityChange(ctx, "api_key.update", "api_key", req, response, err) }()
	return apikeys.UpdateAPIKey(ctx, req)
}

func (s *APIKeysService) DeleteAPIKey(ctx context.Context, req *pb.DeleteAPIKeyRequest) (response *pb.DeleteAPIKeyResponse, err error) {
	defer func() { securityChange(ctx, "api_key.delete", "api_key", req, response, err) }()
	return apikeys.DeleteAPIKey(ctx, req, s.authService)
}

func (s *APIKeysService) RegenerateAPIKeyToken(ctx context.Context, req *pb.RegenerateAPIKeyTokenRequest) (response *pb.RegenerateAPIKeyTokenResponse, err error) {
	defer func() { securityChange(ctx, "api_key.rotate", "api_key", req, response, err) }()
	return apikeys.RegenerateAPIKeyToken(ctx, req)
}
