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

// FeatureFactorySentryIntake gates the manual "Add intake" entry for Sentry
// in the Backlog column menu until the flow is generally available.
const FeatureFactorySentryIntake = "factory_sentry_intake"

// FeatureWorkspaceModels gates the in-progress workspace Models settings
// page until it is ready for general use.
const FeatureWorkspaceModels = "workspace_models"

func released() *bool {
	v := true
	return &v
}

var registry = []Feature{
	{ID: FeatureClaudeManagedAgents, Label: "Claude Managed Agents", Description: "Chat with a Claude-powered agent against the canvas", Released: released()},
	{ID: FeatureFactories, Label: "Factories", Description: "Software factories for work orders and production workflows"},
	{ID: FeatureNewIntegrationSetupFlow, Label: "New Integration Setup Flow", Description: "Use the multi-step SetupProvider wizard when connecting integrations such as GitHub"},
	{ID: FeatureFactorySentryIntake, Label: "Factory Sentry Intake", Description: "Add Sentry intake from the Backlog column menu"},
	{ID: FeatureWorkspaceModels, Label: "Workspace Models", Description: "Show the in-progress workspace Models settings page"},
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
