package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/grpc/actions/secrets"
	pb "github.com/superplanehq/superplane/pkg/protos/secrets"
)

type SecretService struct {
	encryptor            crypto.Encryptor
	specValidator        executors.SpecValidator
	authorizationService authorization.Authorization
}

func NewSecretService(encryptor crypto.Encryptor, authService authorization.Authorization) *SecretService {
	return &SecretService{
		encryptor:            encryptor,
		specValidator:        executors.SpecValidator{Encryptor: encryptor},
		authorizationService: authService,
	}
}

func (s *SecretService) CreateSecret(ctx context.Context, req *pb.CreateSecretRequest) (*pb.CreateSecretResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainId := ctx.Value(authorization.DomainIdContextKey).(string)
	return secrets.CreateSecret(ctx, s.encryptor, domainType, domainId, req.Secret)
}

func (s *SecretService) UpdateSecret(ctx context.Context, req *pb.UpdateSecretRequest) (*pb.UpdateSecretResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainId := ctx.Value(authorization.DomainIdContextKey).(string)
	return secrets.UpdateSecret(ctx, s.encryptor, domainType, domainId, req.IdOrName, req.Secret)
}

func (s *SecretService) DescribeSecret(ctx context.Context, req *pb.DescribeSecretRequest) (*pb.DescribeSecretResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainId := ctx.Value(authorization.DomainIdContextKey).(string)
	return secrets.DescribeSecret(ctx, s.encryptor, domainType, domainId, req.IdOrName)
}

func (s *SecretService) ListSecrets(ctx context.Context, req *pb.ListSecretsRequest) (*pb.ListSecretsResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainId := ctx.Value(authorization.DomainIdContextKey).(string)
	return secrets.ListSecrets(ctx, s.encryptor, domainType, domainId)
}

func (s *SecretService) DeleteSecret(ctx context.Context, req *pb.DeleteSecretRequest) (*pb.DeleteSecretResponse, error) {
	domainType := ctx.Value(authorization.DomainTypeContextKey).(string)
	domainId := ctx.Value(authorization.DomainIdContextKey).(string)
	return secrets.DeleteSecret(ctx, domainType, domainId, req.IdOrName)
}
