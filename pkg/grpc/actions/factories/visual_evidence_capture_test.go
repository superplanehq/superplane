package factories

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestStartVisualEvidenceCaptureEmitsTheAttachedRevision(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, order, pullRequest, node := visualEvidenceCaptureFixture(t, r)
	const headSHA = "0123456789abcdef0123456789abcdef01234567"
	restore := stubVisualEvidenceGitHub(t, openVisualEvidencePullRequest(headSHA), nil)
	defer restore()

	require.NoError(t, startVisualEvidenceCapture(t.Context(), IntakeDependencies{}, r.Organization.ID, factoryModel.ID, pullRequest.ID))
	require.NoError(t, startVisualEvidenceCapture(t.Context(), IntakeDependencies{}, r.Organization.ID, factoryModel.ID, pullRequest.ID))

	var events []models.CanvasEvent
	require.NoError(t, db.Where("workflow_id = ? AND node_id = ?", node.WorkflowID, node.NodeID).Find(&events).Error)
	require.Len(t, events, 1)
	payload, ok := models.RootEventSourcePayload(events[0].Data.Data()).(map[string]any)
	require.True(t, ok)
	pullRequestPayload, ok := payload["pull_request"].(map[string]any)
	require.True(t, ok)
	head, ok := pullRequestPayload["head"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, headSHA, head["sha"])
	assert.Equal(t, order.ID.String(), payload["workOrder"].(map[string]any)["id"])
}

func TestStartVisualEvidenceCaptureRetriesAfterGitHubFailure(t *testing.T) {
	r := support.Setup(t)
	factoryModel, _, pullRequest, node := visualEvidenceCaptureFixture(t, r)
	const headSHA = "0123456789abcdef0123456789abcdef01234567"
	calls := 0
	restore := stubVisualEvidenceGitHubFetch(t, func() (*github.PullRequest, error) {
		calls++
		pending := countVisualEvidenceCaptureRequests(t, node, models.NodeExecutionRequestStatePending)
		if calls == 1 {
			assert.Equal(t, int64(1), pending)
			return nil, errors.New("github unavailable")
		}
		return openVisualEvidencePullRequest(headSHA), nil
	})
	defer restore()

	require.NoError(t, startVisualEvidenceCapture(t.Context(), IntakeDependencies{}, r.Organization.ID, factoryModel.ID, pullRequest.ID))

	assert.Equal(t, 2, calls)
	assert.Equal(t, int64(1), countCanvasEvents(t, node.WorkflowID))
	assert.Equal(t, int64(0), countVisualEvidenceCaptureRequests(t, node, models.NodeExecutionRequestStatePending))
	assert.Equal(t, int64(1), countVisualEvidenceCaptureRequests(t, node, models.NodeExecutionRequestStateCompleted))
}

func TestStartVisualEvidenceCaptureKeepsTheRequestWhenGitHubStaysDown(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, _, pullRequest, node := visualEvidenceCaptureFixture(t, r)
	const headSHA = "0123456789abcdef0123456789abcdef01234567"
	calls := 0
	restore := stubVisualEvidenceGitHubFetch(t, func() (*github.PullRequest, error) {
		calls++
		assert.Equal(t, int64(1), countVisualEvidenceCaptureRequests(t, node, models.NodeExecutionRequestStatePending))
		return nil, errors.New("github unavailable")
	})
	defer restore()

	err := startVisualEvidenceCapture(t.Context(), IntakeDependencies{}, r.Organization.ID, factoryModel.ID, pullRequest.ID)
	require.Error(t, err)
	assert.Equal(t, visualEvidenceCaptureAttempts, calls)
	assert.Equal(t, int64(0), countCanvasEvents(t, node.WorkflowID))
	assert.Equal(t, int64(1), countVisualEvidenceCaptureRequests(t, node, models.NodeExecutionRequestStatePending))

	restoreSuccess := stubVisualEvidenceGitHub(t, openVisualEvidencePullRequest(headSHA), nil)
	defer restoreSuccess()
	require.NoError(t, db.Model(&models.CanvasNodeRequest{}).
		Where("workflow_id = ? AND node_id = ? AND type = ?", node.WorkflowID, node.NodeID, models.NodeRequestTypeVisualEvidenceCapture).
		Update("run_at", time.Now().Add(-time.Second)).Error)

	_, err = ProcessVisualEvidenceCaptureRequest(t.Context(), IntakeDependencies{}, visualEvidenceCaptureRequestID(t, node))
	require.NoError(t, err)
	assert.Equal(t, int64(1), countCanvasEvents(t, node.WorkflowID))
	assert.Equal(t, int64(1), countVisualEvidenceCaptureRequests(t, node, models.NodeExecutionRequestStateCompleted))
}

