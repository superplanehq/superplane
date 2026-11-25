package factories

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	goyaml "gopkg.in/yaml.v3"
	"gorm.io/datatypes"
)

func TestFactoryConfigurationRejectsInvalidBuiltInModelWithoutWrites(t *testing.T) {
	f := newConfigurationFixture(t)
	bundle := f.export(t)
	var manifest factoryConfigurationManifest
	require.NoError(t, goyaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	found := false
	for _, stage := range manifest.Stages {
		for _, automation := range stage.Automations {
			if automation.Uses != "@superplane/implement" {
				continue
			}
			found = true
			agents := automation.With["agents"].(map[string]any)
			agent := agents[implementationAgentNodeID].(map[string]any)
			agent["model"] = "unknown-model"
		}
	}
	require.True(t, found)
	content, err := goyaml.Marshal(manifest)
	require.NoError(t, err)
	bundle.Files[0].Content = string(content)
	before := configurationResourceCounts(t)
	plan := f.plan(t, bundle)
	require.NotEmpty(t, plan.Diagnostics)
	require.Contains(t, plan.Diagnostics[0].Field, "agents."+implementationAgentNodeID)
	require.Contains(t, plan.Diagnostics[0].Message, "model")
	require.Empty(t, plan.BaseStateToken)
	_, err = f.apply(bundle, "invalid")
	require.Error(t, err)
	require.Equal(t, before, configurationResourceCounts(t))
}

func TestFactoryConfigurationBuiltInModelPolicy(t *testing.T) {
	for _, provider := range []string{models.UsageProviderOpenRouter, models.UsageProviderCustom, models.UsageProviderAnthropic} {
		t.Run(provider, func(t *testing.T) {
			r, factory := resetTestFactory(t)
			db := database.DB(t.Context())
			require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryAsCode))
			support.CompleteFactoryOnboarding(t, factory)
			funding := models.UsageFundingSourceBYOK
			component := "runnerOpenRouter"
			modelKey := func(model string) string { return model }
			var credentials map[string]any
			agentIntegrationID := ""
			vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, models.ProviderGitHub)
			if provider == models.UsageProviderAnthropic {
				funding = models.UsageFundingSourceHosted
				component = models.SuperPlaneRunnerComponent
				modelKey = func(model string) string {
					if model == "" {
						return ""
					}
					return "hosted::anthropic::" + model
				}
				upsertHostedOnboardingProvider(t, db)
				_, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
					Provider: provider, Enabled: true, APIKey: []byte("test-hosted-key"),
					AllowedModels: datatypes.JSONSlice[string]{"sonnet", "allowed-model", "other-model"},
				})
				require.NoError(t, err)
			} else {
				for _, feature := range []string{features.FeatureOrganizationBYOK, features.FeatureOrganizationBYOKCustomProvider} {
					require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, feature))
				}
				app := "openrouter"
				if provider == models.UsageProviderCustom {
					app = models.CustomLLMAppName
				}
				agentIntegrationID = createReadyOnboardingIntegration(t, r.Organization.ID, app)
				credentials = map[string]any{"source": "integration", "integration": map[string]any{"name": integrationName(t, r.Organization.ID, agentIntegrationID)}}
				_, err := models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, provider, datatypes.JSONSlice[string]{"allowed-model", "other-model"})
				require.NoError(t, err)
			}
			require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &vcsID, AgentIntegrationID: &agentIntegrationID}))
			canvas := resetTestTemplate(t, r, factory, "line-implementation", models.ProviderGitHub, func(doc *yaml.Canvas) {
				for i := range doc.Spec.Nodes {
					agent := &doc.Spec.Nodes[i]
					if !isFactoryAgentHarness(agent.Component) {
						continue
					}
					agent.Component = component
					delete(agent.Configuration, "credentials")
					if credentials != nil {
						agent.Configuration["credentials"] = credentials
					}
					agent.Configuration["llmProvider"] = provider
					agent.Configuration["model"] = modelKey("allowed-model")
				}
			})
			line, err := factory.CreateLine(db, "Delivery", []models.FactoryLineStep{{Type: models.FactoryLineStepTypeRunApp, AppID: canvas.ID, Entrypoint: "onrun-implement"}})
			require.NoError(t, err)
			f := configurationFixture{r: r, factory: factory, canvas: canvas, line: line, ctx: onboardingContext(r)}
			configureConfigurationCatalogIntegration(t, f, vcsID, models.ProviderGitHub)
			exported, err := ExportFactoryConfiguration(f.ctx, r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{FactoryId: factory.ID.String(), LineId: line.ID.String()})
			require.NoError(t, err)
			bundle := exported.Bundle
			var manifest factoryConfigurationManifest
			require.NoError(t, goyaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
			var settings map[string]any
			for i := range manifest.Stages {
				for j := range manifest.Stages[i].Automations {
					automation := &manifest.Stages[i].Automations[j]
					if automation.Uses == "@superplane/implement" {
						settings = map[string]any{}
						automation.With = map[string]any{"agents": map[string]any{implementationAgentNodeID: settings}}
					}
				}
			}
			require.NotNil(t, settings)
			modelsToCheck := []string{"unavailable-model", "other-model"}
			if funding == models.UsageFundingSourceHosted {
				modelsToCheck = append(modelsToCheck, "")
			}
			for _, model := range modelsToCheck {
				before := configurationResourceCounts(t)
				settings["model"] = modelKey(model)
				content, err := goyaml.Marshal(manifest)
				require.NoError(t, err)
				bundle.Files[0].Content = string(content)
				plan := f.plan(t, bundle)
				if model == "unavailable-model" {
					require.NotEmpty(t, plan.Diagnostics)
					require.Contains(t, plan.Diagnostics[0].Field, "agents."+implementationAgentNodeID+".model")
					require.Empty(t, plan.BaseStateToken)
					_, err = f.apply(bundle, "invalid")
					require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
					require.Equal(t, before, configurationResourceCounts(t))
					continue
				}
				require.Empty(t, plan.Diagnostics)
				require.NotEmpty(t, plan.BaseStateToken)
				// Apply must recheck workspace policy after a successful plan.
				permitted := datatypes.JSONSlice[string]{"allowed-model"}
				if model == "" {
					permitted = nil
				}
				inherit := false
				_, err = models.UpsertFactoryLLMModelAllowlist(db, r.Organization.ID, factory.ID, provider, funding, permitted, &inherit)
				require.NoError(t, err)
				_, err = f.apply(bundle, plan.BaseStateToken)
				require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
				require.Equal(t, before, configurationResourceCounts(t))
				_, err = models.UpsertFactoryLLMModelAllowlist(db, r.Organization.ID, factory.ID, provider, funding, datatypes.JSONSlice[string]{"allowed-model", "other-model"})
				require.NoError(t, err)
				applied, err := f.apply(bundle, plan.BaseStateToken)
				require.NoError(t, err)
				require.True(t, applied.Changed)
				require.Equal(t, modelKey(model), findIntakeNode(resetTestLiveVersion(t, f.canvas).Nodes, implementationAgentNodeID).Configuration["model"])
			}
		})
	}
}

