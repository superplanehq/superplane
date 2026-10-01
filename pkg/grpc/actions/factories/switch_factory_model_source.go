package factories

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/grpc/actions/organizations"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/integrations/customllm"
	"github.com/superplanehq/superplane/pkg/llm"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const planningAgentNodeID = "planner-agent-no-issue"

const (
	modelSourceHosted     = "hosted"
	modelSourceAnthropic  = "anthropic"
	modelSourceOpenAI     = "openai"
	modelSourceOpenRouter = "openrouter"
	modelSourceCustom     = "custom"
)

type workspaceAgentRewrite struct {
	Component       string
	Model           string
	PlanningModel   string
	IntegrationName string
	Harness         string
	Provider        string
}

func workspaceAgentRewriteFor(source, integrationName string, modelIDs []string) (workspaceAgentRewrite, error) {
	switch strings.TrimSpace(source) {
	case modelSourceHosted:
		return workspaceAgentRewrite{
			Component: models.SuperPlaneRunnerComponent,
			Harness:   models.FactoryOnboardingAgentHarnessSuperPlane,
		}, nil
	case modelSourceAnthropic:
		model, planning := agentModelsForSource(modelSourceAnthropic, modelIDs)
		return workspaceAgentRewrite{
			Component:       "runnerClaudeCode",
			Model:           model,
			PlanningModel:   planning,
			IntegrationName: integrationName,
			Harness:         models.FactoryOnboardingAgentHarnessClaudeCode,
			Provider:        models.UsageProviderAnthropic,
		}, nil
	case modelSourceOpenAI:
		model, planning := agentModelsForSource(modelSourceOpenAI, modelIDs)
		return workspaceAgentRewrite{
			Component:       "runnerCodex",
			Model:           model,
			PlanningModel:   planning,
			IntegrationName: integrationName,
			Harness:         models.FactoryOnboardingAgentHarnessCodex,
			Provider:        models.UsageProviderOpenAI,
		}, nil
	case modelSourceOpenRouter:
		model, planning := agentModelsForSource(modelSourceOpenRouter, modelIDs)
		return workspaceAgentRewrite{
			Component:       "runnerOpenRouter",
			Model:           model,
			PlanningModel:   planning,
			IntegrationName: integrationName,
			Harness:         models.FactoryOnboardingAgentHarnessClaudeCode,
			Provider:        models.UsageProviderOpenRouter,
		}, nil
	case modelSourceCustom:
		model, planning := agentModelsForSource(modelSourceCustom, modelIDs)
		if model == "" {
			return workspaceAgentRewrite{}, grpcerrors.FailedPrecondition(nil, "The provider did not return any models. Check the URL, token, and API type.")
		}
		// runnerOpenRouter starts the OpenCode CLI. The custom provider block
		// replaces the OpenRouter provider in that OpenCode config.
		return workspaceAgentRewrite{
			Component:       "runnerOpenRouter",
			Model:           model,
			PlanningModel:   planning,
			IntegrationName: integrationName,
			Harness:         models.FactoryOnboardingAgentHarnessClaudeCode,
			Provider:        models.UsageProviderCustom,
		}, nil
	default:
		return workspaceAgentRewrite{}, grpcerrors.InvalidArgument(nil, "unsupported model source")
	}
}

func isWorkspaceAgentComponent(name string) bool {
	switch name {
	case models.SuperPlaneRunnerComponent, "runnerClaudeCode", "runnerCodex", "runnerOpenRouter":
		return true
	default:
		return false
	}
}

func rewriteWorkspaceAgentNode(node *models.Node, rewrite workspaceAgentRewrite) bool {
	if node == nil || !isWorkspaceAgentComponent(node.ComponentName()) {
		return false
	}
	if node.Ref.Component == nil {
		node.Ref.Component = &models.ComponentRef{}
	}
	node.Ref.Component.Name = rewrite.Component
	if node.Configuration == nil {
		node.Configuration = map[string]any{}
	}
	if rewrite.Component == models.SuperPlaneRunnerComponent {
		delete(node.Configuration, "credentials")
		delete(node.Configuration, "model")
		delete(node.Configuration, "maxTurns")
		delete(node.Configuration, "llmProvider")
		return true
	}

	if rewrite.Provider == models.UsageProviderCustom {
		node.Configuration["llmProvider"] = models.UsageProviderCustom
	} else {
		delete(node.Configuration, "llmProvider")
	}
	node.Configuration["credentials"] = map[string]any{
		"source": "integration",
		"integration": map[string]any{
			"name": rewrite.IntegrationName,
		},
	}
	node.Configuration["model"] = rewrite.Model
	if node.ID == planningAgentNodeID && rewrite.PlanningModel != "" {
		node.Configuration["model"] = rewrite.PlanningModel
	}
	return true
}

