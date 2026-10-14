package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestApprovalConfiguration(t *testing.T) {
	t.Run("adding an approval component to a canvas", func(t *testing.T) {
		steps := &approvalConfigurationSteps{t: t}
		steps.start()
		steps.givenACanvasExists()
		steps.addApprovalToCanvas("TestApproval")
		steps.saveCanvas()
		steps.canvas.CommitAndPublish()
		steps.verifyApprovalSavedToDB("TestApproval")
	})

	t.Run("configuring approvals for a user role and group", func(t *testing.T) {
		steps := &approvalConfigurationSteps{t: t}
		steps.start()
		steps.givenACanvasExists()
		groupName := createOrgApprovalGroup(t, steps.session.OrgID)
		steps.addApprovalWithUserRoleGroup("ReleaseApproval", models.Position{X: 600, Y: 200}, models.DisplayNameAdmin, groupName)
		steps.saveCanvas()
		steps.canvas.CommitAndPublish()
		steps.verifyApprovalConfigurationPersisted(models.RoleOrgAdmin, groupName)
	})
}

type approvalConfigurationSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *approvalConfigurationSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *approvalConfigurationSteps) givenACanvasExists() {
	s.canvas = shared.NewCanvasSteps("Approval Canvas", s.t, s.session)
	s.canvas.Create()
	s.canvas.Visit()
	s.canvas.EnterEditMode()
}

func (s *approvalConfigurationSteps) addApprovalToCanvas(nodeName string) {
	s.canvas.AddApproval(nodeName, models.Position{X: 600, Y: 200})
}

func (s *approvalConfigurationSteps) saveCanvas() {
	s.canvas.Save()
}

func (s *approvalConfigurationSteps) verifyApprovalSavedToDB(nodeName string) {
	node := s.canvas.GetNodeFromDB(nodeName)
	require.NotNil(s.t, node, "approval node not found in DB")
}

func (s *approvalConfigurationSteps) verifyApprovalConfigurationPersisted(expectedRole string, expectedGroup string) {
	node := s.canvas.GetNodeFromDB("ReleaseApproval")
	require.NotNil(s.t, node, "approval node not found in DB")

	data := node.Configuration.Data()
	items := data["items"].([]any)
	require.Len(s.t, items, 3)

	var userItem map[string]any
	var roleItem map[string]any
	var groupItem map[string]any

	for _, rawItem := range items {
		itemCfg, ok := rawItem.(map[string]any)
		require.True(s.t, ok, "expected item configuration to be a map")

		itemType, _ := itemCfg["type"].(string)
		switch itemType {
		case "user":
			userItem = itemCfg
		case "role":
			roleItem = itemCfg
		case "group":
			groupItem = itemCfg
		}
	}

	require.NotNil(s.t, userItem, "expected user approver configuration")
	require.NotNil(s.t, roleItem, "expected role approver configuration")
	require.NotNil(s.t, groupItem, "expected group approver configuration")
	require.NotEmpty(s.t, userItem["user"])
	require.Equal(s.t, expectedRole, roleItem["role"])
	require.Equal(s.t, expectedGroup, groupItem["group"])
}

func (s *approvalConfigurationSteps) addApprovalWithUserRoleGroup(nodeName string, pos models.Position, roleLabel string, groupLabel string) {
	s.canvas.AddBuildingBlockByTestID("building-block-approval", pos)
	s.session.Sleep(300)

	s.session.FillIn(q.TestID("node-name-input"), nodeName)
	s.session.Click(q.Locator(`button:has-text("Add Approver")`))
	s.session.Sleep(400)
	s.session.Click(q.Locator(`button:has-text("Add Approver")`))
	s.session.Sleep(400)

	typeSelects := s.session.Page().Locator(`[data-testid="field-type-select"]`)

	// Set type and value per approver so autosave does not persist type=user with an empty user field.
	if err := typeSelects.Nth(0).Click(); err != nil {
		s.t.Fatalf("clicking first approver type select: %v", err)
	}
	s.session.Click(q.Locator(`div[role="option"]:has-text("Specific user")`))
	s.session.Sleep(200)
	userSelect := s.session.Page().Locator(`button:has-text("Select user")`).First()
	if err := userSelect.Click(); err != nil {
		s.t.Fatalf("opening user select: %v", err)
	}
	s.session.Click(q.Locator(`div[role="option"]:has-text("` + s.session.Account.Email + `")`))

	if err := typeSelects.Nth(1).Click(); err != nil {
		s.t.Fatalf("clicking second approver type select: %v", err)
	}
	s.session.Click(q.Locator(`div[role="option"]:has-text("Role")`))
	s.session.Sleep(200)
	s.session.Click(q.Locator(`button:has-text("Select role")`))
	s.session.Click(q.Locator(`div[role="option"]:has-text("` + roleLabel + `")`))

	if err := typeSelects.Nth(2).Click(); err != nil {
		s.t.Fatalf("clicking third approver type select: %v", err)
	}
	s.session.Click(q.Locator(`div[role="option"]:has-text("Group")`))
	s.session.Sleep(200)
	s.session.Click(q.Locator(`button:has-text("Select group")`))
	s.session.Click(q.Locator(`div[role="option"]:has-text("` + groupLabel + `")`))

	// Configuration sidebar autosaves on a 1200ms safety-net timer for scripted flows.
	s.session.Sleep(1500)
}
