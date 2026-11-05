package factories

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/authentication"
	factorycomponent "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

const (
	customAutomationDefaultName      = "Custom automation"
	customAutomationTriggerID        = "on-pull-request"
	customAutomationTaskTriggerID    = "on-task"
	customAutomationRunnerID         = "verify"
	customAutomationScheduleRunnerID = "schedule-run"
	customAutomationVersion          = 1
	backlogTaskResultStepName        = "Write task result"
	backlogTaskRunnerName            = "Run automation"
)

const backlogTaskOutputInstructions = `

When this run has no task and you found work to do, call the create_task tool.
Pass a short imperative title and a description of the change, the reason, and how to verify it.
When this run already has a task, or when there is no task to create, do not call create_task.`

// Older canvases told the agent to write /tmp/proposal.json. Strip that text
// from the editor until the next save publishes the create_task instructions.
const backlogTaskProposalInstructions = `

Do not edit repository files, commit, push, or open a pull request.
Write the task to /tmp/proposal.json as a JSON object with non-empty string fields "title" and "description".
The title is a short imperative sentence. The description explains the change, the evidence, and how to verify it.
If there is no task to create, write exactly {} to /tmp/proposal.json.
Your final chat message is not the structured result.`

func CreateFactoryCustomAutomation(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.CreateFactoryCustomAutomationRequest,
) (*pb.CreateFactoryCustomAutomationResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create custom automation")
	}

	db := database.DB(ctx)
	organization, err := models.FindOrganizationByIDInTransaction(db, orgID.String())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create custom automation")
	}
	if !organization.HasExperimentalFeature(features.FeatureFactoryCustomAutomations) {
		return nil, factoryErrorToStatus(errCustomAutomationsDisabled, "failed to create custom automation")
	}
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create custom automation")
	}

	column := strings.TrimSpace(req.GetColumnKey())
	if column == "" {
		column = models.CanvasColumnKeyVerify
	}
	if column != models.CanvasColumnKeyVerify && column != models.CanvasColumnKeyBacklog {
		return nil, factoryErrorToStatus(invalidArgument("column must be backlog or verify"), "failed to create custom automation")
	}
	settings, err := parseCustomAutomationSettings(db, factory, req.GetSettings())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create custom automation")
	}
	name := strings.TrimSpace(req.GetName())
	if name == "" {
		name = customAutomationDefaultName
	}
	factoryID := factory.ID
	name, err = models.AvailableCanvasName(db, orgID, &factoryID, name)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create custom automation")
	}

	settings.ColumnKey = column
	doc, err := buildCustomAutomationCanvas(db, factory, name, settings)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create custom automation")
	}
	nodes, edges, err := doc.Parse(deps.Registry, orgID.String())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to build custom automation")
	}
	created, err := canvases.CreateCanvas(
		ctx,
		deps.Registry,
		deps.Encryptor,
		deps.AuthService,
		deps.WebhookBaseURL,
		orgID,
		name,
		doc.Metadata.Description,
		&factoryID,
		nodes,
		edges,
	)
	if err != nil {
		return nil, err
	}
	canvasID, err := uuid.Parse(created.GetCanvas().GetMetadata().GetId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create custom automation")
	}
	canvas, err := models.FindCanvasInTransaction(db, orgID, canvasID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create custom automation")
	}
	if err := canvas.SetColumnKey(db, column); err != nil {
		discardIntakeCanvas(db, orgID, canvasID)
		return nil, factoryErrorToStatus(err, "failed to create custom automation")
	}

	automation, err := serializeCustomAutomation(db, factory, canvas)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create custom automation")
	}
	return &pb.CreateFactoryCustomAutomationResponse{CustomAutomation: automation}, nil
}

func DescribeFactoryCustomAutomation(
	ctx context.Context,
	organizationID string,
	req *pb.DescribeFactoryCustomAutomationRequest,
) (*pb.DescribeFactoryCustomAutomationResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe custom automation")
	}
	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe custom automation")
	}
	canvas, err := findCustomAutomationCanvas(db, factory, req.GetAutomationId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe custom automation")
	}
	automation, err := serializeCustomAutomation(db, factory, canvas)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe custom automation")
	}
	return &pb.DescribeFactoryCustomAutomationResponse{CustomAutomation: automation}, nil
}

func UpdateFactoryCustomAutomation(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.UpdateFactoryCustomAutomationRequest,
) (*pb.UpdateFactoryCustomAutomationResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update custom automation")
	}
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	ownerID, err := uuid.Parse(userID)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to update custom automation")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update custom automation")
	}
	canvas, err := findCustomAutomationCanvas(db, factory, req.GetAutomationId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update custom automation")
	}
	settings, err := parseCustomAutomationSettings(db, factory, req.GetSettings())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update custom automation")
	}
	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return nil, factoryErrorToStatus(invalidArgument("name is required"), "failed to update custom automation")
	}
	if canvas.ColumnKey != nil {
		settings.ColumnKey = *canvas.ColumnKey
	}
	doc, err := buildCustomAutomationCanvas(db, factory, name, settings)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update custom automation")
	}
	if err := preserveCustomAutomationNodeIdentities(db, canvas, doc); err != nil {
		return nil, factoryErrorToStatus(err, "failed to update custom automation")
	}
	nodes, edges, err := doc.ParseInTransaction(deps.Registry, orgID.String(), db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to build custom automation")
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := canvases.PublishGeneratedCanvasNodes(
			ctx,
			tx,
			canvas,
			ownerID,
			"Update custom automation",
			nodes,
			edges,
			intakePublisherOptions(deps, orgID),
		); err != nil {
			return err
		}
		return canvases.UpdateCanvasInTransaction(tx, canvas, &name, nil)
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update custom automation")
	}
	canvases.PublishCanvasUpdated(canvas)

	canvas, err = models.FindCanvasInTransaction(db, orgID, canvas.ID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update custom automation")
	}
	automation, err := serializeCustomAutomation(db, factory, canvas)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update custom automation")
	}
	return &pb.UpdateFactoryCustomAutomationResponse{CustomAutomation: automation}, nil
}

