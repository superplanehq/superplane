package factories

import (
	"context"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

type PRFeedbackDependencies = IntakeDependencies

func CreateFactoryPRFeedbackHandler(
	ctx context.Context,
	deps PRFeedbackDependencies,
	organizationID string,
	req *pb.CreateFactoryPRFeedbackHandlerRequest,
) (*pb.CreateFactoryPRFeedbackHandlerResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory PR feedback handler")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory PR feedback handler")
	}
	factoryID := factory.ID

	repository := strings.TrimSpace(req.GetSettings().GetSubject().GetRepository())
	if repository == "" {
		repository = strings.TrimSpace(factory.OnboardingConfigValue().AppRepository)
	}
	if repository == "" {
		return nil, factoryErrorToStatus(invalidArgument("repository is required"), "failed to create factory PR feedback handler")
	}

	subject, err := parseFactoryPRFeedbackHandlerSubject(req.GetSubject())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory PR feedback handler")
	}
	source, err := parseFactoryPRFeedbackHandlerSource(req.GetSource())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory PR feedback handler")
	}

	settings := parsePRFeedbackSettings(defaultPRFeedbackSettings(), req.GetSettings())
	settings.Repository = repository
	if err := validatePRFeedbackSettingsForSource(db, orgID, source, settings, req.GetSettings()); err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory PR feedback handler")
	}
	if err := resolveRunnerIntegrationNames(db, orgID, &settings); err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory PR feedback handler")
	}

	name := strings.TrimSpace(req.GetName())
	if name == "" {
		if source == models.FactoryPRFeedbackHandlerSourcePullRequestChecks {
			name = prFeedbackChecksDefaultName
		} else {
			name = prFeedbackDefaultName
		}
	}
	name, err = models.AvailableCanvasName(db, orgID, &factoryID, name)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory PR feedback handler")
	}

	canvasID, err := createPRFeedbackCanvas(ctx, deps, factory, source, name, settings)
	if err != nil {
		return nil, err
	}

	handler, err := factory.CreatePRFeedbackHandler(db, canvasID, subject, source)
	if err != nil {
		discardIntakeCanvas(db, orgID, canvasID)
		return nil, factoryErrorToStatus(err, "failed to create factory PR feedback handler")
	}
	if source == models.FactoryPRFeedbackHandlerSourcePullRequestChecks {
		if err := handler.SetMaximumAttempts(db, settings.MaximumAttempts); err != nil {
			discardIntakeCanvas(db, orgID, canvasID)
			return nil, factoryErrorToStatus(err, "failed to create factory PR feedback handler")
		}
	}

	handler, err = factory.FindPRFeedbackHandler(db, handler.ID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory PR feedback handler")
	}

	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(db, []uuid.UUID{canvasID})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory PR feedback handler")
	}

	return &pb.CreateFactoryPRFeedbackHandlerResponse{
		Handler: serializeFactoryPRFeedbackHandler(db, orgID, handler, specs[canvasID]),
	}, nil
}

func createPRFeedbackCanvas(
	ctx context.Context,
	deps PRFeedbackDependencies,
	factory *models.Factory,
	source, name string,
	settings prFeedbackSettings,
) (uuid.UUID, error) {
	db := database.DB(ctx)
	binding := resolvePRFeedbackBinding(db, factory, settings.Repository)
	request := prFeedbackBuildRequest{
		Name:                   name,
		Repository:             settings.Repository,
		Mention:                settings.Mention,
		IgnoreBots:             settings.IgnoreBots,
		AllowedBots:            settings.AllowedBots,
		CheckNames:             settings.CheckNames,
		MaximumAttempts:        settings.MaximumAttempts,
		RunnerIntegrationNames: settings.RunnerIntegrationNames,
		Binding:                binding,
		Agent:                  resolveIntakeAgent(db, factory),
		VCSProvider:            factory.OnboardingConfigValue().EffectiveVCSProvider(),
	}
	var canvasDoc *yaml.Canvas
	if source == models.FactoryPRFeedbackHandlerSourcePullRequestChecks {
		canvasDoc = buildChecksPRFeedbackCanvas(request)
	} else {
		canvasDoc = buildDiscussionPRFeedbackCanvas(request)
	}

	nodes, edges, err := canvasDoc.Parse(deps.Registry, factory.OrganizationID.String())
	if err != nil {
		return uuid.Nil, factoryErrorToStatus(err, "failed to build PR feedback automation")
	}

	response, err := canvases.CreateCanvas(
		ctx,
		deps.Registry,
		deps.Encryptor,
		deps.AuthService,
		deps.WebhookBaseURL,
		factory.OrganizationID,
		canvasDoc.Metadata.Name,
		canvasDoc.Metadata.Description,
		&factory.ID,
		nodes,
		edges,
	)
	if err != nil {
		return uuid.Nil, err
	}

	canvasID, err := uuid.Parse(response.GetCanvas().GetMetadata().GetId())
	if err != nil {
		return uuid.Nil, factoryErrorToStatus(err, "failed to create factory PR feedback handler")
	}

	return canvasID, nil
}

func resolvePRFeedbackBinding(tx *gorm.DB, factory *models.Factory, repository string) *intakeBinding {
	config := factory.OnboardingConfigValue()
	if config.VCSIntegrationID == "" {
		return &intakeBinding{Configuration: map[string]any{"repository": repository}}
	}

	integration := findIntakeVCSIntegration(tx, factory, config.VCSIntegrationID)
	if integration == nil {
		return &intakeBinding{Configuration: map[string]any{"repository": repository}}
	}

	return &intakeBinding{
		Integration: &yaml.IntegrationRef{
			ID:   integration.ID.String(),
			Name: integration.InstallationName,
		},
		Configuration: map[string]any{"repository": repository},
		Installation:  integration,
	}
}

// findIntakeVCSIntegration resolves the workspace VCS installation for
// feedback bindings. GitHub and Bitbucket installations both qualify;
// anything else leaves the handler unbound.
func findIntakeVCSIntegration(tx *gorm.DB, factory *models.Factory, integrationID string) *models.Integration {
	id, err := uuid.Parse(strings.TrimSpace(integrationID))
	if err != nil {
		log.Warnf("factory %s: feedback left unbound, invalid integration id %q", factory.ID, integrationID)
		return nil
	}

	integration, err := models.FindIntegrationInTransaction(tx, factory.OrganizationID, id)
	if err != nil {
		log.Warnf("factory %s: feedback left unbound, integration %s not found: %v", factory.ID, id, err)
		return nil
	}

	if integration.AppName != intakeGitHubAppName && integration.AppName != models.ProviderBitbucket {
		log.Warnf("factory %s: feedback left unbound, integration %s is a %s installation", factory.ID, id, integration.AppName)
		return nil
	}
	if integration.State != models.IntegrationStateReady {
		log.Warnf("factory %s: feedback left unbound, integration %s is not ready", factory.ID, id)
		return nil
	}

	return integration
}
