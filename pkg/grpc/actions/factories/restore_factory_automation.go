package factories

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

func restoreFactoryAutomation(ctx context.Context, deps IntakeDependencies, factory *models.Factory, req *pb.CreateFactoryAutomationRequest) (*pb.CreateFactoryAutomationResponse, error) {
	var canvas *models.Canvas
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		factory, err := models.LockAutomationFactory(tx, factory.OrganizationID, factory.ID)
		if err != nil {
			return err
		}
		name := strings.TrimSpace(req.GetName())
		if name == "" {
			name = "Run an agent"
			if req.GetTemplateId() == models.FactoryAppTemplateBacklogID {
				name = backlogDefaultName
			}
		}
		name, err = models.AvailableCanvasName(tx, factory.OrganizationID, &factory.ID, name)
		if err != nil {
			return err
		}
		id := uuid.New()
		doc, err := restoredAutomationDocument(tx, factory, req.GetTemplateId(), id, name)
		if err != nil {
			return err
		}
		nodes, edges, err := doc.Parse(deps.Registry, factory.OrganizationID.String())
		if err != nil {
			return err
		}
		if _, err := canvases.CreateCanvasWithIDInTransaction(ctx, tx, id, deps.Registry, deps.Encryptor, deps.AuthService, deps.WebhookBaseURL,
			factory.OrganizationID, name, doc.Metadata.Description, &factory.ID, nodes, edges); err != nil {
			return err
		}
		canvas, err = models.FindCanvasInTransaction(tx, factory.OrganizationID, id)
		if err != nil {
			return err
		}
		if req.GetTemplateId() == models.FactoryAppTemplateBacklogID {
			return factory.SetBacklogAutomationRemoved(tx, false)
		}
		return bindRestoredAgent(tx, factory, canvas, req.GetInstallParams())
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to restore automation")
	}
	canvases.PublishCanvasUpdated(canvas)
	return &pb.CreateFactoryAutomationResponse{Automation: serializeFactoryAutomation(*canvas)}, nil
}

func restoredAutomationDocument(tx *gorm.DB, factory *models.Factory, templateID string, id uuid.UUID, name string) (*yaml.Canvas, error) {
	if templateID == models.FactoryAppTemplateBacklogID {
		has, err := factoryHasOnWorkOrderCanvas(tx, factory)
		if err != nil {
			return nil, err
		}
		if has {
			return nil, invalidArgument("Task analysis is already installed.")
		}
		return buildBacklogCanvas(backlogCanvasRequest{
			Name: name, Agent: resolveIntakeAgent(tx, factory), VCSName: resolveVCSInstallationName(tx, factory), VCSProvider: factoryVCSProviderOrDefault(factory),
		}), nil
	}
	config := factory.OnboardingConfigValue()
	result, err := materializeFactoryTemplate(templateID, config.EffectiveVCSProvider(), factoryTemplateInput{
		appID: id.String(), appName: name,
		agent:         factoryTemplateAgentFromIntake(resolveIntakeAgent(tx, factory)),
		installParams: map[string]string{"appRepository": config.AppRepository, "backlogRepository": config.BacklogRepository, "defaultBranch": config.DefaultBranch},
		integrations:  map[string]factoryTemplateIntegration{config.EffectiveVCSProvider(): {id: config.VCSIntegrationID, name: resolveVCSInstallationName(tx, factory)}},
	})
	if err != nil {
		return nil, err
	}
	return yaml.CanvasFromYAML([]byte(result.canvasYAML))
}

func bindRestoredAgent(tx *gorm.DB, factory *models.Factory, canvas *models.Canvas, params map[string]string) error {
	lineID, err := parseLineID(params["lineId"])
	if err != nil {
		return err
	}
	line, err := factory.FindLineForUpdate(tx, lineID)
	if err != nil {
		return err
	}
	stages := line.StagesValue()
	for i := range stages {
		stage := &stages[i]
		if stage.Key != params["stageKey"] || stage.Kind != models.FactoryLineStagePhase {
			continue
		}
		if stage.StepIndex != nil {
			return invalidArgument("This stage already has an agent automation.")
		}
		index := len(line.Steps)
		stage.StepIndex = &index
		steps := append([]models.FactoryLineStep(line.Steps), models.FactoryLineStep{Type: models.FactoryLineStepTypeRunApp, AppID: canvas.ID, Entrypoint: "onrun-implement"})
		return line.UpdateConfiguration(tx, nil, steps, nil, stages)
	}
	return invalidArgument("column key is invalid")
}
