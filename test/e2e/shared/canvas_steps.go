package shared

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/pkg/yaml"

	"github.com/superplanehq/superplane/test/e2e/queries"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

type CanvasSteps struct {
	t       *testing.T
	session *session.TestSession

	CanvasName string
	WorkflowID uuid.UUID
}

func NewCanvasSteps(name string, t *testing.T, session *session.TestSession) *CanvasSteps {
	return &CanvasSteps{t: t, session: session, CanvasName: name}
}

// EnterEditMode clicks the Edit button in the header to start editing the live canvas.
// This must be called before making any canvas changes.
func (s *CanvasSteps) EnterEditMode() {
	s.EnterEditModeWithoutStagingActionAssertions()
	if s.userHasStaging() {
		s.AssertStagingActionsVisibleAndEnabled()
		return
	}
	s.AssertStagingActionsVisibleAndDisabled()
}

// EnterEditModeWithoutStagingActionAssertions enters edit mode without assuming
// which staging actions the current user's permissions allow.
func (s *CanvasSteps) EnterEditModeWithoutStagingActionAssertions() {
	s.waitForEnabledEditButton()
	editButton := q.TestID("canvas-edit-button").Run(s.session)
	require.NoError(s.t, editButton.Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))
	s.session.Sleep(500)
	s.waitForEnabledExitEditButton()
	s.AssertEditModeTabChrome()
}

func (s *CanvasSteps) userHasStaging() bool {
	hasStaging, err := models.HasStagedFilesForUser(database.Conn(), s.WorkflowID, s.sessionUserID())
	return err == nil && hasStaging
}

// AssertEditModeTabChrome verifies the header tabs use edit-mode orange styling.
func (s *CanvasSteps) AssertEditModeTabChrome() {
	nav := q.Locator(`nav[aria-label="Canvas view"]`).Run(s.session)
	require.Eventually(s.t, func() bool {
		class, err := nav.GetAttribute("class")
		if err != nil || class == "" {
			return false
		}
		return strings.Contains(class, "bg-orange-200")
	}, 15*time.Second, 200*time.Millisecond, "edit-mode tab bar should use orange styling")

	activeTab := q.Locator(`nav[aria-label="Canvas view"] [aria-current="page"]`).Run(s.session)
	require.Eventually(s.t, func() bool {
		class, err := activeTab.GetAttribute("class")
		if err != nil || class == "" {
			return false
		}
		return strings.Contains(class, "bg-white") &&
			strings.Contains(class, "shadow-sm") &&
			strings.Contains(class, "font-medium") &&
			!strings.Contains(class, "font-bold")
	}, 15*time.Second, 200*time.Millisecond, "active edit-mode tab should use white pill styling")
}

// AssertStagingActionsVisibleAndDisabled verifies commit/reset stay visible but disabled with no staged changes.
func (s *CanvasSteps) AssertStagingActionsVisibleAndDisabled() {
	s.session.AssertVisible(q.TestID("canvas-commit-staging-button"))
	s.session.AssertVisible(q.TestID("canvas-reset-staging-button"))
	s.session.AssertDisabled(q.TestID("canvas-commit-staging-button"))
	s.session.AssertDisabled(q.TestID("canvas-reset-staging-button"))
}

// AssertStagingActionsVisibleAndEnabled verifies commit/reset are visible and actionable once staging exists.
func (s *CanvasSteps) AssertStagingActionsVisibleAndEnabled() {
	s.assertStagingActionEnabled("canvas-commit-staging-button")
	s.assertStagingActionEnabled("canvas-reset-staging-button")
}

func (s *CanvasSteps) assertStagingActionEnabled(testID string) {
	button := q.TestID(testID).Run(s.session)
	require.Eventually(s.t, func() bool {
		visible, err := button.IsVisible()
		if err != nil || !visible {
			return false
		}
		disabled, err := button.IsDisabled()
		return err == nil && !disabled
	}, 15*time.Second, 200*time.Millisecond, "%s should be visible and enabled when staging exists", testID)
}

func (s *CanvasSteps) sessionUserID() uuid.UUID {
	return s.UserIDForEmail(s.session.Account.Email)
}

// UserIDForEmail resolves the organization user id for an account email.
func (s *CanvasSteps) UserIDForEmail(email string) uuid.UUID {
	user, err := models.FindMaybeDeletedUserByEmail(s.session.OrgID.String(), email)
	require.NoError(s.t, err)
	return user.ID
}

// LoginAs switches the browser session to another account in the same organization.
func (s *CanvasSteps) LoginAs(account *models.Account) {
	s.session.Account = account
	s.session.Login()
}

// ExitEditMode leaves edit mode and returns to the live canvas view.
// Commit already ends the edit session in the UI; this is a no-op when live view is active.
func (s *CanvasSteps) ExitEditMode() {
	if s.isOnLiveView() {
		return
	}

	require.Eventually(s.t, func() bool {
		if s.isOnLiveView() {
			return true
		}
		exitEditButton := q.TestID("canvas-exit-edit-button").Run(s.session)
		visible, visibleErr := exitEditButton.IsVisible()
		if visibleErr != nil || !visible {
			return false
		}
		disabled, err := exitEditButton.IsDisabled()
		return err == nil && !disabled
	}, 30*time.Second, 200*time.Millisecond)

	if s.isOnLiveView() {
		return
	}

	s.waitForEnabledExitEditButton()
	exitEditButton := q.TestID("canvas-exit-edit-button").Run(s.session)
	require.NoError(s.t, exitEditButton.Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))
	s.waitForEnabledEditButton()
	s.session.Sleep(500)
}

func (s *CanvasSteps) isOnLiveView() bool {
	editButton := q.TestID("canvas-edit-button").Run(s.session)
	visible, err := editButton.IsVisible()
	if err != nil || !visible {
		return false
	}
	disabled, err := editButton.IsDisabled()
	return err == nil && !disabled
}