type customAutomationTrigger struct {
	Index    int
	Mode     pb.FactoryCustomAutomation_StartMode
	Schedule *pb.FactoryCustomAutomation_Schedule
}

type customAutomationSettings struct {
	ColumnKey               string
	StartMode               pb.FactoryCustomAutomation_StartMode
	Agent                   *intakeAgent
	Steps                   []runner.AgentStep
	Icon                    string
	ExecutionTimeoutSeconds int
	Schedule                *pb.FactoryCustomAutomation_Schedule
	Triggers                []customAutomationTrigger
}

func parseCustomAutomationSettings(
	tx *gorm.DB,
	factory *models.Factory,
	input *pb.FactoryCustomAutomation_Settings,
) (*customAutomationSettings, error) {
	if input == nil {
		return nil, invalidArgument("settings are required")
	}
	triggers, startMode, scheduleSettings, err := parseCustomAutomationTriggers(input)
	if err != nil {
		return nil, err
	}

	agent, err := customAutomationAgent(tx, factory, input.GetAgent())
	if err != nil {
		return nil, err
	}
	if agent == nil || !isFactoryAgentHarness(agent.component()) {
		return nil, invalidArgument("agent component is not supported")
	}
	steps := make([]runner.AgentStep, 0, len(input.GetSteps()))
	for _, step := range input.GetSteps() {
		parsed := runner.AgentStep{
			Name:             strings.TrimSpace(step.GetName()),
			WorkingDirectory: strings.TrimSpace(step.GetWorkingDirectory()),
			Secrets:          step.GetSecrets(),
		}
		switch step.GetType() {
		case pb.FactoryCustomAutomation_Step_TYPE_PROMPT:
			parsed.Type = runner.AgentStepPrompt
			value := step.GetPrompt()
			parsed.Prompt = &value
		case pb.FactoryCustomAutomation_Step_TYPE_BASH:
			parsed.Type = runner.AgentStepBash
			value := step.GetCommand()
			parsed.Command = &value
		default:
			return nil, invalidArgument("step type must be prompt or bash")
		}
		steps = append(steps, parsed)
	}
	if err := runner.ValidateAgentSteps(steps); err != nil {
		return nil, invalidArgument(err.Error())
	}
	timeout := int(input.GetExecutionTimeoutSeconds())
	if timeout <= 0 {
		timeout = prFeedbackTimeoutSeconds
	}
	return &customAutomationSettings{
		StartMode:               startMode,
		Schedule:                scheduleSettings,
		Triggers:                triggers,
		Agent:                   agent,
		Steps:                   steps,
		Icon:                    strings.TrimSpace(input.GetIcon()),
		ExecutionTimeoutSeconds: timeout,
	}, nil
}

func parseCustomAutomationTriggers(
	input *pb.FactoryCustomAutomation_Settings,
) ([]customAutomationTrigger, pb.FactoryCustomAutomation_StartMode, *pb.FactoryCustomAutomation_Schedule, error) {
	if len(input.GetTriggers()) == 0 {
		if input.GetStartMode() != pb.FactoryCustomAutomation_START_MODE_MANUAL &&
			input.GetStartMode() != pb.FactoryCustomAutomation_START_MODE_AUTOMATIC &&
			input.GetStartMode() != pb.FactoryCustomAutomation_START_MODE_SCHEDULED {
			return nil, 0, nil, invalidArgument("Choose Manual, Automatic, or Schedule.")
		}
		var schedule *pb.FactoryCustomAutomation_Schedule
		if input.GetStartMode() == pb.FactoryCustomAutomation_START_MODE_SCHEDULED {
			parsed, err := parseCustomAutomationSchedule(input.GetSchedule())
			if err != nil {
				return nil, 0, nil, err
			}
			schedule = parsed
		}
		return []customAutomationTrigger{{
			Mode: input.GetStartMode(), Schedule: schedule,
		}}, input.GetStartMode(), schedule, nil
	}

	triggers := make([]customAutomationTrigger, 0, len(input.GetTriggers()))
	manualCount := 0
	automaticCount := 0
	var onlySchedule *pb.FactoryCustomAutomation_Schedule
	scheduledCount := 0
	hasAutomatic := false
	hasManual := false
	for index, trigger := range input.GetTriggers() {
		mode := trigger.GetStartMode()
		var schedule *pb.FactoryCustomAutomation_Schedule
		switch mode {
		case pb.FactoryCustomAutomation_START_MODE_MANUAL:
			manualCount++
			hasManual = true
		case pb.FactoryCustomAutomation_START_MODE_AUTOMATIC:
			automaticCount++
			hasAutomatic = true
		case pb.FactoryCustomAutomation_START_MODE_SCHEDULED:
			parsed, err := parseCustomAutomationSchedule(trigger.GetSchedule())
			if err != nil {
				return nil, 0, nil, err
			}
			schedule = parsed
			scheduledCount++
			onlySchedule = parsed
		default:
			return nil, 0, nil, invalidArgument("Choose Manual, Automatic, or Schedule.")
		}
		triggers = append(triggers, customAutomationTrigger{Index: index, Mode: mode, Schedule: schedule})
	}
	if manualCount > 1 || automaticCount > 1 {
		return nil, 0, nil, invalidArgument("Choose each event trigger once.")
	}
	startMode := pb.FactoryCustomAutomation_START_MODE_MANUAL
	if hasAutomatic {
		startMode = pb.FactoryCustomAutomation_START_MODE_AUTOMATIC
	} else if scheduledCount > 0 {
		startMode = pb.FactoryCustomAutomation_START_MODE_SCHEDULED
	} else if !hasManual {
		startMode = pb.FactoryCustomAutomation_START_MODE_UNSPECIFIED
	}
	if scheduledCount != 1 {
		onlySchedule = nil
	}
	return triggers, startMode, onlySchedule, nil
}

