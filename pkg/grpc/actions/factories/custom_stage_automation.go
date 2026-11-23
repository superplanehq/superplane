package factories

import (
	"errors"
	"maps"
	"strings"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

func validateCustomStageAutomation(tx *gorm.DB, factory *models.Factory, column string) error {
	if !strings.HasPrefix(column, "custom-") {
		return nil
	}
	organization, err := models.FindOrganizationByIDInTransaction(tx, factory.OrganizationID.String())
	if err != nil {
		return err
	}
	if !organization.HasExperimentalFeature(features.FeatureFactoryCustomStages) {
		return errFactoryCustomStagesDisabled
	}
	lines, err := factory.ListLines(tx)
	if err != nil {
		return err
	}
	for _, line := range lines {
		for _, stage := range line.StagesValue() {
			if stage.Key == column && stage.Kind == models.FactoryLineStageCustom {
				return nil
			}
		}
	}
	return invalidArgument("Select a custom stage in this factory.")
}

// The line owns gate admission. Event-trigger runs never own that gate.
func buildCustomStageAutomationCanvas(tx *gorm.DB, factory *models.Factory, name string, settings *customAutomationSettings, provider string) (*yaml.Canvas, error) {
	phase := *settings
	phase.ColumnKey = models.CanvasColumnKeyImplement
	doc, err := buildCustomAutomationCanvas(tx, factory, name, &phase)
	if err != nil {
		return nil, err
	}
	metadata := customAutomationNodeMetadata(provider, settings.Icon, settings.RunsOnTask, func(template map[string]any) { template["stageGate"] = true })
	doc.Spec.Nodes = append(doc.Spec.Nodes, yaml.Node{
		ID: "on-run", Name: "On Run", Type: yaml.NodeTypeTrigger, Component: "onRun", Configuration: map[string]any{}, Metadata: metadata,
	}, customAutomationRunnerNode("stage-run", "Run stage automation", settings,
		customAutomationRunnerConfiguration(tx, factory, settings, customAutomationCheckoutEnvironment(factory, pb.FactoryCustomAutomation_START_MODE_SCHEDULED, githubHostForFactory(tx, factory), false)), 80))
	doc.Spec.Edges = append(doc.Spec.Edges, yaml.Edge{SourceID: "on-run", TargetID: "stage-run", Channel: "default"})
	return doc, nil
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
		node.Configuration["steps"] = customAutomationStepConfiguration(settings.Steps)
		addCustomStageStepSecrets(node.Configuration, settings.Steps)
		node.Configuration["executionTimeoutSeconds"] = settings.ExecutionTimeoutSeconds
		if settings.model() != "" {
			node.Configuration["model"] = settings.model()
		} else {
			delete(node.Configuration, "model")
		}
		if settings.SkipSetupCommands != nil {
			node.Configuration["skipSetupCommands"] = *settings.SkipSetupCommands
		}
		if settings.SkipRepositoryCheckout != nil {
			node.Configuration["skipRepositoryCheckout"] = *settings.SkipRepositoryCheckout
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
	}
}

func preserveCustomStageGate(version *models.CanvasVersion, doc *yaml.Canvas) {
	var gate *models.Node
	for i := range version.Nodes {
		if version.Nodes[i].ComponentName() == "onRun" {
			gate = &version.Nodes[i]
			break
		}
	}
	if gate == nil {
		return
	}
	var agent *models.Node
	for _, edge := range version.Edges {
		if edge.SourceID != gate.ID {
			continue
		}
		for i := range version.Nodes {
			if version.Nodes[i].ID == edge.TargetID && isFactoryAgentHarness(version.Nodes[i].ComponentName()) {
				agent = &version.Nodes[i]
			}
		}
	}
	for i := range doc.Spec.Nodes {
		node := &doc.Spec.Nodes[i]
		if node.ID == "stage-run" && agent != nil {
			configuration := maps.Clone(agent.Configuration)
			requested, _ := node.Configuration["environmentFrom"].([]any)
			mergeCustomStageSecrets(configuration, requested)
			for _, key := range []string{"steps", "executionTimeoutSeconds", "model", "credentials", "skipSetupCommands", "skipRepositoryCheckout", "llmProvider"} {
				if value, ok := node.Configuration[key]; ok {
					configuration[key] = value
				} else {
					delete(configuration, key)
				}
			}
			node.Configuration = configuration
			node.ID = customStageAgentID(agent.ID, agent.ComponentName(), node.Component)
			if agent.Concurrency != nil {
				node.Concurrency = &yaml.ConcurrencySpec{Max: agent.Concurrency.Max, Key: agent.Concurrency.Key}
			}
			for j := range doc.Spec.Edges {
				if doc.Spec.Edges[j].TargetID == "stage-run" {
					doc.Spec.Edges[j].TargetID = node.ID
				}
			}
			continue
		}
		if node.ID != "on-run" {
			continue
		}
		node.ID = gate.ID
		node.Configuration = gate.Configuration
		for j := range doc.Spec.Edges {
			if doc.Spec.Edges[j].SourceID == "on-run" {
				doc.Spec.Edges[j].SourceID = gate.ID
			}
		}
	}
}
