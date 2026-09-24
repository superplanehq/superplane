package factories

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/mcp"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/secrets"
	"gorm.io/gorm"
)

func ListFactoryAgentResourceTools(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.ListFactoryAgentResourceToolsRequest,
) (*pb.ListFactoryAgentResourceToolsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list MCP tools")
	}
	resourceID, err := uuid.Parse(strings.TrimSpace(req.GetResourceId()))
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid resource id"), "failed to list MCP tools")
	}
	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list MCP tools")
	}
	resource, err := factory.FindAgentResource(db, resourceID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list MCP tools")
	}
	if resource.Kind != models.FactoryAgentResourceKindMCPServer {
		return nil, factoryErrorToStatus(invalidArgument("this resource is not an MCP server"), "failed to list MCP tools")
	}

	headers, err := mcpHeadersForResource(ctx, deps, db, orgID, resource)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list MCP tools")
	}
	tools, err := mcp.ListTools(ctx, mcpHTTPClient(deps), resource.Config.Data().URL, headers)
	if err != nil {
		return nil, factoryErrorToStatus(errors.Join(errListMCPTools, err), "failed to list MCP tools")
	}

	out := make([]*pb.FactoryAgentResourceTool, 0, len(tools))
	for _, tool := range tools {
		out = append(out, &pb.FactoryAgentResourceTool{
			Name:        tool.Name,
			Description: tool.Description,
		})
	}
	return &pb.ListFactoryAgentResourceToolsResponse{Tools: out}, nil
}

func mcpHTTPClient(deps IntakeDependencies) mcp.HTTPDoer {
	if deps.Registry == nil {
		return mcp.DoerFromCore(nil)
	}
	return mcp.DoerFromCore(deps.Registry.HTTPContext())
}

func mcpHeadersForResource(
	ctx context.Context,
	deps IntakeDependencies,
	db *gorm.DB,
	orgID uuid.UUID,
	resource *models.FactoryAgentResource,
) (map[string]string, error) {
	config := resource.Config.Data()
	if config.MCPAuth() == models.FactoryAgentResourceAuthOAuth {
		return oauthHeadersForResource(ctx, deps, db, resource)
	}

	headers := map[string]string{}
	for _, header := range config.Headers {
		value, err := organizationSecretKey(ctx, deps, db, orgID, header.SecretName, header.SecretKey)
		if err != nil {
			return nil, err
		}
		headers[header.Name] = value
	}
	return headers, nil
}

func oauthHeadersForResource(
	ctx context.Context,
	deps IntakeDependencies,
	db *gorm.DB,
	resource *models.FactoryAgentResource,
) (map[string]string, error) {
	if resource.OAuthState() != models.FactoryAgentResourceOAuthConnected {
		return nil, errFactoryAgentResourceNotConnected
	}
	if deps.Encryptor == nil {
		return nil, errListMCPTools
	}
	token, err := mcp.MintFactoryAgentResourceAccessToken(ctx, deps.Encryptor, mcpHTTPClient(deps), db, resource)
	if err != nil || strings.TrimSpace(token) == "" {
		return nil, errFactoryAgentResourceNotConnected
	}
	return map[string]string{"Authorization": "Bearer " + token}, nil
}

func organizationSecretKey(
	ctx context.Context,
	deps IntakeDependencies,
	db *gorm.DB,
	orgID uuid.UUID,
	secretName, secretKey string,
) (string, error) {
	if deps.Encryptor == nil {
		return "", errListMCPTools
	}
	provider, err := secrets.NewProvider(db, deps.Encryptor, secretName, models.DomainTypeOrganization, orgID)
	if err != nil {
		return "", errListMCPTools
	}
	values, err := provider.Load(ctx)
	if err != nil {
		return "", errListMCPTools
	}
	value := strings.TrimSpace(values[secretKey])
	if value == "" {
		return "", errListMCPTools
	}
	return value, nil
}
