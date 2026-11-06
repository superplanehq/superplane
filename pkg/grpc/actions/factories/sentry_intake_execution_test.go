package factories

import (
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/configuration/expressionvalidation"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/models"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
	"gorm.io/datatypes"
)

func TestSentryIntakeExecution(t *testing.T) {
	t.Setenv("SUPERPLANE_MAX_PAYLOAD_SIZE", "524288")
	latestStack := `{"eventID":"latest","entries":[{"type":"exception","data":{"values":[{"stacktrace":{"frames":[{"filename":"app.go","function":"Latest","lineNo":22}]}}]}}]}`
	standaloneStack := `{"eventID":"latest","entries":[{"type":"stacktrace","data":{"frames":[{"filename":"app.go","function":"Standalone","lineNo":23}]}}]}`
	recommendedStack := strings.ReplaceAll(latestStack, "Latest", "Recommended")
	requestNodeIDs := []string{intakeSentryIssueRequestNodeID, intakeSentryEventRequestNodeID, intakeSentryRecommendedEventRequestNodeID}
	for _, tc := range []struct {
		name                            string
		oversized                       []string
		status                          int
		latest, recommended             string
		latestStatus, recommendedStatus int
		reject                          bool
		legacy                          bool
		wantRequests                    int
		wantDescription                 string
	}{
		{name: "saved unconditional intake", legacy: true, latest: latestStack, wantRequests: 3, wantDescription: "Latest (app.go:22)"},
		{name: "latest exception stack skips recommended", latest: latestStack, wantRequests: 2, wantDescription: "Latest (app.go:22)"},
		{name: "latest standalone stack skips recommended", latest: standaloneStack, wantRequests: 2, wantDescription: "Standalone (app.go:23)"},
		{name: "stackless latest selects recommended stack", recommended: recommendedStack, wantRequests: 3, wantDescription: "Recommended (app.go:22)"},
		{name: "empty latest stack selects recommended", latest: `{"eventID":"latest","entries":[{"type":"exception","data":{"values":[{"stacktrace":{"frames":[]}}]}}]}`, recommended: recommendedStack, wantRequests: 3, wantDescription: "Recommended (app.go:22)"},
		{name: "recommended with more entries", recommended: `{"eventID":"recommended","entries":[{"type":"message","data":{"formatted":"More details"}}]}`, wantRequests: 3, wantDescription: "More details"},
		{name: "failed latest", latest: `{"detail":"unavailable"}`, latestStatus: 503, recommended: recommendedStack, wantRequests: 3, wantDescription: "Recommended (app.go:22)"},
		{name: "malformed latest", latest: `<html>unavailable</html>`, recommended: recommendedStack, wantRequests: 3, wantDescription: "Recommended (app.go:22)"},
		{name: "missing latest", latest: `null`, recommended: recommendedStack, wantRequests: 3, wantDescription: "Recommended (app.go:22)"},
		{name: "invalid latest event", latest: `{"eventID":"latest","entries":"invalid"}`, recommended: recommendedStack, wantRequests: 3, wantDescription: "Recommended (app.go:22)"},
		{name: "failed recommended keeps latest", recommended: `{"detail":"unavailable"}`, recommendedStatus: 503, wantRequests: 3, wantDescription: "Event details"},
		{name: "rejected filter", reject: true},
		{name: "oversized issue", oversized: requestNodeIDs[:1], wantRequests: 3, wantDescription: "Event details"},
		{name: "oversized latest", oversized: requestNodeIDs[1:2], recommended: recommendedStack, wantRequests: 3, wantDescription: "Recommended (app.go:22)"},
		{name: "oversized recommended", oversized: requestNodeIDs[2:], wantRequests: 3, wantDescription: "Event details"},
		{name: "all oversized", oversized: requestNodeIDs, wantRequests: 3},
		{name: "oversized errors", oversized: requestNodeIDs, status: http.StatusServiceUnavailable, wantRequests: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			db := database.DB(t.Context())
			factory, err := models.CreateFactory(db, r.Organization.ID, "Sentry", "", "")
			require.NoError(t, err)
			template, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceSentryExceptions})
			require.NoError(t, err)
			if tc.legacy {
				useUnconditionalSentryRequests(template)
			}
			if tc.reject {
				findSpecNodeOrNil(template, intakeFilterNodeID).Configuration["expression"] = "false"
			}
			knownNodes := map[string]struct{}{}
			for _, node := range template.Spec.Nodes {
				knownNodes[node.Name] = struct{}{}
			}
			for _, node := range template.Spec.Nodes {
				if node.Component == intakeFilterComponent {
					require.NoError(t, expressionvalidation.ValidateExpression(node.Configuration["expression"].(string), knownNodes))
				}
			}
			nodes := []models.CanvasNode{}
			for _, node := range template.Nodes() {
				nodes = append(nodes, models.CanvasNode{
					NodeID: node.ID, Name: node.Name, Type: node.Type,
					Ref: datatypes.NewJSONType(node.Ref), Configuration: datatypes.NewJSONType(node.Configuration),
				})
			}
			canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nodes, template.Edges())
			canvas.FactoryID = &factory.ID
			require.NoError(t, db.Save(canvas).Error)
			issue := sentry.Issue{ID: "123", Title: "Broken deploy", Permalink: "https://sentry.io/issues/123/"}
			root := support.EmitCanvasEventForNodeWithData(t, canvas.ID, intakeTriggerNodeID, "default", nil,
				map[string]any{"data": sentryIssueEvent(issue, "https://sentry.io/api/0/issues/123/")})
			run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, intakeTriggerNodeID, models.CanvasRunStateStarted, "")
			require.NoError(t, err)
			root.RunID = run.ID
			require.NoError(t, db.Save(root).Error)

			host := &contexts.HTTPContext{}
			for _, nodeID := range requestNodeIDs {
				body := `{"eventID":"event","message":"Event details"}`
				status := http.StatusOK
				switch nodeID {
				case intakeSentryIssueRequestNodeID:
					body = `{"id":"123","title":"Broken deploy"}`
				case intakeSentryEventRequestNodeID:
					if tc.latest != "" {
						body = tc.latest
					}
					if tc.latestStatus != 0 {
						status = tc.latestStatus
					}
				case intakeSentryRecommendedEventRequestNodeID:
					if tc.recommended != "" {
						body = tc.recommended
					}
					if tc.recommendedStatus != 0 {
						status = tc.recommendedStatus
					}
				}
				if slices.Contains(tc.oversized, nodeID) {
					body = `{"extra":{"large":"` + strings.Repeat("x", config.MaxPayloadSize()+1) + `"}}`
					if tc.status != 0 {
						status = tc.status
					}
				}
				host.Responses = append(host.Responses, &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))})
			}

			incoming := root
			nodeID := intakeFilterNodeID
			for nodeID != "" {
				node := findSpecNode(t, template, nodeID)
				action, err := r.Registry.GetAction(node.Component)
				require.NoError(t, err)
				execution := support.CreateNextNodeExecution(t, canvas.ID, nodeID, root.ID, incoming.ID, incoming.ExecutionID)
				execution.RunID = run.ID
				builder := workercontexts.NewNodeConfigurationBuilder(db, canvas.ID).
					WithRunID(run.ID).WithNodeID(nodeID).WithRootEvent(&root.ID).
					WithPreviousExecution(incoming.ExecutionID).WithIncomingEventID(&incoming.ID).
					WithInput(incoming.Data.Data()).WithConfigurationFields(action.Configuration())
				resolved, err := builder.Build(node.Configuration)
				require.NoError(t, err)
				execution.Configuration = datatypes.NewJSONType(resolved)
				require.NoError(t, db.Save(execution).Error)
				require.NoError(t, action.Execute(core.ExecutionContext{
					Configuration: resolved, HTTP: host, Logger: log.NewEntry(log.StandardLogger()),
					Secrets: &contexts.SecretsContext{IntegrationKeys: map[string]map[string][]byte{
						"Connect Sentry": {sentry.IntegrationAuthTokenKey: []byte("test-token")},
					}},
					Metadata:       &contexts.MetadataContext{},
					Expressions:    workercontexts.NewExpressionContext(builder),
					ExecutionState: workercontexts.NewExecutionStateContext(db, execution, nil),
					Factory:        workercontexts.NewFactoryContext(db, canvas, execution),
				}), "execute %s", nodeID)
				var outputs []models.CanvasEvent
				require.NoError(t, db.Where("execution_id = ?", execution.ID).Find(&outputs).Error)
				require.Len(t, outputs, 1)
				incoming = &outputs[0]
				if slices.Contains(tc.oversized, nodeID) {
					assert.Equal(t, "failure", incoming.Channel)
				}
				nodeID = ""
				for _, edge := range template.Spec.Edges {
					if edge.SourceID == incoming.NodeID && edge.Channel == incoming.Channel {
						nodeID = edge.TargetID
						break
					}
				}
			}
			orders, err := models.ListWorkOrdersBySourceRunIDs(db, factory.ID, []uuid.UUID{run.ID})
			require.NoError(t, err)
			require.Len(t, host.Requests, tc.wantRequests)
			for i, request := range host.Requests {
				assert.Equal(t, "GET", request.Method)
				assert.Equal(t, "https://sentry.io/api/0/issues/123/"+[]string{"", "events/latest/", "events/recommended/"}[i], request.URL.String())
			}
			if tc.reject {
				assert.Empty(t, orders)
				return
			}
			require.Len(t, orders, 1)
			assert.Equal(t, issue.Title, orders[run.ID].Title)
			assert.Contains(t, orders[run.ID].Description, issue.Permalink)
			if tc.wantDescription != "" {
				assert.Contains(t, orders[run.ID].Description, tc.wantDescription)
			}
		})
	}
}

// Reproduce the saved HTTP template from before conditional event requests.
func useUnconditionalSentryRequests(canvas *yaml.Canvas) {
	canvas.Spec.Nodes = slices.DeleteFunc(canvas.Spec.Nodes, func(node yaml.Node) bool {
		return node.ID == intakeSentryEventHasStackNodeID
	})
	canvas.Spec.Edges = slices.DeleteFunc(canvas.Spec.Edges, func(edge yaml.Edge) bool {
		return edge.SourceID == intakeSentryEventHasStackNodeID
	})
	for i := range canvas.Spec.Edges {
		if canvas.Spec.Edges[i].TargetID == intakeSentryEventHasStackNodeID {
			canvas.Spec.Edges[i].TargetID = intakeSentryRecommendedEventRequestNodeID
		}
	}
	for i := range canvas.Spec.Nodes {
		if canvas.Spec.Nodes[i].ID == intakeCreateNodeID {
			canvas.Spec.Nodes[i].Configuration["description"] = `{{ sentryIssueDescription(root().data.data.issue, $["Get Sentry Issue"].data?.body, $["Get Sentry Event"].data?.body, $["Get Recommended Sentry Event"].data?.body) }}`
		}
	}
}
