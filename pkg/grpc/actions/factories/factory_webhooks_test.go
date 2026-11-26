package factories

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	factorycomponent "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
)

func TestFactoryWebhookLifecycle(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	ctx, deps, orgID := onboardingContext(r), onboardingDeps(r), r.Organization.ID.String()

	_, err := ListFactoryWebhooks(ctx, deps, orgID, &pb.ListFactoryWebhooksRequest{FactoryId: factory.ID.String()})
	requireStatusCode(t, err, codes.FailedPrecondition)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryAsCode))

	created, err := CreateFactoryWebhook(ctx, deps, orgID, &pb.CreateFactoryWebhookRequest{
		FactoryId: factory.ID.String(), Name: " Jira ", Authentication: pb.FactoryWebhook_AUTHENTICATION_SIGNATURE,
	})
	require.NoError(t, err)
	webhook := created.GetWebhook()
	assert.Equal(t, "Jira", webhook.GetName())
	assert.Equal(t, "X-Signature-256", webhook.GetHeaderName())
	assert.Equal(t, "http://localhost:8000/api/v1/webhooks/"+webhook.GetId(), webhook.GetUrl())
	assert.NotEmpty(t, created.GetSecret())

	_, err = CreateFactoryWebhook(ctx, deps, orgID, &pb.CreateFactoryWebhookRequest{
		FactoryId: factory.ID.String(), Name: "Jira", Authentication: pb.FactoryWebhook_AUTHENTICATION_BEARER,
	})
	requireStatusCode(t, err, codes.AlreadyExists)
	for _, invalid := range []*pb.CreateFactoryWebhookRequest{
		{FactoryId: factory.ID.String(), Name: " ", Authentication: pb.FactoryWebhook_AUTHENTICATION_BEARER},
		{FactoryId: factory.ID.String(), Name: "No auth"},
		{FactoryId: factory.ID.String(), Name: "Bad header", Authentication: pb.FactoryWebhook_AUTHENTICATION_HEADER_TOKEN, HeaderName: "X Token"},
	} {
		_, err = CreateFactoryWebhook(ctx, deps, orgID, invalid)
		requireStatusCode(t, err, codes.InvalidArgument)
	}

	updated, err := UpdateFactoryWebhook(ctx, deps, orgID, &pb.UpdateFactoryWebhookRequest{
		FactoryId: factory.ID.String(), WebhookId: webhook.GetId(), Name: "Jira issues",
		Authentication: pb.FactoryWebhook_AUTHENTICATION_BEARER, HeaderName: "X-Ignored",
	})
	require.NoError(t, err)
	assert.Equal(t, "Jira issues", updated.GetWebhook().GetName())
	assert.Empty(t, updated.GetWebhook().GetHeaderName())

	regenerated, err := RegenerateFactoryWebhookSecret(ctx, deps, orgID, &pb.RegenerateFactoryWebhookSecretRequest{
		FactoryId: factory.ID.String(), WebhookId: webhook.GetId(),
	})
	require.NoError(t, err)
	assert.NotEqual(t, created.GetSecret(), regenerated.GetSecret())

	other, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	_, err = DeleteFactoryWebhook(ctx, orgID, &pb.DeleteFactoryWebhookRequest{FactoryId: other.ID.String(), WebhookId: webhook.GetId()})
	requireStatusCode(t, err, codes.NotFound)

	listed, err := ListFactoryWebhooks(ctx, deps, orgID, &pb.ListFactoryWebhooksRequest{FactoryId: factory.ID.String()})
	require.NoError(t, err)
	require.Len(t, listed.GetWebhooks(), 1)
	assert.Equal(t, int32(0), listed.GetWebhooks()[0].GetAutomationCount())

	_, err = DeleteFactoryWebhook(ctx, orgID, &pb.DeleteFactoryWebhookRequest{FactoryId: factory.ID.String(), WebhookId: webhook.GetId()})
	require.NoError(t, err)
	listed, err = ListFactoryWebhooks(ctx, deps, orgID, &pb.ListFactoryWebhooksRequest{FactoryId: factory.ID.String()})
	require.NoError(t, err)
	assert.Empty(t, listed.GetWebhooks())
}

