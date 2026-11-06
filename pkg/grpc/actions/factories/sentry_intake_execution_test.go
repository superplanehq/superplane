package factories

import (
	"encoding/json"
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
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/models"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
	"gorm.io/datatypes"
)

func TestSentryIntakeCreatesTaskAfterOversizedResponse(t *testing.T) {
	t.Setenv("SUPERPLANE_MAX_PAYLOAD_SIZE", "524288")
	requestNodeIDs := []string{intakeSentryIssueRequestNodeID, intakeSentryEventRequestNodeID, intakeSentryRecommendedEventRequestNodeID}
	for _, tc := range []struct {
		name      string
		oversized []string
		status    int
	}{
		{"issue", requestNodeIDs[:1], http.StatusOK},
		{"latest event", requestNodeIDs[1:2], http.StatusOK},
		{"recommended event", requestNodeIDs[2:], http.StatusOK},
		{"all responses", requestNodeIDs, http.StatusOK},
		{"error responses", requestNodeIDs, http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			db := database.DB(t.Context())
			factory, err := models.CreateFactory(db, r.Organization.ID, "Sentry", "", "")
			require.NoError(t, err)
			template, err := buildIntakeCanvas(intakeCanvasRequest{Source: models.FactoryIntakeSourceSentryExceptions})
			require.NoError(t, err)
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
				body := map[string]any{"eventID": "event", "message": "Event details"}
				if nodeID == intakeSentryIssueRequestNodeID {
					body = map[string]any{"id": issue.ID, "title": issue.Title}
				}
				status := http.StatusOK
				if slices.Contains(tc.oversized, nodeID) {
					body["extra"] = map[string]any{"large": strings.Repeat("x", config.MaxPayloadSize()+1)}
					status = tc.status
				}
				encoded, err := json.Marshal(body)
				require.NoError(t, err)
				host.Responses = append(host.Responses, &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(string(encoded)))})
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
			require.Len(t, orders, 1)
			assert.Equal(t, issue.Title, orders[run.ID].Title)
			assert.Contains(t, orders[run.ID].Description, issue.Permalink)
			if len(tc.oversized) == 1 {
				assert.Contains(t, orders[run.ID].Description, "Event details")
			}
		})
	}
}
