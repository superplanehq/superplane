package factories

import (
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	factorycomponent "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
)

func TestWebhookIntakeFieldPathsLiveInTheCreateNode(t *testing.T) {
	settings := defaultWebhookIntakeSettings()
	settings.WebhookTitlePath = "issue.summary"
	settings.WebhookDescriptionPath = ""
	settings.WebhookLinkPath = "links.self-url"
	canvas, err := buildIntakeCanvas(intakeCanvasRequest{
		Source:   models.FactoryIntakeSourceWebhook,
		Settings: settings,
		Binding:  &intakeBinding{Configuration: map[string]any{"webhook": "webhook-1"}},
	})
	require.NoError(t, err)

	trigger := findSpecNode(t, canvas, intakeTriggerNodeID)
	assert.Equal(t, factorycomponent.OnWebhookTriggerName, trigger.Component)
	assert.Nil(t, findSpecNodeOrNil(canvas, intakeFilterNodeID))

	spec := models.LiveCanvasSpec{Nodes: canvas.Nodes(), Edges: canvas.Edges()}
	graph := resolveIntakeGraph(models.FactoryIntakeSourceWebhook, spec)
	assert.True(t, graph.Healthy(spec.Edges))
	assert.Equal(t, "webhook-1", graph.TriggerResourceID(spec))

	read := serializeIntakeSettings(models.FactoryIntakeSourceWebhook, intakeSettingsFromGraph(models.FactoryIntakeSourceWebhook, graph, spec))
	assert.Equal(t, "issue.summary", read.GetWebhookTitlePath())
	assert.Empty(t, read.GetWebhookDescriptionPath())
	assert.Equal(t, "links.self-url", read.GetWebhookLinkPath())

	nodes, _, err := applyIntakeSettingsToGraph(models.FactoryIntakeSourceWebhook, graph, spec,
		&pb.FactoryIntake_Settings{WebhookDescriptionPath: proto.String(" issue.body ")}, slices.Clone(spec.Nodes), spec.Edges)
	require.NoError(t, err)
	updated := models.LiveCanvasSpec{Nodes: nodes, Edges: spec.Edges}
	read = serializeIntakeSettings(models.FactoryIntakeSourceWebhook, intakeSettingsFromGraph(models.FactoryIntakeSourceWebhook, graph, updated))
	assert.Equal(t, "issue.summary", read.GetWebhookTitlePath(), "an omitted path is kept")
	assert.Equal(t, "issue.body", read.GetWebhookDescriptionPath())

	for _, invalid := range []*pb.FactoryIntake_Settings{
		{WebhookTitlePath: proto.String("")},
		{WebhookTitlePath: proto.String("issue..summary")},
		{WebhookDescriptionPath: proto.String(`issue."body`)},
		{WebhookLinkPath: proto.String("links.self url")},
	} {
		_, _, err := applyIntakeSettingsToGraph(models.FactoryIntakeSourceWebhook, graph, spec, invalid, slices.Clone(spec.Nodes), spec.Edges)
		assert.Error(t, err, "settings %v", invalid)
	}
}

func TestWebhookIntakeRunTitleReadsTheTitlePath(t *testing.T) {
	event := func(body map[string]any) models.CanvasEvent {
		return models.CanvasEvent{Data: models.NewJSONValue(map[string]any{
			"type": "webhook",
			"data": map[string]any{"body": body},
		})}
	}
	assert.Equal(t, "Checkout fails", webhookIntakeRunTitle("issue.summary", event(map[string]any{"issue": map[string]any{"summary": "Checkout fails"}})))
	assert.Equal(t, webhookIntakeTitleFallback, webhookIntakeRunTitle("issue.summary", event(map[string]any{})))
}

