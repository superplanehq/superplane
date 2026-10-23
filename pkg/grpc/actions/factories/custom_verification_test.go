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
)

func TestCustomVerificationSettingsAllowBashOnly(t *testing.T) {
	r := support.Setup(t)
	factory := seedHandOffFactory(t, r, database.DB(t.Context()))

	settings, err := parseCustomVerificationSettings(database.DB(t.Context()), factory, &pb.FactoryCustomVerification_Settings{
		StartMode: pb.FactoryCustomVerification_START_MODE_MANUAL,
		Agent:     &pb.FactoryAppTemplateAgent{Component: "runnerClaudeCode"},
		Steps: []*pb.FactoryCustomVerification_Step{
			{Name: "Lint", Type: pb.FactoryCustomVerification_Step_TYPE_BASH, Command: "make lint"},
		},
	})
	require.NoError(t, err)
	require.Len(t, settings.Steps, 1)
	assert.Equal(t, runner.AgentStepBash, settings.Steps[0].Type)
}

func TestCustomVerificationGraphUsesProviderAndFocusedSettings(t *testing.T) {
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
			settings, err := parseCustomVerificationSettings(db, factory, &pb.FactoryCustomVerification_Settings{
				StartMode: pb.FactoryCustomVerification_START_MODE_AUTOMATIC,
				Agent: &pb.FactoryAppTemplateAgent{
					Component:                 "runnerClaudeCode",
					Model:                     "claude-sonnet-4-6",
					CredentialSource:          runner.CredentialsSourceIntegration,
					CredentialIntegrationName: "Claude",
				},
				Steps: []*pb.FactoryCustomVerification_Step{
					{Name: "Review", Type: pb.FactoryCustomVerification_Step_TYPE_PROMPT, Prompt: prompt, WorkingDirectory: "repo"},
					{
						Name:             "Test",
						Type:             pb.FactoryCustomVerification_Step_TYPE_BASH,
						Command:          command,
						WorkingDirectory: "repo",
						Secrets:          []string{"deploy-credentials"},
					},
				},
			})
			require.NoError(t, err)

			doc, err := buildCustomVerificationCanvas(db, factory, "Verify changes", settings)
			require.NoError(t, err)
			require.Len(t, doc.Spec.Nodes, 2)
			trigger := doc.Spec.Nodes[0]
			agent := doc.Spec.Nodes[1]

			assert.Equal(t, prFeedbackTriggerComponent(provider), trigger.Component)
			assert.Equal(t, prFeedbackTriggerActions(provider), trigger.Configuration["actions"])
			assert.Equal(t, integrationID, trigger.Integration.ID)
			template := trigger.Metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
			assert.Equal(t, models.FactoryAppTemplateCustomVerifyID, template["id"])
			assert.Equal(t, "automatic", template["startMode"])
			assert.Equal(t, "runnerClaudeCode", agent.Component)
			assert.Equal(t, "claude-sonnet-4-6", agent.Configuration["model"])
			assert.Equal(t, customVerificationStepConfiguration(settings.Steps), agent.Configuration["steps"])
			environmentFrom := agent.Configuration["environmentFrom"].([]any)
			require.Len(t, environmentFrom, 2)
			secretSource := environmentFrom[1].(map[string]any)
			assert.Equal(t, runner.EnvironmentFromSourceSecret, secretSource["source"])
			assert.Equal(t, "deploy-credentials", secretSource["secret"].(map[string]any)["secret"])
		})
	}
}
