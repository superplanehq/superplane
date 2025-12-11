package e2e

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
	"gorm.io/datatypes"

	q "github.com/superplanehq/superplane/test/e2e/queries"
	v "github.com/superplanehq/superplane/test/e2e/vcr"
)

func TestGithubTrigger(t *testing.T) {
	steps := &GithubTriggerSteps{t: t}

	const githubOwner = "puppies-inc"
	const githubTokenValue = "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"

	// v.Run(t, "addding a github trigger node", func(t *testing.T) {
	// 	steps.start()
	// 	steps.givenAGithubIntegrationExists(githubOwner, githubTokenValue)
	// 	steps.givenACanvasExists()
	// 	steps.addGithubTriggerNode()
	// 	steps.saveCanvas()
	// 	steps.assertGithubTriggerNodeExistsInDB()
	// })

	v.Run(t, "receiving github trigger events", func(t *testing.T) {
		steps.start()
		steps.givenAGithubIntegrationExists(githubOwner, githubTokenValue)
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

	// integrationName is the name as shown in the UI dropdown
	integrationName string
}

func (s *GithubTriggerSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *GithubTriggerSteps) givenAGithubIntegrationExists(ownerSlug, token string) {
	// Use the UI flow to create the integration, as in integrations_test.go
	ownerInput := q.Locator(`input[data-testid="github-owner-input"]`)
	tokenInput := q.Locator(`input[data-testid="integration-api-token-input"]`)

	s.session.Visit("/" + s.session.OrgID.String() + "/settings/integrations")
	s.session.AssertText("Integrations")

	s.session.Click(q.Text("Add Integration"))
	s.session.AssertText("Select Integration Type")

	s.session.Click(q.Locator(`button:has-text("GitHub")`))

	s.session.FillIn(ownerInput, ownerSlug)
	s.session.FillIn(tokenInput, token)

	s.session.Click(q.TestID("create-integration-button"))

	s.integrationName = ownerSlug + "-account"

	s.session.Sleep(1000)
}

func (s *GithubTriggerSteps) givenACanvasExists() {
	s.canvas = shared.NewCanvasSteps("E2E Github Trigger", s.t, s.session)
	s.canvas.Create()
	s.canvas.Visit()
}

func (s *GithubTriggerSteps) addGithubTriggerNode() {
	source := q.TestID("building-block-github")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, 800, 200)
	s.session.Sleep(300)

	s.session.FillIn(q.TestID("node-name-input"), "GitHub Trigger")

	// Select the GitHub integration
	integrationTrigger := q.Locator(`label:has-text("GitHub integration") + div button`)
	s.session.Click(integrationTrigger)
	s.session.Click(q.Locator(`div[role="option"]:has-text("` + s.integrationName + `")`))

	// Select the repository
	repositoryTrigger := q.Locator(`label:has-text("Repository") + div button`)
	s.session.Click(repositoryTrigger)
	s.session.Click(q.Locator(`div[role="option"]:has-text("front")`))

	// Select the event type
	eventTypeTrigger := q.Locator(`label:has-text("Event Type") + div button`)
	s.session.Click(eventTypeTrigger)
	s.session.Click(q.Locator(`div[role="option"]:has-text("Push")`))

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
	s.canvas.WaitForExecution("GitHub Trigger", models.WorkflowNodeExecutionStateFinished, 10*time.Second)

	executions := s.canvas.GetExecutionsForNode("GitHub Trigger")
	require.NotEmpty(s.t, executions, "expected at least one execution for github trigger node")
}

func (s *GithubTriggerSteps) assertSecondNodeExecuted() {
	executions := s.canvas.GetExecutionsForNode("Second")
	require.NotEmpty(s.t, executions, "expected at least one execution for second node")
}
