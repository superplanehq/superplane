package factories

import (
	"context"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/llm"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

// Test__DispatchWorkOrder__CreatesLineDispatchWithSnapshot covers acceptance
// criterion 1: dispatching a work order creates one line dispatch with the
// line's current steps snapshotted, and step 1's execution references it.
func Test__DispatchWorkOrder__CreatesLineDispatchWithSnapshot(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	creator := support.CreateUser(t, r, r.Organization.ID)
	owner := support.CreateUser(t, r, r.Organization.ID)
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &creator.ID, []uuid.UUID{creator.ID, owner.ID}, nil)
	require.NoError(t, err)

	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	resp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)

	require.Len(t, resp.Order.LineDispatches, 1)
	dispatch := resp.Order.LineDispatches[0]
	assert.Equal(t, pb.WorkOrderLineDispatch_STATE_ACTIVE, dispatch.State)
	assert.Equal(t, line.Name, dispatch.Line.Name)
	assert.Empty(t, dispatch.Model)
	require.Len(t, dispatch.Steps, 1)
	assert.Equal(t, app.Name, dispatch.Steps[0].Name)
	require.Len(t, dispatch.StepExecutions, 1)
	assert.Equal(t, pb.WorkOrderExecution_STATE_PENDING, dispatch.StepExecutions[0].State)

	reloaded, err := factoryModel.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateOpen, reloaded.State)
	assert.ElementsMatch(t, []uuid.UUID{creator.ID, owner.ID}, reloaded.AssigneeIDs())

	active, err := order.FindActiveLineDispatch(db)
	require.NoError(t, err)
	assert.Equal(t, dispatch.Id, active.ID.String())
}

func TestAutoDispatchThinkingLevelsRequireCompatibleLineRunners(t *testing.T) {
	accepted := map[string][]string{
		runnerCodex:                          {"xhigh", "high", "default"},
		runnerClaudeCode:                     {"xhigh", "max", "high", "default"},
		models.SuperPlaneRunnerComponent:     {"xhigh", "high", "default"},
		runnerOpenRouter:                     {"minimal", "xhigh", "max", "high", "default"},
		runnerCodex + "+" + runnerClaudeCode: {"xhigh", "high", "default"},
	}
	for key, levels := range accepted {
		components := strings.Split(key, "+")
		for _, level := range []string{"minimal", "xhigh", "max", "high", "default"} {
			t.Run(key+"/"+level, func(t *testing.T) {
				r := support.Setup(t)
				db := database.DB(t.Context())
				require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
				modelByComponent := map[string]string{
					runnerCodex: "gpt-5.4", runnerClaudeCode: "claude-opus-5-5", runnerOpenRouter: "deepseek/model",
					models.SuperPlaneRunnerComponent: "hosted::openai::gpt-5.4",
				}
				for provider, model := range map[string]string{
					models.UsageProviderOpenAI: "gpt-5.4", models.UsageProviderAnthropic: "claude-opus-5-5", models.UsageProviderOpenRouter: "deepseek/model",
				} {
					_, err := models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, provider, []string{model})
					require.NoError(t, err)
				}
				seedHostedModels(t, db, models.UsageProviderOpenAI, "gpt-5.4")
				require.NoError(t, models.ReplaceLLMModelCapabilities(db, models.UsageProviderOpenRouter, llm.DefaultOpenRouterBaseURL, []models.LLMModelCapability{{
					Model: "deepseek/model", ThinkingLevels: datatypes.NewJSONSlice([]string{"minimal", "high", "xhigh", "max"}),
				}}))
				factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
				require.NoError(t, err)
				order, err := factory.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
				require.NoError(t, err)
				var steps []models.FactoryLineStep
				for _, component := range components {
					app := createLineAppWithRunner(t, r, factory.ID, component, "integration", modelByComponent[component])
					steps = append(steps, models.FactoryLineStep{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"})
				}
				line, err := factory.CreateLine(db, "ship", steps)
				require.NoError(t, err)
				ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
				response, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
					FactoryId: factory.ID.String(), OrderId: order.ID.String(), LineName: line.Name, ThinkingLevel: level,
				})
				if slices.Contains(levels, level) {
					require.NoError(t, err)
					require.Len(t, response.Order.LineDispatches, 1)
					assert.Equal(t, level, response.Order.LineDispatches[0].ThinkingLevel)
					return
				}
				assertDispatchRejected(t, db, order, err)
			})
		}
	}
}

func TestDispatchThinkingLevelMustBeSupportedBySelectedModel(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	seedHostedModels(t, db, models.UsageProviderAnthropic, "claude-opus-5-5", "claude-haiku-4-5")
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	app := createLineAppWithRunner(t, r, factory.ID, runnerClaudeCode, "hosted", "claude-opus-5-5")
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())

	order, err := factory.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factory.ID.String(), OrderId: order.ID.String(), LineName: line.Name,
		Model: "claude-haiku-4-5", ThinkingLevel: "high",
	})
	assertDispatchRejected(t, db, order, err)

	response, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factory.ID.String(), OrderId: order.ID.String(), LineName: line.Name,
		Model: "claude-opus-5-5", ThinkingLevel: "max",
	})
	require.NoError(t, err)
	require.Len(t, response.Order.LineDispatches, 1)
	assert.Equal(t, "max", response.Order.LineDispatches[0].ThinkingLevel)
}

func TestDispatchOpenRouterThinkingFollowsProviderCatalog(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	const model = "deepseek/deepseek-v4.1-flash"
	seedHostedModels(t, db, models.UsageProviderOpenRouter, model)
	require.NoError(t, models.ReplaceLLMModelCapabilities(db, models.UsageProviderOpenRouter, llm.DefaultOpenRouterBaseURL, []models.LLMModelCapability{{
		Model: model, ThinkingLevels: datatypes.NewJSONSlice([]string{"max", "high", "low", "none"}), RefreshedAt: time.Now(),
	}}))
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	app := createLineAppWithRunner(t, r, factory.ID, runnerOpenRouter, "hosted", model)
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
	options, err := ListFactoryLineRunnerModels(ctx, r.Organization.ID.String(), &pb.ListFactoryLineRunnerModelsRequest{
		FactoryId: factory.ID.String(), LineName: line.Name,
	})
	require.NoError(t, err)
	require.Len(t, options.Models, 1)
	assert.Equal(t, []string{"none", "low", "high", "max"}, options.Models[0].ThinkingLevels)

	order, err := factory.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factory.ID.String(), OrderId: order.ID.String(), LineName: line.Name,
		Model: model, ThinkingLevel: "medium",
	})
	assertDispatchRejected(t, db, order, err)
	response, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factory.ID.String(), OrderId: order.ID.String(), LineName: line.Name,
		Model: model, ThinkingLevel: "max",
	})
	require.NoError(t, err)
	require.Len(t, response.Order.LineDispatches, 1)
	assert.Equal(t, "max", response.Order.LineDispatches[0].ThinkingLevel)
}

