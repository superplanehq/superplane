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
	// WorkspaceCheckoutScriptPath is the checkout command shipped on
	// factory tasks. It lives in bin/ so the injected step finds it on
	// PATH. The main repo lands in repo/ and every context repo in its
	// directory. The runner inserts the command. Stored canvases that
	// already clone are left unchanged.
	WorkspaceCheckoutScriptPath = "bin/checkout"
	// WorkspaceCheckoutCommitSetupEnv opts a checkout into the factory
	// commit setup (agent git identity, trailer hook, branch handling).
	// Implement sets it on the node environment. Pull request checkouts
	// leave it off so the helper never pushes an initial commit.
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

	// The checkout helper needs no configured resources. Ship it on every
	// factory task. The runner inserts the command before the canvas steps
	// when those steps do not already clone.
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

// WorkspaceCheckoutScript is the checkout command the runner inserts before
// factory agent steps. It checks out the main repo to repo/, positions a
// pull request when the task environment describes one, then clones every
// workspace_context.json entry. Setup commands are separate steps.
// Authentication comes from the VCS integration setup that already ran
// (gh auth setup-git / Bitbucket credential helper), so the script uses
// plain HTTPS URLs and never embeds tokens. It is idempotent: directories
// that already contain a checkout are reused, never wiped. The factory
// commit setup (agent git identity, trailer hook, branch handling) runs
// only for a default-branch checkout when CHECKOUT_COMMIT_SETUP=1.
func WorkspaceCheckoutScript() string {
	return `#!/bin/bash
set -euo pipefail
: "${SUPERPLANE_TASK_DIR:?SUPERPLANE_TASK_DIR is required}"
# Later steps resolve repo/ from the launch directory in task_cwd. Clone
# there. Without that file, use the task directory so a helper started in
# another directory does not nest the clone.
if [ -f "$SUPERPLANE_TASK_DIR/task_cwd" ]; then
  _sp_root=$(cat "$SUPERPLANE_TASK_DIR/task_cwd")
else
  _sp_root="$SUPERPLANE_TASK_DIR"
fi
cd "$_sp_root" || exit 1

if [ -z "${REPO_URL:-}" ] && [ -n "${REPO:-}" ]; then
  # An extra Bitbucket credential on a GitHub run must not change the host.
  # Bitbucket is selected from the source repository, or from its token when
  # the GitHub token is absent.
  if [ -n "${SOURCE_REPO:-}" ] || { [ -n "${BITBUCKET_TOKEN:-}" ] && [ -z "${GITHUB_TOKEN:-}" ]; }; then
    REPO_URL="https://bitbucket.org/${REPO}.git"
  else
    REPO_URL="https://github.com/${REPO}.git"
  fi
fi
if [ -z "${REPO_URL:-}" ] && [ -z "${SOURCE_REPO:-}" ]; then
  echo "No repository to check out."
  exit 0
fi
if ! command -v git >/dev/null 2>&1; then
  echo "git not found on PATH; required for checkout" >&2
  exit 1
fi
branch="${BASE:-main}"

clone_default_branch() {
  if [ -d repo/.git ]; then
    echo "repo/ already checked out, reusing."
    return 0
  fi
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
}

revision_matches() {
  expected="$(git rev-parse --verify "${PR_REVISION}^{commit}")"
  [ "$(git rev-parse HEAD)" = "${expected}" ]
}

checkout_github_revision() {
  clone_default_branch
  cd repo || exit 1
  if [ -n "${PR_NUMBER:-}" ]; then
    git fetch --depth 1 origin "pull/${PR_NUMBER}/head" || true
  fi
  if ! git cat-file -e "${PR_REVISION}^{commit}" >/dev/null 2>&1; then
    git fetch origin "${PR_REVISION}"
  fi
  git checkout --detach "${PR_REVISION}"
  if ! revision_matches; then
    echo "Checked out $(git rev-parse HEAD). Expected ${PR_REVISION}." >&2
    exit 1
  fi
  if [ -n "${REPO:-}" ] && [ -n "${BASE:-}" ]; then
    revision="$(git rev-parse HEAD)"
    base_ref="$(printf '%s' "${BASE}" | sed 's#/#%2F#g')"
    gh api -H "Accept: application/vnd.github.v3.diff" "/repos/${REPO}/compare/${base_ref}...${revision}" > /tmp/pr.diff
  fi
  cd "$_sp_root" || exit 1
}

checkout_bitbucket_revision() {
  if [ ! -d repo/.git ]; then
    rm -rf repo
    git clone --no-checkout "${REPO_URL}" repo
  fi
  cd repo || exit 1
  if [ -n "${BASE:-}" ]; then
    git fetch origin "${BASE}:${BASE}" || true
  fi
  if [ -n "${PR_SOURCE:-}" ]; then
    if [ "${SOURCE_REPO}" != "${REPO:-}" ]; then
      git fetch "https://bitbucket.org/${SOURCE_REPO}.git" "${PR_SOURCE}"
    else
      git fetch origin "${PR_SOURCE}"
    fi
  fi
  git cat-file -e "${PR_REVISION}^{commit}"
  git checkout --detach "${PR_REVISION}"
  if ! revision_matches; then
    echo "Checked out the wrong revision." >&2
    exit 1
  fi
  if [ -n "${BASE:-}" ]; then
    git diff "$(git merge-base "${BASE}" HEAD)" HEAD > /tmp/pr.diff
  fi
  cd "$_sp_root" || exit 1
}

checkout_github_branch() {
  if [ ! -d repo/.git ]; then
    rm -rf repo
    git clone --depth 1 "${REPO_URL}" repo
  fi
  cd repo || exit 1
  if [ -z "${PR_HEAD:-}" ] || [ "${PR_HEAD}" = "null" ]; then
    if [ -z "${GITHUB_TOKEN:-}" ] || [ -z "${REPO:-}" ] || [ -z "${PR_NUMBER:-}" ]; then
      echo "Could not resolve the pull request head branch." >&2
      exit 1
    fi
    PR_HEAD=$(curl -fsSL -H "Authorization: Bearer ${GITHUB_TOKEN}" -H "Accept: application/vnd.github+json" "https://api.github.com/repos/${REPO}/pulls/${PR_NUMBER}" | jq -r .head.ref)
  fi
  if [ -z "${PR_HEAD}" ] || [ "${PR_HEAD}" = "null" ]; then
    echo "Could not resolve the pull request head branch." >&2
    exit 1
  fi
  if [ -z "${PR_NUMBER:-}" ]; then
    echo "The pull request number is missing." >&2
    exit 1
  fi
  git fetch origin "pull/${PR_NUMBER}/head:${PR_HEAD}"
  git checkout "${PR_HEAD}"
  cd "$_sp_root" || exit 1
}

checkout_bitbucket_pull_request() {
  if [ -n "${BITBUCKET_EMAIL:-}" ]; then
    AUTH_HEADER="Authorization: Basic $(printf '%s:%s' "${BITBUCKET_EMAIL}" "${BITBUCKET_TOKEN}" | base64 | tr -d '\n')"
  else
    AUTH_HEADER="Authorization: Bearer ${BITBUCKET_TOKEN}"
  fi
  PR_JSON=$(curl -fsSL -H "${AUTH_HEADER}" "https://api.bitbucket.org/2.0/repositories/${REPO}/pullrequests/${PR_NUMBER}")
  SOURCE_REPO=$(printf '%s' "${PR_JSON}" | jq -r '.source.repository.full_name // empty')
  SOURCE_BRANCH=$(printf '%s' "${PR_JSON}" | jq -r '.source.branch.name // empty')
  SOURCE_HASH=$(printf '%s' "${PR_JSON}" | jq -r '.source.commit.hash // empty')
  if [ -z "${SOURCE_REPO}" ] || [ "${SOURCE_REPO}" = "null" ]; then
    SOURCE_REPO="${REPO}"
  fi
  if [ -z "${SOURCE_BRANCH}" ] || [ "${SOURCE_BRANCH}" = "null" ] || [ -z "${SOURCE_HASH}" ] || [ "${SOURCE_HASH}" = "null" ]; then
    echo "Could not resolve the pull request source repository." >&2
    exit 1
  fi
  if ! printf '%s' "${SOURCE_REPO}" | grep -Eq '^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$'; then
    echo "Invalid pull request source repository." >&2
    exit 1
  fi
  if ! git check-ref-format --branch "${SOURCE_BRANCH}" >/dev/null 2>&1; then
    echo "Invalid pull request source branch." >&2
    exit 1
  fi
  if ! printf '%s' "${SOURCE_HASH}" | grep -Eq '^[0-9a-fA-F]{7,64}$'; then
    echo "Invalid pull request source commit." >&2
    exit 1
  fi
  mkdir -p .superplane
  printf '%s' "${SOURCE_REPO}" > .superplane/source-repo
  printf '%s' "${SOURCE_BRANCH}" > .superplane/source-branch
  if [ ! -d repo/.git ]; then
    rm -rf repo
    git clone "https://bitbucket.org/${SOURCE_REPO}.git" repo
  fi
  cd repo || exit 1
  git fetch origin -- "${SOURCE_BRANCH}"
  SOURCE_HASH=$(git rev-parse --verify --end-of-options "${SOURCE_HASH}^{commit}")
  if [ -n "${PR_REVISION:-}" ]; then
    if ! printf '%s' "${PR_REVISION}" | grep -Eq '^[0-9a-fA-F]{7,64}$'; then
      echo "Invalid pull request revision." >&2
      exit 1
    fi
    PR_REVISION=$(git rev-parse --verify --end-of-options "${PR_REVISION}^{commit}")
  fi
  if [ -n "${PR_REVISION:-}" ] && [ "${SOURCE_HASH}" != "${PR_REVISION}" ]; then
    echo "Remote pull request head changed. Stop without checking out." >&2
    exit 1
  fi
  FETCHED=$(git rev-parse FETCH_HEAD)
  if [ "${FETCHED}" != "${SOURCE_HASH}" ]; then
    echo "Source branch tip does not match the pull request commit. Stop without changing it." >&2
    exit 1
  fi
  printf '%s' "${SOURCE_HASH}" > ../.superplane/source-hash
  git checkout -B "${SOURCE_BRANCH}" "${SOURCE_HASH}"
  if [ "$(git rev-parse HEAD)" != "${SOURCE_HASH}" ]; then
    echo "Local checkout does not match the pull request source commit." >&2
    exit 1
  fi
  cd "$_sp_root" || exit 1
}

if [ -n "${SOURCE_REPO:-}" ] && [ -n "${PR_REVISION:-}" ]; then
  checkout_bitbucket_revision
elif [ -n "${PR_HEAD:-}" ] && [ "${PR_HEAD}" != "null" ] && { [ -z "${BITBUCKET_TOKEN:-}" ] || [ -n "${GITHUB_TOKEN:-}" ]; }; then
  checkout_github_branch
elif [ -n "${BITBUCKET_TOKEN:-}" ] && [ -n "${PR_NUMBER:-}" ]; then
  checkout_bitbucket_pull_request
elif [ -n "${PR_NUMBER:-}" ] && [ -z "${PR_REVISION:-}" ]; then
  checkout_github_branch
elif [ -n "${PR_REVISION:-}" ]; then
  checkout_github_revision
else
  clone_default_branch
` + workspaceCheckoutCommitSetup() + `
fi

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
// handling inside repo/. Implement sets CHECKOUT_COMMIT_SETUP=1 on the node
// environment. The script is sourced, so it saves and restores the working
// directory instead of a subshell: the exports below must persist in the
// caller, and the context clones after this block use paths relative to
// the checkout root. Pull request checkouts skip this block.
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
		Mode:    "0755",
	})
}

func taskShipsCheckout(files []BrokerTaskFile) bool {
	for i := range files {
		if files[i].Path == WorkspaceCheckoutScriptPath {
			return true
		}
	}
	return false
}

const (
	setupHintPending   = "\nSetup commands (run in repo/ after clone, in order): "
	setupHintDone      = "\nSetup commands already ran in repo/ after checkout, in order: "
	contextHintPending = "\nCheckout: run `checkout` to check out repo/ and the context repositories. From repo/, read: "
	contextHintDone    = "\nContext repositories are already checked out next to repo/. From repo/, read: "
)

// ApplyFactoryCheckout inserts the checkout command after setup and before
// the canvas steps. Factory tasks ship the helper file. A canvas that
// already clones keeps that step and does not get a second checkout.
// Prompt hints stay on the saved-clone instructions unless this inserts
// the command, because only then have setup and context checkout run.
func ApplyFactoryCheckout(files []BrokerTaskFile, commands []BrokerCommand, steps []AgentStep) []BrokerCommand {
	if !taskShipsCheckout(files) {
		return commands
	}
	return InsertFactoryCheckoutCommand(commands, steps, files)
}

// InsertFactoryCheckoutCommand inserts checkout before the first canvas
// step, then one Setup step for every workspace setup command. Both are
// regular commands so the task log shows them apart. Steps that already
// run git clone, or the checkout helper, are left unchanged so saved
// automations keep their own clone and run setup themselves.
func InsertFactoryCheckoutCommand(commands []BrokerCommand, steps []AgentStep, files []BrokerTaskFile) []BrokerCommand {
	if stepsAlreadyCheckout(steps) {
		return commands
	}
	markCheckoutHintsComplete(files)
	inserted := factoryCheckoutCommands(files)
	for i, command := range commands {
		if command.Kind == LiveLogKindSetup {
			continue
		}
		out := make([]BrokerCommand, 0, len(commands)+len(inserted))
		out = append(out, commands[:i]...)
		out = append(out, inserted...)
		return append(out, commands[i:]...)
	}
	return append(commands, inserted...)
}

func factoryCheckoutCommands(files []BrokerTaskFile) []BrokerCommand {
	commands := []BrokerCommand{{
		Name:    "Checkout",
		Command: WrapAgentStepCommand("checkout"),
		Kind:    LiveLogKindBash,
		Preview: LiveLogText("Checkout"),
	}}
	setup := setupCommandsFromFiles(files)
	if len(setup) == 0 {
		return commands
	}
	return append(commands, BrokerCommand{
		Name:    "Setup",
		Command: WrapAgentStepCommand(WrapCommandInWorkingDirectory("repo", setupCommandBody(setup))),
		Kind:    LiveLogKindBash,
		Preview: LiveLogText(setupCommandScript(setup)),
	})
}

func setupCommandScript(commands []string) string {
	return strings.Join(commands, "\n")
}

// setupCommandBody runs each command in its own shell. The checkout step
// wraps this body in a status handler that ignores set -e, so a failing
// command must be a separate process.
func setupCommandBody(commands []string) string {
	lines := make([]string, 0, len(commands))
	for _, command := range commands {
		lines = append(lines, "bash -e -c "+ShellSingleQuote(command)+" || exit 1")
	}
	return strings.Join(lines, "\n")
}

func setupCommandsFromFiles(files []BrokerTaskFile) []string {
	for i := range files {
		if files[i].Path != WorkspaceSetupConfigPath {
			continue
		}
		var setup workspaceSetupFile
		if err := json.Unmarshal([]byte(files[i].Content), &setup); err != nil {
			return nil
		}
		return models.NormalizeSetupCommands(setup.Commands)
	}
	return nil
}

func markCheckoutHintsComplete(files []BrokerTaskFile) {
	for i := range files {
		if !strings.HasPrefix(files[i].Path, "prompts/") || !strings.HasSuffix(files[i].Path, ".txt") {
			continue
		}
		content := files[i].Content
		content = strings.ReplaceAll(content, setupHintPending, setupHintDone)
		content = strings.ReplaceAll(content, contextHintPending, contextHintDone)
		files[i].Content = content
	}
}

func stepsAlreadyCheckout(steps []AgentStep) bool {
	for _, step := range steps {
		if NormalizeAgentStepType(step.Type) != AgentStepBash {
			continue
		}
		if commandChecksOutRepository(stringPtrValue(step.Command)) {
			return true
		}
	}
	return false
}

func commandChecksOutRepository(command string) bool {
	if strings.Contains(command, "git clone") {
		return true
	}
	for _, line := range strings.Split(command, "\n") {
		line = strings.TrimSpace(line)
		if line == "checkout" || strings.HasPrefix(line, "checkout ") {
			return true
		}
	}
	return false
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
		builder.WriteString(setupHintPending)
		builder.WriteString(strings.Join(setupCommands, " && "))
	}
	if len(contextRepos) > 0 {
		names := make([]string, 0, len(contextRepos))
		for i := range contextRepos {
			repository := strings.TrimSpace(contextRepos[i].Config.Data().Repository)
			if repository == "" {
				repository = contextRepos[i].Name
			}
			names = append(names, repository+" (../"+WorkspaceContextRepoDirectory(repository)+")")
		}
		builder.WriteString(contextHintPending)
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
