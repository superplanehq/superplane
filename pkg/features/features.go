// Package features holds the registry of experimental features that can be
// enabled per organization. The registry is the source of truth: feature IDs
// referenced by enable/disable APIs must exist here, and a feature can be
// graduated to all organizations by setting Released to a pointer to true.
package features

type Feature struct {
	ID          string
	Label       string
	Description string
	Released    *bool
}

// FeatureClaudeManagedAgents enables the managed-agents chat experience on a
// per-organization basis until the integration is generally available.
const FeatureClaudeManagedAgents = "claude_managed_agents"

// FeatureFactories enables software factories (work orders, lines, factory apps)
// for a organization until the feature is generally available.
const FeatureFactories = "factories"

// FeatureNewIntegrationSetupFlow enables the SetupProvider wizard for
// integrations that register one (for example GitHub). When disabled, create
// still uses the legacy IntegrationCreateDialog path.
const FeatureNewIntegrationSetupFlow = "new_integration_setup_flow"

// FeatureFactoryProductiveIntake gates the manual "Add intake" entry for
// Productive.io in the Backlog column menu until the flow is generally
// available.
const FeatureFactoryProductiveIntake = "factory_productive_intake"

// FeatureFactoryDatadogIntake gates Datadog Error Tracking intake from the
// Backlog column menu until the flow is generally available.
const FeatureFactoryDatadogIntake = "factory_datadog_intake"

// FeatureFactoryLinearIntake gates Linear issue intake from the Backlog
// column menu until the flow is generally available.
const FeatureFactoryLinearIntake = "factory_linear_intake"

// FeatureFactoryBitbucket gates Bitbucket as a workspace Git host until the
// flow is generally available.
const FeatureFactoryBitbucket = "factory_bitbucket"

// FeatureWorkspaceModels gates the in-progress workspace Models settings
// page until it is ready for general use.
const FeatureWorkspaceModels = "workspace_models"

// FeatureOrganizationBYOK gates the organization LLM Models settings page
// until the BYOK UX is ready for general use.
const FeatureOrganizationBYOK = "organization_byok"

// FeatureOrganizationBYOKCustomProvider gates a custom model provider on the
// organization LLM Models page. It requires FeatureOrganizationBYOK as well.
const FeatureOrganizationBYOKCustomProvider = "organization_byok_custom_provider"

// FeatureFactoryCustomAutomations gates blank custom automations on Verify,
// Done, and phase columns until the flow is generally available.
const FeatureFactoryCustomAutomations = "factory_custom_automations"

// FeatureWorkspaceMCP gates workspace MCP settings, MCP APIs, and MCP attach.
const FeatureWorkspaceMCP = "workspace_mcp"

// FeatureWorkspaceSkills gates workspace skill settings, skill APIs, and skill attach.
const FeatureWorkspaceSkills = "workspace_skills"

// FeatureNewRunners routes newly created runner tasks through SuperPlane's
// integrated runner API. Existing executions keep their persisted backend.
const FeatureNewRunners = "new_runners"

// FeatureFactoryPullRequestMerge gates the Mergeable chip on task cards and
// the Merge button on pull request review until the feature is generally
// available.
const FeatureFactoryPullRequestMerge = "factory_pull_request_merge"

// FeatureFactoryRiskScore gates the merge confidence automation on the Verify
// column until the flow is generally available.
const FeatureFactoryRiskScore = "factory_risk_score"

// FeatureSuperPlaneMCPServer is released. Every organization can authorize
// Cursor and other MCP clients against one workspace. The factories flag
// still applies.
const FeatureSuperPlaneMCPServer = "superplane_mcp_server"

// FeatureMobileFactoryBoard gates the mobile-first workspace shell: a
// one-column-at-a-time line board with a bottom action bar and a full-screen
// task view on phone-width screens. Desktop screens keep the standard shell.
const FeatureMobileFactoryBoard = "mobile_factory_board"

