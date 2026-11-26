package factories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/google/uuid"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	githubcommon "github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

type preparedFactoryConfigurationBuiltIn struct {
	TemplateID         string
	Nodes              []models.Node
	Edges              []models.Edge
	Defaults           *models.CanvasVersion
	Settings           map[string]any
	Binding            factoryConfigurationBinding
	DefaultBinding     factoryConfigurationBinding
	IntakeSource       string
	PRFeedbackSource   string
	MaximumAttempts    int
	Paused             bool
	Description        string
	ExplicitBinding    *factoryConfigurationBinding
	GitHubRepositoryID int64
	State              any
}

type configurationIntegrationState struct {
	ID            uuid.UUID
	Provider      string
	Name          string
	State         string
	Capabilities  []models.CapabilityState
	Host          string
	Resources     map[string]any
	Configuration map[string]any
	Properties    map[string]any
}

type configurationBuiltInState struct {
	Binding               factoryConfigurationBinding
	Integration           configurationIntegrationState
	Agent                 *configurationIntegrationState
	AgentHarness          string
	AgentModel            string
	DefaultOwner          string
	Repository            string
	WorkspaceRepositories []string
	Backlog               string
	Branch                string
	GitProvider           string
	Subscription          bool
	HostedModel           models.DefaultHostedLLMModel
	HostedProviders       []configurationHostedProviderState
	RunnerIntegrations    []configurationIntegrationState
}

func configurationBuiltInTemplateID(alias string) (string, bool) {
	for _, entry := range builtInAutomations {
		candidate, _ := factoryConfigurationBuiltInAlias(entry.ID)
		if candidate == alias {
			return entry.ID, true
		}
	}
	return "", false
}

