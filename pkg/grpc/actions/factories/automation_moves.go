package factories

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/automations"
	"github.com/superplanehq/superplane/pkg/automationspec"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func (b CanvasAutomationBackend) moveAutomationCards(ctx context.Context, tx *gorm.DB, factory *models.Factory, line *models.FactoryLine, stages []models.FactoryLineStage, config *pb.FactoryLine_StageConfiguration) error {
	destinations := map[string]*models.FactoryLineStage{}
	for i, requested := range config.Stages {
		if requested.Key == "" {
			destinations[stages[i].Key] = &stages[i]
			continue
		}
		if destinations[requested.Key] != nil {
			return invalidArgument("stage key is duplicated")
		}
		destinations[requested.Key] = &stages[i]
		destinations[stages[i].Key] = &stages[i]
	}
	saved := map[string]bool{}
	for _, stage := range line.Stages {
		saved[stage.Key] = true
		if stage.AgentDisplayKey != "" && destinations[stage.Key] == nil {
			return invalidArgument("Keep the execution stage for the moved agent step.")
		}
	}
	seen := map[string]bool{}
	for _, move := range config.AutomationMoves {
		id, err := uuid.Parse(move.GetAutomationId())
		if err != nil {
			return invalidArgument("automation id is invalid")
		}
		target := destinations[move.DestinationKey]
		if target == nil || move.DestinationKey == "" || !saved[move.SourceKey] {
			return invalidArgument("Select a stage on this line.")
		}
		if target.Key == "backlog" && move.Type != "intake" {
			return invalidArgument("Select a stage other than Backlog for this automation.")
		}
		if move.Type == "agent-step" {
			stage := destinations[move.ExecutionKey]
			if stage == nil || stage.StepIndex == nil || *stage.StepIndex < 0 || *stage.StepIndex >= len(line.Steps) || line.Steps[*stage.StepIndex].AppID != id {
				return invalidArgument("Keep the execution stage for this agent step.")
			}
			if seen[stage.Key] || boardSource(stage.AgentDisplayKey, stage.Key, saved) != move.SourceKey {
				return invalidArgument("The agent placement changed. Reload the board.")
			}
			seen[stage.Key] = true
			stage.AgentDisplayKey = displayOverride(stage.Key, target.Key)
			continue
		}
		canvas, origin, err := resolveBoardCanvas(tx, factory, move.Type, id)
		if err != nil {
			return err
		}
		if seen[canvas.ID.String()] || !saved[origin] || (move.ExecutionKey != "" && origin != move.ExecutionKey) || boardSource(canvas.DisplayColumnKey, origin, saved) != move.SourceKey {
			return invalidArgument("The automation placement changed. Reload the board.")
		}
		seen[canvas.ID.String()] = true
		runtimeMoved, err := b.moveCustomExecution(ctx, tx, factory, canvas, target.Key)
		if err != nil {
			return err
		}
		if runtimeMoved {
			origin = target.Key
		}
		if destinations[origin] == nil {
			return invalidArgument("Keep the execution stage for this automation.")
		}
		if err := canvas.SetDisplayColumn(tx, displayOverride(origin, target.Key)); err != nil {
			return err
		}
	}
	for i := range stages {
		if destinations[stages[i].AgentDisplayKey] == nil {
			stages[i].AgentDisplayKey = ""
		}
	}
	hasRemovedStages := false
	for key := range saved {
		if destinations[key] == nil {
			hasRemovedStages = true
			break
		}
	}
	if hasRemovedStages {
		// Removing an execution stage must not orphan a card displayed elsewhere.
		apps, err := factory.ListCanvases(tx)
		if err != nil {
			return err
		}
		handlers, err := factory.ListPRFeedbackHandlers(tx)
		if err != nil {
			return err
		}
		feedback := map[uuid.UUID]bool{}
		for _, handler := range handlers {
			feedback[handler.CanvasID] = true
		}
		for _, app := range apps {
			if app.DisplayColumnKey == "" {
				continue
			}
			origin := ""
			if app.ColumnKey != nil {
				origin = *app.ColumnKey
				if origin == "implement" {
					origin = "phase-0"
				}
			}
			if feedback[app.ID] {
				origin = "verify"
			}
			if saved[origin] && destinations[origin] == nil {
				return invalidArgument("Keep the execution stage for the moved automation.")
			}
		}
	}
	return nil
}

