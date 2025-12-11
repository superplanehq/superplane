package e2e

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/secrets"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
	v "github.com/superplanehq/superplane/test/e2e/vcr"
	"gorm.io/datatypes"
)

func TestGithubTrigger(t *testing.T) {
	steps := &GithubTriggerSteps{t: t}

	org := "https://github.com/puppies-inc"
	token := "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"

	v.Run(t, "addding a github trigger node", func(t *testing.T) {
		steps.start()
		steps.givenACanvasExists()
		steps.givenAGithubIntegrationExists("Integration", org, token)
		steps.addGithubTriggerNode()
		steps.saveCanvas()
		steps.assertGithubTriggerNodeExistsInDB()
	})

	v.Run(t, "receiving github trigger events", func(t *testing.T) {
		steps.start()
		steps.givenACanvasWithGithubTriggerAndNoop()
		steps.saveCanvas()
		steps.simulateReceivingGithubEvent()
		steps.assertGithubTriggerExecutionCreated()
		steps.assertSecondNodeExecuted()
	})
}

type GithubTriggerSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *GithubTriggerSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *GithubTriggerSteps) givenAGithubIntegrationExists(name, url, token string) {
	secretValues := map[string]string{"value": token}

	secretData, err := json.Marshal(secretValues)
	require.NoError(s.t, err)

	_, err = models.CreateSecret(
		"github-token",
		secrets.ProviderLocal,
		s.session.Account.ID.String(),
		models.DomainTypeOrganization,
		s.session.OrgID,
		secretData,
	)
	require.NoError(s.t, err)

	auth := models.IntegrationAuth{
		Token: &models.IntegrationAuthToken{
			ValueFrom: models.ValueDefinitionFrom{
				Secret: &models.ValueDefinitionFromSecret{
					Name: "github-token",
					Key:  "value",
				},
			},
		},
	}

	now := time.Now()
	integration := &models.Integration{
		ID:         uuid.New(),
		Name:       name,
		DomainType: models.DomainTypeOrganization,
		DomainID:   s.session.OrgID,
		Type:       models.IntegrationTypeGithub,
		URL:        url,
		AuthType:   models.IntegrationAuthTypeToken,
		Auth:       datatypes.NewJSONType(auth),
		CreatedAt:  &now,
		CreatedBy:  s.session.Account.ID,
	}

	_, err = models.CreateIntegration(integration)
	require.NoError(s.t, err)
}

func (s *GithubTriggerSteps) givenACanvasExists() {
	s.canvas = shared.NewCanvasSteps("E2E Github Trigger", s.t, s.session)
	s.canvas.Create()
	s.canvas.Visit()
}

func (s *GithubTriggerSteps) addGithubTriggerNode() {
	source := q.TestID("building-block-github")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, 500, 200)
	s.session.Sleep(300)

	s.session.FillIn(q.TestID("node-name-input"), "GitHub Trigger")

	// Select the GitHub integration
	integrationTrigger := q.Locator(`label:has-text("GitHub integration") + div button`)
	s.session.Click(integrationTrigger)
	s.session.Click(q.Locator(`div[role="option"]:has-text("Integration")`))

	s.session.Click(q.TestID("add-node-button"))
	s.session.Sleep(300)
}

func (s *GithubTriggerSteps) saveCanvas() {
	s.canvas.Save()
}

func (s *GithubTriggerSteps) assertGithubTriggerNodeExistsInDB() {
	node := s.canvas.GetNodeFromDB("GitHub Trigger")
	require.NotNil(s.t, node, "github trigger node not found in DB")
}

func (s *GithubTriggerSteps) givenACanvasWithGithubTriggerAndNoop() {
	s.givenACanvasExists()

	s.addGithubTriggerNode()

	s.canvas.AddNoop("Second", models.Position{X: 900, Y: 200})
	s.canvas.Connect("GitHub Trigger", "Second")
}

func (s *GithubTriggerSteps) simulateReceivingGithubEvent() {
	node := s.canvas.GetNodeFromDB("GitHub Trigger")
	require.NotNil(s.t, node, "github trigger node not found in DB")

	// Insert a workflow event that looks like a GitHub webhook payload
	event := &models.WorkflowEvent{
		ID:         uuid.New(),
		WorkflowID: s.canvas.WorkflowID,
		NodeID:     node.NodeID,
		Channel:    "github-webhook",
		Data: datatypes.NewJSONType[any](map[string]any{
			"head_commit": map[string]any{
				"message": "Initial commit",
				"id":      "abc123",
				"author": map[string]any{
					"name": "E2E Tester",
				},
			},
		}),
		State: models.WorkflowEventStatePending,
	}

	err := database.Conn().Create(event).Error
	require.NoError(s.t, err)

	// Give the workflow_event_router and node executor some time to process the event
	time.Sleep(3 * time.Second)
}

func (s *GithubTriggerSteps) assertGithubTriggerExecutionCreated() {
	executions := s.canvas.GetExecutionsForNode("GitHub Trigger")
	require.NotEmpty(s.t, executions, "expected at least one execution for github trigger node")
}

func (s *GithubTriggerSteps) assertSecondNodeExecuted() {
	executions := s.canvas.GetExecutionsForNode("Second")
	require.NotEmpty(s.t, executions, "expected at least one execution for second node")
}