func rewriteFactoryCanvases(
	tx *gorm.DB,
	factory *models.Factory,
	ownerID uuid.UUID,
	rewrite workspaceAgentRewrite,
) ([]uuid.UUID, error) {
	canvases, err := factory.ListCanvases(tx)
	if err != nil {
		return nil, err
	}

	changed := make([]uuid.UUID, 0)
	for i := range canvases {
		canvas := &canvases[i]
		if canvas.LiveVersionID == nil {
			continue
		}
		updated, err := rewriteCanvasAgents(tx, canvas, ownerID, rewrite)
		if err != nil {
			return nil, err
		}
		if updated {
			changed = append(changed, canvas.ID)
		}
	}
	return changed, nil
}

func rewriteCanvasAgents(
	tx *gorm.DB,
	canvas *models.Canvas,
	ownerID uuid.UUID,
	rewrite workspaceAgentRewrite,
) (bool, error) {
	live, err := models.FindLiveCanvasVersionInTransaction(tx, canvas.ID)
	if err != nil {
		return false, err
	}

	nodes := slices.Clone(live.Nodes)
	rewritten := make([]models.Node, 0)
	for i := range nodes {
		if rewriteWorkspaceAgentNode(&nodes[i], rewrite) {
			rewritten = append(rewritten, nodes[i])
		}
	}
	if len(rewritten) == 0 {
		return false, nil
	}

	owner := ownerID
	if owner == uuid.Nil && live.OwnerID != nil {
		owner = *live.OwnerID
	}
	version, err := models.CreateCommitVersionWithSpecInTransaction(
		tx,
		canvas.ID,
		owner,
		"Switch model source",
		nodes,
		nil,
	)
	if err != nil {
		return false, err
	}
	if err := models.PromoteToLiveInTransaction(tx, version, nodes, live.Edges); err != nil {
		return false, err
	}

	now := time.Now()
	for _, node := range rewritten {
		if err := freezeActiveExecutionComponent(tx, canvas.ID, node.ID); err != nil {
			return false, err
		}
		err = tx.Model(&models.CanvasNode{}).
			Where("workflow_id = ? AND node_id = ?", canvas.ID, node.ID).
			Updates(map[string]any{
				"ref":           datatypes.NewJSONType(node.Ref),
				"configuration": datatypes.NewJSONType(node.Configuration),
				"state":         models.CanvasNodeStateReady,
				"state_reason":  nil,
				"updated_at":    now,
			}).Error
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func freezeActiveExecutionComponent(tx *gorm.DB, canvasID uuid.UUID, nodeID string) error {
	var runtime models.CanvasNode
	err := tx.Clauses(clause.Locking{Strength: "NO KEY UPDATE"}).
		Where("workflow_id = ? AND node_id = ?", canvasID, nodeID).
		First(&runtime).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	component := runtime.Ref.Data().Component
	if component == nil || strings.TrimSpace(component.Name) == "" {
		return nil
	}

	var executions []models.CanvasNodeExecution
	err = tx.Where("workflow_id = ? AND node_id = ? AND state IN ?", canvasID, nodeID, models.CanvasNodeExecutionActiveStates).
		Find(&executions).Error
	if err != nil {
		return err
	}
	for i := range executions {
		if executions[i].FrozenComponentName() != "" {
			continue
		}
		metadata := executions[i].Metadata.Data()
		if metadata == nil {
			metadata = map[string]any{}
		}
		metadata[models.CanvasNodeExecutionFrozenComponentKey] = component.Name
		if err := tx.Model(&executions[i]).Update("metadata", datatypes.NewJSONType(metadata)).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureWorkspaceProviderIntegration(
	ctx context.Context,
	tx *gorm.DB,
	encryptor crypto.Encryptor,
	orgID uuid.UUID,
	provider string,
	apiKey string,
	baseURL string,
	apiType string,
) (*models.Integration, error) {
	existing, err := models.FindReadyBYOKIntegration(tx, orgID, provider)
	if err != nil {
		return nil, err
	}
	if provider == modelSourceCustom {
		return ensureCustomProviderIntegration(ctx, tx, encryptor, orgID, existing, apiKey, baseURL, apiType)
	}
	if existing != nil {
		return existing, nil
	}

	key := strings.TrimSpace(apiKey)
	if key == "" {
		return nil, grpcerrors.InvalidArgument(nil, "api key is required")
	}

	appName, err := models.BYOKIntegrationAppName(provider)
	if err != nil {
		return nil, err
	}
	installationName, err := nextInstallationName(tx, orgID, appName)
	if err != nil {
		return nil, err
	}

	integrationID := uuid.New()
	encrypted, err := encryptAPIKey(ctx, encryptor, integrationID, key)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	integration := &models.Integration{
		ID:               integrationID,
		OrganizationID:   orgID,
		AppName:          appName,
		InstallationName: installationName,
		State:            models.IntegrationStateReady,
		Configuration:    datatypes.NewJSONType(map[string]any{"apiKey": encrypted}),
		CreatedAt:        &now,
		UpdatedAt:        &now,
	}
	if err := tx.Create(integration).Error; err != nil {
		return nil, err
	}
	return integration, nil
}

func ensureCustomProviderIntegration(
	ctx context.Context,
	tx *gorm.DB,
	encryptor crypto.Encryptor,
	orgID uuid.UUID,
	existing *models.Integration,
	apiKey string,
	baseURL string,
	apiType string,
) (*models.Integration, error) {
	enabled, err := models.OrganizationHasExperimentalFeatures(
		tx,
		orgID,
		features.FeatureOrganizationBYOK,
		features.FeatureOrganizationBYOKCustomProvider,
	)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, grpcerrors.PermissionDenied(nil, "Custom provider is not enabled for this organization.")
	}

	key := strings.TrimSpace(apiKey)
	url := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	parsedType := strings.TrimSpace(apiType)
	if existing != nil && key == "" && url == "" && parsedType == "" {
		return existing, nil
	}
	if url == "" {
		return nil, grpcerrors.InvalidArgument(nil, "Enter a public http or https URL.")
	}
	if err := llm.ValidateBaseURL(url); err != nil {
		return nil, grpcerrors.InvalidArgument(err, "Enter a public http or https URL.")
	}
	normalizedType, err := customllm.NormalizeAPIType(parsedType)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, err.Error())
	}
	if existing == nil && key == "" {
		return nil, grpcerrors.InvalidArgument(nil, "API token is required.")
	}

	if existing != nil {
		return updateCustomProviderIntegration(ctx, tx, encryptor, existing, key, url, normalizedType)
	}

	appName, err := models.BYOKIntegrationAppName(modelSourceCustom)
	if err != nil {
		return nil, err
	}
	installationName, err := nextInstallationName(tx, orgID, appName)
	if err != nil {
		return nil, err
	}
	integrationID := uuid.New()
	encrypted, err := encryptAPIKey(ctx, encryptor, integrationID, key)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	integration := &models.Integration{
		ID:               integrationID,
		OrganizationID:   orgID,
		AppName:          appName,
		InstallationName: installationName,
		State:            models.IntegrationStateReady,
		Configuration: datatypes.NewJSONType(map[string]any{
			"apiKey":  encrypted,
			"baseURL": url,
			"apiType": normalizedType,
		}),
		CreatedAt: &now,
		UpdatedAt: &now,
	}
	if err := tx.Create(integration).Error; err != nil {
		return nil, err
	}
	return integration, nil
}

func updateCustomProviderIntegration(
	ctx context.Context,
	tx *gorm.DB,
	encryptor crypto.Encryptor,
	existing *models.Integration,
	apiKey string,
	baseURL string,
	apiType string,
) (*models.Integration, error) {
	data := existing.Configuration.Data()
	if data == nil {
		data = map[string]any{}
	}
	if apiKey != "" {
		encrypted, err := encryptAPIKey(ctx, encryptor, existing.ID, apiKey)
		if err != nil {
			return nil, err
		}
		data["apiKey"] = encrypted
	}
	data["baseURL"] = baseURL
	data["apiType"] = apiType
	now := time.Now()
	existing.Configuration = datatypes.NewJSONType(data)
	existing.UpdatedAt = &now
	if err := tx.Save(existing).Error; err != nil {
		return nil, err
	}
	return existing, nil
}

func encryptAPIKey(ctx context.Context, encryptor crypto.Encryptor, integrationID uuid.UUID, apiKey string) (string, error) {
	if encryptor == nil {
		return "", fmt.Errorf("encryptor is required")
	}
	encrypted, err := encryptor.Encrypt(ctx, []byte(apiKey), []byte(integrationID.String()))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func nextInstallationName(tx *gorm.DB, orgID uuid.UUID, appName string) (string, error) {
	name := appName
	for i := 2; i < 100; i++ {
		_, err := models.FindIntegrationByName(tx, orgID, name)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return name, nil
			}
			return "", err
		}
		name = fmt.Sprintf("%s-%d", appName, i)
	}
	return "", fmt.Errorf("could not choose an installation name for %s", appName)
}

