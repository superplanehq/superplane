package factories

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/linear"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support/contexts"
	"google.golang.org/protobuf/types/known/structpb"
)

func Test__intakeFilterExpressionFor_Linear(t *testing.T) {
	expression := intakeLinearFilterExpression(intakeSettings{
		LinearProjectIDs: []string{"project-1", "project-2"},
		LinearLabels:     []string{"bug"},
	})

	assert.Contains(t, expression, `root().data.data.projectId`)
	assert.Contains(t, expression, `"project-1"`)
	assert.Contains(t, expression, `"project-2"`)
	assert.Contains(t, expression, `any(root().data.data.labels, .name in ["bug"])`)
}

func Test__intakeFilterExpressionFor_Linear__Empty(t *testing.T) {
	assert.Equal(t, "true", intakeLinearFilterExpression(intakeSettings{}))
}

func Test__applyIntakeSettingsToGraph__LinearRequiresAProject(t *testing.T) {
	_, _, err := applyIntakeSettingsToGraph(
		models.FactoryIntakeSourceLinearIssues,
		intakeGraph{TriggerNodeID: intakeTriggerNodeID},
		models.LiveCanvasSpec{},
		&pb.FactoryIntake_Settings{},
		nil,
		nil,
	)
	require.ErrorContains(t, err, "at least one Linear project is required")

	nodes, _, err := applyIntakeSettingsToGraph(
		models.FactoryIntakeSourceLinearIssues,
		intakeGraph{TriggerNodeID: intakeTriggerNodeID, FilterNodeID: intakeFilterNodeID},
		models.LiveCanvasSpec{Nodes: []models.Node{{
			ID:            intakeTriggerNodeID,
			Configuration: map[string]any{"projects": []any{"project-1"}},
		}}},
		&pb.FactoryIntake_Settings{LinearProjectIds: []string{"project-1", "project-2"}},
		[]models.Node{{
			ID:            intakeTriggerNodeID,
			Configuration: map[string]any{"projects": []any{"project-1"}},
		}},
		nil,
	)
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, []any{"project-1", "project-2"}, nodes[0].Configuration["projects"])
	_, err = structpb.NewStruct(nodes[0].Configuration)
	require.NoError(t, err)
}

func Test__withLinearSeedLabels(t *testing.T) {
	binding := withLinearSeedLabels(&intakeBinding{
		Configuration: map[string]any{"projects": []string{"project-1"}},
	}, []string{"bug"})

	assert.Equal(t, []string{"bug"}, linearLabelsFromConfiguration(binding.Configuration["labels"]))
}

func Test__newestLinearSeedIssues__KeepsOnlyMatchingLabels(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{"data":{"issues":{"nodes":[
					{"id":"1","identifier":"ENG-1","title":"Bug","labels":{"nodes":[{"id":"l1","name":"bug"}]}},
					{"id":"2","identifier":"ENG-2","title":"Chore","labels":{"nodes":[{"id":"l2","name":"chore"}]}}
				]}}}`)),
			},
		},
	}
	client, err := linear.NewClient(httpContext, &contexts.IntegrationContext{
		CurrentSecrets: map[string]core.IntegrationSecret{
			linear.OAuthAccessToken: {Name: linear.OAuthAccessToken, Value: []byte("token")},
		},
	})
	require.NoError(t, err)

	issues, err := newestLinearSeedIssues(client, []string{"project-1"}, []string{"bug"})
	require.NoError(t, err)
	require.Len(t, issues, 1)
	assert.Equal(t, "ENG-1", issues[0].Identifier)
}
