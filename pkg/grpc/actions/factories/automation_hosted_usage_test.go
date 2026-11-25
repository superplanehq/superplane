package factories

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/automations"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/components/runner/superplane"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
	"gorm.io/datatypes"
)

func TestCanvasAutomationHostedModelUsage(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &vcsID}))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureAutomationsAsCode))
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	upsertHostedOnboardingProvider(t, db)
	const model = "anthropic/claude-sonnet-4-6"
	_, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider: models.UsageProviderOpenRouter, Enabled: true,
		APIKey: []byte("test-hosted-key"), ManagementKey: []byte("test-management-key"),
		AllowedModels: datatypes.JSONSlice[string]{model},
	})
	require.NoError(t, err)
	markup := 2_000
	require.NoError(t, billingmodels.UpsertOrganizationLLMMarkup(db, r.Organization.ID, &markup))

	ctx := onboardingContext(r)
	service := automations.Service{Backend: CanvasAutomationBackend{Deps: onboardingDeps(r)}}
	placement := automations.Placement{OrganizationID: r.Organization.ID, FactoryID: factory.ID, Stage: models.CanvasColumnKeyBacklog}
	for name, yamlModel := range map[string]string{"bare ID": model, "legacy key": "hosted::openrouter::" + model} {
		t.Run(name, func(t *testing.T) {
			source := "apiVersion: v2\nkind: Automation\nname: Inspect " + name + "\nagent: {model: " + yamlModel + "}\n" +
				"triggers: [{use: manual}]\nsteps: [{name: Inspect, prompt: Inspect the changes.}]\n"
			created, err := service.Create(ctx, placement, r.User, source)
			require.NoError(t, err)
			live, err := models.FindLiveCanvasVersionInTransaction(db, created.ID)
			require.NoError(t, err)
			node := selectCustomAutomationAgent(live.Nodes)
			require.NotNil(t, node)

			tasks := &contexts.RunnerTaskContext{}
			config := node.Configuration
			component := &superplane.RunSuperPlane{}
			require.NoError(t, component.Execute(core.ExecutionContext{
				Configuration: config,
				RunnerTasks:   tasks,
				HTTP: &contexts.HTTPContext{Responses: []*http.Response{{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"key":"test-child-key","data":{"hash":"test-child-hash"}}`)),
				}}},
				HostedLLM: &contexts.HostedLLMContext{Access: core.HostedLLMAccess{
					APIKey: "test-hosted-key", ManagementKey: "test-management-key", AllowedModels: []string{model},
				}},
				ExecutionState:      &contexts.ExecutionStateContext{KVs: map[string]string{}},
				ResolvedEnvironment: &runner.ResolvedEnvironment{},
				Requests:            &contexts.RequestContext{},
				Webhook:             &contexts.NodeWebhookContext{},
			}))
			require.Contains(t, string(tasks.Payload), model)

			event := support.EmitCanvasEventForNode(t, created.ID, customAutomationTaskTriggerID, "default", nil)
			run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, event)
			require.NoError(t, err)
			for _, pricing := range []struct {
				name                string
				result              json.RawMessage
				providerCost, total int64
			}{
				{"price book", json.RawMessage(`{"usage":{"input_tokens":1000000,"output_tokens":100000}}`), 4_500_000, 5_400_000},
				{"provider reported", json.RawMessage(`{"usage":{"input_tokens":1000000,"output_tokens":100000},"total_cost_usd":2.5}`), 2_500_000, 3_000_000},
			} {
				t.Run(pricing.name, func(t *testing.T) {
					execution := &models.CanvasNodeExecution{ID: uuid.New(), RunID: run.ID, NodeID: node.ID}
					usage := workercontexts.NewUsageContext(r.Organization.ID, execution)
					for range 2 {
						runner.RecordRunnerLLMUsage(usage, nil, superplane.FinishedEventType, config, pricing.result)
					}

					var entries []models.WorkspaceUsageEvent
					require.NoError(t, db.Where("node_execution_id = ?", execution.ID).Find(&entries).Error)
					require.Len(t, entries, 1)
					entry := entries[0]
					require.Equal(t, models.UsageFundingSourceHosted, entry.FundingSource)
					require.Equal(t, models.UsageProviderOpenRouter, entry.Provider)
					require.Equal(t, model, entry.Model)
					require.EqualValues(t, 1_000_000, entry.InputTokens)
					require.EqualValues(t, 100_000, entry.OutputTokens)
					require.EqualValues(t, 1_100_000, entry.TotalTokens)
					require.Equal(t, pricing.providerCost, entry.ProviderCostMicros)
					require.Equal(t, pricing.total, entry.CostMicros)
				})
			}
		})
	}
}
