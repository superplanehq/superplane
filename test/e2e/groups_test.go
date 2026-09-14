package e2e

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestGroups(t *testing.T) {
	t.Run("creating a new group", func(t *testing.T) {
		steps := &GroupsSteps{t: t}
		steps.start()
		steps.visitCreateGroupPage()
		steps.fillInCreateGroupForm("E2E Example Group")
		steps.assertGroupSavedInDB("E2E Example Group")
	})
}

type GroupsSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *GroupsSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *GroupsSteps) visitCreateGroupPage() {
	s.session.Visit("/" + s.session.OrgID.String() + "/settings/create-group")
}

func (s *GroupsSteps) fillInCreateGroupForm(name string) {
	nameInput := q.Locator(`input[placeholder="Enter group name"]`)
	createButton := q.Text("Create Group")

	s.session.FillIn(nameInput, name)
	s.session.Click(createButton)
	s.session.WaitUntilURLDoesNotContain("/create-group")
}

func (s *GroupsSteps) assertGroupSavedInDB(displayName string) {
	groupName := normalizeGroupName(displayName)
	require.Eventually(s.t, func() bool {
		metadata, err := models.FindGroupMetadata(groupName, models.DomainTypeOrganization, s.session.OrgID.String())
		return err == nil && metadata.DisplayName == displayName
	}, 10*time.Second, 200*time.Millisecond, "group %q was not saved", displayName)
}

func normalizeGroupName(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	normalized = strings.ReplaceAll(normalized, " ", "_")
	return normalized
}