// Preparation shares the install builders but never runs Setup or creates records.
func prepareFactoryConfigurationBuiltIn(ctx context.Context, tx *gorm.DB, deps IntakeDependencies, target *factoryConfigurationTarget, stage *factoryConfigurationStageTarget, automation factoryConfigurationAutomation) (result *preparedFactoryConfigurationBuiltIn, err error) {
	namedConnection := automation.Binding != nil && automation.Binding.IntegrationName != ""
	defer func() {
		if setting, ok := err.(*factoryConfigurationSettingError); namedConnection && ok && setting.Field == "binding.integration_id" {
			setting.Field = "binding.integration_name"
		}
	}()
	automation.Binding, err = resolveConfigurationBindingName(tx, target.factory.OrganizationID, automation.Binding)
	if err != nil {
		return nil, err
	}
	templateID, exists := configurationBuiltInTemplateID(automation.Uses)
	if !exists {
		return nil, configurationPreparationError("uses", "This built-in automation is not supported.")
	}
	if err := validateConfigurationBuiltInStage(templateID, stage); err != nil {
		return nil, err
	}
	if err := validateConfigurationBuiltInFeature(tx, target.factory, templateID); err != nil {
		return nil, err
	}

	if err := validateFactoryConfigurationBuiltInSettingSections(templateID, automation.With); err != nil {
		return nil, prefixConfigurationPreparationSettings(err)
	}
	if templateID == intakeTemplateID(models.FactoryIntakeSourceWebhook) && automation.current != nil {
		return prepareInstalledConfigurationWebhookIntake(tx, deps, target.factory, automation)
	}
	if factoryVCSProviderOrDefault(target.factory) == models.ProviderBitbucket && templateID == "risk-score" {
		section, err := configurationSettingsSection(automation.With, "template")
		if err != nil {
			return nil, err
		}
		for _, field := range []string{"publish_pr_check", "results_comment"} {
			if _, present := section[field]; present {
				return nil, configurationPreparationError("with.template."+field, "This setting is not supported for Bitbucket.")
			}
		}
	}
	prepared := &preparedFactoryConfigurationBuiltIn{TemplateID: templateID, ExplicitBinding: automation.Binding}
	selection, selectionField, err := configurationBuiltInSettingsBinding(templateID, automation)
	if err != nil {
		return nil, err
	}
	binding, integration, err := resolveConfigurationBuiltInBinding(tx, target.factory, templateID, selection)
	if err != nil {
		if setting, ok := err.(*factoryConfigurationSettingError); ok && setting.Field == "binding.resource_id" && selectionField != "" {
			setting.Field = selectionField
		}
		return nil, err
	}
	if integration.AppName == models.ProviderGitHub && binding.ResourceID != "" {
		prepared.GitHubRepositoryID, _, err = authorizeConfigurationPrimaryRepository(ctx, tx, target.factory, integration, binding.ResourceID)
		if err != nil {
			code, message, ok := grpcerrors.HandlerStatus(err)
			if ok && code == codes.PermissionDenied {
				return nil, configurationPreparationError("binding.resource_id", message)
			}
			return nil, err
		}
	}
	prepared.Binding = binding
	prepared.DefaultBinding = configurationBuiltInDefaultBinding(target.factory, templateID)
	agent := resolveIntakeAgent(tx, target.factory)
	if configurationBuiltInNeedsAgent(templateID) && agent == nil {
		return nil, configurationPreparationError("with.agents", "Connect a ready workspace agent before you install this automation.")
	}
	baseline, err := buildConfigurationBuiltInGraph(tx, target.factory, automation.Name, templateID, binding, integration, agent, nil, nil)
	if err != nil {
		return nil, err
	}
	document, err := buildConfigurationBuiltInGraph(tx, target.factory, automation.Name, templateID, binding, integration, agent, automation.With, prepared)
	if err != nil {
		return nil, err
	}
	prepared.Description = document.Metadata.Description
	prepared.Defaults = &models.CanvasVersion{Nodes: baseline.Nodes(), Edges: baseline.Edges()}
	version := &models.CanvasVersion{Nodes: document.Nodes(), Edges: document.Edges()}
	prepared.Nodes, prepared.Edges, err = copyFactoryConfigurationBuiltIn(version, version, templateID, automation.With)
	if err != nil {
		return nil, prefixConfigurationPreparationSettings(err)
	}
	generated, err := exportFactoryConfigurationBuiltIn(&models.CanvasVersion{Nodes: prepared.Nodes, Edges: prepared.Edges}, templateID)
	if err != nil {
		return nil, err
	}
	configurationPreparedRecordSettings(prepared, automation.With, generated)
	current := automation.current
	keepInstalled := current != nil && current.binding == binding && configurationBindingIntentEqual(current, automation.Binding) && configurationPublicSectionsEqual(current.settings, generated)
	if keepInstalled {
		prepared.Nodes, prepared.Edges, err = copyFactoryConfigurationBuiltIn(current.version, current.defaults, templateID, automation.With)
		if err != nil {
			return nil, prefixConfigurationPreparationSettings(err)
		}
	}
	if current != nil {
		prepared.Nodes, prepared.Edges = alignConfigurationIntakeTrigger(current.version, prepared.IntakeSource, prepared.Nodes, prepared.Edges)
	}
	if err := validateFactoryConfigurationBuiltInNodes(tx, deps, target.factory, prepared.Nodes); err != nil {
		return nil, prefixConfigurationPreparationSettings(err)
	}
	// Retain installed managed graphs for public agent changes. Legacy graphs
	// can contain components or errors that the current template no longer uses.
	if !keepInstalled {
		// Graph parsing validates local references and schemas without trigger setup.
		encoded, err := yaml.VersionToCanvasYAML(automation.Name, document.Metadata.Description, &models.CanvasVersion{Nodes: prepared.Nodes, Edges: prepared.Edges})
		if err != nil {
			return nil, err
		}
		document, err = yaml.CanvasFromYAML([]byte(encoded))
		if err != nil {
			return nil, err
		}
		if reads := factoryAutomationReads(tx, target.factory); reads != nil {
			prepared.Nodes, prepared.Edges, err = document.ParseInTransactionWithIntegrations(deps.Registry, target.factory.OrganizationID.String(), tx, reads.integrationsByID)
		} else {
			prepared.Nodes, prepared.Edges, err = document.ParseInTransaction(deps.Registry, target.factory.OrganizationID.String(), tx)
		}
		if err != nil {
			return nil, configurationPreparationError("with", err.Error())
		}
		for _, node := range prepared.Nodes {
			if node.ErrorMessage != nil {
				field := "with"
				if strings.Contains(*node.ErrorMessage, "is not enabled for integration") {
					field = "binding.integration_id"
				}
				return nil, configurationPreparationError(field, fmt.Sprintf("%s: %s", node.Name, *node.ErrorMessage))
			}
		}
	}
	prepared.Settings, err = exportFactoryConfigurationBuiltIn(&models.CanvasVersion{Nodes: prepared.Nodes, Edges: prepared.Edges}, templateID)
	if err != nil {
		return nil, err
	}
	state, err := configurationPreparedState(tx, target.factory, binding, integration, agent, prepared.Nodes)
	if err != nil {
		return nil, err
	}
	configurationPreparedRecordSettings(prepared, automation.With, prepared.Settings)
	if prepared.PRFeedbackSource == models.FactoryPRFeedbackHandlerSourcePullRequestChecks {
		section, _ := configurationSettingsSection(automation.With, "pr_feedback")
		feedback, err := parseFactoryConfigurationPRFeedbackSettings(prepared.PRFeedbackSource, section)
		if err != nil {
			return nil, prefixConfigurationPreparationSettings(err)
		}
		public, _ := prepared.Settings["pr_feedback"].(map[string]any)
		public["maximum_attempts"] = prepared.MaximumAttempts
		ids := make([]string, 0, len(feedback.RunnerIntegrationIDs))
		for _, raw := range feedback.RunnerIntegrationIDs {
			id, _ := uuid.Parse(raw)
			installation, err := factoryAutomationIntegrationByID(tx, target.factory.OrganizationID, id)
			if err != nil {
				return nil, configurationPreparationError("with.pr_feedback.runner_integration_ids", "Runner integrations must belong to this organization.")
			}
			if installation.State != models.IntegrationStateReady {
				return nil, configurationPreparationError("with.pr_feedback.runner_integration_ids", "Runner integrations must be ready.")
			}
			state.RunnerIntegrations = append(state.RunnerIntegrations, configurationIntegrationSnapshot(installation))
			ids = append(ids, id.String())
		}
		public["runner_integration_ids"] = ids
	}
	prepared.State = state
	return prepared, nil
}

