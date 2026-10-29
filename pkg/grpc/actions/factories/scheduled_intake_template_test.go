package factories

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func TestScheduledIntakeCreation(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("scheduled"), "", "")
	require.NoError(t, err)
	repository, branch := "acme/app", "develop"
	vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{AppRepository: &repository, DefaultBranch: &branch, VCSIntegrationID: &vcsID}))
	enableHostedSuperPlaneAgent(t, db)
	deps := IntakeDependencies{Registry: r.Registry, Encryptor: r.Encryptor, AuthService: r.AuthService, WebhookBaseURL: "http://localhost:8000"}
	request := &pb.CreateFactoryAutomationRequest{FactoryId: factory.ID.String(), TemplateId: scheduledIntakeTemplateID, InstallParams: map[string]string{"hour": "11", "minute": "45", "timezone": "5.5", "prompt": "Review AGENTS.md.\nPreserve quotes: ' and {{ user text }}."}}
	_, err = CreateFactoryAutomation(ctx, deps, r.Organization.ID.String(), request)
	code, _, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	require.Equal(t, codes.FailedPrecondition, code)
	canvases, err := factory.ListCanvases(db)
	require.NoError(t, err)
	require.Empty(t, canvases)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryScheduledIntake))
	for _, component := range []string{models.SuperPlaneRunnerComponent, "runnerCodex"} {
		t.Run(component, func(t *testing.T) {
			if component == "runnerCodex" {
				id := createReadyOnboardingIntegration(t, r.Organization.ID, "openai")
				require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{AgentIntegrationID: &id}))
			}
			response, err := CreateFactoryAutomation(ctx, deps, r.Organization.ID.String(), request)
			require.NoError(t, err)
			require.Equal(t, "backlog", response.Automation.ColumnKey)
			version, err := models.FindLiveCanvasVersionInTransaction(db, uuid.MustParse(response.Automation.Id))
			require.NoError(t, err)
			require.Equal(t, scheduledIntakeTemplateID, models.FactoryAppTemplateID(version.Nodes))
			require.Len(t, version.Nodes, 5)
			require.Len(t, version.Edges, 4)
			for _, edge := range version.Edges {
				if edge.SourceID == "scheduled-agent" {
					if edge.TargetID == "has-proposal" {
						require.Equal(t, "passed", edge.Channel)
					}
					if edge.TargetID == "add-run-error" {
						require.Equal(t, "failed", edge.Channel)
					}
				}
			}
			for _, node := range version.Nodes {
				if node.ID == "schedule" {
					require.Equal(t, float64(11), node.Configuration["hour"])
					require.Equal(t, "5.5", node.Configuration["timezone"])
				}
				if node.ID == "scheduled-agent" {
					require.Equal(t, component, node.ComponentName())
					steps := node.Configuration["steps"].([]any)
					require.Equal(t, "set -euo pipefail\nrm -f /tmp/proposal.json", steps[0].(map[string]any)["command"])
					require.Contains(t, steps[1].(map[string]any)["prompt"], request.InstallParams["prompt"])
					env := node.Configuration["environment"].([]any)
					require.Equal(t, "https://github.com/acme/app.git", env[0].(map[string]any)["value"])
					require.Equal(t, "develop", env[1].(map[string]any)["value"])
					if component == models.SuperPlaneRunnerComponent {
						require.NotContains(t, node.Configuration, "credentials")
					}
				}
			}
		})
	}
	request.InstallParams["timezone"] = "Europe/Belgrade"
	_, err = CreateFactoryAutomation(ctx, deps, r.Organization.ID.String(), request)
	code, _, ok = grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, code)
}

func TestScheduledIntakeLeavesBaseEmptyWhenWorkspaceBranchIsEmpty(t *testing.T) {
	result, err := materializeFactoryTemplate(scheduledIntakeTemplateID, models.ProviderGitHub, factoryTemplateInput{
		installParams: map[string]string{"appRepository": "acme/app", "defaultBranch": ""},
	})
	require.NoError(t, err)
	canvas, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
	require.NoError(t, err)
	require.NoError(t, configureScheduledIntake(canvas, models.ProviderGitHub, map[string]string{
		"appRepository": "acme/app",
		"defaultBranch": "",
	}))

	agent := findYAMLNode(t, canvas, "scheduled-agent")
	env := agent.Configuration["environment"].([]any)
	require.GreaterOrEqual(t, len(env), 2)
	require.Equal(t, "BASE", env[1].(map[string]any)["name"])
	require.Equal(t, "", env[1].(map[string]any)["value"])
}

func TestScheduledIntakeProposalResult(t *testing.T) {
	result, err := materializeFactoryTemplate(scheduledIntakeTemplateID, models.ProviderGitHub, factoryTemplateInput{installParams: map[string]string{"appRepository": "acme/app"}})
	require.NoError(t, err)
	canvas, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
	require.NoError(t, err)
	agent := findYAMLNode(t, canvas, "scheduled-agent")
	steps := agent.Configuration["steps"].([]any)
	script := steps[2].(map[string]any)["command"].(string)
	for _, tc := range []struct {
		name, proposal string
		missing, fails bool
	}{
		{name: "proposal", proposal: `{"title":"Add regression coverage","description":"Cover the missing error path."}`},
		{name: "nothing useful", proposal: `{}`},
		{name: "missing file", missing: true, fails: true},
		{name: "invalid JSON", proposal: `broken`, fails: true},
		{name: "missing description", proposal: `{"title":"A task"}`, fails: true},
		{name: "blank title", proposal: `{"title":"  ","description":"A task"}`, fails: true},
		{name: "wrong type", proposal: `{"title":42,"description":"A task"}`, fails: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()
			proposal, output := filepath.Join(directory, "proposal.json"), filepath.Join(directory, "result.json")
			if !tc.missing {
				require.NoError(t, os.WriteFile(proposal, []byte(tc.proposal), 0600))
			}
			command := exec.CommandContext(t.Context(), "bash", "-c", strings.ReplaceAll(script, "/tmp/proposal.json", proposal))
			command.Env = append(os.Environ(), "SUPERPLANE_RESULT_FILE="+output)
			logs, err := command.CombinedOutput()
			if tc.fails {
				require.Error(t, err, string(logs))
				return
			}
			require.NoError(t, err, string(logs))
			data, err := os.ReadFile(output)
			require.NoError(t, err)
			var decoded map[string]any
			require.NoError(t, json.Unmarshal(data, &decoded))
			if tc.proposal == "{}" {
				require.Empty(t, decoded)
			} else {
				require.Equal(t, "Add regression coverage", decoded["title"])
			}
		})
	}
}