func TestWebhookIntakeCreatesTasksFromRequests(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))
	ctx, deps, orgID := onboardingContext(r), onboardingDeps(r), r.Organization.ID.String()
	newWebhook := func(name string) string {
		created, err := CreateFactoryWebhook(ctx, deps, orgID, &pb.CreateFactoryWebhookRequest{
			FactoryId: factory.ID.String(), Name: name, Authentication: pb.FactoryWebhook_AUTHENTICATION_BEARER,
		})
		require.NoError(t, err)
		return created.GetWebhook().GetId()
	}
	webhookID := newWebhook("Tracker")

	_, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
		FactoryId: factory.ID.String(), Source: pb.FactoryIntake_SOURCE_WEBHOOK, ResourceId: webhookID,
	})
	requireStatusCode(t, err, codes.FailedPrecondition)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryWebhookIntake))

	_, err = CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
		FactoryId: factory.ID.String(), Source: pb.FactoryIntake_SOURCE_WEBHOOK, ResourceId: uuid.NewString(),
	})
	requireStatusCode(t, err, codes.InvalidArgument)

	created, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
		FactoryId: factory.ID.String(), Source: pb.FactoryIntake_SOURCE_WEBHOOK, ResourceId: webhookID,
		Settings: &pb.FactoryIntake_Settings{
			WebhookTitlePath: proto.String("issue.summary"),
			WebhookLinkPath:  proto.String("issue.url"),
		},
	})
	require.NoError(t, err)
	intake := created.GetIntake()
	assert.Equal(t, webhookID, intake.GetResourceId())
	assert.Equal(t, "description", intake.GetSettings().GetWebhookDescriptionPath())
	assert.Equal(t, pb.FactoryIntake_HEALTH_OK, intake.GetHealth())

	canvasID := uuid.MustParse(intake.GetCanvasId())
	trigger, err := models.FindCanvasNode(db, canvasID, intakeTriggerNodeID)
	require.NoError(t, err)
	require.NotNil(t, trigger.WebhookID, "the trigger listens on the workspace webhook")
	assert.Equal(t, webhookID, trigger.WebhookID.String())

	createNode, err := models.FindCanvasNode(db, canvasID, intakeCreateNodeID)
	require.NoError(t, err)
	render := func(body map[string]any) map[string]any {
		configuration, err := contexts.NewNodeConfigurationBuilder(db, canvasID).
			WithRootPayload(map[string]any{"type": "webhook", "data": map[string]any{"body": body}}).
			Build(createNode.Configuration.Data())
		require.NoError(t, err)
		return configuration
	}
	rendered := render(map[string]any{
		"issue":       map[string]any{"summary": "Checkout fails {{ secrets }}", "url": "https://tracker.example.com/42"},
		"description": map[string]any{"steps": 3},
	})
	assert.Equal(t, "Checkout fails {{ secrets }}", rendered["title"], "request text is not evaluated again")
	assert.JSONEq(t, `{"steps": 3}`, rendered["description"].(string))
	assert.Equal(t, "https://tracker.example.com/42", rendered["originUrl"])
	rendered = render(map[string]any{})
	assert.Equal(t, webhookIntakeTitleFallback, rendered["title"])
	assert.Empty(t, rendered["description"])
	assert.Empty(t, rendered["originUrl"])

	otherWebhookID := newWebhook("Other tracker")
	updated, err := UpdateFactoryIntake(ctx, IntakeDependencies{Registry: r.Registry, Encryptor: r.Encryptor, AuthService: r.AuthService, WebhookBaseURL: "http://localhost:8000"}, orgID, &pb.UpdateFactoryIntakeRequest{
		FactoryId: factory.ID.String(), IntakeId: intake.GetId(), ResourceId: proto.String(otherWebhookID),
	})
	require.NoError(t, err)
	assert.Equal(t, otherWebhookID, updated.GetIntake().GetResourceId())
	assert.Equal(t, "issue.summary", updated.GetIntake().GetSettings().GetWebhookTitlePath())
	trigger, err = models.FindCanvasNode(db, canvasID, intakeTriggerNodeID)
	require.NoError(t, err)
	require.NotNil(t, trigger.WebhookID)
	assert.Equal(t, otherWebhookID, trigger.WebhookID.String())

	require.NoError(t, resetTestOrganization(t, r))
	listed, err := ListFactoryIntakes(ctx, orgID, &pb.ListFactoryIntakesRequest{FactoryId: factory.ID.String()})
	require.NoError(t, err)
	require.Len(t, listed.GetIntakes(), 1)
	assert.Equal(t, otherWebhookID, listed.GetIntakes()[0].GetResourceId(), "reset keeps the webhook")
	assert.Equal(t, "issue.url", listed.GetIntakes()[0].GetSettings().GetWebhookLinkPath(), "reset keeps the field paths")
}
