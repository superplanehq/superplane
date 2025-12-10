package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestApprovals(t *testing.T) {
	steps := &ApprovalSteps{t: t}

	t.Run("adding an approval component to a canvas", func(t *testing.T) {
		steps.start()
		steps.givenACanvasExists()
		steps.addApprovalToCanvas("TestApproval")
		steps.saveCanvas()
		steps.verifyApprovalSavedToDB("TestApproval")
	})

	t.Run("configuring approvals for a user", func(t *testing.T) {
		steps.start()
		steps.givenACanvasExists()
		steps.addApprovalToCanvas("ReleaseApproval")
		steps.saveCanvas()
		steps.verifyApprovalConfigurationPersisted()
	})

	t.Run("running and approving on a canvas", func(t *testing.T) {
		steps.start()
		steps.givenCanvasWithManualTriggerApprovalAndNoop()
		steps.runManualTrigger()
		steps.approveFirstPendingRequirement()
		steps.assertApprovalExecutionFinishedAndOutputNodeProcessed()
	})
}

type ApprovalSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *ApprovalSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *ApprovalSteps) givenACanvasExists() {
	s.canvas = shared.NewCanvasSteps("Approval Canvas", s.t, s.session)
	s.canvas.Create()
	s.canvas.Visit()
}

func (s *ApprovalSteps) addApprovalToCanvas(nodeName string) {
	s.canvas.AddApproval(nodeName, models.Position{X: 600, Y: 200})
}

func (s *ApprovalSteps) saveCanvas() {
	s.canvas.Save()
}

func (s *ApprovalSteps) verifyApprovalSavedToDB(nodeName string) {
	node := s.canvas.GetNodeFromDB(nodeName)
	require.NotNil(s.t, node, "approval node not found in DB")
}

func (s *ApprovalSteps) configureApprovalForCurrentUser() {
	s.canvas.StartEditingNode("ReleaseApproval")

	s.session.Click(q.Locator(`button:has-text("Add Item")`))
	s.session.Click(q.Locator(`button:has-text("Select Type")`))
	s.session.Click(q.Locator(`div[role="option"]:has-text("User")`))

	s.session.Click(q.Locator(`button:has-text("Select user")`))
	s.session.Click(q.Locator(`div[role="option"]:has-text("e2e@superplane.local")`))

	s.session.Click(q.TestID("add-node-button"))
	s.session.Sleep(300)
}

func (s *ApprovalSteps) verifyApprovalConfigurationPersisted() {
	node := s.canvas.GetNodeFromDB("ReleaseApproval")
	require.NotNil(s.t, node, "approval node not found in DB")

	data := node.Configuration.Data()
	items := data["items"].([]any)
	require.Len(s.t, items, 1)

	itemCfg, ok := items[0].(map[string]any)
	require.True(s.t, ok, "expected item configuration to be a map")
	require.Equal(s.t, "user", itemCfg["type"])
	require.NotEmpty(s.t, itemCfg["user"])
}

func (s *ApprovalSteps) givenCanvasWithManualTriggerApprovalAndNoop() {
	s.canvas = shared.NewCanvasSteps("Approval Canvas", s.t, s.session)
	s.canvas.Create()
	s.canvas.Visit()

	s.canvas.AddManualTrigger("Start", models.Position{X: 600, Y: 200})
	s.canvas.AddApproval("Approval", models.Position{X: 1000, Y: 200})
	s.canvas.AddNoop("Output", models.Position{X: 1600, Y: 200})

	s.canvas.Connect("Start", "Approval")
	s.canvas.Connect("Approval", "Output")

	s.saveCanvas()
}

func (s *ApprovalSteps) runManualTrigger() {
	s.canvas.RunManualTrigger("Start")
	s.canvas.WaitForExecution("Approval", models.WorkflowNodeExecutionStatePending, 5*time.Second)
}

func (s *ApprovalSteps) approveFirstPendingRequirement() {
	s.session.Click(q.Locator(`button:has-text("Approve")`))
	s.session.FillIn(q.Locator(`input:has-placeholder("Enter comment")`), "Do it")
	s.session.Click(q.Locator(`button:has-text("Confirm Approval")`))
}

func (s *ApprovalSteps) assertApprovalExecutionFinishedAndOutputNodeProcessed() {
	s.canvas.WaitForExecution("Output", models.WorkflowNodeExecutionStateFinished, 10*time.Second)

	approvaExecs := s.canvas.GetExecutionsForNode("Approval")
	outputExecs := s.canvas.GetExecutionsForNode("Output")

	require.Len(s.t, approvaExecs, 1, "expected one execution for approval node")
	require.Len(s.t, outputExecs, 1, "expected one execution for output node")
}