func parseCustomAutomationSchedule(
	input *pb.FactoryCustomAutomation_Schedule,
) (*pb.FactoryCustomAutomation_Schedule, error) {
	if input == nil {
		return nil, invalidArgument("Enter a cron expression and UTC offset.")
	}
	config, err := scheduledIntakeSchedule(map[string]string{
		"type": "cron", "cronExpression": input.GetCronExpression(),
		"timezone": strconv.FormatFloat(input.GetUtcOffset(), 'f', -1, 64),
	})
	if err != nil {
		return nil, err
	}
	return &pb.FactoryCustomAutomation_Schedule{
		CronExpression: *config.CronExpression, UtcOffset: input.GetUtcOffset(),
	}, nil
}

func customAutomationAgent(tx *gorm.DB, factory *models.Factory, input *pb.FactoryAppTemplateAgent) (*intakeAgent, error) {
	if input == nil || strings.TrimSpace(input.GetComponent()) == "" {
		return resolveIntakeAgent(tx, factory), nil
	}
	agent := &intakeAgent{
		Component: strings.TrimSpace(input.GetComponent()),
		Model:     strings.TrimSpace(input.GetModel()),
	}
	source := strings.TrimSpace(input.GetCredentialSource())
	integrationName := strings.TrimSpace(input.GetCredentialIntegrationName())
	switch source {
	case "", runner.CredentialsSourceSecret, runner.CredentialsSourceIntegration, runner.CredentialsSourceHosted:
	case runner.CredentialsSourceSubscription:
		if agent.Component != "runnerCodex" {
			return nil, invalidArgument("ChatGPT subscription uses Codex")
		}
	default:
		return nil, invalidArgument("agent credential source is not supported")
	}
	if source == runner.CredentialsSourceIntegration && integrationName == "" {
		return nil, invalidArgument("agent credential integration name is required")
	}
	// Run SuperPlane Agent has no credential field. A hosted source is the
	// workspace model, and storing it makes node setup fail.
	if agent.Component == models.SuperPlaneRunnerComponent {
		return agent, nil
	}
	if source != "" {
		agent.Credentials = map[string]any{"source": source}
		if integrationName != "" {
			agent.Credentials["integration"] = map[string]any{"name": integrationName}
		}
		return agent, nil
	}
	workspace := resolveIntakeAgent(tx, factory)
	if workspace != nil && workspace.Component == agent.Component {
		agent.Credentials = workspace.Credentials
		agent.LLMProvider = workspace.LLMProvider
		if agent.Model == "" {
			agent.Model = workspace.Model
		}
	}
	return agent, nil
}

func buildCustomAutomationCanvas(
	tx *gorm.DB,
	factory *models.Factory,
	name string,
	settings *customAutomationSettings,
) (*yaml.Canvas, error) {
	config := factory.OnboardingConfigValue()
	repository := strings.TrimSpace(config.AppRepository)
	integrationID := strings.TrimSpace(config.VCSIntegrationID)
	if repository == "" {
		return nil, invalidArgument("workspace repository is required")
	}
	if integrationID == "" {
		return nil, invalidArgument("workspace VCS integration is required")
	}
	provider := config.EffectiveVCSProvider()
	if len(settings.Triggers) > 1 {
		return buildMultiTriggerCustomAutomationCanvas(tx, factory, name, settings, repository, integrationID, provider)
	}
	mode := customAutomationStartModeName(settings.StartMode)
	metadata := models.FactoryAppTemplateMetadataFor(
		models.FactoryAppTemplateCustomAutomationID,
		customAutomationVersion,
		provider,
	)
	template := metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
	template["startMode"] = mode
	if settings.Icon != "" {
		template["icon"] = settings.Icon
	}

	triggerConfiguration := map[string]any{
		"repository":              repository,
		"actions":                 prFeedbackTriggerActions(provider),
		"onlyFactoryPullRequests": true,
	}
	if provider == models.ProviderGitHub {
		triggerConfiguration["ignoreDrafts"] = true
	}
	triggerID := customAutomationTriggerID
	triggerName := "On Pull Request"
	triggerComponent := prFeedbackTriggerComponent(provider)
	triggerIntegration := &yaml.IntegrationRef{ID: integrationID}
	if settings.StartMode == pb.FactoryCustomAutomation_START_MODE_SCHEDULED {
		// A new identity retires the old scheduling chain through canvas publication.
		// Never reuse a deleted identity: its pending requests can still exist.
		triggerID = "schedule-" + uuid.NewString()
		triggerName, triggerComponent, triggerIntegration = "Schedule", "schedule", nil
		triggerConfiguration = map[string]any{
			"type": "cron", "cronExpression": settings.Schedule.GetCronExpression(),
			"timezone": strconv.FormatFloat(settings.Schedule.GetUtcOffset(), 'f', -1, 64),
		}
	}
	runnerConfiguration := customAutomationRunnerConfiguration(
		tx, factory, settings,
		customAutomationCheckoutEnvironment(factory, settings.StartMode, githubHostForFactory(tx, factory)),
	)

	doc := &yaml.Canvas{
		APIVersion: yaml.APIVersion,
		Kind:       yaml.KindCanvas,
		Metadata: &yaml.CanvasMetadata{
			Name:        name,
			Description: "Run ordered prompt and Bash automation steps.",
		},
		Spec: &yaml.CanvasSpec{
			Nodes: []yaml.Node{
				{
					ID:            triggerID,
					Name:          triggerName,
					Type:          yaml.NodeTypeTrigger,
					Component:     triggerComponent,
					Configuration: triggerConfiguration,
					Integration:   triggerIntegration,
					Metadata:      metadata,
					Position:      yaml.Position{X: 120, Y: 240},
				},
				{
					ID:            customAutomationRunnerID,
					Name:          "Verify",
					Type:          yaml.NodeTypeAction,
					Component:     settings.Agent.component(),
					Configuration: runnerConfiguration,
					Position:      yaml.Position{X: 520, Y: 240},
				},
			},
			Edges: []yaml.Edge{
				{SourceID: triggerID, TargetID: customAutomationRunnerID, Channel: "default"},
			},
		},
	}
	if settings.ColumnKey == models.CanvasColumnKeyBacklog {
		if settings.StartMode == pb.FactoryCustomAutomation_START_MODE_SCHEDULED {
			attachBacklogScheduleTask(doc, customAutomationRunnerID, provider, settings.Icon, runnerConfiguration)
			return doc, nil
		}
		trigger := &doc.Spec.Nodes[0]
		trigger.ID = customAutomationTaskTriggerID
		trigger.Name = "On Task"
		trigger.Component = factorycomponent.OnWorkOrderTriggerName
		trigger.Configuration = map[string]any{}
		trigger.Integration = nil
		doc.Spec.Edges[0].SourceID = trigger.ID
		doc.Spec.Nodes[1].Name = "Run automation"
		doc.Metadata.Description = "Run prompt and Bash steps for a Backlog task."
		runnerConfiguration["environment"] = []any{
			checkoutEnvironmentVar("REPO", "{{ $['on-task'].data.workOrder.repository }}"),
			checkoutEnvironmentVar("REPO_URL", "{{ $['on-task'].data.workOrder.repository_url }}"),
			checkoutEnvironmentVar("BASE", "{{ $['on-task'].data.workOrder.default_branch }}"),
		}
	}
	return doc, nil
}

