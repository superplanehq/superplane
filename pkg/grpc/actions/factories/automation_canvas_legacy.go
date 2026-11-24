package factories

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/automations"
	"github.com/superplanehq/superplane/pkg/automationspec"
	"github.com/superplanehq/superplane/pkg/components/runner"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

func (b CanvasAutomationBackend) ListLegacy(_ context.Context, tx *gorm.DB, placement automations.Placement) ([]automations.LegacyAutomation, error) {
	factory, err := models.FindFactory(tx, placement.OrganizationID, placement.FactoryID)
	if err != nil {
		return nil, err
	}
	canvases, err := factory.ListCanvases(tx)
	if err != nil {
		return nil, err
	}
	var ids []uuid.UUID
	for _, canvas := range canvases {
		if canvas.LiveVersionID != nil && canvas.ColumnKey != nil && customAutomationColumnAllowed(*canvas.ColumnKey) {
			ids = append(ids, canvas.ID)
		}
	}
	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(tx, ids)
	if err != nil {
		return nil, err
	}
	result := make([]automations.LegacyAutomation, 0)
	for _, canvas := range canvases {
		if spec, ok := specs[canvas.ID]; ok && (models.FactoryAppTemplateID(spec.Nodes) == models.FactoryAppTemplateCustomAutomationID || legacyCustomStageGraph(&canvas, &models.CanvasVersion{Nodes: spec.Nodes, Edges: spec.Edges})) {
			result = append(result, legacyCanvasMetadata(&canvas))
		}
	}
	return result, nil
}

func (b CanvasAutomationBackend) FindLegacy(_ context.Context, tx *gorm.DB, organizationID, id uuid.UUID) (automations.LegacyAutomation, error) {
	canvas, err := models.FindCanvasInTransaction(tx, organizationID, id)
	if err != nil {
		return automations.LegacyAutomation{}, err
	}
	automation, _, err := canvasLegacyAutomation(tx, canvas)
	return automation, err
}

func (b CanvasAutomationBackend) ImportLegacy(_ context.Context, tx *gorm.DB, legacy automations.LegacyAutomation) (string, error) {
	factory, err := models.FindFactory(tx, legacy.Placement.OrganizationID, legacy.Placement.FactoryID)
	if err != nil {
		return "", err
	}
	canvas, err := models.FindCanvasInTransaction(tx, legacy.Placement.OrganizationID, legacy.ID)
	if err != nil {
		return "", err
	}
	current, version, err := canvasLegacyAutomation(tx, canvas)
	if err != nil {
		return "", err
	}
	if current.Artifact != legacy.Artifact || current.Placement != legacy.Placement {
		return "", grpcerrors.FailedPrecondition(nil, "The automation changed. Reload it and try again.")
	}
	source, err := importAutomationCanvas(tx, factory, canvas, version)
	if errors.Is(err, errUnrepresentableAutomation) && legacyFormPreservesCanvas(tx, factory, canvas, version) {
		return "", grpcerrors.FailedPrecondition(&AutomationFormFallback{LiveVersionID: version.ID}, "Open the form to edit this automation.")
	}
	return source, err
}

// AutomationFormFallback identifies a version the existing form can edit losslessly.
type AutomationFormFallback struct{ LiveVersionID uuid.UUID }

func (*AutomationFormFallback) Error() string { return "automation requires the form editor" }

func legacyFormPreservesCanvas(tx *gorm.DB, factory *models.Factory, canvas *models.Canvas, version *models.CanvasVersion) bool {
	saved, err := serializeCustomAutomationVersion(factory, canvas, version)
	if err != nil || saved.Settings.RunsOnTask == nil {
		// The legacy form infers a single run context when this setting is absent.
		return false
	}
	settings, err := parseCustomAutomationSettings(tx, factory, saved.Settings)
	if err != nil {
		return false
	}
	settings.ColumnKey = *canvas.ColumnKey
	expected, err := buildCustomAutomationCanvas(tx, factory, canvas.Name, settings)
	if err != nil || preserveCustomAutomationIdentities(version, expected) != nil {
		return false
	}
	return equalAutomationExecutionGraph(version, expected)
}