func TestDispatchCodexSubscriptionThinkingFollowsCatalog(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	connection, err := models.CreateCodexConnection(db, r.Organization.ID, r.User, time.Now())
	require.NoError(t, err)
	require.NoError(t, connection.StoreUsage(db, nil, []models.CodexModel{
		{Model: "gpt-6-astra", IsDefault: true, SupportedReasoningEfforts: []models.CodexReasoningEffortOption{
			{ReasoningEffort: "low"}, {ReasoningEffort: "max"}, {ReasoningEffort: "ultra"}, {ReasoningEffort: "future-effort"},
		}},
		{Model: "gpt-6-luna", SupportedReasoningEfforts: []models.CodexReasoningEffortOption{{ReasoningEffort: "max"}}},
	}, time.Now()))
	require.NoError(t, db.Model(connection).Updates(map[string]any{"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("bundle")}).Error)
	app := createLineAppWithRunner(t, r, factory.ID, runnerCodex, "subscription", "")
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
	for _, tc := range []struct {
		name, model, effort string
		savedModel          string
		savedEffort         string
		refreshCatalog      bool
		wantErr             bool
	}{
		{name: "max", model: "gpt-6-astra", effort: "max"},
		{name: "ultra", model: "gpt-6-astra", effort: "ultra"},
		{name: "future catalog effort", model: "gpt-6-astra", effort: "future-effort"},
		{name: "native default model", effort: "ultra"},
		{name: "saved model expression defers explicit effort validation", savedModel: "{{ 'gpt-6-astra' }}", effort: "ultra"},
		{name: "saved model expression keeps inherited effort", savedModel: "{{ 'gpt-6-astra' }}", savedEffort: "ultra"},
		{name: "concrete override replaces saved model expression", savedModel: "{{ 'gpt-6-luna' }}", model: "gpt-6-astra", effort: "ultra"},
		{name: "concrete override still rejects unsupported effort", savedModel: "{{ 'gpt-6-astra' }}", model: "gpt-6-luna", effort: "ultra", wantErr: true},
		{name: "concrete saved model rejects unsupported effort", savedModel: "gpt-6-luna", effort: "ultra", wantErr: true},
		{name: "unreported effort", model: "gpt-6-astra", effort: "high", wantErr: true},
		{name: "unsupported model effort", model: "gpt-6-luna", effort: "ultra", wantErr: true},
		{name: "unsupported native default effort", effort: "high", wantErr: true},
		{name: "inherited effort with model override", model: "gpt-6-luna", savedEffort: "ultra", wantErr: true},
		{name: "supported inherited effort", model: "gpt-6-astra", savedEffort: "ultra"},
		{name: "default clears inherited effort", model: "gpt-6-luna", savedEffort: "ultra", effort: "default"},
		{name: "catalog refresh invalidates saved effort", savedEffort: "ultra", refreshCatalog: true, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			version, err := models.FindLiveCanvasVersionInTransaction(db, app.ID)
			require.NoError(t, err)
			for i := range version.Nodes {
				if version.Nodes[i].ComponentName() == runnerCodex {
					version.Nodes[i].Configuration["model"] = ""
					if tc.savedEffort != "" {
						version.Nodes[i].Configuration["model"] = "gpt-6-astra"
					}
					if tc.savedModel != "" {
						version.Nodes[i].Configuration["model"] = tc.savedModel
					}
					version.Nodes[i].Configuration["thinkingLevel"] = tc.savedEffort
				}
			}
			require.NoError(t, db.Model(version).Update("nodes", version.Nodes).Error)
			if tc.refreshCatalog {
				require.NoError(t, connection.StoreUsage(db, nil, []models.CodexModel{{
					Model: "gpt-6-astra", IsDefault: true,
					SupportedReasoningEfforts: []models.CodexReasoningEffortOption{{ReasoningEffort: "low"}},
				}}, time.Now()))
			}
			order, err := factory.CreateWorkOrder(db, "Ship it", "", &r.User, []uuid.UUID{r.User}, nil)
			require.NoError(t, err)
			response, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
				FactoryId: factory.ID.String(), OrderId: order.ID.String(), LineName: line.Name,
				Model: tc.model, ThinkingLevel: tc.effort,
			})
			if tc.wantErr {
				require.ErrorContains(t, err, "thinking level")
				assertDispatchRejected(t, db, order, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, response.Order.LineDispatches, 1)
			assert.Equal(t, tc.effort, response.Order.LineDispatches[0].ThinkingLevel)
		})
	}
}

func TestDispatchThinkingValidatesOnlyStepsToExecute(t *testing.T) {
	for _, tc := range []struct {
		name        string
		retry       bool
		invalidStep int
		wantErr     bool
	}{
		{name: "start skips earlier agent", invalidStep: 0},
		{name: "retry skips earlier agent", retry: true, invalidStep: 0},
		{name: "retry checks selected agent", retry: true, invalidStep: 1, wantErr: true},
		{name: "retry checks later agent", retry: true, invalidStep: 2, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			seedHostedModels(t, db, models.UsageProviderAnthropic, "claude-sonnet-5-5")
			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			var steps []models.FactoryLineStep
			for i := range 3 {
				app := createLineAppWithRunner(t, r, factory.ID, runnerClaudeCode, "hosted", "claude-sonnet-5-5")
				level := "low"
				if i == tc.invalidStep {
					level = "high"
				}
				setLineAppRunnerThinkingLevel(t, db, app, level)
				steps = append(steps, models.FactoryLineStep{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"})
			}
			line, err := factory.CreateLine(db, "ship", steps)
			require.NoError(t, err)
			order, err := factory.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
			require.NoError(t, err)
			ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
			var originalDispatchID string
			if tc.retry {
				response, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
					FactoryId: factory.ID.String(), OrderId: order.ID.String(), LineName: line.Name,
				})
				require.NoError(t, err)
				originalDispatchID = response.Order.LineDispatches[0].Id
				dispatch, err := models.FindWorkOrderLineDispatch(db, uuid.MustParse(originalDispatchID))
				require.NoError(t, err)
				require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
					if _, err := dispatch.EnqueueOrStartStep(tx, order, 1); err != nil {
						return err
					}
					return dispatch.Finish(tx, models.CanvasRunResultFailed)
				}))
			}
			// A catalog refresh removes high after the original traversal.
			require.NoError(t, models.UpsertLLMModelCapabilities(db, []models.LLMModelCapability{{
				Provider: models.UsageProviderAnthropic, BaseURL: llm.DefaultAnthropicBaseURL,
				Model: "claude-sonnet-5-5", ThinkingLevels: datatypes.NewJSONSlice([]string{"low"}), RefreshedAt: time.Now(),
			}}))
			response, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
				FactoryId: factory.ID.String(), OrderId: order.ID.String(), LineName: line.Name,
				StartStepIndex: 1, ReplaceActive: tc.retry,
			})
			if tc.wantErr {
				require.ErrorContains(t, err, "thinking level")
				assertDispatchRejected(t, db, order, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, response.Order.LineDispatches, 1)
			dispatch := response.Order.LineDispatches[0]
			assert.Equal(t, int32(1), dispatch.StepExecutions[len(dispatch.StepExecutions)-1].StepIndex)
			if tc.retry {
				assert.Equal(t, originalDispatchID, dispatch.Id)
				var earlierExecutions int
				for _, execution := range dispatch.StepExecutions {
					if execution.StepIndex == 0 {
						earlierExecutions++
					}
				}
				assert.Equal(t, 1, earlierExecutions)
			}
		})
	}
}

