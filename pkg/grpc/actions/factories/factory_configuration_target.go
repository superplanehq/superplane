package factories

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

type factoryConfigurationTarget struct {
	reads      *factoryAutomationReadInputs
	validation *factoryConfigurationValidationInputs
	factory    *models.Factory
	line       *models.FactoryLine
	stages     []*factoryConfigurationStageTarget
}

type factoryConfigurationStageTarget struct {
	stage       models.FactoryLineStage
	key         string
	run         string
	automations []*factoryConfigurationAutomationTarget
}

type factoryConfigurationAutomationMetadata struct {
	source  *models.AutomationSource
	intake  *models.FactoryIntake
	handler *models.FactoryPRFeedbackHandler
}

// Rendering uses workspace-owned attachments. Removal also checks the raw
// canvas attachments, including an inconsistent owner, before it deletes them.
func (m factoryConfigurationAutomationMetadata) forFactory(factory *models.Factory) factoryConfigurationAutomationMetadata {
	if m.intake != nil && (m.intake.OrganizationID != factory.OrganizationID || m.intake.FactoryID != factory.ID) {
		m.intake = nil
	}
	if m.handler != nil && (m.handler.OrganizationID != factory.OrganizationID || m.handler.FactoryID != factory.ID) {
		m.handler = nil
	}
	return m
}

func (m factoryConfigurationAutomationMetadata) materializeDefaults(tx *gorm.DB, factory *models.Factory, canvas *models.Canvas, version *models.CanvasVersion) (*materializedFactoryTemplate, error) {
	if m.intake != nil {
		return materializeIntakeDefaults(tx, canvas, version, m.intake, nil)
	}
	if m.handler != nil {
		return materializePRFeedbackDefaults(tx, factory, canvas, version, m.handler, nil)
	}
	return materializeFactoryAppGraphDefaults(tx, factory, canvas, version, nil)
}

type factoryConfigurationAutomationTarget struct {
	metadata        factoryConfigurationAutomationMetadata
	canvas          *models.Canvas
	version         *models.CanvasVersion
	templateID      string
	placement       string
	id              string
	uses            string
	settings        map[string]any
	defaults        *models.CanvasVersion
	defaultSettings map[string]any
	source          string
	sourceID        uuid.UUID
}

func requireFactoryAsCode(tx *gorm.DB, orgID uuid.UUID) error {
	enabled, err := models.OrganizationHasExperimentalFeatures(tx, orgID, features.FeatureFactories, features.FeatureFactoryAsCode)
	if err != nil {
		return err
	}
	if !enabled {
		return grpcerrors.FailedPrecondition(nil, "Factory-as-Code is not enabled for this organization.")
	}
	return nil
}

func loadFactoryConfigurationTarget(tx *gorm.DB, orgID uuid.UUID, factoryRef, lineRef string) (*factoryConfigurationTarget, error) {
	if err := requireFactoryAsCode(tx, orgID); err != nil {
		return nil, err
	}
	factory, err := findFactory(tx, orgID, factoryRef)
	if err != nil {
		return nil, err
	}
	lines, err := factory.ListLines(tx)
	if err != nil {
		return nil, err
	}
	lineID, err := resolveConfigurationLineID(factory, lines, lineRef)
	if err != nil {
		return nil, err
	}
	var line *models.FactoryLine
	for i := range lines {
		if lines[i].ID == lineID {
			line = &lines[i]
			break
		}
	}
	if line == nil {
		return nil, gorm.ErrRecordNotFound
	}
	canvases, err := factory.ListCanvases(tx)
	if err != nil {
		return nil, err
	}
	return inspectFactoryConfigurationTarget(tx, factory, line, canvases, lines)
}

func lockFactoryConfigurationTarget(tx *gorm.DB, orgID uuid.UUID, factoryRef, lineRef string) (*factoryConfigurationTarget, error) {
	factory, err := findFactory(tx, orgID, factoryRef)
	if err != nil {
		return nil, err
	}
	// Lock the workspace before its line and canvases. The row lock also
	// keeps new workspace attachments from appearing during application.
	if err := factory.LockForUpdate(tx); err != nil {
		return nil, err
	}
	lines, err := factory.ListLines(tx)
	if err != nil {
		return nil, err
	}
	lineID, err := resolveConfigurationLineID(factory, lines, lineRef)
	if err != nil {
		return nil, err
	}
	line, err := factory.FindLineForUpdate(tx, lineID)
	if err != nil {
		return nil, err
	}
	canvases, err := factory.ListCanvases(tx)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(canvases, func(a, b models.Canvas) int { return strings.Compare(a.ID.String(), b.ID.String()) })
	for i := range canvases {
		canvas, err := models.LockCanvasForUpdate(tx, orgID, canvases[i].ID)
		if err != nil {
			return nil, err
		}
		canvases[i] = *canvas
	}
	if err := requireFactoryAsCode(tx, orgID); err != nil {
		return nil, err
	}
	return inspectFactoryConfigurationTarget(tx, factory, line, canvases, lines)
}

