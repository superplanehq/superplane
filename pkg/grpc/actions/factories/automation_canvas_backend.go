package factories

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/automations"
	"github.com/superplanehq/superplane/pkg/automationspec"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

type CanvasAutomationBackend struct {
	Deps IntakeDependencies
}

var (
	_ automations.Backend        = CanvasAutomationBackend{}
	_ automations.LegacyImporter = CanvasAutomationBackend{}
)

func (b CanvasAutomationBackend) ArtifactStages(tx *gorm.DB, placement automations.Placement, artifactIDs []string) (map[string]string, error) {
	canvasIDs := make([]uuid.UUID, 0, len(artifactIDs))
	for _, artifactID := range artifactIDs {
		id, err := uuid.Parse(artifactID)
		if err != nil {
			return nil, err
		}
		canvasIDs = append(canvasIDs, id)
	}
	if len(canvasIDs) == 0 {
		return nil, nil
	}

	canvases, err := models.FindFactoryCanvasesByIDs(tx, placement.OrganizationID, placement.FactoryID, canvasIDs)
	if err != nil {
		return nil, err
	}
	stages := make(map[string]string, len(canvases))
	for _, canvas := range canvases {
		if canvas.ColumnKey != nil {
			stages[canvas.ID.String()] = *canvas.ColumnKey
		}
	}
	return stages, nil
}

func (b CanvasAutomationBackend) Check(ctx context.Context, tx *gorm.DB, placement automations.Placement, source *automationspec.Document) error {
	_, _, err := b.compile(tx, placement, source, nil, versionOwnerID(ctx))
	return err
}

func (b CanvasAutomationBackend) Apply(
	ctx context.Context, tx *gorm.DB, placement automations.Placement, actor, id uuid.UUID,
	previous automations.Artifact, source *automationspec.Document,
) (automations.Artifact, func(), error) {
	var canvas *models.Canvas
	var live *models.CanvasVersion
	if previous.ID != "" {
		var err error
		canvas, live, err = lockCustomAutomationArtifact(tx, placement, previous)
		if err != nil {
			return automations.Artifact{}, nil, err
		}
	}

	nodes, edges, err := b.compile(tx, placement, source, live, actor)
	if err != nil {
		return automations.Artifact{}, nil, err
	}
	name := strings.TrimSpace(source.Name)
	if canvas == nil {
		createCtx := authentication.SetUserIdInMetadata(ctx, actor.String())
		_, err = canvases.CreateCanvasWithIDInTransaction(
			createCtx, tx, id, b.Deps.Registry, b.Deps.Encryptor, b.Deps.AuthService, b.Deps.WebhookBaseURL,
			placement.OrganizationID, name, "Run ordered prompt and Bash automation steps.", &placement.FactoryID, nodes, edges,
		)
		if err != nil {
			return automations.Artifact{}, nil, automationCanvasWriteError(source, err)
		}
		canvas, err = models.FindCanvasInTransaction(tx, placement.OrganizationID, id)
		if err != nil {
			return automations.Artifact{}, nil, err
		}
		if err := canvas.SetColumnKey(tx, placement.Stage); err != nil {
			return automations.Artifact{}, nil, err
		}
	} else {
		err := canvases.PublishGeneratedCanvasNodes(ctx, tx, canvas, actor, "Update custom automation", nodes, edges, intakePublisherOptions(b.Deps, placement.OrganizationID))
		if err != nil {
			return automations.Artifact{}, nil, automationCanvasWriteError(source, err)
		}
		if err := canvases.UpdateCanvasInTransaction(tx, canvas, &name, nil); err != nil {
			return automations.Artifact{}, nil, automationCanvasWriteError(source, err)
		}
	}

	published, err := models.FindLiveCanvasVersionInTransaction(tx, canvas.ID)
	if err != nil {
		return automations.Artifact{}, nil, err
	}
	if err := checkAutomationCanvasNodes(source, published.Nodes); err != nil {
		return automations.Artifact{}, nil, err
	}
	return automations.Artifact{ID: canvas.ID.String(), Revision: published.ID.String()}, func() {
		canvases.PublishCanvasUpdated(canvas)
	}, nil
}

func (b CanvasAutomationBackend) Remove(_ context.Context, tx *gorm.DB, placement automations.Placement, previous automations.Artifact) (func(), error) {
	canvas, _, err := lockCustomAutomationArtifact(tx, placement, previous)
	if err != nil {
		return nil, err
	}
	factory, err := models.FindFactory(tx, placement.OrganizationID, placement.FactoryID)
	if err != nil {
		return nil, err
	}
	if err := rejectReservedFactoryAutomation(tx, factory, canvas); err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory automation")
	}
	if err := canvas.SoftDeleteInTransaction(tx); err != nil {
		return nil, err
	}
	return func() { canvases.PublishCanvasDeleted(canvas) }, nil
}