func TestDispatchThinkingLevelWithModelMustBeAcceptedByEveryLineRunner(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	seedHostedModels(t, db, models.UsageProviderAnthropic, "claude-opus-5-5")
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	claudeApp := createLineAppWithRunner(t, r, factory.ID, runnerClaudeCode, "hosted", "claude-opus-5-5")
	codexApp := createLineAppWithRunner(t, r, factory.ID, runnerCodex, "integration", "gpt-5-codex")
	steps := []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: claudeApp.ID, Entrypoint: "start"},
		{Type: models.FactoryLineStepTypeRunApp, AppID: codexApp.ID, Entrypoint: "start"},
	}
	line, err := factory.CreateLine(db, "ship", steps)
	require.NoError(t, err)
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())

	order, err := factory.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factory.ID.String(), OrderId: order.ID.String(), LineName: line.Name,
		Model: "claude-opus-5-5", ThinkingLevel: "max",
	})
	assertDispatchRejected(t, db, order, err)

	response, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factory.ID.String(), OrderId: order.ID.String(), LineName: line.Name,
		Model: "claude-opus-5-5", ThinkingLevel: "high",
	})
	require.NoError(t, err)
	require.Len(t, response.Order.LineDispatches, 1)
}

func TestDispatchThinkingChecksTheRetainedSuperPlaneModel(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	seedHostedModels(t, db, models.UsageProviderOpenAI, "gpt-5-codex")
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
	_, err := models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderAnthropic, []string{"claude-opus-5-5"})
	require.NoError(t, err)
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	claude := createLineAppWithRunner(t, r, factory.ID, runnerClaudeCode, "integration", "claude-opus-5-5")
	retained := createLineAppWithRunner(t, r, factory.ID, models.SuperPlaneRunnerComponent, "hosted", "hosted::openai::gpt-5-codex")
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: claude.ID, Entrypoint: "start"},
		{Type: models.FactoryLineStepTypeRunApp, AppID: retained.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
	order, err := factory.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factory.ID.String(), OrderId: order.ID.String(), LineName: line.Name,
		Model: "claude-opus-5-5", ThinkingLevel: "xhigh",
	})
	assertDispatchRejected(t, db, order, err)
	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factory.ID.String(), OrderId: order.ID.String(), LineName: line.Name,
		Model: "claude-opus-5-5", ThinkingLevel: "high",
	})
	require.NoError(t, err)
}

func TestDispatchThinkingLevelUsesProviderOfSelectedModelForSuperPlaneAgent(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	seedHostedModels(t, db, models.UsageProviderAnthropic, "claude-opus-5-5")
	seedHostedModels(t, db, models.UsageProviderOpenAI, "gpt-5.1")
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	savedModel := models.FormatSelectableLLMModelKey(models.UsageFundingSourceHosted, models.UsageProviderOpenAI, "gpt-5.1")
	app := createLineAppWithRunner(t, r, factory.ID, models.SuperPlaneRunnerComponent, "hosted", savedModel)
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
	selectedModel := models.FormatSelectableLLMModelKey(models.UsageFundingSourceHosted, models.UsageProviderAnthropic, "claude-opus-5-5")

	response, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factory.ID.String(), OrderId: order.ID.String(), LineName: line.Name,
		Model: selectedModel, ThinkingLevel: "max",
	})
	require.NoError(t, err)
	require.Len(t, response.Order.LineDispatches, 1)
	assert.Equal(t, "max", response.Order.LineDispatches[0].ThinkingLevel)
}

func assertDispatchRejected(t *testing.T, db *gorm.DB, order *models.FactoryWorkOrder, err error) {
	t.Helper()
	require.Error(t, err)
	code, _, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, code)
	_, err = order.FindActiveLineDispatch(db)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func Test__DispatchWorkOrder__ClearsAutoStartLine(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)
	creator := support.CreateUser(t, r, r.Organization.ID)
	owner := support.CreateUser(t, r, r.Organization.ID)
	order, err := factoryModel.CreateWorkOrderWithAutoStart(db, "Ship it", "", &creator.ID, []uuid.UUID{creator.ID, owner.ID}, nil, &line.ID)
	require.NoError(t, err)
	require.NotNil(t, order.AutoStartLineID)

	resp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Order.GetAutoStartLineId())

	reloaded, err := factoryModel.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateOpen, reloaded.State)
	assert.Nil(t, reloaded.AutoStartLineID)
	assert.ElementsMatch(t, []uuid.UUID{creator.ID, owner.ID}, reloaded.AssigneeIDs())
}

