package workers

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
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

func TestNodeRequestExecutorDispatchesBodySectionPublication(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{NodeID: "assessment", Type: models.NodeTypeTrigger},
		{NodeID: "publish", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: factorycomponent.UpdatePullRequestBodySectionComponentName}})},
	}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
	root := support.EmitCanvasEventForNode(t, canvas.ID, "assessment", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "publish", root.ID, root.ID)
	execution.Configuration = datatypes.NewJSONType(map[string]any{"pullRequestId": uuid.NewString(), "sectionKey": "release-notes", "body": "arbitrary"})
	execution.State = models.CanvasNodeExecutionStateStarted
	require.NoError(t, db.Save(execution).Error)
	require.NoError(t, (&factorycomponent.UpdatePullRequestBodySection{}).Execute(core.ExecutionContext{
		RunID: execution.RunID, Configuration: execution.Configuration.Data(), Metadata: contexts.NewExecutionMetadataContext(db, execution), Requests: contexts.NewExecutionRequestContext(db, execution),
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

func TestNodeRequestExecutorDispatchesIntegrationBodySectionWithoutFactory(t *testing.T) {
	for _, provider := range []string{"github", "bitbucket"} {
		for _, enabled := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/enabled=%t", provider, enabled), func(t *testing.T) {
				r := support.Setup(t)
				defer r.Close()
				db := database.DB(t.Context())
				component := provider + ".updatePullRequestBodySection"
				canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
					{NodeID: "trigger", Type: models.NodeTypeTrigger},
					{NodeID: "publish", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: component}})},
				}, nil)
				root := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
				execution := support.CreateCanvasNodeExecution(t, canvas.ID, "publish", root.ID, root.ID)
				execution.Configuration = datatypes.NewJSONType(map[string]any{"repository": "acme/app", "pullNumber": "42", "sectionKey": "notes", "body": "report", "enabled": enabled})
				execution.State = models.CanvasNodeExecutionStateStarted
				require.NoError(t, db.Save(execution).Error)
				action, err := r.Registry.GetAction(component)
				require.NoError(t, err)
				require.NoError(t, action.Execute(core.ExecutionContext{RunID: execution.RunID, Configuration: execution.Configuration.Data(), Metadata: contexts.NewExecutionMetadataContext(db, execution), Requests: contexts.NewExecutionRequestContext(db, execution)}))
				var request models.CanvasNodeRequest
				require.NoError(t, db.Where("execution_id = ?", execution.ID).First(&request).Error)
				worker := NewNodeRequestExecutor(r.Encryptor, r.Registry, "", nil)
				require.NoError(t, worker.RunNow(request))
				require.NoError(t, db.First(execution, "id = ?", execution.ID).Error)
				assert.Equal(t, models.CanvasNodeExecutionResultPassed, execution.Result)
				assert.Equal(t, "skipped", execution.Metadata.Data()["status"])
				if enabled {
					assert.Contains(t, execution.Metadata.Data()["reason"], "integration")
				} else {
					assert.Contains(t, execution.Metadata.Data()["reason"], "disabled")
				}
			})
		}
	}
}
