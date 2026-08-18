package factories

import (
	"context"
	"slices"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func UpdateFactoryOnboarding(
	ctx context.Context,
	organizationID string,
	req *pb.UpdateFactoryOnboardingRequest,
) (*pb.UpdateFactoryOnboardingResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory onboarding")
	}

	id, err := parseFactoryID(req.GetId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory onboarding")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, id)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory onboarding")
	}

	patch, err := factoryOnboardingPatchFromRequest(req)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory onboarding")
	}

	if req.Complete != nil && *req.Complete {
		config, configErr := factory.OnboardingConfigAfter(patch)
		if configErr != nil {
			return nil, factoryErrorToStatus(configErr, "failed to update factory onboarding")
		}
		if validationErr := validateFactoryOnboardingResources(db, orgID, factory, config); validationErr != nil {
			return nil, factoryErrorToStatus(validationErr, "failed to update factory onboarding")
		}
		err = factory.CompleteOnboarding(db, patch)
	} else {
		err = factory.UpdateOnboarding(db, patch)
	}
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory onboarding")
	}

	lines, err := factory.ListLines(db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory onboarding")
	}

	return &pb.UpdateFactoryOnboardingResponse{
		Factory: serializeFactoryWithLines(factory, lines),
	}, nil
}

func validateFactoryOnboardingResources(
	db *gorm.DB,
	organizationID uuid.UUID,
	factory *models.Factory,
	config models.FactoryOnboardingConfig,
) error {
	if config.VCSIntegrationID != "" {
		integration, err := findReadyOnboardingIntegration(db, organizationID, config.VCSIntegrationID)
		if err != nil {
			return err
		}
		if integration.AppName != "github" && integration.AppName != "gitlab" {
			return invalidArgument("version control integration must be GitHub or GitLab")
		}
	}

	if config.AgentIntegrationID != "" {
		integration, err := findReadyOnboardingIntegration(db, organizationID, config.AgentIntegrationID)
		if err != nil {
			return err
		}
		expectedName := map[string]string{
			models.FactoryOnboardingAgentHarnessClaudeCode: "claude",
			models.FactoryOnboardingAgentHarnessCursor:     "cursor",
			models.FactoryOnboardingAgentHarnessCodex:      "openai",
		}[config.AgentHarness]
		if expectedName != "" && integration.AppName != expectedName {
			return invalidArgument("coding agent integration does not match the selected agent")
		}
	}

	if config.ProvisionedAppID != "" {
		appID, err := uuid.Parse(config.ProvisionedAppID)
		if err != nil {
			return models.ErrFactoryOnboardingInvalidAppID
		}
		app, err := models.FindCanvasInTransaction(db, organizationID, appID)
		if err != nil || app.FactoryID == nil || *app.FactoryID != factory.ID {
			return invalidArgument("provisioned app does not belong to this workspace")
		}
	}

	if config.ProvisionedLineID != "" {
		lineID, err := uuid.Parse(config.ProvisionedLineID)
		if err != nil {
			return models.ErrFactoryOnboardingInvalidLineID
		}
		line, err := factory.FindLine(db, lineID)
		if err != nil {
			return invalidArgument("provisioned line does not belong to this workspace")
		}
		appID, err := uuid.Parse(config.ProvisionedAppID)
		if err != nil {
			return models.ErrFactoryOnboardingInvalidAppID
		}
		if !slices.ContainsFunc(line.Steps, func(step models.FactoryLineStep) bool {
			return step.AppID == appID
		}) {
			return invalidArgument("provisioned line does not run the provisioned app")
		}
	}

	return nil
}

func findReadyOnboardingIntegration(
	db *gorm.DB,
	organizationID uuid.UUID,
	integrationID string,
) (*models.Integration, error) {
	id, err := uuid.Parse(integrationID)
	if err != nil {
		return nil, models.ErrFactoryOnboardingInvalidIntegrationID
	}
	integration, err := models.FindIntegrationInTransaction(db, organizationID, id)
	if err != nil {
		return nil, invalidArgument("integration does not belong to this organization")
	}
	if integration.State != models.IntegrationStateReady {
		return nil, invalidArgument("integration is not ready")
	}
	return integration, nil
}

func factoryOnboardingPatchFromRequest(req *pb.UpdateFactoryOnboardingRequest) (models.FactoryOnboardingPatch, error) {
	patch := models.FactoryOnboardingPatch{
		VCSIntegrationID:   req.VcsIntegrationId,
		AgentIntegrationID: req.AgentIntegrationId,
		AppRepository:      req.AppRepository,
		BacklogRepository:  req.BacklogRepository,
		ProvisionedAppID:   req.ProvisionedAppId,
		ProvisionedLineID:  req.ProvisionedLineId,
	}

	if req.IssuesSource != nil {
		source, err := factoryOnboardingIssuesSourceFromProto(*req.IssuesSource)
		if err != nil {
			return models.FactoryOnboardingPatch{}, err
		}
		patch.IssuesSource = &source
	}

	if req.AgentHarness != nil {
		harness, err := factoryOnboardingAgentHarnessFromProto(*req.AgentHarness)
		if err != nil {
			return models.FactoryOnboardingPatch{}, err
		}
		patch.AgentHarness = &harness
	}

	return patch, nil
}

func factoryOnboardingIssuesSourceFromProto(source pb.FactoryOnboarding_IssuesSource) (string, error) {
	switch source {
	case pb.FactoryOnboarding_ISSUES_SOURCE_UNSPECIFIED:
		return "", nil
	case pb.FactoryOnboarding_ISSUES_SOURCE_VCS:
		return models.FactoryOnboardingIssuesSourceVCS, nil
	case pb.FactoryOnboarding_ISSUES_SOURCE_LINEAR:
		return models.FactoryOnboardingIssuesSourceLinear, nil
	case pb.FactoryOnboarding_ISSUES_SOURCE_JIRA:
		return models.FactoryOnboardingIssuesSourceJira, nil
	case pb.FactoryOnboarding_ISSUES_SOURCE_SKIP:
		return models.FactoryOnboardingIssuesSourceSkip, nil
	default:
		return "", invalidArgument("invalid issues source")
	}
}

func factoryOnboardingAgentHarnessFromProto(harness pb.FactoryOnboarding_AgentHarness) (string, error) {
	switch harness {
	case pb.FactoryOnboarding_AGENT_HARNESS_UNSPECIFIED:
		return "", nil
	case pb.FactoryOnboarding_AGENT_HARNESS_CLAUDE_CODE:
		return models.FactoryOnboardingAgentHarnessClaudeCode, nil
	case pb.FactoryOnboarding_AGENT_HARNESS_CURSOR:
		return models.FactoryOnboardingAgentHarnessCursor, nil
	case pb.FactoryOnboarding_AGENT_HARNESS_CODEX:
		return models.FactoryOnboardingAgentHarnessCodex, nil
	default:
		return "", invalidArgument("invalid agent harness")
	}
}
