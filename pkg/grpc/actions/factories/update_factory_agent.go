package factories

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

type factoryAgentPlan struct {
	provider        string
	component       string
	harness         string
	integrationID   string
	integrationName string
	model           string
	planningModel   string
}

type factoryAgentProviderSpec struct {
	provider       string
	integrationApp string
	component      string
	harness        string
	model          string
	planningModel  string
	modelHint      string
	planningHint   string
}

var factoryAgentProviderSpecs = map[string]factoryAgentProviderSpec{
	models.FactoryOnboardingAgentProviderAnthropic: {
		provider:       models.FactoryOnboardingAgentProviderAnthropic,
		integrationApp: "claude",
		component:      "runnerClaudeCode",
		harness:        models.FactoryOnboardingAgentHarnessClaudeCode,
		model:          "sonnet",
		planningModel:  "opus",
		modelHint:      "sonnet",
		planningHint:   "opus",
	},
	models.FactoryOnboardingAgentProviderOpenAI: {
		provider:       models.FactoryOnboardingAgentProviderOpenAI,
		integrationApp: "openai",
		component:      "runnerCodex",
		harness:        models.FactoryOnboardingAgentHarnessCodex,
		model:          "gpt-5",
		planningModel:  "gpt-5",
		modelHint:      "gpt-5",
		planningHint:   "gpt-5",
	},
	models.FactoryOnboardingAgentProviderOpenRouter: {
		provider:       models.FactoryOnboardingAgentProviderOpenRouter,
		integrationApp: "openrouter",
		component:      "runnerOpenRouter",
		harness:        models.FactoryOnboardingAgentHarnessClaudeCode,
		model:          "anthropic/claude-sonnet-4-6",
		planningModel:  "anthropic/claude-opus-4-6",
		modelHint:      "sonnet",
		planningHint:   "opus",
	},
}

func UpdateFactoryAgent(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.UpdateFactoryAgentRequest,
) (*pb.UpdateFactoryAgentResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory agent")
	}
	factoryID, err := parseFactoryID(req.GetId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory agent")
	}
	actorID, err := factoryAgentActorID(ctx)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory agent")
	}

	db := database.DB(ctx)
	var factory *models.Factory
	err = db.Transaction(func(tx *gorm.DB) error {
		loaded, err := models.FindFactory(tx, orgID, factoryID)
		if err != nil {
			return err
		}
		factory = loaded

		plan, err := resolveFactoryAgentPlan(tx, orgID, factoryID, req)
		if err != nil {
			return err
		}
		if err := factory.UpdateOnboarding(tx, models.FactoryOnboardingPatch{
			AgentIntegrationID: &plan.integrationID,
			AgentHarness:       &plan.harness,
			AgentProvider:      &plan.provider,
			AgentModel:         &plan.model,
			AgentPlanningModel: &plan.planningModel,
		}); err != nil {
			return err
		}

		return reconcileFactoryAgent(ctx, tx, deps, factory, actorID, plan)
	})
	if err != nil {
		if _, _, ok := grpcerrors.HandlerStatus(err); ok {
			return nil, err
		}
		return nil, factoryErrorToStatus(err, "failed to update factory agent")
	}

	lines, err := factory.ListLines(db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory agent")
	}
	serialized, err := serializeFactoryWithLineMetrics(db, factory, lines)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory agent")
	}
	return &pb.UpdateFactoryAgentResponse{Factory: serialized}, nil
}

func factoryAgentActorID(ctx context.Context) (uuid.UUID, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return uuid.Nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	actorID, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, invalidArgument("invalid user id")
	}
	return actorID, nil
}

func resolveFactoryAgentPlan(
	tx *gorm.DB,
	organizationID, factoryID uuid.UUID,
	req *pb.UpdateFactoryAgentRequest,
) (factoryAgentPlan, error) {
	provider, err := factoryAgentProviderFromProto(req.GetProvider())
	if err != nil {
		return factoryAgentPlan{}, err
	}
	spec := factoryAgentProviderSpecs[provider]
	if spec.provider == "" {
		return factoryAgentPlan{}, invalidArgument("unsupported agent provider")
	}

	switch req.GetCredentialSource() {
	case pb.FactoryOnboarding_AGENT_CREDENTIAL_SOURCE_HOSTED:
		if req.IntegrationId != nil && strings.TrimSpace(req.GetIntegrationId()) != "" {
			return factoryAgentPlan{}, invalidArgument("hosted credentials do not use an integration")
		}
		credit, err := models.DescribeOrganizationLLMCredit(tx, organizationID)
		if err != nil {
			return factoryAgentPlan{}, err
		}
		if credit.RemainingMicros <= 0 {
			return factoryAgentPlan{}, invalidArgument("SuperPlane-hosted credit is empty")
		}
		modelsForFactory, err := models.ResolveSelectableLLMModels(
			tx,
			organizationID,
			&factoryID,
			provider,
			models.UsageFundingSourceHosted,
		)
		if err != nil {
			return factoryAgentPlan{}, err
		}
		model, planningModel, ok := factoryHostedAgentModels(modelsForFactory, spec)
		if !ok {
			return factoryAgentPlan{}, invalidArgument("SuperPlane-hosted models are not available for this provider")
		}
		return factoryAgentPlan{
			provider: provider, component: spec.component, harness: spec.harness, model: model, planningModel: planningModel,
		}, nil
	case pb.FactoryOnboarding_AGENT_CREDENTIAL_SOURCE_INTEGRATION:
		integrationID := strings.TrimSpace(req.GetIntegrationId())
		if integrationID == "" {
			return factoryAgentPlan{}, invalidArgument("integration id is required")
		}
		integration, err := findReadyOnboardingIntegration(tx, organizationID, integrationID)
		if err != nil {
			return factoryAgentPlan{}, err
		}
		if integration.AppName != spec.integrationApp {
			return factoryAgentPlan{}, invalidArgument("agent provider does not match the selected integration")
		}
		return factoryAgentPlan{
			provider: provider, component: spec.component, harness: spec.harness,
			integrationID: integration.ID.String(), integrationName: integration.InstallationName,
			model: spec.model, planningModel: spec.planningModel,
		}, nil
	default:
		return factoryAgentPlan{}, invalidArgument("agent credential source is required")
	}
}

