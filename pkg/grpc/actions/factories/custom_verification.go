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
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	customVerificationDefaultName = "Custom verification"
	customVerificationTriggerID   = "on-pull-request"
	customVerificationRunnerID    = "verify"
	customVerificationVersion     = 1
)

func CreateFactoryCustomVerification(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.CreateFactoryCustomVerificationRequest,
) (*pb.CreateFactoryCustomVerificationResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create custom verification")
	}

	db := database.DB(ctx)
	organization, err := models.FindOrganizationByIDInTransaction(db, orgID.String())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create custom verification")
	}
	if !organization.HasExperimentalFeature(features.FeatureFactoryCustomAutomations) {
		return nil, factoryErrorToStatus(errCustomAutomationsDisabled, "failed to create custom verification")
	}
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create custom verification")
	}

	settings, err := parseCustomVerificationSettings(db, factory, req.GetSettings())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create custom verification")
	}
	name := strings.TrimSpace(req.GetName())
	if name == "" {
		name = customVerificationDefaultName
	}
	factoryID := factory.ID
	name, err = models.AvailableCanvasName(db, orgID, &factoryID, name)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create custom verification")
	}

	doc, err := buildCustomVerificationCanvas(db, factory, name, settings)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create custom verification")
	}
	nodes, edges, err := doc.Parse(deps.Registry, orgID.String())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to build custom verification")
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
		return nil, factoryErrorToStatus(err, "failed to create custom verification")
	}
	canvas, err := models.FindCanvasInTransaction(db, orgID, canvasID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create custom verification")
	}
	if err := canvas.SetColumnKey(db, models.CanvasColumnKeyVerify); err != nil {
		discardIntakeCanvas(db, orgID, canvasID)
		return nil, factoryErrorToStatus(err, "failed to create custom verification")
	}

	verification, err := serializeCustomVerification(db, factory, canvas)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create custom verification")
	}
	return &pb.CreateFactoryCustomVerificationResponse{Verification: verification}, nil
}

func DescribeFactoryCustomVerification(
	ctx context.Context,
	organizationID string,
	req *pb.DescribeFactoryCustomVerificationRequest,
) (*pb.DescribeFactoryCustomVerificationResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe custom verification")
	}
	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe custom verification")
	}
	canvas, err := findCustomVerificationCanvas(db, factory, req.GetAutomationId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe custom verification")
	}
	verification, err := serializeCustomVerification(db, factory, canvas)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe custom verification")
	}
	return &pb.DescribeFactoryCustomVerificationResponse{Verification: verification}, nil
}

func UpdateFactoryCustomVerification(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.UpdateFactoryCustomVerificationRequest,
) (*pb.UpdateFactoryCustomVerificationResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update custom verification")
	}
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	ownerID, err := uuid.Parse(userID)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to update custom verification")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update custom verification")
	}
	canvas, err := findCustomVerificationCanvas(db, factory, req.GetAutomationId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update custom verification")
	}
	settings, err := parseCustomVerificationSettings(db, factory, req.GetSettings())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update custom verification")
	}
	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return nil, factoryErrorToStatus(invalidArgument("name is required"), "failed to update custom verification")
	}
	doc, err := buildCustomVerificationCanvas(db, factory, name, settings)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update custom verification")
	}
	if err := preserveCustomVerificationRunnerIdentity(db, canvas, doc); err != nil {
		return nil, factoryErrorToStatus(err, "failed to update custom verification")
	}
	nodes, edges, err := doc.ParseInTransaction(deps.Registry, orgID.String(), db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to build custom verification")
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := canvases.PublishGeneratedCanvasNodes(
			ctx,
			tx,
			canvas,
			ownerID,
			"Update custom verification",
			nodes,
			edges,
			intakePublisherOptions(deps, orgID),
		); err != nil {
			return err
		}
		return canvases.UpdateCanvasInTransaction(tx, canvas, &name, nil, nil)
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update custom verification")
	}
	canvases.PublishCanvasUpdated(canvas)

	verification, err := serializeCustomVerification(db, factory, canvas)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update custom verification")
	}
	return &pb.UpdateFactoryCustomVerificationResponse{Verification: verification}, nil
}

type customVerificationSettings struct {
	StartMode pb.FactoryCustomVerification_StartMode
	Agent     *intakeAgent
	Steps     []runner.AgentStep
	Icon      string
}

func parseCustomVerificationSettings(
	tx *gorm.DB,
	factory *models.Factory,
	input *pb.FactoryCustomVerification_Settings,
) (*customVerificationSettings, error) {
	if input == nil {
		return nil, invalidArgument("settings are required")
	}
	if input.GetStartMode() != pb.FactoryCustomVerification_START_MODE_MANUAL &&
		input.GetStartMode() != pb.FactoryCustomVerification_START_MODE_AUTOMATIC {
		return nil, invalidArgument("start mode must be manual or automatic")
	}

	agent, err := customVerificationAgent(tx, factory, input.GetAgent())
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
		case pb.FactoryCustomVerification_Step_TYPE_PROMPT:
			parsed.Type = runner.AgentStepPrompt
			value := step.GetPrompt()
			parsed.Prompt = &value
		case pb.FactoryCustomVerification_Step_TYPE_BASH:
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
	return &customVerificationSettings{
		StartMode: input.GetStartMode(),
		Agent:     agent,
		Steps:     steps,
		Icon:      strings.TrimSpace(input.GetIcon()),
	}, nil
}