func resolveConfigurationLineID(factory *models.Factory, lines []models.FactoryLine, ref string) (uuid.UUID, error) {
	if strings.TrimSpace(ref) != "" {
		return parseLineID(ref)
	}
	provisioned := factory.OnboardingConfigValue().ProvisionedLineID
	for _, line := range lines {
		if line.ID.String() == provisioned {
			return line.ID, nil
		}
	}
	if len(lines) != 1 {
		return uuid.Nil, unsupportedConfigurationTarget("Select a workspace with one line or a valid onboarding line. The destination is ambiguous.")
	}
	return lines[0].ID, nil
}

func inspectFactoryConfigurationTarget(tx *gorm.DB, factory *models.Factory, line *models.FactoryLine, canvases []models.Canvas, lines []models.FactoryLine) (*factoryConfigurationTarget, error) {
	if !factory.IsOnboardingComplete() {
		return nil, unsupportedConfigurationTarget("Select an onboarded workspace.")
	}
	stages := line.StagesValue()
	if len(stages) == 0 {
		var err error
		stages, err = line.ReconcileStages(tx, line.Steps)
		if err != nil {
			return nil, err
		}
	}
	if err := validateFactoryLineSystemStages(stages); err != nil {
		return nil, err
	}
	target := &factoryConfigurationTarget{factory: factory, line: line}
	byKey := make(map[string]*factoryConfigurationStageTarget, len(stages))
	for _, stage := range stages {
		key := stage.Key
		if key == "phase-0" {
			key = "implementation"
		}
		entry := &factoryConfigurationStageTarget{stage: stage, key: key}
		target.stages = append(target.stages, entry)
		byKey[stage.Key] = entry
	}
	placements := map[uuid.UUID]string{}
	metadata := map[uuid.UUID]factoryConfigurationAutomationMetadata{}
	canvasIDs := make([]uuid.UUID, 0, len(canvases))
	for _, canvas := range canvases {
		canvasIDs = append(canvasIDs, canvas.ID)
	}
	intakes, err := models.FindFactoryIntakesByCanvasIDs(tx, canvasIDs)
	if err != nil {
		return nil, err
	}
	for _, intake := range intakes {
		metadata[intake.CanvasID] = factoryConfigurationAutomationMetadata{intake: &intake}
		if intake.OrganizationID == factory.OrganizationID && intake.FactoryID == factory.ID {
			placements[intake.CanvasID] = models.CanvasColumnKeyBacklog
		}
	}
	handlers, err := models.FindFactoryPRFeedbackHandlersByCanvasIDs(tx, canvasIDs)
	if err != nil {
		return nil, err
	}
	for _, handler := range handlers {
		entry := metadata[handler.CanvasID]
		entry.handler = &handler
		metadata[handler.CanvasID] = entry
		if handler.OrganizationID == factory.OrganizationID && handler.FactoryID == factory.ID {
			placements[handler.CanvasID] = models.CanvasColumnKeyVerify
		}
	}
	slices.SortFunc(canvases, func(a, b models.Canvas) int {
		if name := strings.Compare(a.Name, b.Name); name != 0 {
			return name
		}
		return strings.Compare(a.ID.String(), b.ID.String())
	})
	byCanvas := make(map[uuid.UUID]*models.Canvas, len(canvases))
	artifactIDs := make([]string, 0, len(canvases))
	for i := range canvases {
		byCanvas[canvases[i].ID] = &canvases[i]
		artifactIDs = append(artifactIDs, canvases[i].ID.String())
	}
	sources, err := models.FindAutomationSourcesByArtifacts(tx, factory.OrganizationID, artifactIDs)
	if err != nil {
		return nil, err
	}
	for i := range sources {
		source := &sources[i]
		canvasID, err := uuid.Parse(source.ArtifactID)
		if err != nil {
			return nil, err
		}
		entry := metadata[canvasID]
		entry.source = source
		metadata[canvasID] = entry
	}
	versions, err := models.FindLiveCanvasVersions(tx, canvases)
	if err != nil {
		return nil, err
	}
	reads, err := newFactoryAutomationReadInputs(tx, factory, canvases, versions, lines)
	if err != nil {
		return nil, err
	}
	target.reads = reads
	tx = withFactoryAutomationReadInputs(tx, reads)
	for _, stage := range target.stages {
		if stage.stage.Kind != models.FactoryLineStagePhase || stage.stage.StepIndex == nil {
			continue
		}
		index := stage.stage.StepIndex
		if *index < 0 || *index >= len(line.Steps) {
			return nil, unsupportedConfigurationTarget("A phase has an invalid runner binding.")
		}
		step := line.Steps[*index]
		canvas := byCanvas[step.AppID]
		if step.Type != models.FactoryLineStepTypeRunApp || canvas == nil {
			return nil, unsupportedConfigurationTarget("A phase must have a workspace automation runner.")
		}
		automation, err := inspectConfigurationAutomation(tx, factory, canvas, versions[canvas.ID], metadata[canvas.ID], "phase-"+strconv.Itoa(*index))
		if err != nil {
			return nil, err
		}
		trigger := findIntakeNode(automation.version.Nodes, step.Entrypoint)
		if trigger == nil || trigger.Type != models.NodeTypeTrigger || trigger.ComponentName() != "onRun" {
			return nil, unsupportedConfigurationTarget(fmt.Sprintf("Automation %q must use its On Run trigger.", canvas.Name))
		}
		stage.automations = append(stage.automations, automation)
	}
	for i := range canvases {
		canvas := &canvases[i]
		placement := placements[canvas.ID]
		if placement == "" && canvas.ColumnKey != nil {
			placement = *canvas.ColumnKey
		}
		if placement == "implement" {
			placement = "phase-0"
		}
		if placement == "" && canvas.LiveVersionID != nil {
			version := versions[canvas.ID]
			if version == nil {
				return nil, gorm.ErrRecordNotFound
			}
			entry := metadata[canvas.ID].forFactory(factory)
			template, builtIn := builtInAutomationTemplateID(version, entry.intake, entry.handler)
			if builtIn {
				switch template {
				case models.FactoryAppTemplateBacklogID, scheduledIntakeTemplateID, "issue-intake":
					placement = "backlog"
				case "pr-closure":
					placement = "done"
				case "risk-score", "visual-evidence", reviewRequestTemplateID, prFeedbackChecksTemplateID, prFeedbackDiscussionTemplateID:
					placement = "verify"
				}
			}
		}
		stage := byKey[placement]
		if stage == nil {
			continue
		}
		// A phase runner can also execute in an attached custom stage. Keep
		// both bindings, while listing the runner only once in its own phase.
		if slices.ContainsFunc(stage.automations, func(automation *factoryConfigurationAutomationTarget) bool {
			return automation.canvas.ID == canvas.ID
		}) {
			continue
		}
		compilerPlacement := placement
		if placement == "phase-0" {
			compilerPlacement = "implement"
		}
		automation, err := inspectConfigurationAutomation(tx, factory, canvas, versions[canvas.ID], metadata[canvas.ID], compilerPlacement)
		if err != nil {
			return nil, err
		}
		stage.automations = append(stage.automations, automation)
	}
	customPaths := map[uuid.UUID]string{}
	usedPaths := map[string]bool{}
	for _, stage := range target.stages {
		counts := map[string]int{}
		for _, automation := range stage.automations {
			if automation.templateID == "" {
				reference, exists := customPaths[automation.canvas.ID]
				if !exists {
					base := "automations/" + models.Slugify(automation.canvas.Name)
					reference = base + ".yaml"
					for suffix := 2; usedPaths[reference]; suffix++ {
						reference = base + "-" + strconv.Itoa(suffix) + ".yaml"
					}
					customPaths[automation.canvas.ID] = reference
					usedPaths[reference] = true
				}
				automation.uses = reference
			}
			counts[configurationAutomationKey(automation)]++
		}
		for i, automation := range stage.automations {
			id := configurationAutomationKey(automation)
			if automation.templateID != "" && counts[id] > 1 {
				id += "-" + configurationCanvasKey(automation.canvas.ID)
			}
			automation.id = id
			if i == 0 && stage.stage.StepIndex != nil {
				stage.run = id
			}
		}
	}
	return target, nil
}

