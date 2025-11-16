package e2e

import (
	"fmt"
	"testing"

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
		steps.configureApprovalForCurrentUser()
		steps.saveCanvas()
		steps.verifyApprovalConfigurationPersisted()
	})

	// t.Run("running and approving on a canvas", func(t *testing.T) {
	// 	steps.start()
	// 	steps.givenCanvasWithManualTriggerApprovalAndNoop()
	// 	steps.runManualTrigger()
	// 	steps.openApprovalRunFromSidebar()
	// 	steps.approveFirstPendingRequirement()
	// 	steps.assertApprovalExecutionFinishedAndOutputNodeProcessed()
	// })
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

	fmt.Println(node.Configuration.Data())

	data := node.Configuration.Data()
	items := data["items"].([]any)
	require.Len(s.t, items, 1)

	itemCfg, ok := items[0].(map[string]any)
	require.True(s.t, ok, "expected item configuration to be a map")
	require.Equal(s.t, "user", itemCfg["type"])
	require.NotEmpty(s.t, itemCfg["user"])
}

// func (s *ApprovalSteps) givenCanvasWithManualTriggerApprovalAndNoop() {
// 	s.canvas = shared.NewCanvasSteps("Approval Canvas", s.t, s.session)
// 	s.canvas.Create()
// 	s.canvas.Visit()

// 	s.canvas.AddManualTrigger("Start", models.Position{X: 400, Y: 200})
// 	s.canvas.AddApproval("Approval", models.Position{X: 800, Y: 200})
// 	s.canvas.AddNoop("Output", models.Position{X: 1200, Y: 200})

// 	// Configure approval for current user
// 	s.session.Click(q.TestID("node", "approval", "header"))
// 	s.session.Click(q.TestID("edit-node-button"))

// 	itemTypeSelect := q.Locator(`button:has-text("Select Type")`)
// 	s.session.Click(itemTypeSelect)
// 	s.session.Click(q.Locator(`div[role="option"]:has-text("User")`))

// 	userFieldTrigger := q.Locator(`button:has-text("Select user")`)
// 	s.session.Click(userFieldTrigger)
// 	s.session.Click(q.Locator(`div[role="option"]:has-text("e2e@superplane.local")`))

// 	s.session.Click(q.TestID("add-node-button"))
// 	s.session.Sleep(300)

// 	// Connect Start -> Approval -> Output
// 	s.canvas.Connect("start", "approval")
// 	s.canvas.Connect("approval", "output")

// 	s.saveCanvas()
// }

// func (s *ApprovalSteps) runManualTrigger() {
// 	dropdown := q.TestID("node-start-header-dropdown")
// 	runOption := q.Locator("button:has-text('Run')")

// 	s.session.Click(dropdown)
// 	s.session.Click(runOption)
// 	s.session.Click(q.TestID("emit-event-submit-button"))
// 	s.session.Sleep(1000)
// }

// func (s *ApprovalSteps) openApprovalRunFromSidebar() {
// 	s.session.Sleep(500)

// 	// Click Approval node header to open sidebar, then expand run details
// 	s.session.Click(q.TestID("node", "approval", "header"))
// 	s.session.Sleep(300)
// 	s.session.Click(q.TestID("expand-run-button"))
// 	s.session.Sleep(500)
// }

// func (s *ApprovalSteps) approveFirstPendingRequirement() {
// 	// Click the first "Approve" button in the approval list
// 	s.session.Click(q.Locator(`button:has-text("Approve")`))
// 	s.session.Sleep(200)
// 	// Confirm approval (in the inner dialog)
// 	s.session.Click(q.Locator(`button:has-text("Confirm Approval")`))
// 	s.session.Sleep(1000)
// }

// func (s *ApprovalSteps) assertApprovalExecutionFinishedAndOutputNodeProcessed() {
// wf, err := models.FindWorkflow(s.session.OrgID, s.workflowID)
// require.NoError(s.t, err)

// nodes, err := models.FindWorkflowNodes(wf.ID)
// require.NoError(s.t, err)

// var outputNode *models.WorkflowNode
// for _, n := range nodes {
// 	if n.Name == "Output" {
// 		outputNode = &n
// 		break
// 	}
// }
// require.NotNil(s.t, outputNode, "output node not found")

// var executions []models.WorkflowNodeExecution
// err = models.ListWorkflowNodeExecutionsByNodeID(wf.ID, outputNode.NodeID, &executions)
// require.NoError(s.t, err)
// require.NotEmpty(s.t, executions, "expected at least one execution for output node")
// }
