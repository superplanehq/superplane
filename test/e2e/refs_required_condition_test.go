package e2e

// -----------------------------------------------------------------------------
// Deleting the last condition of a required predicate list must not resurrect
// the field default as a phantom row (user-reported on github.onPush "Refs").
//
// Before the fix, AnyPredicateListFieldRenderer cleared the field with
// onChange(undefined); the autosave then dropped the "refs" key from the staged
// configuration, and the settings form re-merges field DEFAULTS for keys
// missing from the saved configuration. The UI re-displayed the default
// "Equals refs/heads/main" row while the stored config contained no refs at
// all, so the required warning was suppressed and committing produced a live
// node with the error "field 'refs' is required".
//
// After the fix, removing the last condition of a REQUIRED predicate list
// keeps the key present as an empty list: the saved configuration matches the
// UI (zero rows plus an inline "At least one condition is required" hint), and
// adding a condition back heals the node before commit.
// -----------------------------------------------------------------------------

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

const onPushTriggerName = "github.onPush"

const requiredConditionHint = "At least one condition is required"

func TestRefsRequiredCondition(t *testing.T) {
	steps := &refsRequiredConditionSteps{t: t}

	t.Run("deleting the last refs condition keeps the UI and the stored config in sync", func(t *testing.T) {
		steps.start()
		steps.givenAReadyGithubIntegrationExists("gh-e2e-integration")
		steps.givenACanvasExists()
		steps.whenIAddAGithubOnPushTrigger()
		steps.whenISetTheRepositoryViaExpressionMode()
		steps.thenTheRefsFieldShowsTheDefaultConditionAndItIsStored()
		steps.whenIDeleteTheLastRefsCondition()
		steps.thenNoPhantomRowReappears()
		steps.thenTheStagedConfigKeepsRefsAsAnEmptyList()
		steps.whenIAddARefsCondition("refs/heads/develop")
		steps.thenTheStagedConfigContainsTheNewConditionAndTheHintClears()
		steps.whenICommitTheDraft()
		steps.thenTheCommittedNodeHasNoError()
	})
}

type refsRequiredConditionSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *refsRequiredConditionSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *refsRequiredConditionSteps) givenAReadyGithubIntegrationExists(name string) {
	integration, err := models.CreateIntegration(uuid.New(), s.session.OrgID, "github", name, nil)
	require.NoError(s.t, err)
	require.NoError(s.t, database.Conn().Model(integration).Update("state", models.IntegrationStateReady).Error)
}

func (s *refsRequiredConditionSteps) givenACanvasExists() {
	s.canvas = shared.NewCanvasSteps("Refs Required Condition", s.t, s.session)
	s.canvas.Create()
	s.canvas.EnterEditMode()
}

func (s *refsRequiredConditionSteps) whenIAddAGithubOnPushTrigger() {
	s.canvas.OpenBuildingBlockCategory("GitHub")
	s.canvas.AddBuildingBlockByTestID("building-block-github.onpush", models.Position{X: 700, Y: 250})
	s.session.AssertVisible(q.TestID("node-name-input"))
}

// whenISetTheRepositoryViaExpressionMode fills the required repository field
// through the resource field's Expression mode (free text), so server-side
// validation of the committed node is exercised on the refs field rather than
// stopping at an empty repository.
func (s *refsRequiredConditionSteps) whenISetTheRepositoryViaExpressionMode() {
	s.session.Click(q.Locator(`button:has-text("Expression")`))
	expressionInput := q.Locator(`textarea[placeholder^="e.g. {{"]`)
	s.session.AssertVisible(expressionInput)
	s.session.FillIn(expressionInput, `{{ "superplanehq/superplane" }}`)
	// Blur so text-like editors persist their value.
	s.session.Click(q.TestID("node-name-input"))

	require.Eventually(s.t, func() bool {
		config, ok := s.onPushNodeConfig()
		if !ok {
			return false
		}
		repository, hasRepository := config["repository"]
		if !hasRepository {
			return false
		}
		str, ok := repository.(string)
		return ok && str != ""
	}, 15*time.Second, 200*time.Millisecond, "staged config should contain the repository expression")
}

// refsValueTextarea is the predicate row's value control (AutoCompleteInput
// renders a textarea with the field placeholder, "Value" for refs).
func (s *refsRequiredConditionSteps) refsValueTextarea() pw.Locator {
	return s.session.Page().Locator(`textarea[placeholder="Value"]`)
}

func (s *refsRequiredConditionSteps) onPushNodeConfig() (map[string]any, bool) {
	nodes, _ := s.canvas.DraftEffectiveSpec()
	for _, node := range nodes {
		if node.ComponentName() == onPushTriggerName {
			return node.Configuration, true
		}
	}
	return nil, false
}

func (s *refsRequiredConditionSteps) refsPredicates() ([]any, bool) {
	config, ok := s.onPushNodeConfig()
	if !ok {
		return nil, false
	}
	refs, hasRefs := config["refs"]
	if !hasRefs {
		return nil, false
	}
	list, ok := refs.([]any)
	return list, ok
}

