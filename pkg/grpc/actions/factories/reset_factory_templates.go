package factories

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

const resetFactoryTemplateCommitMessage = "Reset factory template defaults"

// ErrUnknownOnboardingFactoryTemplate is returned when reset is asked for a
// template that SuperPlane does not create during onboarding.
var ErrUnknownOnboardingFactoryTemplate = errors.New("unknown factory template")

var errFactoryTemplateResetStale = errors.New("canvas changed during reset")

const (
	onboardingTemplateBacklog      = models.FactoryAppTemplateBacklogID
	onboardingTemplateImplement    = "line-implementation"
	onboardingTemplatePRClosure    = "pr-closure"
	onboardingTemplateIntake       = "intake"
	onboardingTemplateIntakePrefix = "intake:"
)

type OnboardingFactoryTemplate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Count       int    `json:"count"`
}

type ResetFactoryTemplateResult struct {
	Reset    int                           `json:"reset"`
	Failures []ResetFactoryTemplateFailure `json:"failures"`
}

type ResetFactoryTemplateFailure struct {
	CanvasID string `json:"canvas_id"`
	Name     string `json:"name"`
	Error    string `json:"error"`
}

type factoryAppMatch struct {
	factory *models.Factory
	canvas  *models.Canvas
}

func onboardingFactoryTemplateCatalog() []OnboardingFactoryTemplate {
	return []OnboardingFactoryTemplate{
		{
			ID:          onboardingTemplateBacklog,
			Name:        "Backlog",
			Description: "Plan new draft tasks.",
		},
		{
			ID:          onboardingTemplateImplement,
			Name:        "Implement",
			Description: "Create a branch, implement the task, and open a pull request.",
		},
		{
			ID:          onboardingTemplatePRClosure,
			Name:        "PR Closure",
			Description: "Close the task when the attached pull request merges or closes.",
		},
		{
			ID:          onboardingTemplateIntake,
			Name:        "Intake",
			Description: "Import work from GitHub, Jira, or Linear.",
		},
	}
}

func isOnboardingFactoryTemplate(id string) bool {
	switch strings.TrimSpace(id) {
	case onboardingTemplateBacklog, onboardingTemplateImplement, onboardingTemplatePRClosure, onboardingTemplateIntake:
		return true
	default:
		return false
	}
}

// ListOnboardingFactoryTemplates counts onboarding automations on this
// installation by template.
func ListOnboardingFactoryTemplates(ctx context.Context) ([]OnboardingFactoryTemplate, error) {
	items := onboardingFactoryTemplateCatalog()
	counts, err := countOnboardingFactoryTemplates(database.DB(ctx))
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Count = counts[items[i].ID]
	}
	return items, nil
}

// ResetOnboardingFactoryTemplate publishes the current SuperPlane defaults
// onto every matching onboarding automation on this installation.
func ResetOnboardingFactoryTemplate(
	ctx context.Context,
	deps IntakeDependencies,
	templateID string,
) (*ResetFactoryTemplateResult, error) {
	templateID = strings.TrimSpace(templateID)
	if !isOnboardingFactoryTemplate(templateID) {
		return nil, ErrUnknownOnboardingFactoryTemplate
	}

	db := database.DB(ctx)
	matches, err := listOnboardingFactoryTemplateCanvases(db, templateID)
	if err != nil {
		return nil, err
	}

	result := &ResetFactoryTemplateResult{}
	for _, match := range matches {
		if err := resetFactoryAppCanvas(ctx, db, deps, match.factory, match.canvas); err != nil {
			result.Failures = append(result.Failures, ResetFactoryTemplateFailure{
				CanvasID: match.canvas.ID.String(),
				Name:     match.canvas.Name,
				Error:    err.Error(),
			})
			continue
		}
		result.Reset++
	}
	return result, nil
}

func countOnboardingFactoryTemplates(db *gorm.DB) (map[string]int, error) {
	counts := map[string]int{}
	matches, err := listOnboardingFactoryTemplateCanvases(db, "")
	if err != nil {
		return nil, err
	}
	for _, match := range matches {
		counts[onboardingTemplateIDFor(db, match.factory, match.canvas)]++
	}
	return counts, nil
}

func listOnboardingFactoryTemplateCanvases(db *gorm.DB, templateID string) ([]factoryAppMatch, error) {
	factories, err := models.ListFactoriesAll(db)
	if err != nil {
		return nil, err
	}

	byID := make(map[uuid.UUID]*models.Factory, len(factories))
	for i := range factories {
		byID[factories[i].ID] = &factories[i]
	}

	var matches []factoryAppMatch
	for _, factory := range factories {
		factoryCanvases, err := factory.ListCanvases(db)
		if err != nil {
			return nil, err
		}
		ids := make([]uuid.UUID, 0, len(factoryCanvases))
		canvasesByID := make(map[uuid.UUID]*models.Canvas, len(factoryCanvases))
		for i := range factoryCanvases {
			if factoryCanvases[i].LiveVersionID == nil {
				continue
			}
			ids = append(ids, factoryCanvases[i].ID)
			canvasesByID[factoryCanvases[i].ID] = &factoryCanvases[i]
		}
		if len(ids) == 0 {
			continue
		}
		specs, err := models.FindLiveCanvasSpecsByCanvasIDs(db, ids)
		if err != nil {
			return nil, err
		}
		for canvasID, spec := range specs {
			canvas := canvasesByID[canvasID]
			id := classifyOnboardingTemplate(db, byID[factory.ID], canvas, spec)
			if id == "" {
				continue
			}
			if templateID != "" && id != templateID {
				continue
			}
			matches = append(matches, factoryAppMatch{factory: byID[factory.ID], canvas: canvas})
		}
	}
	return matches, nil
}

