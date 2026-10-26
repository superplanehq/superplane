package factories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestCustomAutomationSettingsAllowBashOnly(t *testing.T) {
	r := support.Setup(t)
	factory := seedHandOffFactory(t, r, database.DB(t.Context()))

	settings, err := parseCustomAutomationSettings(database.DB(t.Context()), factory, &pb.FactoryCustomAutomation_Settings{
		StartMode: pb.FactoryCustomAutomation_START_MODE_MANUAL,
		Agent:     &pb.FactoryAppTemplateAgent{Component: "runnerClaudeCode"},
		Steps: []*pb.FactoryCustomAutomation_Step{
			{Name: "Lint", Type: pb.FactoryCustomAutomation_Step_TYPE_BASH, Command: "make lint"},
		},
	})
	require.NoError(t, err)
	require.Len(t, settings.Steps, 1)
	assert.Equal(t, runner.AgentStepBash, settings.Steps[0].Type)
}

func TestCustomAutomationGraphUsesProviderAndFocusedSettings(t *testing.T) {
	for _, provider := range []string{models.ProviderGitHub, models.ProviderBitbucket} {
		t.Run(provider, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			factory := seedHandOffFactory(t, r, db)
			integrationID := uuid.NewString()
			require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
				VCSProvider:      &provider,
				VCSIntegrationID: &integrationID,
			}))
			prompt := "Review the pull request."
			command := "make test"
			settings, err := parseCustomAutomationSettings(db, factory, &pb.FactoryCustomAutomation_Settings{
				StartMode: pb.FactoryCustomAutomation_START_MODE_AUTOMATIC,
				Agent: &pb.FactoryAppTemplateAgent{
					Component:                 "runnerClaudeCode",
					Model:                     "claude-sonnet-4-6",
					CredentialSource:          runner.CredentialsSourceIntegration,
					CredentialIntegrationName: "Claude",
				},
				Steps: []*pb.FactoryCustomAutomation_Step{
					{Name: "Review", Type: pb.FactoryCustomAutomation_Step_TYPE_PROMPT, Prompt: prompt, WorkingDirectory: "repo"},
					{
						Name:             "Test",
						Type:             pb.FactoryCustomAutomation_Step_TYPE_BASH,
						Command:          command,
						WorkingDirectory: "repo",
						Secrets:          []string{"deploy-credentials"},
					},
				},
			})
			require.NoError(t, err)

			doc, err := buildCustomAutomationCanvas(db, factory, "Verify changes", settings)
			require.NoError(t, err)
			require.Len(t, doc.Spec.Nodes, 2)
			trigger := doc.Spec.Nodes[0]
			agent := doc.Spec.Nodes[1]

			assert.Equal(t, prFeedbackTriggerComponent(provider), trigger.Component)
			assert.Equal(t, prFeedbackTriggerActions(provider), trigger.Configuration["actions"])
			assert.Equal(t, integrationID, trigger.Integration.ID)
			template := trigger.Metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
			assert.Equal(t, models.FactoryAppTemplateCustomAutomationID, template["id"])
			assert.Equal(t, "automatic", template["startMode"])
			assert.Equal(t, "runnerClaudeCode", agent.Component)
			assert.Equal(t, "claude-sonnet-4-6", agent.Configuration["model"])
			assert.Equal(t, customAutomationStepConfiguration(settings.Steps), agent.Configuration["steps"])
			environmentFrom := agent.Configuration["environmentFrom"].([]any)
			require.Len(t, environmentFrom, 2)
			secretSource := environmentFrom[1].(map[string]any)
			assert.Equal(t, runner.EnvironmentFromSourceSecret, secretSource["source"])
			assert.Equal(t, "deploy-credentials", secretSource["secret"].(map[string]any)["secret"])
			got := environmentVars(t, agent.Configuration["environment"])
			assert.Equal(t, "acme/app", got["REPO"])
			assert.Equal(t, "acme/app", got["SOURCE_REPO"])
			assert.Equal(t, runner.WorkspaceContextRepoURL(provider, "acme/app"), got["REPO_URL"])
			assert.Equal(t, "main", got["BASE"])
			assert.Equal(t, prFeedbackPRNumberExpression(), got["PR_NUMBER"])
			assert.Equal(t, prFeedbackPRHeadExpression(), got["PR_HEAD"])
			_, err = structpb.NewStruct(agent.Configuration)
			require.NoError(t, err)
		})
	}
}

func environmentVars(t *testing.T, raw any) map[string]string {
	t.Helper()
	entries, ok := raw.([]any)
	require.True(t, ok)
	got := make(map[string]string, len(entries))
	for _, entry := range entries {
		item, ok := entry.(map[string]any)
		require.True(t, ok)
		name, _ := item["name"].(string)
		value, _ := item["value"].(string)
		require.NotEmpty(t, name)
		got[name] = value
	}
	return got
}
