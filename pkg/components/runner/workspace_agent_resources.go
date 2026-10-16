package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	workspaceClaudeSkillPath        = ".claude/skills/%s/SKILL.md"
	workspaceAgentsSkillPath        = ".agents/skills/%s/SKILL.md"
	WorkspaceSetupConfigPath        = "workspace_setup.json"
	WorkspaceContextConfigPath      = "workspace_context.json"
)

type workspaceMCPServer struct {
	Name          string            `json:"name"`
	URL           string            `json:"url"`
	Headers       map[string]string `json:"headers,omitempty"`
	DisabledTools []string          `json:"disabledTools,omitempty"`
}

type workspaceMCPFile struct {
	Servers []workspaceMCPServer `json:"servers"`
}

type workspaceSetupFile struct {
	Commands []string `json:"commands"`
}

type workspaceContextRepository struct {
	Name       string `json:"name"`
	Repository string `json:"repository"`
	URL        string `json:"url"`
	Directory  string `json:"directory"`
}

type workspaceContextFile struct {
	Repositories []workspaceContextRepository `json:"repositories"`
}

// AttachWorkspaceAgentResources ships enabled workspace MCP connections,
// inline skills, setup commands, and context repositories on the broker
// task. It is a no-op when the canvas is not factory-owned.
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
	mcpEnabled, err := models.HasExperimentalFeature(orgID, features.FeatureWorkspaceMCP)
	if err != nil {
		logger.WithError(err).Warn("skip workspace agent resources: MCP feature check failed")
		return environment, files
	}
	skillsEnabled, err := models.HasExperimentalFeature(orgID, features.FeatureWorkspaceSkills)
	if err != nil {
		logger.WithError(err).Warn("skip workspace agent resources: skills feature check failed")
		return environment, files
	}
	agentEnabled, err := models.HasExperimentalFeature(orgID, features.FeatureWorkspaceAgent)
	if err != nil {
		logger.WithError(err).Warn("skip workspace agent resources: agent feature check failed")
		return environment, files
	}
	if !mcpEnabled && !skillsEnabled && !agentEnabled {
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

	var mcpServers []models.FactoryAgentResource
	if mcpEnabled {
		mcpServers, err = factory.ListEnabledMCPServers(db)
		if err != nil {
			logger.WithError(err).Warn("skip workspace agent resources: list MCP failed")
			return environment, files
		}
	}
	var skills []models.FactoryAgentResource
	if skillsEnabled {
		skills, err = factory.ListEnabledSkills(db)
		if err != nil {
			logger.WithError(err).Warn("skip workspace agent resources: list skills failed")
			return environment, files
		}
	}
	var setupCommands []string
	var contextRepos []models.FactoryAgentResource
	if agentEnabled {
		setupCommands, contextRepos, err = loadWorkspaceSetupAndContext(db, factory)
		if err != nil {
			logger.WithError(err).Warn("skip workspace setup and context: list failed")
			return environment, files
		}
	}

	disabled := disabledAgentResourceIDs(ctx.Configuration)
	disabledTools := disabledAgentResourceTools(ctx.Configuration)
	enabledTools := enabledAgentResourceTools(ctx.Configuration)
	mcpServers = rejectDisabledAgentResources(mcpServers, disabled)
	skills = rejectDisabledAgentResources(skills, disabled)
	contextRepos = rejectDisabledAgentResources(contextRepos, disabled)
	setupCommands = rejectDisabledSetupCommands(db, factory, setupCommands, disabled)

	var mcpNames []string
	environment, files, mcpNames = attachWorkspaceMCPServers(ctx, db, mcpServers, disabledTools, enabledTools, environment, files)
	files = appendWorkspaceSkillFiles(skills, files)
	files = appendWorkspaceSetupFiles(setupCommands, files)
	files = appendWorkspaceContextFiles(factory, contextRepos, files)
	files = appendWorkspaceAgentResourcesHint(files, mcpNames, setupCommands, contextRepos)
	return environment, files
}

func loadWorkspaceSetupAndContext(db *gorm.DB, factory *models.Factory) ([]string, []models.FactoryAgentResource, error) {
	commands, err := factory.ListEnabledSetupCommands(db)
	if err != nil {
		return nil, nil, err
	}
	repos, err := factory.ListEnabledContextRepos(db)
	if err != nil {
		return nil, nil, err
	}
	return commands, repos, nil
}

