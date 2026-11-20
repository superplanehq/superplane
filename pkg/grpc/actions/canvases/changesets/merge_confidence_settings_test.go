package changesets

import (
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
)

func TestCanvasPublisherMergeConfidencePublication(t *testing.T) {
	for _, tc := range []struct {
		name             string
		settings         map[string]any
		bodyEnabled      bool
		commentAvailable bool
		expectedError    string
		checkEnabled     *bool
		expectedCode     codes.Code
	}{
		{name: "cannot enable a check without GitHub even with results off", settings: map[string]any{"publishResults": false, "publishPRCheck": true}, checkEnabled: github.Ptr(true), expectedError: "Connect GitHub", expectedCode: codes.FailedPrecondition},
		{name: "disabled checks need no GitHub permissions", settings: map[string]any{"publishResults": false, "publishPRCheck": false}, checkEnabled: github.Ptr(false)},
		{
			name:        "older automation defaults to body publication",
			bodyEnabled: true,
		},
		{
			name:        "older automation can keep publication disabled",
			settings:    map[string]any{"resultsBody": false},
			bodyEnabled: false,
		},
		{
			name:        "disabled results need no destination",
			settings:    map[string]any{"publishResults": false, "resultsBody": false, "resultsComment": true},
			bodyEnabled: true,
		},
		{
			name:             "comment is a valid destination without body publication",
			settings:         map[string]any{"publishResults": true, "resultsBody": false, "resultsComment": true},
			commentAvailable: true,
		},
		{
			name:          "enabled results require a destination",
			settings:      map[string]any{"publishResults": true, "resultsBody": false},
			expectedError: "Select at least one destination",
		},
		{
			name:          "legacy enabled results require a destination",
			settings:      map[string]any{"resultsBody": false},
			bodyEnabled:   true,
			expectedError: "Select at least one destination",
		},
		{
			name:          "comment publication requires a managed comment node",
			settings:      map[string]any{"resultsComment": true},
			bodyEnabled:   true,
			expectedError: "PR comment publication is unavailable",
		},
		{
			name:          "explicit results cannot enable an unavailable comment",
			settings:      map[string]any{"publishResults": true, "resultsComment": true},
			expectedError: "PR comment publication is unavailable",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			// No-op components isolate graph validation from provider setup.
			agent := componentCanvasNode("assess-risk", "Assess risk", "noop", nil)
			agent.Metadata = datatypes.NewJSONType(models.FactoryAppTemplateMetadata("risk-score", 1))
			nodes := []models.CanvasNode{
				agent,
				componentCanvasNode("publish-merge-confidence", "Publish results", "noop", map[string]any{"enabled": tc.bodyEnabled}),
			}
			if tc.checkEnabled != nil {
				nodes = append(nodes, componentCanvasNode("start-assessment-check", "Start assessment check", "github.setPullRequestCheckRun", map[string]any{"enabled": false}))
			}
			if tc.commentAvailable {
				nodes = append(nodes, componentCanvasNode("comment-merge-confidence", "Publish comment", "github.publishPullRequestComment", nil))
			}
			canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nodes, nil)
			canvas.FactoryID = &factory.ID
			require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
			live, err := models.FindLiveCanvasVersionInTransaction(db, canvas.ID)
			require.NoError(t, err)
			draftNodes := append([]models.Node(nil), live.Nodes...)
			for i := range draftNodes {
				if draftNodes[i].ID == "start-assessment-check" {
					draftNodes[i].Configuration = map[string]any{"enabled": *tc.checkEnabled}
				}
				if draftNodes[i].ID == "assess-risk" {
					draftNodes[i].Name = "Updated assessment"
					draftNodes[i].Configuration = tc.settings
				}
			}
			draft, err := models.CreateCommitVersionWithSpecInTransaction(db, canvas.ID, r.User, "Update publication settings", draftNodes, nil)
			require.NoError(t, err)
			publisher, err := NewCanvasPublisher(db, canvas, draft, live, canvasPublisherOptions(r))
			require.NoError(t, err)

			err = publisher.Publish(t.Context())
			if tc.expectedError != "" {
				expectedCode := tc.expectedCode
				if expectedCode == codes.OK {
					expectedCode = codes.InvalidArgument
				}
				require.Equal(t, expectedCode, grpcerrors.Code(err))
				require.ErrorContains(t, err, tc.expectedError)
				unchanged, err := models.FindLiveCanvasVersionInTransaction(db, canvas.ID)
				require.NoError(t, err)
				require.Equal(t, live.ID, unchanged.ID)
				stored, err := models.FindCanvasNode(db, canvas.ID, "assess-risk")
				require.NoError(t, err)
				require.Equal(t, "Assess risk", stored.Name)
				return
			}
			require.NoError(t, err)
			published, err := models.FindLiveCanvasVersionInTransaction(db, canvas.ID)
			require.NoError(t, err)
			require.Equal(t, draft.ID, published.ID)
			require.Equal(t, tc.settings, findVersionNode(t, published.Nodes, "assess-risk").Configuration)
		})
	}
}
