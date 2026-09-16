package runner

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/mcp"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	WorkspaceMCPConfigPath          = "workspace_mcp.json"
	EnvSuperplaneWorkspaceMCPConfig = "SUPERPLANE_WORKSPACE_MCP_CONFIG"
	workspaceMCPConfigTaskPath      = "/task/" + WorkspaceMCPConfigPath
)

type workspaceMCPServer struct {
	Name    string            `json:"name"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

type workspaceMCPFile struct {
	Servers []workspaceMCPServer `json:"servers"`
}

// AttachWorkspaceAgentResources ships enabled workspace MCP connections on
// the broker task. It is a no-op when the canvas is not factory-owned.
func AttachWorkspaceAgentResources(
	ctx core.ExecutionContext,
	environment []BrokerEnvironmentVariable,
	files []BrokerTaskFile,
) ([]BrokerEnvironmentVariable, []BrokerTaskFile) {
	orgID, err := uuid.Parse(strings.TrimSpace(ctx.OrganizationID))
	if err != nil {
		return environment, files
	}
	canvasID, err := uuid.Parse(strings.TrimSpace(ctx.WorkflowID))
	if err != nil {
		return environment, files
	}

	db := database.DB(context.Background())
	factoryID, err := models.FindFactoryIDForCanvas(db, orgID, canvasID)
	if err != nil || factoryID == nil {
		return environment, files
	}

	factory, err := models.FindFactory(db, orgID, *factoryID)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.WithError(err).Warn("skip workspace agent resources: factory not found")
		}
		return environment, files
	}

	resources, err := factory.ListEnabledMCPServers(db)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.WithError(err).Warn("skip workspace agent resources: list failed")
		}
		return environment, files
	}
	if len(resources) == 0 {
		return environment, files
	}

	encryptor, _ := crypto.FromEnv()
	httpClient := mcp.DoerFromCore(ctx.HTTP)
	servers := make([]workspaceMCPServer, 0, len(resources))
	for i := range resources {
		server, ok := assembleWorkspaceMCPServer(ctx, encryptor, httpClient, db, &resources[i])
		if !ok {
			continue
		}
		servers = append(servers, server)
	}
	if len(servers) == 0 {
		return environment, files
	}

	payload, err := json.Marshal(workspaceMCPFile{Servers: servers})
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.WithError(err).Warn("skip workspace agent resources: encode failed")
		}
		return environment, files
	}

	files = append(files, BrokerTaskFile{
		Path:    WorkspaceMCPConfigPath,
		Content: string(payload) + "\n",
		Mode:    "0644",
	})
	environment = append(environment, BrokerEnvironmentVariable{
		Name:  EnvSuperplaneWorkspaceMCPConfig,
		Value: workspaceMCPConfigTaskPath,
	})
	return environment, files
}

func assembleWorkspaceMCPServer(
	ctx core.ExecutionContext,
	encryptor crypto.Encryptor,
	httpClient mcp.HTTPDoer,
	db *gorm.DB,
	resource *models.FactoryAgentResource,
) (workspaceMCPServer, bool) {
	config := resource.Config.Data()
	if strings.TrimSpace(config.URL) == "" || resource.Name == models.ReservedFactoryAgentResourceName {
		return workspaceMCPServer{}, false
	}

	headers := map[string]string{}
	switch config.MCPAuth() {
	case models.FactoryAgentResourceAuthHeaders:
		if ctx.Secrets == nil {
			return workspaceMCPServer{}, false
		}
		for _, header := range config.Headers {
			value, err := ctx.Secrets.GetKey(header.SecretName, header.SecretKey)
			if err != nil {
				if ctx.Logger != nil {
					ctx.Logger.WithError(err).WithField("mcp", resource.Name).Warn("skip workspace MCP: secret missing")
				}
				return workspaceMCPServer{}, false
			}
			headers[header.Name] = string(value)
		}
	case models.FactoryAgentResourceAuthOAuth:
		if encryptor == nil {
			if ctx.Logger != nil {
				ctx.Logger.WithField("mcp", resource.Name).Warn("skip workspace MCP: encryptor missing")
			}
			return workspaceMCPServer{}, false
		}
		token, err := factories.MintFactoryAgentResourceAccessToken(context.Background(), encryptor, httpClient, db, resource)
		if err != nil || strings.TrimSpace(token) == "" {
			if ctx.Logger != nil {
				ctx.Logger.WithError(err).WithField("mcp", resource.Name).Warn("skip workspace MCP: sign-in required")
			}
			return workspaceMCPServer{}, false
		}
		headers["Authorization"] = "Bearer " + token
	default:
		return workspaceMCPServer{}, false
	}

	return workspaceMCPServer{
		Name:    resource.Name,
		URL:     config.URL,
		Headers: headers,
	}, true
}
