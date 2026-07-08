package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"

	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestCanvasPage(t *testing.T) {
	t.Run("adding a node to canvas displays custom node name", func(t *testing.T) {
		steps := &CanvasPageSteps{t: t}
		steps.start()
		steps.givenACanvasExists()
		steps.addNoop("Hello")
		steps.assertNodeIsAdded("Hello")
	})

	t.Run("adding multiple nodes generates unique names", func(t *testing.T) {
		steps := &CanvasPageSteps{t: t}
		steps.start()
		steps.givenACanvasExists()

		// Add three noop nodes with auto-generated names
		name1 := steps.addNoopWithDefaultName(models.Position{X: 500, Y: 200})
		name2 := steps.addNoopWithDefaultName(models.Position{X: 500, Y: 400})
		name3 := steps.addNoopWithDefaultName(models.Position{X: 500, Y: 600})

		// First should be "noop", second "noop 2", third "noop 3"
		require.Equal(t, "noop", name1, "first node should be named 'noop'")
		require.Equal(t, "noop 2", name2, "second node should be named 'noop2'")
		require.Equal(t, "noop 3", name3, "third node should be named 'noop3'")

		// Verify all nodes exist on canvas
		steps.assertNodeIsAdded("noop")
		steps.assertNodeIsAdded("noop 2")
		steps.assertNodeIsAdded("noop 3")
	})

	// Note: "duplicating a node on canvas" test removed - duplicate action no longer available in UI

	t.Run("collapsing and expanding a node on canvas", func(t *testing.T) {
		steps := &CanvasPageSteps{t: t}
		steps.start()
		steps.givenACanvasExists()
		steps.addNoop("Hello")
		steps.assertIsNodeExpanded("Hello")
		steps.toggleNodeViewOnCanvas("Hello")
		steps.assertIsNodeCollapsed("Hello")
		steps.toggleNodeViewOnCanvas("Hello")
		steps.assertIsNodeExpanded("Hello")
	})

	t.Run("deleting a node from a canvas", func(t *testing.T) {
		steps := &CanvasPageSteps{t: t}
		steps.start()
		steps.givenACanvasExistsWithANoopNode()
		steps.deleteNodeFromCanvas("DeleteMe")
		steps.assertNodeDeletedInDB("DeleteMe")
	})

	t.Run("deleting a newly added node updates the canvas before publish", func(t *testing.T) {
		steps := &CanvasPageSteps{t: t}
		steps.start()
		steps.givenACanvasExists()
		steps.addManualTrigger("Start")
		steps.addNoop("DeleteMe")
		steps.deleteNodeFromCanvas("DeleteMe")
		steps.assertNodeIsHidden("DeleteMe")
		steps.publishCanvas()
		steps.assertNodeDeletedInDB("DeleteMe")
		steps.assertNodeIsAdded("Start")
	})

	t.Run("viewing queued items in the sidebar", func(t *testing.T) {
		steps := &CanvasPageSteps{t: t}
		steps.start()
		steps.givenACanvasWithManualTriggerAndWaitNodeAndQueuedItems(2)
		steps.openSidebarForNode("Wait")
		steps.assertRunningItemsCount("Wait", 1)
		steps.assertQueuedItemsCount("Wait", 1)
		steps.assertQueuedItemsVisibleInSidebar()
	})

	t.Run("canceling queued items from the sidebar", func(t *testing.T) {
		steps := &CanvasPageSteps{t: t}
		steps.start()
		steps.givenACanvasWithManualTriggerAndWaitNodeAndQueuedItems(2)
		steps.openSidebarForNode("Wait")

		steps.assertRunningItemsCount("Wait", 1)
		steps.assertQueuedItemsCount("Wait", 1)
		steps.cancelFirstQueueItemFromSidebar()
		steps.assertQueuedItemsCount("Wait", 0)
	})

	t.Run("canceling running execution from the sidebar", func(t *testing.T) {
		steps := &CanvasPageSteps{t: t}
		steps.start()
		steps.givenACanvasWithManualTriggerAndWaitNodeAndQueuedItems(1)
		steps.openSidebarForNode("Wait")

		steps.session.Sleep(1000)

		steps.assertRunningItemsCount("Wait", 1)
		steps.assertQueuedItemsCount("Wait", 0)
		steps.cancelRunningExecutionFromSidebar()
		steps.assertExecutionWasCancelled("Wait")
	})

	t.Run("deleting a connection between nodes", func(t *testing.T) {
		steps := &CanvasPageSteps{t: t}
		steps.start()
		steps.givenACanvasExists()
		steps.addTwoNodesAndConnect()
		steps.saveCanvas()
		steps.publishCanvas()
		steps.enterEditMode()
		steps.deleteConnectionBetweenNodes("First", "Second")
		steps.saveCanvas()
		steps.publishCanvas()
		steps.assertNodesAreNotConnectedInDB("First", "Second")
	})

	t.Run("autocomplete suggests node data in filter expression", func(t *testing.T) {
		steps := &CanvasPageSteps{t: t}
		steps.start()
		steps.givenACanvasExists()
		steps.addManualTrigger("Start")
		steps.addFilter("Filter")
		steps.connectNodes("Start", "Filter")
		steps.saveCanvas()
		steps.waitForDraftConnection("Start", "Filter")
		steps.openNodeSettings("Filter")
		steps.typeExpression("$")
		steps.assertAutocompleteNodeSuggestionVisible()
	})
}

