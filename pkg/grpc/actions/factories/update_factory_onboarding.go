package factories

import (
	"context"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/billing"
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

	config, err := factory.OnboardingConfigAfter(patch)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory onboarding")
	}
	if config.AgentIntegrationID != "" && (patch.AgentIntegrationID != nil || patch.AgentHarness != nil || req.GetComplete()) {
		integration, err := findReadyOnboardingIntegration(db, orgID, config.AgentIntegrationID)
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to update factory onboarding")
		}
		provider := ""
		if integration.AppName == models.CustomLLMAppName {
			provider = models.UsageProviderCustom
		}
		if err := models.RequireOrganizationBYOKAccess(db, orgID, provider); err != nil {
			return nil, factoryErrorToStatus(err, "failed to update factory onboarding")
		}
	}

	if req.GetComplete() && !wasComplete && billing.Enabled() {
		if err := billing.AssertOrganizationPlanAllowsRuns(db, orgID); err != nil {
			return nil, factoryErrorToStatus(err, "failed to update factory onboarding")
		}
		if err := billing.AssertOrganizationPaymentMethod(db, orgID); err != nil {
			return nil, factoryErrorToStatus(err, "failed to update factory onboarding")
		}
	}

	if req.Complete != nil && *req.Complete {
		err = db.Transaction(func(tx *gorm.DB) error {
			if err := validateFactoryOnboardingResources(tx, orgID, factory, config); err != nil {
				return err
			}
			if !wasComplete {
				if err := synchronizeOnboardingSubscriptionApps(ctx, tx, deps, factory, config); err != nil {
					return err
				}
			}
			if config.CodexSubscription {
				if err := validateSubscriptionCanvases(tx, factory); err != nil {
					return err
				}
			}
			if !wasComplete {
				if err := ensureOnboardingDiscussionHandler(ctx, deps, organizationID, tx, factory, patch, config); err != nil {
					return err
				}
			}
			return factory.CompleteOnboarding(tx, patch)
		})
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
		_, err = createFactoryPRFeedbackHandler(ctx, deps, organizationID, tx, &pb.CreateFactoryPRFeedbackHandlerRequest{
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
	if config.IssuesSource == models.FactoryOnboardingIssuesSourceVCS && config.EffectiveVCSProvider() != models.ProviderGitHub {
		return invalidArgument("GitHub Issues requires a GitHub repository")
	}
	if config.VCSIntegrationID != "" {
		integration, err := findReadyOnboardingIntegration(db, organizationID, config.VCSIntegrationID)
		if err != nil {
			return err
		}
		if integration.AppName != models.ProviderGitHub && integration.AppName != "gitlab" && integration.AppName != models.ProviderBitbucket {
			return invalidArgument("version control integration must be GitHub, GitLab, or Bitbucket")
		}
	}

	if config.CodexSubscription {
		if err := validateCodexSubscriptionOnboarding(db, organizationID, factory, config); err != nil {
			return err
		}
	} else if config.AgentIntegrationID != "" {
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
	if req.CodexSubscription != nil {
		value := req.GetCodexSubscription()
		patch.CodexSubscription = &value
	}

	return patch, nil
}

func validateCodexSubscriptionOnboarding(
	db *gorm.DB,
	organizationID uuid.UUID,
	factory *models.Factory,
	config models.FactoryOnboardingConfig,
) error {
	if config.AgentHarness != models.FactoryOnboardingAgentHarnessCodex {
		return invalidArgument("ChatGPT subscription uses Codex")
	}
	if config.AgentIntegrationID != "" {
		return invalidArgument("ChatGPT subscription does not use an API key")
	}
	if _, err := models.SoleHumanOrganizationMember(db, organizationID); err != nil {
		return err
	}
	connection, err := models.FindFactoryCodexConnection(db, organizationID, factory.ID)
	if err != nil {
		return invalidArgument("Connect ChatGPT before you finish setup")
	}
	if connection.Status != models.CodexConnectionStatusReady || connection.Reserved() {
		return invalidArgument("Wait until the ChatGPT sign-in runner has stopped")
	}
	return nil
}

func validateSubscriptionCanvases(db *gorm.DB, factory *models.Factory) error {
	canvases, err := factory.ListCanvases(db)
	if err != nil {
		return err
	}
	for i := range canvases {
		nodes, err := models.FindCanvasNodesInTransaction(db, canvases[i].ID)
		if err != nil {
			return err
		}
		for _, node := range nodes {
			if err := validateSubscriptionCodexNode(node); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateSubscriptionCodexNode(node models.CanvasNode) error {
	name := node.ComponentName()
	switch name {
	case "", "runnerClaudeCode", "runnerOpenRouter", models.SuperPlaneRunnerComponent:
		if name == "" {
			return nil
		}
		return invalidArgument("This workspace must use Codex with a ChatGPT subscription")
	case "runnerCodex":
		configuration := node.Configuration.Data()
		credentials, _ := configuration["credentials"].(map[string]any)
		source, _ := credentials["source"].(string)
		if source != "subscription" {
			return invalidArgument("This workspace must use Codex with a ChatGPT subscription")
		}
	}
	return nil
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