// Workspace webhooks have no integration. Keep their installed trigger and
// field paths while applying the settings supported by workspace YAML.
func prepareInstalledConfigurationWebhookIntake(tx *gorm.DB, deps IntakeDependencies, factory *models.Factory, automation factoryConfigurationAutomation) (*preparedFactoryConfigurationBuiltIn, error) {
	current := automation.current
	if !equalCanvasJSON(current.exportedBinding, automation.Binding) {
		return nil, configurationPreparationError("binding", "Keep the existing workspace webhook binding.")
	}
	section, err := configurationSettingsSection(automation.With, "intake")
	if err != nil {
		return nil, err
	}
	source := models.FactoryIntakeSourceWebhook
	settings, err := parseFactoryConfigurationIntakeSettings(source, section)
	if err != nil {
		return nil, prefixConfigurationPreparationSettings(err)
	}
	if factory.OnboardingConfigValue().CodexSubscription {
		if _, present := section["default_owner_id"]; !present {
			settings.DefaultOwnerID = factory.OnboardingConfigValue().CodexDefaultOwnerID
		}
	}
	if err := validateConfigurationIntakeOwner(tx, factory, source, settings.DefaultOwnerID); err != nil {
		return nil, configurationPreparationError("with.intake.default_owner_id", err.Error())
	}
	nodes, edges, err := copyFactoryConfigurationBuiltIn(current.version, current.defaults, current.templateID, automation.With)
	if err != nil {
		return nil, prefixConfigurationPreparationSettings(err)
	}
	spec := models.LiveCanvasSpec{Nodes: nodes, Edges: edges}
	nodes, edges, err = applyIntakeSettingsToGraph(source, resolveIntakeGraph(source, spec), spec, &pb.FactoryIntake_Settings{
		ConfidencePct: int32(settings.ConfidencePct), DefaultOwnerId: proto.String(settings.DefaultOwnerID),
	}, nodes, edges)
	if err != nil {
		return nil, configurationPreparationError("with.intake", err.Error())
	}
	if err := validateFactoryConfigurationBuiltInNodes(tx, deps, factory, nodes); err != nil {
		return nil, prefixConfigurationPreparationSettings(err)
	}
	public, err := exportFactoryConfigurationBuiltIn(&models.CanvasVersion{Nodes: nodes, Edges: edges}, current.templateID)
	if err != nil {
		return nil, err
	}
	prepared := &preparedFactoryConfigurationBuiltIn{
		TemplateID: current.templateID, Nodes: nodes, Edges: edges, Defaults: current.defaults,
		Settings: public, Binding: current.binding, ExplicitBinding: automation.Binding,
		IntakeSource: source, Description: current.canvas.Description,
	}
	configurationPreparedRecordSettings(prepared, automation.With, prepared.Settings)
	return prepared, nil
}

func configurationPreparationError(field, message string) error {
	return &factoryConfigurationSettingError{Field: field, Message: message}
}

func prefixConfigurationPreparationSettings(err error) error {
	return prefixFactoryConfigurationSettingError("with.", err)
}

func validateConfigurationBuiltInStage(templateID string, stage *factoryConfigurationStageTarget) error {
	kind := models.FactoryLineStagePhase
	switch {
	case templateID == models.FactoryAppTemplateBacklogID || templateID == scheduledIntakeTemplateID || templateID == "issue-intake" || strings.HasPrefix(templateID, "intake:"):
		kind = models.FactoryLineStageBacklog
	case templateID == "pr-closure":
		kind = models.FactoryLineStageDone
	case templateID != lineImplementationTemplateID:
		if stage.key != models.CanvasColumnKeyVerify {
			return configurationPreparationError("uses", "This built-in automation belongs in Verify.")
		}
		return nil
	}
	if stage.stage.Kind != kind {
		return configurationPreparationError("uses", "This built-in automation is not supported in this stage.")
	}
	return nil
}

func validateConfigurationBuiltInFeature(tx *gorm.DB, factory *models.Factory, templateID string) error {
	if templateID != scheduledIntakeTemplateID && templateID != reviewRequestTemplateID {
		return nil
	}
	organization, err := factoryAutomationPolicy(tx, factory, "configuration-organization", nil, func() (*models.Organization, error) {
		return models.FindOrganizationByIDInTransaction(tx, factory.OrganizationID.String())
	})
	if err != nil {
		return err
	}
	if templateID == scheduledIntakeTemplateID {
		if err := requireScheduledIntake(organization); err != nil {
			return configurationPreparationError("uses", err.Error())
		}
		return nil
	}
	if err := requireReviewRequest(organization); err != nil {
		return configurationPreparationError("uses", err.Error())
	}
	return nil
}

func configurationBuiltInNeedsAgent(templateID string) bool {
	return !strings.HasPrefix(templateID, "intake:") && templateID != "issue-intake" && templateID != "pr-closure"
}

func configurationBuiltInDefaultBinding(factory *models.Factory, templateID string) factoryConfigurationBinding {
	config := factory.OnboardingConfigValue()
	if source := strings.TrimPrefix(templateID, "intake:"); source != templateID && source != models.FactoryIntakeSourceGitHubIssues && source != models.FactoryIntakeSourceDependabotAlerts {
		return factoryConfigurationBinding{}
	}
	binding := factoryConfigurationBinding{IntegrationID: config.VCSIntegrationID, ResourceID: config.AppRepository}
	if templateID == "issue-intake" || templateID == intakeTemplateID(models.FactoryIntakeSourceGitHubIssues) || templateID == intakeTemplateID(models.FactoryIntakeSourceDependabotAlerts) {
		binding.ResourceID = config.BacklogRepository
	}
	if templateID == models.FactoryAppTemplateBacklogID {
		binding.ResourceID = ""
	}
	return binding
}