func canvasLegacyAutomation(tx *gorm.DB, canvas *models.Canvas) (automations.LegacyAutomation, *models.CanvasVersion, error) {
	if canvas.FactoryID == nil || canvas.ColumnKey == nil || !customAutomationColumnAllowed(*canvas.ColumnKey) {
		return automations.LegacyAutomation{}, nil, gorm.ErrRecordNotFound
	}
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
	if err != nil {
		return automations.LegacyAutomation{}, nil, err
	}
	if models.FactoryAppTemplateID(version.Nodes) != models.FactoryAppTemplateCustomAutomationID && !legacyCustomStageGraph(canvas, version) {
		return automations.LegacyAutomation{}, nil, gorm.ErrRecordNotFound
	}
	return legacyCanvasMetadata(canvas), version, nil
}

func legacyCanvasMetadata(canvas *models.Canvas) automations.LegacyAutomation {
	return automations.LegacyAutomation{
		ID: canvas.ID, Name: canvas.Name,
		Placement: automations.Placement{
			OrganizationID: canvas.OrganizationID, FactoryID: *canvas.FactoryID, Stage: *canvas.ColumnKey,
		},
		Artifact: automations.Artifact{ID: canvas.ID.String(), Revision: canvas.LiveVersionID.String()},
	}
}

func importAutomationCanvas(tx *gorm.DB, factory *models.Factory, canvas *models.Canvas, version *models.CanvasVersion) (string, error) {
	saved, err := readCustomAutomationVersion(factory, canvas, version)
	if err != nil {
		return "", unrepresentableAutomationCanvas()
	}
	settings := saved.GetSettings()
	definition := automationspec.Definition{
		Name: canvas.Name, Icon: settings.GetIcon(), TimeoutSeconds: int(settings.GetExecutionTimeoutSeconds()),
		RunsOnTask:        settings.RunsOnTask,
		SkipSetupCommands: settings.SkipSetupCommands, SkipRepositoryCheckout: settings.SkipRepositoryCheckout,
	}
	definition.Triggers, err = legacyAutomationTriggers(*canvas.ColumnKey, version.Nodes)
	if err != nil {
		return "", unrepresentableAutomationCanvas()
	}
	hasSchedule := slices.ContainsFunc(definition.Triggers, func(trigger automationspec.Trigger) bool {
		return trigger.Use == automationspec.Scheduled || trigger.Use == automationspec.Webhook
	})
	if *canvas.ColumnKey == models.CanvasColumnKeyBacklog && ((settings.RunsOnTask != nil && !*settings.RunsOnTask) || hasSchedule) {
		stripCustomAutomationBacklogPrompt(settings.Steps)
	}
	agent := intakeAgentFromCanvasNodes(version.Nodes)
	definition.Agent, err = exportAutomationAgent(tx, factory.OrganizationID, factory.ID, agent)
	if err != nil {
		return "", err
	}
	for _, step := range settings.GetSteps() {
		item := automationspec.Step{
			Name: step.GetName(), Secrets: step.GetSecrets(), WorkingDirectory: step.GetWorkingDirectory(),
		}
		if step.GetType() == pb.FactoryCustomAutomation_Step_TYPE_BASH {
			item.Bash = step.GetCommand()
		} else {
			item.Prompt = step.GetPrompt()
		}
		definition.Steps = append(definition.Steps, item)
	}
	source, err := automationspec.Encode(definition)
	if err != nil {
		return "", err
	}
	document, err := automationspec.Parse(source)
	if err != nil {
		return "", unrepresentableAutomationCanvas()
	}
	if legacyCustomStageGraph(canvas, version) {
		// This graph has no event subscription. Keep its dispatch-owned start.
		return source, nil
	}

	// Rebuild the supported graph and compare all execution settings and edges.
	// Ignore presentation and metadata written by trigger setup.
	agentSettings, err := automationAgentSettings(tx, factory, document)
	if err != nil {
		var diagnostic *automationspec.Diagnostic
		if errors.As(err, &diagnostic) {
			return "", unrepresentableAutomationCanvas()
		}
		return "", err
	}
	agent, err = customAutomationAgent(tx, factory, agentSettings)
	if err != nil || agent == nil {
		return "", unrepresentableAutomationCanvas()
	}
	if credentials := definition.Agent.Credentials; credentials != nil && credentials.Source == automationspec.Integration {
		connection, err := models.FindIntegrationByName(tx, factory.OrganizationID, credentials.Name)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
		if connection != nil && connection.AppName == models.CustomLLMAppName && agent.Component == "runnerOpenRouter" {
			agent.LLMProvider = models.UsageProviderCustom
		}
	}
	rebuild, err := automationCanvasSettings(*canvas.ColumnKey, document, agent)
	if err != nil {
		return "", unrepresentableAutomationCanvas()
	}
	expected, err := buildCustomAutomationCanvas(tx, factory, canvas.Name, rebuild)
	if err != nil {
		return "", unrepresentableAutomationCanvas()
	}
	if err := preserveCustomAutomationIdentities(version, expected); err != nil {
		return "", unrepresentableAutomationCanvas()
	}
	if !equalAutomationExecutionGraph(version, expected) {
		return "", unrepresentableAutomationCanvas()
	}
	workspace := resolveIntakeAgent(tx, factory)
	if workspace != nil && workspace.Component == agent.Component && reflect.DeepEqual(workspace.Credentials, agent.Credentials) {
		definition.Agent.Harness = ""
		definition.Agent.Credentials = nil
		if definition.Agent.Model == "" {
			definition.Agent = nil
		}
	}
	return automationspec.Encode(definition)
}