func TestCanvasPageYamlViewer(t *testing.T) {
	t.Run("Files tab shows canvas YAML definition", func(t *testing.T) {
		steps := &CanvasPageSteps{t: t}
		steps.start()
		steps.givenACanvasExists()
		steps.addNoop("YamlTestNode")
		steps.openFilesTab()
		steps.assertFileIsOpen("canvas.yaml")
		steps.assertYamlContentVisible("YamlTestNode")
		steps.assertYamlContentVisible("metadata:")
	})

	t.Run("Files tab can return to canvas", func(t *testing.T) {
		steps := &CanvasPageSteps{t: t}
		steps.start()
		steps.givenACanvasExists()
		steps.addNoop("SwitchTest")
		steps.openFilesTab()
		steps.assertYamlContentVisible("SwitchTest")
		steps.returnToCanvasTab()
		steps.assertNodeIsAdded("SwitchTest")
	})
}

type CanvasPageSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *CanvasPageSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *CanvasPageSteps) givenACanvasExists() {
	s.canvas = shared.NewCanvasSteps("E2E Canvas", s.t, s.session)
	s.canvas.Create()
	s.canvas.EnterEditMode()
}

func (s *CanvasPageSteps) addNoop(name string) {
	s.canvas.AddNoop(name, models.Position{X: 500, Y: 200})
}

func (s *CanvasPageSteps) addManualTrigger(name string) {
	s.canvas.AddManualTrigger(name, models.Position{X: 500, Y: 200})
}

func (s *CanvasPageSteps) addFilter(name string) {
	s.canvas.AddFilter(name, models.Position{X: 900, Y: 200})
}

func (s *CanvasPageSteps) addNoopWithDefaultName(pos models.Position) string {
	return s.canvas.AddNoopWithDefaultName(pos)
}

func (s *CanvasPageSteps) addTwoNodesAndConnect() {
	s.canvas.AddManualTrigger("First", models.Position{X: 500, Y: 200})
	s.canvas.AddNoop("Second", models.Position{X: 900, Y: 200})
	s.canvas.Connect("First", "Second")
}

func (s *CanvasPageSteps) connectNodes(sourceName, targetName string) {
	s.canvas.Connect(sourceName, targetName)
}

func (s *CanvasPageSteps) saveCanvas() {
	s.canvas.Save()
}

func (s *CanvasPageSteps) publishCanvas() {
	s.canvas.CommitAndPublish()
}

func (s *CanvasPageSteps) enterEditMode() {
	s.canvas.EnterEditMode()
}

func (s *CanvasPageSteps) deleteConnectionBetweenNodes(sourceName, targetName string) {
	s.canvas.DeleteConnection(sourceName, targetName)
}

func (s *CanvasPageSteps) waitForDraftConnection(sourceName, targetName string) {
	require.Eventually(s.t, func() bool {
		nodes, edges := s.canvas.DraftEffectiveSpec()
		if len(nodes) == 0 {
			return false
		}

		sourceID := ""
		targetID := ""
		for _, node := range nodes {
			if node.Name == sourceName {
				sourceID = node.ID
			}
			if node.Name == targetName {
				targetID = node.ID
			}
		}
		if sourceID == "" || targetID == "" {
			return false
		}

		for _, edge := range edges {
			if edge.SourceID == sourceID && edge.TargetID == targetID {
				return true
			}
		}
		return false
	}, 10*time.Second, 200*time.Millisecond)
}

