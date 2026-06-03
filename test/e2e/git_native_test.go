package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
	"github.com/superplanehq/superplane/test/e2e/shared"
)

func TestGitNativeCanvas(t *testing.T) {
	t.Run("edit commit publish and run workflow", func(t *testing.T) {
		steps := &gitNativeCanvasSteps{t: t}
		steps.Start()
		steps.givenACanvasExists()
		steps.whenIAddNoopAndPublish()
		steps.thenLiveVersionHasNoopNode()
	})

	t.Run("first edit creates draft then menu shows continue", func(t *testing.T) {
		steps := &gitNativeCanvasSteps{t: t}
		steps.Start()
		steps.givenACanvasExists()
		steps.whenIClickEdit()
		steps.thenIAmEditing()
		steps.whenIExitEditMode()
		steps.whenIClickEdit()
		steps.thenStartEditingMenuShowsContinueDraft()
	})
}

type gitNativeCanvasSteps struct {
	t       *testing.T
	session *session.TestSession
	canvas  *shared.CanvasSteps
}

func (s *gitNativeCanvasSteps) Start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *gitNativeCanvasSteps) givenACanvasExists() {
	s.canvas = shared.NewCanvasSteps("Git Native Canvas", s.t, s.session)
	s.canvas.Create()
}

func (s *gitNativeCanvasSteps) whenIAddNoopAndPublish() {
	s.canvas.EnterEditMode()
	s.canvas.AddNoop("Git Native Noop", models.Position{X: 500, Y: 200})
	s.canvas.Save()
	s.canvas.Publish()
}

func (s *gitNativeCanvasSteps) thenLiveVersionHasNoopNode() {
	canvas, err := models.FindCanvasWithoutOrgScope(s.canvas.WorkflowID)
	require.NoError(s.t, err)
	require.NotNil(s.t, canvas.LiveVersionID)

	liveVersion, err := models.FindCanvasVersion(s.canvas.WorkflowID, *canvas.LiveVersionID)
	require.NoError(s.t, err)

	found := false
	for _, node := range liveVersion.Nodes {
		if node.Name == "Git Native Noop" {
			found = true
			break
		}
	}
	require.True(s.t, found, "expected live version to include Git Native Noop node")
}

func (s *gitNativeCanvasSteps) whenIClickEdit() {
	s.canvas.Visit()
	s.canvas.ClickEditButton()
}

func (s *gitNativeCanvasSteps) thenIAmEditing() {
	s.session.AssertVisible(q.TestID("canvas-exit-edit-button"))
}

func (s *gitNativeCanvasSteps) whenIExitEditMode() {
	s.canvas.ExitEditMode()
}

func (s *gitNativeCanvasSteps) thenStartEditingMenuShowsContinueDraft() {
	s.session.AssertVisible(q.TestID("start-editing-menu"))
	s.session.AssertVisible(q.TestID("start-editing-continue"))
	s.session.AssertVisible(q.TestID("start-editing-create"))
}