func buildMultiTriggerCustomAutomationCanvas(
	tx *gorm.DB,
	factory *models.Factory,
	name string,
	settings *customAutomationSettings,
	repository, integrationID, provider string,
) (*yaml.Canvas, error) {
	githubHost := githubHostForFactory(tx, factory)
	backlog := settings.ColumnKey == models.CanvasColumnKeyBacklog
	var taskKinds []string
	var taskIndexes []any
	var schedules []customAutomationTrigger
	for _, trigger := range settings.Triggers {
		switch trigger.Mode {
		case pb.FactoryCustomAutomation_START_MODE_MANUAL:
			taskKinds = append(taskKinds, "manual")
			taskIndexes = append(taskIndexes, trigger.Index)
		case pb.FactoryCustomAutomation_START_MODE_AUTOMATIC:
			taskKinds = append(taskKinds, "automatic")
			taskIndexes = append(taskIndexes, trigger.Index)
		default:
			schedules = append(schedules, trigger)
		}
	}
	hasTask := len(taskKinds) > 0
	hasAutomatic := false
	hasManual := false
	for _, kind := range taskKinds {
		if kind == "automatic" {
			hasAutomatic = true
		}
		if kind == "manual" {
			hasManual = true
		}
	}

	nodes := make([]yaml.Node, 0, len(settings.Triggers)+4)
	edges := make([]yaml.Edge, 0, len(settings.Triggers)+3)
	if hasTask {
		taskTriggerID := customAutomationTriggerID
		taskName := "On Pull Request"
		taskComponent := prFeedbackTriggerComponent(provider)
		taskIntegration := &yaml.IntegrationRef{ID: integrationID}
		taskConfiguration := map[string]any{
			"repository":              repository,
			"actions":                 prFeedbackTriggerActions(provider),
			"onlyFactoryPullRequests": true,
		}
		if provider == models.ProviderGitHub {
			taskConfiguration["ignoreDrafts"] = true
		}
		taskEnvironment := customAutomationCheckoutEnvironment(factory, pb.FactoryCustomAutomation_START_MODE_MANUAL, githubHost)
		runnerName := "Verify"
		if backlog {
			taskTriggerID = customAutomationTaskTriggerID
			taskName = "On Task"
			taskComponent = factorycomponent.OnWorkOrderTriggerName
			taskIntegration = nil
			taskConfiguration = map[string]any{}
			taskEnvironment = []any{
				checkoutEnvironmentVar("REPO", "{{ $['on-task'].data.workOrder.repository }}"),
				checkoutEnvironmentVar("REPO_URL", "{{ $['on-task'].data.workOrder.repository_url }}"),
				checkoutEnvironmentVar("BASE", "{{ $['on-task'].data.workOrder.default_branch }}"),
			}
			runnerName = "Run on task"
			if len(schedules) > 0 {
				runnerName = backlogTaskRunnerName
				taskEnvironment = backlogSharedCheckoutEnvironment(factory, githubHost)
			}
		}
		nodes = append(nodes, yaml.Node{
			ID:            taskTriggerID,
			Name:          taskName,
			Type:          yaml.NodeTypeTrigger,
			Component:     taskComponent,
			Configuration: taskConfiguration,
			Integration:   taskIntegration,
			Metadata: customAutomationNodeMetadata(provider, settings.Icon, func(template map[string]any) {
				if hasAutomatic {
					template["startMode"] = "automatic"
				} else {
					template["startMode"] = "manual"
				}
				template["triggerKinds"] = stringsToAny(taskKinds)
				template["triggerIndexes"] = taskIndexes
				if hasManual && hasAutomatic {
					template["manualStart"] = true
				}
			}),
			Position: yaml.Position{X: 120, Y: 240},
		})
		edges = append(edges, yaml.Edge{SourceID: taskTriggerID, TargetID: customAutomationRunnerID, Channel: "default"})
		nodes = append(nodes, customAutomationRunnerNode(
			customAutomationRunnerID, runnerName, settings, customAutomationRunnerConfiguration(tx, factory, settings, taskEnvironment), 240,
		))
	}

	shareRunner := backlog && hasTask && len(schedules) > 0
	scheduleRunnerID := customAutomationRunnerID
	if hasTask && !shareRunner {
		scheduleRunnerID = customAutomationScheduleRunnerID
	}
	if len(schedules) > 0 {
		for offset, schedule := range schedules {
			triggerID := "schedule-" + uuid.NewString()
			nodes = append(nodes, yaml.Node{
				ID:        triggerID,
				Name:      "Schedule",
				Type:      yaml.NodeTypeTrigger,
				Component: "schedule",
				Configuration: map[string]any{
					"type": "cron", "cronExpression": schedule.Schedule.GetCronExpression(),
					"timezone": strconv.FormatFloat(schedule.Schedule.GetUtcOffset(), 'f', -1, 64),
				},
				Metadata: customAutomationNodeMetadata(provider, settings.Icon, func(template map[string]any) {
					template["startMode"] = "scheduled"
					template["triggerIndex"] = schedule.Index
				}),
				Position: yaml.Position{X: 120, Y: 240 + (offset+1)*180},
			})
			edges = append(edges, yaml.Edge{SourceID: triggerID, TargetID: scheduleRunnerID, Channel: "default"})
		}
		if !shareRunner {
			scheduleConfiguration := customAutomationRunnerConfiguration(
				tx, factory, settings,
				customAutomationCheckoutEnvironment(factory, pb.FactoryCustomAutomation_START_MODE_SCHEDULED, githubHost),
			)
			runnerName := "Run schedule"
			if !hasTask {
				runnerName = "Verify"
			}
			if backlog {
				runnerName = backlogTaskRunnerName
			}
			nodes = append(nodes, customAutomationRunnerNode(scheduleRunnerID, runnerName, settings, scheduleConfiguration, 480))
		}
	}

	description := "Run ordered prompt and Bash automation steps."
	if backlog && len(schedules) > 0 && hasTask {
		description = "Run prompt and Bash steps on a schedule or for a Backlog task."
	} else if backlog && len(schedules) > 0 {
		description = "Run prompt and Bash steps on a schedule. The agent creates a task."
	} else if backlog {
		description = "Run prompt and Bash steps for a Backlog task."
	}
	doc := &yaml.Canvas{
		APIVersion: yaml.APIVersion,
		Kind:       yaml.KindCanvas,
		Metadata:   &yaml.CanvasMetadata{Name: name, Description: description},
		Spec:       &yaml.CanvasSpec{Nodes: nodes, Edges: edges},
	}
	if backlog && len(schedules) > 0 {
		for i := range doc.Spec.Nodes {
			if doc.Spec.Nodes[i].ID != scheduleRunnerID {
				continue
			}
			attachBacklogScheduleTask(doc, scheduleRunnerID, provider, settings.Icon, doc.Spec.Nodes[i].Configuration)
			break
		}
	}
	return doc, nil
}

