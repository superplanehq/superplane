package factories

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/expr-lang/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/linear"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support/contexts"
	"google.golang.org/protobuf/proto"
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

func Test__intakeFilterExpressionFor_Linear__LegacyProjectsExpressionIsUnchanged(t *testing.T) {
	expression := intakeLinearFilterExpression(intakeSettings{
		LinearProjectIDs:       []string{"project-1"},
		LinearIncludeNoProject: true,
		LinearLabels:           []string{"bug"},
	})

	assert.Equal(t, `(root().data.data.projectId ?? "") in ["project-1"] && any(root().data.data.labels, .name in ["bug"])`, expression)
}

// linearIntakeEvent builds the body of a Linear issue webhook as the intake
// filter sees it under root().data.
func linearIntakeEvent(action string, issue map[string]any, updatedFrom map[string]any) map[string]any {
	data := map[string]any{
		"id":        "issue-1",
		"teamId":    "team-1",
		"projectId": nil,
		"state":     map[string]any{"type": "unstarted"},
		"labels":    []any{map[string]any{"name": "bug"}},
	}
	for key, value := range issue {
		data[key] = value
	}
	event := map[string]any{"action": action, "type": "Issue", "data": data}
	if updatedFrom != nil {
		event["updatedFrom"] = updatedFrom
	}
	return event
}

func evalLinearIntakeFilter(t *testing.T, expression string, event map[string]any) bool {
	t.Helper()

	payload := map[string]any{"data": event}
	program, err := expr.Compile(expression, expr.AsBool(), expr.Function("root", func(params ...any) (any, error) {
		return payload, nil
	}))
	require.NoError(t, err)

	got, err := expr.Run(program, map[string]any{})
	require.NoError(t, err)
	passed, ok := got.(bool)
	require.True(t, ok)
	return passed
}

func Test__intakeFilterExpressionFor_Linear__TeamScope(t *testing.T) {
	teamOnly := intakeSettings{LinearTeamID: "team-1", LinearIncludeNoProject: true}
	projectsWithNoProject := intakeSettings{LinearTeamID: "team-1", LinearProjectIDs: []string{"p1", "p2"}, LinearIncludeNoProject: true}
	projectsOnly := intakeSettings{LinearTeamID: "team-1", LinearProjectIDs: []string{"p1", "p2"}, LinearIncludeNoProject: false}
	withLabel := intakeSettings{LinearTeamID: "team-1", LinearLabels: []string{"bug"}}
	withOtherLabel := intakeSettings{LinearTeamID: "team-1", LinearLabels: []string{"chore"}}

	for _, tc := range []struct {
		name     string
		settings intakeSettings
		event    map[string]any
		want     bool
	}{
		{"create without a project", teamOnly, linearIntakeEvent("create", nil, nil), true},
		{"create in a project", teamOnly, linearIntakeEvent("create", map[string]any{"projectId": "p9"}, nil), true},
		{"edit with no scope change", teamOnly, linearIntakeEvent("update", map[string]any{"title": "New"}, map[string]any{"title": "Old"}), false},
		{"update without updatedFrom", teamOnly, linearIntakeEvent("update", nil, nil), false},
		{"move into the team", teamOnly, linearIntakeEvent("update", nil, map[string]any{"teamId": "team-2"}), true},
		{"move into a project", teamOnly, linearIntakeEvent("update", map[string]any{"projectId": "p1"}, map[string]any{"projectId": nil}), true},
		{"closed issue", teamOnly, linearIntakeEvent("create", map[string]any{"state": map[string]any{"type": "completed"}}, nil), false},
		{"canceled issue", teamOnly, linearIntakeEvent("update", map[string]any{"state": map[string]any{"type": "canceled"}}, map[string]any{"teamId": "team-2"}), false},
		{"issue with no state", teamOnly, linearIntakeEvent("create", map[string]any{"state": nil}, nil), true},
		{"selected project", projectsOnly, linearIntakeEvent("create", map[string]any{"projectId": "p2"}, nil), true},
		{"project not allowed", projectsOnly, linearIntakeEvent("create", map[string]any{"projectId": "p9"}, nil), false},
		{"no project when not included", projectsOnly, linearIntakeEvent("create", nil, nil), false},
		{"no project when included", projectsWithNoProject, linearIntakeEvent("create", nil, nil), true},
		{"project not allowed when no project is included", projectsWithNoProject, linearIntakeEvent("create", map[string]any{"projectId": "p9"}, nil), false},
		{"project removed on update when no project is included", projectsWithNoProject, linearIntakeEvent("update", nil, map[string]any{"projectId": "p1"}), true},
		{"matching label", withLabel, linearIntakeEvent("create", nil, nil), true},
		{"other label", withOtherLabel, linearIntakeEvent("create", nil, nil), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expression := intakeLinearFilterExpression(tc.settings)
			assert.Equal(t, tc.want, evalLinearIntakeFilter(t, expression, tc.event))
		})
	}
}