func factoryHostedAgentModels(modelsForFactory []string, spec factoryAgentProviderSpec) (string, string, bool) {
	modelsForFactory = models.CompactModelIDs(modelsForFactory)
	if len(modelsForFactory) == 0 {
		return "", "", false
	}
	slices.SortFunc(modelsForFactory, func(left, right string) int { return strings.Compare(strings.ToLower(left), strings.ToLower(right)) })
	model := factoryAgentModelMatching(modelsForFactory, spec.modelHint)
	if model == "" {
		model = modelsForFactory[0]
	}
	planningModel := factoryAgentModelMatching(modelsForFactory, spec.planningHint)
	if planningModel == "" {
		planningModel = model
	}
	return model, planningModel, true
}

func factoryAgentModelMatching(modelIDs []string, hint string) string {
	for _, model := range modelIDs {
		if strings.Contains(strings.ToLower(model), hint) {
			return model
		}
	}
	return ""
}

func factoryAgentProviderFromProto(provider pb.FactoryOnboarding_AgentProvider) (string, error) {
	return factoryOnboardingAgentProviderFromProto(provider)
}

func reconcileFactoryAgent(
	ctx context.Context,
	tx *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	actorID uuid.UUID,
	plan factoryAgentPlan,
) error {
	canvasesForFactory, err := factory.ListCanvases(tx)
	if err != nil {
		return err
	}

	for i := range canvasesForFactory {
		canvas := &canvasesForFactory[i]
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
		if err != nil {
			return err
		}
		nodes := slices.Clone(liveVersion.Nodes)
		changed := reconcileGeneratedFactoryAgentNodes(nodes, plan)
		if !changed {
			continue
		}
		if err := canvases.PublishGeneratedCanvasNodes(
			ctx,
			tx,
			canvas,
			actorID,
			"Update workspace agent",
			nodes,
			slices.Clone(liveVersion.Edges),
			changesets.CanvasPublisherOptions{
				Registry:       deps.Registry,
				OrgID:          canvas.OrganizationID,
				Encryptor:      deps.Encryptor,
				AuthService:    deps.AuthService,
				WebhookBaseURL: deps.WebhookBaseURL,
				GitProvider:    deps.GitProvider,
			},
		); err != nil {
			return fmt.Errorf("publish %q agent update: %w", canvas.Name, err)
		}
	}
	return nil
}

func reconcileGeneratedFactoryAgentNodes(nodes []models.Node, plan factoryAgentPlan) bool {
	template, isTemplate := resolveFactoryTemplate(nodes)
	if isTemplate {
		switch template.id {
		case "line-planning":
			return updateFactoryAgentNode(nodes, "planner-agent-no-issue", plan, plan.planningModel)
		case "line-implementation":
			return updateFactoryAgentNode(nodes, "implementation-agent-no-issue", plan, plan.model)
		}
	}
	return updateGeneratedBacklogAgentNode(nodes, plan)
}

func updateGeneratedBacklogAgentNode(nodes []models.Node, plan factoryAgentPlan) bool {
	if !isGeneratedBacklog(nodes) {
		return false
	}
	return updateFactoryAgentNode(nodes, intakeAnalysisNodeID, plan, plan.model)
}

func isGeneratedBacklog(nodes []models.Node) bool {
	hasBacklogTrigger := false
	hasAnalysisNode := false
	for _, node := range nodes {
		if node.ID == backlogTriggerNodeID && node.ComponentName() == "onWorkOrder" && node.Name == backlogTriggerName {
			hasBacklogTrigger = true
		}
		if node.ID == intakeAnalysisNodeID && node.Name == intakeAnalysisNodeName {
			hasAnalysisNode = true
		}
	}
	return hasBacklogTrigger && hasAnalysisNode
}

func updateFactoryAgentNode(nodes []models.Node, nodeID string, plan factoryAgentPlan, model string) bool {
	for i := range nodes {
		if nodes[i].ID != nodeID {
			continue
		}
		nodes[i].Ref.Component = &models.ComponentRef{Name: plan.component}
		nodes[i].Configuration = maps.Clone(nodes[i].Configuration)
		if nodes[i].Configuration == nil {
			nodes[i].Configuration = map[string]any{}
		}
		if plan.integrationID == "" {
			nodes[i].Configuration["credentials"] = map[string]any{"source": "hosted"}
		} else {
			nodes[i].Configuration["credentials"] = map[string]any{
				"source":      "integration",
				"integration": map[string]any{"name": plan.integrationName},
			}
		}
		nodes[i].Configuration["model"] = model
		return true
	}
	return false
}
