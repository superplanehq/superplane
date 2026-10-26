package workers

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	runnerapi "github.com/superplanehq/superplane/pkg/runners/api"
	runnercontrol "github.com/superplanehq/superplane/pkg/runners/control"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
)

func TestRunnerQueueNotificationsFollowTheOwningTransaction(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		name := "commit"
		if rollback {
			name = "rollback"
		}
		t.Run(name, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			url, err := config.RabbitMQURL()
			require.NoError(t, err)
			broker, err := amqp.Dial(url)
			require.NoError(t, err)
			defer broker.Close()
			channel, err := broker.Channel()
			require.NoError(t, err)
			defer channel.Close()
			require.NoError(t, channel.ExchangeDeclare(runnercontrol.Exchange, "fanout", true, false, false, false, nil))
			queue, err := channel.QueueDeclare("", false, true, true, false, nil)
			require.NoError(t, err)
			require.NoError(t, channel.QueueBind(queue.Name, "", runnercontrol.Exchange, false, nil))

			fleet := models.DefaultInstallationRunnerFleets("dev")[0]
			require.NoError(t, fleet.Create(db))
			taskID := uuid.New()
			payload, err := json.Marshal(runnerapi.TaskPayload{ID: taskID.String()})
			require.NoError(t, err)
			rollbackErr := errors.New("rollback after queue creation")
			const component = "queue-notification-test"
			r.Registry.Actions[component] = impl.NewDummyAction(impl.DummyActionOptions{
				Name: component, Hooks: []core.Hook{{Name: "queue"}},
				HandleHookFunc: func(ctx core.ActionHookContext) error {
					if err := ctx.RunnerTasks.Create(taskID.String(), fleet.Slug, payload); err != nil {
						return err
					}
					_, published, err := channel.Get(queue.Name, true)
					require.NoError(t, err)
					require.False(t, published, "a waiting runner must not wake before the job commits")
					if rollback {
						return rollbackErr
					}
					return nil
				},
			})
			canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
				NodeID: "queue", Type: models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: component}}),
			}}, nil)
			event := support.EmitCanvasEventForNode(t, canvas.ID, "queue", "default", nil)
			execution := support.CreateCanvasNodeExecution(t, canvas.ID, "queue", event.ID, event.ID)
			request := models.CanvasNodeRequest{ID: uuid.New(), WorkflowID: canvas.ID, NodeID: "queue", ExecutionID: &execution.ID,
				Type: models.NodeRequestTypeInvokeAction, State: models.NodeExecutionRequestStatePending,
				Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{InvokeAction: &models.InvokeAction{ActionName: "queue"}})}
			require.NoError(t, db.Create(&request).Error)

			err = NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService).RunNow(request)
			if rollback {
				require.ErrorIs(t, err, rollbackErr)
			} else {
				require.NoError(t, err)
			}
			message, published, err := channel.Get(queue.Name, true)
			require.NoError(t, err)
			require.Equal(t, !rollback, published)
			task, findErr := models.FindRunnerTask(db, taskID)
			if rollback {
				require.ErrorIs(t, findErr, models.ErrRunnerTaskNotFound)
				return
			}
			require.NoError(t, findErr)
			assert.Equal(t, models.RunnerTaskStateQueued, task.State)
			var notification runnercontrol.Notification
			require.NoError(t, json.Unmarshal(message.Body, &notification))
			assert.Equal(t, fleet.ID.String(), notification.FleetID)
		})
	}
}
