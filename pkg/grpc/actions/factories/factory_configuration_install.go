package factories

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func factoryConfigurationBindingFromGraph(version *models.CanvasVersion, templateID string) factoryConfigurationBinding {
	binding := factoryConfigurationBinding{}
	nodes := version.Nodes
	if source := strings.TrimPrefix(templateID, "intake:"); source != templateID {
		spec := models.LiveCanvasSpec{Nodes: version.Nodes, Edges: version.Edges}
		graph := resolveIntakeGraph(source, spec)
		if trigger := findIntakeNode(version.Nodes, graph.TriggerNodeID); trigger != nil {
			nodes = []models.Node{*trigger}
			binding.ResourceID = graph.TriggerResourceID(spec)
			binding.TeamID = configString(trigger.Configuration, "team")
		}
	}
	for _, node := range nodes {
		if isFactoryAgentHarness(node.ComponentName()) {
			continue
		}
		if binding.IntegrationID == "" && node.IntegrationID != nil {
			binding.IntegrationID = *node.IntegrationID
		}
		if binding.ResourceID == "" {
			binding.ResourceID = staticConfigString(node.Configuration, "repository")
		}
	}
	return binding
}

func configurationBindingOverrides(binding, defaults factoryConfigurationBinding) *factoryConfigurationBinding {
	result := binding
	if result.IntegrationID == defaults.IntegrationID {
		result.IntegrationID = ""
	}
	if result.ResourceID == defaults.ResourceID {
		result.ResourceID = ""
	}
	if result.TeamID == defaults.TeamID {
		result.TeamID = ""
	}
	if result == (factoryConfigurationBinding{}) {
		return nil
	}
	return &result
}

func supplementFactoryConfigurationRecordSettings(tx *gorm.DB, automation *factoryConfigurationAutomationTarget, metadata factoryConfigurationAutomationMetadata) {
	if metadata.intake != nil {
		settings, _ := automation.settings["intake"].(map[string]any)
		settings["paused"] = metadata.intake.Paused()
		defaults, _ := automation.defaultSettings["intake"].(map[string]any)
		defaults["paused"] = false
	}
	if metadata.handler != nil && automation.templateID == prFeedbackChecksTemplateID {
		settings, _ := automation.settings["pr_feedback"].(map[string]any)
		settings["maximum_attempts"] = prFeedbackResetMaximumAttempts(metadata.handler)
		defaults, _ := automation.defaultSettings["pr_feedback"].(map[string]any)
		defaults["maximum_attempts"] = prFeedbackDefaultMaximumAttempts
		spec := models.LiveCanvasSpec{Nodes: automation.version.Nodes, Edges: automation.version.Edges}
		feedback := prFeedbackSettingsFromGraph(resolvePRFeedbackGraph(spec), spec)
		ids := make([]string, 0, len(feedback.RunnerIntegrationNames))
		for _, name := range feedback.RunnerIntegrationNames {
			installation, err := factoryAutomationIntegrationByName(tx, automation.canvas.OrganizationID, name)
			if err == nil {
				ids = append(ids, installation.ID.String())
			}
		}
		settings["runner_integration_ids"] = ids
	}
}

func configurationManifestReference(stage, automation string) string {
	return "__MANIFEST_AUTOMATION__" + stage + "/" + automation
}

func configurationReviewReference(target *factoryConfigurationTarget, canvasID string) map[string]any {
	for _, stage := range target.stages {
		for _, automation := range stage.automations {
			if automation.canvas.ID.String() == canvasID && automation.templateID == "risk-score" {
				return map[string]any{"stage": stage.key, "automation": automation.id}
			}
		}
	}
	for _, owner := range target.ownership {
		if owner.LineID == target.line.ID && owner.CanvasID.String() == canvasID {
			return map[string]any{"stage": owner.StageKey, "automation": owner.ManifestID}
		}
	}
	return nil
}

func exportFactoryConfigurationDependencies(target *factoryConfigurationTarget) error {
	for _, stage := range target.stages {
		for _, automation := range stage.automations {
			if automation.templateID != reviewRequestTemplateID {
				continue
			}
			trigger := findIntakeNode(automation.version.Nodes, reviewRequestTriggerNodeID)
			if trigger == nil {
				return fmt.Errorf("Review request has no source trigger")
			}
			reference := configurationReviewReference(target, configString(trigger.Configuration, "app"))
			if reference == nil {
				continue
			}
			section, _ := automation.settings["template"].(map[string]any)
			if section == nil {
				section = map[string]any{}
				automation.settings["template"] = section
			}
			section["merge_confidence"] = reference
		}
	}
	return nil
}

