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
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type factoryConfigurationTarget struct {
	reads        *factoryAutomationReadInputs
	validation   *factoryConfigurationValidationInputs
	factory      *models.Factory
	line         *models.FactoryLine
	stages       []*factoryConfigurationStageTarget
	ownership    []models.FactoryConfigurationAutomation
	repositories *factoryConfigurationRepositories
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
	binding         factoryConfigurationBinding
	exportedBinding *factoryConfigurationBinding
	explicitBinding *factoryConfigurationBinding
	owned           bool
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
	if _, err := factory.ListContextRepos(tx.Clauses(clause.Locking{Strength: "UPDATE"})); err != nil {
		return nil, err
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
	owners, err := models.ListFactoryConfigurationAutomations(tx, factory.OrganizationID, factory.ID, uuid.Nil)
	if err != nil {
		return nil, err
	}
	target := &factoryConfigurationTarget{factory: factory, line: line, ownership: owners}
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
	if err := loadFactoryConfigurationRepositories(tx, target); err != nil {
		return nil, err
	}
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
			for _, owner := range target.ownership {
				if owner.CanvasID != automation.canvas.ID {
					continue
				}
				if len(owner.Binding) > 0 {
					binding := &factoryConfigurationBinding{}
					encoded, err := json.Marshal(owner.Binding)
					if err != nil {
						return nil, err
					}
					if err := json.Unmarshal(encoded, binding); err != nil {
						return nil, err
					}
					automation.explicitBinding = binding
					automation.exportedBinding = mergeConfigurationBindingIntent(automation.exportedBinding, binding)
					if binding.IntegrationID != "" {
						automation.binding.IntegrationID = binding.IntegrationID
					}
					if binding.ResourceID != "" {
						automation.binding.ResourceID = binding.ResourceID
					}
					if binding.TeamID != "" {
						automation.binding.TeamID = binding.TeamID
					}
				}
				if owner.LineID == line.ID && owner.StageKey == stage.key {
					id = owner.ManifestID
					automation.owned = true
				}
			}

			if strings.HasPrefix(automation.templateID, "intake:") && automation.binding.IntegrationID != "" {
				binding := factoryConfigurationBinding{}
				if automation.exportedBinding != nil {
					binding = *automation.exportedBinding
				}
				binding.IntegrationID = automation.binding.IntegrationID
				automation.exportedBinding = &binding
			}
			automation.exportedBinding = portableConfigurationBinding(reads.integrations, automation.exportedBinding)
			automation.id = id
			if i == 0 && stage.stage.StepIndex != nil {
				stage.run = id
			}
		}
	}
	if err := exportFactoryConfigurationDependencies(target); err != nil {
		return nil, err
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
		automation.binding = configurationBuiltInDefaultBinding(factory, template)
		selected := factoryConfigurationBindingFromGraph(version, template)
		if selected.IntegrationID != "" {
			automation.binding.IntegrationID = selected.IntegrationID
		}
		if selected.ResourceID != "" {
			automation.binding.ResourceID = selected.ResourceID
		}
		if selected.TeamID != "" {
			automation.binding.TeamID = selected.TeamID
		}
		automation.exportedBinding = configurationBindingOverrides(automation.binding, configurationBuiltInDefaultBinding(factory, template))
		automation.defaults, err = buildFreshFactoryConfigurationBuiltInDefaults(tx, factory, template, automation.binding)
		if err != nil {
			return nil, err
		}

		automation.defaultSettings, err = exportFactoryConfigurationBuiltIn(automation.defaults, template)
		if err != nil {
			return nil, err
		}
		automation.settings, err = exportFactoryConfigurationBuiltIn(version, template)
		if err == nil {
			supplementFactoryConfigurationRecordSettings(tx, automation, metadata)
		}
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
		Defaults  map[string]any
	}
	revisions := []revision{}
	for _, stage := range target.stages {
		for _, automation := range stage.automations {
			revisions = append(revisions, revision{automation.canvas.ID, automation.version.ID, automation.placement, automation.defaultSettings})
		}
	}
	dependencies := []any{}
	for _, stage := range manifest.Stages {
		for _, automation := range stage.Automations {
			if automation.prepared != nil {
				console := ""
				if template, exists := lookupFactoryAppTemplate(automation.prepared.TemplateID, factoryVCSProviderOrDefault(target.factory)); exists {
					var err error
					console, err = materializeFactoryConsole(template, factoryCanvasIDPlaceholder, automation.Name)
					if err != nil {
						return "", err
					}
				}
				dependencies = append(dependencies, struct {
					State   any
					Nodes   []models.Node
					Edges   []models.Edge
					Console string
				}{automation.prepared.State, configurationTokenNodes(automation.prepared.Nodes), automation.prepared.Edges, console})
			}
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
		Dependencies []any
	}{target.factory.OrganizationID, target.factory.ID, target.line.ID, target.line.Steps, target.line.Stages, exportFactoryConfigurationManifest(target), revisions, manifest, dependencies}
	encoded, err := json.Marshal(state)
	if err != nil {
		return "", fmt.Errorf("encode configuration base state: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}