// FeatureTaskPlanningReview enables the new task planning UI per organization.
const FeatureTaskPlanningReview = "task_planning_review"

func released() *bool {
	v := true
	return &v
}

var registry = []Feature{
	{ID: FeatureTaskPlanningReview, Label: "Task Planning Review", Description: "Show the new plan card, focused questions, and implementation controls"},
	{ID: FeatureClaudeManagedAgents, Label: "Claude Managed Agents", Description: "Chat with a Claude-powered agent against the canvas", Released: released()},
	{ID: FeatureFactories, Label: "Factories", Description: "Software factories for work orders and production workflows"},
	{ID: FeatureNewIntegrationSetupFlow, Label: "New Integration Setup Flow", Description: "Use the multi-step SetupProvider wizard when connecting integrations such as GitHub"},
	{ID: FeatureFactoryProductiveIntake, Label: "Factory Productive Intake", Description: "Add Productive intake from the Backlog column menu"},
	{ID: FeatureFactoryDatadogIntake, Label: "Factory Datadog Intake", Description: "Add Datadog intake from the Backlog column menu"},
	{ID: FeatureFactoryLinearIntake, Label: "Factory Linear Intake", Description: "Add Linear intake from the Backlog column menu"},
	{ID: FeatureFactoryBitbucket, Label: "Bitbucket Workspaces", Description: "Connect a Bitbucket workspace and open pull requests from Implement"},
	{ID: FeatureWorkspaceModels, Label: "Workspace Models", Description: "Show the in-progress workspace Models settings page"},
	{ID: FeatureOrganizationBYOK, Label: "Organization BYOK", Description: "Show the organization LLM Models settings page"},
	{ID: FeatureOrganizationBYOKCustomProvider, Label: "Organization BYOK Custom Provider", Description: "Add a custom model provider with a URL, token, and API type"},
	{ID: FeatureFactoryCustomAutomations, Label: "Custom Automations", Description: "Add a blank custom automation to a board column"},
	{ID: FeatureWorkspaceMCP, Label: "Workspace MCP", Description: "Add MCP servers for workspace agents"},
	{ID: FeatureWorkspaceSkills, Label: "Workspace Skills", Description: "Add skills for workspace agents"},
	{ID: FeatureNewRunners, Label: "New Runners", Description: "Run tasks with the integrated SuperPlane runner architecture"},
	{ID: FeatureFactoryPullRequestMerge, Label: "Pull Request Merge", Description: "Show the Mergeable chip on task cards and the Merge button on pull request review"},
	{ID: FeatureFactoryRiskScore, Label: "Factory Merge Confidence", Description: "Add a merge confidence automation to the Verify column"},
	{ID: FeatureSuperPlaneMCPServer, Label: "MCP Server", Description: "Allow Cursor and other MCP clients to connect to workspaces in this organization", Released: released()},
	{ID: FeatureMobileFactoryBoard, Label: "Mobile Board", Description: "Show the mobile workspace shell on phone-width screens"},
}

func All() []Feature {
	out := make([]Feature, len(registry))
	copy(out, registry)
	return out
}

func Get(id string) (Feature, bool) {
	for _, f := range registry {
		if f.ID == id {
			return f, true
		}
	}
	return Feature{}, false
}

func Exists(id string) bool {
	_, ok := Get(id)
	return ok
}

// IsReleased reports whether the feature with the given id is in the registry
// and marked as released. Released features are considered enabled for every
// organization regardless of per-organization state.
func IsReleased(id string) bool {
	f, ok := Get(id)
	if !ok {
		return false
	}
	return f.Released != nil && *f.Released
}

// WithRegistryForTest temporarily replaces the registry. Intended for tests
// that need to assert behavior of features not present in the production
// registry (e.g. released features). The returned function restores the
// previous registry and should be invoked from a t.Cleanup callback.
func WithRegistryForTest(r []Feature) func() {
	original := registry
	registry = r
	return func() { registry = original }
}