func (s *CanvasPageSteps) assertIsNodeCollapsed(nodeName string) {
	safe := strings.ToLower(nodeName)
	safe = strings.ReplaceAll(safe, " ", "-")
	selector := q.Locator(`[data-testid="node-` + safe + `-header"][data-view-mode="compact"]`)
	s.session.AssertVisible(selector)
}

func (s *CanvasPageSteps) assertIsNodeExpanded(nodeName string) {
	safe := strings.ToLower(nodeName)
	safe = strings.ReplaceAll(safe, " ", "-")
	selector := q.Locator(`[data-testid="node-` + safe + `-header"][data-view-mode="expanded"]`)
	s.session.AssertVisible(selector)
}

func (s *CanvasPageSteps) assertNodeIsAdded(nodeName string) {
	s.session.AssertText(nodeName)
}

func (s *CanvasPageSteps) assertNodeIsHidden(nodeName string) {
	s.session.AssertHidden(q.TestID("node", nodeName, "header"))
}

func (s *CanvasPageSteps) givenACanvasExistsWithANoopNode() {
	s.canvas = shared.NewCanvasSteps("E2E Canvas With Noop", s.t, s.session)

	s.canvas.Create()
	s.canvas.Visit()
	s.canvas.EnterEditMode()
	s.canvas.AddNoop("DeleteMe", models.Position{X: 500, Y: 200})
}

func (s *CanvasPageSteps) toggleNodeViewOnCanvas(nodeName string) {
	safe := strings.ToLower(nodeName)
	safe = strings.ReplaceAll(safe, " ", "-")
	node := q.Locator(`.react-flow__node:has([data-testid="node-` + safe + `-header"])`)
	toggleButton := q.Locator(
		`.react-flow__node:has([data-testid="node-` + safe + `-header"]) [data-testid="node-action-toggle-view"]`,
	)

	s.session.HoverOver(node)
	s.session.Sleep(100)
	s.session.Click(toggleButton)
	s.session.Sleep(300)
}

func (s *CanvasPageSteps) deleteNodeFromCanvas(nodeName string) {
	nodeHeader := q.TestID("node", nodeName, "header")
	safe := strings.ToLower(nodeName)
	safe = strings.ReplaceAll(safe, " ", "-")
	deleteButton := q.Locator(
		`.react-flow__node:has([data-testid="node-` + safe + `-header"]) [data-testid="node-action-delete"]`,
	)
	s.session.HoverOver(nodeHeader)
	s.session.Sleep(100)
	s.session.Click(deleteButton)
	s.session.Sleep(300)
}

func (s *CanvasPageSteps) assertNodeDeletedInDB(nodeName string) {
	deadline := time.Now().Add(2 * time.Second)

	for {
		canvas, err := models.FindCanvas(s.session.OrgID, s.canvas.WorkflowID)
		require.NoError(s.t, err)

		nodes, err := models.FindCanvasNodes(canvas.ID)
		require.NoError(s.t, err)

		exists := false
		for _, n := range nodes {
			if n.Name == nodeName {
				exists = true
				break
			}
		}

		if !exists {
			return
		}

		if time.Now().After(deadline) {
			s.t.Fatalf("expected node %q to be deleted, but it still exists in DB", nodeName)
		}

		time.Sleep(200 * time.Millisecond)
	}
}

func (s *CanvasPageSteps) givenACanvasWithManualTriggerAndWaitNodeAndQueuedItems(itemsAmount int) {
	s.canvas = shared.NewCanvasSteps("E2E Canvas With Queue", s.t, s.session)

	s.canvas.Create()
	s.canvas.EnterEditMode()
	s.canvas.AddManualTrigger("Start", models.Position{X: 600, Y: 200})
	s.canvas.AddWait("Wait", models.Position{X: 1000, Y: 200}, 10, "Seconds")
	s.session.TakeScreenshot()
	s.canvas.Connect("Start", "Wait")
	s.canvas.Save()
	s.canvas.CommitAndPublish()
	s.t.Cleanup(func() {
		s.cleanupQueuedWaitWork("Wait")
	})

	for i := 0; i < itemsAmount; i++ {
		s.canvas.EmitManualTrigger("Start")
		s.session.Sleep(100)
	}

	// wait for the first item to start processing
	s.session.Sleep(500)
}