func customAutomationRunnerNode(
	id, name string,
	settings *customAutomationSettings,
	configuration map[string]any,
	y int,
) yaml.Node {
	return yaml.Node{
		ID:            id,
		Name:          name,
		Type:          yaml.NodeTypeAction,
		Component:     settings.Agent.component(),
		Configuration: configuration,
		Position:      yaml.Position{X: 520, Y: y},
	}
}

func customAutomationRunnerConfiguration(
	tx *gorm.DB,
	factory *models.Factory,
	settings *customAutomationSettings,
	environment []any,
) map[string]any {
	environmentFrom := []any{
		map[string]any{
			"source":      runner.CredentialsSourceIntegration,
			"integration": map[string]any{"name": resolveVCSInstallationName(tx, factory)},
		},
	}
	for _, secret := range customAutomationSecrets(settings.Steps) {
		environmentFrom = append(environmentFrom, map[string]any{
			"source": runner.EnvironmentFromSourceSecret,
			"secret": map[string]any{"secret": secret},
		})
	}
	configuration := map[string]any{
		"machineType":             runner.MachineTypeE1LargeAMD64,
		"executionTimeoutSeconds": settings.ExecutionTimeoutSeconds,
		"steps":                   customAutomationStepConfiguration(settings.Steps),
		"environmentFrom":         environmentFrom,
		"environment":             environment,
	}
	if credentials := settings.Agent.credentials(); credentials != nil && settings.Agent.component() != models.SuperPlaneRunnerComponent {
		configuration["credentials"] = credentials
	}
	if model := settings.Agent.model(); model != "" {
		configuration["model"] = model
	}
	settings.Agent.applyLLMProvider(configuration)
	return configuration
}

func customAutomationNodeMetadata(provider, icon string, apply func(map[string]any)) map[string]any {
	metadata := models.FactoryAppTemplateMetadataFor(
		models.FactoryAppTemplateCustomAutomationID,
		customAutomationVersion,
		provider,
	)
	template := metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
	if icon != "" {
		template["icon"] = icon
	}
	apply(template)
	return metadata
}

func stringsToAny(values []string) []any {
	result := make([]any, len(values))
	for i, value := range values {
		result[i] = value
	}
	return result
}

func attachBacklogScheduleTask(doc *yaml.Canvas, runnerID, provider, icon string, runnerConfiguration map[string]any) {
	var runner *yaml.Node
	for i := range doc.Spec.Nodes {
		if doc.Spec.Nodes[i].ID == runnerID {
			runner = &doc.Spec.Nodes[i]
			break
		}
	}
	if runner == nil {
		return
	}
	runner.Name = backlogTaskRunnerName
	runner.Metadata = customAutomationNodeMetadata(provider, icon, func(template map[string]any) {
		template["startMode"] = "scheduled"
	})
	max := 1
	runner.Concurrency = &yaml.ConcurrencySpec{Max: &max}
	runnerConfiguration["steps"] = backlogTaskSteps(runnerConfiguration["steps"])
	if !customAutomationDocHasNode(doc, customAutomationTaskTriggerID) {
		doc.Metadata.Description = "Run prompt and Bash steps on a schedule. The agent creates a task."
	}
}