func switchFactoryModelSource(
	ctx context.Context,
	tx *gorm.DB,
	reg *registry.Registry,
	factory *models.Factory,
	source string,
	apiKey string,
	baseURL string,
	apiType string,
) ([]uuid.UUID, string, error) {
	var integration *models.Integration
	if strings.TrimSpace(source) != modelSourceHosted {
		provider := strings.TrimSpace(source)
		saved, err := ensureWorkspaceProviderIntegration(ctx, tx, reg.Encryptor, factory.OrganizationID, provider, apiKey, baseURL, apiType)
		if err != nil {
			return nil, "", err
		}
		integration = saved
	}

	integrationName := ""
	integrationID := ""
	var modelIDs []string
	if integration != nil {
		integrationName = integration.InstallationName
		integrationID = integration.ID.String()
		ids, err := organizations.ListConnectedBYOKModelIDs(tx, reg, integration)
		if strings.TrimSpace(source) == modelSourceCustom {
			if err != nil {
				return nil, "", err
			}
			modelIDs = ids
		} else if err != nil && !providerModelListUsesDefaults(err) {
			return nil, "", err
		} else if err != nil {
			log.WithError(err).Warn("model list unavailable; switch uses default agent models")
		} else {
			modelIDs = ids
		}
	}
	rewrite, err := workspaceAgentRewriteFor(source, integrationName, modelIDs)
	if err != nil {
		return nil, "", err
	}

	patch := models.FactoryOnboardingPatch{AgentHarness: &rewrite.Harness}
	if integration != nil {
		id := integration.ID.String()
		patch.AgentIntegrationID = &id
	}
	if rewrite.Component == models.SuperPlaneRunnerComponent {
		cleared := ""
		patch.AgentIntegrationID = &cleared
	}
	if err := factory.UpdateOnboarding(tx, patch); err != nil {
		return nil, "", err
	}

	ownerID := versionOwnerID(ctx)
	changed, err := rewriteFactoryCanvases(tx, factory, ownerID, rewrite)
	if err != nil {
		return nil, "", err
	}
	return changed, integrationID, nil
}

