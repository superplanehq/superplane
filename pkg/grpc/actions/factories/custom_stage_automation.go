package factories

import (
	"errors"
	"maps"
	"slices"
	"strings"

	factorycomponent "github.com/superplanehq/superplane/pkg/components/factory"
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
	if settings.hasLineTrigger() {
		metadata := customAutomationNodeMetadata(provider, settings.Icon, settings.RunsOnTask, func(template map[string]any) {
			applyCustomStageTriggerMetadata(template, settings)
		})
		doc.Spec.Nodes = []yaml.Node{{
			ID: "on-run", Name: "On Run", Type: yaml.NodeTypeTrigger, Component: "onRun", Configuration: map[string]any{}, Metadata: metadata,
		}, customAutomationRunnerNode(customStageRunnerID, "Run stage automation", settings,
			customAutomationRunnerConfiguration(tx, factory, settings, []any{
				checkoutEnvironmentVar("REPO", "{{ order().repository }}"),
				checkoutEnvironmentVar("REPO_URL", "{{ order().repository_url }}"),
				checkoutEnvironmentVar("BASE", "{{ order().default_branch }}"),
			}), 80)}
		doc.Spec.Edges = []yaml.Edge{{SourceID: "on-run", TargetID: customStageRunnerID, Channel: "default"}}
	}
	if !settings.hasWebhookTrigger() {
		return doc, nil
	}
	webhookRunner := customAutomationRunnerNode(customStageWebhookRunnerID, "Run webhook", settings,
		customAutomationRunnerConfiguration(tx, factory, settings,
			customAutomationCheckoutEnvironment(factory, pb.FactoryCustomAutomation_START_MODE_SCHEDULED, githubHostForFactory(tx, factory), false),
		), 480)
	markNoTaskRunner(&webhookRunner, provider, settings.Icon)
	appendWebhookPayloadInstructions(webhookRunner.Configuration)
	doc.Spec.Nodes = append(doc.Spec.Nodes, webhookRunner)
	offset := 0
	for _, trigger := range settings.Triggers {
		if trigger.Mode != pb.FactoryCustomAutomation_START_MODE_WEBHOOK {
			continue
		}
		offset++
		node := customAutomationWebhookNode(provider, settings, trigger, 80+offset*180)
		doc.Spec.Nodes = append(doc.Spec.Nodes, node)
		doc.Spec.Edges = append(doc.Spec.Edges, yaml.Edge{SourceID: node.ID, TargetID: customStageWebhookRunnerID, Channel: "default"})
	}
	doc.Metadata.Description = "Run prompt and Bash steps when a webhook receives a request."
	if settings.hasLineTrigger() {
		doc.Metadata.Description = "Run prompt and Bash steps on a task in the stage or when a webhook receives a request."
	}
	return doc, nil
}

func validateCustomStageTriggers(settings *customAutomationSettings) error {
	automatic := 0
	for _, trigger := range settings.Triggers {
		switch trigger.Mode {
		case pb.FactoryCustomAutomation_START_MODE_AUTOMATIC:
			automatic++
		case pb.FactoryCustomAutomation_START_MODE_MANUAL, pb.FactoryCustomAutomation_START_MODE_WEBHOOK:
		default:
			return invalidArgument("Custom stage automations support only Manual, task_enters_stage, and webhook triggers.")
		}
	}
	if automatic > 1 || len(settings.Triggers) == 0 || settings.runsWithoutTask() {
		return invalidArgument("Custom stage automations need a Manual trigger, one task_enters_stage trigger, or a webhook trigger.")
	}
	return nil
}

// The on-run trigger holds only the task triggers. Webhook triggers have
// their own nodes.
func applyCustomStageTriggerMetadata(template map[string]any, settings *customAutomationSettings) {
	kinds := make([]any, 0, len(settings.Triggers))
	for _, trigger := range settings.Triggers {
		if trigger.Mode != pb.FactoryCustomAutomation_START_MODE_WEBHOOK {
			kinds = append(kinds, customAutomationStartModeName(trigger.Mode))
		}
	}
	mode := settings.StartMode
	if mode == pb.FactoryCustomAutomation_START_MODE_WEBHOOK || mode == pb.FactoryCustomAutomation_START_MODE_UNSPECIFIED {
		mode = pb.FactoryCustomAutomation_START_MODE_MANUAL
		if settings.hasTrigger(pb.FactoryCustomAutomation_START_MODE_AUTOMATIC) {
			mode = pb.FactoryCustomAutomation_START_MODE_AUTOMATIC
		}
	}
	template["startMode"] = customAutomationStartModeName(mode)
	template["triggerKinds"] = kinds
	if settings.RunsOnTask != nil {
		template["runsOnTask"] = *settings.RunsOnTask
	}
}