func inspectConfigurationAutomation(tx *gorm.DB, factory *models.Factory, canvas *models.Canvas, version *models.CanvasVersion, metadata factoryConfigurationAutomationMetadata, placement string) (*factoryConfigurationAutomationTarget, error) {
	if canvas.LiveVersionID == nil {
		return nil, unsupportedConfigurationTarget(fmt.Sprintf("Automation %q must be published before export.", canvas.Name))
	}
	if version == nil {
		return nil, gorm.ErrRecordNotFound
	}
	automation := &factoryConfigurationAutomationTarget{canvas: canvas, version: version, placement: placement, metadata: metadata}
	metadata = metadata.forFactory(factory)
	template, builtIn := builtInAutomationTemplateID(version, metadata.intake, metadata.handler)
	var err error
	if builtIn {
		alias, ok := factoryConfigurationBuiltInAlias(template)
		if !ok {
			return nil, unsupportedConfigurationTarget(fmt.Sprintf("Automation %q uses an unsupported built-in template %q.", canvas.Name, template))
		}
		automation.templateID, automation.uses = template, alias
		defaults, defaultErr := metadata.materializeDefaults(tx, factory, canvas, version)
		if defaultErr != nil {
			return nil, defaultErr
		}
		document, defaultErr := yaml.CanvasFromYAML([]byte(defaults.canvasYAML))
		if defaultErr != nil {
			return nil, defaultErr
		}
		// Reset preserves installed models; YAML defaults use the workspace model.
		agent := resolveIntakeAgent(tx, factory)
		for i := range document.Spec.Nodes {
			node := &document.Spec.Nodes[i]
			if agent == nil || !isFactoryAgentHarness(node.Component) {
				continue
			}
			if model := agent.model(); model != "" {
				node.Configuration["model"] = model
				continue
			}
			delete(node.Configuration, "model")
		}
		automation.defaults = &models.CanvasVersion{Nodes: document.Nodes()}
		automation.defaultSettings, err = exportFactoryConfigurationBuiltIn(automation.defaults, template)
		if err != nil {
			return nil, err
		}
		automation.settings, err = exportFactoryConfigurationBuiltIn(version, template)
	} else {
		automation.source, err = renderFactoryConfigurationCustom(tx, factory, canvas, version, placement, metadata.source)
	}
	if err != nil {
		message := err.Error()
		if !builtIn {
			_, safeMessage, ok := grpcerrors.HandlerStatus(err)
			if !ok {
				return nil, err
			}
			message = safeMessage
		}
		return nil, unsupportedConfigurationTarget(fmt.Sprintf("Cannot export automation %q: %s", canvas.Name, message))
	}
	if !builtIn && metadata.source != nil && metadata.source.FactoryID == factory.ID {
		automation.sourceID = metadata.source.ID
	}

	return automation, nil
}