func TestBacklogWebhookAutomationCreatesTasksFromThePayload(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryAsCode))
	integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &integrationID}))
	ctx, deps, orgID := onboardingContext(r), onboardingDeps(r), r.Organization.ID.String()
	created, err := CreateFactoryWebhook(ctx, deps, orgID, &pb.CreateFactoryWebhookRequest{
		FactoryId: factory.ID.String(), Name: "Jira", Authentication: pb.FactoryWebhook_AUTHENTICATION_BEARER,
	})
	require.NoError(t, err)
	webhookID := created.GetWebhook().GetId()

	runsOnTask := false
	prompt := "Create a task for the Jira issue."
	settings := &pb.FactoryCustomAutomation_Settings{
		StartMode:  pb.FactoryCustomAutomation_START_MODE_WEBHOOK,
		RunsOnTask: &runsOnTask,
		Triggers: []*pb.FactoryCustomAutomation_StartTrigger{
			{StartMode: pb.FactoryCustomAutomation_START_MODE_WEBHOOK, WebhookId: webhookID},
		},
		Agent:                   &pb.FactoryAppTemplateAgent{Component: "runnerClaudeCode", Model: "claude-sonnet-4-6"},
		ExecutionTimeoutSeconds: 3600,
		Steps: []*pb.FactoryCustomAutomation_Step{
			{Name: "Triage", Type: pb.FactoryCustomAutomation_Step_TYPE_PROMPT, Prompt: "Read the Jira issue."},
			{Name: "Create", Type: pb.FactoryCustomAutomation_Step_TYPE_PROMPT, Prompt: prompt},
		},
	}
	automation, err := CreateFactoryCustomAutomation(ctx, deps, orgID, &pb.CreateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), Name: "Jira intake", ColumnKey: "backlog", Settings: settings,
	})
	require.NoError(t, err)
	assert.True(t, proto.Equal(settings, automation.GetCustomAutomation().GetSettings()))

	canvasID := uuid.MustParse(automation.GetCustomAutomation().GetAutomation().GetId())
	trigger, err := models.FindCanvasNode(db, canvasID, "webhook-"+webhookID)
	require.NoError(t, err)
	assert.Equal(t, factorycomponent.OnWebhookTriggerName, trigger.Ref.Data().Trigger.Name)
	require.NotNil(t, trigger.WebhookID)
	assert.Equal(t, webhookID, trigger.WebhookID.String())

	runnerNode, err := models.FindCanvasNode(db, canvasID, customAutomationRunnerID)
	require.NoError(t, err)
	template := runnerNode.Metadata.Data()[models.FactoryAppTemplateMetadataKey].(map[string]any)
	assert.Equal(t, "scheduled", template["startMode"], "the runner token needs the create_task scope")

	resolve := func(payload map[string]any, step int) string {
		configuration, err := contexts.NewNodeConfigurationBuilder(db, canvasID).WithRootPayload(payload).Build(runnerNode.Configuration.Data())
		require.NoError(t, err)
		return configuration["steps"].([]any)[step].(map[string]any)["prompt"].(string)
	}
	webhookPayload := map[string]any{"type": "webhook", "data": map[string]any{
		"body": map[string]any{"issue": map[string]any{"key": "ENG-42", "summary": "{{ secrets }}"}},
	}}
	assert.Contains(t, resolve(webhookPayload, 0), `"key": "ENG-42"`, "every prompt gets the request body")
	fromWebhook := resolve(webhookPayload, 1)
	assert.Contains(t, fromWebhook, prompt)
	assert.Contains(t, fromWebhook, "create_task")
	assert.Contains(t, fromWebhook, `"key": "ENG-42"`)
	assert.Contains(t, fromWebhook, `"summary": "{{ secrets }}"`, "payload text must not run as an expression")
	assert.NotContains(t, resolve(map[string]any{"type": "manual.run", "data": map[string]any{}}, 1), "webhook started this run")

	canvas, err := models.FindCanvasWithoutOrgScope(canvasID)
	require.NoError(t, err)
	_, version, err := canvasLegacyAutomation(db, canvas)
	require.NoError(t, err)
	source, err := importAutomationCanvas(db, factory, canvas, version)
	require.NoError(t, err, "the YAML source keeps each prompt as written")
	assert.Contains(t, source, "webhook_id: "+webhookID)
	assert.NotContains(t, source, "webhook started this run")

	listed, err := ListFactoryWebhooks(ctx, deps, orgID, &pb.ListFactoryWebhooksRequest{FactoryId: factory.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, int32(1), listed.GetWebhooks()[0].GetAutomationCount())
	_, err = DeleteFactoryWebhook(ctx, orgID, &pb.DeleteFactoryWebhookRequest{FactoryId: factory.ID.String(), WebhookId: webhookID})
	requireStatusCode(t, err, codes.FailedPrecondition)

	_, err = UpdateFactoryCustomAutomation(ctx, deps, orgID, &pb.UpdateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: canvasID.String(), Name: "Jira intake", Settings: settings,
	})
	require.NoError(t, err)
	runnerNode, err = models.FindCanvasNode(db, canvasID, customAutomationRunnerID)
	require.NoError(t, err)
	for _, step := range runnerNode.Configuration.Data()["steps"].([]any) {
		assert.Equal(t, 1, strings.Count(step.(map[string]any)["prompt"].(string), "webhook started this run"))
	}

	for name, change := range map[string]func(*pb.FactoryCustomAutomation_Settings){
		"unknown webhook": func(s *pb.FactoryCustomAutomation_Settings) { s.Triggers[0].WebhookId = uuid.NewString() },
		"duplicate webhook": func(s *pb.FactoryCustomAutomation_Settings) {
			s.Triggers = append(s.Triggers, proto.Clone(s.Triggers[0]).(*pb.FactoryCustomAutomation_StartTrigger))
		},
		"runs on task": func(s *pb.FactoryCustomAutomation_Settings) {
			value := true
			s.RunsOnTask = &value
		},
	} {
		t.Run(name, func(t *testing.T) {
			broken := proto.Clone(settings).(*pb.FactoryCustomAutomation_Settings)
			change(broken)
			_, err := CreateFactoryCustomAutomation(ctx, deps, orgID, &pb.CreateFactoryCustomAutomationRequest{
				FactoryId: factory.ID.String(), Name: "Broken " + name, ColumnKey: "backlog", Settings: broken,
			})
			requireStatusCode(t, err, codes.InvalidArgument)
		})
	}
	_, err = CreateFactoryCustomAutomation(ctx, deps, orgID, &pb.CreateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), Name: "Implement webhook", ColumnKey: "implement", Settings: settings,
	})
	requireStatusCode(t, err, codes.InvalidArgument)
}

func requireStatusCode(t *testing.T, err error, expected codes.Code) {
	t.Helper()
	code, _, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok, "expected a status error, got %v", err)
	require.Equal(t, expected, code)
}