func resolveConfigurationBuiltInBinding(tx *gorm.DB, factory *models.Factory, templateID string, requested *factoryConfigurationBinding) (factoryConfigurationBinding, *models.Integration, error) {
	binding := configurationBuiltInDefaultBinding(factory, templateID)
	if requested != nil {
		if requested.IntegrationID != "" {
			binding.IntegrationID = requested.IntegrationID
		}
		if requested.ResourceID != "" {
			binding.ResourceID = requested.ResourceID
		}
		if requested.TeamID != "" {
			binding.TeamID = requested.TeamID
		}
	}
	source := strings.TrimPrefix(templateID, "intake:")
	provider := factoryVCSProviderOrDefault(factory)
	if source != templateID {
		provider = configurationIntakeProvider(source)
	}
	if provider == "" {
		return binding, nil, configurationPreparationError("uses", "This intake source is not supported.")
	}
	if factoryVCSProviderOrDefault(factory) == models.ProviderBitbucket && (provider == models.ProviderGitHub && (source != templateID || templateID == "issue-intake")) {
		return binding, nil, configurationPreparationError("uses", "This built-in automation is not supported for Bitbucket workspaces.")
	}
	if templateID == reviewRequestTemplateID || templateID == "visual-evidence" || templateID == "issue-intake" {
		if provider != models.ProviderGitHub {
			return binding, nil, configurationPreparationError("uses", "This built-in automation requires a GitHub workspace.")
		}
	}
	if binding.IntegrationID == "" {
		return binding, nil, configurationPreparationError("binding.integration_id", "Select an existing integration for this automation.")
	}
	integrationID, err := uuid.Parse(binding.IntegrationID)
	if err != nil {
		return binding, nil, configurationPreparationError("binding.integration_id", "Use an existing integration ID.")
	}
	integration, err := factoryAutomationIntegrationByID(tx, factory.OrganizationID, integrationID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return binding, nil, configurationPreparationError("binding.integration_id", "The integration must belong to this organization.")
	}
	if err != nil {
		return binding, nil, err
	}
	if integration.AppName != provider {
		return binding, nil, configurationPreparationError("binding.integration_id", "The integration provider does not match this automation.")
	}
	if integration.State != models.IntegrationStateReady {
		return binding, nil, configurationPreparationError("binding.integration_id", "The selected integration is not ready.")
	}
	if source == models.FactoryIntakeSourceDependabotAlerts && githubcommon.HostFromModel(integration).IsGHES() {
		return binding, nil, configurationPreparationError("binding.integration_id", "Dependabot alerts are not available on GitHub Enterprise Server.")
	}
	if templateID == models.FactoryAppTemplateBacklogID {
		if requested != nil && requested.ResourceID != "" {
			return binding, nil, configurationPreparationError("binding.resource_id", "Task analysis does not support a resource binding.")
		}
	} else if source == models.FactoryIntakeSourceLinearIssues {
		if binding.TeamID == "" && binding.ResourceID == "" {
			return binding, nil, configurationPreparationError("binding.team_id", "Select a Linear team or project list.")
		}
	} else if binding.ResourceID == "" {
		return binding, nil, configurationPreparationError("binding.resource_id", "Select an existing resource for this automation.")
	}
	if binding.TeamID != "" && source != models.FactoryIntakeSourceLinearIssues {
		return binding, nil, configurationPreparationError("binding.team_id", "This built-in automation does not support a team binding.")
	}
	if binding.ResourceID != "" {
		if strings.TrimSpace(binding.ResourceID) != binding.ResourceID || strings.ContainsAny(binding.ResourceID, "\r\n") || containsTemplateExpression(binding.ResourceID) {
			return binding, nil, configurationPreparationError("binding.resource_id", "Use an explicit resource ID without surrounding spaces.")
		}
		if provider == models.ProviderGitHub || provider == models.ProviderBitbucket {
			parts := strings.Split(binding.ResourceID, "/")
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" || strings.ContainsAny(binding.ResourceID, " \t\"'\\") {
				return binding, nil, configurationPreparationError("binding.resource_id", "Use a repository in owner/name format.")
			}
		}
	}
	if integration.AppName == models.ProviderBitbucket && binding.ResourceID != "" {
		metadata := integration.Metadata.Data()
		if workspace, ok := metadata["workspace"].(map[string]any); ok {
			slug, _ := workspace["slug"].(string)
			if slug != "" && !strings.HasPrefix(binding.ResourceID, slug+"/") {
				return binding, nil, configurationPreparationError("binding.resource_id", "The repository must belong to the integration workspace.")
			}
		}
		if repository, ok := metadata["repository"].(map[string]any); ok {
			fullName, _ := repository["full_name"].(string)
			if fullName != "" && fullName != binding.ResourceID {
				return binding, nil, configurationPreparationError("binding.resource_id", "The repository is not available to this integration.")
			}
		}
	}
	// Hosted GitHub selections use the catalog and caller authorization, not a stale resource cache.
	if integration.AppName == models.ProviderGitHub {
		if _, err := factoryAutomationVCSBinding(tx, factory, integration.ID); err == nil {
			return binding, integration, nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return binding, nil, err
		}
	}
	if err := validateConfigurationCachedResource(integration, binding); err != nil {
		return binding, nil, err
	}
	return binding, integration, nil
}

func configurationIntakeProvider(source string) string {
	switch source {
	case models.FactoryIntakeSourceGitHubIssues, models.FactoryIntakeSourceDependabotAlerts:
		return models.ProviderGitHub
	case models.FactoryIntakeSourceJiraIssues:
		return intakeJiraAppName
	case models.FactoryIntakeSourceSentryExceptions:
		return intakeSentryAppName
	case models.FactoryIntakeSourcePagerDutyIncidents:
		return "pagerduty"
	case models.FactoryIntakeSourceProductiveTasks:
		return intakeProductiveAppName
	case models.FactoryIntakeSourceDatadog:
		return intakeDatadogAppName
	case models.FactoryIntakeSourceLinearIssues:
		return intakeLinearAppName
	default:
		return ""
	}
}

func configurationIntakeBinding(binding factoryConfigurationBinding, integration *models.Integration, source string) *intakeBinding {
	configuration := map[string]any{}
	switch source {
	case models.FactoryIntakeSourceGitHubIssues, models.FactoryIntakeSourceDependabotAlerts:
		configuration["repository"] = binding.ResourceID
	case models.FactoryIntakeSourceJiraIssues, models.FactoryIntakeSourceProductiveTasks:
		configuration["project"] = binding.ResourceID
	case models.FactoryIntakeSourcePagerDutyIncidents, models.FactoryIntakeSourceDatadog:
		configuration["service"] = binding.ResourceID
	case models.FactoryIntakeSourceSentryExceptions:
		applySentryProjectConfiguration(configuration, sentryProjectIDsFromResource(binding.ResourceID))
	case models.FactoryIntakeSourceLinearIssues:
		configuration = linearBindingConfiguration(binding.TeamID, linearProjectIDsFromResource(binding.ResourceID))
	}
	return &intakeBinding{Integration: &yaml.IntegrationRef{ID: integration.ID.String(), Name: integration.InstallationName}, Installation: integration, Configuration: configuration}
}

