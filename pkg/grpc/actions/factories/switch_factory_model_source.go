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
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const planningAgentNodeID = "planner-agent-no-issue"

const (
	modelSourceHosted     = "hosted"
	modelSourceAnthropic  = "anthropic"
	modelSourceOpenAI     = "openai"
	modelSourceOpenRouter = "openrouter"
)

type workspaceAgentRewrite struct {
	Component       string
	Model           string
	PlanningModel   string
	IntegrationName string
	Harness         string
	Provider        string
}

func workspaceAgentRewriteFor(source, integrationName string) (workspaceAgentRewrite, error) {
	switch strings.TrimSpace(source) {
	case modelSourceHosted:
		return workspaceAgentRewrite{
			Component: models.SuperPlaneRunnerComponent,
			Harness:   models.FactoryOnboardingAgentHarnessSuperPlane,
		}, nil
	case modelSourceAnthropic:
		return workspaceAgentRewrite{
			Component:       "runnerClaudeCode",
			Model:           "sonnet",
			PlanningModel:   "opus",
			IntegrationName: integrationName,
			Harness:         models.FactoryOnboardingAgentHarnessClaudeCode,
			Provider:        models.UsageProviderAnthropic,
		}, nil
	case modelSourceOpenAI:
		return workspaceAgentRewrite{
			Component:       "runnerCodex",
			Model:           "gpt-5",
			PlanningModel:   "gpt-5",
			IntegrationName: integrationName,
			Harness:         models.FactoryOnboardingAgentHarnessCodex,
			Provider:        models.UsageProviderOpenAI,
		}, nil
	case modelSourceOpenRouter:
		return workspaceAgentRewrite{
			Component:       "runnerOpenRouter",
			Model:           "anthropic/claude-sonnet-4-6",
			PlanningModel:   "anthropic/claude-opus-4-6",
			IntegrationName: integrationName,
			Harness:         models.FactoryOnboardingAgentHarnessClaudeCode,
			Provider:        models.UsageProviderOpenRouter,
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
		return true
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
		err := tx.Model(&models.CanvasNode{}).
			Where("workflow_id = ? AND node_id = ?", canvas.ID, node.ID).
			Updates(map[string]any{
				"ref":           datatypes.NewJSONType(node.Ref),
				"configuration": datatypes.NewJSONType(node.Configuration),
				"updated_at":    now,
			}).Error
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func ensureWorkspaceProviderIntegration(
	ctx context.Context,
	tx *gorm.DB,
	encryptor crypto.Encryptor,
	orgID uuid.UUID,
	provider string,
	apiKey string,
) (*models.Integration, error) {
	existing, err := models.FindReadyBYOKIntegration(tx, orgID, provider)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if strings.TrimSpace(apiKey) != "" {
			if err := storeIntegrationAPIKey(ctx, tx, encryptor, existing, apiKey); err != nil {
				return nil, err
			}
		}
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

func storeIntegrationAPIKey(
	ctx context.Context,
	tx *gorm.DB,
	encryptor crypto.Encryptor,
	integration *models.Integration,
	apiKey string,
) error {
	encrypted, err := encryptAPIKey(ctx, encryptor, integration.ID, apiKey)
	if err != nil {
		return err
	}
	config := integration.Configuration.Data()
	if config == nil {
		config = map[string]any{}
	}
	config["apiKey"] = encrypted
	now := time.Now()
	integration.Configuration = datatypes.NewJSONType(config)
	integration.UpdatedAt = &now
	return tx.Model(integration).Updates(map[string]any{
		"configuration": integration.Configuration,
		"updated_at":    now,
	}).Error
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
	encryptor crypto.Encryptor,
	factory *models.Factory,
	source string,
	apiKey string,
) ([]uuid.UUID, string, error) {
	var integration *models.Integration
	if strings.TrimSpace(source) != modelSourceHosted {
		provider := strings.TrimSpace(source)
		saved, err := ensureWorkspaceProviderIntegration(ctx, tx, encryptor, factory.OrganizationID, provider, apiKey)
		if err != nil {
			return nil, "", err
		}
		integration = saved
	}

	integrationName := ""
	integrationID := ""
	if integration != nil {
		integrationName = integration.InstallationName
		integrationID = integration.ID.String()
	}
	rewrite, err := workspaceAgentRewriteFor(source, integrationName)
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
	encryptor crypto.Encryptor,
	organizationID string,
	factoryID string,
	source string,
	apiKey string,
) ([]uuid.UUID, string, error) {
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
		changed, savedID, err = switchFactoryModelSource(ctx, tx, encryptor, factory, source, apiKey)
		integrationID = savedID
		return err
	})
	return changed, integrationID, err
}
