package contexts

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/components/runner/claude"
	"github.com/superplanehq/superplane/pkg/components/runner/codex"
	"github.com/superplanehq/superplane/pkg/components/runner/openrouter"
	"github.com/superplanehq/superplane/pkg/components/runner/superplane"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	runnerapi "github.com/superplanehq/superplane/pkg/runners/api"
	"github.com/superplanehq/superplane/test/support"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
	"gorm.io/datatypes"
)

func TestFactoryAgentStartupResolvesEnvironmentOnce(t *testing.T) {
	for _, provider := range []struct {
		action core.Action
		apiKey string
	}{
		{&claude.RunClaudeCode{}, "ANTHROPIC_API_KEY"},
		{&codex.RunCodex{}, "OPENAI_API_KEY"},
		{&openrouter.RunOpenRouter{}, "OPENROUTER_API_KEY"},
		{&superplane.RunSuperPlane{}, "ANTHROPIC_API_KEY"},
	} {
		for _, marker := range []string{"ordinary", "integration", "secret", "explicit override"} {
			t.Run(provider.action.Name()+"/"+marker, func(t *testing.T) {
				r := support.Setup(t)
				defer r.Close()
				db := database.DB(t.Context())
				t.Setenv("JWT_SECRET", "test")
				workspace, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("startup"), "", "")
				require.NoError(t, err)
				order, err := workspace.CreateWorkOrder(db, "Run agent", "", &r.User, nil, nil)
				require.NoError(t, err)
				canvas, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, workspace.ID, "agent", "start")
				require.NoError(t, db.Model(&models.CanvasNode{}).Where("workflow_id = ?", canvas.ID).
					Update("metadata", datatypes.NewJSONType(map[string]any{
						models.FactoryAppTemplateMetadataKey: map[string]any{"id": models.FactoryAppTemplateCustomAutomationID},
					})).Error)
				root := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "start", "default", nil,
					map[string]any{"workOrder": map[string]any{"id": order.ID.String()}})
				execution := support.CreateCanvasNodeExecution(t, canvas.ID, "start", root.ID, root.ID)
				fleet := models.RunnerFleet{ID: uuid.New(), Slug: support.RandomName("startup"),
					ScopeType: models.RunnerFleetScopeInstallation, Enabled: true,
					Spec: datatypes.NewJSONType(models.RunnerFleetSpec{}), RunnerVersion: models.DefaultRunnerVersion}
				require.NoError(t, fleet.Create(db))

				// Each execution gets distinct credentials, so sharing results across executions also fails.
				credential := uuid.NewString()
				secrets := &startupSecrets{SecretsContext: testcontexts.SecretsContext{
					Values: map[string][]byte{"explicit/token": []byte(credential), "agent/key": []byte("agent-key")},
					SecretKeys: map[string]map[string][]byte{"imported": {
						"IMPORTED_SECRET": []byte(credential), "OVERRIDE": []byte("stale"),
						"SUPERPLANE_MERGE_CONFIDENCE_TOKEN": []byte("stale"),
					}},
					IntegrationKeys:      map[string]map[string][]byte{"github": {"GITHUB_TOKEN": []byte(credential)}},
					IntegrationUsage:     map[string]string{"github": "Use the imported GitHub credentials."},
					IntegrationSetup:     map[string]string{"github": "echo prepare-github"},
					IntegrationSetupName: map[string]string{"github": "Set up GitHub"},
				}, reads: map[string]int{}}
				environment := []map[string]any{{"name": " OVERRIDE ", "valueSource": "secret",
					"secret": map[string]any{"secret": "explicit", "key": "token"}}}
				switch marker {
				case "integration":
					secrets.IntegrationKeys["github"][" "+runner.EnvSuperplaneMergeConfidenceOrderID+" "] = []byte(order.ID.String())
				case "secret":
					secrets.SecretKeys["imported"][runner.EnvSuperplaneMergeConfidenceOrderID] = []byte(order.ID.String())
				case "explicit override":
					secrets.SecretKeys["imported"][runner.EnvSuperplaneMergeConfidenceOrderID] = []byte("invalid-order")
					environment = append(environment, map[string]any{"name": " " + runner.EnvSuperplaneMergeConfidenceOrderID + " ",
						"valueSource": "literal", "value": order.ID.String()})
				}
				configuration := map[string]any{
					"machineType": fleet.Slug,
					"model":       "test-model",
					"credentials": map[string]any{"source": "secret", "secret": map[string]any{"secret": "agent", "key": "key"}},
					"environmentFrom": []map[string]any{
						{"source": "integration", "integration": map[string]any{"name": "github"}},
						{"source": "secret", "secret": map[string]any{"secret": "imported"}},
					},
					"environment": environment,
					"steps":       []map[string]any{{"name": "Review", "type": "prompt", "prompt": "Merge check: security.\nReview the change."}},
				}
				if provider.action.Name() == superplane.ComponentName {
					configuration["credentials"] = map[string]any{"source": "hosted"}
					configuration["hostedProvider"] = models.UsageProviderAnthropic
					configuration["model"] = "claude-sonnet-4-6"
				}
				state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
				ctx := core.ExecutionContext{
					ID: execution.ID, RunID: execution.RunID, WorkflowID: canvas.ID.String(), NodeID: execution.NodeID,
					OrganizationID: r.Organization.ID.String(), BaseURL: "https://app.example",
					Configuration: configuration, Secrets: secrets, ExecutionState: state,
					RunnerTasks: NewRunnerTaskContext(db, crypto.NewNoOpEncryptor(), r.Organization.ID),
					Requests:    &testcontexts.RequestContext{}, HTTP: &testcontexts.HTTPContext{},
					HostedLLM: &testcontexts.HostedLLMContext{Access: core.HostedLLMAccess{
						APIKey: "agent-key", AllowedModels: []string{"claude-sonnet-4-6"},
					}},
				}
				require.NoError(t, PrepareFactoryAgentContext(db, &ctx, provider.action.Name()))
				require.NoError(t, provider.action.Execute(ctx))
				task, err := models.FindRunnerTask(db, uuid.MustParse(state.KVs["task_id"]))
				require.NoError(t, err)
				var payload runnerapi.TaskPayload
				require.NoError(t, json.Unmarshal(task.PayloadCiphertext, &payload))
				variables := map[string]string{}
				for _, variable := range payload.Environment {
					variables[variable.Name] = variable.Value
				}
				assert.Equal(t, credential, variables["GITHUB_TOKEN"])
				assert.Equal(t, credential, variables["IMPORTED_SECRET"])
				assert.Equal(t, credential, variables["OVERRIDE"])
				assert.Equal(t, "agent-key", variables[provider.apiKey])
				assert.NotContains(t, variables, runner.EnvSuperplaneMergeConfidenceOrderID)
				assert.NotContains(t, variables, "SUPERPLANE_MERGE_CONFIDENCE_TOKEN")
				scope, err := runner.ParseRunnerToken(jwt.NewSigner("test"), variables[runner.EnvSuperplaneRunnerToken])
				require.NoError(t, err)
				assert.Equal(t, order.ID, scope.WorkOrderID)
				assert.Equal(t, execution.ID, scope.NodeExecutionID)
				if marker == "ordinary" {
					assert.Equal(t, runner.RunnerFlowTask, scope.Flow)
					assert.NotContains(t, scope.Scopes, runner.ScopeMergeChecksWrite)
				} else {
					assert.Equal(t, runner.RunnerFlowMergeConfidence, scope.Flow)
					assert.Equal(t, []string{"security"}, scope.MergeChecks)
					assert.Contains(t, scope.Scopes, runner.ScopeMergeChecksWrite)
				}
				files := map[string]string{}
				for _, file := range payload.Files {
					files[file.Path] = file.Content
				}
				assert.Contains(t, files["prompts/01-review.txt"], "Use the imported GitHub credentials.")
				assert.Equal(t, "echo prepare-github", files["setup/01-set-up-github.sh"])
				assert.Contains(t, payload.Commands[1].Command, "setup/01-set-up-github.sh")
				assert.Equal(t, 1, secrets.reads["integration:github"])
				assert.Equal(t, 1, secrets.reads["secret:imported"])
				assert.Equal(t, 1, secrets.reads["key:explicit/token"])
			})
		}
	}
}

type startupSecrets struct {
	testcontexts.SecretsContext
	reads map[string]int
}

func (s *startupSecrets) GetKey(secret, key string) ([]byte, error) {
	s.reads["key:"+secret+"/"+key]++
	return s.SecretsContext.GetKey(secret, key)
}

func (s *startupSecrets) GetSecretKeys(secret string) (map[string][]byte, error) {
	s.reads["secret:"+secret]++
	return s.SecretsContext.GetSecretKeys(secret)
}

func (s *startupSecrets) GetIntegrationSecrets(name string) (core.IntegrationSecrets, error) {
	s.reads["integration:"+name]++
	return s.SecretsContext.GetIntegrationSecrets(name)
}