// providerModelListUsesDefaults reports a temporary catalog failure.
// An invalid key still stops the switch. A rate limit, outage, or transport
// error uses the provider default models, the same path as an empty list.
func providerModelListUsesDefaults(err error) bool {
	var providerErr *core.ProviderAPIError
	if !errors.As(err, &providerErr) {
		return false
	}
	return providerErr.IsRateLimited() || providerErr.IsUnavailable() || providerErr.IsTransport()
}

func versionOwnerID(ctx context.Context) uuid.UUID {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return uuid.Nil
	}
	parsed, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil
	}
	return parsed
}

func SwitchFactoryModelSourceInTransaction(
	ctx context.Context,
	reg *registry.Registry,
	organizationID string,
	factoryID string,
	source string,
	apiKey string,
	baseURL string,
	apiType string,
) ([]uuid.UUID, string, error) {
	if reg == nil {
		return nil, "", fmt.Errorf("integration registry is required")
	}

	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, "", err
	}

	var changed []uuid.UUID
	var integrationID string
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		factory, err := findFactory(tx, orgID, factoryID)
		if err != nil {
			return err
		}
		var savedID string
		changed, savedID, err = switchFactoryModelSource(ctx, tx, reg, factory, source, apiKey, baseURL, apiType)
		integrationID = savedID
		return err
	})
	return changed, integrationID, err
}
