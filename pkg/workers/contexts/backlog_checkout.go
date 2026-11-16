package contexts

import (
	"fmt"
	"maps"
	"strings"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/models"
)

// BacklogSharedCheckoutEnvironment preserves task checkout values and supports standalone runs.
func BacklogSharedCheckoutEnvironment(factory *models.Factory, githubHost string) []any {
	config := factory.OnboardingConfigValue()
	repository := strings.TrimSpace(config.AppRepository)
	branch := strings.TrimSpace(config.DefaultBranch)
	url := runner.WorkspaceContextRepoURL(config.EffectiveVCSProvider(), repository, githubHost)
	quote := func(value string) string { return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value) }
	env := []any{
		map[string]any{"name": "REPO", "value": fmt.Sprintf(`{{ root().data.workOrder?.repository ?? "%s" }}`, quote(repository)), "valueSource": runner.EnvironmentValueSourceLiteral},
		map[string]any{"name": "REPO_URL", "value": fmt.Sprintf(`{{ root().data.workOrder?.repository_url ?? "%s" }}`, quote(url)), "valueSource": runner.EnvironmentValueSourceLiteral},
		map[string]any{"name": "SOURCE_REPO", "value": repository, "valueSource": runner.EnvironmentValueSourceLiteral},
		map[string]any{"name": "BASE", "value": fmt.Sprintf(`{{ root().data.workOrder?.default_branch ?? "%s" }}`, quote(branch)), "valueSource": runner.EnvironmentValueSourceLiteral},
	}
	return env
}

// Old saved Backlog runners must also work without another settings save.
func (b *NodeConfigurationBuilder) withBacklogCheckoutFallback(configuration map[string]any) (map[string]any, error) {
	legacy := map[string]string{
		"REPO":     "{{ $['on-task'].data.workOrder.repository }}",
		"REPO_URL": "{{ $['on-task'].data.workOrder.repository_url }}",
		"BASE":     "{{ $['on-task'].data.workOrder.default_branch }}",
	}
	environment, _ := configuration["environment"].([]any)
	needsFallback := false
	for _, item := range environment {
		variable, _ := item.(map[string]any)
		name, _ := variable["name"].(string)
		if value, ok := legacy[name]; ok && variable["value"] == value {
			needsFallback = true
		}
	}
	if !needsFallback || b.nodeID != "verify" {
		return configuration, nil
	}
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(b.tx, b.workflowID)
	if err != nil {
		return nil, err
	}
	if canvas.FactoryID == nil || canvas.ColumnKey == nil || *canvas.ColumnKey != models.CanvasColumnKeyBacklog {
		return configuration, nil
	}
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(b.tx, canvas)
	if err != nil {
		return nil, err
	}
	if models.FactoryAppTemplateID(version.Nodes) != models.FactoryAppTemplateCustomAutomationID {
		return configuration, nil
	}
	factory, err := models.FindFactory(b.tx, canvas.OrganizationID, *canvas.FactoryID)
	if err != nil {
		return nil, err
	}
	host, err := NewFactoryContext(b.tx, canvas, nil).GitHubHost()
	if err != nil {
		return nil, err
	}
	fallbacks := map[string]any{}
	for _, item := range BacklogSharedCheckoutEnvironment(factory, host) {
		variable := item.(map[string]any)
		fallbacks[variable["name"].(string)] = variable["value"]
	}
	updated := make([]any, 0, len(environment))
	for _, item := range environment {
		variable, _ := item.(map[string]any)
		name, _ := variable["name"].(string)
		if value, ok := legacy[name]; ok && variable["value"] == value {
			variable = maps.Clone(variable)
			variable["value"] = fallbacks[name]
			item = variable
		}
		updated = append(updated, item)
	}
	result := maps.Clone(configuration)
	result["environment"] = updated
	return result, nil
}