func resolveFactoryConfigurationDependencies(manifest *factoryConfigurationManifest, target *factoryConfigurationTarget) *pb.FactoryConfigurationDiagnostic {
	type candidate struct{ stage, id, canvas string }
	candidates := map[string]candidate{}
	for _, stage := range manifest.Stages {
		for _, automation := range stage.Automations {
			if automation.prepared == nil || automation.prepared.TemplateID != "risk-score" {
				continue
			}
			canvas := configurationManifestReference(stage.Key, automation.ID)
			if current := automation.current; current != nil {
				canvas = current.canvas.ID.String()
			}
			candidates[stage.Key+"/"+automation.ID] = candidate{stage.Key, automation.ID, canvas}
		}
	}
	for i := range manifest.Stages {
		stage := &manifest.Stages[i]
		for j := range stage.Automations {
			automation := &stage.Automations[j]
			if automation.prepared == nil || automation.prepared.TemplateID != reviewRequestTemplateID {
				continue
			}
			field := fmt.Sprintf("stages[%d].automations[%d].with.template.merge_confidence", i, j)
			var selected candidate
			if automation.dependencyPresent {
				reference, ok := automation.dependency.(map[string]any)
				if !ok || len(reference) != 2 {
					return configurationDiagnostic(field, "Use a mapping with stage and automation identities.")
				}
				stageKey, stageOK := reference["stage"].(string)
				id, idOK := reference["automation"].(string)
				if !stageOK || !idOK {
					return configurationDiagnostic(field, "Use explicit stage and automation strings.")
				}
				var exists bool
				selected, exists = candidates[stageKey+"/"+id]
				if !exists {
					return configurationDiagnostic(field, "Select a Merge confidence automation included in this manifest.")
				}
			} else {
				if len(candidates) != 1 {
					return configurationDiagnostic(field, "Select a Merge confidence automation. Inference requires exactly one candidate.")
				}
				for _, candidate := range candidates {
					selected = candidate
				}
			}
			trigger := findIntakeNode(automation.prepared.Nodes, reviewRequestTriggerNodeID)
			if trigger == nil {
				return configurationDiagnostic(field, "The Review request trigger is missing.")
			}
			trigger.Configuration["app"] = selected.canvas
			section, _ := automation.With["template"].(map[string]any)
			if section == nil {
				section = map[string]any{}
				automation.With["template"] = section
			}
			section["merge_confidence"] = map[string]any{"stage": selected.stage, "automation": selected.id}
		}
	}
	return nil
}

func configurationBuiltInGraph(current *factoryConfigurationAutomationTarget, desired factoryConfigurationAutomation) ([]models.Node, []models.Edge, error) {
	prepared := desired.prepared
	if current == nil {
		return prepared.Nodes, prepared.Edges, nil
	}
	if factoryConfigurationReviewSourceChanged(current, desired) || current.binding != prepared.Binding || factoryConfigurationBindingChanged(current, desired) && configurationHasVCSListener(current) || !equalCanvasJSON(current.settings["intake"], desired.With["intake"]) || !equalCanvasJSON(current.settings["pr_feedback"], desired.With["pr_feedback"]) || !equalCanvasJSON(current.settings["template"], desired.With["template"]) {
		return prepared.Nodes, prepared.Edges, nil
	}
	return copyFactoryConfigurationBuiltIn(current.version, current.defaults, current.templateID, desired.With)
}

func configurationHasVCSListener(current *factoryConfigurationAutomationTarget) bool {
	for _, node := range current.version.Nodes {
		if node.Type == models.NodeTypeTrigger && (strings.HasPrefix(node.ComponentName(), "github.") || strings.HasPrefix(node.ComponentName(), "bitbucket.")) {
			return true
		}
	}
	return false
}

