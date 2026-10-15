package runner

import (
	"context"
	"encoding/json"
	"fmt"
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
	workspaceClaudeSkillPath        = ".claude/skills/%s/SKILL.md"
	workspaceAgentsSkillPath        = ".agents/skills/%s/SKILL.md"
	WorkspaceSetupConfigPath        = "workspace_setup.json"
	WorkspaceContextConfigPath      = "workspace_context.json"
	// WorkspaceCheckoutScriptPath is the smart checkout helper shipped on
	// factory tasks. New factory templates source it as a single line so
	// the main repo lands in repo/ and every context repo in its
	// directory. Stored canvases are never rewritten, so existing
	// automations keep their inline Clone Repo step until their owner
	// adopts the helper.
	WorkspaceCheckoutScriptPath = "setup/checkout-workspace.sh"
	// WorkspaceCheckoutCommitSetupEnv opts a checkout into the factory
	// commit setup (agent git identity, trailer hook, branch handling).
	// The Implement Clone step sets it inline; read-only checkouts such
	// as intake analysis leave it off so sourcing the helper never
	// mutates the remote.
	WorkspaceCheckoutCommitSetupEnv = "CHECKOUT_COMMIT_SETUP"
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
// task. Every factory-owned task also ships the checkout helper, so
// factory Clone steps stay a single line with no inline fallback. It is a
// no-op when the canvas is not factory-owned.
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

	// The checkout helper needs no configured resources: it clones the
	// main repo on its own and context repos when the agent screen adds
	// them. Ship it on every factory task so Clone steps never branch.
	files = appendWorkspaceCheckoutFile(files)

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

// WorkspaceCheckoutScript is the smart checkout helper factory Clone steps
// source as a single line. It checks out the main repo to repo/ with the
// same branch handling as the legacy Clone Repo step, then clones every
// workspace_context.json entry to its directory. Authentication comes from
// the VCS integration setup that already ran (gh auth setup-git /
// Bitbucket credential helper), so the script uses plain HTTPS URLs and
// never embeds tokens. It is idempotent: directories that already contain
// a checkout are reused, never wiped. The factory commit setup (agent git
// identity, trailer hook, branch handling) runs only when
// CHECKOUT_COMMIT_SETUP=1, so read-only checkouts stay side-effect free.
func WorkspaceCheckoutScript() string {
	return `set -euo pipefail
: "${SUPERPLANE_TASK_DIR:?SUPERPLANE_TASK_DIR is required}"
: "${REPO_URL:?REPO_URL is required}"
if ! command -v git >/dev/null 2>&1; then
  echo "git not found on PATH; required for checkout" >&2
  exit 1
fi
branch="${BASE:-main}"

if [ -d repo/.git ]; then
  echo "repo/ already checked out, reusing."
else
  rm -rf repo
  heads="$(git ls-remote --heads "${REPO_URL}")" || exit 1
  if printf '%s\n' "$heads" | awk '{print $2}' | grep -Fxq "refs/heads/${branch}"; then
    git clone --depth 1 --branch "${branch}" "${REPO_URL}" repo
  elif [ -z "$heads" ]; then
    git clone "${REPO_URL}" repo
  else
    echo "Remote branch ${branch} not found. The repository has other branches." >&2
    exit 1
  fi
fi
` + workspaceCheckoutCommitSetup() + `
ctx="$SUPERPLANE_TASK_DIR/workspace_context.json"
if [ -f "$ctx" ]; then
  if ! command -v node >/dev/null 2>&1; then
    echo "node not found on PATH; skipping context repositories" >&2
  else
    _sp_tab="$(printf '\t')"
    _sp_ctx_list="$(node -e 'const fs=require("fs");try{const d=JSON.parse(fs.readFileSync(process.argv[1],"utf8"));const r=Array.isArray(d.repositories)?d.repositories:[];for(const e of r){const u=((e&&e.url)||"").trim();const dir=((e&&e.directory)||"").trim();if(u&&dir){console.log(u+"\t"+dir);}}}catch(e){console.error("workspace_context.json parse failed: "+e.message);process.exit(1);}' "$ctx")" || exit 1
    if [ -n "$_sp_ctx_list" ]; then
      printf '%s\n' "$_sp_ctx_list" | while IFS="$_sp_tab" read -r url dir; do
        case "$dir" in
          ""|"."|"/"|*..*)
            echo "Skipping context repository with unsafe directory: $dir" >&2
            continue
            ;;
        esac
        case "$url" in
          "")
            echo "Skipping context repository with missing URL: $dir" >&2
            continue
            ;;
        esac
        if [ -d "$dir/.git" ]; then
          echo "$dir already checked out, reusing."
          continue
        fi
        mkdir -p "$(dirname "$dir")"
        rm -rf "$dir"
        git clone --depth 1 "$url" "$dir" || exit 1
      done
    fi
    unset _sp_tab _sp_ctx_list
  fi
fi
echo "Checkout complete: repo/ and context repositories (if any)."
`
}

// workspaceCheckoutCommitSetup runs the factory commit identity and branch
// handling inside repo/. The caller opts in with CHECKOUT_COMMIT_SETUP=1
// (the Implement Clone step sets it inline on the source line). The script
// is sourced, so it saves and restores the working directory instead of a
// subshell: the exports below must persist in the caller, and the context
// clones after this block use paths relative to the task directory.
func workspaceCheckoutCommitSetup() string {
	return `if [ "${` + WorkspaceCheckoutCommitSetupEnv + `:-0}" = "1" ]; then
git config --global user.email "` + FactoryAgentEmail + `"
git config --global user.name "` + FactoryAgentName + `"
_sp_cwd="$(pwd)"
cd repo || exit 1
` + FactoryRepoCommitSetup() + `
use_existing_branch() {
  echo "Another run created ${branch}. Using that branch." >&2
  git fetch origin "${branch}"
  git checkout -B "${branch}" "origin/${branch}"
}

if ! git rev-parse --verify --quiet "refs/remotes/origin/${branch}" >/dev/null && ! git rev-parse --verify --quiet HEAD >/dev/null; then
  remote_head="$(git ls-remote --heads origin "refs/heads/${branch}")" || exit 1
  if [ -n "$remote_head" ]; then
    use_existing_branch
  else
    echo "Remote has no branches. Creating ${branch}." >&2
    git checkout -B "${branch}"
    git commit --allow-empty -s -m "chore: Initialize repository"
    if ! git push -u origin "HEAD:refs/heads/${branch}"; then
      remote_head="$(git ls-remote --heads origin "refs/heads/${branch}")" || exit 1
      if [ -z "$remote_head" ]; then
        echo "Failed to create ${branch}." >&2
        exit 1
      fi
      use_existing_branch
    fi
  fi
fi
cd "$_sp_cwd" || exit 1
unset _sp_cwd
fi
`
}

func appendWorkspaceCheckoutFile(files []BrokerTaskFile) []BrokerTaskFile {
	for i := range files {
		if files[i].Path == WorkspaceCheckoutScriptPath {
			return files
		}
	}
	return append(files, BrokerTaskFile{
		Path:    WorkspaceCheckoutScriptPath,
		Content: WorkspaceCheckoutScript(),
		Mode:    "0644",
	})
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
		builder.WriteString("\nCheckout: source \"$SUPERPLANE_TASK_DIR/" + WorkspaceCheckoutScriptPath + "\" to check out repo/ and context repositories. ")
		builder.WriteString("Extra repositories (already checked out by the script when the automation sources it): ")
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
