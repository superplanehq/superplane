package factories

import (
	"context"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/productanalytics"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func UpdateFactoryOnboarding(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.UpdateFactoryOnboardingRequest,
) (*pb.UpdateFactoryOnboardingResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory onboarding")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory onboarding")
	}
	wasComplete := factory.IsOnboardingComplete()

	patch, err := factoryOnboardingPatchFromRequest(req)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory onboarding")
	}
	setOnboardingVCSProvider(db, orgID, &patch)

	if err := rejectVCSProviderChange(factory, patch); err != nil {
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
		if !wasComplete {
			if err := ensureOnboardingDiscussionHandler(ctx, deps, organizationID, db, factory, patch, config); err != nil {
				return nil, factoryErrorToStatus(err, "failed to update factory onboarding")
			}
		}
		err = factory.CompleteOnboarding(db, patch)
	} else {
		err = factory.UpdateOnboarding(db, patch)
	}
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory onboarding")
	}

	if err := ensureFactoryMergeabilityWebhook(ctx, db, deps, factory); err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory onboarding")
	}

	if !wasComplete && factory.IsOnboardingComplete() {
		config := factory.OnboardingConfigValue()
		captureFactoryProductEvent(
			ctx,
			db,
			factory,
			uuid.Nil,
			productanalytics.EventFactoryOnboardingCompleted,
			productanalytics.StableEventID(productanalytics.EventFactoryOnboardingCompleted, factory.ID.String()),
			productanalytics.Properties{
				"vcs_provider":  config.EffectiveVCSProvider(),
				"issues_source": config.IssuesSource,
				"agent_harness": config.AgentHarness,
			},
		)
	}

	lines, err := factory.ListLines(db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory onboarding")
	}

	serialized, err := serializeFactoryWithLineMetrics(db, factory, lines)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory onboarding")
	}

	return &pb.UpdateFactoryOnboardingResponse{
		Factory: serialized,
	}, nil
}

func ensureOnboardingDiscussionHandler(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	db *gorm.DB,
	factory *models.Factory,
	patch models.FactoryOnboardingPatch,
	config models.FactoryOnboardingConfig,
) error {
	if err := config.ReadyForCompletion(); err != nil {
		return nil
	}
	repository := strings.TrimSpace(config.AppRepository)
	if repository == "" {
		return nil
	}
	if err := factory.UpdateOnboarding(db, patch); err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			"SELECT pg_advisory_xact_lock(hashtextextended(?, 0))",
			"factory-discussion-handler:"+factory.ID.String(),
		).Error; err != nil {
			return err
		}
		handlers, err := factory.ListPRFeedbackHandlers(tx)
		if err != nil {
			return err
		}
		for _, handler := range handlers {
			if handler.Source == models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion {
				return nil
			}
		}
		_, err = CreateFactoryPRFeedbackHandler(ctx, deps, organizationID, &pb.CreateFactoryPRFeedbackHandlerRequest{
			FactoryId: factory.ID.String(),
			Settings: &pb.FactoryPRFeedbackHandler_Settings{
				Subject: &pb.FactoryPRFeedbackHandler_SubjectSettings{
					Repository: repository,
				},
				Discussion: &pb.FactoryPRFeedbackHandler_DiscussionSettings{
					Mention:        prFeedbackDefaultMention,
					IgnoreBots:     true,
					AddressAllBots: true,
				},
			},
		})
		return err
	})
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
		if integration.AppName != models.ProviderGitHub && integration.AppName != "gitlab" && integration.AppName != models.ProviderBitbucket {
			return invalidArgument("version control integration must be GitHub, GitLab, or Bitbucket")
		}
	}

	if config.AgentIntegrationID != "" {
		integration, err := findReadyOnboardingIntegration(db, organizationID, config.AgentIntegrationID)
		if err != nil {
			return err
		}
		expectedNames := map[string][]string{
			// OpenCode agents (OpenRouter and a custom provider) use this harness.
			models.FactoryOnboardingAgentHarnessClaudeCode: {"claude", "openrouter", models.CustomLLMAppName},
			models.FactoryOnboardingAgentHarnessCursor:     {"cursor"},
			models.FactoryOnboardingAgentHarnessCodex:      {"openai"},
		}[config.AgentHarness]
		if len(expectedNames) > 0 && !slices.Contains(expectedNames, integration.AppName) {
			return invalidArgument("coding agent integration does not match the selected agent")
		}
	} else {
		offered, hostedErr := models.HasOfferedHostedLLMProvider(db)
		if hostedErr != nil {
			return hostedErr
		}
		if !offered {
			return models.ErrFactoryOnboardingHostedAgentUnavailable
		}
		defaultModel, modelErr := models.GetInstallationDefaultHostedLLMModel(db)
		if modelErr != nil {
			return modelErr
		}
		if !defaultModel.IsSet() {
			return models.ErrFactoryOnboardingHostedAgentUnavailable
		}
		if err := models.AssertDefaultHostedLLMModelAllowed(db, defaultModel); err != nil {
			return models.ErrFactoryOnboardingHostedAgentUnavailable
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

// rejectVCSProviderChange locks the provider at the first saved VCS binding.
// Clearing the integration keeps the lock; switching hosts is rejected.
func rejectVCSProviderChange(factory *models.Factory, patch models.FactoryOnboardingPatch) error {
	if patch.VCSProvider == nil {
		return nil
	}
	next := strings.TrimSpace(*patch.VCSProvider)
	if next == "" {
		return nil
	}
	current := factory.OnboardingConfigValue()
	if strings.TrimSpace(current.VCSIntegrationID) == "" && strings.TrimSpace(current.VCSProvider) == "" {
		return nil
	}
	if !strings.EqualFold(next, current.EffectiveVCSProvider()) {
		return invalidArgument("version control provider cannot be changed")
	}
	return nil
}

func setOnboardingVCSProvider(db *gorm.DB, organizationID uuid.UUID, patch *models.FactoryOnboardingPatch) {
	if patch.VCSIntegrationID == nil {
		return
	}
	integrationID := strings.TrimSpace(*patch.VCSIntegrationID)
	if integrationID == "" {
		// ponytail: clearing the integration keeps the provider lock
		return
	}
	id, err := uuid.Parse(integrationID)
	if err != nil {
		return
	}
	integration, err := models.FindIntegrationInTransaction(db, organizationID, id)
	if err != nil {
		return
	}
	provider, ok := models.VCSProviderForIntegrationApp(integration.AppName)
	if !ok {
		return
	}
	patch.VCSProvider = &provider
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
		DefaultBranch:      req.DefaultBranch,
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
	case pb.FactoryOnboarding_AGENT_HARNESS_SUPERPLANE:
		return models.FactoryOnboardingAgentHarnessSuperPlane, nil
	default:
		return "", invalidArgument("invalid agent harness")
	}
}