// The line owns the task trigger and its runner. Webhook triggers and their
// runner are generated, so they do not change the line-owned graph. A
// webhook-only graph has no line-owned part.
func legacyCustomStageGraph(canvas *models.Canvas, version *models.CanvasVersion) bool {
	core := customStageCore(version)
	if canvas.ColumnKey == nil || !strings.HasPrefix(*canvas.ColumnKey, "custom-") {
		return false
	}
	if len(core.Nodes) == 0 && len(core.Edges) == 0 {
		return savedCustomStageWebhookRunner(version.Nodes) != nil
	}
	version = core
	if len(version.Nodes) != 2 || len(version.Edges) != 1 {
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
		previousID := node.ID
		editCustomStageRunner(node, customStageRunnerID, settings)
		if node.ID != previousID {
			// A provider change retires the old runner identity.
			for j := range edges {
				if edges[j].TargetID == previousID {
					edges[j].TargetID = node.ID
				}
			}
		}
	}
	return nodes, edges
}

// Only form-owned fields change. Canvas-owned values, such as extra
// environment entries and secret sources, stay.
func editCustomStageRunner(node *models.Node, baseID string, settings *customAutomationSettings) {
	node.Configuration = maps.Clone(node.Configuration)
	if node.ComponentName() != settings.Agent.component() {
		node.ID = baseID + "-" + strings.ToLower(strings.ReplaceAll(settings.Agent.component(), ".", "-"))
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

// Saved runner settings stay. The build supplies only new webhook nodes.
func editCustomStageGraph(live *models.CanvasVersion, settings *customAutomationSettings, built []models.Node, builtEdges []models.Edge) ([]models.Node, []models.Edge) {
	var nodes []models.Node
	var edges []models.Edge
	if settings.hasLineTrigger() {
		core := customStageCore(live)
		if len(core.Nodes) == 0 {
			core = customStageCore(&models.CanvasVersion{Nodes: built, Edges: builtEdges})
		} else {
			core.Nodes, core.Edges = editLegacyCustomStageGraph(core, settings)
		}
		nodes, edges = core.Nodes, core.Edges
	}
	for i := range nodes {
		if nodes[i].ComponentName() == "onRun" {
			setCustomStageRunContext(&nodes[i], settings.RunsOnTask)
		}
	}
	runnerID := customStageWebhookRunnerID
	for _, node := range built {
		if !customStageWebhookPart(node) {
			continue
		}
		if saved := savedCustomStageWebhookRunner(live.Nodes); saved != nil && isFactoryAgentHarness(node.ComponentName()) {
			node = *saved
			editCustomStageRunner(&node, customStageWebhookRunnerID, settings)
			appendWebhookPayloadInstructions(node.Configuration)
		}
		if isFactoryAgentHarness(node.ComponentName()) {
			runnerID = node.ID
		}
		nodes = append(nodes, node)
	}
	for _, edge := range builtEdges {
		if edge.TargetID == customStageWebhookRunnerID {
			edge.TargetID = runnerID
			edges = append(edges, edge)
		}
	}
	return nodes, edges
}

// A webhook runs with no task, so the line trigger must not keep runsOnTask: true.
func setCustomStageRunContext(node *models.Node, runsOnTask *bool) {
	template, ok := node.Metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
	if !ok {
		return
	}
	template = maps.Clone(template)
	delete(template, "runsOnTask")
	if runsOnTask != nil {
		template["runsOnTask"] = *runsOnTask
	}
	node.Metadata = maps.Clone(node.Metadata)
	node.Metadata[models.FactoryAppTemplateMetadataKey] = template
}

func savedCustomStageWebhookRunner(nodes []models.Node) *models.Node {
	for i := range nodes {
		if customStageWebhookPart(nodes[i]) && isFactoryAgentHarness(nodes[i].ComponentName()) {
			return &nodes[i]
		}
	}
	return nil
}

func customStageCore(version *models.CanvasVersion) *models.CanvasVersion {
	core := &models.CanvasVersion{}
	webhookParts := map[string]bool{}
	for _, node := range version.Nodes {
		if customStageWebhookPart(node) {
			webhookParts[node.ID] = true
			continue
		}
		core.Nodes = append(core.Nodes, node)
	}
	for _, edge := range version.Edges {
		if !webhookParts[edge.SourceID] && !webhookParts[edge.TargetID] {
			core.Edges = append(core.Edges, edge)
		}
	}
	return core
}

func customStageWebhookPart(node models.Node) bool {
	return node.ID == customStageWebhookRunnerID || strings.HasPrefix(node.ID, customStageWebhookRunnerID+"-") ||
		node.ComponentName() == factorycomponent.OnWebhookTriggerName
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