func TestFactoryConfigurationBuiltInSettingsSurviveReset(t *testing.T) {
	f := newConfigurationFixture(t)
	canvas := resetTestTemplate(t, f.r, f.factory, "risk-score", models.ProviderGitHub, func(doc *yaml.Canvas) {
		agent := findYAMLNode(t, doc, "assess-risk")
		agent.Configuration["additionalInstructions"] = "Keep the previous user instructions."
		agent.Configuration["skipSetupCommands"] = true
	})
	before := resetTestLiveVersion(t, canvas)
	exported, err := ExportFactoryConfiguration(f.ctx, f.r.Organization.ID.String(), &pb.ExportFactoryConfigurationRequest{FactoryId: f.factory.Key, LineId: f.line.ID.String()})
	require.NoError(t, err)
	var manifest map[string]any
	require.NoError(t, goyaml.Unmarshal([]byte(exported.Bundle.Files[0].Content), &manifest))
	found := false
	for _, rawStage := range manifest["stages"].([]any) {
		stage := rawStage.(map[string]any)
		automations, _ := stage["automations"].([]any)
		for _, rawAutomation := range automations {
			automation := rawAutomation.(map[string]any)
			if automation["uses"] != "@superplane/merge-confidence" {
				continue
			}
			found = true
			automation["with"] = map[string]any{"agents": map[string]any{"assess-risk": map[string]any{
				"additional_instructions": "", "model": "", "thinking_level": "low", "execution_timeout_seconds": 0,
				"skip_setup_commands": false, "skip_repository_checkout": true,
			}}}
		}
	}
	require.True(t, found, "Verify must export its built-in automation")
	encoded, err := goyaml.Marshal(manifest)
	require.NoError(t, err)
	exported.Bundle.Files[0].Content = string(encoded)
	plan := f.plan(t, exported.Bundle)
	require.Empty(t, plan.Diagnostics)
	require.NotEmpty(t, plan.Changes)
	applied, err := f.apply(exported.Bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, applied.Changed)
	current := resetTestLiveVersion(t, canvas)
	agent := findIntakeNode(current.Nodes, "assess-risk")
	require.NotNil(t, agent)
	for field, expected := range map[string]any{
		"additionalInstructions": "", "model": "", "thinkingLevel": "low", "executionTimeoutSeconds": float64(0),
		"skipSetupCommands": false, "skipRepositoryCheckout": true,
	} {
		require.Equal(t, expected, agent.Configuration[field], field)
		findIntakeNode(before.Nodes, "assess-risk").Configuration[field] = expected
	}
	// Configuration application keeps the installed template and managed prompts.
	require.ElementsMatch(t, before.Nodes, current.Nodes)
	require.ElementsMatch(t, before.Edges, current.Edges)
	require.NoError(t, resetTestOrganization(t, f.r))
	agent = findIntakeNode(resetTestLiveVersion(t, canvas).Nodes, "assess-risk")
	require.NotNil(t, agent)
	require.Equal(t, "", agent.Configuration["additionalInstructions"])
	require.Equal(t, "", agent.Configuration["model"])
	require.Equal(t, "low", agent.Configuration["thinkingLevel"])
	require.EqualValues(t, 0, agent.Configuration["executionTimeoutSeconds"])
	require.Equal(t, false, agent.Configuration["skipSetupCommands"])
	require.Equal(t, true, agent.Configuration["skipRepositoryCheckout"])
}

