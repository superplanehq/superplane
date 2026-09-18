package runner

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/mcp"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	WorkspaceMCPConfigPath          = "workspace_mcp.json"
	EnvSuperplaneWorkspaceMCPConfig = "SUPERPLANE_WORKSPACE_MCP_CONFIG"
	workspaceMCPConfigEnvValue      = "$SUPERPLANE_TASK_DIR/" + WorkspaceMCPConfigPath
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
	logger := workspaceAgentResourcesLogger(ctx)
	orgID, err := uuid.Parse(strings.TrimSpace(ctx.OrganizationID))
	if err != nil {
		return environment, files
	}
	enabled, err := models.HasExperimentalFeature(orgID, features.FeatureWorkspaceAgentResources)
	if err != nil {
		logger.WithError(err).Warn("skip workspace agent resources: feature check failed")
		return environment, files
	}
	if !enabled {
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
		logger.WithError(err).Warn("skip workspace agent resources: factory not found")
		return environment, files
	}

	resources, err := factory.ListEnabledMCPServers(db)
	if err != nil {
		logger.WithError(err).Warn("skip workspace agent resources: list failed")
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
		logger.WithError(err).Warn("skip workspace agent resources: encode failed")
		return environment, files
	}

	files = append(files, BrokerTaskFile{
		Path:    WorkspaceMCPConfigPath,
		Content: string(payload) + "\n",
		Mode:    "0644",
	})
	environment = append(environment, BrokerEnvironmentVariable{
		Name:  EnvSuperplaneWorkspaceMCPConfig,
		Value: workspaceMCPConfigEnvValue,
	})
	return environment, files
}

func workspaceAgentResourcesLogger(ctx core.ExecutionContext) *log.Entry {
	if ctx.Logger != nil {
		return ctx.Logger
	}
	return log.WithField("component", "workspace_agent_resources")
}

func assembleWorkspaceMCPServer(
	ctx core.ExecutionContext,
	encryptor crypto.Encryptor,
	httpClient mcp.HTTPDoer,
	db *gorm.DB,
	resource *models.FactoryAgentResource,
) (workspaceMCPServer, bool) {
	logger := workspaceAgentResourcesLogger(ctx)
	config := resource.Config.Data()
	if strings.TrimSpace(config.URL) == "" || resource.Name == models.ReservedFactoryAgentResourceName {
		return workspaceMCPServer{}, false
	}

	headers := map[string]string{}
	switch config.MCPAuth() {
	case models.FactoryAgentResourceAuthHeaders:
		for _, header := range config.Headers {
			if ctx.Secrets == nil {
				logger.WithField("mcp", resource.Name).Warn("skip workspace MCP header: secrets context missing")
				break
			}
			value, err := ctx.Secrets.GetKey(header.SecretName, header.SecretKey)
			if err != nil {
				logger.WithError(err).WithField("mcp", resource.Name).WithField("header", header.Name).
					Warn("skip workspace MCP header: secret missing")
				continue
			}
			headers[header.Name] = string(value)
		}
	case models.FactoryAgentResourceAuthOAuth:
		if encryptor == nil {
			logger.WithField("mcp", resource.Name).Warn("skip workspace MCP: encryptor missing")
			return workspaceMCPServer{}, false
		}
		token, err := mcp.MintFactoryAgentResourceAccessToken(context.Background(), encryptor, httpClient, db, resource)
		if err != nil || strings.TrimSpace(token) == "" {
			logger.WithError(err).WithField("mcp", resource.Name).Warn("skip workspace MCP: sign-in required")
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