func legacyAutomationTriggers(stage string, nodes []models.Node) ([]automationspec.Trigger, error) {
	decoded, err := decodeCustomAutomationTriggers(nodes)
	if err != nil {
		return nil, err
	}
	var result []automationspec.Trigger
	for _, trigger := range decoded {
		switch trigger.mode {
		case pb.FactoryCustomAutomation_START_MODE_MANUAL:
			result = append(result, automationspec.Trigger{Use: automationspec.Manual})
		case pb.FactoryCustomAutomation_START_MODE_AUTOMATIC:
			use := automationspec.TaskEntersStage
			if stage == models.CanvasColumnKeyVerify {
				use = automationspec.PullRequestUpdated
			}
			result = append(result, automationspec.Trigger{Use: use})
		case pb.FactoryCustomAutomation_START_MODE_ON_LEAVE:
			result = append(result, automationspec.Trigger{Use: automationspec.TaskLeavesStage})
		case pb.FactoryCustomAutomation_START_MODE_SCHEDULED:
			result = append(result, automationspec.Trigger{
				Use:    automationspec.Scheduled,
				Config: &automationspec.TriggerConfig{Cron: trigger.schedule.GetCronExpression(), UTCOffset: trigger.schedule.GetUtcOffset()},
			})
		case pb.FactoryCustomAutomation_START_MODE_WEBHOOK:
			result = append(result, automationspec.Trigger{
				Use:    automationspec.Webhook,
				Config: &automationspec.TriggerConfig{WebhookID: trigger.webhookID},
			})
		}
	}
	return result, nil
}

func equalAutomationExecutionGraph(live *models.CanvasVersion, expected *yaml.Canvas) bool {
	if len(live.Nodes) != len(expected.Spec.Nodes) || len(live.Edges) != len(expected.Spec.Edges) {
		return false
	}
	nodes := make(map[string]models.Node, len(live.Nodes))
	triggerIDs := make(map[string]string)
	for _, node := range live.Nodes {
		nodes[node.ID] = executionNode(node)
		if key := generatedTriggerKey(node); key != "" {
			triggerIDs[key] = node.ID
		}
	}
	// The builder may generate fresh trigger IDs on save. Compare the trigger
	// and its outgoing edges using its live ID without publishing the graph.
	liveIDByGeneratedID := make(map[string]string)
	for _, node := range expected.Spec.Nodes {
		candidate := node.Model()
		if key := generatedTriggerKey(candidate); key != "" {
			candidate.ID = triggerIDs[key]
			liveIDByGeneratedID[node.ID] = candidate.ID
		}
		existing, ok := nodes[candidate.ID]
		if !ok || !equalCanvasJSON(existing, executionNode(candidate)) {
			return false
		}
	}
	edges := make(map[models.Edge]int, len(live.Edges))
	for _, edge := range live.Edges {
		edges[edge]++
	}
	for _, edge := range expected.Spec.Edges {
		key := edge.Model()
		if id, ok := liveIDByGeneratedID[key.SourceID]; ok {
			key.SourceID = id
		}
		if edges[key] == 0 {
			return false
		}
		edges[key]--
	}
	return true
}