func factoryConfigurationBindingChanged(current *factoryConfigurationAutomationTarget, desired factoryConfigurationAutomation) bool {
	if desired.prepared == nil {
		return false
	}
	// Existing workspace webhook bindings are fixed, so exporting their selected
	// webhook does not introduce a new binding intent.
	if current.templateID == intakeTemplateID(models.FactoryIntakeSourceWebhook) {
		return current.binding != desired.prepared.Binding
	}
	return current.binding != desired.prepared.Binding || !equalCanvasJSON(current.explicitBinding, desired.Binding)
}

func factoryConfigurationReviewSourceChanged(current *factoryConfigurationAutomationTarget, desired factoryConfigurationAutomation) bool {
	if desired.prepared == nil || current.templateID != reviewRequestTemplateID {
		return false
	}
	previous := findIntakeNode(current.version.Nodes, reviewRequestTriggerNodeID)
	requested := findIntakeNode(desired.prepared.Nodes, reviewRequestTriggerNodeID)
	return previous != nil && requested != nil && previous.Configuration["app"] != requested.Configuration["app"]
}

func mergeConfigurationBindingIntent(resolved, explicit *factoryConfigurationBinding) *factoryConfigurationBinding {
	binding := factoryConfigurationBinding{}
	if resolved != nil {
		binding = *resolved
	}
	if explicit.IntegrationID != "" {
		binding.IntegrationID = explicit.IntegrationID
	}
	if explicit.ResourceID != "" {
		binding.ResourceID = explicit.ResourceID
	}
	if explicit.TeamID != "" {
		binding.TeamID = explicit.TeamID
	}
	return &binding
}

func configurationTokenNodes(nodes []models.Node) []models.Node {
	snapshot := append([]models.Node(nil), nodes...)
	for i := range snapshot {
		configuration := make(map[string]any, len(snapshot[i].Configuration))
		for key, value := range snapshot[i].Configuration {
			if key != "credentials" {
				configuration[key] = value
			}
		}
		snapshot[i].Configuration = configuration
	}
	return snapshot
}

func replaceConfigurationCanvasReferences(nodes []models.Node, replacements map[string]string) ([]models.Node, error) {
	encoded, err := json.Marshal(nodes)
	if err != nil {
		return nil, err
	}
	var result []models.Node
	if err := json.Unmarshal(encoded, &result); err != nil {
		return nil, err
	}
	for i := range result {
		node := &result[i]
		app, _ := node.Configuration["app"].(string)
		if id, exists := replacements[app]; exists {
			node.Configuration["app"] = id
		}
	}
	return result, nil
}

func configurationShouldSaveOwnership(current *factoryConfigurationAutomationTarget, desired factoryConfigurationAutomation) bool {
	return current == nil || current.owned || desired.Binding != nil || current.id != desired.ID
}

func configurationNeedsStableIdentity(stage factoryConfigurationStage, desired factoryConfigurationAutomation, current *factoryConfigurationAutomationTarget) bool {
	if current.owned {
		return false
	}
	if current.id != configurationAutomationKey(current) {
		return true
	}
	count := 0
	for _, entry := range stage.Automations {
		if entry.Uses == desired.Uses {
			count++
		}
	}
	return count > 1
}

func saveFactoryConfigurationOwnership(tx *gorm.DB, target *factoryConfigurationTarget, stage *factoryConfigurationStageTarget, desired factoryConfigurationAutomation, canvasID uuid.UUID) error {
	binding := datatypes.JSONMap{}
	if desired.Binding != nil {
		encoded, err := json.Marshal(desired.Binding)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(encoded, &binding); err != nil {
			return err
		}
	}
	owner := models.FactoryConfigurationAutomation{OrganizationID: target.factory.OrganizationID, FactoryID: target.factory.ID, LineID: target.line.ID, StageKey: stage.key, ManifestID: desired.ID, CanvasID: canvasID, Binding: binding}
	return owner.Save(tx)
}