func configurationSettingsSection(settings map[string]any, section string) (map[string]any, error) {
	raw, present := settings[section]
	if !present {
		return nil, nil
	}
	fields, ok := raw.(map[string]any)
	if !ok {
		return nil, configurationPreparationError("with."+section, "Use a mapping of supported settings.")
	}
	return fields, nil
}

func buildConfigurationBuiltInGraph(tx *gorm.DB, factory *models.Factory, name, templateID string, binding factoryConfigurationBinding, integration *models.Integration, agent *intakeAgent, settings map[string]any, prepared *preparedFactoryConfigurationBuiltIn) (*yaml.Canvas, error) {
	if source := strings.TrimPrefix(templateID, "intake:"); source != templateID {
		section, err := configurationSettingsSection(settings, "intake")
		if err != nil {
			return nil, err
		}
		intakeSettings, err := parseFactoryConfigurationIntakeSettings(source, section)
		if err != nil {
			return nil, prefixConfigurationPreparationSettings(err)
		}
		if source == models.FactoryIntakeSourceLinearIssues {
			intakeSettings.LinearTeamID = binding.TeamID
			if binding.TeamID == "" {
				intakeSettings.LinearProjectIDs = linearProjectIDsFromResource(binding.ResourceID)
			}
			if err := validateConfigurationResourceSelections(integration, "projects", intakeSettings.LinearProjectIDs, "with.intake.linear_project_ids"); err != nil {
				return nil, err
			}
		}
		if _, explicit := section["sentry_project_ids"]; source == models.FactoryIntakeSourceSentryExceptions && !explicit {
			intakeSettings.SentryProjectIDs = sentryProjectIDsFromResource(binding.ResourceID)
		}
		if factory.OnboardingConfigValue().CodexSubscription {
			if _, present := section["default_owner_id"]; !present {
				intakeSettings.DefaultOwnerID = factory.OnboardingConfigValue().CodexDefaultOwnerID
			}
		}
		if err := validateConfigurationIntakeOwner(tx, factory, source, intakeSettings.DefaultOwnerID); prepared != nil && err != nil {
			return nil, configurationPreparationError("with.intake.default_owner_id", err.Error())
		}
		resolved := configurationIntakeBinding(binding, integration, source)
		if source == models.FactoryIntakeSourceLinearIssues {
			resolved = withLinearIntakeScope(resolved, intakeSettings)
		}
		request := intakeCanvasRequest{OrganizationID: factory.OrganizationID, FactoryID: factory.ID, Name: name, Source: source, Binding: resolved, Settings: intakeSettings}
		document, err := buildIntakeCanvas(request)
		if err != nil {
			return nil, err
		}
		nodes, edges, err := applyDefaultAuthorAccess(request, document.Nodes(), document.Edges())
		if err != nil {
			return nil, err
		}
		encoded, err := yaml.VersionToCanvasYAML(name, document.Metadata.Description, &models.CanvasVersion{Nodes: nodes, Edges: edges})
		if err != nil {
			return nil, err
		}
		document, err = yaml.CanvasFromYAML([]byte(encoded))
		if err != nil {
			return nil, err
		}
		stampFactoryTemplateMetadata(document, intakeTriggerNodeID, templateID)
		if prepared != nil {
			prepared.IntakeSource = source
			prepared.Paused, _ = section["paused"].(bool)
		}
		return document, nil
	}
	if templateID == models.FactoryAppTemplateBacklogID {
		return buildBacklogCanvas(backlogCanvasRequest{Name: name, Agent: agent, VCSName: integration.InstallationName, VCSProvider: factoryVCSProviderOrDefault(factory)}), nil
	}
	if templateID == prFeedbackDiscussionTemplateID || templateID == prFeedbackChecksTemplateID {
		return buildConfigurationPRFeedback(tx, factory, name, templateID, binding, integration, agent, settings, prepared)
	}
	document, err := buildConfigurationFactoryTemplate(tx, factory, name, templateID, binding, integration, agent, settings)
	if err != nil {
		return nil, err
	}
	if slices.Contains([]string{"risk-score", "pr-closure", "visual-evidence"}, templateID) {
		repository := models.WorkspaceRepositoriesExpression
		if prepared != nil && prepared.ExplicitBinding != nil && prepared.ExplicitBinding.ResourceID != "" {
			repository = binding.ResourceID
		}
		for i := range document.Spec.Nodes {
			node := &document.Spec.Nodes[i]
			if node.Type == yaml.NodeTypeTrigger && strings.HasPrefix(node.Component, factoryVCSProviderOrDefault(factory)+".") {
				node.Configuration["repository"] = repository
			}
		}
	}
	return document, nil
}