func Test__DispatchWorkOrder__CompletesActiveAnalysisRun(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	analysisCanvas := createOnWorkOrderCanvas(t, r, factoryModel.ID)
	analysisRun, err := models.CreateCanvasRunInTransaction(
		db,
		analysisCanvas.ID,
		"start",
		models.CanvasRunStateStarted,
		"",
	)
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, models.AttachAnalysisSessionParams{
		Repository:  "acme/payments",
		CanvasID:    analysisCanvas.ID,
		CanvasRunID: analysisRun.ID,
		WorkOrderID: order.ID,
	})
	require.NoError(t, err)

	event := support.EmitCanvasEventForNode(t, analysisCanvas.ID, "start", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, analysisCanvas.ID, backlogRefinementNodeID, event.ID, event.ID)
	require.NoError(t, db.Model(execution).Update("run_id", analysisRun.ID).Error)

	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)

	updatedSession, err := models.FindPlanningSession(db, r.Organization.ID, factoryModel.ID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.PlanningSessionStateEnded, updatedSession.State)

	updatedRun, err := models.FindUnscopedCanvasRun(db, analysisRun.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateCancelling, updatedRun.State)
	assert.Equal(t, models.CanvasRunResultPassed, updatedRun.Result)
	result, err := updatedRun.CalculateResult(db)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunResultPassed, result)

	updatedExecution, err := models.FindNodeExecutionInTransaction(db, analysisCanvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateCancelling, updatedExecution.State)
}

// Test__DispatchWorkOrder__RejectsWhenAlreadyActive covers acceptance
// criterion 5: a work order with an active line dispatch cannot be
// dispatched again.
func Test__DispatchWorkOrder__RejectsWhenAlreadyActive(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)

	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.Error(t, err)
	code, _, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, code)
}

// Test__DispatchWorkOrder__SecondDispatchAfterFirstFinishesCreatesSeparateTraversal
// covers acceptance criterion 4: dispatching a work order to the same line
// twice, after the first traversal finishes, produces two traversals shown
// separately by the API.
func Test__DispatchWorkOrder__SecondDispatchAfterFirstFinishesCreatesSeparateTraversal(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	firstResp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)
	require.Len(t, firstResp.Order.LineDispatches, 1)
	firstDispatchID := firstResp.Order.LineDispatches[0].Id

	firstDispatch, err := models.FindWorkOrderLineDispatch(db, uuid.MustParse(firstDispatchID))
	require.NoError(t, err)
	require.NoError(t, firstDispatch.Finish(db, models.CanvasRunResultFailed))

	secondResp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)

	require.Len(t, secondResp.Order.LineDispatches, 2,
		"two separate traversals of the same line, not one merged bucket")
	assert.Equal(t, firstDispatchID, secondResp.Order.LineDispatches[0].Id)
	assert.Equal(t, pb.WorkOrderLineDispatch_STATE_FINISHED, secondResp.Order.LineDispatches[0].State)
	assert.Equal(t, pb.WorkOrderLineDispatch_RESULT_FAILED, secondResp.Order.LineDispatches[0].Result)
	assert.NotEqual(t, firstDispatchID, secondResp.Order.LineDispatches[1].Id)
	assert.Equal(t, pb.WorkOrderLineDispatch_STATE_ACTIVE, secondResp.Order.LineDispatches[1].State)
}

func Test__DispatchWorkOrder__ReplaceActiveFinishesCurrentAndStartsAgain(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(database.DB(t.Context()), "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	firstResp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)
	firstID := firstResp.Order.LineDispatches[0].Id

	resp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId:     factoryModel.ID.String(),
		OrderId:       order.ID.String(),
		LineName:      line.Name,
		ReplaceActive: true,
	})
	require.NoError(t, err)
	require.Len(t, resp.Order.LineDispatches, 2)
	assert.Equal(t, firstID, resp.Order.LineDispatches[0].Id)
	assert.Equal(t, pb.WorkOrderLineDispatch_STATE_FINISHED, resp.Order.LineDispatches[0].State)
	assert.Equal(t, pb.WorkOrderLineDispatch_STATE_ACTIVE, resp.Order.LineDispatches[1].State)
	assert.Equal(t, int32(0), resp.Order.LineDispatches[1].StepExecutions[0].StepIndex)
}

func Test__DispatchWorkOrder__StartsAtRequestedStep(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	firstApp, firstEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	secondApp, secondEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-two", "start-two")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: firstApp.ID, Entrypoint: firstEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: secondApp.ID, Entrypoint: secondEntry},
	})
	require.NoError(t, err)

	resp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId:      factoryModel.ID.String(),
		OrderId:        order.ID.String(),
		LineName:       line.Name,
		StartStepIndex: 1,
	})
	require.NoError(t, err)
	require.Len(t, resp.Order.LineDispatches, 1)
	require.Len(t, resp.Order.LineDispatches[0].StepExecutions, 1)
	assert.Equal(t, int32(1), resp.Order.LineDispatches[0].StepExecutions[0].StepIndex)
	assert.Equal(t, secondApp.Name, resp.Order.LineDispatches[0].StepExecutions[0].Step)
}

