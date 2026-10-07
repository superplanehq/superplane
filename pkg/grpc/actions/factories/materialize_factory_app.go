package factories

import (
	"context"
	"errors"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func MaterializeFactoryAppTemplate(
	ctx context.Context,
	organizationID string,
	req *pb.MaterializeFactoryAppTemplateRequest,
) (*pb.MaterializeFactoryAppTemplateResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to materialize factory app template")
	}
	appID, err := parseFactoryAutomationID(req.GetAppId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to materialize factory app template")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to materialize factory app template")
	}
	if featureErr := requireBitbucketWorkspace(db, orgID, factory.OnboardingConfigValue().EffectiveVCSProvider()); featureErr != nil {
		return nil, factoryErrorToStatus(featureErr, "failed to materialize factory app template")
	}
	factoryID := factory.ID
	canvas, _, err := findFactoryAppForDefaults(db, orgID, factoryID, appID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to materialize factory app template")
	}

	input := factoryTemplateInputFromRequest(req)
	input.appID = canvas.ID.String()
	input.appName = canvas.Name
	// The template names the agent integration "claude". That is a type
	// placeholder, not an installation. An install that omits the agent uses
	// the workspace agent, the same one the other line apps already run.
	if input.agent == nil {
		input.agent = factoryTemplateAgentFromIntake(resolveIntakeAgent(db, factory))
	}
	provider := factory.OnboardingConfigValue().EffectiveVCSProvider()
	result, err := materializeFactoryTemplate(req.GetTemplateId(), provider, input)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to materialize factory app template")
	}
	if err := attachFactoryTemplateColumn(db, canvas, req.GetTemplateId(), provider); err != nil {
		return nil, factoryErrorToStatus(err, "failed to materialize factory app template")
	}

	return &pb.MaterializeFactoryAppTemplateResponse{
		TemplateId:  result.templateID,
		CanvasYaml:  result.canvasYAML,
		ConsoleYaml: result.consoleYAML,
	}, nil
}

func MaterializeFactoryAutomationDefaults(
	ctx context.Context,
	organizationID string,
	req *pb.MaterializeFactoryAutomationDefaultsRequest,
) (*pb.MaterializeFactoryAutomationDefaultsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to materialize factory automation defaults")
	}
	automationID, err := parseFactoryAutomationID(req.GetAutomationId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to materialize factory automation defaults")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to materialize factory automation defaults")
	}
	if featureErr := requireBitbucketWorkspace(db, orgID, factory.OnboardingConfigValue().EffectiveVCSProvider()); featureErr != nil {
		return nil, factoryErrorToStatus(featureErr, "failed to materialize factory automation defaults")
	}
	factoryID := factory.ID
	canvas, version, err := findFactoryAppForDefaults(db, orgID, factoryID, automationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to materialize factory automation defaults")
	}

	var result *materializedFactoryTemplate
	intake, intakeErr := models.FindFactoryIntakeByCanvasID(db, automationID)
	switch {
	case intakeErr == nil && intake.FactoryID == factoryID:
		result, err = materializeIntakeDefaults(db, canvas, version, intake)
	case intakeErr != nil && !errors.Is(intakeErr, models.ErrFactoryIntakeNotFound):
		err = intakeErr
	default:
		result, err = materializeNonIntakeFactoryAppDefaults(db, factory, canvas, version)
	}
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to materialize factory automation defaults")
	}

	return &pb.MaterializeFactoryAutomationDefaultsResponse{
		TemplateId:  result.templateID,
		CanvasYaml:  result.canvasYAML,
		ConsoleYaml: result.consoleYAML,
	}, nil
}
