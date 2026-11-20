package factories

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/automations"
	"github.com/superplanehq/superplane/pkg/automationspec"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestCanvasAutomationPinsHostedModel(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &vcsID}))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureAutomationsAsCode))
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	upsertHostedOnboardingProvider(t, db)

	const model = "deepseek/deepseek-v4.1-flash-1017"
	_, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider: models.UsageProviderOpenRouter, Enabled: true,
		APIKey: []byte("test-hosted-key"), ManagementKey: []byte("test-management-key"),
		AllowedModels: datatypes.JSONSlice[string]{model},
	})
	require.NoError(t, err)

	ctx := onboardingContext(r)
	backend := CanvasAutomationBackend{Deps: onboardingDeps(r)}
	service := automations.Service{Backend: backend}
	placement := automations.Placement{
		OrganizationID: r.Organization.ID, FactoryID: factory.ID, Stage: models.CanvasColumnKeyBacklog,
	}
	const source = `apiVersion: v2
kind: Automation
name: Inspect task with a hosted model
agent:
  harness: superplane
  model: openrouter::deepseek/deepseek-v4.1-flash-1017
triggers:
  - use: manual
steps:
  - name: Inspect
    prompt: Summarize the task.
`
	created, err := service.Create(ctx, placement, r.User, source)
	require.NoError(t, err)
	live, err := models.FindLiveCanvasVersionInTransaction(db, created.ID)
	require.NoError(t, err)
	node := selectCustomAutomationAgent(live.Nodes)
	require.NotNil(t, node)
	require.Equal(t, models.SuperPlaneRunnerComponent, node.ComponentName())
	require.Equal(t, "openrouter::"+model, node.Configuration["model"])

	legacy, err := backend.FindLegacy(ctx, db, r.Organization.ID, created.ID)
	require.NoError(t, err)
	imported, err := backend.ImportLegacy(ctx, db, legacy)
	require.NoError(t, err)
	document, err := automationspec.Parse(imported)
	require.NoError(t, err)
	require.Equal(t, "openrouter::"+model, document.Agent.Model)

	inherit := false
	_, err = models.UpsertFactoryLLMModelAllowlist(db, r.Organization.ID, factory.ID,
		models.UsageProviderOpenRouter, models.UsageFundingSourceHosted, nil, &inherit)
	require.NoError(t, err)
	_, err = service.Validate(ctx, placement, source)
	var diagnostic *automationspec.Diagnostic
	require.ErrorAs(t, err, &diagnostic)
	require.Equal(t, "agent.model", diagnostic.Path)
}

