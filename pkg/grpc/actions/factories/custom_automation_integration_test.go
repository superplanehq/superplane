package factories

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/automations"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestIntegrationTriggerAutomationSettings(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryAsCode))
	githubID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &githubID}))
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	upsertHostedOnboardingProvider(t, db)
	workspaceAgent := resolveIntakeAgent(db, factory)
	require.NotNil(t, workspaceAgent)
	ctx, deps, orgID := onboardingContext(r), onboardingDeps(r), r.Organization.ID.String()

	pullRequestTrigger := func(actions ...any) *pb.FactoryCustomAutomation_StartTrigger {
		configuration, err := structpb.NewStruct(map[string]any{"repository": "acme/app", "actions": actions, "onlyFactoryPullRequests": true})
		require.NoError(t, err)
		return &pb.FactoryCustomAutomation_StartTrigger{
			StartMode: pb.FactoryCustomAutomation_START_MODE_INTEGRATION,
			Integration: &pb.FactoryCustomAutomation_IntegrationTrigger{
				IntegrationId: githubID, Trigger: "github.onPullRequest", Configuration: configuration,
			},
		}
	}
	runsOnTask := false
	command := `superplane task move --task "{{ root().data.workOrder.key }}" --next`
	settings := &pb.FactoryCustomAutomation_Settings{
		StartMode:               pb.FactoryCustomAutomation_START_MODE_INTEGRATION,
		RunsOnTask:              &runsOnTask,
		Triggers:                []*pb.FactoryCustomAutomation_StartTrigger{pullRequestTrigger("closed")},
		Agent:                   &pb.FactoryAppTemplateAgent{Component: workspaceAgent.Component, Model: workspaceAgent.Model},
		ExecutionTimeoutSeconds: 3600,
		Steps:                   []*pb.FactoryCustomAutomation_Step{{Name: "Move", Type: pb.FactoryCustomAutomation_Step_TYPE_BASH, Command: command}},
	}
	created, err := CreateFactoryCustomAutomation(ctx, deps, orgID, &pb.CreateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), Name: "Move on merge", ColumnKey: "backlog", Settings: settings,
	})
	require.NoError(t, err)
	assert.True(t, proto.Equal(settings, created.GetCustomAutomation().GetSettings()), "the integration trigger reads back as saved")

	canvasID := uuid.MustParse(created.GetCustomAutomation().GetAutomation().GetId())
	version, err := models.FindLiveCanvasVersionInTransaction(db, canvasID)
	require.NoError(t, err)
	var trigger *models.Node
	for i := range version.Nodes {
		if isCustomAutomationIntegrationNode(version.Nodes[i]) {
			trigger = &version.Nodes[i]
		}
	}
	require.NotNil(t, trigger)
	assert.True(t, strings.HasPrefix(trigger.ID, "integration-"))
	assert.Equal(t, "github.onPullRequest", trigger.ComponentName())
	require.NotNil(t, trigger.IntegrationID)
	assert.Equal(t, githubID, *trigger.IntegrationID)

	service := automations.Service{Backend: CanvasAutomationBackend{Deps: deps}}
	read, err := service.Read(ctx, r.Organization.ID, canvasID)
	require.NoError(t, err)
	assert.Contains(t, read.YAML, "use: integration")
	assert.Contains(t, read.YAML, "integration_id: "+githubID)
	assert.Contains(t, read.YAML, "trigger: github.onPullRequest")
	assert.Contains(t, read.YAML, "- closed")

	settings.Triggers = []*pb.FactoryCustomAutomation_StartTrigger{pullRequestTrigger("opened")}
	updated, err := UpdateFactoryCustomAutomation(ctx, deps, orgID, &pb.UpdateFactoryCustomAutomationRequest{
		FactoryId: factory.ID.String(), AutomationId: canvasID.String(), Name: "Move on merge", Settings: settings,
	})
	require.NoError(t, err)
	assert.True(t, proto.Equal(settings, updated.GetCustomAutomation().GetSettings()), "the changed event reads back")

	for name, change := range map[string]func(*pb.FactoryCustomAutomation_Settings){
		"no integration": func(s *pb.FactoryCustomAutomation_Settings) { s.Triggers[0].Integration.IntegrationId = "" },
		"unknown trigger": func(s *pb.FactoryCustomAutomation_Settings) {
			s.Triggers[0].Integration.Trigger = "github.onNothing"
		},
		"missing repository": func(s *pb.FactoryCustomAutomation_Settings) {
			delete(s.Triggers[0].Integration.Configuration.Fields, "repository")
		},
		"duplicate event": func(s *pb.FactoryCustomAutomation_Settings) {
			s.Triggers = append(s.Triggers, pullRequestTrigger("opened"))
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
}