func Test__DispatchWorkOrder__RerunStepKeepsEarlierExecutions(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	seedHostedModels(t, db, models.UsageProviderAnthropic, "claude-sonnet-4-6", "claude-opus-4-6")

	order, err := factoryModel.CreateWorkOrder(db, "Improve AGENTS.md", "", &r.User, nil, nil)
	require.NoError(t, err)

	firstApp, firstEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "plan", "start-plan")
	secondApp := createLineAppWithRunner(t, r, factoryModel.ID, runnerClaudeCode, "hosted", "claude-sonnet-4-6")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: firstApp.ID, Entrypoint: firstEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: secondApp.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)

	firstResp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId:     factoryModel.ID.String(),
		OrderId:       order.ID.String(),
		LineName:      line.Name,
		Model:         "claude-sonnet-4-6",
		ThinkingLevel: "low",
	})
	require.NoError(t, err)
	require.Len(t, firstResp.Order.LineDispatches, 1)
	firstID := firstResp.Order.LineDispatches[0].Id

	firstDispatch, err := models.FindWorkOrderLineDispatch(db, uuid.MustParse(firstID))
	require.NoError(t, err)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, startErr := firstDispatch.EnqueueOrStartStep(tx, order, 1)
		if startErr != nil {
			return startErr
		}
		return firstDispatch.Finish(tx, models.CanvasRunResultFailed)
	}))

	resp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId:      factoryModel.ID.String(),
		OrderId:        order.ID.String(),
		LineName:       line.Name,
		StartStepIndex: 1,
		ReplaceActive:  true,
		Model:          "claude-opus-4-6",
		ThinkingLevel:  "high",
	})
	require.NoError(t, err)

	var planCount int
	for _, dispatch := range resp.Order.LineDispatches {
		if dispatch.Id == firstID {
			for _, execution := range dispatch.StepExecutions {
				if execution.StepIndex == 0 {
					planCount++
				}
			}
		}
	}
	assert.Equal(t, 1, planCount, "rerun of Implementation must keep the Planning execution")
	require.Len(t, resp.Order.LineDispatches, 1)
	assert.Equal(t, "claude-sonnet-4-6", resp.Order.LineDispatches[0].Model)
	assert.Equal(t, "low", resp.Order.LineDispatches[0].ThinkingLevel)
	require.GreaterOrEqual(t, len(resp.Order.LineDispatches[0].StepExecutions), 2)
}

func Test__DispatchWorkOrder__KeepsSavedOwnersOfOpenTaskOutsideSubscriptionWorkspaces(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	creator := support.CreateUser(t, r, r.Organization.ID)
	owner := support.CreateUser(t, r, r.Organization.ID)
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &creator.ID, []uuid.UUID{creator.ID, owner.ID}, nil)
	require.NoError(t, err)
	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
	require.NoError(t, err)

	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)

	reloaded, err := factoryModel.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{creator.ID, owner.ID}, reloaded.AssigneeIDs())
}

func Test__DispatchWorkOrder__ReturnsOwnerNameAfterStart(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", nil, nil, nil)
	require.NoError(t, err)

	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	resp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)
	require.Len(t, resp.Order.Assignees, 1)
	assert.Equal(t, r.User.String(), resp.Order.Assignees[0].Id)
	assert.Equal(t, r.UserModel.Name, resp.Order.Assignees[0].Name)
	assert.NotEqual(t, r.User.String(), resp.Order.Assignees[0].Name)
}

func Test__DispatchWorkOrder__PersistsChosenModel(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	seedHostedModels(t, db, models.UsageProviderAnthropic, "claude-sonnet-4-6", "claude-opus-4-6")

	app := createLineAppWithRunner(t, r, factoryModel.ID, runnerClaudeCode, "hosted", "claude-sonnet-4-6")
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)

	resp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
		Model:     "claude-opus-4-6",
	})
	require.NoError(t, err)
	require.Len(t, resp.Order.LineDispatches, 1)
	assert.Equal(t, "claude-opus-4-6", resp.Order.LineDispatches[0].Model)

	active, err := order.FindActiveLineDispatch(db)
	require.NoError(t, err)
	assert.Equal(t, "claude-opus-4-6", active.Model)
}

func Test__DispatchWorkOrder__RejectsModelNotOnLine(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	seedHostedModels(t, db, models.UsageProviderAnthropic, "claude-sonnet-4-6")
	seedHostedModels(t, db, models.UsageProviderOpenAI, "gpt-5")

	app := createLineAppWithRunner(t, r, factoryModel.ID, runnerClaudeCode, "hosted", "claude-sonnet-4-6")
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)

	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
		Model:     "gpt-5",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
}

func Test__DispatchWorkOrder__FillsMissingTitleBeforeStart(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	longLine := strings.Repeat("a", 300)
	skippedLead := "\n\n  ![shot](sp-file://abc)\nnull\nUntitled task\nn/a\n"
	tests := []struct {
		name        string
		title       string
		description string
		wantTitle   string
	}{
		{
			name:        "empty title",
			title:       "",
			description: "Refunds fail on retry.\n\nMore context.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "null",
			title:       "null",
			description: "Refunds fail on retry.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "unknown",
			title:       "unknown",
			description: "Refunds fail on retry.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "Unknown",
			title:       "Unknown",
			description: "Refunds fail on retry.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "quoted null",
			title:       `"null"`,
			description: "Refunds fail on retry.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "n/a",
			title:       "n/a",
			description: "Refunds fail on retry.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "quoted null with trailing punctuation",
			title:       `"null."`,
			description: "Refunds fail on retry.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "meaningful title stays",
			title:       "Ship the retry fix",
			description: "A different first line.",
			wantTitle:   "Ship the retry fix",
		},
		{
			name:        "null pointer stays",
			title:       "null pointer",
			description: "A different first line.",
			wantTitle:   "null pointer",
		},
		{
			name:        "skips blank image and placeholder lines",
			title:       "",
			description: skippedLead + "Fix the checkout retry.",
			wantTitle:   "Fix the checkout retry.",
		},
		{
			name:        "heading line",
			title:       "",
			description: "## Refunds fail on retry.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "fully wrapped emphasis",
			title:       "null",
			description: "## **Refunds fail on retry.**",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "collapses whitespace",
			title:       "",
			description: "Refunds   fail\ton retry.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "cuts a long derived title",
			title:       "unknown",
			description: longLine + "\nMore context.",
			wantTitle:   longLine[:workOrderTitleMaxLength],
		},
		{
			name:        "untitled task is still a placeholder",
			title:       "Untitled task",
			description: "Refunds fail on retry.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "skips a list placeholder",
			title:       "null",
			description: "- n/a\nRefunds fail on retry.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "skips an attachment file link",
			title:       "unknown",
			description: "[notes.pdf](sp-file://aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee)\nRefunds fail on retry.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "attachment only becomes untitled task",
			title:       "",
			description: "[notes.pdf](sp-file://aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee)",
			wantTitle:   untitledWorkOrderTitle,
		},
		{
			name:        "no real description line",
			title:       "",
			description: skippedLead,
			wantTitle:   untitledWorkOrderTitle,
		},
		{
			name:        "title with no letter or digit",
			title:       "???",
			description: "Refunds fail on retry.",
			wantTitle:   "Refunds fail on retry.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			order, err := factoryModel.CreateWorkOrder(db, "Seed title", tc.description, &r.User, nil, nil)
			require.NoError(t, err)
			title := tc.title
			require.NoError(t, order.UpdateContent(db, &title, nil))

			resp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
				FactoryId: factoryModel.ID.String(),
				OrderId:   order.ID.String(),
				LineName:  line.Name,
			})
			require.NoError(t, err)
			assert.Equal(t, tc.wantTitle, resp.Order.Title)
			require.NotEmpty(t, resp.Order.LineDispatches)
			assert.Equal(t, pb.WorkOrderLineDispatch_STATE_ACTIVE, resp.Order.LineDispatches[0].State)

			reloaded, err := models.FindUnscopedWorkOrder(db, order.ID)
			require.NoError(t, err)
			assert.Equal(t, tc.wantTitle, reloaded.Title)
			assert.Equal(t, tc.description, reloaded.Description)
			assert.Equal(t, models.FactoryWorkOrderStateOpen, reloaded.State)
		})
	}
}