func configurationCanvasKey(canvasID uuid.UUID) string {
	digest := sha256.Sum256([]byte(canvasID.String()))
	return hex.EncodeToString(digest[:8])
}

func configurationAutomationKey(automation *factoryConfigurationAutomationTarget) string {
	if automation.templateID == "" {
		// Keep manifest identity stable when the display name and filename change.
		return configurationCanvasKey(automation.canvas.ID)
	}
	return strings.ReplaceAll(strings.TrimPrefix(automation.uses, "@superplane/"), "/", "-")
}

func unsupportedConfigurationTarget(message string) error {
	return grpcerrors.FailedPrecondition(nil, message)
}

func factoryConfigurationToken(target *factoryConfigurationTarget, manifest *factoryConfigurationManifest) (string, error) {
	type revision struct {
		Canvas    uuid.UUID
		Version   uuid.UUID
		Placement string
	}
	revisions := []revision{}
	for _, stage := range target.stages {
		for _, automation := range stage.automations {
			revisions = append(revisions, revision{automation.canvas.ID, automation.version.ID, automation.placement})
		}
	}
	state := struct {
		Organization uuid.UUID
		Factory      uuid.UUID
		Line         uuid.UUID
		Steps        []models.FactoryLineStep
		Stages       []models.FactoryLineStage
		Live         factoryConfigurationManifest
		Revisions    []revision
		Desired      *factoryConfigurationManifest
	}{target.factory.OrganizationID, target.factory.ID, target.line.ID, target.line.Steps, target.line.Stages, exportFactoryConfigurationManifest(target), revisions, manifest}
	encoded, err := json.Marshal(state)
	if err != nil {
		return "", fmt.Errorf("encode configuration base state: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}
