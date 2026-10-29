package factories

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/authentication"
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
	customAutomationDefaultName = "Custom automation"
	customAutomationTriggerID   = "on-pull-request"
	customAutomationRunnerID    = "verify"
	customAutomationVersion     = 1
)

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
	if err := canvas.SetColumnKey(db, models.CanvasColumnKeyVerify); err != nil {
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
	doc, err := buildCustomAutomationCanvas(db, factory, name, settings)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update custom automation")
	}
	if err := preserveCustomAutomationRunnerIdentity(db, canvas, doc); err != nil {
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
		return canvases.UpdateCanvasInTransaction(tx, canvas, &name, nil, nil)
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update custom automation")
	}
	canvases.PublishCanvasUpdated(canvas)

	automation, err := serializeCustomAutomation(db, factory, canvas)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update custom automation")
	}
	return &pb.UpdateFactoryCustomAutomationResponse{CustomAutomation: automation}, nil
}

type customAutomationSettings struct {
	StartMode               pb.FactoryCustomAutomation_StartMode
	Agent                   *intakeAgent
	Steps                   []runner.AgentStep
	Icon                    string
	ExecutionTimeoutSeconds int
}

func parseCustomAutomationSettings(
	tx *gorm.DB,
	factory *models.Factory,
	input *pb.FactoryCustomAutomation_Settings,
) (*customAutomationSettings, error) {
	if input == nil {
		return nil, invalidArgument("settings are required")
	}
	if input.GetStartMode() != pb.FactoryCustomAutomation_START_MODE_MANUAL &&
		input.GetStartMode() != pb.FactoryCustomAutomation_START_MODE_AUTOMATIC {
		return nil, invalidArgument("start mode must be manual or automatic")
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
		StartMode:               input.GetStartMode(),
		Agent:                   agent,
		Steps:                   steps,
		Icon:                    strings.TrimSpace(input.GetIcon()),
		ExecutionTimeoutSeconds: timeout,
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
	}
	if agent.Credentials != nil {
		return agent, nil
	}
	workspace := resolveIntakeAgent(tx, factory)
	if workspace == nil || workspace.Component != agent.Component {
		return agent, nil
	}
	agent.Credentials = workspace.Credentials
	agent.LLMProvider = workspace.LLMProvider
	if agent.Model == "" {
		agent.Model = workspace.Model
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
	runnerConfiguration := map[string]any{
		"machineType":             runner.MachineTypeE1LargeAMD64,
		"executionTimeoutSeconds": settings.ExecutionTimeoutSeconds,
		"steps":                   customAutomationStepConfiguration(settings.Steps),
		"environmentFrom":         environmentFrom,
		"environment":             customAutomationCheckoutEnvironment(factory),
	}
	if credentials := settings.Agent.credentials(); credentials != nil && settings.Agent.component() != models.SuperPlaneRunnerComponent {
		runnerConfiguration["credentials"] = credentials
	}
	if model := settings.Agent.model(); model != "" {
		runnerConfiguration["model"] = model
	}
	settings.Agent.applyLLMProvider(runnerConfiguration)

	return &yaml.Canvas{
		APIVersion: yaml.APIVersion,
		Kind:       yaml.KindCanvas,
		Metadata: &yaml.CanvasMetadata{
			Name:        name,
			Description: "Run a focused prompt and Bash automation for a task pull request.",
		},
		Spec: &yaml.CanvasSpec{
			Nodes: []yaml.Node{
				{
					ID:            customAutomationTriggerID,
					Name:          "On Pull Request",
					Type:          yaml.NodeTypeTrigger,
					Component:     prFeedbackTriggerComponent(provider),
					Configuration: triggerConfiguration,
					Integration:   &yaml.IntegrationRef{ID: integrationID},
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
				{SourceID: customAutomationTriggerID, TargetID: customAutomationRunnerID, Channel: "default"},
			},
		},
	}, nil
}

func customAutomationCheckoutEnvironment(factory *models.Factory) []any {
	config := factory.OnboardingConfigValue()
	repository := strings.TrimSpace(config.AppRepository)
	url := runner.WorkspaceContextRepoURL(config.EffectiveVCSProvider(), repository)
	env := []any{
		checkoutEnvironmentVar("REPO", repository),
		checkoutEnvironmentVar("REPO_URL", url),
		checkoutEnvironmentVar("SOURCE_REPO", repository),
		checkoutEnvironmentVar("PR_NUMBER", prFeedbackPRNumberExpression()),
		checkoutEnvironmentVar("PR_HEAD", prFeedbackPRHeadExpression()),
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

func preserveCustomAutomationRunnerIdentity(tx *gorm.DB, canvas *models.Canvas, doc *yaml.Canvas) error {
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
	if err != nil {
		return err
	}
	var current *models.Node
	for i := range version.Nodes {
		if isFactoryAgentHarness(version.Nodes[i].ComponentName()) {
			current = &version.Nodes[i]
			break
		}
	}
	if current == nil {
		return errors.New("custom automation agent is missing")
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
		node.ID = targetID
		for j := range doc.Spec.Edges {
			if doc.Spec.Edges[j].TargetID == customAutomationRunnerID {
				doc.Spec.Edges[j].TargetID = targetID
			}
		}
		return nil
	}
	return errors.New("custom automation agent is missing")
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
	var trigger *models.Node
	var agent *models.Node
	for i := range version.Nodes {
		node := &version.Nodes[i]
		if node.ID == customAutomationTriggerID {
			trigger = node
		}
		if isFactoryAgentHarness(node.ComponentName()) {
			agent = node
		}
	}
	if trigger == nil || agent == nil || models.FactoryAppTemplateID(version.Nodes) != models.FactoryAppTemplateCustomAutomationID {
		return nil, errors.New("custom automation graph is incomplete")
	}

	steps, err := customAutomationProtoSteps(agent.Configuration["steps"])
	if err != nil {
		return nil, err
	}
	template, _ := trigger.Metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
	mode, _ := template["startMode"].(string)
	settings := &pb.FactoryCustomAutomation_Settings{
		StartMode: customAutomationStartMode(mode),
		Agent: &pb.FactoryAppTemplateAgent{
			Component: agent.ComponentName(),
			Model:     configString(agent.Configuration, "model"),
		},
		Steps:                   steps,
		Icon:                    configString(template, "icon"),
		ExecutionTimeoutSeconds: int32(configInt(agent.Configuration, "executionTimeoutSeconds")),
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
		if runner.NormalizeAgentStepType(step.Type) == runner.AgentStepBash {
			value.Type = pb.FactoryCustomAutomation_Step_TYPE_BASH
			if step.Command != nil {
				value.Command = *step.Command
			}
		} else {
			value.Type = pb.FactoryCustomAutomation_Step_TYPE_PROMPT
			if step.Prompt != nil {
				value.Prompt = *step.Prompt
			}
		}
		result = append(result, value)
	}
	return result, nil
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
	if mode == pb.FactoryCustomAutomation_START_MODE_AUTOMATIC {
		return "automatic"
	}
	return "manual"
}

func customAutomationStartMode(mode string) pb.FactoryCustomAutomation_StartMode {
	if strings.TrimSpace(mode) == "automatic" {
		return pb.FactoryCustomAutomation_START_MODE_AUTOMATIC
	}
	return pb.FactoryCustomAutomation_START_MODE_MANUAL
}