func Test__DispatchWorkOrder__ReplaceActiveLogsAdmittedRunOnItsOrder(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	logs := capturePublishedRunLogs(t)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	limit := 1
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint, MaxParallelism: &limit},
	})
	require.NoError(t, err)

	running, err := factoryModel.CreateWorkOrder(db, "Running", "", &r.User, nil, nil)
	require.NoError(t, err)
	queued, err := factoryModel.CreateWorkOrder(db, "Queued", "", &r.User, nil, nil)
	require.NoError(t, err)

	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   running.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)

	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   queued.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)

	var queuedBefore int64
	require.NoError(t, db.Model(&models.FactoryWorkOrderExecution{}).Where("work_order_id = ?", queued.ID).Count(&queuedBefore).Error)
	require.Zero(t, queuedBefore, "the second order must wait for the step slot")

	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId:     factoryModel.ID.String(),
		OrderId:       running.ID.String(),
		LineName:      line.Name,
		ReplaceActive: true,
	})
	require.NoError(t, err)

	var admitted models.FactoryWorkOrderExecution
	require.NoError(t, db.Where("work_order_id = ?", queued.ID).First(&admitted).Error)
	require.NotNil(t, admitted.RunID)

	orderID, ok := logs.orderIDForRun(*admitted.RunID)
	require.True(t, ok, "replacing a dispatch must log the admitted run")
	assert.Equal(t, queued.ID, orderID)
}

type publishedRunLogHook struct {
	mu      sync.Mutex
	entries []map[string]any
}

func (h *publishedRunLogHook) Levels() []log.Level {
	return []log.Level{log.InfoLevel, log.ErrorLevel}
}

func (h *publishedRunLogHook) Fire(entry *log.Entry) error {
	if !strings.Contains(entry.Message, "pending canvas run") {
		return nil
	}

	copied := make(map[string]any, len(entry.Data))
	for key, value := range entry.Data {
		copied[key] = value
	}

	h.mu.Lock()
	h.entries = append(h.entries, copied)
	h.mu.Unlock()
	return nil
}

func (h *publishedRunLogHook) orderIDForRun(runID uuid.UUID) (uuid.UUID, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, entry := range h.entries {
		loggedRun, ok := uuidField(entry["run_id"])
		if !ok || loggedRun != runID {
			continue
		}
		orderID, ok := uuidField(entry["order_id"])
		return orderID, ok
	}

	return uuid.Nil, false
}

func capturePublishedRunLogs(t *testing.T) *publishedRunLogHook {
	t.Helper()

	hook := &publishedRunLogHook{}
	logger := log.StandardLogger()
	previous := logger.ReplaceHooks(make(log.LevelHooks))
	for _, hooks := range previous {
		for _, existing := range hooks {
			logger.AddHook(existing)
		}
	}
	logger.AddHook(hook)
	t.Cleanup(func() {
		logger.ReplaceHooks(previous)
	})
	return hook
}

func Test__DispatchWorkOrder__RequiresASavedBranchBeforeImplement(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "implement", "start-impl")
	require.NoError(t, app.StampFactoryAppTemplate(db, entrypoint, "line-implementation", 1))
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	pinnedRepo := "acme/api"
	pinnedBranch := "develop"
	pinned, err := factoryModel.CreateWorkOrder(db, "Pinned task", "", &r.User, nil, nil)
	require.NoError(t, err)
	require.NoError(t, db.Model(pinned).Updates(map[string]any{
		"repository":     pinnedRepo,
		"default_branch": pinnedBranch,
	}).Error)
	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   pinned.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(db, "Needs a branch", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.ErrorContains(t, err, "Set the default branch in Workspace settings → Repository before you start Implement.")
	code, _, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, code)

	reloaded, err := factoryModel.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateDraft, reloaded.State)
	_, err = order.FindActiveLineDispatch(db)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	branch := "develop"
	require.NoError(t, factoryModel.UpdateOnboarding(db, models.FactoryOnboardingPatch{DefaultBranch: &branch}))
	response, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)
	require.Len(t, response.Order.LineDispatches, 1)

	openOrder, err := factoryModel.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	_, err = openOrder.SetStatusNote(db, models.FactoryWorkOrderStatusNoteParams{
		Key:      missingImplementBranchNoteKey,
		Headline: "Set the default branch",
		Body:     missingImplementBranchMessage,
	})
	require.NoError(t, err)
	active, err := openOrder.FindActiveLineDispatch(db)
	require.NoError(t, err)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return active.Finish(tx, models.CanvasRunResultCancelled)
	}))
	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)
	openOrder, err = factoryModel.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	notes, err := openOrder.StatusNotes()
	require.NoError(t, err)
	assert.Empty(t, notes)
}

