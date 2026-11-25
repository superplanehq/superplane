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

// FeatureFactoryDatadogIntake gates Datadog Error Tracking intake from the
// Backlog column menu until the flow is generally available.
const FeatureFactoryDatadogIntake = "factory_datadog_intake"

const FeatureFactoryScheduledIntake = "factory_scheduled_intake"

// FeatureFactoryReviewRequest gates new reviewer-request automations. Existing
// automations continue to run when the organization disables discovery.
const FeatureFactoryReviewRequest = "factory_review_request"

// FeatureOrganizationBYOK gates the organization Models settings page
// until the BYOK UX is ready for general use.
const FeatureOrganizationBYOK = "organization_byok"

// FeatureModelStatus gates the hosted Model status page in organization settings.
const FeatureModelStatus = "model_status"

// FeatureOrganizationBYOKCustomProvider gates a custom model provider on the
// organization Models page. It requires FeatureOrganizationBYOK as well.
const FeatureOrganizationBYOKCustomProvider = "organization_byok_custom_provider"

// FeatureFactoryCustomAutomations gates custom automations and line-level
// custom board stages until the flow is generally available.
const FeatureFactoryCustomAutomations = "factory_custom_automations"

const FeatureAutomationsAsCode = "automations-as-code"

// FeatureWorkspaceSkills gates workspace skill settings, skill APIs, and skill attach.
const FeatureWorkspaceSkills = "workspace_skills"

// FeatureFactoryPullRequestImport gates manual PR import into Verify.
const FeatureFactoryPullRequestImport = "factory_pull_request_import"

// FeatureFactoryPullRequestMerge gates the Mergeable chip on task cards and
// the Merge button on pull request review until the feature is generally
// available.
const FeatureFactoryPullRequestMerge = "factory_pull_request_merge"

// FeatureSuperPlaneMCPServer is released. Every organization can authorize
// Cursor and other MCP clients against one workspace. The factories flag
// still applies.
const FeatureSuperPlaneMCPServer = "superplane_mcp_server"

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
	{ID: FeatureGitHubEnterpriseServer, Label: "GitHub Enterprise Server", Description: "Connect GitHub Enterprise Server. This option is not available yet", Hidden: true},
	{ID: FeatureOrganizationRunners, Label: "Organization runners", Description: "Manage self-hosted fleets and runners for an organization"},
	{ID: FeatureFactories, Label: "Factories", Description: "Software factories for work orders and production workflows"},
	{ID: FeatureNewIntegrationSetupFlow, Label: "New Integration Setup Flow", Description: "Use the multi-step SetupProvider wizard when connecting integrations such as GitHub"},
	{ID: FeatureFactoryScheduledIntake, Label: "Scheduled Intake", Description: "Propose backlog tasks on a schedule"},
	{ID: FeatureFactoryReviewRequest, Label: "Review request", Description: "Request human reviews after Merge confidence completes"},
	{ID: FeatureFactoryDatadogIntake, Label: "Factory Datadog Intake", Description: "Add Datadog intake from the Backlog column menu"},
	{ID: FeatureOrganizationBYOK, Label: "Organization BYOK", Description: "Show the organization Models settings page"},
	{ID: FeatureModelStatus, Label: "Model status", Description: "Show the hosted Model status page in organization settings"},
	{ID: FeatureOrganizationBYOKCustomProvider, Label: "Organization BYOK Custom Provider", Description: "Add a custom model provider with a URL, token, and API type"},
	{ID: FeatureFactoryCustomAutomations, Label: "Custom automations and stages", Description: "Add custom automations to board columns and manage custom stages on factory line boards"},
	{ID: FeatureAutomationsAsCode, Label: "Automations as code", Description: "Edit custom automations in YAML or through the API"},
	{ID: FeatureWorkspaceSkills, Label: "Workspace Skills", Description: "Add skills for workspace agents"},
	{ID: FeatureFactoryPullRequestMerge, Label: "Pull Request Merge", Description: "Show the Mergeable chip on task cards and the Merge button on pull request review"},
	{ID: FeatureSuperPlaneMCPServer, Label: "MCP Server", Description: "Allow Cursor and other MCP clients to connect to workspaces in this organization", Released: released()},
	{ID: FeatureFactoryStepActions, Label: "Step actions", Description: "Show agent action reasons on the step card"},
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