func backlogTaskSteps(raw any) []any {
	steps, _ := raw.([]any)
	next := make([]any, 0, len(steps)+1)
	lastPrompt := -1
	for i, item := range steps {
		step, _ := item.(map[string]any)
		if step["type"] == runner.AgentStepPrompt || step["type"] == "prompt" {
			lastPrompt = i
		}
		next = append(next, step)
	}
	if lastPrompt >= 0 {
		step := next[lastPrompt].(map[string]any)
		prompt, _ := step["prompt"].(string)
		step["prompt"] = strings.TrimSpace(stripBacklogTaskInstructions(prompt)) + backlogTaskOutputInstructions
	}
	return next
}

func backlogSharedCheckoutEnvironment(factory *models.Factory, githubHost string) []any {
	config := factory.OnboardingConfigValue()
	repository := strings.TrimSpace(config.AppRepository)
	url := runner.WorkspaceContextRepoURL(config.EffectiveVCSProvider(), repository, githubHost)
	env := []any{
		checkoutEnvironmentVar("REPO", fmt.Sprintf(`{{ root().data.workOrder?.repository ?? "%s" }}`, celString(repository))),
		checkoutEnvironmentVar("REPO_URL", fmt.Sprintf(`{{ root().data.workOrder?.repository_url ?? "%s" }}`, celString(url))),
		checkoutEnvironmentVar("SOURCE_REPO", repository),
	}
	if branch := strings.TrimSpace(config.DefaultBranch); branch != "" {
		env = append(env, checkoutEnvironmentVar("BASE", fmt.Sprintf(`{{ root().data.workOrder?.default_branch ?? "%s" }}`, celString(branch))))
	}
	return env
}

func celString(value string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value)
}

func stripBacklogTaskInstructions(prompt string) string {
	for _, suffix := range []string{backlogTaskOutputInstructions, backlogTaskProposalInstructions} {
		prompt = strings.TrimSuffix(prompt, suffix)
	}
	return prompt
}

func customAutomationCheckoutEnvironment(factory *models.Factory, mode pb.FactoryCustomAutomation_StartMode, githubHost string) []any {
	config := factory.OnboardingConfigValue()
	repository := strings.TrimSpace(config.AppRepository)
	url := runner.WorkspaceContextRepoURL(config.EffectiveVCSProvider(), repository, githubHost)
	env := []any{
		checkoutEnvironmentVar("REPO", repository),
		checkoutEnvironmentVar("REPO_URL", url),
		checkoutEnvironmentVar("SOURCE_REPO", repository),
	}
	if mode != pb.FactoryCustomAutomation_START_MODE_SCHEDULED {
		env = append(env, checkoutEnvironmentVar("PR_NUMBER", prFeedbackPRNumberExpression()),
			checkoutEnvironmentVar("PR_HEAD", prFeedbackPRHeadExpression()))
	}
	if branch := strings.TrimSpace(config.DefaultBranch); branch != "" {
		env = append(env, checkoutEnvironmentVar("BASE", branch))
	}
	return env
}

func checkoutEnvironmentVar(name, value string) map[string]any {
	return map[string]any{
		"name":        name,
		"value":       value,
		"valueSource": runner.EnvironmentValueSourceLiteral,
	}
}

func customAutomationStepConfiguration(steps []runner.AgentStep) []any {
	values := make([]any, 0, len(steps))
	for _, step := range steps {
		value := map[string]any{
			"name":             step.Name,
			"type":             step.Type,
			"workingDirectory": step.WorkingDirectory,
		}
		if step.Prompt != nil {
			value["prompt"] = *step.Prompt
		}
		if step.Command != nil {
			value["command"] = *step.Command
		}
		if len(step.Secrets) > 0 {
			secrets := make([]any, 0, len(step.Secrets))
			for _, secret := range step.Secrets {
				secrets = append(secrets, secret)
			}
			value["secrets"] = secrets
		}
		values = append(values, value)
	}
	return values
}

func customAutomationSecrets(steps []runner.AgentStep) []string {
	seen := map[string]struct{}{}
	var result []string
	for _, step := range steps {
		for _, secret := range step.Secrets {
			name := strings.TrimSpace(secret)
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			result = append(result, name)
		}
	}
	return result
}

func findCustomAutomationCanvas(tx *gorm.DB, factory *models.Factory, rawID string) (*models.Canvas, error) {
	id, err := uuid.Parse(strings.TrimSpace(rawID))
	if err != nil {
		return nil, invalidArgument("automation id must be a UUID")
	}
	canvas, err := models.FindCanvasInTransaction(tx, factory.OrganizationID, id)
	if err != nil {
		return nil, err
	}
	if canvas.FactoryID == nil || *canvas.FactoryID != factory.ID {
		return nil, gorm.ErrRecordNotFound
	}
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
	if err != nil {
		return nil, err
	}
	if models.FactoryAppTemplateID(version.Nodes) != models.FactoryAppTemplateCustomAutomationID {
		return nil, invalidArgument("automation is not a custom automation")
	}
	return canvas, nil
}

