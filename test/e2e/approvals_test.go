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

// TODO: Add e2e tests for role and group approval when RBAC is enabled
func TestApprovals(t *testing.T) {
	t.Run("adding an approval component to a canvas", func(t *testing.T) {
		steps := &ApprovalSteps{t: t}
		steps.start()
		steps.givenACanvasExists()
		steps.addApprovalToCanvas("TestApproval")
		steps.verifyApprovalSavedToDB("TestApproval")
	})

	t.Run("configuring approvals for a user", func(t *testing.T) {
		steps := &ApprovalSteps{t: t}
		steps.start()
		steps.givenACanvasExists()
		steps.addApprovalToCanvas("ReleaseApproval")
		steps.verifyApprovalConfigurationPersisted()
	})

	t.Run("running and approving on a canvas", func(t *testing.T) {
		steps := &ApprovalSteps{t: t}
		steps.start()
		steps.givenCanvasWithManualTriggerApprovalAndNoop()
		steps.runManualTrigger()
		steps.approveFirstPendingRequirement()
		steps.assertApprovalExecutionFinishedAndOutputNodeProcessed()
	})

	t.Run("preventing duplicate approvals across approver types", func(t *testing.T) {
		steps := &ApprovalSteps{t: t}
		steps.start()
		steps.givenCanvasWithManualTriggerAnyoneAndUserApprovalAndNoop()
		steps.runManualTrigger()
		steps.approveAnyoneRequirement()
		steps.waitForApprovalMetadata("Approval", 1, 1, "user")
		steps.assertNoApproveButtons()
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

// TODO: Update to test role and group configuration when RBAC is enabled
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

func (s *ApprovalSteps) givenCanvasWithManualTriggerAnyoneAndUserApprovalAndNoop() {
	s.canvas = shared.NewCanvasSteps("Approval Canvas", s.t, s.session)
	s.canvas.Create()
	s.canvas.Visit()

	s.canvas.AddManualTrigger("Start", models.Position{X: 600, Y: 200})
	s.addApprovalWithAnyAndSpecificUser("Approval", models.Position{X: 1000, Y: 200})
	s.canvas.AddNoop("Output", models.Position{X: 1600, Y: 200})

	s.canvas.Connect("Start", "Approval")
	s.canvas.Connect("Approval", "Output")

	s.saveCanvas()
}

func (s *ApprovalSteps) addApprovalWithAnyAndSpecificUser(nodeName string, pos models.Position) {
	s.canvas.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-approval")
	target := q.TestID("rf__wrapper")

	s.session.DragAndDrop(source, target, pos.X, pos.Y)
	s.session.Sleep(300)

	s.session.FillIn(q.TestID("node-name-input"), nodeName)
	s.session.Click(q.Locator(`button:has-text("Add Approver")`))
	s.session.Sleep(200)

	typeSelects := s.session.Page().Locator(`[data-testid="field-type-select"]`)
	s.session.Click(q.Locator(`[data-testid="field-type-select"]`))
	s.session.Click(q.Locator(`div[role="option"]:has-text("Any user")`))

	if err := typeSelects.Nth(1).Click(); err != nil {
		s.t.Fatalf("clicking second approver type select: %v", err)
	}
	s.session.Click(q.Locator(`div[role="option"]:has-text("Specific user")`))

	userSelect := s.session.Page().Locator(`button:has-text("Select user")`).First()
	if err := userSelect.Click(); err != nil {
		s.t.Fatalf("opening user select: %v", err)
	}
	s.session.Click(q.Locator(`div[role="option"]:has-text("e2e@superplane.local")`))

	s.session.Click(q.TestID("save-node-button"))
	s.session.Sleep(500)
}

func (s *ApprovalSteps) runManualTrigger() {
	s.canvas.RunManualTrigger("Start")
	s.canvas.WaitForExecution("Approval", models.WorkflowNodeExecutionStateStarted, 5*time.Second)
}

func (s *ApprovalSteps) approveFirstPendingRequirement() {
	s.session.Click(q.Locator(`button:has-text("Approve")`))
	s.session.FillIn(q.Locator(`input:has-placeholder("Enter comment")`), "Do it")
	s.session.Click(q.Locator(`button:has-text("Confirm Approval")`))
}

func (s *ApprovalSteps) approveAnyoneRequirement() {
	item := s.session.Page().Locator(`[data-slot="item"]:has([data-slot="item-title"]:has-text("Any user"))`)
	approveButton := item.Locator(`button:has-text("Approve")`).First()
	if err := approveButton.Click(); err != nil {
		s.t.Fatalf("clicking approve button for any user: %v", err)
	}
	s.session.FillIn(q.Locator(`input:has-placeholder("Enter comment")`), "Do it")
	s.session.Click(q.Locator(`button:has-text("Confirm Approval")`))
}

func (s *ApprovalSteps) waitForApprovalMetadata(nodeName string, approvedCount int, pendingCount int, approvedType string) {
	found := false
	start := time.Now()

	for time.Since(start) < 5*time.Second {
		executions := s.canvas.GetExecutionsForNode(nodeName)
		if len(executions) == 0 {
			s.session.Sleep(500)
			continue
		}

		metadata := executions[0].Metadata.Data()
		rawRecords, ok := metadata["records"].([]any)
		require.True(s.t, ok, "expected approval records metadata")

		approved := 0
		pending := 0
		approvedTypeMatch := false
		for _, rawRecord := range rawRecords {
			record, ok := rawRecord.(map[string]any)
			require.True(s.t, ok, "expected approval record metadata")
			state, _ := record["state"].(string)
			recordType, _ := record["type"].(string)
			switch state {
			case "approved":
				approved++
				if recordType == approvedType {
					approvedTypeMatch = true
				}
			case "pending":
				pending++
			}
		}

		if approved == approvedCount && pending == pendingCount && approvedTypeMatch {
			found = true
			break
		}

		s.session.Sleep(500)
	}

	require.True(s.t, found, "timed out waiting for approval metadata to update")
}

func (s *ApprovalSteps) assertNoApproveButtons() {
	approveButtons := s.session.Page().Locator(`button:has-text("Approve")`)
	count, err := approveButtons.Count()
	require.NoError(s.t, err)
	require.Equal(s.t, 0, count, "expected no approve buttons for current user")
}

func (s *ApprovalSteps) assertApprovalExecutionFinishedAndOutputNodeProcessed() {
	s.canvas.WaitForExecution("Output", models.WorkflowNodeExecutionStateFinished, 10*time.Second)

	approvaExecs := s.canvas.GetExecutionsForNode("Approval")
	outputExecs := s.canvas.GetExecutionsForNode("Output")

	require.Len(s.t, approvaExecs, 1, "expected one execution for approval node")
	require.Len(s.t, outputExecs, 1, "expected one execution for output node")
}