func TestCanvasAutomationSchedulesRetainPendingDelivery(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	agentID := createReadyOnboardingIntegration(t, r.Organization.ID, "claude")
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSIntegrationID: &vcsID, AgentIntegrationID: &agentID,
	}))
	r.Registry.Integrations[models.ProviderGitHub] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
		Triggers: []core.Trigger{impl.NewDummyIntegrationTrigger(impl.DummyIntegrationTriggerOptions{Name: prFeedbackTriggerComponent(models.ProviderGitHub)})},
	})
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
	_, err := models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderAnthropic, datatypes.JSONSlice[string]{"claude-sonnet-4-6"})
	require.NoError(t, err)

	ctx := onboardingContext(r)
	backend := CanvasAutomationBackend{Deps: onboardingDeps(r)}
	placement := automations.Placement{OrganizationID: r.Organization.ID, FactoryID: factory.ID, Stage: models.CanvasColumnKeyVerify}
	id := uuid.New()
	actorID := uuid.MustParse(r.User.String())
	definition := automationspec.Definition{
		APIVersion: "v2", Kind: "Automation", Name: "Inspect changes", Icon: "search", TimeoutSeconds: 900,
		Agent: &automationspec.Agent{Harness: automationspec.ClaudeCode, Model: "claude-sonnet-4-6"},
		Triggers: []automationspec.Trigger{
			{Use: automationspec.Manual},
			{ID: "morning", Use: automationspec.Scheduled, Config: &automationspec.Schedule{Cron: "0 9 * * *"}},
			{ID: "evening", Use: automationspec.Scheduled, Config: &automationspec.Schedule{Cron: "0 17 * * *"}},
		},
		Steps: []automationspec.Step{{Name: "Print", Bash: "echo first"}, {Name: "Inspect", Prompt: "  Inspect the task.\n"}},
	}
	var artifact automations.Artifact
	save := func() {
		t.Helper()
		source, err := automationspec.Encode(definition)
		require.NoError(t, err)
		doc, err := automationspec.Parse(source)
		require.NoError(t, err)
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			if err := backend.Check(ctx, tx, placement, doc); err != nil {
				return err
			}
			next, _, err := backend.Apply(ctx, tx, placement, actorID, id, artifact, doc)
			if err == nil {
				artifact = next
			}
			return err
		}))
		require.Equal(t, id.String(), artifact.ID)
	}
	pending := func(cron string) models.CanvasNodeRequest {
		t.Helper()
		live, err := models.FindLiveCanvasVersionInTransaction(db, id)
		require.NoError(t, err)
		for _, node := range live.Nodes {
			if node.ComponentName() == "schedule" && node.Configuration["cronExpression"] == cron {
				request, err := models.FindPendingRequestForNode(db, id, node.ID)
				require.NoError(t, err)
				return *request
			}
		}
		t.Fatalf("no published schedule for %q", cron)
		return models.CanvasNodeRequest{}
	}

	save()
	live, err := models.FindLiveCanvasVersionInTransaction(db, id)
	require.NoError(t, err)
	require.Len(t, live.Edges, 3)
	taskRunner := customAutomationAgentForRunner(live.Nodes, customAutomationRunnerID)
	scheduleRunner := customAutomationAgentForRunner(live.Nodes, customAutomationScheduleRunnerID)
	require.NotNil(t, taskRunner)
	require.NotNil(t, scheduleRunner)
	require.Equal(t, taskRunner.Configuration["steps"], scheduleRunner.Configuration["steps"])
	steps, err := customAutomationProtoSteps(taskRunner.Configuration["steps"])
	require.NoError(t, err)
	require.Equal(t, "echo first", steps[0].GetCommand())
	require.Equal(t, "  Inspect the task.\n", steps[1].GetPrompt())

	morning := pending("0 9 * * *")
	evening := pending("0 17 * * *")
	// Leave a due delivery in the queue while the developer saves a new draft.
	due := time.Now().Add(-time.Minute)
	require.NoError(t, morning.Reschedule(db, due))
	node, err := models.FindCanvasNode(db, id, morning.NodeID)
	require.NoError(t, err)
	metadata := node.Metadata.Data()
	metadata["nextTrigger"] = due.Format(time.RFC3339)
	require.NoError(t, db.Model(node).Update("metadata", datatypes.NewJSONType(metadata)).Error)
	definition.Name = "Inspect changes after edits"
	definition.Steps[0].Bash = "echo second"
	definition.Triggers[1], definition.Triggers[2] = definition.Triggers[2], definition.Triggers[1]
	save()
	require.Equal(t, morning.ID, pending("0 9 * * *").ID)
	require.WithinDuration(t, due, pending("0 9 * * *").RunAt, time.Millisecond)
	require.Equal(t, evening.ID, pending("0 17 * * *").ID)

	definition.Triggers[1].Config.Cron = "0 18 * * *"
	definition.Triggers = definition.Triggers[:2]
	save()
	require.NotEqual(t, evening.NodeID, pending("0 18 * * *").NodeID)
	for _, retired := range []models.CanvasNodeRequest{morning, evening} {
		node, err := models.FindUnscopedCanvasNode(db, id, retired.NodeID)
		require.NoError(t, err)
		require.True(t, node.DeletedAt.Valid)
	}
}

func TestCanvasAutomationLegacyImportProtectsGraphBehavior(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	agentID := createReadyOnboardingIntegration(t, r.Organization.ID, "claude")
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSIntegrationID: &vcsID, AgentIntegrationID: &agentID,
	}))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureAutomationsAsCode))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	_, err := models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderAnthropic, datatypes.JSONSlice[string]{"claude-sonnet-4-6"})
	require.NoError(t, err)
	ctx, deps := onboardingContext(r), onboardingDeps(r)
	runsOnTask := false
	created, err := CreateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.CreateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), ColumnKey: models.CanvasColumnKeyBacklog, Name: "Inspect task",
		Settings: &pb.FactoryCustomAutomation_Settings{
			StartMode:  pb.FactoryCustomAutomation_START_MODE_MANUAL,
			RunsOnTask: &runsOnTask,
			Agent:      &pb.FactoryAppTemplateAgent{Component: "runnerClaudeCode", Model: "claude-sonnet-4-6"},
			Steps: []*pb.FactoryCustomAutomation_Step{
				{Name: "Inspect", Type: pb.FactoryCustomAutomation_Step_TYPE_PROMPT, Prompt: "  Inspect the task.\n"},
			},
		},
	})
	require.NoError(t, err)
	id := uuid.MustParse(created.GetCustomAutomation().GetAutomation().GetId())
	backend := CanvasAutomationBackend{Deps: deps}
	legacy, err := backend.FindLegacy(ctx, db, r.Organization.ID, id)
	require.NoError(t, err)
	source, err := backend.ImportLegacy(ctx, db, legacy)
	require.NoError(t, err)
	doc, err := automationspec.Parse(source)
	require.NoError(t, err)
	require.NotNil(t, doc.RunsOnTask)
	require.False(t, *doc.RunsOnTask)
	require.Equal(t, "  Inspect the task.\n", doc.Steps[0].Prompt)
	version, err := models.FindLiveCanvasVersionInTransaction(db, id)
	require.NoError(t, err)
	require.Equal(t, version.ID.String(), legacy.Artifact.Revision)
	customAutomationAgentForRunner(version.Nodes, customAutomationRunnerID).Configuration["skipSetupCommands"] = true
	require.NoError(t, db.Save(version).Error)
	_, err = backend.ImportLegacy(ctx, db, legacy)
	require.ErrorContains(t, err, "cannot be represented in YAML")

	delete(customAutomationAgentForRunner(version.Nodes, customAutomationRunnerID).Configuration, "skipSetupCommands")
	require.NoError(t, db.Save(version).Error)
	service := automations.Service{Backend: backend}
	adopted, err := service.Update(ctx, r.Organization.ID, id, r.User, source)
	require.NoError(t, err)
	require.Equal(t, id, adopted.ID)
	require.Equal(t, int64(1), adopted.Revision)
	started, err := RunFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.RunFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: id.String(),
	})
	require.NoError(t, err)
	require.NotEmpty(t, started.GetRunId())
	require.NoError(t, models.DisableExperimentalFeature(r.Organization.ID, features.FeatureAutomationsAsCode))
	settings := created.GetCustomAutomation().GetSettings()
	settings.Steps[0].Prompt = "Inspect the new task.\n"
	_, err = UpdateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.UpdateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: id.String(), Name: "Inspect latest task", Settings: settings,
	})
	require.NoError(t, err)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureAutomationsAsCode))
	reopened, err := service.Read(ctx, r.Organization.ID, id)
	require.NoError(t, err)
	doc, err = automationspec.Parse(reopened.YAML)
	require.NoError(t, err)
	require.Equal(t, "Inspect latest task", reopened.Name)
	require.Equal(t, "Inspect the new task.\n", doc.Steps[0].Prompt)
	require.Equal(t, int64(2), reopened.Revision)
}