func Test__applyIntakeSettingsToGraph__LinearRequiresATeamOrProjects(t *testing.T) {
	_, _, err := applyIntakeSettingsToGraph(
		models.FactoryIntakeSourceLinearIssues,
		intakeGraph{TriggerNodeID: intakeTriggerNodeID},
		models.LiveCanvasSpec{},
		&pb.FactoryIntake_Settings{},
		nil,
		nil,
	)
	require.ErrorContains(t, err, "a Linear team is required")

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

func Test__withLinearIntakeScope(t *testing.T) {
	t.Run("legacy intake keeps its projects on the trigger", func(t *testing.T) {
		binding := withLinearIntakeScope(&intakeBinding{
			Configuration: map[string]any{"projects": []any{"project-1"}},
		}, intakeSettings{LinearProjectIDs: []string{"project-1"}, LinearLabels: []string{"bug"}})

		assert.Equal(t, []any{"project-1"}, binding.Configuration["projects"])
		assert.Equal(t, []string{"bug"}, linearLabelsFromConfiguration(binding.Configuration["labels"]))
		assert.NotContains(t, binding.Configuration, "team")
	})

	t.Run("team intake moves its projects into an intake filter", func(t *testing.T) {
		binding := withLinearIntakeScope(&intakeBinding{}, intakeSettings{
			LinearTeamID:           "team-1",
			LinearProjectIDs:       []string{"project-1"},
			LinearIncludeNoProject: false,
		})

		assert.Equal(t, "team-1", binding.Configuration["team"])
		assert.Equal(t, []any{}, binding.Configuration["projects"])
		assert.Equal(t, []any{"project-1"}, binding.Configuration["intakeProjects"])
		assert.Equal(t, false, binding.Configuration["includeIssuesWithoutProject"])
		_, err := structpb.NewStruct(binding.Configuration)
		require.NoError(t, err)
	})
}

func Test__linearScopeSettingsFromConfiguration__RoundTrips(t *testing.T) {
	for _, tc := range []intakeSettings{
		{LinearTeamID: "team-1", LinearProjectIDs: []string{}, LinearIncludeNoProject: true},
		{LinearTeamID: "team-1", LinearProjectIDs: []string{"p1", "p2"}, LinearIncludeNoProject: false},
		{LinearProjectIDs: []string{"p1"}, LinearIncludeNoProject: true},
	} {
		configuration := map[string]any{}
		tc.LinearLabels = []string{"bug"}
		applyLinearTriggerConfiguration(configuration, tc)

		got := linearScopeSettingsFromConfiguration(configuration, defaultLinearIntakeSettings())
		assert.Equal(t, tc.LinearTeamID, got.LinearTeamID)
		assert.Equal(t, tc.LinearProjectIDs, got.LinearProjectIDs)
		assert.Equal(t, tc.LinearIncludeNoProject, got.LinearIncludeNoProject)
		assert.Equal(t, tc.LinearLabels, got.LinearLabels)
	}
}

func Test__applyIntakeSettingsToGraph__LinearTeamIntake(t *testing.T) {
	apply := func(t *testing.T, trigger map[string]any, settings *pb.FactoryIntake_Settings) map[string]any {
		t.Helper()
		node := models.Node{ID: intakeTriggerNodeID, Configuration: trigger}
		nodes, _, err := applyIntakeSettingsToGraph(
			models.FactoryIntakeSourceLinearIssues,
			intakeGraph{TriggerNodeID: intakeTriggerNodeID, FilterNodeID: intakeFilterNodeID},
			models.LiveCanvasSpec{Nodes: []models.Node{node}},
			settings,
			[]models.Node{node, {ID: intakeFilterNodeID, Configuration: map[string]any{}}},
			nil,
		)
		require.NoError(t, err)
		return nodes[0].Configuration
	}

	t.Run("accepts a team without projects", func(t *testing.T) {
		configuration := apply(t, map[string]any{"projects": []any{"project-1"}}, &pb.FactoryIntake_Settings{LinearTeamId: "team-1"})

		assert.Equal(t, "team-1", configuration["team"])
		assert.Equal(t, []any{}, configuration["projects"])
		assert.Equal(t, []any{}, configuration["intakeProjects"])
	})

	t.Run("keeps the saved team when the request omits it", func(t *testing.T) {
		configuration := apply(t, map[string]any{
			"team": "team-1", "projects": []any{}, "intakeProjects": []any{"p1"}, "includeIssuesWithoutProject": true,
		}, &pb.FactoryIntake_Settings{LinearProjectIds: []string{"p2"}, LinearIncludeIssuesWithoutProject: proto.Bool(false)})

		assert.Equal(t, "team-1", configuration["team"])
		assert.Equal(t, []any{"p2"}, configuration["intakeProjects"])
		assert.Equal(t, false, configuration["includeIssuesWithoutProject"])
	})

	t.Run("saves a legacy intake with projects and no team unchanged", func(t *testing.T) {
		configuration := apply(t, map[string]any{"projects": []any{"project-1"}}, &pb.FactoryIntake_Settings{LinearProjectIds: []string{"project-1"}})

		assert.Equal(t, []any{"project-1"}, configuration["projects"])
		assert.NotContains(t, configuration, "team")
		assert.NotContains(t, configuration, "intakeProjects")
	})
}

func Test__linearTriggerSettings__KeepsTheHeldScopeWhenSettingsHaveNone(t *testing.T) {
	configuration := map[string]any{
		"team": "team-1", "projects": []any{}, "intakeProjects": []any{"p1"}, "includeIssuesWithoutProject": false,
	}

	settings := linearTriggerSettings(intakeSettings{}, configuration)

	assert.Equal(t, "team-1", settings.LinearTeamID)
	assert.Equal(t, []string{"p1"}, settings.LinearProjectIDs)
	assert.False(t, settings.LinearIncludeNoProject)
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

	issues, err := newestLinearSeedIssues(client, linear.IssueScope{TeamID: "team-1"}, []string{"bug"})
	require.NoError(t, err)
	require.Len(t, issues, 1)
	assert.Equal(t, "ENG-1", issues[0].Identifier)
}