func customVerificationAgent(tx *gorm.DB, factory *models.Factory, input *pb.FactoryAppTemplateAgent) (*intakeAgent, error) {
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
	default:
		return nil, invalidArgument("agent credential source is not supported")
	}
	if source == runner.CredentialsSourceIntegration && integrationName == "" {
		return nil, invalidArgument("agent credential integration name is required")
	}
	if source != "" {
		agent.Credentials = map[string]any{"source": source}
		if integrationName != "" {
			agent.Credentials["integration"] = map[string]any{"name": integrationName}
		}
	}
	return agent, nil
}

func buildCustomVerificationCanvas(
	tx *gorm.DB,
	factory *models.Factory,
	name string,
	settings *customVerificationSettings,
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
	mode := customVerificationStartModeName(settings.StartMode)
	metadata := models.FactoryAppTemplateMetadataFor(
		models.FactoryAppTemplateCustomVerifyID,
		customVerificationVersion,
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
	for _, secret := range customVerificationSecrets(settings.Steps) {
		environmentFrom = append(environmentFrom, map[string]any{
			"source": runner.EnvironmentFromSourceSecret,
			"secret": map[string]any{"secret": secret},
		})
	}
	runnerConfiguration := map[string]any{
		"machineType":             runner.MachineTypeE1LargeAMD64,
		"executionTimeoutSeconds": prFeedbackTimeoutSeconds,
		"steps":                   customVerificationStepConfiguration(settings.Steps),
		"environmentFrom":         environmentFrom,
		"environment":             customVerificationCheckoutEnvironment(factory),
	}
	if credentials := settings.Agent.credentials(); credentials != nil {
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
			Description: "Run a focused prompt and bash verification for a task pull request.",
		},
		Spec: &yaml.CanvasSpec{
			Nodes: []yaml.Node{
				{
					ID:            customVerificationTriggerID,
					Name:          "On Pull Request",
					Type:          yaml.NodeTypeTrigger,
					Component:     prFeedbackTriggerComponent(provider),
					Configuration: triggerConfiguration,
					Integration:   &yaml.IntegrationRef{ID: integrationID},
					Metadata:      metadata,
					Position:      yaml.Position{X: 120, Y: 240},
				},
				{
					ID:            customVerificationRunnerID,
					Name:          "Verify",
					Type:          yaml.NodeTypeAction,
					Component:     settings.Agent.component(),
					Configuration: runnerConfiguration,
					Position:      yaml.Position{X: 520, Y: 240},
				},
			},
			Edges: []yaml.Edge{
				{SourceID: customVerificationTriggerID, TargetID: customVerificationRunnerID, Channel: "default"},
			},
		},
	}, nil
}

