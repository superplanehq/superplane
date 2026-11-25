package factories

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/automationspec"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"google.golang.org/protobuf/proto"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type configurationQueryCounts map[string]int

type configurationQueryCounters map[string]*atomic.Int64

type configurationQueryCountKey struct{}

func countConfigurationQueries(ctx context.Context, run func(context.Context)) configurationQueryCounts {
	counts := configurationQueryCounters{}
	for _, category := range []string{"sources", "versions", "metadata", "ownership", "integrations", "policy", "secrets"} {
		counts[category] = &atomic.Int64{}
	}
	run(context.WithValue(ctx, configurationQueryCountKey{}, counts))
	snapshot := configurationQueryCounts{}
	for category, count := range counts {
		snapshot[category] = int(count.Load())
	}
	return snapshot
}

func TestFactoryConfigurationBatchesSharedReads(t *testing.T) {
	db := database.DB(t.Context())
	const callback = "test:configuration_shared_queries"
	require.NoError(t, db.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
		counts, ok := tx.Statement.Context.Value(configurationQueryCountKey{}).(configurationQueryCounters)
		if !ok {
			return
		}
		switch tx.Statement.Table {
		case "automations_as_code":
			counts["sources"].Add(1)
		case "workflow_versions":
			counts["versions"].Add(1)
		case "factory_intakes", "factory_pr_feedback_handlers", "factory_lines":
			counts["metadata"].Add(1)
		case "factory_configuration_automations":
			counts["ownership"].Add(1)
		case "app_installations":
			counts["integrations"].Add(1)
		case "hosted_llm_providers", "installation_llm_settings", "organization_byok_model_allowlists", "organization_hosted_model_allowlists", "factory_llm_model_allowlists", "factory_codex_connections":
			counts["policy"].Add(1)
		case "secrets":
			counts["secrets"].Add(1)
		}
	}))
	t.Cleanup(func() { require.NoError(t, db.Callback().Query().Remove(callback)) })

	for _, source := range []string{"hosted", "custom provider", "subscription"} {
		t.Run(source, func(t *testing.T) {
			f := newConfigurationLifecycleFixture(t)
			_, err := models.CreateSecret("configuration-token", "local", f.r.User.String(), models.DomainTypeOrganization, f.r.Organization.ID, []byte("{}"))
			require.NoError(t, err)
			model := "sonnet"
			switch source {
			case "custom provider":
				agentID := createReadyOnboardingIntegration(t, f.r.Organization.ID, models.CustomLLMAppName)
				require.NoError(t, f.factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{AgentIntegrationID: &agentID}))
				require.NoError(t, models.EnableExperimentalFeature(f.r.Organization.ID, features.FeatureOrganizationBYOK))
				require.NoError(t, models.EnableExperimentalFeature(f.r.Organization.ID, features.FeatureOrganizationBYOKCustomProvider))
				_, err := models.UpsertOrganizationBYOKModelAllowlist(db, f.r.Organization.ID, models.UsageProviderCustom, datatypes.JSONSlice[string]{"alpha-model", "beta-model"})
				require.NoError(t, err)
				model = "alpha-model"
				integration, err := models.FindIntegrationInTransaction(db, f.r.Organization.ID, uuid.MustParse(agentID))
				require.NoError(t, err)
				live := resetTestLiveVersion(t, f.canvas)
				for i := range live.Nodes {
					node := &live.Nodes[i]
					if !isFactoryAgentHarness(node.ComponentName()) {
						continue
					}
					node.Ref.Component = &models.ComponentRef{Name: "runnerOpenRouter"}
					node.Configuration["credentials"] = map[string]any{"source": "integration", "integration": map[string]any{"name": integration.InstallationName}}
					node.Configuration["llmProvider"] = models.UsageProviderCustom
					node.Configuration["model"] = model
				}
				require.NoError(t, db.Model(live).Update("nodes", datatypes.NewJSONSlice(live.Nodes)).Error)
			case "subscription":
				enabled := true
				require.NoError(t, f.factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &enabled}))
				connection, err := models.CreateCodexConnection(db, f.r.Organization.ID, f.r.User, time.Now())
				require.NoError(t, err)
				require.NoError(t, db.Model(connection).Updates(map[string]any{
					"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("test-bundle"),
					"available_models": datatypes.JSONSlice[models.CodexModel]{{Model: "gpt-5.4"}},
				}).Error)
				model = "gpt-5.4"
			}

			export := func(ctx context.Context) *pb.FactoryConfigurationBundle {
				t.Helper()
				response, err := ExportFactoryConfiguration(ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{
					FactoryId: f.factory.ID.String(), LineId: f.line.ID.String(),
				})
				require.NoError(t, err)
				return response.Bundle
			}
			plan := func(ctx context.Context, bundle *pb.FactoryConfigurationBundle) *pb.PlanFactoryConfigurationResponse {
				t.Helper()
				response, err := PlanFactoryConfiguration(ctx, testIntakeDependencies(f.r), f.r.Organization.ID.String(), &pb.PlanFactoryConfigurationRequest{
					FactoryId: f.factory.ID.String(), LineId: f.line.ID.String(), Bundle: bundle,
				})
				require.NoError(t, err)
				require.Empty(t, response.Diagnostics)
				return response
			}
			install := func(bundle *pb.FactoryConfigurationBundle, start, count int) {
				t.Helper()
				for i := start; i < start+count; i++ {
					addConfigurationLifecycleAutomation(t, bundle, fmt.Sprintf("audit-%d", i), fmt.Sprintf("Audit %d", i))
					file := bundle.Files[len(bundle.Files)-1]
					file.Content = strings.Replace(file.Content, "triggers:", "agent: {model: "+model+"}\ntriggers:", 1)
					file.Content += "  - name: Read secret\n    bash: echo ready\n    secrets: [configuration-token]\n"
				}
				result := plan(f.ctx, bundle)
				_, err := f.apply(bundle, result.BaseStateToken)
				require.NoError(t, err)
			}
			install(export(f.ctx), 0, 1)
			provisioned, err := models.FindCanvasByName(db, f.r.Organization.ID, &f.factory.ID, "Audit 0")
			require.NoError(t, err)
			provisionedID := provisioned.ID.String()
			require.NoError(t, f.factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{ProvisionedAppID: &provisionedID}))

			measure := func() map[string]configurationQueryCounts {
				t.Helper()
				var bundle *pb.FactoryConfigurationBundle
				counts := map[string]configurationQueryCounts{}
				counts["export"] = countConfigurationQueries(f.ctx, func(ctx context.Context) { bundle = export(ctx) })
				var unchanged *pb.PlanFactoryConfigurationResponse
				counts["unchanged plan"] = countConfigurationQueries(f.ctx, func(ctx context.Context) {
					unchanged = plan(ctx, bundle)
					require.Empty(t, unchanged.Changes)
				})
				counts["unchanged apply"] = countConfigurationQueries(f.ctx, func(ctx context.Context) {
					response, err := ApplyFactoryConfiguration(ctx, testIntakeDependencies(f.r), f.r.Organization.ID.String(), &pb.ApplyFactoryConfigurationRequest{
						FactoryId: f.factory.ID.String(), LineId: f.line.ID.String(), Bundle: bundle, BaseStateToken: unchanged.BaseStateToken,
					})
					require.NoError(t, err)
					require.False(t, response.Changed)
				})
				for _, changed := range []string{"changed plan", "one changed plan"} {
					desired := proto.Clone(bundle).(*pb.FactoryConfigurationBundle)
					for _, file := range desired.Files[1:] {
						document, err := automationspec.Parse(file.Content)
						require.NoError(t, err)
						document.Steps[0].Prompt = fmt.Sprintf("Review %d task automations.", len(bundle.Files))
						file.Content, err = automationspec.Encode(document.Definition)
						require.NoError(t, err)
						if changed == "one changed plan" {
							break
						}
					}
					var planned *pb.PlanFactoryConfigurationResponse
					counts[changed] = countConfigurationQueries(f.ctx, func(ctx context.Context) {
						planned = plan(ctx, desired)
						require.NotEmpty(t, planned.Changes)
					})
					if changed == "one changed plan" {
						counts["one changed apply"] = countConfigurationQueries(f.ctx, func(ctx context.Context) {
							response, err := ApplyFactoryConfiguration(ctx, testIntakeDependencies(f.r), f.r.Organization.ID.String(), &pb.ApplyFactoryConfigurationRequest{
								FactoryId: f.factory.ID.String(), LineId: f.line.ID.String(), Bundle: desired, BaseStateToken: planned.BaseStateToken,
							})
							require.NoError(t, err)
							require.True(t, response.Changed)
						})
					}
				}
				desired := export(f.ctx)
				editConfigurationManifest(t, desired, func(manifest *factoryConfigurationManifest) {
					for i := range manifest.Stages {
						if manifest.Stages[i].Key == "backlog" {
							manifest.Stages[i].Automations = append(manifest.Stages[i].Automations, factoryConfigurationAutomation{
								ID: "planned-analysis", Name: "Planned analysis", Uses: "@superplane/task-analysis",
							})
						}
					}
				})
				counts["one built-in addition plan"] = countConfigurationQueries(f.ctx, func(ctx context.Context) {
					planned := plan(ctx, desired)
					require.Len(t, planned.Changes, 1)
				})
				return counts
			}
			single := measure()
			if source == "hosted" {
				for range 4 {
					resetTestTemplate(t, f.r, f.factory, "risk-score", models.ProviderGitHub, func(*yaml.Canvas) {})
				}
			}
			install(export(f.ctx), 1, 4)
			many := measure()
			for operation, expected := range single {
				t.Run(operation, func(t *testing.T) {
					t.Logf("one custom automation: %v; five custom automations: %v", expected, many[operation])
					for _, category := range []string{"sources", "versions", "metadata", "ownership", "integrations"} {
						require.Positive(t, expected[category], "%s reads must be observed", category)
					}
					if operation == "changed plan" {
						require.Positive(t, expected["policy"])
						require.Positive(t, expected["secrets"])
					}
					for category, count := range expected {
						assert.Equal(t, count, many[operation][category], "%s reads must not grow with unchanged automations", category)
					}
				})
			}
			if source == "hosted" {
				_, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
					Provider: models.UsageProviderAnthropic, Enabled: true, APIKey: []byte("test-hosted-key"),
					AllowedModels: datatypes.JSONSlice[string]{"sonnet", "opus"},
				})
				require.NoError(t, err)
				desired := export(f.ctx)
				for i, file := range desired.Files[1:3] {
					document, err := automationspec.Parse(file.Content)
					require.NoError(t, err)
					if i == 0 {
						document.Agent.Model = "opus"
					} else {
						document.Agent = nil
					}
					document.Steps[0].Prompt = "Check model isolation."
					file.Content, err = automationspec.Encode(document.Definition)
					require.NoError(t, err)
				}
				planned := plan(f.ctx, desired)
				result, err := f.apply(desired, planned.BaseStateToken)
				require.NoError(t, err)
				require.True(t, result.Changed)
				for i, expectedModel := range []string{"hosted::anthropic::opus", ""} {
					canvas, err := models.FindCanvasByName(db, f.r.Organization.ID, &f.factory.ID, fmt.Sprintf("Audit %d", i))
					require.NoError(t, err)
					agent := selectCustomAutomationAgent(resetTestLiveVersion(t, canvas).Nodes)
					require.NotNil(t, agent)
					assert.Equal(t, expectedModel, configString(agent.Configuration, "model"), "one document's override must not affect the next document")
				}
			}
		})
	}
}