func publishFactoryConfigurationBuiltIn(ctx context.Context, tx *gorm.DB, deps IntakeDependencies, target *factoryConfigurationTarget, stage *factoryConfigurationStageTarget, desired factoryConfigurationAutomation, current *factoryConfigurationAutomationTarget, canvasID, actor uuid.UUID, references map[string]string) (func(), error) {
	prepared := desired.prepared
	nodes, edges, err := configurationBuiltInGraph(current, desired)
	if err != nil {
		return nil, err
	}
	nodes, edges, err = replaceConfigurationIntakeTriggerForBinding(tx, current, prepared, nodes, edges)
	if err != nil {
		return nil, err
	}
	replacements := map[string]string{factoryCanvasIDPlaceholder: canvasID.String()}
	for key, value := range references {
		replacements[key] = value
	}
	nodes, err = replaceConfigurationCanvasReferences(nodes, replacements)
	if err != nil {
		return nil, err
	}
	var canvas *models.Canvas
	if current == nil {
		// Setup runs only after all canvas rows exist, including dependency sources.
		canvas, err = models.FindCanvasInTransaction(tx, target.factory.OrganizationID, canvasID)
	} else {
		canvas = current.canvas
	}
	if err != nil {
		return nil, err
	}
	// Setup resolves workspace expressions through the saved binding intent.
	if configurationShouldSaveOwnership(current, desired) {
		if err := saveFactoryConfigurationOwnership(tx, target, stage, desired, canvas.ID); err != nil {
			return nil, err
		}
	}
	if err := canvases.PublishGeneratedCanvasNodes(ctx, tx, canvas, actor, "Apply workspace built-in automation", nodes, edges, intakePublisherOptions(deps, canvas.OrganizationID)); err != nil {
		return nil, err
	}
	if err := canvases.UpdateCanvasInTransaction(tx, canvas, &desired.Name, nil); err != nil {
		return nil, err
	}
	live, err := models.FindLiveCanvasVersionInTransaction(tx, canvas.ID)
	if err != nil {
		return nil, err
	}
	canvas.LiveVersionID = &live.ID
	for _, node := range live.Nodes {
		if node.ErrorMessage == nil {
			continue
		}
		if current != nil {
			previous := findIntakeNode(current.version.Nodes, node.ID)
			// An unrelated settings change can retain an existing unsupported node.
			if previous != nil && previous.ErrorMessage != nil && *previous.ErrorMessage == *node.ErrorMessage && equalCanvasJSON(previous.Configuration, node.Configuration) && equalCanvasJSON(previous.Ref, node.Ref) {
				continue
			}
		}
		return nil, grpcerrors.FailedPrecondition(nil, fmt.Sprintf("Cannot set up %q, node %q: %s", desired.Name, node.Name, *node.ErrorMessage))
	}
	if current == nil && prepared.IntakeSource != "" {
		intake, err := target.factory.CreateIntake(tx, canvas.ID, prepared.IntakeSource)
		if err != nil {
			return nil, err
		}
		if err := intake.SkipInitialImport(tx); err != nil {
			return nil, err
		}
		if prepared.Paused {
			if err := intake.SetPaused(tx, true); err != nil {
				return nil, err
			}
		}
	}
	metadata := factoryConfigurationAutomationMetadata{}
	if current != nil {
		metadata = current.metadata.forFactory(target.factory)
	}
	if metadata.intake != nil && metadata.intake.Paused() != prepared.Paused {
		if err := metadata.intake.SetPaused(tx, prepared.Paused); err != nil {
			return nil, err
		}
	}
	handler := metadata.handler
	if current == nil && prepared.PRFeedbackSource != "" {
		handler, err = target.factory.CreatePRFeedbackHandler(tx, canvas.ID, models.FactoryPRFeedbackHandlerSubjectGitHubPullRequest, prepared.PRFeedbackSource)
		if err != nil {
			return nil, err
		}
	}
	if handler != nil && prepared.PRFeedbackSource == models.FactoryPRFeedbackHandlerSourcePullRequestChecks && (handler.MaximumAttempts == nil || *handler.MaximumAttempts != prepared.MaximumAttempts) {
		if err := handler.SetMaximumAttempts(tx, prepared.MaximumAttempts); err != nil {
			return nil, err
		}
	}
	if current == nil {
		if template, exists := lookupFactoryAppTemplate(prepared.TemplateID, factoryVCSProviderOrDefault(target.factory)); exists {
			content, err := materializeFactoryConsole(template, canvas.ID.String(), desired.Name)
			if err != nil {
				return nil, err
			}
			console, err := yaml.ConsoleFromYML([]byte(content))
			if err != nil {
				return nil, err
			}
			if _, err := models.UpsertCanvasVersionConsoleInTransaction(tx, canvas.ID, console.Panels(), console.Layout()); err != nil {
				return nil, err
			}
		}
	}
	return func() { canvases.PublishCanvasUpdated(canvas) }, nil
}
