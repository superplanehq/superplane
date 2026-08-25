package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	linesRunFitTriggerNodeID = "kickoff"
	// Factory run inspection re-lays a linear run out in a single vertical
	// spine from a fixed, small origin (see layoutFactoryRunLeafGraph),
	// independent of any saved editor position. A dozen chained steps stack
	// past the compact run canvas at the default (un-fit) zoom, so only
	// an explicit participant fit brings the last one into view.
	linesRunFitStepCount = 12
)

func linesRunFitStepNodeID(index int) string {
	return fmt.Sprintf("step-%d", index)
}

// TestLinesPhaseRunFitsIntoView guards against a regression where the
// automation run page showed the compact run canvas with whatever
// default/empty viewport ReactFlow happened to initialize with, instead of
// framing the run's participant nodes. See PR #6715. The work-order popup
// on the line board no longer embeds this canvas.
func TestLinesPhaseRunFitsIntoView(t *testing.T) {
	t.Run("opening the automation run page fits its participant nodes into view", func(t *testing.T) {
		steps := &linesRunFitSteps{t: t}

		steps.start()
		steps.givenAFactory()
		steps.givenAFactoryAppWithAWideParticipantChain()
		steps.givenALineDispatchedForThatApp()
		steps.whenIOpenTheAutomationRunPage()
		steps.thenTheFirstAndLastParticipantsFitIntoView()
	})
}

type linesRunFitSteps struct {
	t       *testing.T
	session *session.TestSession

	canvas  *models.Canvas
	factory *models.Factory
	line    *models.FactoryLine
	runID   uuid.UUID
}

func (s *linesRunFitSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
	require.NoError(s.t, models.EnableExperimentalFeature(s.session.OrgID, features.FeatureFactories))
}

func (s *linesRunFitSteps) givenAFactory() {
	tx := database.DB(s.t.Context())
	factory, err := models.CreateFactory(tx, s.session.OrgID, support.RandomName("factory"), "", "")
	require.NoError(s.t, err)
	support.CompleteFactoryOnboarding(s.t, factory)
	s.factory = factory
}

// Builds an app with a trigger followed by a long, linear chain of
// components. Factory run inspection re-lays these out into a single spine
// from a fixed, small origin regardless of their saved positions (an
// ephemeral, display-only layout — see layoutFactoryRunLeafGraph), so it's
// the chain length, not the stored coordinates, that pushes the last node
// off the default view.
func (s *linesRunFitSteps) givenAFactoryAppWithAWideParticipantChain() {
	user, err := models.FindMaybeDeletedUserByEmail(s.session.OrgID.String(), s.session.Account.Email)
	require.NoError(s.t, err)

	nodes := []models.CanvasNode{
		{
			NodeID: linesRunFitTriggerNodeID,
			Name:   "Kickoff",
			Type:   models.NodeTypeTrigger,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "onRun"},
			}),
			Configuration: datatypes.NewJSONType(map[string]any{
				"parameters": []any{},
			}),
			Position: datatypes.NewJSONType(models.Position{X: 0, Y: 0}),
		},
	}

	var edges []models.Edge
	previousNodeID := linesRunFitTriggerNodeID
	for i := 0; i < linesRunFitStepCount; i++ {
		nodeID := linesRunFitStepNodeID(i)
		nodes = append(nodes, models.CanvasNode{
			NodeID: nodeID,
			Name:   fmt.Sprintf("Step %d", i),
			Type:   models.NodeTypeComponent,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: "noop"},
			}),
			Configuration: datatypes.NewJSONType(map[string]any{}),
			Position:      datatypes.NewJSONType(models.Position{X: (i + 1) * 200, Y: 0}),
		})
		edges = append(edges, models.Edge{SourceID: previousNodeID, TargetID: nodeID, Channel: "default"})
		previousNodeID = nodeID
	}

	canvas, _ := support.CreateCanvas(s.t, s.session.OrgID, user.ID, nodes, edges)

	// The Factory app canvas page redirects away unless the canvas is
	// associated with the current factory, so link it up explicitly —
	// support.CreateCanvas builds a plain (non-factory) canvas.
	require.NoError(s.t, database.Conn().
		Model(&models.Canvas{}).
		Where("id = ?", canvas.ID).
		Update("factory_id", s.factory.ID).
		Error)

	s.canvas = canvas
}