func (s *refsRequiredConditionSteps) thenTheRefsFieldShowsTheDefaultConditionAndItIsStored() {
	textarea := s.refsValueTextarea()
	require.NoError(s.t, textarea.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(15000),
	}))
	value, err := textarea.InputValue()
	require.NoError(s.t, err)
	require.Equal(s.t, "refs/heads/main", value, "refs default row should be displayed")

	require.Eventually(s.t, func() bool {
		list, ok := s.refsPredicates()
		return ok && len(list) == 1
	}, 15*time.Second, 200*time.Millisecond, "staged config should initially contain the refs default")
}

func (s *refsRequiredConditionSteps) whenIDeleteTheLastRefsCondition() {
	deleteButton := q.Locator(`button:has(svg.text-red-500)`)
	s.session.AssertVisible(deleteButton)
	s.session.Click(deleteButton)
}

// thenNoPhantomRowReappears waits for the autosave triggered by the deletion to
// land, then samples the predicate row count over the same 100ms windows the
// bug repro used: before the fix the row disappeared for ~300ms and then
// reappeared pre-filled with the field default while the stored config had no
// refs value at all.
func (s *refsRequiredConditionSteps) thenNoPhantomRowReappears() {
	require.Eventually(s.t, func() bool {
		config, ok := s.onPushNodeConfig()
		if !ok {
			return false
		}
		refs, hasRefs := config["refs"]
		if !hasRefs {
			return true // key dropped (pre-fix behavior); the autosave landed
		}
		list, ok := refs.([]any)
		return ok && len(list) == 0
	}, 15*time.Second, 200*time.Millisecond, "the deletion autosave should update the staged config")

	samples := make([]int, 0, 25)
	phantomSeen := false
	for i := 0; i < 25; i++ {
		count, err := s.refsValueTextarea().Count()
		if err == nil {
			samples = append(samples, count)
			if count > 0 {
				phantomSeen = true
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	s.t.Logf("REFS ROW COUNT SAMPLES AFTER DELETE (100ms apart): %v", samples)
	require.False(s.t, phantomSeen,
		"no predicate row may reappear after deleting the last condition (samples: %v)", samples)

	s.session.AssertVisible(q.Locator(fmt.Sprintf(`text=%s`, requiredConditionHint)))
	s.takeNamedScreenshot("refs-fixed-1-empty-list-with-hint")
}

// thenTheStagedConfigKeepsRefsAsAnEmptyList pins the fix's contract: the refs
// key must stay PRESENT as an empty list. A present-but-empty value overrides
// the field default when the settings form recomputes its local state, which is
// what keeps the phantom row from coming back.
func (s *refsRequiredConditionSteps) thenTheStagedConfigKeepsRefsAsAnEmptyList() {
	require.Eventually(s.t, func() bool {
		list, ok := s.refsPredicates()
		return ok && len(list) == 0
	}, 15*time.Second, 200*time.Millisecond,
		"staged config should keep the refs key as an empty list after deleting the last condition")
}

func (s *refsRequiredConditionSteps) whenIAddARefsCondition(value string) {
	s.session.Click(q.Locator(`button:has-text("Add Condition")`))
	textarea := s.refsValueTextarea()
	require.NoError(s.t, textarea.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(15000),
	}))
	require.NoError(s.t, textarea.Fill(value))
	// Blur the value input so the pending edit autosaves.
	s.session.Click(q.TestID("node-name-input"))
	s.takeNamedScreenshot("refs-fixed-2-condition-restored")
}

func (s *refsRequiredConditionSteps) thenTheStagedConfigContainsTheNewConditionAndTheHintClears() {
	require.Eventually(s.t, func() bool {
		list, ok := s.refsPredicates()
		if !ok || len(list) != 1 {
			return false
		}
		entry, ok := list[0].(map[string]any)
		return ok && entry["value"] == "refs/heads/develop"
	}, 15*time.Second, 200*time.Millisecond,
		"staged config should contain the restored refs condition")

	hint := s.session.Page().Locator(fmt.Sprintf(`text=%s`, requiredConditionHint))
	require.Eventually(s.t, func() bool {
		count, err := hint.Count()
		return err == nil && count == 0
	}, 15*time.Second, 200*time.Millisecond, "the required hint should clear once a condition exists")
}

func (s *refsRequiredConditionSteps) whenICommitTheDraft() {
	s.canvas.CommitStaging()
}

func (s *refsRequiredConditionSteps) thenTheCommittedNodeHasNoError() {
	require.Eventually(s.t, func() bool {
		live, err := models.FindLiveCanvasVersion(s.canvas.WorkflowID)
		if err != nil {
			return false
		}
		for _, node := range live.Nodes {
			if node.ComponentName() != onPushTriggerName {
				continue
			}
			return node.ErrorMessage == nil
		}
		return false
	}, 15*time.Second, 200*time.Millisecond, "committed onPush node should carry no validation error")

	s.session.Sleep(1000)
	s.takeNamedScreenshot("refs-fixed-3-committed-clean")
}

func (s *refsRequiredConditionSteps) takeNamedScreenshot(name string) {
	path := fmt.Sprintf("/app/tmp/screenshots/%s.png", name)
	s.t.Logf("Taking screenshot: %s", path)
	if _, err := s.session.Page().Screenshot(pw.PageScreenshotOptions{
		Path:     pw.String(path),
		FullPage: pw.Bool(true),
		Type:     pw.ScreenshotTypePng,
	}); err != nil {
		s.t.Logf("screenshot error: %v", err)
	}
}
