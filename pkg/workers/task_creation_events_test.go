package workers

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	factorycomp "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/proto"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestNodeExecutorPublishesTaskEventsAfterCommit(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		name := "commit"
		if rollback {
			name = "rollback"
		}
		t.Run(name, func(t *testing.T) {
			r := support.Setup(t)
			db := database.Conn()
			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			intake, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
				NodeID: "create-task", Type: models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: factorycomp.ComponentName}}),
			}}, nil)
			backlog, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
				NodeID: "on-task", Type: models.NodeTypeTrigger,
				Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: factorycomp.OnWorkOrderTriggerName}}),
			}}, nil)
			for _, canvas := range []*models.Canvas{intake, backlog} {
				require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
			}
			root := support.EmitCanvasEventForNode(t, intake.ID, "create-task", "default", nil)
			execution := support.CreateCanvasNodeExecution(t, intake.ID, "create-task", root.ID, root.ID)
			require.NoError(t, db.Model(execution).Update("configuration", datatypes.NewJSONType(map[string]any{"title": "Imported task"})).Error)

			amqpURL, err := config.RabbitMQURL()
			require.NoError(t, err)
			connection, err := amqp.Dial(amqpURL)
			require.NoError(t, err)
			t.Cleanup(func() { _ = connection.Close() })
			channel, err := connection.Channel()
			require.NoError(t, err)
			t.Cleanup(func() { _ = channel.Close() })
			require.NoError(t, channel.ExchangeDeclare(messages.EventsExchange, "direct", true, false, false, false, nil))
			queue, err := channel.QueueDeclare("", false, true, true, false, nil)
			require.NoError(t, err)
			require.NoError(t, channel.QueueBind(queue.Name, messages.EventCreatedRoutingKey, messages.EventsExchange, false, nil))

			writeErr := errors.New("cannot save completed execution")
			if rollback {
				const callback = "test:task_creation_write_failure"
				require.NoError(t, db.Callback().Update().Before("gorm:update").Register(callback, func(tx *gorm.DB) {
					completed, ok := tx.Statement.Model.(*models.CanvasNodeExecution)
					if ok && completed.State == models.CanvasNodeExecutionStateFinished {
						tx.AddError(writeErr)
					}
				}))
				t.Cleanup(func() { _ = db.Callback().Update().Remove(callback) })
			}

			pool, err := db.DB()
			require.NoError(t, err)
			maxConnections := pool.Stats().MaxOpenConnections
			pool.SetMaxOpenConns(1)
			t.Cleanup(func() {
				pool.SetMaxOpenConns(maxConnections)
				pool.SetMaxIdleConns(maxConnections)
			})
			executor := newTestNodeExecutor(t, r)
			result := make(chan error, 1)
			go func() { result <- executor.LockAndProcessNodeExecution(execution.ID) }()
			var processErr error
			select {
			case processErr = <-result:
			case <-time.After(3 * time.Second):
				// Release the blocked lookup so the regression fails without
				// leaving a worker running against the next test's database.
				pool.SetMaxOpenConns(maxConnections)
				require.NoError(t, pool.PingContext(t.Context()))
				processErr = <-result
				t.Error("task creation could not finish with one database connection")
			}
			pool.SetMaxOpenConns(maxConnections)
			if rollback {
				require.ErrorIs(t, processErr, writeErr)
			} else {
				require.NoError(t, processErr)
			}

			var taskCount int64
			require.NoError(t, db.Model(&models.FactoryWorkOrder{}).Where("factory_id = ?", factory.ID).Count(&taskCount).Error)
			if rollback {
				assert.Zero(t, taskCount)
			} else {
				assert.EqualValues(t, 1, taskCount)
			}

			var notifiedEvents []uuid.UUID
			for {
				delivery, received, err := channel.Get(queue.Name, true)
				require.NoError(t, err)
				if !received {
					break
				}
				var event pb.CanvasNodeEventMessage
				require.NoError(t, proto.Unmarshal(delivery.Body, &event))
				assert.Equal(t, r.Organization.ID.String(), event.OrganizationId)
				notifiedEvents = append(notifiedEvents, uuid.MustParse(event.Id))
			}
			if rollback {
				assert.Empty(t, notifiedEvents, "rolled-back task events must not reach RabbitMQ")
				return
			}
			var persistedEvents []models.CanvasEvent
			require.NoError(t, db.Where("id <> ?", root.ID).Find(&persistedEvents).Error)
			require.Len(t, persistedEvents, 2, "notify the task trigger and the intake output")
			for _, event := range persistedEvents {
				assert.Contains(t, notifiedEvents, event.ID)
			}
		})
	}
}