func boardSource(display, origin string, saved map[string]bool) string {
	if saved[display] {
		return display
	}
	return origin
}
func displayOverride(origin, target string) string {
	if origin == target {
		return ""
	}
	return target
}

func resolveBoardCanvas(tx *gorm.DB, factory *models.Factory, kind string, id uuid.UUID) (*models.Canvas, string, error) {
	origin := ""
	switch kind {
	case "intake":
		intake, err := factory.FindIntake(tx, id)
		if err != nil {
			return nil, "", err
		}
		id, origin = intake.CanvasID, "backlog"
	case "feedback":
		handler, err := factory.FindPRFeedbackHandler(tx, id)
		if err != nil {
			return nil, "", err
		}
		id, origin = handler.CanvasID, "verify"
	case "canvas":
	default:
		return nil, "", invalidArgument("Unknown automation type.")
	}
	canvas, err := models.LockCanvasForUpdate(tx, factory.OrganizationID, id)
	if err != nil {
		return nil, "", err
	}
	if canvas.FactoryID == nil || *canvas.FactoryID != factory.ID {
		return nil, "", invalidArgument("The automation does not belong to this workspace.")
	}
	if origin != "" {
		return canvas, origin, nil
	}
	if canvas.ColumnKey != nil {
		origin = *canvas.ColumnKey
		if origin == "implement" {
			origin = "phase-0"
		}
		return canvas, origin, nil
	}
	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(tx, []uuid.UUID{id})
	if err != nil {
		return nil, "", err
	}
	switch factoryAutomationTemplateID(specs[id]) {
	case models.FactoryAppTemplateBacklogID:
		origin = "backlog"
	case prClosureTemplateID:
		origin = "done"
	case reviewRequestTemplateID:
		origin = "verify"
	default:
		switch canvas.Name {
		case "Backlog", "Ingest":
			origin = "backlog"
		case "PR Closure":
			origin = "done"
		default:
			return nil, "", invalidArgument("The automation is not attached to the board.")
		}
	}
	return canvas, origin, nil
}

// Move only task-triggered custom graphs. Event and scheduled graphs keep all subscriptions.
func (b CanvasAutomationBackend) moveCustomExecution(ctx context.Context, tx *gorm.DB, factory *models.Factory, canvas *models.Canvas, target string) (bool, error) {
	if canvas.ColumnKey == nil {
		return false, nil
	}
	if strings.HasPrefix(*canvas.ColumnKey, "custom-") {
		if !strings.HasPrefix(target, "custom-") {
			return false, nil
		}
		return true, canvas.SetColumnKey(tx, target)
	}
	if canvas.LiveVersionID == nil {
		return false, nil
	}
	live, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
	if err != nil {
		return false, err
	}
	if models.FactoryAppTemplateID(live.Nodes) != models.FactoryAppTemplateCustomAutomationID {
		return false, nil
	}
	triggers, err := legacyAutomationTriggers(*canvas.ColumnKey, live.Nodes)
	if err != nil {
		return false, err
	}
	hasTask := false
	for _, trigger := range triggers {
		switch trigger.Uses {
		case automationspec.Scheduled, automationspec.Webhook, automationspec.PullRequestUpdated:
			return false, nil
		case automationspec.TaskEntersStage, automationspec.TaskLeavesStage:
			hasTask = true
		}
	}
	if !hasTask {
		return false, nil
	}
	if target == "phase-0" {
		target = "implement"
	}
	// Placement can differ from execution when the destination uses another trigger contract.
	if !compatibleTaskAutomationDestination(triggers, target) {
		return false, nil
	}
	authored, err := findLegacyAutomationSource(tx, canvas)
	if err != nil {
		return false, err
	}
	actor := versionOwnerID(ctx)
	if actor == uuid.Nil {
		return false, invalidArgument("A user is required to move this task automation.")
	}
	nodes, edges, err := b.compileMovedAutomation(tx, factory, canvas, live, authored, triggers, target, actor)
	if err != nil {
		return false, err
	}
	if err := canvas.SetColumnKey(tx, target); err != nil {
		return false, err
	}
	if err := canvases.PublishGeneratedCanvasNodes(ctx, tx, canvas, actor, "Move custom automation", nodes, edges, intakePublisherOptions(b.Deps, factory.OrganizationID)); err != nil {
		return false, err
	}
	published, err := models.FindLiveCanvasVersionInTransaction(tx, canvas.ID)
	if err != nil {
		return false, err
	}
	for _, node := range published.Nodes {
		if node.ErrorMessage != nil {
			return false, invalidArgument("SuperPlane could not activate the automation in this stage.")
		}
	}
	if authored != nil {
		authored.ArtifactRevision = published.ID.String()
		authored.Revision++
		return true, authored.Save(tx)
	}
	return true, nil
}

