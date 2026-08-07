package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions/secrets"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/secrets"
)

type SecretService struct {
	encryptor            crypto.Encryptor
	authorizationService authorization.Authorization
}

func NewSecretService(encryptor crypto.Encryptor, authService authorization.Authorization) *SecretService {
	return &SecretService{
		encryptor:            encryptor,
		authorizationService: authService,
	}
}

// resolveDomain determines which secret domain (organization or app/canvas)
// a request should operate on. See secrets.ResolveSecretDomain for details
// on why the organization id always comes from the authorized request
// context rather than the request body/query.
func (s *SecretService) resolveDomain(ctx context.Context, domainType pbAuth.DomainType, domainID string) (string, string, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return secrets.ResolveSecretDomain(organizationID, domainType, domainID)
}

func (s *SecretService) CreateSecret(ctx context.Context, req *pb.CreateSecretRequest) (*pb.CreateSecretResponse, error) {
	domainType, domainId, err := s.resolveDomain(ctx, req.GetDomainType(), req.GetDomainId())
	if err != nil {
		return nil, err
	}
	return secrets.CreateSecret(ctx, s.encryptor, domainType, domainId, req.Secret)
}

func (s *SecretService) UpdateSecret(ctx context.Context, req *pb.UpdateSecretRequest) (*pb.UpdateSecretResponse, error) {
	domainType, domainId, err := s.resolveDomain(ctx, req.GetDomainType(), req.GetDomainId())
	if err != nil {
		return nil, err
	}
	return secrets.UpdateSecret(ctx, s.encryptor, domainType, domainId, req.IdOrName, req.Secret)
}

func (s *SecretService) DescribeSecret(ctx context.Context, req *pb.DescribeSecretRequest) (*pb.DescribeSecretResponse, error) {
	domainType, domainId, err := s.resolveDomain(ctx, req.GetDomainType(), req.GetDomainId())
	if err != nil {
		return nil, err
	}
	return secrets.DescribeSecret(ctx, s.encryptor, domainType, domainId, req.IdOrName)
}

func (s *SecretService) ListSecrets(ctx context.Context, req *pb.ListSecretsRequest) (*pb.ListSecretsResponse, error) {
	domainType, domainId, err := s.resolveDomain(ctx, req.GetDomainType(), req.GetDomainId())
	if err != nil {
		return nil, err
	}
	return secrets.ListSecrets(ctx, s.encryptor, domainType, domainId)
}

func (s *SecretService) DeleteSecret(ctx context.Context, req *pb.DeleteSecretRequest) (*pb.DeleteSecretResponse, error) {
	domainType, domainId, err := s.resolveDomain(ctx, req.GetDomainType(), req.GetDomainId())
	if err != nil {
		return nil, err
	}
	return secrets.DeleteSecret(ctx, domainType, domainId, req.IdOrName)
}

func (s *SecretService) SetSecretKey(ctx context.Context, req *pb.SetSecretKeyRequest) (*pb.SetSecretKeyResponse, error) {
	domainType, domainId, err := s.resolveDomain(ctx, req.GetDomainType(), req.GetDomainId())
	if err != nil {
		return nil, err
	}
	return secrets.SetSecretKey(ctx, s.encryptor, domainType, domainId, req.IdOrName, req.KeyName, req.Value)
}

func (s *SecretService) DeleteSecretKey(ctx context.Context, req *pb.DeleteSecretKeyRequest) (*pb.DeleteSecretKeyResponse, error) {
	domainType, domainId, err := s.resolveDomain(ctx, req.GetDomainType(), req.GetDomainId())
	if err != nil {
		return nil, err
	}
	return secrets.DeleteSecretKey(ctx, s.encryptor, domainType, domainId, req.IdOrName, req.KeyName)
}

func (s *SecretService) UpdateSecretName(ctx context.Context, req *pb.UpdateSecretNameRequest) (*pb.UpdateSecretNameResponse, error) {
	domainType, domainId, err := s.resolveDomain(ctx, req.GetDomainType(), req.GetDomainId())
	if err != nil {
		return nil, err
	}
	return secrets.UpdateSecretName(ctx, s.encryptor, domainType, domainId, req.IdOrName, req.Name)
}