func (b CanvasAutomationBackend) compile(
	tx *gorm.DB, placement automations.Placement, source *automationspec.Document, live *models.CanvasVersion, callerID uuid.UUID,
) ([]models.Node, []models.Edge, error) {
	if err := validateAutomationCanvasPlacement(source, placement.Stage); err != nil {
		return nil, nil, err
	}
	var factory *models.Factory
	var err error
	if reads := factoryAutomationReadsForOrganization(tx, placement.OrganizationID); reads != nil && reads.factory.ID == placement.FactoryID {
		factory = reads.factory
	} else {
		factory, err = models.FindFactory(tx, placement.OrganizationID, placement.FactoryID)
		if err != nil {
			return nil, nil, err
		}
	}
	if err := checkAutomationRepository(tx, factory, source); err != nil {
		return nil, nil, err
	}
	if err := validateCustomStageAutomation(tx, factory, placement.Stage); err != nil {
		return nil, nil, err
	}
	agent, err := resolveAutomationAgent(tx, factory, source, live, callerID)
	if err != nil {
		return nil, nil, err
	}
	settings, err := automationCanvasSettings(placement.Stage, source, agent)
	if err != nil {
		return nil, nil, err
	}
	if err := validateCustomAutomationWebhooks(tx, factory, settings.Triggers); err != nil {
		return nil, nil, source.Diagnostic("triggers", grpcerrors.StatusMessage(err))
	}
	if strings.HasPrefix(placement.Stage, "custom-") {
		if err := validateCustomStageTriggers(settings); err != nil {
			return nil, nil, source.Diagnostic("triggers", grpcerrors.StatusMessage(err))
		}
	}
	legacy := live != nil && legacyCustomStageGraph(&models.Canvas{ColumnKey: &placement.Stage}, live)
	if legacy && !settings.hasWebhookTrigger() && len(customStageCore(live).Nodes) > 0 {
		nodes, edges := editLegacyCustomStageGraph(customStageCore(live), settings)
		return nodes, edges, nil
	}
	graph, err := buildCustomAutomationCanvas(tx, factory, strings.TrimSpace(source.Name), settings)
	if err != nil {
		return nil, nil, err
	}
	if live != nil {
		if err := preserveCustomAutomationIdentities(live, graph); err != nil {
			return nil, nil, err
		}
	}
	var nodes []models.Node
	var edges []models.Edge
	if reads := factoryAutomationReads(tx, factory); reads != nil {
		nodes, edges, err = graph.ParseInTransactionWithIntegrations(b.Deps.Registry, placement.OrganizationID.String(), tx, reads.integrationsByID)
	} else {
		nodes, edges, err = graph.ParseInTransaction(b.Deps.Registry, placement.OrganizationID.String(), tx)
	}
	if err != nil {
		return nil, nil, source.Diagnostic("$", "SuperPlane cannot build this automation. Check the triggers and steps.")
	}
	if err := checkAutomationCanvasNodes(source, nodes); err != nil {
		return nil, nil, err
	}
	for i := range nodes {
		node := &nodes[i]
		if !isFactoryAgentHarness(node.ComponentName()) {
			continue
		}
		action, err := b.Deps.Registry.GetAction(node.ComponentName())
		if err != nil {
			return nil, nil, err
		}
		if validator, ok := action.(core.NodeConfigurationValidator); ok {
			if err := validator.ValidateNodeConfiguration(node.Configuration); err != nil {
				return nil, nil, source.Diagnostic("agent", "SuperPlane cannot apply this agent configuration. Check the agent and its steps.")
			}
		}
	}
	if legacy {
		nodes, edges = editCustomStageGraph(live, settings, nodes, edges)
	}
	return nodes, edges, nil
}

func validateAutomationCanvasPlacement(source *automationspec.Document, stage string) error {
	if customAutomationUsesDispatch(stage) {
		return automationspec.ValidateDispatch(source)
	}
	return automationspec.ValidateStage(source, automationspec.Stage(stage))
}

func lockCustomAutomationArtifact(
	tx *gorm.DB, placement automations.Placement, artifact automations.Artifact,
) (*models.Canvas, *models.CanvasVersion, error) {
	id, err := uuid.Parse(artifact.ID)
	if err != nil {
		return nil, nil, gorm.ErrRecordNotFound
	}
	canvas, err := models.LockCanvasForUpdate(tx, placement.OrganizationID, id)
	if err != nil {
		return nil, nil, err
	}
	legacy, version, err := canvasLegacyAutomation(tx, canvas)
	if err != nil {
		return nil, nil, err
	}
	if legacy.Placement != placement {
		return nil, nil, gorm.ErrRecordNotFound
	}
	if artifact.Revision != legacy.Artifact.Revision {
		return nil, nil, grpcerrors.FailedPrecondition(nil, "The automation changed during this save. Save again to use the current version.")
	}
	return canvas, version, nil
}

