package factories

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"

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
		if spec, ok := specs[canvas.ID]; ok && models.FactoryAppTemplateID(spec.Nodes) == models.FactoryAppTemplateCustomAutomationID {
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
	return importAutomationCanvas(tx, factory, canvas, version)
}

func canvasLegacyAutomation(tx *gorm.DB, canvas *models.Canvas) (automations.LegacyAutomation, *models.CanvasVersion, error) {
	if canvas.FactoryID == nil || canvas.ColumnKey == nil || !customAutomationColumnAllowed(*canvas.ColumnKey) {
		return automations.LegacyAutomation{}, nil, gorm.ErrRecordNotFound
	}
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
	if err != nil {
		return automations.LegacyAutomation{}, nil, err
	}
	if models.FactoryAppTemplateID(version.Nodes) != models.FactoryAppTemplateCustomAutomationID {
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
	saved, err := serializeCustomAutomationVersion(factory, canvas, version)
	if err != nil {
		return "", unrepresentableAutomationCanvas()
	}
	settings := saved.GetSettings()
	definition := automationspec.Definition{
		Name: canvas.Name, Icon: settings.GetIcon(), TimeoutSeconds: int(settings.GetExecutionTimeoutSeconds()),
		RunsOnTask: settings.RunsOnTask,
	}
	definition.Triggers, err = legacyAutomationTriggers(*canvas.ColumnKey, version.Nodes)
	if err != nil {
		return "", unrepresentableAutomationCanvas()
	}
	savedAgent := settings.GetAgent()
	if savedAgent.GetComponent() != models.SuperPlaneRunnerComponent || savedAgent.GetCredentialSource() != "" {
		return "", unrepresentableAutomationCanvas()
	}
	definition.Agent = &automationspec.Agent{Harness: automationspec.SuperPlane, Model: savedAgent.GetModel()}
	for _, step := range settings.GetSteps() {
		item := automationspec.Step{
			Name: step.GetName(), Secrets: step.GetSecrets(),
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

	// Rebuild the supported graph and compare all execution settings and edges.
	// Ignore presentation and metadata written by trigger setup.
	agent := &intakeAgent{Component: models.SuperPlaneRunnerComponent, Model: definition.Agent.Model}
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
	preserveCustomAutomationScheduleIdentities(version.Nodes, expected)
	if !equalAutomationExecutionGraph(version, expected) {
		return "", unrepresentableAutomationCanvas()
	}
	return source, nil
}

func legacyAutomationTriggers(stage string, nodes []models.Node) ([]automationspec.Trigger, error) {
	decoded, err := decodeCustomAutomationTriggers(nodes)
	if err != nil {
		return nil, err
	}
	var result []automationspec.Trigger
	count := 0
	for _, trigger := range decoded {
		if trigger.schedule != nil {
			count++
		}
	}
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
				ID: customAutomationScheduleLogicalID(trigger.node, count), Use: automationspec.Scheduled,
				Config: &automationspec.Schedule{Cron: trigger.schedule.GetCronExpression(), UTCOffset: trigger.schedule.GetUtcOffset()},
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
	for _, node := range live.Nodes {
		nodes[node.ID] = executionNode(node)
	}
	for _, node := range expected.Spec.Nodes {
		existing, ok := nodes[node.ID]
		if !ok || !equalCanvasJSON(existing, executionNode(node.Model())) {
			return false
		}
	}
	edges := make(map[models.Edge]int, len(live.Edges))
	for _, edge := range live.Edges {
		edges[edge]++
	}
	for _, edge := range expected.Spec.Edges {
		key := edge.Model()
		if edges[key] == 0 {
			return false
		}
		edges[key]--
	}
	return true
}

func executionNode(node models.Node) models.Node {
	node.Position = models.Position{}
	node.IsCollapsed = false
	node.ErrorMessage = nil
	node.WarningMessage = nil

	// An explicit false uses the same runner behavior as an omitted option.
	node.Configuration = maps.Clone(node.Configuration)
	for _, field := range []string{runner.SkipSetupCommandsField, runner.SkipRepositoryCheckoutField} {
		if node.Configuration[field] == false {
			delete(node.Configuration, field)
		}
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
		delete(template, customAutomationScheduleLogicalIDKey)
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

func unrepresentableAutomationCanvas() error {
	return grpcerrors.FailedPrecondition(nil, "This automation has canvas behavior that cannot be represented in YAML. Open the developer canvas to inspect it.")
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