func TestFactoryConfigurationOmitsDefaultsAndRestoresOmittedOverrides(t *testing.T) {
	f := newConfigurationFixture(t)
	parallelism := models.DefaultFactoryLineStepMaxParallelism
	f.line.Steps[0].MaxParallelism = &parallelism
	require.NoError(t, f.line.Update(database.DB(t.Context()), nil, f.line.Steps, nil))
	resetTestTemplate(t, f.r, f.factory, "risk-score", models.ProviderGitHub, func(*yaml.Canvas) {})
	bundle := f.export(t)
	initialPlan := f.plan(t, bundle)
	require.Empty(t, initialPlan.Diagnostics)
	require.Empty(t, initialPlan.Changes)
	require.NotContains(t, bundle.Files[0].Content, "max_parallelism:")
	require.NotContains(t, bundle.Files[0].Content, "execution_timeout_seconds:")
	require.NotContains(t, bundle.Files[0].Content, "skip_setup_commands:")

	for _, field := range []string{"additional_instructions", "agents"} {
		var manifest factoryConfigurationManifest
		require.NoError(t, goyaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
		for i := range manifest.Stages {
			for j := range manifest.Stages[i].Automations {
				automation := &manifest.Stages[i].Automations[j]
				if automation.Uses == "@superplane/merge-confidence" {
					require.Empty(t, automation.With)
				}
				if automation.Uses == "@superplane/implement" {
					if field == "additional_instructions" {
						agents := automation.With["agents"].(map[string]any)
						agent := agents[implementationAgentNodeID].(map[string]any)
						require.Equal(t, "Original instructions", agent[field])
						delete(agent, field)
					} else {
						delete(automation.With, field)
					}
				}
			}
		}
		content, err := goyaml.Marshal(manifest)
		require.NoError(t, err)
		bundle.Files[0].Content = string(content)
		plan := f.plan(t, bundle)
		require.Empty(t, plan.Diagnostics)
		require.NotEmpty(t, plan.Changes, "Removing %s must restore its defaults", field)
		result, err := f.apply(bundle, plan.BaseStateToken)
		require.NoError(t, err)
		require.True(t, result.Changed)
		bundle = f.export(t)
	}
	agent := findIntakeNode(resetTestLiveVersion(t, f.canvas).Nodes, implementationAgentNodeID)
	require.Empty(t, agent.Configuration["additionalInstructions"])
	require.Empty(t, agent.Configuration["thinkingLevel"])
	require.EqualValues(t, 3600, agent.Configuration["executionTimeoutSeconds"])
	bundle = f.export(t)
	require.NotContains(t, bundle.Files[0].Content, "with:")
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
	result, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.False(t, result.Changed)
}
