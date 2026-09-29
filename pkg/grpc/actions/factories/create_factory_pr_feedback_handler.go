package factories

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PRFeedbackDependencies = IntakeDependencies

var errFactoryPRFeedbackHandlerSourceExists = errors.New("factory already has a pull request discussion handler")

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

	if source == models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion {
		hasDiscussion, err := factory.HasPRFeedbackHandlerSource(db, source)
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to create factory PR feedback handler")
		}
		if hasDiscussion {
			return nil, grpcerrors.AlreadyExists(
				errFactoryPRFeedbackHandlerSourceExists,
				"factory already has a pull request discussion handler",
			)
		}
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

	handler, err := createPRFeedbackHandlerOnce(db, factory, canvasID, subject, source)
	if err != nil {
		discardIntakeCanvas(db, orgID, canvasID)
		if errors.Is(err, errFactoryPRFeedbackHandlerSourceExists) {
			return nil, grpcerrors.AlreadyExists(err, "factory already has a pull request discussion handler")
		}
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

// createPRFeedbackHandlerOnce inserts the handler while serializing concurrent
// creates for the same factory. The factory row lock makes a second in-flight
// request wait for the first to commit, so the duplicate re-check sees the
// committed handler and no factory can end up with two live handlers for the
// same source.
func createPRFeedbackHandlerOnce(
	db *gorm.DB,
	factory *models.Factory,
	canvasID uuid.UUID,
	subject, source string,
) (*models.FactoryPRFeedbackHandler, error) {
	var handler *models.FactoryPRFeedbackHandler
	err := db.Transaction(func(tx *gorm.DB) error {
		var locked models.Factory
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", factory.ID).
			First(&locked).
			Error; err != nil {
			return err
		}

		if source == models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion {
			hasDiscussion, err := factory.HasPRFeedbackHandlerSource(tx, source)
			if err != nil {
				return err
			}
			if hasDiscussion {
				return errFactoryPRFeedbackHandlerSourceExists
			}
		}

		var err error
		handler, err = factory.CreatePRFeedbackHandler(tx, canvasID, subject, source)
		return err
	})
	if err != nil {
		return nil, err
	}
	return handler, nil
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

	integration := findIntakeGitHubIntegration(tx, factory, config.VCSIntegrationID)
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