func Test__DispatchWorkOrder__RequiresASavedBranchBeforeALaterImplementStep(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	refine, refineEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "refine", "start-refine")
	implement, implementEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "implement", "start-impl")
	require.NoError(t, implement.StampFactoryAppTemplate(db, implementEntry, "line-implementation", 1))
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: refine.ID, Entrypoint: refineEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: implement.ID, Entrypoint: implementEntry},
	})
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(db, "Needs a branch", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.ErrorContains(t, err, "Set the default branch in Workspace settings → Repository before you start Implement.")
	reloaded, err := factoryModel.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateDraft, reloaded.State)

	readOnly, err := factoryModel.CreateLine(db, "refine-only", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: refine.ID, Entrypoint: refineEntry},
	})
	require.NoError(t, err)
	response, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  readOnly.Name,
	})
	require.NoError(t, err)
	require.NotEmpty(t, response.Order.LineDispatches)
}

func TestDispatchRetryChecksSavedBranchForSavedSteps(t *testing.T) {
	for _, tc := range []struct {
		name           string
		savedImplement bool
	}{
		{name: "requires a branch for a removed Implement step", savedImplement: true},
		{name: "ignores an Implement step added after the saved run"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			branch := "main"
			require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{DefaultBranch: &branch}))
			readOnly, readOnlyEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "read-only", "start-read")
			implement, implementEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "implement", "start-impl")
			require.NoError(t, implement.StampFactoryAppTemplate(db, implementEntry, "line-implementation", 1))
			readOnlyStep := models.FactoryLineStep{Type: models.FactoryLineStepTypeRunApp, AppID: readOnly.ID, Entrypoint: readOnlyEntry}
			implementStep := models.FactoryLineStep{Type: models.FactoryLineStepTypeRunApp, AppID: implement.ID, Entrypoint: implementEntry}
			steps := []models.FactoryLineStep{readOnlyStep, readOnlyStep, readOnlyStep}
			if tc.savedImplement {
				steps[2] = implementStep
			}
			line, err := factory.CreateLine(db, "ship", steps)
			require.NoError(t, err)
			order, err := factory.CreateWorkOrder(db, "Retry the saved steps", "", &r.User, nil, nil)
			require.NoError(t, err)
			ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
			started, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
				FactoryId: factory.ID.String(), OrderId: order.ID.String(), LineName: line.Name,
			})
			require.NoError(t, err)
			dispatch, err := models.FindWorkOrderLineDispatch(db, uuid.MustParse(started.Order.LineDispatches[0].Id))
			require.NoError(t, err)
			require.NoError(t, dispatch.Finish(db, models.CanvasRunResultFailed))
			currentSteps := []models.FactoryLineStep{readOnlyStep, readOnlyStep, implementStep}
			if tc.savedImplement {
				currentSteps = steps[:2]
			}
			require.NoError(t, line.Update(db, nil, currentSteps, nil))
			branch = ""
			require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{DefaultBranch: &branch}))
			response, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
				FactoryId: factory.ID.String(), OrderId: order.ID.String(), LineName: line.Name,
				StartStepIndex: 2, ReplaceActive: true,
			})
			if tc.savedImplement {
				require.ErrorContains(t, err, missingImplementBranchMessage)
				assertDispatchRejected(t, db, order, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, response.Order.LineDispatches, 1)
			retried := response.Order.LineDispatches[0]
			assert.Equal(t, dispatch.ID.String(), retried.Id)
			assert.Equal(t, int32(2), retried.StepExecutions[len(retried.StepExecutions)-1].StepIndex)
		})
	}
}

func uuidField(value any) (uuid.UUID, bool) {
	switch typed := value.(type) {
	case uuid.UUID:
		return typed, typed != uuid.Nil
	case string:
		parsed, err := uuid.Parse(typed)
		if err != nil || parsed == uuid.Nil {
			return uuid.Nil, false
		}
		return parsed, true
	default:
		return uuid.Nil, false
	}
}

func TestDispatchSavedThinkingDistinguishesMissingAndDisabledCapabilities(t *testing.T) {
	for _, tc := range []struct {
		name, component, source, provider, model string
	}{
		{"OpenRouter hosted", runnerOpenRouter, "hosted", models.UsageProviderOpenRouter, "deepseek/deepseek-v4.1-flash"},
		{"OpenRouter BYOK", runnerOpenRouter, "integration", models.UsageProviderOpenRouter, "deepseek/deepseek-v4.1-flash"},
		{"Claude hosted", runnerClaudeCode, "hosted", models.UsageProviderAnthropic, "claude-opus-5-5"},
		{"Claude BYOK", runnerClaudeCode, "integration", models.UsageProviderAnthropic, "claude-opus-5-5"},
		{"SuperPlane OpenRouter", models.SuperPlaneRunnerComponent, "hosted", models.UsageProviderOpenRouter, "deepseek/deepseek-v4.1-flash"},
		{"SuperPlane Claude", models.SuperPlaneRunnerComponent, "hosted", models.UsageProviderAnthropic, "claude-opus-5-5"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			if tc.source == "hosted" {
				seedHostedModels(t, db, tc.provider, tc.model)
			}
			if tc.source == "integration" {
				require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
				_, err := models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, tc.provider, []string{tc.model})
				require.NoError(t, err)
			}
			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			model := tc.model
			if tc.component == models.SuperPlaneRunnerComponent {
				model = "hosted::" + tc.provider + "::" + model
			}
			app := createLineAppWithRunner(t, r, factory.ID, tc.component, tc.source, model)
			setLineAppRunnerThinkingLevel(t, db, app, "high")
			line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{
				{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"},
			})
			require.NoError(t, err)
			ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
			baseURL := llm.DefaultOpenRouterBaseURL
			if tc.provider == models.UsageProviderAnthropic {
				baseURL = llm.DefaultAnthropicBaseURL
			}
			for _, state := range []struct {
				name         string
				capabilities []models.LLMModelCapability
				wantErr      bool
			}{
				{name: "uncached"},
				{name: "omitted efforts", capabilities: []models.LLMModelCapability{{Model: tc.model}}},
				{name: "supported effort", capabilities: []models.LLMModelCapability{{Model: tc.model, ThinkingLevels: datatypes.NewJSONSlice([]string{"high"})}}},
				{name: "disabled effort", capabilities: []models.LLMModelCapability{{Model: tc.model, ThinkingLevels: datatypes.NewJSONSlice([]string{})}}, wantErr: true},
				{name: "omitted after disabled", capabilities: []models.LLMModelCapability{{Model: tc.model}}, wantErr: true},
			} {
				t.Run(state.name, func(t *testing.T) {
					require.NoError(t, models.ReplaceLLMModelCapabilities(db, tc.provider, baseURL, state.capabilities))
					order, err := factory.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
					require.NoError(t, err)
					request := &pb.DispatchWorkOrderRequest{
						FactoryId: factory.ID.String(), OrderId: order.ID.String(), LineName: line.Name,
					}
					response, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), request)
					if state.wantErr {
						require.ErrorContains(t, err, "saved thinking level")
						assertDispatchRejected(t, db, order, err)
						request.ThinkingLevel = "default"
						response, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), request)
					}
					require.NoError(t, err)
					require.Len(t, response.Order.LineDispatches, 1)
				})
			}
		})
	}
}