func TestStartVisualEvidenceCaptureSkipsDraftsAndBrokenCanvases(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, _, pullRequest, node := visualEvidenceCaptureFixture(t, r)
	restore := stubVisualEvidenceGitHub(t, &github.PullRequest{
		Draft: github.Ptr(true),
		State: github.Ptr("open"),
		Head:  &github.PullRequestBranch{SHA: github.Ptr("0123456789abcdef0123456789abcdef01234567")},
		Base:  &github.PullRequestBranch{Ref: github.Ptr("main")},
	}, nil)
	defer restore()

	require.NoError(t, startVisualEvidenceCapture(t.Context(), IntakeDependencies{}, r.Organization.ID, factoryModel.ID, pullRequest.ID))
	assert.Equal(t, int64(0), countCanvasEvents(t, node.WorkflowID))

	require.NoError(t, db.Model(&models.CanvasNode{}).
		Where("workflow_id = ? AND node_id = ?", node.WorkflowID, node.NodeID).
		Update("state", models.CanvasNodeStateError).Error)
	restoreOpen := stubVisualEvidenceGitHub(t, openVisualEvidencePullRequest("0123456789abcdef0123456789abcdef01234567"), nil)
	defer restoreOpen()
	require.NoError(t, startVisualEvidenceCapture(t.Context(), IntakeDependencies{}, r.Organization.ID, factoryModel.ID, pullRequest.ID))
	assert.Equal(t, int64(0), countCanvasEvents(t, node.WorkflowID))
}

func openVisualEvidencePullRequest(headSHA string) *github.PullRequest {
	return &github.PullRequest{
		Draft: github.Ptr(false),
		State: github.Ptr("open"),
		Head:  &github.PullRequestBranch{SHA: github.Ptr(headSHA)},
		Base:  &github.PullRequestBranch{Ref: github.Ptr("main")},
	}
}

func stubVisualEvidenceGitHub(t *testing.T, pullRequest *github.PullRequest, fetchErr error) func() {
	t.Helper()
	return stubVisualEvidenceGitHubFetch(t, func() (*github.PullRequest, error) {
		return pullRequest, fetchErr
	})
}

func stubVisualEvidenceGitHubFetch(t *testing.T, fetch func() (*github.PullRequest, error)) func() {
	t.Helper()
	previousFetch := fetchVisualEvidencePullRequest
	previousPublish := publishVisualEvidenceEvent
	fetchVisualEvidencePullRequest = func(
		context.Context,
		*gorm.DB,
		IntakeDependencies,
		*models.Factory,
		string,
		int,
	) (*github.PullRequest, error) {
		return fetch()
	}
	publishVisualEvidenceEvent = func(*models.CanvasEvent) error { return nil }
	return func() {
		fetchVisualEvidencePullRequest = previousFetch
		publishVisualEvidenceEvent = previousPublish
	}
}

func visualEvidenceCaptureFixture(
	t *testing.T,
	r *support.ResourceRegistry,
) (*models.Factory, *models.FactoryWorkOrder, *models.FactoryPullRequest, *models.CanvasNode) {
	t.Helper()
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Show checkout", "Show the checkout state", &r.User, nil, nil)
	require.NoError(t, err)
	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		Provider:   models.FactoryPullRequestProviderGitHub,
		Repository: "acme/app",
		Number:     42,
		URL:        "https://github.com/acme/app/pull/42",
		Title:      "Show checkout",
		State:      models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)

	canvas, nodes := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: "on-pr-visual-evidence",
		Name:   "On Pull Request",
		Type:   models.NodeTypeTrigger,
		Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata(
			models.FactoryAppTemplateVisualEvidenceID,
			1,
		)),
	}}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", factoryModel.ID).Error)
	return factoryModel, order, pullRequest, &nodes[0]
}

func countCanvasEvents(t *testing.T, canvasID uuid.UUID) int64 {
	t.Helper()
	var count int64
	require.NoError(t, database.DB(t.Context()).Model(&models.CanvasEvent{}).Where("workflow_id = ?", canvasID).Count(&count).Error)
	return count
}

func countVisualEvidenceCaptureRequests(t *testing.T, node *models.CanvasNode, state string) int64 {
	t.Helper()
	var count int64
	require.NoError(t, database.DB(t.Context()).Model(&models.CanvasNodeRequest{}).
		Where("workflow_id = ? AND node_id = ? AND type = ? AND state = ?", node.WorkflowID, node.NodeID, models.NodeRequestTypeVisualEvidenceCapture, state).
		Count(&count).Error)
	return count
}

func visualEvidenceCaptureRequestID(t *testing.T, node *models.CanvasNode) uuid.UUID {
	t.Helper()
	var request models.CanvasNodeRequest
	require.NoError(t, database.DB(t.Context()).
		Where("workflow_id = ? AND node_id = ? AND type = ?", node.WorkflowID, node.NodeID, models.NodeRequestTypeVisualEvidenceCapture).
		First(&request).Error)
	return request.ID
}
