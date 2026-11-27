package factories

import (
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

func validateCustomStageAutomation(tx *gorm.DB, factory *models.Factory, column string) error {
	if !strings.HasPrefix(column, "custom-") {
		return nil
	}
	lines, err := factoryAutomationLines(tx, factory)
	if err != nil {
		return err
	}
	for _, line := range lines {
		for _, stage := range line.StagesValue() {
			if stage.Key == column && stage.Kind == models.FactoryLineStagePhase {
				return nil
			}
		}
	}
	return invalidArgument("Select a custom stage in this factory.")
}

// The line owns stage admission. Editing must not add event subscriptions.
func buildCustomStageAutomationCanvas(tx *gorm.DB, factory *models.Factory, name string, settings *customAutomationSettings, provider string) (*yaml.Canvas, error) {
	if err := validateCustomStageTriggers(settings); err != nil {
		return nil, err
	}
	doc := &yaml.Canvas{
		APIVersion: yaml.APIVersion,
		Kind:       yaml.KindCanvas,
		Metadata:   &yaml.CanvasMetadata{Name: name, Description: "Run ordered prompt and Bash automation steps."},
		Spec:       &yaml.CanvasSpec{},
	}
	metadata := customAutomationNodeMetadata(provider, settings.Icon, settings.RunsOnTask, func(template map[string]any) {
		applyCustomStageTriggerMetadata(template, settings)
	})
	doc.Spec.Nodes = []yaml.Node{{
		ID: "on-run", Name: "On Run", Type: yaml.NodeTypeTrigger, Component: "onRun", Configuration: map[string]any{}, Metadata: metadata,
	}, customAutomationRunnerNode("stage-run", "Run stage automation", settings,
		customAutomationRunnerConfiguration(tx, factory, settings, []any{
			checkoutEnvironmentVar("REPO", "{{ order().repository }}"),
			checkoutEnvironmentVar("REPO_URL", "{{ order().repository_url }}"),
			checkoutEnvironmentVar("BASE", "{{ order().default_branch }}"),
		}), 80)}
	doc.Spec.Edges = []yaml.Edge{{SourceID: "on-run", TargetID: "stage-run", Channel: "default"}}
	return doc, nil
}

func validateCustomStageTriggers(settings *customAutomationSettings) error {
	if len(settings.Triggers) == 0 || settings.runsWithoutTask() {
		return invalidArgument("Custom stage automations require a task and a Manual or task_enters_stage trigger.")
	}
	for _, trigger := range settings.Triggers {
		if trigger.Mode != pb.FactoryCustomAutomation_START_MODE_AUTOMATIC && trigger.Mode != pb.FactoryCustomAutomation_START_MODE_MANUAL {
			return invalidArgument("Custom stage automations support only Manual and task_enters_stage triggers.")
		}
	}
	return nil
}

func applyCustomStageTriggerMetadata(template map[string]any, settings *customAutomationSettings) {
	template["startMode"] = customAutomationStartModeName(settings.StartMode)
	kinds := make([]any, 0, len(settings.Triggers))
	for _, trigger := range settings.Triggers {
		kinds = append(kinds, customAutomationStartModeName(trigger.Mode))
	}
	template["triggerKinds"] = kinds
	if settings.RunsOnTask != nil {
		template["runsOnTask"] = *settings.RunsOnTask
	}
}

func legacyCustomStageGraph(canvas *models.Canvas, version *models.CanvasVersion) bool {
	if canvas.ColumnKey == nil || !strings.HasPrefix(*canvas.ColumnKey, "custom-") || len(version.Nodes) != 2 || len(version.Edges) != 1 {
		return false
	}
	var trigger, agent *models.Node
	for i := range version.Nodes {
		node := &version.Nodes[i]
		if node.ComponentName() == "onRun" {
			trigger = node
		}
		if isFactoryAgentHarness(node.ComponentName()) {
			agent = node
		}
	}
	return trigger != nil && agent != nil && version.Edges[0].SourceID == trigger.ID && version.Edges[0].TargetID == agent.ID && version.Edges[0].Channel == "default"
}

func editLegacyCustomStageGraph(version *models.CanvasVersion, settings *customAutomationSettings) ([]models.Node, []models.Edge) {
	nodes := append([]models.Node(nil), version.Nodes...)
	edges := append([]models.Edge(nil), version.Edges...)
	for i := range nodes {
		node := &nodes[i]
		if node.ComponentName() == "onRun" {
			node.Metadata = maps.Clone(node.Metadata)
			if node.Metadata == nil {
				node.Metadata = map[string]any{}
			}
			template, _ := node.Metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
			template = maps.Clone(template)
			if template == nil {
				template = map[string]any{}
			}
			applyCustomStageTriggerMetadata(template, settings)
			node.Metadata[models.FactoryAppTemplateMetadataKey] = template
		}
		if !isFactoryAgentHarness(node.ComponentName()) {
			continue
		}
		node.Configuration = maps.Clone(node.Configuration)
		previousID := node.ID
		node.ID = customStageAgentID(node.ID, node.ComponentName(), settings.Agent.component())
		if node.ID != previousID {
			// A provider change retires the old runner identity.
			for j := range edges {
				if edges[j].TargetID == previousID {
					edges[j].TargetID = node.ID
				}
			}
		}
		node.Ref = models.NodeRef{Component: &models.ComponentRef{Name: settings.Agent.component()}}
		if credentials := settings.Agent.credentials(); credentials != nil && settings.Agent.component() != models.SuperPlaneRunnerComponent {
			node.Configuration["credentials"] = credentials
		} else {
			delete(node.Configuration, "credentials")
		}
		delete(node.Configuration, "llmProvider")
		settings.Agent.applyLLMProvider(node.Configuration)
		addCustomStageStepSecrets(node.Configuration, settings.Steps)
		node.Configuration["steps"] = customAutomationStepConfiguration(settings.Steps)
		node.Configuration["executionTimeoutSeconds"] = settings.ExecutionTimeoutSeconds
		if settings.model() != "" {
			node.Configuration["model"] = settings.model()
		} else {
			delete(node.Configuration, "model")
		}
		if settings.SkipSetupCommands != nil {
			node.Configuration["skipSetupCommands"] = *settings.SkipSetupCommands
		} else {
			delete(node.Configuration, "skipSetupCommands")
		}
		if settings.SkipRepositoryCheckout != nil {
			node.Configuration["skipRepositoryCheckout"] = *settings.SkipRepositoryCheckout
		} else {
			delete(node.Configuration, "skipRepositoryCheckout")
		}
	}
	return nodes, edges
}

func customStageAgentID(id, previousComponent, component string) string {
	if previousComponent == component {
		return id
	}
	return "stage-run-" + strings.ToLower(strings.ReplaceAll(component, ".", "-"))
}

func checkCustomStageFormGraph(tx *gorm.DB, factory *models.Factory, canvas *models.Canvas, version *models.CanvasVersion) error {
	if canvas.ColumnKey == nil || !strings.HasPrefix(*canvas.ColumnKey, "custom-") {
		return nil
	}
	_, err := importAutomationCanvas(tx, factory, canvas, version)
	if errors.Is(err, errUnrepresentableAutomation) && legacyFormPreservesCanvas(tx, factory, canvas, version) {
		return nil
	}
	return err
}

func addCustomStageStepSecrets(configuration map[string]any, steps []runner.AgentStep) {
	var requested []any
	for _, name := range customAutomationSecrets(steps) {
		requested = append(requested, map[string]any{"source": runner.EnvironmentFromSourceSecret, "secret": map[string]any{"secret": name}})
	}
	mergeCustomStageSecrets(configuration, requested)
}

func mergeCustomStageSecrets(configuration map[string]any, requested []any) {
	sources, _ := configuration["environmentFrom"].([]any)
	previousSteps, _ := configuration["steps"].([]any)
	var previousSecrets []string
	for _, value := range previousSteps {
		step, _ := value.(map[string]any)
		secrets, _ := step["secrets"].([]any)
		for _, value := range secrets {
			if name, ok := value.(string); ok {
				previousSecrets = append(previousSecrets, name)
			}
		}
	}
	// Step-owned references are replaced. Canvas-owned sources stay intact.
	sources = slices.DeleteFunc(slices.Clone(sources), func(source any) bool {
		entry, _ := source.(map[string]any)
		secret, _ := entry["secret"].(map[string]any)
		name, _ := secret["secret"].(string)
		return entry["source"] == runner.EnvironmentFromSourceSecret && slices.Contains(previousSecrets, name)
	})
	for _, source := range requested {
		entry, _ := source.(map[string]any)
		secret, _ := entry["secret"].(map[string]any)
		if entry["source"] != runner.EnvironmentFromSourceSecret {
			continue
		}
		name, _ := secret["secret"].(string)
		found := false
		for _, source := range sources {
			entry, _ := source.(map[string]any)
			secret, _ := entry["secret"].(map[string]any)
			if entry["source"] == runner.EnvironmentFromSourceSecret && secret["secret"] == name {
				found = true
				break
			}
		}
		if !found {
			sources = append(sources, map[string]any{"source": runner.EnvironmentFromSourceSecret, "secret": map[string]any{"secret": name}})
		}
	}
	if len(sources) > 0 {
		configuration["environmentFrom"] = sources
	} else {
		delete(configuration, "environmentFrom")
	}
}