func TestDispatchRetryChecksSavedStepsAfterTheLineShrinks(t *testing.T) {
	for _, tc := range []struct {
		name          string
		removedLevel  string
		startThinking string
		wantErr       string
	}{
		{name: "retries the removed step", removedLevel: "low"},
		{name: "checks the removed step", removedLevel: "high", wantErr: "saved thinking level"},
		{name: "checks the saved run thinking level", removedLevel: "low", startThinking: "high", wantErr: "Start the line again"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			seedHostedModels(t, db, models.UsageProviderAnthropic, "claude-sonnet-5-5")
			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			var steps []models.FactoryLineStep
			for i := range 3 {
				app := createLineAppWithRunner(t, r, factory.ID, runnerClaudeCode, "hosted", "claude-sonnet-5-5")
				level := "low"
				if i == 2 {
					level = tc.removedLevel
				}
				setLineAppRunnerThinkingLevel(t, db, app, level)
				steps = append(steps, models.FactoryLineStep{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"})
			}
			line, err := factory.CreateLine(db, "ship", steps)
			require.NoError(t, err)
			order, err := factory.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
			require.NoError(t, err)
			ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
			started, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
				FactoryId: factory.ID.String(), OrderId: order.ID.String(), LineName: line.Name, ThinkingLevel: tc.startThinking,
			})
			require.NoError(t, err)
			originalDispatchID := started.Order.LineDispatches[0].Id
			dispatch, err := models.FindWorkOrderLineDispatch(db, uuid.MustParse(originalDispatchID))
			require.NoError(t, err)
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				if _, err := dispatch.EnqueueOrStartStep(tx, order, 2); err != nil {
					return err
				}
				return dispatch.Finish(tx, models.CanvasRunResultFailed)
			}))
			require.NoError(t, line.Update(db, nil, steps[:2], nil))
			// A catalog refresh removes high after the original traversal.
			require.NoError(t, models.UpsertLLMModelCapabilities(db, []models.LLMModelCapability{{
				Provider: models.UsageProviderAnthropic, BaseURL: llm.DefaultAnthropicBaseURL,
				Model: "claude-sonnet-5-5", ThinkingLevels: datatypes.NewJSONSlice([]string{"low"}), RefreshedAt: time.Now(),
			}}))

			response, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
				FactoryId: factory.ID.String(), OrderId: order.ID.String(), LineName: line.Name,
				StartStepIndex: 2, ReplaceActive: true,
			})
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Len(t, response.Order.LineDispatches, 1)
			retried := response.Order.LineDispatches[0]
			assert.Equal(t, originalDispatchID, retried.Id)
			assert.Equal(t, int32(2), retried.StepExecutions[len(retried.StepExecutions)-1].StepIndex)
		})
	}
}

func TestDispatchRejectsSavedThinkingLevelThatTheModelCatalogLacks(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	seedHostedModels(t, db, models.UsageProviderAnthropic, "claude-opus-4-5-20251101")
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	app := createLineAppWithRunner(t, r, factory.ID, runnerClaudeCode, "hosted", "claude-opus-4-5-20251101")
	setLineAppRunnerThinkingLevel(t, db, app, "max")
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())

	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factory.ID.String(), OrderId: order.ID.String(), LineName: line.Name,
	})
	require.ErrorContains(t, err, "saved thinking level")
	assertDispatchRejected(t, db, order, err)

	response, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factory.ID.String(), OrderId: order.ID.String(), LineName: line.Name, ThinkingLevel: "high",
	})
	require.NoError(t, err)
	require.Len(t, response.Order.LineDispatches, 1)
}

func TestDispatchLeavesSavedSubscriptionThinkingToExecutionWithoutConnection(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	withoutEffort := createLineAppWithRunner(t, r, factory.ID, runnerCodex, "subscription", "gpt-6-astra")
	withEffort := createLineAppWithRunner(t, r, factory.ID, runnerCodex, "subscription", "gpt-6-astra")
	setLineAppRunnerThinkingLevel(t, db, withEffort, "high")
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())

	var results []error
	for _, app := range []*models.Canvas{withoutEffort, withEffort} {
		line, err := factory.CreateLine(db, support.RandomName("ship"), []models.FactoryLineStep{
			{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"},
		})
		require.NoError(t, err)
		order, err := factory.CreateWorkOrder(db, "Ship it", "", &r.User, []uuid.UUID{r.User}, nil)
		require.NoError(t, err)
		_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
			FactoryId: factory.ID.String(), OrderId: order.ID.String(), LineName: line.Name,
		})
		results = append(results, err)
	}
	assert.Equal(t, results[0] == nil, results[1] == nil, "a saved thinking level must not change how Start handles a missing ChatGPT connection: %v", results)
}

func setLineAppRunnerThinkingLevel(t *testing.T, db *gorm.DB, app *models.Canvas, level string) {
	t.Helper()
	version, err := models.FindLiveCanvasVersionInTransaction(db, app.ID)
	require.NoError(t, err)
	for i := range version.Nodes {
		if version.Nodes[i].ID == "agent" {
			version.Nodes[i].Configuration["thinkingLevel"] = level
		}
	}
	require.NoError(t, db.Model(version).Update("nodes", version.Nodes).Error)
}