func compatibleTaskAutomationDestination(triggers []automationspec.Trigger, target string) bool {
	if strings.HasPrefix(target, "custom-") {
		return len(triggers) == 1 && triggers[0].Uses == automationspec.TaskEntersStage
	}
	switch target {
	case "implement", "done":
		return true
	case "verify":
		for _, trigger := range triggers {
			if trigger.Uses == automationspec.TaskEntersStage {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (b CanvasAutomationBackend) compileMovedAutomation(tx *gorm.DB, factory *models.Factory, canvas *models.Canvas, live *models.CanvasVersion, authored *models.AutomationSource, triggers []automationspec.Trigger, target string, actor uuid.UUID) ([]models.Node, []models.Edge, error) {
	if authored != nil {
		if authored.ArtifactRevision != live.ID.String() {
			return nil, nil, invalidArgument("The automation changed. Reload it before moving it.")
		}
		document, err := automationspec.Parse(authored.Source)
		if err != nil {
			return nil, nil, err
		}
		if err := automationspec.ValidateStage(document, automationspec.Stage(target)); err != nil {
			return nil, nil, invalidArgument(err.Error())
		}
		return b.compile(tx, automations.Placement{OrganizationID: factory.OrganizationID, FactoryID: factory.ID, Stage: target}, document, live, actor)
	}
	contract := &automationspec.Document{Definition: automationspec.Definition{Triggers: triggers}}
	if err := automationspec.ValidateStage(contract, automationspec.Stage(target)); err != nil {
		return nil, nil, invalidArgument(err.Error())
	}
	saved, err := readCustomAutomationVersion(factory, canvas, live)
	if err != nil {
		return nil, nil, err
	}
	settings, err := parseCustomAutomationSettings(tx, factory, saved.Settings)
	if err != nil {
		return nil, nil, err
	}
	settings.Agent = intakeAgentFromCanvasNodes(live.Nodes)
	settings.ColumnKey = *canvas.ColumnKey
	original, err := buildCustomAutomationCanvas(tx, factory, canvas.Name, settings)
	if err != nil {
		return nil, nil, err
	}
	if err := preserveCustomAutomationIdentities(live, original); err != nil {
		return nil, nil, err
	}
	if !equalAutomationExecutionGraph(live, original) {
		return nil, nil, invalidArgument("This automation has custom canvas changes. Keep its execution stage.")
	}
	settings.ColumnKey = target
	graph, err := buildCustomAutomationCanvas(tx, factory, canvas.Name, settings)
	if err != nil {
		return nil, nil, err
	}
	if err := preserveCustomAutomationIdentities(live, graph); err != nil {
		return nil, nil, err
	}
	return graph.ParseInTransaction(b.Deps.Registry, factory.OrganizationID.String(), tx)
}
