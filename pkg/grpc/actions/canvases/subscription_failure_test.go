package canvases

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestCanvasRunAPIsPreserveSubscriptionFailureWithoutExecutionOutputs(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}}, nil)
	event := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	run := createFinishedRun(t, event, models.CanvasRunResultFailed)
	execution := createRunExecution(t, run, event.ID, "trigger", models.CanvasNodeExecutionResultFailed)
	fleet := models.DefaultInstallationRunnerFleets("test")[0]
	fleet.Slug = support.RandomName("quota")
	require.NoError(t, fleet.Create(db))
	task := &models.RunnerTask{ID: uuid.New(), OrganizationID: r.Organization.ID, FleetID: fleet.ID, Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateFailed, QueuedAt: time.Now(), PayloadCiphertext: []byte("encrypted"), Result: datatypes.JSON(`{"usage_state":"partial","failure":{"reason":"chatgpt_quota_exhausted","subscription_owner_id":"` + r.User.String() + `"}}`)}
	require.NoError(t, db.Create(task).Error)
	require.NoError(t, models.CreateNodeExecutionKVInTransaction(db, canvas.ID, execution.NodeID, execution.ID, "task_id", task.ID.String()))

	listed, err := ListRuns(t.Context(), db, canvas, &pb.ListRunsRequest{})
	require.NoError(t, err)
	require.Len(t, listed.Runs, 1)
	described, err := DescribeRun(t.Context(), db, canvas, run.ID.String(), false)
	require.NoError(t, err)
	for _, saved := range []*pb.CanvasRun{listed.Runs[0], described.Run} {
		assert.Equal(t, pb.CanvasRun_RESULT_FAILED, saved.Result)
		assert.Equal(t, "partial", saved.UsageState)
		require.NotNil(t, saved.SubscriptionFailure)
		assert.Equal(t, models.SubscriptionFailureChatGPTQuotaExhausted, saved.SubscriptionFailure.Reason)
		assert.Equal(t, r.User.String(), saved.SubscriptionFailure.SubscriptionOwnerId)
	}
}