func rejectDisabledSetupCommands(db *gorm.DB, factory *models.Factory, commands []string, disabled map[string]struct{}) []string {
	if len(disabled) == 0 || len(commands) == 0 {
		return commands
	}
	setup, err := factory.FindSetupCommands(db)
	if err != nil {
		return commands
	}
	if _, skip := disabled[setup.ID.String()]; skip {
		return nil
	}
	return commands
}

func attachWorkspaceMCPServers(
	ctx core.ExecutionContext,
	db *gorm.DB,
	resources []models.FactoryAgentResource,
	disabledTools map[string][]string,
	enabledTools map[string][]string,
	environment []BrokerEnvironmentVariable,
	files []BrokerTaskFile,
) ([]BrokerEnvironmentVariable, []BrokerTaskFile, []string) {
	if len(resources) == 0 {
		return environment, files, nil
	}
	logger := workspaceAgentResourcesLogger(ctx)
	encryptor, _ := crypto.FromEnv()
	httpClient := mcp.DoerFromCore(ctx.HTTP)
	tokenLifetime := workspaceMCPAccessTokenLifetime(ctx.Configuration)
	servers := make([]workspaceMCPServer, 0, len(resources))
	for i := range resources {
		server, ok := assembleWorkspaceMCPServer(
			ctx,
			encryptor,
			httpClient,
			db,
			&resources[i],
			disabledTools[resources[i].ID.String()],
			enabledTools[resources[i].ID.String()],
			tokenLifetime,
		)
		if !ok {
			continue
		}
		servers = append(servers, server)
	}
	if len(servers) == 0 {
		return environment, files, nil
	}

	payload, err := json.Marshal(workspaceMCPFile{Servers: servers})
	if err != nil {
		logger.WithError(err).Warn("skip workspace agent resources: encode failed")
		return environment, files, nil
	}

	names := make([]string, 0, len(servers))
	for _, server := range servers {
		names = append(names, server.Name)
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
	return environment, files, names
}

func appendWorkspaceSkillFiles(resources []models.FactoryAgentResource, files []BrokerTaskFile) []BrokerTaskFile {
	for i := range resources {
		markdown := strings.TrimSpace(resources[i].Config.Data().Markdown)
		if markdown == "" || resources[i].Name == models.ReservedFactoryAgentResourceName {
			continue
		}
		if err := resources[i].Config.Data().ValidateSkill(); err != nil {
			continue
		}
		content := markdown + "\n"
		files = append(files,
			BrokerTaskFile{Path: fmt.Sprintf(workspaceClaudeSkillPath, resources[i].Name), Content: content, Mode: "0644"},
			BrokerTaskFile{Path: fmt.Sprintf(workspaceAgentsSkillPath, resources[i].Name), Content: content, Mode: "0644"},
		)
	}
	return files
}

func appendWorkspaceSetupFiles(commands []string, files []BrokerTaskFile) []BrokerTaskFile {
	commands = models.NormalizeSetupCommands(commands)
	if len(commands) == 0 {
		return files
	}
	payload, err := json.Marshal(workspaceSetupFile{Commands: commands})
	if err != nil {
		return files
	}
	return append(files, BrokerTaskFile{
		Path:    WorkspaceSetupConfigPath,
		Content: string(payload) + "\n",
		Mode:    "0644",
	})
}

func WorkspaceContextRepoURL(provider, repository string) string {
	repository = strings.TrimSpace(repository)
	if strings.HasPrefix(repository, "https://") {
		return repository
	}
	if strings.EqualFold(strings.TrimSpace(provider), models.ProviderBitbucket) {
		return "https://bitbucket.org/" + repository + ".git"
	}
	return "https://github.com/" + repository + ".git"
}

func WorkspaceContextRepoDirectory(repository string) string {
	repository = strings.TrimSpace(strings.Trim(repository, "/"))
	if repository == "" {
		return "context/repo"
	}
	return "context/" + repository
}

func appendWorkspaceContextFiles(factory *models.Factory, resources []models.FactoryAgentResource, files []BrokerTaskFile) []BrokerTaskFile {
	if len(resources) == 0 {
		return files
	}
	factoryProvider := ""
	if factory != nil {
		factoryProvider = factory.OnboardingConfigValue().EffectiveVCSProvider()
	}
	repos := make([]workspaceContextRepository, 0, len(resources))
	for i := range resources {
		config := resources[i].Config.Data()
		repository := strings.TrimSpace(config.Repository)
		if repository == "" {
			continue
		}
		if err := config.ValidateContextRepo(); err != nil {
			continue
		}
		provider := strings.TrimSpace(config.VCSProvider)
		if provider == "" {
			provider = factoryProvider
		}
		repos = append(repos, workspaceContextRepository{
			Name:       resources[i].Name,
			Repository: repository,
			URL:        WorkspaceContextRepoURL(provider, repository),
			Directory:  WorkspaceContextRepoDirectory(repository),
		})
	}
	if len(repos) == 0 {
		return files
	}
	payload, err := json.Marshal(workspaceContextFile{Repositories: repos})
	if err != nil {
		return files
	}
	return append(files, BrokerTaskFile{
		Path:    WorkspaceContextConfigPath,
		Content: string(payload) + "\n",
		Mode:    "0644",
	})
}

func workspaceMCPAccessTokenLifetime(configuration any) time.Duration {
	seconds := DefaultExecutionTimeoutSeconds
	config, ok := configuration.(map[string]any)
	if ok {
		if value, ok := asInt(config["executionTimeoutSeconds"]); ok && value > 0 {
			seconds = int(value)
		}
	}
	if seconds > maxExecutionTimeoutSecondsRequest {
		seconds = maxExecutionTimeoutSecondsRequest
	}
	return time.Duration(seconds) * time.Second
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
	automationDisabledTools []string,
	enabledOverride []string,
	tokenLifetime time.Duration,
) (workspaceMCPServer, bool) {
	logger := workspaceAgentResourcesLogger(ctx)
	config := resource.Config.Data()
	if strings.TrimSpace(config.URL) == "" || resource.Name == models.ReservedFactoryAgentResourceName {
		return workspaceMCPServer{}, false
	}
	if !config.ToolsDefaultApplied {
		logger.WithField("mcp", resource.Name).Warn("skip workspace MCP: tool defaults are not saved")
		return workspaceMCPServer{}, false
	}

	headers := map[string]string{}
	switch config.MCPAuth() {
	case models.FactoryAgentResourceAuthHeaders:
		for _, header := range config.Headers {
			if ctx.Secrets == nil {
				logger.WithField("mcp", resource.Name).Warn("skip workspace MCP: secrets context missing")
				return workspaceMCPServer{}, false
			}
			value, err := ctx.Secrets.GetKey(header.SecretName, header.SecretKey)
			if err != nil {
				logger.WithError(err).WithField("mcp", resource.Name).WithField("header", header.Name).
					Warn("skip workspace MCP: secret missing")
				return workspaceMCPServer{}, false
			}
			headers[header.Name] = string(value)
		}
	case models.FactoryAgentResourceAuthOAuth:
		if encryptor == nil {
			logger.WithField("mcp", resource.Name).Warn("skip workspace MCP: encryptor missing")
			return workspaceMCPServer{}, false
		}
		token, err := mcp.MintFactoryAgentResourceAccessTokenWithLifetime(
			context.Background(),
			encryptor,
			httpClient,
			db,
			resource,
			tokenLifetime,
		)
		if err != nil || strings.TrimSpace(token) == "" {
			logger.WithError(err).WithField("mcp", resource.Name).Warn("skip workspace MCP: sign-in required")
			return workspaceMCPServer{}, false
		}
		headers["Authorization"] = "Bearer " + token
	default:
		return workspaceMCPServer{}, false
	}

	return workspaceMCPServer{
		Name:          resource.Name,
		URL:           config.URL,
		Headers:       headers,
		DisabledTools: effectiveMCPDisabledTools(config.DisabledTools, enabledOverride, automationDisabledTools),
	}, true
}

const workspaceAgentResourcesHintPrefix = "You can use these resources."

func disabledAgentResourceIDs(configuration any) map[string]struct{} {
	ids := map[string]struct{}{}
	config, ok := configuration.(map[string]any)
	if !ok {
		return ids
	}
	switch values := config["disabledAgentResourceIds"].(type) {
	case []string:
		for _, id := range values {
			id = strings.TrimSpace(id)
			if id != "" {
				ids[id] = struct{}{}
			}
		}
	case []any:
		for _, value := range values {
			id, ok := value.(string)
			if !ok {
				continue
			}
			id = strings.TrimSpace(id)
			if id != "" {
				ids[id] = struct{}{}
			}
		}
	}
	return ids
}

func disabledAgentResourceTools(configuration any) map[string][]string {
	return agentResourceToolNames(configuration, "disabledAgentResourceTools")
}

func enabledAgentResourceTools(configuration any) map[string][]string {
	return agentResourceToolNames(configuration, "enabledAgentResourceTools")
}

func agentResourceToolNames(configuration any, field string) map[string][]string {
	out := map[string][]string{}
	config, ok := configuration.(map[string]any)
	if !ok {
		return out
	}
	raw, ok := config[field]
	if !ok {
		return out
	}
	switch values := raw.(type) {
	case map[string][]string:
		for id, names := range values {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			out[id] = models.NormalizeDisabledTools(names)
		}
	case map[string]any:
		for id, value := range values {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			out[id] = models.NormalizeDisabledTools(stringSlice(value))
		}
	}
	return out
}

func effectiveMCPDisabledTools(workspaceDisabled, enabledOverride, automationDisabled []string) []string {
	enabled := map[string]struct{}{}
	for _, name := range models.NormalizeDisabledTools(enabledOverride) {
		enabled[name] = struct{}{}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(workspaceDisabled)+len(automationDisabled))
	for _, name := range models.NormalizeDisabledTools(workspaceDisabled) {
		if _, skip := enabled[name]; skip {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	for _, name := range models.NormalizeDisabledTools(automationDisabled) {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func stringSlice(value any) []string {
	switch names := value.(type) {
	case []string:
		return names
	case []any:
		out := make([]string, 0, len(names))
		for _, entry := range names {
			name, ok := entry.(string)
			if !ok {
				continue
			}
			out = append(out, name)
		}
		return out
	default:
		return nil
	}
}

func rejectDisabledAgentResources(resources []models.FactoryAgentResource, disabled map[string]struct{}) []models.FactoryAgentResource {
	if len(disabled) == 0 {
		return resources
	}
	out := make([]models.FactoryAgentResource, 0, len(resources))
	for i := range resources {
		if _, skip := disabled[resources[i].ID.String()]; skip && !resources[i].IsSystemSkill() {
			continue
		}
		out = append(out, resources[i])
	}
	return out
}

func workspaceAgentResourcesHint(mcpNames []string, setupCommands []string, contextRepos []models.FactoryAgentResource) string {
	if len(mcpNames) == 0 && len(setupCommands) == 0 && len(contextRepos) == 0 {
		return ""
	}
	var builder strings.Builder
	builder.WriteString(workspaceAgentResourcesHintPrefix)
	if len(mcpNames) > 0 {
		builder.WriteString("\nMCP servers: ")
		builder.WriteString(strings.Join(mcpNames, ", "))
	}
	if len(setupCommands) > 0 {
		builder.WriteString("\nSetup commands (run in repo/ after clone, in order): ")
		builder.WriteString(strings.Join(setupCommands, " && "))
	}
	if len(contextRepos) > 0 {
		names := make([]string, 0, len(contextRepos))
		for i := range contextRepos {
			repository := strings.TrimSpace(contextRepos[i].Config.Data().Repository)
			if repository == "" {
				repository = contextRepos[i].Name
			}
			names = append(names, repository+" ("+WorkspaceContextRepoDirectory(repository)+")")
		}
		builder.WriteString("\nExtra repositories (clone with the workspace token when needed): ")
		builder.WriteString(strings.Join(names, ", "))
	}
	return builder.String()
}

func appendWorkspaceAgentResourcesHint(files []BrokerTaskFile, mcpNames []string, setupCommands []string, contextRepos []models.FactoryAgentResource) []BrokerTaskFile {
	hint := workspaceAgentResourcesHint(mcpNames, setupCommands, contextRepos)
	if hint == "" {
		return files
	}
	for i := range files {
		if !strings.HasPrefix(files[i].Path, "prompts/") || !strings.HasSuffix(files[i].Path, ".txt") {
			continue
		}
		if strings.Contains(files[i].Content, workspaceAgentResourcesHintPrefix) {
			continue
		}
		content := strings.TrimRight(files[i].Content, "\n")
		files[i].Content = content + "\n\n" + hint + "\n"
	}
	return files
}