// Builds a factory line with a single "Build" phase backed by the app above,
// dispatches a work order onto it, and seeds the run/executions the phase
// run card links to — without needing a full temporal worker run. Every
// chain node gets its own execution row, so every one of them is a run
// participant.
func (s *linesRunFitSteps) givenALineDispatchedForThatApp() {
	line, err := s.factory.CreateLine(database.Conn(), support.RandomName("line"), []models.FactoryLineStep{
		{
			Type:       models.FactoryLineStepTypeRunApp,
			AppID:      s.canvas.ID,
			Entrypoint: linesRunFitTriggerNodeID,
		},
	})
	require.NoError(s.t, err)
	s.line = line

	order, err := s.factory.CreateWorkOrder(database.Conn(), support.RandomName("work order"), "", nil, nil, nil)
	require.NoError(s.t, err)
	require.NoError(s.t, order.TransitionOnDispatch(database.Conn(), nil))

	var result *models.FactoryLineStepResult
	require.NoError(s.t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var dispatchErr error
		_, result, dispatchErr = line.Dispatch(tx, order)
		return dispatchErr
	}))
	require.NotNil(s.t, result)
	require.NotNil(s.t, result.Execution)
	require.NotNil(s.t, result.Run)
	s.runID = result.Run.ID

	createdAt := time.Now()
	rootEvent := models.CanvasEvent{
		ID:         uuid.New(),
		WorkflowID: s.canvas.ID,
		NodeID:     linesRunFitTriggerNodeID,
		Channel:    "default",
		Data:       models.NewJSONValue(map[string]any{"message": "kickoff"}),
		RunID:      s.runID,
		State:      models.CanvasEventStateRouted,
		CreatedAt:  &createdAt,
	}
	require.NoError(s.t, database.Conn().Create(&rootEvent).Error)

	for i := 0; i < linesRunFitStepCount; i++ {
		execution := models.CanvasNodeExecution{
			ID:            uuid.New(),
			WorkflowID:    s.canvas.ID,
			NodeID:        linesRunFitStepNodeID(i),
			RootEventID:   rootEvent.ID,
			RunID:         s.runID,
			EventID:       rootEvent.ID,
			State:         models.CanvasNodeExecutionStateStarted,
			Metadata:      datatypes.NewJSONType(map[string]any{}),
			Configuration: datatypes.NewJSONType(map[string]any{}),
			CreatedAt:     &createdAt,
			UpdatedAt:     &createdAt,
		}
		require.NoError(s.t, database.Conn().Create(&execution).Error)
	}
}

func (s *linesRunFitSteps) whenIOpenTheAutomationRunPage() {
	s.session.Visit("/" + s.session.OrgID.String() + "/workspaces/" + s.factory.Key +
		"/apps/" + s.canvas.ID.String() + "/split-run?from=lines&lineId=" + s.line.ID.String() +
		"&run=" + s.runID.String())
	s.session.AssertVisible(q.TestID("factory-app-split-run-page"))
	s.session.AssertVisible(q.TestID("run-overlay-compact-canvas"))
}

func (s *linesRunFitSteps) thenTheFirstAndLastParticipantsFitIntoView() {
	s.assertNodeFitsIntoView(linesRunFitTriggerNodeID)
	s.assertNodeFitsIntoView(linesRunFitStepNodeID(linesRunFitStepCount - 1))
}

// Waits for the node card to render, then polls its bounding box until the
// compact canvas has panned/zoomed it into the run pane — the observable
// effect of the participant-fit request this test guards. Targets the
// compact-canvas test id rather than a label, since every chain node past
// the first shares the same "No Operation" component label.
func (s *linesRunFitSteps) assertNodeFitsIntoView(nodeID string) {
	canvas := q.TestID("run-overlay-compact-canvas").Run(s.session)
	locator := canvas.Locator(fmt.Sprintf(`.react-flow__node[data-id="%s"]`, nodeID))
	require.NoError(s.t, locator.WaitFor(pw.LocatorWaitForOptions{
		State:   pw.WaitForSelectorStateVisible,
		Timeout: pw.Float(15000),
	}))

	var lastCanvas, lastNode string
	fitted := assert.Eventually(s.t, func() bool {
		canvasBox, canvasErr := canvas.BoundingBox()
		box, err := locator.BoundingBox()
		if canvasErr != nil || canvasBox == nil || err != nil || box == nil {
			lastCanvas, lastNode = fmt.Sprintf("%v", canvasErr), fmt.Sprintf("%v", err)
			return false
		}

		lastCanvas = fmt.Sprintf("x=%.1f y=%.1f w=%.1f h=%.1f", canvasBox.X, canvasBox.Y, canvasBox.Width, canvasBox.Height)
		lastNode = fmt.Sprintf("x=%.1f y=%.1f w=%.1f h=%.1f", box.X, box.Y, box.Width, box.Height)
		centerX := box.X + box.Width/2
		centerY := box.Y + box.Height/2
		return centerX >= canvasBox.X && centerX <= canvasBox.X+canvasBox.Width &&
			centerY >= canvasBox.Y && centerY <= canvasBox.Y+canvasBox.Height
	}, 10*time.Second, 200*time.Millisecond)
	if !fitted {
		s.t.Fatalf("expected node %q to be panned/zoomed into the compact canvas (canvas %s, node %s)", nodeID, lastCanvas, lastNode)
	}
}
