package factories

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/pulls"
	"github.com/superplanehq/superplane/pkg/models"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support/contexts"
)

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
					WithConfigurationFields((&pulls.SetPullRequestCheck{}).Configuration())
				if previous != nil {
					builder = builder.WithPreviousExecution(&previous.ID).WithIncomingEventID(&incoming.ID).WithInput(map[string]any{incoming.NodeID: incoming.Data.Data()})
				}
				configuration, err := builder.Build(findYAMLNode(t, doc, id).Configuration)
				require.NoError(t, err)
				return configuration
			}
			start := build("start-assessment-check", nil, nil)
			require.Equal(t, "pending", start["state"])
			require.Equal(t, "abc", start["assessedSha"])
			require.NotEmpty(t, start["resultUrl"])
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
		})
	}
}