func TestCanvasAutomationScheduleUsesFactorySubscriptionOwner(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	subscription, owner := true, r.User.String()
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSIntegrationID: &vcsID, CodexSubscription: &subscription, CodexDefaultOwnerID: &owner,
	}))
	connection, err := models.CreateCodexConnection(db, r.Organization.ID, r.User, time.Now())
	require.NoError(t, err)
	require.NoError(t, db.Model(connection).Updates(map[string]any{
		"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("test-bundle"),
	}).Error)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureAutomationsAsCode))
	ctx, deps := onboardingContext(r), onboardingDeps(r)
	service := automations.Service{Backend: CanvasAutomationBackend{Deps: deps}}
	source := `apiVersion: v2
kind: Automation
name: Find tasks
agent:
  harness: codex
  credentials:
    source: subscription
triggers:
  - use: schedule
    config:
      cron: "0 9 * * *"
steps:
  - name: Inspect
    prompt: Look for tasks.
`
	created, err := service.Create(ctx, automations.Placement{
		OrganizationID: r.Organization.ID, FactoryID: factory.ID, Stage: models.CanvasColumnKeyBacklog,
	}, r.User, source)
	require.NoError(t, err)
	result, err := RunFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.RunFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: created.ID.String(),
	})
	require.NoError(t, err)
	ownerID, err := models.RequireSubscriptionRunOwner(db, r.Organization.ID, factory.ID, uuid.MustParse(result.GetRunId()))
	require.NoError(t, err)
	require.Equal(t, r.User, ownerID)

	// Simulate a raw version that predates YAML adoption with matching owner metadata.
	published, err := models.FindLiveCanvasVersionInTransaction(db, created.ID)
	require.NoError(t, err)
	raw := *published
	raw.ID = uuid.New()
	raw.AutomationManaged = false
	require.NoError(t, db.Create(&raw).Error)
	rawRun, err := models.CreateCanvasRunInTransaction(db, created.ID, published.Nodes[0].ID, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	require.NoError(t, db.Model(rawRun).Update("version_id", raw.ID).Error)
	_, err = models.RequireSubscriptionRunOwner(db, r.Organization.ID, factory.ID, rawRun.ID)
	require.ErrorIs(t, err, models.ErrSubscriptionTaskAssociation)

	// A later YAML save must keep the original validated run authorized.
	_, err = service.Update(ctx, r.Organization.ID, created.ID, r.User, source)
	require.NoError(t, err)
	ownerID, err = models.RequireSubscriptionRunOwner(db, r.Organization.ID, factory.ID, uuid.MustParse(result.GetRunId()))
	require.NoError(t, err)
	require.Equal(t, r.User, ownerID)

	// Form saves use the generated publisher and must retain validated provenance.
	described, err := DescribeFactoryCustomAutomation(ctx, r.Organization.ID.String(), &pb.DescribeFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: created.ID.String(),
	})
	require.NoError(t, err)
	settings := described.GetCustomAutomation().GetSettings()
	settings.Steps[0].Prompt = "Look for new tasks."
	_, err = UpdateFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.UpdateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: created.ID.String(), Name: created.Name, Settings: settings,
	})
	require.NoError(t, err)
	formRun, err := RunFactoryCustomAutomation(ctx, deps, r.Organization.ID.String(), &pb.RunFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: created.ID.String(),
	})
	require.NoError(t, err)
	ownerID, err = models.RequireSubscriptionRunOwner(db, r.Organization.ID, factory.ID, uuid.MustParse(formRun.GetRunId()))
	require.NoError(t, err)
	require.Equal(t, r.User, ownerID)

}