func preserveCustomAutomationNodeIdentities(tx *gorm.DB, canvas *models.Canvas, doc *yaml.Canvas) error {
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
	if err != nil {
		return err
	}
	for i := range version.Nodes {
		if version.Nodes[i].Type == models.NodeTypeTrigger {
			preserveCustomAutomationPRTriggerIdentity(&version.Nodes[i], doc)
		}
	}
	// The task agent and the schedule agent are different steps. Saved node
	// order is not stable, so the schedule agent must not supply the task id.
	current := customAutomationAgentForRunner(version.Nodes, customAutomationRunnerID)
	if current == nil {
		if !customAutomationHasAgent(version.Nodes) {
			return errors.New("custom automation agent is missing")
		}
		return nil
	}
	for i := range doc.Spec.Nodes {
		node := &doc.Spec.Nodes[i]
		if node.ID != customAutomationRunnerID {
			continue
		}
		targetID := current.ID
		if current.ComponentName() != node.Component {
			targetID = customAutomationRunnerID + "-" + strings.ToLower(strings.ReplaceAll(node.Component, ".", "-"))
		}
		if targetID != node.ID && customAutomationDocHasNode(doc, targetID) {
			return nil
		}
		node.ID = targetID
		for j := range doc.Spec.Edges {
			if doc.Spec.Edges[j].TargetID == customAutomationRunnerID {
				doc.Spec.Edges[j].TargetID = targetID
			}
			if doc.Spec.Edges[j].SourceID == customAutomationRunnerID {
				doc.Spec.Edges[j].SourceID = targetID
			}
		}
		return nil
	}
	return errors.New("custom automation agent is missing")
}

func customAutomationAgentForRunner(nodes []models.Node, runnerID string) *models.Node {
	var prefixed *models.Node
	prefix := runnerID + "-"
	for i := range nodes {
		node := &nodes[i]
		if !isFactoryAgentHarness(node.ComponentName()) {
			continue
		}
		if node.ID == runnerID {
			return node
		}
		if strings.HasPrefix(node.ID, prefix) {
			prefixed = node
		}
	}
	return prefixed
}

func customAutomationHasAgent(nodes []models.Node) bool {
	for i := range nodes {
		if isFactoryAgentHarness(nodes[i].ComponentName()) {
			return true
		}
	}
	return false
}

func customAutomationDocHasNode(doc *yaml.Canvas, id string) bool {
	for i := range doc.Spec.Nodes {
		if doc.Spec.Nodes[i].ID == id {
			return true
		}
	}
	return false
}

func preserveCustomAutomationPRTriggerIdentity(previous *models.Node, doc *yaml.Canvas) {
	for i := range doc.Spec.Nodes {
		trigger := &doc.Spec.Nodes[i]
		if trigger.ID != customAutomationTriggerID {
			continue
		}
		trigger.ID = previous.ID
		if previous.ComponentName() != trigger.Component {
			trigger.ID = customAutomationTriggerID + "-" + uuid.NewString()
		}
		for j := range doc.Spec.Edges {
			if doc.Spec.Edges[j].SourceID == customAutomationTriggerID {
				doc.Spec.Edges[j].SourceID = trigger.ID
			}
		}
		return
	}
}

type decodedCustomAutomationTrigger struct {
	index    int
	mode     pb.FactoryCustomAutomation_StartMode
	schedule *pb.FactoryCustomAutomation_Schedule
}

func decodeCustomAutomationTriggers(nodes []models.Node) ([]decodedCustomAutomationTrigger, error) {
	decoded := make([]decodedCustomAutomationTrigger, 0, 2)
	for i := range nodes {
		node := &nodes[i]
		if node.Type != models.NodeTypeTrigger {
			continue
		}
		template, _ := node.Metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
		mode, _ := template["startMode"].(string)
		if node.ComponentName() == "schedule" || mode == "scheduled" {
			offset, err := strconv.ParseFloat(configString(node.Configuration, "timezone"), 64)
			if err != nil {
				return nil, fmt.Errorf("decode schedule UTC offset: %w", err)
			}
			decoded = append(decoded, decodedCustomAutomationTrigger{
				index: configInt(template, "triggerIndex"),
				mode:  pb.FactoryCustomAutomation_START_MODE_SCHEDULED,
				schedule: &pb.FactoryCustomAutomation_Schedule{
					CronExpression: configString(node.Configuration, "cronExpression"),
					UtcOffset:      offset,
				},
			})
			continue
		}
		kinds := metadataStrings(template, "triggerKinds")
		indexes := metadataInts(template, "triggerIndexes")
		if len(kinds) > 0 {
			for position, kind := range kinds {
				index := position
				if position < len(indexes) {
					index = indexes[position]
				}
				decoded = append(decoded, decodedCustomAutomationTrigger{
					index: index, mode: customAutomationStartMode(kind),
				})
			}
			continue
		}
		decoded = append(decoded, decodedCustomAutomationTrigger{
			index: configInt(template, "triggerIndex"),
			mode:  customAutomationStartMode(mode),
		})
	}
	sort.SliceStable(decoded, func(i, j int) bool { return decoded[i].index < decoded[j].index })
	return decoded, nil
}

func selectCustomAutomationAgent(nodes []models.Node) *models.Node {
	var fallback *models.Node
	for i := range nodes {
		node := &nodes[i]
		if !isFactoryAgentHarness(node.ComponentName()) {
			continue
		}
		if fallback == nil {
			fallback = node
		}
		steps, _ := node.Configuration["steps"].([]any)
		managed := false
		for _, step := range steps {
			value, _ := step.(map[string]any)
			name, _ := value["name"].(string)
			stepType, _ := value["type"].(string)
			command, _ := value["command"].(string)
			if managedBacklogTaskResultStep(name, stepType, command) {
				managed = true
			}
		}
		if !managed {
			return node
		}
	}
	return fallback
}

func metadataStrings(template map[string]any, key string) []string {
	switch raw := template[key].(type) {
	case []string:
		return raw
	case []any:
		values := make([]string, 0, len(raw))
		for _, item := range raw {
			value, ok := item.(string)
			if !ok {
				continue
			}
			values = append(values, value)
		}
		return values
	default:
		return nil
	}
}

func metadataInts(template map[string]any, key string) []int {
	switch raw := template[key].(type) {
	case []int:
		return raw
	case []any:
		values := make([]int, 0, len(raw))
		for _, item := range raw {
			values = append(values, configInt(map[string]any{"value": item}, "value"))
		}
		return values
	default:
		return nil
	}
}

