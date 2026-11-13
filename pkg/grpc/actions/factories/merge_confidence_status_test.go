package factories

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/pulls"
	"github.com/superplanehq/superplane/pkg/models"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
	"gorm.io/datatypes"
)

func TestMergeConfidenceCanvasAcceptsAssessmentCheckStateExpression(t *testing.T) {
	raw, err := factoryTemplateFiles.ReadFile("templates/github/risk-score.canvas.yaml")
	require.NoError(t, err)
	canvas, err := yaml.CanvasFromYAML(raw)
	require.NoError(t, err)

	check := findYAMLNode(t, canvas, "finish-assessment-check")
	require.NoError(t, configuration.ValidateConfiguration((&pulls.SetPullRequestCheckRun{}).Configuration(), check.Configuration))
}

func TestMergeConfidenceCanvasAssessmentStatus(t *testing.T) {
	for _, scenario := range []struct {
		name, channel, expected string
		score                   float64
	}{
		{"complete", "passed", "success", 4},
		{"critical findings remain advisory", "passed", "success", 1},
		{"failed", "failed", "failure", 4},
		{"incomplete", "passed", "failure", 0},
		{"stale publication", "passed", "failure", 4},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			f := newPublicationFixture(t)
			root, err := f.run.RootEvent(f.db)
			require.NoError(t, err)
			require.NoError(t, f.db.Model(root).Update("data", models.NewJSONValue(map[string]any{"data": map[string]any{
				"repository":   map[string]any{"full_name": "acme/app"},
				"pull_request": map[string]any{"number": 42, "head": map[string]any{"sha": "abc"}},
			}})).Error)
			raw, err := factoryTemplateFiles.ReadFile("templates/github/risk-score.canvas.yaml")
			require.NoError(t, err)
			doc, err := yaml.CanvasFromYAML(raw)
			require.NoError(t, err)
			build := func(id string, previous *models.CanvasNodeExecution, incoming *models.CanvasEvent) map[string]any {
				builder := workercontexts.NewNodeConfigurationBuilder(f.db, f.canvas.ID).
					WithRunID(f.run.ID).WithNodeID(id).WithRootEvent(&root.ID).
					WithConfigurationFields((&pulls.SetPullRequestCheckRun{}).Configuration())
				if previous != nil {
					builder = builder.WithPreviousExecution(&previous.ID).WithIncomingEventID(&incoming.ID).WithInput(map[string]any{incoming.NodeID: incoming.Data.Data()})
				}
				configuration, err := builder.Build(findYAMLNode(t, doc, id).Configuration)
				require.NoError(t, err)
				return configuration
			}
			installation, err := models.CreateIntegration(uuid.New(), f.canvas.OrganizationID, "github", "github", nil)
			require.NoError(t, err)
			var checkRunIdentity string
			publishCheck := func(id string, config map[string]any) {
				node := &models.CanvasNode{WorkflowID: f.canvas.ID, NodeID: id, Type: models.NodeTypeComponent,
					AppInstallationID: &installation.ID, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: findYAMLNode(t, doc, id).Component}})}
				require.NoError(t, f.db.Create(node).Error)
				action, err := f.deps.Registry.GetAction(node.ComponentName())
				require.NoError(t, err)
				execution := support.CreateCanvasNodeExecution(t, f.canvas.ID, id, root.ID, root.ID)
				publications := workercontexts.NewPublicationContext(f.db, f.canvas.OrganizationID, node, execution)
				require.NoError(t, action.Execute(core.ExecutionContext{
					ID: execution.ID, RunID: f.run.ID, WorkflowID: f.canvas.ID.String(), NodeID: id,
					Configuration: config, Integration: reviewRequestIntegration(), HTTP: &contexts.HTTPContext{},
					Publications: publications, Metadata: workercontexts.NewExecutionMetadataContext(f.db, execution),
					Requests: workercontexts.NewExecutionRequestContext(f.db, execution), ExecutionState: workercontexts.NewExecutionStateContext(f.db, execution, nil),
				}))
				records, err := publications.List("github.review-check/")
				require.NoError(t, err)
				require.Len(t, records, 1)
				var payload struct {
					Configuration struct{ State string }
					CheckRun      struct{ ExternalID string }
				}
				require.NoError(t, json.Unmarshal(records[0].Payload, &payload))
				require.Equal(t, config["state"], payload.Configuration.State)
				require.NotEmpty(t, payload.CheckRun.ExternalID)
				if checkRunIdentity != "" {
					require.Equal(t, checkRunIdentity, payload.CheckRun.ExternalID)
				}
				checkRunIdentity = payload.CheckRun.ExternalID
			}
			start := build("start-assessment-check", nil, nil)
			require.Equal(t, "pending", start["state"])
			require.Equal(t, "abc", start["assessedSha"])
			require.NotEmpty(t, start["resultUrl"])
			publishCheck("start-assessment-check", start)
			if scenario.score > 0 {
				f.report(t, f.run, scenario.score)
			}
			execution, request := f.publicationRequest(t, scenario.channel)
			read := descriptionResponse(t, models.ProviderGitHub, "User content")
			if scenario.name == "stale publication" {
				read = publicationHTTPResponse(200, `{"state":"open","head":{"sha":"new"}}`)
			}
			installPublicationGitHub(t, &contexts.HTTPContext{Responses: []*http.Response{read, publicationHTTPResponse(200, `{}`)}})
			events, err := ProcessBodySectionPublicationRequest(t.Context(), IntakeDependencies{}, request.ID, models.LockNodeRequestNow)
			require.NoError(t, err)
			require.Len(t, events, 1)
			incoming, err := models.FindCanvasEventInTransaction(f.db, events[0].ID)
			require.NoError(t, err)
			finish := build("finish-assessment-check", execution, incoming)
			require.Equal(t, scenario.expected, finish["state"])
			require.Equal(t, start["context"], finish["context"])
			require.Equal(t, start["assessedSha"], finish["assessedSha"])
			publishCheck("finish-assessment-check", finish)
		})
	}
}
