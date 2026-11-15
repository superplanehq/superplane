package sso

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/enterprise"
	pb "github.com/superplanehq/superplane/pkg/protos/sso"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func organizationID(ctx context.Context) (uuid.UUID, error) {
	value, _ := ctx.Value(authorization.DomainIdContextKey).(string)
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, status.Error(codes.InvalidArgument, "The organization is not valid.")
	}
	return id, nil
}

func accountID(ctx context.Context) (uuid.UUID, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return uuid.Nil, status.Error(codes.Unauthenticated, "account not found")
	}
	values := md.Get("x-account-id")
	if len(values) == 0 || values[0] == "" {
		return uuid.Nil, status.Error(codes.Unauthenticated, "account not found")
	}
	raw := values[0]
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, status.Error(codes.InvalidArgument, "The account is not valid.")
	}
	return id, nil
}

func GetConnection(ctx context.Context, service enterprise.SSO) (*pb.GetConnectionResponse, error) {
	orgID, err := organizationID(ctx)
	if err != nil {
		return nil, err
	}
	view, err := service.GetConnection(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return &pb.GetConnectionResponse{Exists: view.Exists, Connection: connectionToProto(view)}, nil
}

func PutConnection(ctx context.Context, service enterprise.SSO, req *pb.PutConnectionRequest) (*pb.ConnectionResponse, error) {
	orgID, err := organizationID(ctx)
	if err != nil {
		return nil, err
	}
	actor, err := accountID(ctx)
	if err != nil {
		return nil, err
	}
	view, err := service.PutConnection(ctx, orgID, actor, enterprise.SSOConnectionInput{
		Preset:               presetFromProto(req.GetPreset()),
		DisplayName:          req.GetDisplayName(),
		Issuer:               req.GetIssuer(),
		ClientID:             req.GetClientId(),
		ClientSecret:         req.GetClientSecret(),
		Scopes:               req.GetScopes(),
		SubjectClaim:         req.GetSubjectClaim(),
		EmailClaim:           req.GetEmailClaim(),
		NameClaim:            req.GetNameClaim(),
		JITEnabled:           req.GetJitEnabled(),
		DefaultRole:          req.GetDefaultRole(),
		Enforcement:          enforcementFromProto(req.GetEnforcement()),
		AllowOwnerBypass:     req.GetAllowOwnerBypass(),
		MaxSessionAgeSeconds: req.GetMaxSessionAgeSeconds(),
		IDPLogoutEnabled:     req.GetIdpLogoutEnabled(),
	})
	if err != nil {
		return nil, err
	}
	return &pb.ConnectionResponse{Connection: connectionToProto(view)}, nil
}

func ActivateConnection(ctx context.Context, service enterprise.SSO) (*pb.ConnectionResponse, error) {
	orgID, err := organizationID(ctx)
	if err != nil {
		return nil, err
	}
	view, err := service.ActivateConnection(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return &pb.ConnectionResponse{Connection: connectionToProto(view)}, nil
}

func DisableConnection(ctx context.Context, service enterprise.SSO) (*pb.ConnectionResponse, error) {
	orgID, err := organizationID(ctx)
	if err != nil {
		return nil, err
	}
	view, err := service.DisableConnection(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return &pb.ConnectionResponse{Connection: connectionToProto(view)}, nil
}

func DeleteConnection(ctx context.Context, service enterprise.SSO) (*empty.Empty, error) {
	orgID, err := organizationID(ctx)
	if err != nil {
		return nil, err
	}
	if err := service.DeleteConnection(ctx, orgID); err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func ListDomains(ctx context.Context, service enterprise.SSO) (*pb.ListDomainsResponse, error) {
	orgID, err := organizationID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := service.ListDomains(ctx, orgID)
	if err != nil {
		return nil, err
	}
	domains := make([]*pb.Domain, 0, len(rows))
	for _, row := range rows {
		domains = append(domains, domainToProto(row))
	}
	return &pb.ListDomainsResponse{Domains: domains}, nil
}

func CreateDomain(ctx context.Context, service enterprise.SSO, req *pb.CreateDomainRequest) (*pb.DomainResponse, error) {
	orgID, err := organizationID(ctx)
	if err != nil {
		return nil, err
	}
	actor, err := accountID(ctx)
	if err != nil {
		return nil, err
	}
	view, err := service.CreateDomain(ctx, orgID, actor, req.GetDomain())
	if err != nil {
		return nil, err
	}
	return &pb.DomainResponse{Domain: domainToProto(*view)}, nil
}

func VerifyDomain(ctx context.Context, service enterprise.SSO, req *pb.VerifyDomainRequest) (*pb.DomainResponse, error) {
	orgID, err := organizationID(ctx)
	if err != nil {
		return nil, err
	}
	domainID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "The domain is not valid.")
	}
	view, err := service.VerifyDomain(ctx, orgID, domainID)
	if err != nil {
		return nil, err
	}
	return &pb.DomainResponse{Domain: domainToProto(*view)}, nil
}

func DeleteDomain(ctx context.Context, service enterprise.SSO, req *pb.DeleteDomainRequest) (*empty.Empty, error) {
	orgID, err := organizationID(ctx)
	if err != nil {
		return nil, err
	}
	domainID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "The domain is not valid.")
	}
	if err := service.DeleteDomain(ctx, orgID, domainID); err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func ListSCIMTokens(ctx context.Context, service enterprise.SSO) (*pb.ListSCIMTokensResponse, error) {
	orgID, err := organizationID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := service.ListSCIMTokens(ctx, orgID)
	if err != nil {
		return nil, err
	}
	tokens := make([]*pb.SCIMToken, 0, len(rows))
	for _, row := range rows {
		token := &pb.SCIMToken{Id: row.ID.String(), Description: row.Description, CreatedAt: timestamppb.New(row.CreatedAt)}
		if row.LastUsedAt != nil {
			token.LastUsedAt = timestamppb.New(*row.LastUsedAt)
		}
		tokens = append(tokens, token)
	}
	return &pb.ListSCIMTokensResponse{Tokens: tokens}, nil
}

func CreateSCIMToken(ctx context.Context, service enterprise.SSO, req *pb.CreateSCIMTokenRequest) (*pb.CreateSCIMTokenResponse, error) {
	orgID, err := organizationID(ctx)
	if err != nil {
		return nil, err
	}
	actor, err := accountID(ctx)
	if err != nil {
		return nil, err
	}
	created, err := service.CreateSCIMToken(ctx, orgID, actor, req.GetDescription())
	if err != nil {
		return nil, err
	}
	return &pb.CreateSCIMTokenResponse{
		Token: &pb.SCIMToken{
			Id:          created.Token.ID.String(),
			Description: created.Token.Description,
			CreatedAt:   timestamppb.New(created.Token.CreatedAt),
		},
		Secret: created.Secret,
	}, nil
}

func RevokeSCIMToken(ctx context.Context, service enterprise.SSO, req *pb.RevokeSCIMTokenRequest) (*empty.Empty, error) {
	orgID, err := organizationID(ctx)
	if err != nil {
		return nil, err
	}
	tokenID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "The token is not valid.")
	}
	if err := service.RevokeSCIMToken(ctx, orgID, tokenID); err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}
