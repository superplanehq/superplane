package factories

import (
	"context"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

const mcpAPIResourcePath = "/mcp"

func CreateFactoryMCPAPIToken(
	ctx context.Context,
	organizationID string,
	req *pb.CreateFactoryMCPAPITokenRequest,
) (*pb.CreateFactoryMCPAPITokenResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create MCP token")
	}
	if err := requireSuperPlaneMCPServer(orgID); err != nil {
		return nil, factoryErrorToStatus(err, "failed to create MCP token")
	}
	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return nil, grpcerrors.InvalidArgument(nil, "name is required")
	}
	resource, err := normalizeMCPAPIResource(req.GetResource())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create MCP token")
	}
	user, err := mcpAPITokenCaller(ctx, orgID)
	if err != nil {
		return nil, err
	}

	secret, err := crypto.Base64String(32)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to create MCP token")
	}
	plaintext := models.MCPAPITokenPrefix + secret
	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create MCP token")
	}
	token := models.NewMCPAPIToken(
		user.ID,
		orgID,
		factory.ID,
		name,
		resource,
		crypto.HashToken(plaintext),
		models.MCPGrantedScopes,
	)
	if err := models.CreateMCPAPIToken(db, token); err != nil {
		return nil, factoryErrorToStatus(err, "failed to create MCP token")
	}

	return &pb.CreateFactoryMCPAPITokenResponse{
		Token:     serializeMCPAPIToken(token, user),
		Plaintext: plaintext,
	}, nil
}

func RevokeFactoryMCPAPIToken(
	ctx context.Context,
	organizationID string,
	req *pb.RevokeFactoryMCPAPITokenRequest,
) (*pb.RevokeFactoryMCPAPITokenResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to revoke MCP token")
	}
	if err := requireSuperPlaneMCPServer(orgID); err != nil {
		return nil, factoryErrorToStatus(err, "failed to revoke MCP token")
	}
	tokenID, err := parseMCPClientID(req.GetTokenId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to revoke MCP token")
	}
	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to revoke MCP token")
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		locked, err := models.FindMCPAPITokenForFactory(tx, orgID, factory.ID, tokenID)
		if err != nil {
			return err
		}
		return locked.HardDelete(tx)
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to revoke MCP token")
	}
	return &pb.RevokeFactoryMCPAPITokenResponse{}, nil
}

func mcpAPITokenCaller(ctx context.Context, orgID uuid.UUID) (*models.User, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	user, err := models.FindActiveUserByIDInTransaction(database.DB(ctx), orgID.String(), userID)
	if err != nil {
		return nil, grpcerrors.Unauthenticated(err, "user not authenticated")
	}
	if !user.IsHuman() {
		return nil, grpcerrors.PermissionDenied(nil, "API keys cannot create MCP tokens")
	}
	return user, nil
}

func normalizeMCPAPIResource(raw string) (string, error) {
	resource := strings.TrimSpace(raw)
	parsed, err := url.Parse(resource)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.Path != mcpAPIResourcePath {
		return "", invalidArgument("resource is not valid")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", invalidArgument("resource is not valid")
	}
	return resource, nil
}

func serializeMCPAPIToken(token *models.MCPAPIToken, user *models.User) *pb.FactoryMCPClient {
	client := &pb.FactoryMCPClient{
		Id:         token.ID.String(),
		Kind:       mcpClientKindAPIToken,
		ClientName: token.Name,
		UserId:     token.UserID.String(),
		CreatedAt:  timestamppb.New(token.CreatedAt),
	}
	if user != nil {
		client.UserName = user.Name
		client.UserEmail = user.GetEmail()
	}
	if token.LastUsedAt != nil {
		client.LastUsedAt = timestamppb.New(*token.LastUsedAt)
	}
	return client
}