func buildConfigurationPRFeedback(tx *gorm.DB, factory *models.Factory, name, templateID string, binding factoryConfigurationBinding, integration *models.Integration, agent *intakeAgent, settings map[string]any, prepared *preparedFactoryConfigurationBuiltIn) (*yaml.Canvas, error) {
	source := models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion
	if templateID == prFeedbackChecksTemplateID {
		source = models.FactoryPRFeedbackHandlerSourcePullRequestChecks
	}
	section, err := configurationSettingsSection(settings, "pr_feedback")
	if err != nil {
		return nil, err
	}
	feedback := defaultPRFeedbackSettings()
	if prepared != nil {
		feedback, err = parseFactoryConfigurationPRFeedbackSettings(source, section)
		if err != nil {
			return nil, prefixConfigurationPreparationSettings(err)
		}
	}
	feedback.Repository = binding.ResourceID
	// Defaults can be exported before a check list is selected. Preparation of
	// the requested graph still requires the provider's public settings.
	if prepared != nil {
		_, err := factoryAutomationPolicy(tx, factory, "configuration-pr-feedback-settings", []any{source, feedback}, func() (bool, error) {
			err := validatePRFeedbackSettingsForSource(tx, factory.OrganizationID, source, feedback, nil)
			return err == nil, err
		})
		if err != nil {
			return nil, configurationPreparationError("with.pr_feedback.check_names", err.Error())
		}
	}
	names, err := factoryAutomationPolicy(tx, factory, "configuration-pr-feedback-runner-names", []any{feedback.RunnerIntegrationIDs}, func() ([]string, error) {
		resolved := feedback
		if err := resolveRunnerIntegrationNames(tx, factory.OrganizationID, &resolved); err != nil {
			return nil, err
		}
		return resolved.RunnerIntegrationNames, nil
	})
	if err != nil {
		return nil, configurationPreparationError("with.pr_feedback.runner_integration_ids", err.Error())
	}
	feedback.RunnerIntegrationNames = names
	request := prFeedbackBuildRequest{Name: name, Repository: feedback.Repository, Mention: feedback.Mention, IgnoreBots: feedback.IgnoreBots, AddressAllBots: feedback.AddressAllBots, AllowedBots: feedback.AllowedBots, CheckNames: feedback.CheckNames, MaximumAttempts: feedback.MaximumAttempts, RunnerIntegrationNames: feedback.RunnerIntegrationNames, Binding: configurationIntakeBinding(binding, integration, models.FactoryIntakeSourceGitHubIssues), Agent: agent, VCSProvider: factoryVCSProviderOrDefault(factory)}
	document := buildDiscussionPRFeedbackCanvas(request)
	trigger := prFeedbackCommentTriggerNodeID
	if templateID == prFeedbackChecksTemplateID {
		document = buildChecksPRFeedbackCanvas(request)
		trigger = prFeedbackPullRequestTriggerNodeID
	}
	stampFactoryTemplateMetadata(document, trigger, templateID)
	if prepared != nil && prepared.ExplicitBinding != nil && prepared.ExplicitBinding.ResourceID != "" {
		for i := range document.Spec.Nodes {
			node := &document.Spec.Nodes[i]
			if node.Type == yaml.NodeTypeTrigger && strings.HasPrefix(node.Component, factoryVCSProviderOrDefault(factory)+".") {
				node.Configuration["repository"] = binding.ResourceID
			}
		}
	}
	if prepared != nil {
		prepared.PRFeedbackSource = source
		prepared.MaximumAttempts = feedback.MaximumAttempts
	}
	return document, nil
}

func buildConfigurationFactoryTemplate(tx *gorm.DB, factory *models.Factory, name, templateID string, binding factoryConfigurationBinding, integration *models.Integration, agent *intakeAgent, settings map[string]any) (*yaml.Canvas, error) {
	config := factory.OnboardingConfigValue()
	input := factoryTemplateInput{appID: factoryCanvasIDPlaceholder, appName: name, installParams: map[string]string{"appRepository": binding.ResourceID, "backlogRepository": config.BacklogRepository, "defaultBranch": config.DefaultBranch, "githubHost": githubHostForVCSIntegration(tx, factory.OrganizationID, integration.ID.String()), "mergeConfidenceAppId": "__FACTORY_MERGE_CONFIDENCE__"}, integrations: map[string]factoryTemplateIntegration{integration.AppName: {id: integration.ID.String(), name: integration.InstallationName}}, agent: factoryTemplateAgentFromIntake(agent)}
	if templateID == "issue-intake" {
		input.installParams["backlogRepository"] = binding.ResourceID
	}
	if config.CodexSubscription && templateID == scheduledIntakeTemplateID {
		input.installParams["defaultOwnerId"] = config.CodexDefaultOwnerID
	}
	section, err := configurationSettingsSection(settings, "template")
	if err != nil {
		return nil, err
	}
	if err := applyFactoryConfigurationTemplateSettings(templateID, section, &input); err != nil {
		return nil, prefixConfigurationPreparationSettings(err)
	}
	if templateID == scheduledIntakeTemplateID {
		owner := input.installParams["defaultOwnerId"]
		if err := validateConfigurationIntakeOwner(tx, factory, models.FactoryIntakeSourceGitHubIssues, owner); err != nil {
			return nil, configurationPreparationError("with.template.default_owner_id", err.Error())
		}
	}
	result, err := materializeFactoryTemplate(templateID, factoryVCSProviderOrDefault(factory), input)
	if err != nil {
		return nil, configurationPreparationError("with.template", err.Error())
	}
	document, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
	if err != nil {
		return nil, err
	}
	if templateID == "issue-intake" {
		for i := range document.Spec.Nodes {
			node := &document.Spec.Nodes[i]
			if node.Component == "github.onIssue" {
				node.Integration = &yaml.IntegrationRef{ID: integration.ID.String(), Name: integration.InstallationName}
			}
		}
	}
	return document, nil
}

