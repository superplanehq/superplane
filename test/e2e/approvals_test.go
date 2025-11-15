package e2e

import (
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestApprovals(t *testing.T) {
	steps := &ApprovalSteps{t: t}

	t.Run("adding an approval component to a canvas", func(t *testing.T) {
		steps.Start()
		steps.GivenACanvasExists()
		steps.VisitCanvasPage()
		steps.AddApprovalToCanvas("Test Approval")
		steps.SaveCanvas()
		steps.VerifyApprovalSavedToDB("Test Approval")
	})
}

type ApprovalSteps struct {
	t          *testing.T
	session    *session.TestSession
	canvasName string
	workflowID uuid.UUID
}

func (s *ApprovalSteps) Start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *ApprovalSteps) GivenACanvasExists() {
	s.canvasName = "E2E Canvas"

	s.session.VisitHomePage()
	s.session.Click(q.Text("New Canvas"))
	s.session.FillIn(q.TestID("canvas-name-input"), s.canvasName)
	s.session.Click(q.Text("Create canvas"))
	s.session.Sleep(300)

	wf, err := models.FindWorkflowByName(s.canvasName, s.session.OrgID)
	require.NoError(s.t, err)
	s.workflowID = wf.ID
}

func (s *ApprovalSteps) VisitCanvasPage() {
	s.session.Visit("/" + s.session.OrgID.String() + "/workflows/" + s.workflowID.String())
}

func (s *ApprovalSteps) AddApprovalToCanvas(nodeName string) {
	source := q.TestID("building-block-approval")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, 500, 250)
	s.session.Sleep(300)

	if nodeName == "" {
		nodeName = "approval"
	}

	s.session.FillIn(q.TestID("node-name-input"), nodeName)
	s.session.Click(q.TestID("add-node-button"))
	s.session.Sleep(300)
}

func (s *ApprovalSteps) SaveCanvas() {
	s.session.Click(q.TestID("save-canvas-button"))
	s.session.Sleep(500)
	s.session.AssertText("Canvas changes saved")
}

func (s *ApprovalSteps) VerifyApprovalSavedToDB(nodeName string) {
	wf, err := models.FindWorkflow(s.session.OrgID, s.workflowID)
	require.NoError(s.t, err)

	nodes, err := models.FindWorkflowNodes(wf.ID)
	require.NoError(s.t, err)

	for _, n := range nodes {
		if n.Name == nodeName {
			return
		}
	}

	s.t.Fatalf("expected approval node %q to be saved in DB, but it was not found", nodeName)
}