func checkAutomationRepository(tx *gorm.DB, factory *models.Factory, source *automationspec.Document) error {
	setup := factory.OnboardingConfigValue()
	if strings.TrimSpace(setup.AppRepository) == "" {
		return source.Diagnostic("triggers", "Set a repository for this factory before you save the automation.")
	}
	provider := setup.EffectiveVCSProvider()
	if provider != models.ProviderGitHub && provider != models.ProviderBitbucket {
		return source.Diagnostic("triggers", "Connect a GitHub or Bitbucket repository to this factory.")
	}
	id, err := uuid.Parse(setup.VCSIntegrationID)
	if err != nil {
		return source.Diagnostic("triggers", "Connect the factory repository before you save the automation.")
	}
	integration, err := factoryAutomationIntegrationByID(tx, factory.OrganizationID, id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if integration == nil || integration.State != models.IntegrationStateReady || integration.AppName != provider {
		return source.Diagnostic("triggers", "Reconnect the factory repository before you save the automation.")
	}
	return nil
}

func checkAutomationCanvasNodes(source *automationspec.Document, nodes []models.Node) error {
	for i := range nodes {
		node := &nodes[i]
		if node.ErrorMessage == nil {
			continue
		}
		switch {
		case node.ComponentName() == "schedule":
			template, _ := node.Metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
			return source.Diagnostic(fmt.Sprintf("triggers[%d].config.cron", configInt(template, "triggerIndex")), "SuperPlane cannot activate this schedule. Check the cron expression and UTC offset.")
		case node.Type == models.NodeTypeTrigger:
			return source.Diagnostic("triggers", "SuperPlane cannot activate this trigger. Check the factory repository connection.")
		default:
			return source.Diagnostic("agent", "SuperPlane cannot prepare the agent. Check the agent connection and step settings.")
		}
	}
	return nil
}

func automationCanvasWriteError(source *automationspec.Document, err error) error {
	if errors.Is(err, models.ErrCanvasNameAlreadyExists) || grpcerrors.Code(err) == codes.AlreadyExists {
		return source.Diagnostic("name", "Choose a name that another automation in this factory does not use.")
	}
	return err
}

func automationCanvasSettings(stage string, source *automationspec.Document, agent *intakeAgent) (*customAutomationSettings, error) {
	input := &pb.FactoryCustomAutomation_Settings{}
	for _, trigger := range source.Triggers {
		start := &pb.FactoryCustomAutomation_StartTrigger{}
		switch trigger.Use {
		case automationspec.Manual:
			start.StartMode = pb.FactoryCustomAutomation_START_MODE_MANUAL
		case automationspec.TaskEntersStage, automationspec.PullRequestUpdated:
			start.StartMode = pb.FactoryCustomAutomation_START_MODE_AUTOMATIC
		case automationspec.TaskLeavesStage:
			start.StartMode = pb.FactoryCustomAutomation_START_MODE_ON_LEAVE
		case automationspec.Scheduled:
			start.StartMode = pb.FactoryCustomAutomation_START_MODE_SCHEDULED
			start.Schedule = &pb.FactoryCustomAutomation_Schedule{
				CronExpression: trigger.Config.Cron, UtcOffset: trigger.Config.UTCOffset,
			}
		case automationspec.Webhook:
			start.StartMode = pb.FactoryCustomAutomation_START_MODE_WEBHOOK
			start.WebhookId = trigger.Config.WebhookID
		}
		input.Triggers = append(input.Triggers, start)
	}
	triggers, startMode, schedule, err := parseCustomAutomationTriggers(input)
	if err != nil {
		return nil, source.Diagnostic("triggers", "Check the trigger types and schedule settings.")
	}
	if err := validateCustomAutomationRunContext(source.RunsOnTask, triggers); err != nil {
		return nil, source.Diagnostic("runs_on_task", grpcerrors.StatusMessage(err))
	}
	steps := make([]runner.AgentStep, 0, len(source.Steps))
	for index, step := range source.Steps {
		directory := strings.TrimSpace(step.WorkingDirectory)
		if err := runner.ValidateStepWorkingDirectory(directory); err != nil {
			return nil, source.Diagnostic(fmt.Sprintf("steps[%d].working_directory", index), "The working directory "+err.Error()+".")
		}
		translated := runner.AgentStep{Name: strings.TrimSpace(step.Name), Secrets: step.Secrets, WorkingDirectory: directory}
		if step.Bash != "" {
			translated.Type = runner.AgentStepBash
			translated.Command = &step.Bash
		} else {
			translated.Type = runner.AgentStepPrompt
			translated.Prompt = &step.Prompt
		}
		steps = append(steps, translated)
	}
	return &customAutomationSettings{
		ColumnKey: stage, StartMode: startMode, Agent: agent, Steps: steps, Icon: source.Icon,
		ExecutionTimeoutSeconds: source.TimeoutSeconds, Schedule: schedule, Triggers: triggers,
		RunsOnTask:        source.RunsOnTask,
		SkipSetupCommands: source.SkipSetupCommands, SkipRepositoryCheckout: source.SkipRepositoryCheckout,
	}, nil
}