func configurationPreparedState(tx *gorm.DB, factory *models.Factory, binding factoryConfigurationBinding, integration *models.Integration, agent *intakeAgent, nodes []models.Node) (configurationBuiltInState, error) {
	config := factory.OnboardingConfigValue()
	state := configurationBuiltInState{Binding: binding, Integration: configurationIntegrationSnapshot(integration), DefaultOwner: config.CodexDefaultOwnerID, Repository: config.AppRepository, Backlog: config.BacklogRepository, Branch: config.DefaultBranch, GitProvider: config.EffectiveVCSProvider(), Subscription: config.CodexSubscription}
	repositories, err := factoryAutomationPolicy(tx, factory, "configuration-workspace-repositories", nil, func() ([]string, error) {
		if reads := factoryAutomationReads(tx, factory); reads != nil && reads.workspaceRepositories != nil {
			return reads.workspaceRepositories, nil
		}
		return factory.WorkspaceRepositories(tx)
	})
	if err != nil {
		return state, err
	}
	state.WorkspaceRepositories = repositories
	if agent == nil {
		return state, nil
	}
	state.AgentHarness, state.AgentModel = agent.component(), agent.model()
	if agent.Component == models.SuperPlaneRunnerComponent {
		model, err := factoryAutomationPolicy(tx, factory, "configuration-default-hosted-model", nil, func() (models.DefaultHostedLLMModel, error) {
			return models.GetInstallationDefaultHostedLLMModel(tx)
		})
		if err != nil {
			return state, err
		}
		state.HostedModel = model
		providers := map[string]bool{}
		if model.IsSet() {
			providers[model.Provider] = true
		}
		for _, node := range nodes {
			if node.ComponentName() != models.SuperPlaneRunnerComponent {
				continue
			}
			selected, err := models.ParseSelectableLLMModelKey(configString(node.Configuration, "model"))
			if err == nil && selected.Source.ID == models.UsageFundingSourceHosted {
				providers[selected.Provider.ID] = true
			}
		}
		for provider := range providers {
			row, err := factoryAutomationPolicy(tx, factory, "configuration-hosted-provider", []any{provider}, func() (*models.HostedLLMProvider, error) {
				return models.FindHostedLLMProvider(tx, provider)
			})
			if errors.Is(err, models.ErrHostedLLMProviderNotFound) {
				continue
			}
			if err != nil {
				return state, err
			}
			allowed := slices.Clone([]string(row.AllowedModels))
			slices.Sort(allowed)
			state.HostedProviders = append(state.HostedProviders, configurationHostedProviderState{Provider: row.Provider, Enabled: row.Enabled, BaseURL: row.BaseURL, AllowedModels: allowed, Ready: row.CanRunHosted()})
		}
		slices.SortFunc(state.HostedProviders, func(a, b configurationHostedProviderState) int { return strings.Compare(a.Provider, b.Provider) })
	}
	credentials, _ := agent.credentials()["integration"].(map[string]any)
	name, _ := credentials["name"].(string)
	if name == "" {
		return state, nil
	}
	installation, err := factoryAutomationIntegrationByName(tx, factory.OrganizationID, name)
	if err != nil {
		return state, err
	}
	snapshot := configurationIntegrationSnapshot(installation)
	state.Agent = &snapshot
	return state, nil
}

func configurationIntegrationSnapshot(integration *models.Integration) configurationIntegrationState {
	state := configurationIntegrationState{ID: integration.ID, Provider: integration.AppName, Name: integration.InstallationName, State: integration.State, Capabilities: slices.Clone(integration.Capabilities)}
	configuration := maps.Clone(integration.Configuration.Data())
	state.Host, _ = configuration["host"].(string)
	state.Configuration = map[string]any{}
	for _, key := range []string{"host", "baseUrl", "baseURL", "apiType", "site", "authType", "region", "organization", "organizationId", "subdomain"} {
		if value, present := configuration[key]; present {
			state.Configuration[key] = value
		}
	}
	state.Resources = map[string]any{}
	metadata := integration.Metadata.Data()
	for _, key := range []string{"repositories", "repository", "workspace", "authType", "forgeInstallationId", "owner", "githubApp", "projects", "teams", "services", "organization", "organizationId", "installationId", "installationUUID", "hostedApp", "appSlug", "cloudId", "siteUrl", "issueWebhookScopesRequested", "oauthScopes"} {
		if value, present := metadata[key]; present {
			state.Resources[key] = configurationResourceSnapshot(value)
		}
	}
	state.Properties = map[string]any{}
	for _, property := range integration.Properties {
		if slices.Contains([]string{"authMethod", "owner", "ownerType", "appID", "appInstallationID", "host"}, property.Name) {
			state.Properties[property.Name] = property.Value
		}
	}
	return state
}

type configurationHostedProviderState struct {
	Provider      string
	Enabled       bool
	Ready         bool
	BaseURL       string
	AllowedModels []string
}

// Export compares public settings with a new installation. Reset defaults
// intentionally retain installed settings and cannot serve as this baseline.
func buildFreshFactoryConfigurationBuiltInDefaults(tx *gorm.DB, factory *models.Factory, templateID string, binding factoryConfigurationBinding) (*models.CanvasVersion, error) {
	defaults := configurationBuiltInDefaultBinding(factory, templateID)
	if binding.IntegrationID != "" {
		defaults.IntegrationID = binding.IntegrationID
	}
	if binding.ResourceID != "" {
		defaults.ResourceID = binding.ResourceID
	}
	if binding.TeamID != "" {
		defaults.TeamID = binding.TeamID
	}
	provider := factoryVCSProviderOrDefault(factory)
	if source := strings.TrimPrefix(templateID, "intake:"); source != templateID {
		provider = configurationIntakeProvider(source)
	}
	integration := &models.Integration{AppName: provider, InstallationName: provider}
	if id, err := uuid.Parse(defaults.IntegrationID); err == nil {
		integration.ID = id
		if existing, err := factoryAutomationIntegrationByID(tx, factory.OrganizationID, id); err == nil {
			integration = existing
		}
	}
	document, err := buildConfigurationBuiltInGraph(tx, factory, "Defaults", templateID, defaults, integration, resolveIntakeAgent(tx, factory), nil, nil)
	if err != nil {
		return nil, err
	}
	return &models.CanvasVersion{Nodes: document.Nodes(), Edges: document.Edges()}, nil
}