// OpenVersionsSidebar reveals the versions sidebar from the current canvas mode.
// If the header toggle is not available yet, it enters edit mode first.
func (s *CanvasSteps) OpenVersionsSidebar() {
	toggle := q.TestID("canvas-versions-sidebar-toggle").Run(s.session)
	if visible, _ := toggle.IsVisible(); !visible {
		s.EnterEditMode()
	}

	s.ensureVersionsSidebarOpen()
	s.session.AssertVisible(q.TestID("canvas-versions-sidebar"))
	s.session.Sleep(300)
}

func (s *CanvasSteps) ensureEditMode() {
	exitEditButton := q.TestID("canvas-exit-edit-button").Run(s.session)
	if visible, _ := exitEditButton.IsVisible(); visible {
		return
	}

	s.EnterEditMode()
}

func (s *CanvasSteps) ensureVersionsSidebarOpen() {
	toggle := q.TestID("canvas-versions-sidebar-toggle").Run(s.session)
	require.NoError(s.t, toggle.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(15000),
	}))

	pressed, err := toggle.GetAttribute("aria-pressed", pw.LocatorGetAttributeOptions{Timeout: pw.Float(15000)})
	require.NoError(s.t, err)
	if pressed == "true" {
		return
	}

	s.session.Click(q.TestID("canvas-versions-sidebar-toggle"))
}

// SelectVersionInHistorySidebar selects a commit in the versions sidebar history list.
func (s *CanvasSteps) SelectVersionInHistorySidebar(versionLabel string) {
	s.ensureEditMode()
	s.OpenVersionsSidebar()
	selector := q.Locator(fmt.Sprintf(`[data-testid="canvas-live-version-row"]:has-text("%s")`, versionLabel))
	s.waitForVisible(selector, 15*time.Second)
	s.session.Click(selector)
	s.session.Sleep(300)
}

// AssertPreviewingPreviousVersionBarVisible verifies the previous-version preview chrome is shown.
func (s *CanvasSteps) AssertPreviewingPreviousVersionBarVisible() {
	s.session.AssertVisible(q.TestID("canvas-mode-floating-bar"))
	s.session.AssertText("Previewing previous version")
	s.session.AssertVisible(q.TestID("canvas-mode-floating-bar-action"))
}

// ClickSeeCurrentVersionFromPreviewBar returns to the live version from previous-version preview.
func (s *CanvasSteps) ClickSeeCurrentVersionFromPreviewBar() {
	s.session.Click(q.TestID("canvas-mode-floating-bar-action"))
}

// WaitForCanvasDraftLoadingToFinish waits for the edit-session draft loading overlay to disappear.
func (s *CanvasSteps) WaitForCanvasDraftLoadingToFinish() {
	loading := q.Text("Loading canvas...").Run(s.session)
	require.Eventually(s.t, func() bool {
		visible, err := loading.IsVisible()
		return err == nil && !visible
	}, 30*time.Second, 200*time.Millisecond, "canvas draft loading overlay did not disappear")
}

// AssertEditSessionReady verifies the canvas finished bootstrapping an active edit session.
func (s *CanvasSteps) AssertEditSessionReady() {
	s.WaitForCanvasDraftLoadingToFinish()
	s.session.AssertHidden(q.TestID("canvas-mode-floating-bar"))
	s.waitForEnabledExitEditButton()
	s.AssertEditModeTabChrome()
	s.session.AssertVisible(q.TestID("building-blocks-sidebar"))
}

func (s *CanvasSteps) waitForVisible(query q.Query, timeout time.Duration) {
	locator := query.Run(s.session)
	require.Eventually(s.t, func() bool {
		visible, err := locator.IsVisible()
		return err == nil && visible
	}, timeout, 200*time.Millisecond, "%s did not become visible", query.Describe())
}