func customVerificationCheckoutEnvironment(factory *models.Factory) []any {
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

func applyCustomVerificationCheckoutEnvironment(configuration map[string]any, factory *models.Factory) {
	wanted := customVerificationCheckoutEnvironment(factory)
	existing, _ := configuration["environment"].([]any)
	replaced := make(map[string]struct{}, len(wanted))
	for i, raw := range existing {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		name, _ := entry["name"].(string)
		for _, item := range wanted {
			want := item.(map[string]any)
			if want["name"] != name {
				continue
			}
			existing[i] = want
			replaced[name] = struct{}{}
			break
		}
	}
	for _, item := range wanted {
		name := item.(map[string]any)["name"].(string)
		if _, ok := replaced[name]; ok {
			continue
		}
		existing = append(existing, item)
	}
	configuration["environment"] = existing
}

func ensureCustomVerificationCheckoutEnvironment(tx *gorm.DB, factory *models.Factory, canvas *models.Canvas) error {
	node, err := findCustomVerificationAgentNode(tx, canvas)
	if err != nil {
		return err
	}
	configuration := node.Configuration.Data()
	if configuration == nil {
		configuration = map[string]any{}
	}
	applyCustomVerificationCheckoutEnvironment(configuration, factory)
	return tx.Model(node).Update("configuration", datatypes.NewJSONType(configuration)).Error
}

func findCustomVerificationAgentNode(tx *gorm.DB, canvas *models.Canvas) (*models.CanvasNode, error) {
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
	if err != nil {
		return nil, err
	}
	for i := range version.Nodes {
		node := version.Nodes[i]
		if !isFactoryAgentHarness(node.ComponentName()) {
			continue
		}
		return models.FindCanvasNode(tx, canvas.ID, node.ID)
	}
	return nil, errors.New("custom verification agent is missing")
}

func customVerificationStepConfiguration(steps []runner.AgentStep) []any {
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

func customVerificationSecrets(steps []runner.AgentStep) []string {
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

func findCustomVerificationCanvas(tx *gorm.DB, factory *models.Factory, rawID string) (*models.Canvas, error) {
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
	if models.FactoryAppTemplateID(version.Nodes) != models.FactoryAppTemplateCustomVerifyID {
		return nil, invalidArgument("automation is not a custom verification")
	}
	return canvas, nil
}

func preserveCustomVerificationRunnerIdentity(tx *gorm.DB, canvas *models.Canvas, doc *yaml.Canvas) error {
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
		return errors.New("custom verification agent is missing")
	}
	for i := range doc.Spec.Nodes {
		node := &doc.Spec.Nodes[i]
		if node.ID != customVerificationRunnerID {
			continue
		}
		targetID := current.ID
		if current.ComponentName() != node.Component {
			targetID = customVerificationRunnerID + "-" + strings.ToLower(strings.ReplaceAll(node.Component, ".", "-"))
		}
		node.ID = targetID
		for j := range doc.Spec.Edges {
			if doc.Spec.Edges[j].TargetID == customVerificationRunnerID {
				doc.Spec.Edges[j].TargetID = targetID
			}
		}
		return nil
	}
	return errors.New("custom verification agent is missing")
}

func serializeCustomVerification(
	tx *gorm.DB,
	factory *models.Factory,
	canvas *models.Canvas,
) (*pb.FactoryCustomVerification, error) {
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
	if err != nil {
		return nil, err
	}
	var trigger *models.Node
	var agent *models.Node
	for i := range version.Nodes {
		node := &version.Nodes[i]
		if node.ID == customVerificationTriggerID {
			trigger = node
		}
		if isFactoryAgentHarness(node.ComponentName()) {
			agent = node
		}
	}
	if trigger == nil || agent == nil || models.FactoryAppTemplateID(version.Nodes) != models.FactoryAppTemplateCustomVerifyID {
		return nil, errors.New("custom verification graph is incomplete")
	}

	steps, err := customVerificationProtoSteps(agent.Configuration["steps"])
	if err != nil {
		return nil, err
	}
	template, _ := trigger.Metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
	mode, _ := template["startMode"].(string)
	settings := &pb.FactoryCustomVerification_Settings{
		StartMode: customVerificationStartMode(mode),
		Agent: &pb.FactoryAppTemplateAgent{
			Component: agent.ComponentName(),
			Model:     configString(agent.Configuration, "model"),
		},
		Steps: steps,
		Icon:  configString(template, "icon"),
	}
	if credentials, ok := agent.Configuration["credentials"].(map[string]any); ok {
		settings.Agent.CredentialSource = configString(credentials, "source")
		if integration, ok := credentials["integration"].(map[string]any); ok {
			settings.Agent.CredentialIntegrationName = configString(integration, "name")
		}
	}
	automation := serializeFactoryAutomation(*canvas)
	automation.TemplateId = models.FactoryAppTemplateCustomVerifyID
	return &pb.FactoryCustomVerification{
		Automation: automation,
		Settings:   settings,
		Provider:   factory.OnboardingConfigValue().EffectiveVCSProvider(),
	}, nil
}

func customVerificationProtoSteps(raw any) ([]*pb.FactoryCustomVerification_Step, error) {
	var steps []runner.AgentStep
	if err := mapstructure.Decode(raw, &steps); err != nil {
		return nil, fmt.Errorf("decode custom verification steps: %w", err)
	}
	result := make([]*pb.FactoryCustomVerification_Step, 0, len(steps))
	for _, step := range steps {
		value := &pb.FactoryCustomVerification_Step{
			Name:             step.Name,
			WorkingDirectory: step.WorkingDirectory,
			Secrets:          step.Secrets,
		}
		if runner.NormalizeAgentStepType(step.Type) == runner.AgentStepBash {
			value.Type = pb.FactoryCustomVerification_Step_TYPE_BASH
			if step.Command != nil {
				value.Command = *step.Command
			}
		} else {
			value.Type = pb.FactoryCustomVerification_Step_TYPE_PROMPT
			if step.Prompt != nil {
				value.Prompt = *step.Prompt
			}
		}
		result = append(result, value)
	}
	return result, nil
}

func customVerificationStartModeName(mode pb.FactoryCustomVerification_StartMode) string {
	if mode == pb.FactoryCustomVerification_START_MODE_AUTOMATIC {
		return "automatic"
	}
	return "manual"
}

func customVerificationStartMode(mode string) pb.FactoryCustomVerification_StartMode {
	if strings.TrimSpace(mode) == "automatic" {
		return pb.FactoryCustomVerification_START_MODE_AUTOMATIC
	}
	return pb.FactoryCustomVerification_START_MODE_MANUAL
}