func (s *CanvasPageSteps) cleanupQueuedWaitWork(nodeName string) {
	if s.canvas == nil {
		return
	}

	node, err := s.findNodeByName(nodeName)
	if err != nil {
		s.t.Logf("cleanup queued wait work: find node %q: %v", nodeName, err)
		return
	}
	if node == nil {
		s.t.Logf("cleanup queued wait work: node %q not found", nodeName)
		return
	}

	activeStates := []string{
		models.CanvasNodeExecutionStatePending,
		models.CanvasNodeExecutionStateStarted,
	}

	var executions []models.CanvasNodeExecution
	err = database.Conn().
		Where("workflow_id = ?", node.WorkflowID).
		Where("node_id = ?", node.NodeID).
		Where("state IN ?", activeStates).
		Find(&executions).
		Error
	if err != nil {
		s.t.Logf("cleanup queued wait work: find executions: %v", err)
		return
	}

	for i := range executions {
		if err := executions[i].Cancel(nil); err != nil {
			s.t.Logf("cleanup queued wait work: cancel execution %s: %v", executions[i].ID, err)
		}
	}

	if err := database.Conn().
		Where("workflow_id = ?", node.WorkflowID).
		Where("node_id = ?", node.NodeID).
		Delete(&models.CanvasNodeQueueItem{}).
		Error; err != nil {
		s.t.Logf("cleanup queued wait work: delete queue items: %v", err)
	}

	if err := database.Conn().
		Where("workflow_id = ?", node.WorkflowID).
		Where("node_id = ?", node.NodeID).
		Where("state = ?", models.NodeExecutionRequestStatePending).
		Delete(&models.CanvasNodeRequest{}).
		Error; err != nil {
		s.t.Logf("cleanup queued wait work: delete node requests: %v", err)
	}
}

func (s *CanvasPageSteps) findNodeByName(nodeName string) (*models.CanvasNode, error) {
	canvas, err := models.FindCanvas(s.session.OrgID, s.canvas.WorkflowID)
	if err != nil {
		return nil, err
	}

	nodes, err := models.FindCanvasNodes(canvas.ID)
	if err != nil {
		return nil, err
	}

	for i := range nodes {
		if nodes[i].Name == nodeName {
			return &nodes[i], nil
		}
	}

	return nil, nil
}

func (s *CanvasPageSteps) openSidebarForNode(node string) {
	s.session.Click(q.TestID("node", node, "header"))
	s.session.AssertVisible(q.TestID("run-inspector-panel"))
}

func (s *CanvasPageSteps) openNodeSettings(node string) {
	s.canvas.StartEditingNode(node)
	s.session.Click(q.Text("Configuration"))
	s.session.Sleep(200)
}

func (s *CanvasPageSteps) typeExpression(value string) {
	s.session.FillIn(q.TestID("expression-field-expression"), value)
}

func (s *CanvasPageSteps) assertAutocompleteNodeSuggestionVisible() {
	s.session.AssertVisible(q.Locator(`div[data-suggestion-index="0"]`))
	s.session.AssertVisible(q.Locator(`div[data-suggestion-index="0"] span:has-text("node")`))
}

func (s *CanvasPageSteps) assertQueuedItemsCount(nodeName string, expected int) {
	deadline := time.Now().Add(10 * time.Second)
	lastCount := -1

	for time.Now().Before(deadline) {
		canvas, err := models.FindCanvas(s.session.OrgID, s.canvas.WorkflowID)
		require.NoError(s.t, err)

		nodes, err := models.FindCanvasNodes(canvas.ID)
		require.NoError(s.t, err)

		var waitNode *models.CanvasNode
		for _, n := range nodes {
			if n.Name == nodeName {
				waitNode = &n
				break
			}
		}
		require.NotNil(s.t, waitNode, nodeName+" node not found")

		queueItems, err := models.ListNodeQueueItems(waitNode.WorkflowID, waitNode.NodeID, 100, nil)
		require.NoError(s.t, err)

		lastCount = len(queueItems)
		if lastCount == expected {
			return
		}

		time.Sleep(250 * time.Millisecond)
	}

	require.Equal(s.t, expected, lastCount)
}

