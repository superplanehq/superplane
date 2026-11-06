// Package features holds the registry of experimental features that can be
// enabled per organization. The registry is the source of truth: feature IDs
// referenced by enable/disable APIs must exist here, and a feature can be
// graduated to all organizations by setting Released to a pointer to true.
// Hidden features stay in the registry but are omitted from UIs and cannot
// be enabled through the admin API.
package features

type Feature struct {
	ID          string
	Label       string
	Description string
	Released    *bool
	// Hidden features stay in the registry so later work can reuse the ID.
	// They do not appear in experimental-feature UIs and cannot be enabled
	// through the admin API.
	Hidden bool
}

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

const FeatureFactoryScheduledIntake = "factory_scheduled_intake"

// FeatureOrganizationBYOK gates the organization Models settings page
// until the BYOK UX is ready for general use.
const FeatureOrganizationBYOK = "organization_byok"

// FeatureOrganizationBYOKCustomProvider gates a custom model provider on the
// organization Models page. It requires FeatureOrganizationBYOK as well.
const FeatureOrganizationBYOKCustomProvider = "organization_byok_custom_provider"

// FeatureFactoryCustomAutomations gates custom automations on Verify, Done,
// and phase columns until the flow is generally available.
const FeatureFactoryCustomAutomations = "factory_custom_automations"

// FeatureWorkspaceSkills gates workspace skill settings, skill APIs, and skill attach.
const FeatureWorkspaceSkills = "workspace_skills"

// FeatureFactoryPullRequestImport gates manual PR import into Verify.
const FeatureFactoryPullRequestImport = "factory_pull_request_import"

// FeatureFactoryPullRequestMerge gates the Mergeable chip on task cards and
// the Merge button on pull request review until the feature is generally
// available.
const FeatureFactoryPullRequestMerge = "factory_pull_request_merge"

// FeatureFactoryMultiRepoTasks lets agents change every Read & Write
// workspace repository and open one pull request per repository. Without
// it, context repositories stay read-only and the Implement app opens the
// pull request from the canvas. The admin toggle republishes the Implement
// apps of the organization. Releasing the flag does not: run the
// installation-wide reset of "line-implementation" after the release.
const FeatureFactoryMultiRepoTasks = "factory_multi_repo_tasks"

// FeatureSuperPlaneMCPServer is released. Every organization can authorize
// Cursor and other MCP clients against one workspace. The factories flag
// still applies.
const FeatureSuperPlaneMCPServer = "superplane_mcp_server"

// FeatureGitHubEnterprise is the org rollout gate for GitHub Enterprise Cloud
// (ghe.com). Connecting a host other than github.com also requires the
// github_enterprise license.
const FeatureGitHubEnterprise = "github_enterprise"

// FeatureGitHubEnterpriseServer is registered and hidden. GitHub Enterprise
// Server setup answers that the server is not available yet. Connecting a
// GitHub Enterprise Server host also requires the github_enterprise license.
const FeatureGitHubEnterpriseServer = "github_enterprise_server"

// FeatureFactoryStepActions shows the Actions timeline on a step card.
// An organization must enable it.
const FeatureFactoryStepActions = "factory_step_actions"

const FeatureOrganizationRunners = "organization_runners"

func released() *bool {
	v := true
	return &v
}

var registry = []Feature{
	{ID: FeatureFactoryPullRequestImport, Label: "Pull request import", Description: "Import open pull requests into Verify"},
	{ID: FeatureGitHubEnterprise, Label: "GitHub Enterprise Cloud", Description: "Connect GitHub Enterprise Cloud (ghe.com) with a private GitHub App"},
	{ID: FeatureGitHubEnterpriseServer, Label: "GitHub Enterprise Server", Description: "Connect GitHub Enterprise Server. This option is not available yet", Hidden: true},
	{ID: FeatureOrganizationRunners, Label: "Organization runners", Description: "Manage self-hosted fleets and runners for an organization"},
	{ID: FeatureFactories, Label: "Factories", Description: "Software factories for work orders and production workflows"},
	{ID: FeatureNewIntegrationSetupFlow, Label: "New Integration Setup Flow", Description: "Use the multi-step SetupProvider wizard when connecting integrations such as GitHub"},
	{ID: FeatureFactoryProductiveIntake, Label: "Factory Productive Intake", Description: "Add Productive intake from the Backlog column menu"},
	{ID: FeatureFactoryScheduledIntake, Label: "Scheduled Intake", Description: "Propose backlog tasks on a schedule"},
	{ID: FeatureFactoryDatadogIntake, Label: "Factory Datadog Intake", Description: "Add Datadog intake from the Backlog column menu"},
	{ID: FeatureOrganizationBYOK, Label: "Organization BYOK", Description: "Show the organization Models settings page"},
	{ID: FeatureOrganizationBYOKCustomProvider, Label: "Organization BYOK Custom Provider", Description: "Add a custom model provider with a URL, token, and API type"},
	{ID: FeatureFactoryCustomAutomations, Label: "Custom Automations", Description: "Add custom automations to board columns"},
	{ID: FeatureWorkspaceSkills, Label: "Workspace Skills", Description: "Add skills for workspace agents"},
	{ID: FeatureFactoryPullRequestMerge, Label: "Pull Request Merge", Description: "Show the Mergeable chip on task cards and the Merge button on pull request review"},
	{ID: FeatureSuperPlaneMCPServer, Label: "MCP Server", Description: "Allow Cursor and other MCP clients to connect to workspaces in this organization", Released: released()},
	{ID: FeatureFactoryStepActions, Label: "Step actions", Description: "Show agent action reasons on the step card"},
	{ID: FeatureFactoryMultiRepoTasks, Label: "Multi-repo tasks", Description: "Let agents change every Read & Write workspace repository and open one pull request per repository"},
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

// IsHidden reports whether the feature is in the registry and omitted from
// experimental-feature UIs. Hidden features cannot be enabled through the
// admin API.
func IsHidden(id string) bool {
	f, ok := Get(id)
	return ok && f.Hidden
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