func serializeCustomAutomation(
	tx *gorm.DB,
	factory *models.Factory,
	canvas *models.Canvas,
) (*pb.FactoryCustomAutomation, error) {
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
	if err != nil {
		return nil, err
	}
	triggers, err := decodeCustomAutomationTriggers(version.Nodes)
	if err != nil {
		return nil, err
	}
	agent := selectCustomAutomationAgent(version.Nodes)
	if len(triggers) == 0 || agent == nil || models.FactoryAppTemplateID(version.Nodes) != models.FactoryAppTemplateCustomAutomationID {
		return nil, errors.New("custom automation graph is incomplete")
	}

	steps, err := customAutomationProtoSteps(agent.Configuration["steps"])
	if err != nil {
		return nil, err
	}
	icon := ""
	for i := range version.Nodes {
		if version.Nodes[i].Type != models.NodeTypeTrigger {
			continue
		}
		template, _ := version.Nodes[i].Metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
		icon = configString(template, "icon")
		if icon != "" {
			break
		}
	}
	settings := &pb.FactoryCustomAutomation_Settings{
		Agent: &pb.FactoryAppTemplateAgent{
			Component: agent.ComponentName(),
			Model:     configString(agent.Configuration, "model"),
		},
		Steps:                   steps,
		Icon:                    icon,
		ExecutionTimeoutSeconds: int32(configInt(agent.Configuration, "executionTimeoutSeconds")),
	}
	if len(triggers) == 1 {
		settings.StartMode = triggers[0].mode
		settings.Schedule = triggers[0].schedule
	} else {
		settings.StartMode = pb.FactoryCustomAutomation_START_MODE_MANUAL
		settings.Triggers = make([]*pb.FactoryCustomAutomation_StartTrigger, 0, len(triggers))
		scheduled := 0
		for _, trigger := range triggers {
			settings.Triggers = append(settings.Triggers, &pb.FactoryCustomAutomation_StartTrigger{
				StartMode: trigger.mode, Schedule: trigger.schedule,
			})
			switch trigger.mode {
			case pb.FactoryCustomAutomation_START_MODE_AUTOMATIC:
				settings.StartMode = pb.FactoryCustomAutomation_START_MODE_AUTOMATIC
			case pb.FactoryCustomAutomation_START_MODE_SCHEDULED:
				scheduled++
				if settings.StartMode != pb.FactoryCustomAutomation_START_MODE_AUTOMATIC {
					settings.StartMode = pb.FactoryCustomAutomation_START_MODE_SCHEDULED
				}
				if scheduled == 1 {
					settings.Schedule = trigger.schedule
				}
			}
		}
		if scheduled != 1 {
			settings.Schedule = nil
		}
	}
	if credentials, ok := agent.Configuration["credentials"].(map[string]any); ok {
		settings.Agent.CredentialSource = configString(credentials, "source")
		if integration, ok := credentials["integration"].(map[string]any); ok {
			settings.Agent.CredentialIntegrationName = configString(integration, "name")
		}
	}
	automation := serializeFactoryAutomation(*canvas)
	automation.TemplateId = models.FactoryAppTemplateCustomAutomationID
	return &pb.FactoryCustomAutomation{
		Automation: automation,
		Settings:   settings,
		Provider:   factory.OnboardingConfigValue().EffectiveVCSProvider(),
	}, nil
}

func customAutomationProtoSteps(raw any) ([]*pb.FactoryCustomAutomation_Step, error) {
	var steps []runner.AgentStep
	if err := mapstructure.Decode(raw, &steps); err != nil {
		return nil, fmt.Errorf("decode custom automation steps: %w", err)
	}
	result := make([]*pb.FactoryCustomAutomation_Step, 0, len(steps))
	for _, step := range steps {
		value := &pb.FactoryCustomAutomation_Step{
			Name:             step.Name,
			WorkingDirectory: step.WorkingDirectory,
			Secrets:          step.Secrets,
		}
		command := ""
		if step.Command != nil {
			command = *step.Command
		}
		if managedBacklogTaskResultStep(step.Name, step.Type, command) {
			continue
		}
		if runner.NormalizeAgentStepType(step.Type) == runner.AgentStepBash {
			value.Type = pb.FactoryCustomAutomation_Step_TYPE_BASH
			if step.Command != nil {
				value.Command = *step.Command
			}
		} else {
			value.Type = pb.FactoryCustomAutomation_Step_TYPE_PROMPT
			if step.Prompt != nil {
				value.Prompt = strings.TrimSpace(stripBacklogTaskInstructions(*step.Prompt))
			}
		}
		result = append(result, value)
	}
	return result, nil
}

func managedBacklogTaskResultStep(name, stepType, command string) bool {
	return name == backlogTaskResultStepName &&
		runner.NormalizeAgentStepType(stepType) == runner.AgentStepBash &&
		strings.Contains(command, "/tmp/proposal.json")
}

func configInt(configuration map[string]any, key string) int {
	switch value := configuration[key].(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func customAutomationStartModeName(mode pb.FactoryCustomAutomation_StartMode) string {
	if mode == pb.FactoryCustomAutomation_START_MODE_SCHEDULED {
		return "scheduled"
	}
	if mode == pb.FactoryCustomAutomation_START_MODE_AUTOMATIC {
		return "automatic"
	}
	return "manual"
}

func customAutomationStartMode(mode string) pb.FactoryCustomAutomation_StartMode {
	if strings.TrimSpace(mode) == "scheduled" {
		return pb.FactoryCustomAutomation_START_MODE_SCHEDULED
	}
	if strings.TrimSpace(mode) == "automatic" {
		return pb.FactoryCustomAutomation_START_MODE_AUTOMATIC
	}
	return pb.FactoryCustomAutomation_START_MODE_MANUAL
}