func (s *CanvasPageSteps) assertRunningItemsCount(nodeName string, expected int) {
	deadline := time.Now().Add(10 * time.Second)
	lastCount := -1

	for time.Now().Before(deadline) {
		canvas, err := models.FindCanvas(s.session.OrgID, s.canvas.WorkflowID)
		require.NoError(s.t, err)

		nodes, err := models.FindCanvasNodes(canvas.ID)
		require.NoError(s.t, err)

		var waitNode *models.CanvasNode
		for _, n := range nodes {
			if n.Name == nodeName {
				waitNode = &n
				break
			}
		}
		require.NotNil(s.t, waitNode, nodeName+" node not found")

		var executions []models.CanvasNodeExecution
		query := database.Conn().
			Where("workflow_id = ?", waitNode.WorkflowID).
			Where("node_id = ?", waitNode.NodeID).
			Order("created_at DESC")

		err = query.Find(&executions).Error
		require.NoError(s.t, err)

		lastCount = len(executions)
		if lastCount == expected {
			return
		}

		time.Sleep(250 * time.Millisecond)
	}

	require.Equal(s.t, expected, lastCount)
}

func (s *CanvasPageSteps) assertQueuedItemsVisibleInSidebar() {
	s.session.AssertVisible(q.TestID("run-inspector-panel"))
	s.session.AssertVisible(q.Locator(`[data-testid="run-inspector-panel"] button:has-text("Stop")`))
}

func (s *CanvasPageSteps) cancelFirstQueueItemFromSidebar() {
	s.stopRunFromInspector()
}

func (s *CanvasPageSteps) cancelRunningExecutionFromSidebar() {
	s.stopRunFromInspector()
}

func (s *CanvasPageSteps) stopRunFromInspector() {
	s.session.Click(q.Locator(`[data-testid="run-inspector-panel"] button:has-text("Stop")`))
	s.session.Sleep(500) // wait for the cancellation to be processed
}

func (s *CanvasPageSteps) assertExecutionWasCancelled(nodeName string) {
	executions := s.canvas.GetExecutionsForNode(nodeName)
	require.Greater(s.t, len(executions), 0, "expected at least one execution")

	execution := executions[0]
	require.Equal(s.t, models.CanvasNodeExecutionResultCancelled, execution.Result, "expected execution to be cancelled")
}

func (s *CanvasPageSteps) assertNodesAreNotConnectedInDB(sourceName, targetName string) {
	deadline := time.Now().Add(5 * time.Second)

	for {
		workflow := s.canvas.GetWorkflowFromDB()
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.Conn(), workflow)
		require.NoError(s.t, err)
		sourceNode := s.canvas.GetNodeFromDB(sourceName)
		targetNode := s.canvas.GetNodeFromDB(targetName)

		connected := false
		for _, conn := range liveVersion.Edges {
			if conn.SourceID == sourceNode.NodeID && conn.TargetID == targetNode.NodeID {
				connected = true
				break
			}
		}

		if !connected {
			return
		}

		if time.Now().After(deadline) {
			s.t.Fatalf("expected nodes %q and %q to not be connected, but connection exists in DB", sourceName, targetName)
		}

		time.Sleep(200 * time.Millisecond)
	}
}

func (s *CanvasPageSteps) openFilesTab() {
	s.canvas.Save()
	s.canvas.ClickOnEmptyCanvasArea()
	s.session.Sleep(300)
	filesTab := q.TestID("canvas-view-mode-files")
	s.session.AssertVisible(filesTab)
	s.session.Click(filesTab)
	s.session.AssertVisible(q.TestID("files-overlay"))
	s.session.AssertVisible(q.TestID("file-editor"))
	s.waitForMonacoEditor()
}

func (s *CanvasPageSteps) returnToCanvasTab() {
	s.session.Click(q.TestID("canvas-view-mode-live"))
	s.session.Sleep(500)
}

func (s *CanvasPageSteps) assertFileIsOpen(name string) {
	s.session.AssertText(name)
	s.session.AssertVisible(q.TestID("file-editor"))
}

func (s *CanvasPageSteps) assertYamlContentVisible(text string) {
	s.waitForMonacoEditor()
	s.session.AssertVisible(q.Locator(fmt.Sprintf(`[data-testid="file-editor"] >> text=%s`, text)))
}

func (s *CanvasPageSteps) waitForMonacoEditor() {
	monacoLines := q.Locator(`[data-testid="file-editor"] .view-lines`)
	if err := monacoLines.Run(s.session).WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(15000),
	}); err != nil {
		s.t.Fatalf("monaco editor did not become ready: %v", err)
	}
}