// SelectRunInSidebar opens run inspection by selecting a run from the runs sidebar.
func (s *CanvasSteps) SelectRunInSidebar(runID string) {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		runLink := q.Locator(fmt.Sprintf(`a[href*="run=%s"]`, runID))
		if visible, err := runLink.Run(s.session).IsVisible(); err == nil && visible {
			s.session.Click(runLink)
			s.session.Sleep(300)
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	s.session.Click(q.Locator(fmt.Sprintf(`a[href*="run=%s"]`, runID)))
}

func (s *CanvasSteps) waitForToolSidebarOpen() {
	deadline := time.Now().Add(15 * time.Second)
	sidebar := q.TestID("canvas-tool-sidebar").Run(s.session)
	openButton := q.TestID("canvas-tool-sidebar-toggle").Run(s.session)

	for time.Now().Before(deadline) {
		visible, err := sidebar.IsVisible()
		require.NoError(s.t, err)
		if visible {
			return
		}

		visible, err = openButton.IsVisible()
		require.NoError(s.t, err)
		if visible {
			err = openButton.Click(pw.LocatorClickOptions{Timeout: pw.Float(1000)})
			if err == nil {
				continue
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	s.session.AssertVisible(q.TestID("canvas-tool-sidebar"))
}

// WaitForRunsSidebar waits until the runs sidebar is visible on the canvas tab.
func (s *CanvasSteps) WaitForRunsSidebar() {
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		sidebar := q.TestID("canvas-runs-sidebar").Run(s.session)
		if visible, err := sidebar.IsVisible(); err == nil && visible {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	s.session.AssertVisible(q.TestID("canvas-runs-sidebar"))
}

// ListDraftVersions is deprecated; editable state is per-user staging on the live canvas.
func (s *CanvasSteps) ListDraftVersions() []models.CanvasVersion {
	live, err := models.FindLiveCanvasVersion(s.WorkflowID)
	if err != nil || live == nil {
		return nil
	}
	return []models.CanvasVersion{*live}
}

// AssertDraftCount is deprecated and only checks that a live version exists when expected > 0.
func (s *CanvasSteps) AssertDraftCount(expected int) {
	if expected == 0 {
		return
	}
	require.NotNil(s.t, s.FindCurrentDraft())
}

// AssertVersionHistoryContains verifies commit labels appear in the versions sidebar.
func (s *CanvasSteps) AssertVersionHistoryContains(labels ...string) {
	s.OpenVersionsSidebar()
	for _, label := range labels {
		s.session.AssertVisible(q.Locator(fmt.Sprintf(`[data-testid="versions-sidebar-scroll"] :text-is("%s")`, label)))
	}
}

func (s *CanvasSteps) waitForEnabledEditButton() {
	editButton := q.TestID("canvas-edit-button").Run(s.session)
	deadline := time.Now().Add(15 * time.Second)
	for {
		disabled, err := editButton.IsDisabled()
		require.NoError(s.t, err)
		if !disabled {
			return
		}
		if time.Now().After(deadline) {
			s.t.Fatalf("edit button did not become enabled")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func (s *CanvasSteps) WaitForEnabledExitEditButton() {
	s.waitForEnabledExitEditButton()
}

func (s *CanvasSteps) waitForEnabledExitEditButton() {
	exitEditButton := q.TestID("canvas-exit-edit-button").Run(s.session)
	deadline := time.Now().Add(30 * time.Second)
	for {
		visible, visibleErr := exitEditButton.IsVisible()
		if visibleErr == nil && visible {
			disabled, err := exitEditButton.IsDisabled()
			require.NoError(s.t, err)
			if !disabled {
				return
			}
		}
		if time.Now().After(deadline) {
			s.t.Fatalf("exit edit button did not become enabled")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// WaitForStaging waits until the current user has workflow_staged_files rows.
func (s *CanvasSteps) WaitForStaging(_ uuid.UUID) {
	userID := s.sessionUserID()
	require.Eventually(s.t, func() bool {
		hasStaging, err := models.HasStagedFilesForUser(database.Conn(), s.WorkflowID, userID)
		return err == nil && hasStaging
	}, 15*time.Second, 200*time.Millisecond)
}

// WaitForStagingOnCurrentDraft waits until the current user has staging rows and returns the live version id.
func (s *CanvasSteps) WaitForStagingOnCurrentDraft() uuid.UUID {
	s.WaitForStaging(uuid.Nil)
	live, err := models.FindLiveCanvasVersion(s.WorkflowID)
	require.NoError(s.t, err)
	s.AssertEditModeTabChrome()
	s.AssertStagingActionsVisibleAndEnabled()
	return live.ID
}

const canvasYAMLRepositoryPath = "canvas.yaml"

// DraftEffectiveSpec returns nodes and edges from staged canvas.yaml when present,
// otherwise from the live version row.
func (s *CanvasSteps) DraftEffectiveSpec() ([]models.Node, []models.Edge) {
	userID := s.sessionUserID()
	rows, err := models.ListStagedFilesForUser(database.Conn(), s.WorkflowID, userID)
	require.NoError(s.t, err)

	for _, row := range rows {
		if row.Path == canvasYAMLRepositoryPath && !row.Deleted && row.Content != "" {
			canvas, err := yaml.CanvasFromYAML([]byte(row.Content))
			if err != nil {
				break
			}
			return canvas.Nodes(), canvas.Edges()
		}
	}

	live, err := models.FindLiveCanvasVersion(s.WorkflowID)
	require.NoError(s.t, err)
	return live.Nodes, live.Edges
}

// DraftNodeByName returns a node from the effective draft state (staged or committed).
func (s *CanvasSteps) DraftNodeByName(name string) (models.Node, bool) {
	nodes, _ := s.DraftEffectiveSpec()
	for _, node := range nodes {
		if node.Name == name {
			return node, true
		}
	}
	return models.Node{}, false
}

// CommitAndPublish commits staged edits to main (commit replaces publish).
func (s *CanvasSteps) CommitAndPublish() {
	s.CommitStaging()
}

// CommitStaging opens the commit dialog, submits a message, and waits for staging to clear.
func (s *CanvasSteps) CommitStaging() {
	commitButton := q.TestID("canvas-commit-staging-button").Run(s.session)
	require.Eventually(s.t, func() bool {
		visible, err := commitButton.IsVisible()
		return err == nil && visible
	}, 15*time.Second, 200*time.Millisecond)
	s.AssertStagingActionsVisibleAndEnabled()

	require.NoError(s.t, commitButton.Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))

	messageInput := q.TestID("canvas-commit-message-input").Run(s.session)
	require.Eventually(s.t, func() bool {
		visible, err := messageInput.IsVisible()
		return err == nil && visible
	}, 15*time.Second, 200*time.Millisecond)
	require.NoError(s.t, messageInput.Fill("E2E commit"))

	submit := q.TestID("canvas-commit-message-submit").Run(s.session)
	require.NoError(s.t, submit.Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))

	userID := s.sessionUserID()
	require.Eventually(s.t, func() bool {
		hasStaging, err := models.HasStagedFilesForUser(database.Conn(), s.WorkflowID, userID)
		return err == nil && !hasStaging
	}, 30*time.Second, 200*time.Millisecond)

	// Commit closes the edit session and returns to the live canvas view.
	s.waitForEnabledEditButton()
}

// AssertNoStaging verifies the current user has no workflow_staged_files rows.
func (s *CanvasSteps) AssertNoStaging(_ uuid.UUID) {
	s.AssertNoStagingForUser(s.sessionUserID())
}

// AssertHasStaging verifies the current user has workflow_staged_files rows.
func (s *CanvasSteps) AssertHasStaging(_ uuid.UUID) {
	s.AssertHasStagingForUser(s.sessionUserID())
}

// AssertHasStagingForUser verifies a specific user has workflow_staged_files rows.
func (s *CanvasSteps) AssertHasStagingForUser(userID uuid.UUID) {
	hasStaging, err := models.HasStagedFilesForUser(database.Conn(), s.WorkflowID, userID)
	require.NoError(s.t, err)
	require.True(s.t, hasStaging, "expected staging rows for user %s on workflow %s", userID, s.WorkflowID)
}

// AssertNoStagingForUser verifies a specific user has no workflow_staged_files rows.
func (s *CanvasSteps) AssertNoStagingForUser(userID uuid.UUID) {
	hasStaging, err := models.HasStagedFilesForUser(database.Conn(), s.WorkflowID, userID)
	require.NoError(s.t, err)
	require.False(s.t, hasStaging, "expected no staging rows for user %s on workflow %s", userID, s.WorkflowID)
}

// AssertStagingStaleForUser verifies staged files were created against an older live version.
func (s *CanvasSteps) AssertStagingStaleForUser(userID uuid.UUID) {
	live, err := models.FindLiveCanvasVersion(s.WorkflowID)
	require.NoError(s.t, err)

	rows, err := models.ListStagedFilesForUser(database.Conn(), s.WorkflowID, userID)
	require.NoError(s.t, err)
	require.NotEmpty(s.t, rows)

	for _, row := range rows {
		require.NotEqual(s.t, live.ID, row.BaseVersionID, "expected stale staging for user %s", userID)
	}
}

// StagingContainsNodeForUser reports whether a user's staged canvas.yaml includes a node name.
func (s *CanvasSteps) StagingContainsNodeForUser(userID uuid.UUID, nodeName string) bool {
	rows, err := models.ListStagedFilesForUser(database.Conn(), s.WorkflowID, userID)
	if err != nil {
		return false
	}
	for _, row := range rows {
		if row.Path == canvasYAMLRepositoryPath && strings.Contains(row.Content, nodeName) {
			return true
		}
	}
	return false
}

// FindDraftByDisplayName returns the live version (display names are no longer used).
func (s *CanvasSteps) FindDraftByDisplayName(_ string) *models.CanvasVersion {
	return s.FindCurrentDraft()
}

// DraftCommittedNodeNames returns node names stored on the committed draft version row.
func (s *CanvasSteps) DraftCommittedNodeNames(versionID uuid.UUID) []string {
	version, err := models.FindCanvasVersion(s.WorkflowID, versionID)
	require.NoError(s.t, err)

	names := make([]string, 0, len(version.Nodes))
	for _, node := range version.Nodes {
		names = append(names, node.Name)
	}
	return names
}

// AssertDraftCommittedHasNode verifies a node exists on the committed version row (not staging).
func (s *CanvasSteps) AssertDraftCommittedHasNode(versionID uuid.UUID, nodeName string) {
	require.Eventually(s.t, func() bool {
		return slices.Contains(s.DraftCommittedNodeNames(versionID), nodeName)
	}, 10*time.Second, 200*time.Millisecond)
}

// AssertDraftCommittedLacksNode verifies a node is absent from the committed version row.
func (s *CanvasSteps) AssertDraftCommittedLacksNode(versionID uuid.UUID, nodeName string) {
	require.Eventually(s.t, func() bool {
		return !slices.Contains(s.DraftCommittedNodeNames(versionID), nodeName)
	}, 5*time.Second, 200*time.Millisecond)
}

// AssertLiveCanvasHasNode verifies a node exists on the live (published) canvas.
func (s *CanvasSteps) AssertLiveCanvasHasNode(nodeName string) {
	require.Eventually(s.t, func() bool {
		nodes, err := models.FindCanvasNodes(s.WorkflowID)
		if err != nil {
			return false
		}
		for _, node := range nodes {
			if node.Name == nodeName {
				return true
			}
		}
		return false
	}, 15*time.Second, 200*time.Millisecond)
}

// AssertLiveCanvasLacksNode verifies a node is absent from the materialized live canvas.
func (s *CanvasSteps) AssertLiveCanvasLacksNode(nodeName string) {
	require.Eventually(s.t, func() bool {
		nodes, err := models.FindCanvasNodes(s.WorkflowID)
		if err != nil {
			return false
		}
		for _, node := range nodes {
			if node.Name == nodeName {
				return false
			}
		}
		return true
	}, 10*time.Second, 200*time.Millisecond)
}

// LiveVersionID returns the current live workflow_versions row id.
func (s *CanvasSteps) LiveVersionID() uuid.UUID {
	live, err := models.FindLiveCanvasVersion(s.WorkflowID)
	require.NoError(s.t, err)
	return live.ID
}

// AssertLiveVersionHasNode verifies a node exists on the live version spec row.
func (s *CanvasSteps) AssertLiveVersionHasNode(nodeName string) {
	require.Eventually(s.t, func() bool {
		live, err := models.FindLiveCanvasVersion(s.WorkflowID)
		if err != nil {
			return false
		}
		for _, node := range live.Nodes {
			if node.Name == nodeName {
				return true
			}
		}
		return false
	}, 15*time.Second, 200*time.Millisecond)
}

// AssertLiveVersionLacksNode verifies a node is absent from the live version spec row.
func (s *CanvasSteps) AssertLiveVersionLacksNode(nodeName string) {
	require.Eventually(s.t, func() bool {
		live, err := models.FindLiveCanvasVersion(s.WorkflowID)
		if err != nil {
			return false
		}
		for _, node := range live.Nodes {
			if node.Name == nodeName {
				return false
			}
		}
		return true
	}, 10*time.Second, 200*time.Millisecond)
}

// AssertLiveVersionCommitMessage verifies the live version commit message.
func (s *CanvasSteps) AssertLiveVersionCommitMessage(message string) {
	live, err := models.FindLiveCanvasVersion(s.WorkflowID)
	require.NoError(s.t, err)
	require.Equal(s.t, message, live.CommitMessage)
}

// AssertVersionCountAtLeast verifies the canvas has at least the expected number of versions.
func (s *CanvasSteps) AssertVersionCountAtLeast(expected int) {
	require.Eventually(s.t, func() bool {
		versions, err := models.ListCanvasVersions(s.WorkflowID)
		return err == nil && len(versions) >= expected
	}, 15*time.Second, 200*time.Millisecond)
}

// WaitForPublishEnabled is deprecated; commit replaces publish.
func (s *CanvasSteps) WaitForPublishEnabled() {}

// Publish is deprecated; use CommitStaging or CommitAndPublish instead.
func (s *CanvasSteps) Publish() {}

// FindCurrentDraft returns the live canvas version used as the staging base.
func (s *CanvasSteps) FindCurrentDraft() *models.CanvasVersion {
	live, err := models.FindLiveCanvasVersion(s.WorkflowID)
	if err != nil {
		return nil
	}
	return live
}

func (s *CanvasSteps) Create() {
	user, err := models.FindMaybeDeletedUserByEmail(s.session.OrgID.String(), s.session.Account.Email)
	require.NoError(s.t, err)
	canvas, _ := support.CreateCanvas(s.t, s.session.OrgID, user.ID, nil, nil)
	s.WorkflowID = canvas.ID

	err = database.Conn().
		Model(&models.Canvas{}).
		Where("id = ?", s.WorkflowID).
		Update("name", s.CanvasName).Error
	require.NoError(s.t, err)

	s.Visit()
}

func (s *CanvasSteps) CreatePublishedWithParameterizedManualRun() {
	user, err := models.FindMaybeDeletedUserByEmail(s.session.OrgID.String(), s.session.Account.Email)
	require.NoError(s.t, err)

	startNodeID := "start-trigger"
	outputNodeID := "noop-output"

	canvas, _ := support.CreateCanvas(s.t, s.session.OrgID, user.ID, []models.CanvasNode{
		{
			NodeID: startNodeID,
			Name:   "Start",
			Type:   models.NodeTypeTrigger,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "start"},
			}),
			Configuration: datatypes.NewJSONType(map[string]any{
				"templates": []any{
					map[string]any{
						"name": "Hello World",
						"payload": map[string]any{
							"message": `{{ parameters["message"] }}`,
						},
						"parameters": []any{
							map[string]any{
								"name": "message",
								"type": "string",
							},
						},
					},
				},
			}),
			Position: datatypes.NewJSONType(models.Position{X: 600, Y: 200}),
		},
		{
			NodeID: outputNodeID,
			Name:   "Output",
			Type:   models.NodeTypeComponent,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: "noop"},
			}),
			Configuration: datatypes.NewJSONType(map[string]any{}),
			Position:      datatypes.NewJSONType(models.Position{X: 1000, Y: 200}),
		},
	}, []models.Edge{
		{SourceID: startNodeID, TargetID: outputNodeID, Channel: "default"},
	})
	s.WorkflowID = canvas.ID

	err = database.Conn().
		Model(&models.Canvas{}).
		Where("id = ?", s.WorkflowID).
		Update("name", s.CanvasName).Error
	require.NoError(s.t, err)

	s.Visit()
}

func (s *CanvasSteps) Visit() {
	s.session.Visit("/" + s.session.OrgID.String() + "/apps/" + s.WorkflowID.String())
}

func (s *CanvasSteps) OpenBuildingBlocksSidebar() {
	sidebar := q.TestID("building-blocks-sidebar").Run(s.session)
	if isVisible, _ := sidebar.IsVisible(); isVisible {
		return
	}

	editButton := q.TestID("canvas-edit-button").Run(s.session)
	addComponentButton := q.TestID("canvas-add-component-button").Run(s.session)
	openButton := q.TestID("open-sidebar-button").Run(s.session)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if isVisible, _ := sidebar.IsVisible(); isVisible {
			return
		}

		// Newer canvas UI keeps the component sidebar open after selecting a node.
		// Deselecting the node reveals the floating Components button again.
		s.ClickOnEmptyCanvasArea()
		s.session.Sleep(150)

		if isVisible, _ := sidebar.IsVisible(); isVisible {
			return
		}

		if isVisible, _ := editButton.IsVisible(); isVisible {
			if err := editButton.Click(); err == nil {
				s.session.Sleep(250)
			}
		}

		if isVisible, _ := addComponentButton.IsVisible(); isVisible {
			if err := addComponentButton.Click(); err == nil {
				s.session.Sleep(250)
			}
		}

		if isVisible, _ := openButton.IsVisible(); isVisible {
			if err := openButton.Click(); err == nil {
				s.session.Sleep(250)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	s.session.AssertVisible(q.TestID("building-blocks-sidebar"))
}

func (s *CanvasSteps) OpenBuildingBlockCategory(categoryName string) {
	s.OpenBuildingBlocksSidebar()

	details := q.Locator(fmt.Sprintf(
		`[data-testid="building-blocks-sidebar"] details:has(summary :text-is("%s"))`,
		categoryName,
	)).Run(s.session)

	open, err := details.GetAttribute("open")
	require.NoError(s.t, err)
	if open != "" {
		return
	}

	s.session.Click(q.Locator(fmt.Sprintf(
		`[data-testid="building-blocks-sidebar"] details:has(summary :text-is("%s")) summary`,
		categoryName,
	)))
	s.session.Sleep(200)
}

// ClickOnEmptyCanvasArea clicks on an empty area of the canvas to dismiss
// any open sidebars and deselect all nodes.
func (s *CanvasSteps) ClickOnEmptyCanvasArea() {
	target := q.TestID("rf__wrapper")
	el := target.Run(s.session)
	box, _ := el.BoundingBox()
	if box != nil {
		_ = s.session.Page().Mouse().Click(box.X+600, box.Y+50)
	}
}

// SelectAllNodes performs a rubber-band drag selection across the entire visible
// canvas area to select all nodes. The sidebar must be closed before calling this.
func (s *CanvasSteps) SelectAllNodes() {
	target := q.TestID("rf__wrapper")
	s.session.DragSelectOnCanvas(target, 10, 10, 1100, 700)
}

func (s *CanvasSteps) AddNoop(name string, pos models.Position) {
	s.OpenBuildingBlockCategory("Debugging")

	source := q.TestID("building-block-noop")
	s.addBlockFromSidebar(source, pos)
	s.session.Sleep(500)

	s.selectLatestNoopNode()
	s.session.FillIn(q.TestID("node-name-input"), name)
	s.session.Sleep(300)
	s.waitForDraftNodeID(name)
}

func (s *CanvasSteps) AddNote() {
	// The "Add Note" button only appears in the closed building blocks sidebar.
	// If the sidebar is currently open, close it first by clicking on empty canvas area.
	sidebar := q.TestID("building-blocks-sidebar").Run(s.session)
	if isVisible, _ := sidebar.IsVisible(); isVisible {
		s.ClickOnEmptyCanvasArea()
		s.session.Sleep(300)
	}

	s.session.Click(q.TestID("add-note-button"))
	require.Eventually(s.t, func() bool {
		_, ok := s.DraftNodeByName("Note")
		return ok
	}, 10*time.Second, 200*time.Millisecond)
	s.session.AssertVisible(q.Text("Double click to add and edit notes..."))
	s.session.Sleep(300)
}

// AddNoopWithDefaultName adds a noop node using the auto-generated name and returns that name.
func (s *CanvasSteps) AddNoopWithDefaultName(pos models.Position) string {
	s.OpenBuildingBlockCategory("Debugging")

	source := q.TestID("building-block-noop")
	s.addBlockFromSidebar(source, pos)
	s.session.Sleep(500)

	s.selectLatestNoopNode()

	// Get the auto-generated name from the input field
	nameInput := q.TestID("node-name-input")
	loc := nameInput.Run(s.session)
	generatedName, err := loc.InputValue()
	require.NoError(s.t, err)

	s.session.Sleep(300)

	return generatedName
}

func (s *CanvasSteps) Save() {
	saveButton := q.TestID("save-canvas-button")
	loc := saveButton.Run(s.session)

	if isVisible, _ := loc.IsVisible(); isVisible {
		s.session.Click(saveButton)
		s.session.AssertText("Canvas changes saved")
		s.session.Sleep(500)
		return
	}

	s.session.Sleep(300)
}

func (s *CanvasSteps) AddApproval(nodeName string, pos models.Position) {
	s.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-approval")
	s.addBlockFromSidebar(source, pos)
	s.session.Sleep(300)
	s.openComponentSidebarForLatestBlock("building-block-approval")

	s.session.FillIn(q.TestID("node-name-input"), nodeName)

	s.session.Click(q.TestID("field-type-select"))
	s.session.Click(q.Locator(`div[role="option"]:has-text("Specific user")`))

	s.session.Click(q.Locator(`button:has-text("Select user")`))
	s.session.Click(q.Locator(`div[role="option"]:has-text("e2e@superplane.local")`))

	s.session.Sleep(300)
}

func (s *CanvasSteps) AddManualTrigger(name string, pos models.Position) {
	s.OpenBuildingBlocksSidebar()

	startSource := q.TestID("building-block-start")
	s.addBlockFromSidebar(startSource, pos)
	s.openComponentSidebarForLatestBlock("building-block-start")
	s.session.FillIn(q.TestID("node-name-input"), name)
	s.session.Sleep(300)
}

func (s *CanvasSteps) AddWait(name string, pos models.Position, duration int, unit string) {
	s.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-wait")
	s.addBlockFromSidebar(source, pos)
	s.session.Sleep(300)
	s.openComponentSidebarForLatestBlock("building-block-wait")
	s.session.FillIn(q.TestID("node-name-input"), name)

	modeSelector := q.TestID("field-mode-select")
	s.session.Click(modeSelector)
	s.session.Click(q.Locator(`div[role="option"]:has-text("Interval")`))

	valueInput := q.Locator("textarea[data-testid='string-field-waitfor']")
	s.session.FillIn(valueInput, strconv.Itoa(duration))

	unitTrigger := q.TestID("field-unit-select")
	s.session.Click(unitTrigger)
	s.session.Click(q.Locator(`div[role="option"]:has-text("` + unit + `")`))

	s.session.Sleep(300)
}

func (s *CanvasSteps) AddFilter(name string, pos models.Position) {
	s.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-filter")
	s.addBlockFromSidebar(source, pos)
	s.session.Sleep(300)
	s.openComponentSidebarForLatestBlock("building-block-filter")
	s.session.FillIn(q.TestID("node-name-input"), name)
	s.session.FillIn(q.TestID("expression-field-expression"), "true")
	s.session.Sleep(300)
}

func (s *CanvasSteps) StartAddingTimeGate(name string, pos models.Position) {
	s.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-timeGate")
	s.addBlockFromSidebar(source, pos)
	s.session.Sleep(300)
	s.openComponentSidebarForLatestBlock("building-block-timeGate")

	s.session.FillIn(q.TestID("node-name-input"), name)
}

func (s *CanvasSteps) AddTimeGate(name string, pos models.Position) {
	s.OpenBuildingBlocksSidebar()

	source := q.TestID("building-block-timeGate")
	s.addBlockFromSidebar(source, pos)
	s.session.Sleep(300)
	s.openComponentSidebarForLatestBlock("building-block-timeGate")

	s.session.FillIn(q.TestID("node-name-input"), name)
	s.session.FillIn(q.TestID("time-field-timerange-start"), "00:00")
	s.session.FillIn(q.TestID("time-field-timerange-end"), "23:59")

	s.session.Click(q.TestID("field-timezone-select"))
	s.session.Click(q.Locator(`div[role="option"]:has-text("GMT+0 (London, Dublin, UTC)")`))

	s.session.Sleep(300)
}

func (s *CanvasSteps) AddBuildingBlockByTestID(blockTestID string, pos models.Position) {
	s.OpenBuildingBlocksSidebar()
	s.addBlockFromSidebar(q.TestID(blockTestID), pos)
	s.session.Sleep(500)
	s.openComponentSidebarForLatestBlock(blockTestID)
}

func (s *CanvasSteps) addBlockFromSidebar(source queries.Query, pos models.Position) {
	target := q.TestID("rf__wrapper")
	s.session.DragAndDrop(source, target, pos.X, pos.Y)
}

func (s *CanvasSteps) openComponentSidebarForLatestBlock(blockTestID string) {
	slug := strings.ToLower(strings.TrimPrefix(blockTestID, "building-block-"))
	headers := s.session.Page().Locator(fmt.Sprintf(
		`.react-flow__node [data-testid^="node-%s"][data-testid$="-header"]`,
		slug,
	))
	count, err := headers.Count()
	require.NoError(s.t, err)
	require.Greater(s.t, count, 0, "expected at least one %s node after dropping block", slug)

	require.NoError(s.t, headers.Nth(count-1).Click(pw.LocatorClickOptions{Timeout: pw.Float(15000)}))
	s.session.Sleep(150)
}

func (s *CanvasSteps) selectLatestNoopNode() {
	s.openComponentSidebarForLatestBlock("building-block-noop")
}

func (s *CanvasSteps) Connect(sourceName, targetName string) {
	sourceNodeID := s.waitForDraftNodeID(sourceName)
	targetNodeID := s.waitForDraftNodeID(targetName)

	sourceHandle := q.Locator(`.react-flow__node[data-id="` + sourceNodeID + `"] .react-flow__handle-right`)
	targetHandle := q.Locator(`.react-flow__node[data-id="` + targetNodeID + `"] .react-flow__handle-left`)

	s.session.DragAndDrop(sourceHandle, targetHandle, 6, 6)
	s.session.Sleep(300)
}

func (s *CanvasSteps) waitForDraftNodeID(nodeName string) string {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if nodeID := s.nodeIDFromCanvasDOM(nodeName); nodeID != "" {
			return nodeID
		}

		for _, draft := range s.ListDraftVersions() {
			for _, node := range draft.Nodes {
				if node.Name == nodeName {
					return node.ID
				}
			}

			if s.userStagingYAMLContainsNodeName(nodeName) {
				if nodeID := s.nodeIDFromCanvasDOM(nodeName); nodeID != "" {
					return nodeID
				}
			}
		}
		time.Sleep(200 * time.Millisecond)
	}

	s.t.Fatalf("node %q not found in staged or live canvas", nodeName)
	return ""
}

func (s *CanvasSteps) nodeIDFromCanvasDOM(nodeName string) string {
	safe := strings.ToLower(nodeName)
	safe = strings.ReplaceAll(safe, " ", "-")
	loc := q.Locator(`.react-flow__node:has([data-testid="node-` + safe + `-header"])`).Run(s.session)
	id, err := loc.GetAttribute("data-id")
	if err != nil || id == "" {
		return ""
	}
	return id
}

func (s *CanvasSteps) userStagingYAMLContainsNodeName(nodeName string) bool {
	rows, err := models.ListStagedFilesForUser(database.DB(s.t.Context()), s.WorkflowID, s.sessionUserID())
	if err != nil {
		return false
	}

	for _, row := range rows {
		if row.Path == "canvas.yaml" && strings.Contains(row.Content, nodeName) {
			return true
		}
	}
	return false
}

func (s *CanvasSteps) draftStagingYAMLContainsNodeName(_ uuid.UUID, nodeName string) bool {
	return s.userStagingYAMLContainsNodeName(nodeName)
}

func (s *CanvasSteps) DeleteConnection(sourceName, targetName string) {
	sourceNodeID := s.waitForDraftNodeID(sourceName)
	targetNodeID := s.waitForDraftNodeID(targetName)

	edge := q.Locator(`.react-flow__edge`).Run(s.session)
	require.Eventually(s.t, func() bool {
		count, err := edge.Count()
		return err == nil && count > 0
	}, 10*time.Second, 200*time.Millisecond)

	// The edge midpoint lies on the source node's right handle and the target
	// node's left handle line. Computing it from the handle positions gives a
	// point that is reliably on the (mostly horizontal) edge path. Playwright's
	// Locator.Hover()/Click() target an element's bounding-box center, which is
	// unreliable for an SVG path: the geometric center of the bounding box can
	// fall off the actual stroke, so the action never lands on the edge.
	sourceHandle := q.Locator(`.react-flow__node[data-id="` + sourceNodeID + `"] .react-flow__handle-right`).Run(s.session)
	targetHandle := q.Locator(`.react-flow__node[data-id="` + targetNodeID + `"] .react-flow__handle-left`).Run(s.session)

	sourceBox, err := sourceHandle.BoundingBox()
	require.NoError(s.t, err)
	require.NotNil(s.t, sourceBox)
	targetBox, err := targetHandle.BoundingBox()
	require.NoError(s.t, err)
	require.NotNil(s.t, targetBox)

	midX := (sourceBox.X + sourceBox.Width/2 + targetBox.X + targetBox.Width/2) / 2
	midY := (sourceBox.Y + sourceBox.Height/2 + targetBox.Y + targetBox.Height/2) / 2

	// In edit mode the wide transparent delete hit-area path is always present
	// (canDelete = isEditMode && !isReadOnly), so a hover is not required to
	// reveal it. Move the mouse onto the edge to set the hovered state, then
	// dispatch a raw click at the same on-edge point. Using raw mouse events
	// avoids the unreliable element-center actionability checks.
	hitArea := q.Locator(`.react-flow__renderer [data-testid="edge-delete-hit-area"]`).Run(s.session)
	require.Eventually(s.t, func() bool {
		count, err := hitArea.Count()
		return err == nil && count > 0
	}, 10*time.Second, 200*time.Millisecond)

	mouse := s.session.Page().Mouse()
	require.NoError(s.t, mouse.Move(midX, midY))
	s.session.Sleep(300)
	require.NoError(s.t, mouse.Click(midX, midY))
	s.session.Sleep(500)
	s.waitForDraftEdgeCount(0)
}

func (s *CanvasSteps) waitForDraftEdgeCount(expected int) {
	require.Eventually(s.t, func() bool {
		_, edges := s.DraftEffectiveSpec()
		return len(edges) == expected
	}, 10*time.Second, 200*time.Millisecond, "draft edge count to reach %d", expected)
}

func (s *CanvasSteps) StartEditingNode(name string) {
	// Click on the node header to open the sidebar where settings can be accessed
	nodeHeader := q.TestID("node", name, "header")
	s.session.Click(nodeHeader)
	s.session.Sleep(300)
}

func (s *CanvasSteps) RunManualTrigger(name string) {
	// Use the Start node's template Run button (in the default payload template) instead of the removed header Run button
	startTemplateRun := q.Locator(`.react-flow__node:has([data-testid="node-` + strings.ToLower(name) + `-header"]) [data-testid="start-template-run"]`)
	s.session.Click(startTemplateRun)
	s.session.Click(q.TestID("emit-event-submit-button"))
}

func (s *CanvasSteps) RunParameterizedManualTrigger(name string, parameters map[string]string) {
	startTemplateRun := q.Locator(`.react-flow__node:has([data-testid="node-` + strings.ToLower(name) + `-header"]) [data-testid="start-template-run"]`)
	s.session.Click(startTemplateRun)

	for paramName, value := range parameters {
		s.session.FillIn(q.Locator("#start-run-param-"+paramName), value)
	}

	s.session.Click(q.TestID("emit-event-submit-button"))
}

func (s *CanvasSteps) EmitManualTrigger(name string) {
	node := s.GetNodeFromDB(name)
	context := contexts.NewEventContext(database.Conn(), node, nil, func(events []models.CanvasEvent) {
		for i := range events {
			require.NoError(s.t, messages.PublishCanvasEventCreatedMessage(&events[i]))
		}
	})

	require.NoError(s.t, context.Emit("manual.run", map[string]any{"message": "Hello, World!"}))
}

func (s *CanvasSteps) RenameNode(name string, newName string) {
	node := s.GetNodeFromDB(name)

	query := database.Conn().
		Model(&models.CanvasNode{}).
		Where("workflow_id = ?", s.WorkflowID).
		Where("node_id = ?", node.NodeID).
		Update("name", newName)

	err := query.Error
	require.NoError(s.t, err)
}

func (s *CanvasSteps) GetWorkflowFromDB() *models.Canvas {
	workflow, err := models.FindCanvas(s.session.OrgID, s.WorkflowID)
	require.NoError(s.t, err)

	return workflow
}

func (s *CanvasSteps) GetNodeFromDB(name string) *models.CanvasNode {
	canvas, err := models.FindCanvas(s.session.OrgID, s.WorkflowID)
	require.NoError(s.t, err)

	nodes, err := models.FindCanvasNodes(canvas.ID)
	require.NoError(s.t, err)

	nodeID := ""
	for _, n := range nodes {
		if n.Name == name {
			nodeID = n.NodeID
			break
		}
	}

	if nodeID == "" {
		s.t.Fatalf("node %s not found in database", name)
		return nil
	}

	node, err := models.FindCanvasNode(database.Conn(), s.WorkflowID, nodeID)
	require.NoError(s.t, err)

	return node
}

func (s *CanvasSteps) GetExecutionsForNode(name string) []models.CanvasNodeExecution {
	node := s.GetNodeFromDB(name)

	var executions []models.CanvasNodeExecution

	query := database.Conn().
		Where("workflow_id = ?", s.WorkflowID).
		Where("node_id = ?", node.NodeID).
		Order("created_at DESC")

	err := query.Find(&executions).Error
	require.NoError(s.t, err)

	return executions
}

func (s *CanvasSteps) GetExecutionsForNodeInState(name string, state string) []models.CanvasNodeExecution {
	node := s.GetNodeFromDB(name)

	var executions []models.CanvasNodeExecution

	query := database.Conn().
		Where("workflow_id = ?", s.WorkflowID).
		Where("node_id = ?", node.NodeID).
		Where("state = ?", state).
		Order("created_at DESC")

	err := query.Find(&executions).Error
	require.NoError(s.t, err)

	return executions
}

func (s *CanvasSteps) GetExecutionsForNodeInStates(name string, states []string) []models.CanvasNodeExecution {
	node := s.GetNodeFromDB(name)

	var executions []models.CanvasNodeExecution

	query := database.Conn().
		Where("workflow_id = ?", s.WorkflowID).
		Where("node_id = ?", node.NodeID).
		Where("state IN ?", states).
		Order("created_at DESC")

	err := query.Find(&executions).Error
	require.NoError(s.t, err)

	return executions
}

func (s *CanvasSteps) WaitForExecution(name string, state string, timeout time.Duration) {
	found := false
	start := time.Now()

	for time.Since(start) < timeout {
		executions := s.GetExecutionsForNodeInState(name, state)
		if len(executions) > 0 {
			found = true
			break
		}

		s.t.Log("waiting for execution of node", name)
		s.session.Sleep(1000)
	}

	require.True(s.t, found, "timed out waiting for execution of node %s", name)
}

func (s *CanvasSteps) WaitForExecutionInStates(name string, states []string, timeout time.Duration) {
	found := false
	start := time.Now()

	for time.Since(start) < timeout {
		executions := s.GetExecutionsForNodeInStates(name, states)
		if len(executions) > 0 {
			found = true
			break
		}

		s.t.Log("waiting for execution of node", name)
		s.session.Sleep(1000)
	}

	require.True(s.t, found, "timed out waiting for execution of node %s", name)
}