func onboardingTemplateIDFor(db *gorm.DB, factory *models.Factory, canvas *models.Canvas) string {
	if factory == nil || canvas == nil || canvas.LiveVersionID == nil {
		return ""
	}
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
	if err != nil {
		return ""
	}
	return classifyOnboardingTemplate(db, factory, canvas, models.LiveCanvasSpec{
		Nodes: version.Nodes,
		Edges: version.Edges,
	})
}

func classifyOnboardingTemplate(
	db *gorm.DB,
	factory *models.Factory,
	canvas *models.Canvas,
	spec models.LiveCanvasSpec,
) string {
	if factory == nil || canvas == nil {
		return ""
	}
	if models.IsBacklogFactoryApp(spec.Nodes, spec.Edges) {
		return onboardingTemplateBacklog
	}
	intake, err := models.FindFactoryIntakeByCanvasID(db, canvas.ID)
	if err == nil && intake.FactoryID == factory.ID {
		return onboardingTemplateIntake
	}
	id := models.FactoryAppTemplateID(spec.Nodes)
	if strings.HasPrefix(id, onboardingTemplateIntakePrefix) {
		return onboardingTemplateIntake
	}
	if isOnboardingFactoryTemplate(id) {
		return id
	}
	if resolved, ok := resolveFactoryTemplate(spec.Nodes); ok && isOnboardingFactoryTemplate(resolved.id) {
		return resolved.id
	}
	return ""
}

func resetFactoryAppCanvas(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	canvas *models.Canvas,
) error {
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
	if err != nil {
		return err
	}

	defaults, stampNode, stampID, stampVersion, stampProvider, err := materializeFactoryAppReset(db, factory, canvas, version)
	if err != nil {
		return err
	}

	doc, err := yaml.CanvasFromYAML([]byte(defaults.canvasYAML))
	if err != nil {
		return err
	}
	nodes, edges, err := doc.Parse(deps.Registry, canvas.OrganizationID.String())
	if err != nil {
		return err
	}
	nodes = copyLiveNodeMetadata(version.Nodes, nodes)

	return db.Transaction(func(tx *gorm.DB) error {
		locked, err := models.LockCanvasForUpdate(tx, canvas.OrganizationID, canvas.ID)
		if err != nil {
			return err
		}
		if locked.LiveVersionID == nil || *locked.LiveVersionID != version.ID {
			return errFactoryTemplateResetStale
		}
		*canvas = *locked

		if err := canvases.PublishGeneratedCanvasNodesWithOwner(
			ctx,
			tx,
			canvas,
			nil,
			resetFactoryTemplateCommitMessage,
			nodes,
			edges,
			intakePublisherOptions(deps, canvas.OrganizationID),
		); err != nil {
			return err
		}
		return canvas.StampFactoryAppTemplateFor(tx, stampNode, stampID, stampVersion, stampProvider)
	})
}

func materializeFactoryAppReset(
	db *gorm.DB,
	factory *models.Factory,
	canvas *models.Canvas,
	version *models.CanvasVersion,
) (*materializedFactoryTemplate, string, string, int, string, error) {
	intake, err := models.FindFactoryIntakeByCanvasID(db, canvas.ID)
	if err == nil && intake.FactoryID == factory.ID {
		defaults, materializeErr := materializeIntakeDefaults(db, canvas, version, intake)
		if materializeErr != nil {
			return nil, "", "", 0, "", materializeErr
		}
		return defaults, intakeTriggerNodeID, "intake:" + intake.Source, factoryTemplateVersion, "", nil
	}
	if err != nil && !errors.Is(err, models.ErrFactoryIntakeNotFound) {
		return nil, "", "", 0, "", err
	}

	if models.IsBacklogFactoryApp(version.Nodes, version.Edges) {
		defaults, materializeErr := materializeBacklogDefaults(db, factory, canvas, version)
		if materializeErr != nil {
			return nil, "", "", 0, "", materializeErr
		}
		return defaults, backlogTriggerNodeID, models.FactoryAppTemplateBacklogID, backlogTemplateVersion, "", nil
	}

	defaults, err := materializeNonIntakeFactoryAppDefaults(db, factory, canvas, version)
	if err != nil {
		return nil, "", "", 0, "", err
	}
	resolved, ok := resolveFactoryTemplate(version.Nodes)
	if !ok {
		resolved, ok = lookupFactoryAppTemplate(defaults.templateID, "")
	}
	stampNode := resolved.entrypointNodeID
	if !ok || stampNode == "" {
		stampNode = defaults.templateID
	}
	return defaults, stampNode, defaults.templateID, factoryTemplateVersion, resolved.provider, nil
}

func copyLiveNodeMetadata(liveNodes, proposedNodes []models.Node) []models.Node {
	result := make([]models.Node, len(proposedNodes))
	copy(result, proposedNodes)
	for i, proposed := range result {
		for _, live := range liveNodes {
			if proposed.ID == live.ID {
				result[i].Metadata = live.Metadata
				break
			}
		}
	}
	return result
}
