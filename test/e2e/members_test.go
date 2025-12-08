package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/utils"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestMembersInvitations(t *testing.T) {
	steps := &membersSteps{t: t}

	t.Run("inviting a new organization member", func(t *testing.T) {
		email := "e2e-member-invite@example.com"
		steps.start()
		steps.visitMembersPage()
		steps.fillInviteEmailTextarea(email)
		steps.submitInvitations()
		steps.assertInvitationPersistedInDB(email)
	})

	t.Run("viewing existing organization members", func(t *testing.T) {
		steps.start()
		steps.visitMembersPage()
		steps.assertMembersHeaderVisible()
		steps.assertTabCounts("All (1)", "Active (1)", "Invited (0)")
		steps.assertMemberVisible("e2e@superplane.local")
	})

	t.Run("viewing invited members in the organization", func(t *testing.T) {
		email := "e2e-members-view@example.com"

		steps.start()
		steps.visitMembersPage()
		steps.fillInviteEmailTextarea(email)
		steps.submitInvitations()
		steps.session.Sleep(2000)

		steps.assertTabCounts("All (2)", "Active (1)", "Invited (1)")

		steps.switchToTab("Invited (1)")
		steps.assertMemberVisible(email)
	})
}

type membersSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *membersSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *membersSteps) visitMembersPage() {
	s.session.Visit("/" + s.session.OrgID.String() + "/settings/members")
}

func (s *membersSteps) fillInviteEmailTextarea(email string) {
	textarea := q.Locator(`textarea[placeholder="Email addresses, separated by commas"]`)
	s.session.FillIn(textarea, email)
	s.session.Sleep(300)
}

func (s *membersSteps) submitInvitations() {
	button := q.Locator(`button:has-text("Send Invitations")`)
	s.session.Click(button)
	s.session.Sleep(1500)
}

func (s *membersSteps) assertInvitationPersistedInDB(email string) {
	var invitation models.OrganizationInvitation
	normalizedEmail := utils.NormalizeEmail(email)
	err := database.Conn().Where("email = ? AND organization_id = ?", normalizedEmail, s.session.OrgID.String()).First(&invitation).Error
	require.NoError(s.t, err)
	require.Equal(s.t, normalizedEmail, invitation.Email)
	require.Equal(s.t, s.session.OrgID.String(), invitation.OrganizationID.String())
}

func (s *membersSteps) assertMembersHeaderVisible() {
	s.session.AssertText("Members")
}

func (s *membersSteps) assertTabCounts(allLabel, activeLabel, invitedLabel string) {
	s.session.AssertText(allLabel)
	s.session.AssertText(activeLabel)
	s.session.AssertText(invitedLabel)
}

func (s *membersSteps) assertMemberVisible(email string) {
	s.session.AssertText(email)
}

func (s *membersSteps) switchToTab(label string) {
	s.session.Click(q.Text(label))
	s.session.Sleep(500)
}
