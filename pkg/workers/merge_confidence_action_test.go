package workers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	factorycomponent "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestNodeRequestExecutorDispatchesMergeConfidencePublication(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{NodeID: "assessment", Type: models.NodeTypeTrigger},
		{NodeID: "publication", Type: models.NodeTypeTrigger},
		{NodeID: "publish", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: factorycomponent.PublishMergeConfidenceComponentName}})},
	}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
	source, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "assessment", models.CanvasRunStateFinished, "")
	require.NoError(t, err)
	event := support.EmitCanvasEventForNode(t, canvas.ID, "publication", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "publish", event.ID, event.ID)
	execution.State = models.CanvasNodeExecutionStateStarted
	execution.Configuration = datatypes.NewJSONType(map[string]any{"sourceRunId": source.ID.String()})
	require.NoError(t, db.Save(execution).Error)
	require.NoError(t, (&factorycomponent.PublishMergeConfidence{}).Execute(core.ExecutionContext{
		Configuration: execution.Configuration.Data(), Metadata: contexts.NewExecutionMetadataContext(db, execution), Requests: contexts.NewExecutionRequestContext(db, execution),
	}))
	var request models.CanvasNodeRequest
	require.NoError(t, db.Where("execution_id = ?", execution.ID).First(&request).Error)
	worker := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", nil)
	require.NoError(t, worker.RunNow(request))
	require.NoError(t, worker.RunNow(request))
	require.NoError(t, db.First(execution, "id = ?", execution.ID).Error)
	assert.Equal(t, models.CanvasNodeExecutionResultPassed, execution.Result)
	assert.Equal(t, "skipped", execution.Metadata.Data()["status"])
	assert.NotEmpty(t, execution.Metadata.Data()["reason"])
}