// Integration sync caches provider selections. Validate those caches without
// fetching resources or credentials during plan.
func validateConfigurationCachedResource(integration *models.Integration, binding factoryConfigurationBinding) error {
	key, selection, field := "", binding.ResourceID, "binding.resource_id"
	switch integration.AppName {
	case models.ProviderGitHub, models.ProviderBitbucket:
		key = "repositories"
	case intakeJiraAppName, intakeSentryAppName:
		key = "projects"
	case "pagerduty":
		key = "services"
	case intakeLinearAppName:
		if binding.TeamID != "" {
			key, selection, field = "teams", binding.TeamID, "binding.team_id"
		}
	}
	if key == "" || selection == "" {
		return nil
	}
	selections := []string{selection}
	if integration.AppName == intakeSentryAppName {
		selections = sentryProjectIDsFromResource(selection)
	}
	return validateConfigurationResourceSelections(integration, key, selections, field)
}

func validateConfigurationResourceSelections(integration *models.Integration, key string, selections []string, field string) error {
	if len(selections) == 0 {
		return nil
	}
	raw, present := integration.Metadata.Data()[key]
	if !present {
		return nil
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	var resources []map[string]any
	if err := json.Unmarshal(encoded, &resources); err != nil {
		return configurationPreparationError(field, "The integration resource list is invalid. Sync the integration and plan again.")
	}
	if integration.AppName == models.ProviderGitHub && key == "repositories" {
		host := githubcommon.HostFromModel(integration)
		for _, resource := range resources {
			rawURL, _ := resource["url"].(string)
			if owner, repository, ok := host.ParseRepoURL(rawURL); ok {
				resource["full_name"] = owner + "/" + repository
			}
		}
	}
	for _, selected := range selections {
		found := slices.ContainsFunc(resources, func(resource map[string]any) bool {
			for _, name := range []string{"id", "key", "name", "slug", "full_name", "fullName", "uuid"} {
				if value, exists := resource[name]; exists && fmt.Sprint(value) == selected {
					return true
				}
			}
			return false
		})
		if !found {
			return configurationPreparationError(field, "The selected resource is not available to this integration.")
		}
	}
	return nil
}

func configurationResourceSnapshot(value any) any {
	switch current := value.(type) {
	case []any:
		result := make([]any, 0, len(current))
		for _, item := range current {
			result = append(result, configurationResourceSnapshot(item))
		}
		return result
	case map[string]any:
		result := map[string]any{}
		for _, key := range []string{"id", "key", "name", "slug", "url", "full_name", "fullName", "uuid", "identifier"} {
			if item, present := current[key]; present {
				result[key] = item
			}
		}
		return result
	default:
		return current
	}
}

// A project list can supply the provider resource. Reject conflicting copies
// so reset never chooses between different binding and setting values.
func configurationBuiltInSettingsBinding(templateID string, automation factoryConfigurationAutomation) (*factoryConfigurationBinding, string, error) {
	source := strings.TrimPrefix(templateID, "intake:")
	if source != models.FactoryIntakeSourceSentryExceptions && source != models.FactoryIntakeSourceLinearIssues {
		return automation.Binding, "", nil
	}
	section, err := configurationSettingsSection(automation.With, "intake")
	if err != nil {
		return nil, "", err
	}
	field := "sentry_project_ids"
	if source == models.FactoryIntakeSourceLinearIssues {
		field = "linear_project_ids"
	}
	if _, explicit := section[field]; !explicit {
		return automation.Binding, "", nil
	}
	settings, err := parseFactoryConfigurationIntakeSettings(source, section)
	if err != nil {
		return nil, "", prefixConfigurationPreparationSettings(err)
	}
	binding := factoryConfigurationBinding{}
	if automation.Binding != nil {
		binding = *automation.Binding
	}
	projects := settings.SentryProjectIDs
	if source == models.FactoryIntakeSourceLinearIssues {
		projects = settings.LinearProjectIDs
	}
	selected := strings.Join(projects, ",")
	path := "with.intake." + field
	if binding.ResourceID != "" && binding.ResourceID != selected {
		return nil, "", configurationPreparationError(path, "Use the same project selection as binding.resource_id, or omit binding.resource_id.")
	}
	if source == models.FactoryIntakeSourceLinearIssues && binding.TeamID != "" {
		return &binding, "", nil
	}
	if selected == "" {
		return nil, "", configurationPreparationError(path, "Select at least one project or bind a Linear team.")
	}
	binding.ResourceID = selected
	return &binding, path, nil
}

func configurationPreparedRecordSettings(prepared *preparedFactoryConfigurationBuiltIn, requested, settings map[string]any) {
	if prepared.IntakeSource != "" {
		section, _ := settings["intake"].(map[string]any)
		section["paused"] = prepared.Paused
	}
	if prepared.PRFeedbackSource == models.FactoryPRFeedbackHandlerSourcePullRequestChecks {
		section, _ := settings["pr_feedback"].(map[string]any)
		section["maximum_attempts"] = prepared.MaximumAttempts
		ids := []string{}
		requestedSection, _ := requested["pr_feedback"].(map[string]any)
		if raw, present := requestedSection["runner_integration_ids"]; present {
			ids, _ = configurationStringList(raw)
		}
		section["runner_integration_ids"] = ids
	}
}

func configurationPublicSectionsEqual(current, requested map[string]any) bool {
	for _, section := range []string{"intake", "pr_feedback", "template"} {
		before, _ := current[section].(map[string]any)
		after, _ := requested[section].(map[string]any)
		if section == "template" {
			before, after = maps.Clone(before), maps.Clone(after)
			delete(before, "merge_confidence")
			delete(after, "merge_confidence")
		}
		if !equalCanvasJSON(before, after) {
			return false
		}
	}
	return true
}

func validateConfigurationIntakeOwner(tx *gorm.DB, factory *models.Factory, source, owner string) error {
	_, err := factoryAutomationPolicy(tx, factory, "configuration-intake-owner", []any{source, owner, factory.OnboardingConfigValue().CodexSubscription}, func() (bool, error) {
		err := validateIntakeDefaultOwner(tx, factory, source, "", &pb.FactoryIntake_Settings{DefaultOwnerId: &owner})
		return err == nil, err
	})
	return err
}
