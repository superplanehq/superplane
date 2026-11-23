package factories

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestRetireProductiveIntakes(t *testing.T) {
	for _, state := range []string{"active", "paused", "deleted"} {
		t.Run(state, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			db := database.DB(t.Context())
			ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
			deps := IntakeDependencies{Registry: r.Registry, Encryptor: r.Encryptor, AuthService: r.AuthService, WebhookBaseURL: "http://localhost:8000"}
			f, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			canvas := support.CreateFactoryCanvas(t, r, f.ID, "Productive")
			oldVersion := *canvas.LiveVersionID
			nodes := []models.Node{
				{ID: "trigger", Name: "On Task", Type: models.NodeTypeTrigger, Ref: models.NodeRef{Trigger: &models.TriggerRef{Name: "productive.onTask"}}, Configuration: map[string]any{"project": "42", "actions": []any{"created", "updated"}}},
				{ID: "create", Name: "Create work order", Type: models.NodeTypeComponent, Ref: models.NodeRef{Component: &models.ComponentRef{Name: "createWorkOrder"}}},
			}
			edges := []models.Edge{{SourceID: "trigger", TargetID: "create", Channel: "default"}}
			require.NoError(t, db.Model(&models.CanvasVersion{}).Where("id = ?", oldVersion).Updates(map[string]any{"nodes": datatypes.NewJSONSlice(nodes), "edges": datatypes.NewJSONSlice(edges)}).Error)
			now := time.Now()
			webhook := models.Webhook{ID: uuid.New(), Secret: []byte("stored-test-secret"), State: models.WebhookStateReady, Metadata: datatypes.NewJSONType(any(map[string]any{"ids": []any{"remote-created", "remote-updated"}})), CreatedAt: &now}
			require.NoError(t, db.Create(&webhook).Error)
			for _, node := range nodes {
				stored := models.CanvasNode{WorkflowID: canvas.ID, NodeID: node.ID, Name: node.Name, Type: node.Type, State: models.CanvasNodeStateReady, Ref: datatypes.NewJSONType(node.Ref), Configuration: datatypes.NewJSONType(node.Configuration), CreatedAt: &now, UpdatedAt: &now}
				if node.ID == "trigger" {
					stored.WebhookID = &webhook.ID
				}
				require.NoError(t, db.Create(&stored).Error)
			}
			// A custom canvas may share the same Productive project webhook.
			var shared *models.Canvas
			if state == "paused" {
				shared = support.CreateFactoryCanvas(t, r, f.ID, "Custom Productive workflow")
				require.NoError(t, db.Create(&models.CanvasNode{WorkflowID: shared.ID, NodeID: "custom-trigger", Name: "On Task", Type: models.NodeTypeTrigger, State: models.CanvasNodeStateReady, Ref: datatypes.NewJSONType(nodes[0].Ref), WebhookID: &webhook.ID, CreatedAt: &now, UpdatedAt: &now}).Error)
				sharedNode := nodes[0]
				sharedNode.ID = "custom-trigger"
				require.NoError(t, db.Model(&models.CanvasVersion{}).Where("id = ?", *shared.LiveVersionID).Update("nodes", datatypes.NewJSONSlice([]models.Node{sharedNode})).Error)

			}
			event := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
			execution := support.CreateCanvasNodeExecution(t, canvas.ID, "create", event.ID, event.ID)
			require.NoError(t, db.Model(execution).Update("state", models.CanvasNodeExecutionStateStarted).Error)
			support.CreateQueueItem(t, canvas.ID, "create", event.ID, event.ID)
			intake := models.FactoryIntake{ID: uuid.New(), OrganizationID: r.Organization.ID, FactoryID: f.ID, CanvasID: canvas.ID, Source: models.RetiredProductiveIntakeSource, InitialImportStatus: models.FactoryIntakeInitialImportStatusPending}
			if state == "paused" {
				intake.PausedAt = &now
			}
			require.NoError(t, db.Create(&intake).Error)
			origin := models.WorkOrderOrigin{URL: "https://app.productive.io/123/tasks/91", Label: "91"}
			order, err := f.CreateWorkOrderWithOrigin(db, "Existing Productive task", "Stored attachment", nil, nil, &event.RunID, origin)
			require.NoError(t, err)
			file := models.File{ID: uuid.New(), Scope: "task", OrganizationID: &r.Organization.ID, FactoryID: &f.ID, WorkOrderID: &order.ID, Filename: "task.txt", ContentType: "text/plain", StorageKey: "existing-productive-file", State: models.FileStateReady, Purpose: models.FilePurposeAttachment, CreatedAt: now, UpdatedAt: now}
			require.NoError(t, db.Create(&file).Error)

			deliveryNode, err := models.FindCanvasNode(db, canvas.ID, "trigger")
			require.NoError(t, err)
			assertDeliveryStopped := func() {
				t.Helper()
				delivered := 0
				events := contexts.NewEventContext(db, deliveryNode, nil, func(created []models.CanvasEvent) { delivered += len(created) })
				require.NoError(t, events.Emit("productive.task", map[string]any{"data": map[string]any{"id": "new-task"}}))
				require.Zero(t, delivered)
				var count int64
				require.NoError(t, db.Model(&models.CanvasEvent{}).Where("workflow_id = ?", canvas.ID).Count(&count).Error)
				require.EqualValues(t, 1, count)
			}
			assertDeliveryStopped()
			_, created, err := contexts.NewFactoryContext(db, canvas, execution).CreateWorkOrder(core.WorkOrderParams{Title: "Blocked queued work"})
			require.ErrorIs(t, err, models.ErrFactoryIntakeRetired)
			require.False(t, created)
			_, err = UpdateFactoryIntake(ctx, deps, r.Organization.ID.String(), &pb.UpdateFactoryIntakeRequest{FactoryId: f.ID.String(), IntakeId: intake.ID.String()})
			require.Error(t, err)
			_, err = ImportFactoryIntakeItem(ctx, deps, r.Organization.ID.String(), &pb.ImportFactoryIntakeItemRequest{FactoryId: f.ID.String(), IntakeId: intake.ID.String(), ItemId: "91"})
			require.Error(t, err)
			require.ErrorIs(t, SeedExistingIntake(ctx, deps, db, &intake), models.ErrFactoryIntakeRetired)
			require.ErrorIs(t, db.Transaction(func(tx *gorm.DB) error {
				return canvases.PublishGeneratedCanvasNodesWithOwner(ctx, tx, canvas, nil, "Reconfigure", nodes, edges, intakePublisherOptions(deps, r.Organization.ID))
			}), models.ErrFactoryIntakeRetired)

			if state == "paused" {
				subscription := true
				require.NoError(t, f.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &subscription}))
			}
			if state == "deleted" {
				require.NoError(t, canvas.SoftDeleteInTransaction(db))
			}

			if state == "active" {
				unavailable := deps
				unavailable.Registry = nil
				require.Error(t, RetireProductiveIntakes(ctx, unavailable))
				_, err := f.FindIntake(db, intake.ID)
				require.NoError(t, err)
				var count int64
				require.NoError(t, db.Model(&models.FactoryIntakeRetirement{}).Where("canvas_id = ?", canvas.ID).Count(&count).Error)
				require.Zero(t, count)
				run, err := models.FindCanvasRunInTransaction(db, canvas.ID, event.RunID)
				require.NoError(t, err)
				require.Equal(t, models.CanvasRunStateStarted, run.State)
			}
			require.NoError(t, RetireProductiveIntakes(ctx, deps))
			retained, err := models.FindCanvasInTransaction(db, r.Organization.ID, canvas.ID)
			require.NoError(t, err)
			live, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, retained)
			require.NoError(t, err)
			require.Empty(t, live.Nodes)
			require.Empty(t, live.Edges)
			assertDeliveryStopped()
			require.NotEqual(t, oldVersion, live.ID)
			require.NoError(t, RetireProductiveIntakes(ctx, deps))
			require.Equal(t, live.ID, resetTestLiveVersion(t, retained).ID)
			_, err = f.FindIntake(db, intake.ID)
			require.ErrorIs(t, err, models.ErrFactoryIntakeNotFound)
			var retired models.FactoryIntakeRetirement
			require.NoError(t, db.First(&retired, "canvas_id = ?", canvas.ID).Error)
			require.Equal(t, intake.ID, retired.IntakeID)
			require.NotEmpty(t, retired.Inventory.Data())
			queued, err := models.CountNodeQueueItems(db, canvas.ID, "create")
			require.NoError(t, err)
			require.Zero(t, queued)
			require.NoError(t, db.First(execution, "id = ?", execution.ID).Error)
			require.Equal(t, models.CanvasNodeExecutionStateCancelling, execution.State)
			_, created, err = contexts.NewFactoryContext(db, retained, execution).CreateWorkOrder(core.WorkOrderParams{Title: "Blocked old version"})
			require.ErrorIs(t, err, models.ErrFactoryIntakeRetired)
			require.False(t, created)
			require.ErrorIs(t, db.Transaction(func(tx *gorm.DB) error {
				return canvases.PublishGeneratedCanvasNodesWithOwner(ctx, tx, retained, nil, "Restore old automation", nodes, edges, intakePublisherOptions(deps, r.Organization.ID))
			}), models.ErrFactoryIntakeRetired)
			err = db.First(&webhook, "id = ?", webhook.ID).Error
			if shared != nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, gorm.ErrRecordNotFound)
			}
			automations, err := ListFactoryAutomations(ctx, r.Organization.ID.String(), &pb.ListFactoryAutomationsRequest{FactoryId: f.ID.String()})
			require.NoError(t, err)
			for _, automation := range automations.Automations {
				require.NotEqual(t, canvas.ID.String(), automation.Id)
			}
			// History and attachment records remain queryable through their normal models.
			_, err = models.FindCanvasVersionInTransaction(db, canvas.ID, oldVersion)
			require.NoError(t, err)

			run, err := models.FindCanvasRunInTransaction(db, canvas.ID, event.RunID)
			require.NoError(t, err)
			require.Equal(t, models.CanvasRunStateCancelling, run.State)
			pending, err := models.FindCanvasEventInTransaction(db, event.ID)
			require.NoError(t, err)
			require.Equal(t, models.CanvasEventStateRouted, pending.State)
			trigger, err := models.FindUnscopedCanvasNode(db, canvas.ID, "trigger")
			require.NoError(t, err)
			require.True(t, trigger.DeletedAt.Valid)
			require.Nil(t, trigger.WebhookID)
			references, err := models.FindActiveWebhookNodesInTransaction(db, webhook.ID)
			require.NoError(t, err)
			if shared != nil {
				custom, err := models.FindCanvasNode(db, shared.ID, "custom-trigger")
				require.NoError(t, err)
				delivered := 0
				emitter := contexts.NewEventContext(db, custom, nil, func(events []models.CanvasEvent) { delivered += len(events) })
				require.NoError(t, emitter.Emit("productive.task", map[string]any{"data": map[string]any{"id": "standalone-task"}}))
				require.Equal(t, 1, delivered)
				require.Len(t, resetTestLiveVersion(t, shared).Nodes, 1)
				require.Len(t, references, 1)
				require.Equal(t, shared.ID, references[0].WorkflowID)
			} else {
				require.Empty(t, references)
			}

			saved, err := f.FindWorkOrder(db, order.ID)
			require.NoError(t, err)
			require.Equal(t, origin.URL, saved.Origin().URL)
			files, err := models.ListReadyTaskFiles(db, order.ID)
			require.NoError(t, err)
			require.Len(t, files, 1)
			require.Equal(t, file.StorageKey, files[0].StorageKey)
			match, err := workOrderOriginMatch(origin.URL)
			require.NoError(t, err)
			var matches int64
			require.NoError(t, db.Model(&models.FactoryWorkOrder{}).Where("id = ?", order.ID).Where(match.SQL, match.Args...).Count(&matches).Error)
			require.EqualValues(t, 1, matches)
			_, err = ResetOrganizationBuiltInAutomations(ctx, r.Organization.ID, deps)
			require.NoError(t, err)
			require.Empty(t, resetTestLiveVersion(t, retained).Nodes)
			_, err = MaterializeFactoryAppTemplate(ctx, r.Organization.ID.String(), &pb.MaterializeFactoryAppTemplateRequest{FactoryId: f.ID.String(), AppId: canvas.ID.String(), TemplateId: "intake:productive-tasks"})
			require.Error(t, err)
		})
	}
}