func generatedTriggerKey(node models.Node) string {
	switch node.ComponentName() {
	case "schedule":
		template, _ := node.Metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
		return fmt.Sprintf("schedule-%d", configInt(template, "triggerIndex"))
	case "github.onPullRequest", "bitbucket.onPullRequest":
		return node.ComponentName()
	default:
		return ""
	}
}

func executionNode(node models.Node) models.Node {
	node.Position = models.Position{}
	node.IsCollapsed = false
	node.ErrorMessage = nil
	node.WarningMessage = nil

	node.Configuration = maps.Clone(node.Configuration)
	if node.ComponentName() == models.SuperPlaneRunnerComponent {
		// Compare hosted key spellings using the runner's own parser.
		if hosted, err := models.ParseHostedLLMModelKey(configString(node.Configuration, "model")); err == nil && hosted.IsSet() {
			node.Configuration["model"] = models.FormatSelectableLLMModelKey(models.UsageFundingSourceHosted, hosted.Provider, hosted.Model)
		}
	}
	// An explicit false uses the same runner behavior as an omitted option.
	for _, field := range []string{runner.SkipSetupCommandsField, runner.SkipRepositoryCheckoutField} {
		if node.Configuration[field] == false {
			delete(node.Configuration, field)
		}
	}
	// A workspace switch stores a blank model for subscriptions. The form omits it.
	credentials, _ := node.Configuration["credentials"].(map[string]any)
	if node.ComponentName() == "runnerCodex" && credentials["source"] == runner.CredentialsSourceSubscription && node.Configuration["model"] == "" {
		delete(node.Configuration, "model")
	}

	node.Metadata = maps.Clone(node.Metadata)
	if node.Type == models.NodeTypeTrigger {
		delete(node.Metadata, "appSubscriptionID")
		if node.ComponentName() == "bitbucket.onPullRequest" || node.ComponentName() == "github.onPullRequest" {
			delete(node.Metadata, "repository")
		}
	}
	if node.ComponentName() == "schedule" {
		delete(node.Metadata, "nextTrigger")
		delete(node.Metadata, "referenceTime")
	}
	if template, ok := node.Metadata[models.FactoryAppTemplateMetadataKey].(map[string]any); ok {
		template = maps.Clone(template)
		delete(template, "automationTriggerId")
		if configString(template, "icon") == automationspec.DefaultIcon {
			delete(template, "icon")
		}
		node.Metadata[models.FactoryAppTemplateMetadataKey] = template
	}
	if len(node.Metadata) == 0 {
		node.Metadata = nil
	}
	return node
}

func equalCanvasJSON(a, b any) bool {
	first, firstErr := json.Marshal(a)
	second, secondErr := json.Marshal(b)
	return firstErr == nil && secondErr == nil && bytes.Equal(first, second)
}

var errUnrepresentableAutomation = errors.New("automation has unsupported canvas behavior")

func unrepresentableAutomationCanvas() error {
	return grpcerrors.FailedPrecondition(errUnrepresentableAutomation, "This automation has canvas behavior that cannot be represented in YAML. Open the developer canvas to inspect it.")
}

func refreshLegacyAutomationSource(
	tx *gorm.DB, factory *models.Factory, canvas *models.Canvas, version *models.CanvasVersion, authored *models.AutomationSource,
) error {
	if authored == nil {
		return nil
	}
	for _, node := range version.Nodes {
		if node.ErrorMessage != nil {
			return grpcerrors.FailedPrecondition(nil, "SuperPlane could not activate this automation. Check the triggers and agent configuration.")
		}
	}
	source, err := importAutomationCanvas(tx, factory, canvas, version)
	if err != nil {
		return err
	}
	authored.Name = canvas.Name
	authored.Source = source
	authored.ArtifactRevision = version.ID.String()
	authored.Revision++
	return authored.Save(tx)
}

func findLegacyAutomationSource(tx *gorm.DB, canvas *models.Canvas) (*models.AutomationSource, error) {
	source, err := models.FindAutomationSourceByArtifact(tx, canvas.OrganizationID, canvas.ID.String())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return source, err
}
