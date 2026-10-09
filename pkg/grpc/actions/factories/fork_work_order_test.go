package factories

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	factorycomp "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ForkWorkOrder__IntakeCopiesTheRequestAndLeavesTheSourceRun(t *testing.T) {
	fx := setupForkSource(t, true)
	backlog := attachBacklogCanvas(t, fx)

	resp, err := ForkWorkOrder(fx.ctx, fx.r.Organization.ID.String(), &pb.ForkWorkOrderRequest{
		FactoryId: fx.factory.ID.String(),
		OrderId:   fx.source.ID.String(),
		Mode:      pb.ForkWorkOrderRequest_MODE_INTAKE,
	})
	require.NoError(t, err)
	require.NotEqual(t, fx.source.ID.String(), resp.Order.Id)
	assert.NotEqual(t, fx.source.Number, resp.Order.Number)
	assert.Equal(t, fx.source.Title, resp.Order.Title)
	assert.Equal(t, pb.WorkOrder_STATE_DRAFT, resp.Order.State)
	assert.Empty(t, resp.Order.LineDispatches)
	assert.Nil(t, resp.Order.Origin)
	assert.Empty(t, resp.Order.SourceRunId)
	require.Len(t, resp.Order.Assignees, 1)
	assert.Equal(t, fx.r.User.String(), resp.Order.Assignees[0].Id)
	require.Len(t, resp.Order.Files, 1)
	assert.NotEqual(t, fx.file.ID.String(), resp.Order.Files[0].Id)
	assert.Contains(t, resp.Order.Description, blob.FileRef(uuid.MustParse(resp.Order.Files[0].Id)))
	assert.NotContains(t, resp.Order.Description, blob.FileRef(fx.file.ID))
	assert.Empty(t, resp.Order.Checks)

	assertSourceUnchanged(t, fx)
	events, err := models.ListCanvasEvents(fx.db, backlog.ID, models.FactoryAppBacklogTriggerID, 10, nil)
	require.NoError(t, err)
	assert.Len(t, events, 1)

	forked, err := fx.factory.FindWorkOrder(fx.db, uuid.MustParse(resp.Order.Id))
	require.NoError(t, err)
	_, err = models.FindPlanningSessionByDraftWorkOrder(fx.db, fx.factory.OrganizationID, fx.factory.ID, forked.ID)
	assert.ErrorIs(t, err, models.ErrFactoryPlanningSessionNotFound)
	assert.Equal(t, "acme/widgets", *forked.Repository)
	assert.Equal(t, "release", *forked.DefaultBranch)
	assert.Equal(t, models.ProviderGitHub, *forked.VCSProvider)
}

func Test__ForkWorkOrder__IntakeSkipsPlanningWhenItIsOff(t *testing.T) {
	fx := setupForkSource(t, false)
	require.NoError(t, fx.factory.UpdatePlanning(fx.db, models.FactoryPlanning{Enabled: false, Clarity: true, Confidence: true}))
	backlog := attachBacklogCanvas(t, fx)

	_, err := ForkWorkOrder(fx.ctx, fx.r.Organization.ID.String(), &pb.ForkWorkOrderRequest{
		FactoryId: fx.factory.ID.String(),
		OrderId:   fx.source.ID.String(),
		Mode:      pb.ForkWorkOrderRequest_MODE_INTAKE,
	})
	require.NoError(t, err)

	events, err := models.ListCanvasEvents(fx.db, backlog.ID, models.FactoryAppBacklogTriggerID, 10, nil)
	require.NoError(t, err)
	assert.Empty(t, events)
	assertSourceUnchanged(t, fx)
}

func Test__ForkWorkOrder__PlanCopiesTheSpecAndStaysADraft(t *testing.T) {
	fx := setupForkSource(t, true)
	backlog := attachBacklogCanvas(t, fx)

	resp, err := ForkWorkOrder(fx.ctx, fx.r.Organization.ID.String(), &pb.ForkWorkOrderRequest{
		FactoryId: fx.factory.ID.String(),
		OrderId:   fx.source.ID.String(),
		Mode:      pb.ForkWorkOrderRequest_MODE_PLAN,
	})
	require.NoError(t, err)
	assert.Equal(t, pb.WorkOrder_STATE_DRAFT, resp.Order.State)
	assert.Empty(t, resp.Order.LineDispatches)
	assert.Nil(t, resp.Order.Origin)

	forked, err := fx.factory.FindWorkOrder(fx.db, uuid.MustParse(resp.Order.Id))
	require.NoError(t, err)
	artifact, err := forked.FindArtifactByKey(fx.db, models.PlanningSpecArtifactKey+":"+forked.ID.String())
	require.NoError(t, err)
	body := artifactMarkdownBody(artifact)
	assert.Contains(t, body, "# Retry refunds")
	assert.NotContains(t, body, blob.FileRef(fx.file.ID))
	require.Len(t, resp.Order.Files, 1)
	assert.Contains(t, body, blob.FileRef(uuid.MustParse(resp.Order.Files[0].Id)))

	checks, err := forked.ListChecks(fx.db)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		models.PlanningClarityCheckKey,
		models.PlanningComplexityCheckKey,
		models.PlanningVerifiabilityCheckKey,
	}, checkKeys(checks))

	_, err = models.FindPlanningSessionByDraftWorkOrder(fx.db, fx.factory.OrganizationID, fx.factory.ID, forked.ID)
	assert.ErrorIs(t, err, models.ErrFactoryPlanningSessionNotFound)
	events, err := models.ListCanvasEvents(fx.db, backlog.ID, models.FactoryAppBacklogTriggerID, 10, nil)
	require.NoError(t, err)
	assert.Empty(t, events)
	assertSourceUnchanged(t, fx)

	started, err := DispatchWorkOrder(fx.ctx, fx.r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: fx.factory.ID.String(),
		OrderId:   forked.ID.String(),
		LineName:  fx.line.Name,
	})
	require.NoError(t, err)
	assert.Equal(t, pb.WorkOrder_STATE_OPEN, started.Order.State)
	assertSourceUnchanged(t, fx)
}

func Test__ForkWorkOrder__PlanWithoutSpecLeavesNoTask(t *testing.T) {
	fx := setupForkSource(t, false)
	before := workOrderCount(t, fx.db, fx.factory.ID)

	_, err := ForkWorkOrder(fx.ctx, fx.r.Organization.ID.String(), &pb.ForkWorkOrderRequest{
		FactoryId: fx.factory.ID.String(),
		OrderId:   fx.source.ID.String(),
		Mode:      pb.ForkWorkOrderRequest_MODE_PLAN,
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	assert.Equal(t, before, workOrderCount(t, fx.db, fx.factory.ID))
	assertSourceUnchanged(t, fx)
}

func Test__ForkWorkOrder__RequiresAMode(t *testing.T) {
	fx := setupForkSource(t, false)
	before := workOrderCount(t, fx.db, fx.factory.ID)
	_, err := ForkWorkOrder(fx.ctx, fx.r.Organization.ID.String(), &pb.ForkWorkOrderRequest{
		FactoryId: fx.factory.ID.String(),
		OrderId:   fx.source.ID.String(),
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	assert.Equal(t, before, workOrderCount(t, fx.db, fx.factory.ID))
}

type forkSource struct {
	ctx     context.Context
	db      *gorm.DB
	r       *support.ResourceRegistry
	factory *models.Factory
	source  *models.FactoryWorkOrder
	line    *models.FactoryLine
	file    *models.File
}

func setupForkSource(t *testing.T, withPlan bool) forkSource {
	t.Helper()
	r := support.Setup(t)
	store := setupForkFiles(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	other := support.CreateUser(t, r, r.Organization.ID)

	originURL := "https://github.com/acme/widgets/issues/9"
	originLabel := "widgets#9"
	order, err := factoryModel.CreateWorkOrderWithOrigin(
		db,
		"Retry refunds",
		"Refunds fail on retry.",
		&r.User,
		[]uuid.UUID{r.User, other.ID},
		nil,
		models.WorkOrderOrigin{URL: originURL, Label: originLabel},
	)
	require.NoError(t, err)
	require.NoError(t, db.Model(order).Updates(map[string]any{
		"repository":     "acme/widgets",
		"default_branch": "release",
		"vcs_provider":   models.ProviderGitHub,
	}).Error)
	file := readyTaskFile(t, db, store, r, factoryModel, order.ID)
	description := "See ![bug](" + blob.FileRef(file.ID) + ")"
	require.NoError(t, order.UpdateContent(db, nil, &description))
	order, err = factoryModel.FindWorkOrder(db, order.ID)
	require.NoError(t, err)

	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)
	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)
	sourceRun := forkSourceRun(t, r)
	require.NoError(t, db.Model(order).Update("source_run_id", sourceRun.ID).Error)

	if withPlan {
		require.NoError(t, order.StorePlanningSpec(db, "# Retry refunds\n\n![bug]("+blob.FileRef(file.ID)+")\n"))
		reportScore(t, db, order, models.PlanningClarityCheckKey, models.PlanningClarityCheckName, 3)
		reportScore(t, db, order, models.PlanningComplexityCheckKey, models.PlanningComplexityCheckName, 2)
		reportScore(t, db, order, models.PlanningVerifiabilityCheckKey, models.PlanningVerifiabilityCheckName, 3)
		reportScore(t, db, order, models.PlanningConfidenceCheckKey, models.PlanningConfidenceCheckName, 4)
	}

	return forkSource{ctx: ctx, db: db, r: r, factory: factoryModel, source: order, line: line, file: file}
}

func forkSourceRun(t *testing.T, r *support.ResourceRegistry) *models.CanvasRun {
	t.Helper()
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{
			NodeID: "create-order",
			Name:   "Create",
			Type:   models.NodeTypeComponent,
			Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
		}},
		nil,
	)
	run, err := models.CreateCanvasRunInTransaction(database.DB(t.Context()), canvas.ID, "create-order", models.CanvasRunStateFinished, "")
	require.NoError(t, err)
	return run
}

func setupForkFiles(t *testing.T) blob.Provider {
	t.Helper()
	t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
	t.Setenv("BASE_URL", "http://files.test")
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })
	return store
}

func readyTaskFile(
	t *testing.T,
	db *gorm.DB,
	store blob.Provider,
	r *support.ResourceRegistry,
	factoryModel *models.Factory,
	workOrderID uuid.UUID,
) *models.File {
	t.Helper()
	file, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeTask,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    workOrderID,
		Filename:       "bug.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	require.NoError(t, storedfiles.CompleteUpload(t.Context(), db, store, file, bytes.NewReader([]byte("png-bytes"))))
	return file
}

func attachBacklogCanvas(t *testing.T, fx forkSource) *models.Canvas {
	t.Helper()
	backlog, _ := support.CreateCanvas(
		t,
		fx.r.Organization.ID,
		fx.r.User,
		[]models.CanvasNode{{
			NodeID: models.FactoryAppBacklogTriggerID,
			Name:   "On Task",
			Type:   models.NodeTypeTrigger,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: factorycomp.OnWorkOrderTriggerName},
			}),
			Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata(models.FactoryAppTemplateBacklogID, 3)),
		}},
		nil,
	)
	require.NoError(t, fx.db.Model(backlog).Update("factory_id", fx.factory.ID).Error)
	return backlog
}

func reportScore(t *testing.T, db *gorm.DB, order *models.FactoryWorkOrder, key, name string, score float64) {
	t.Helper()
	_, err := order.ReportCheck(db, models.FactoryWorkOrderCheckParams{
		Key:      key,
		Name:     name,
		Score:    score,
		MaxScore: 3,
		Format:   models.FactoryWorkOrderCheckFormatFraction,
		Level:    models.FactoryWorkOrderCheckLevelPositive,
		Summary:  name,
	})
	require.NoError(t, err)
}

func assertSourceUnchanged(t *testing.T, fx forkSource) {
	t.Helper()
	reloaded, err := fx.factory.FindWorkOrder(fx.db, fx.source.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateOpen, reloaded.State)
	assert.Equal(t, fx.source.Number, reloaded.Number)
	assert.NotNil(t, reloaded.OriginURL)
	assert.NotNil(t, reloaded.SourceRunID)
	active, err := reloaded.FindActiveLineDispatch(fx.db)
	require.NoError(t, err)
	assert.Equal(t, fx.line.ID, active.LineID)
	file, err := models.FindFile(fx.db, fx.file.ID)
	require.NoError(t, err)
	require.NotNil(t, file.WorkOrderID)
	assert.Equal(t, fx.source.ID, *file.WorkOrderID)
}

func workOrderCount(t *testing.T, db *gorm.DB, factoryID uuid.UUID) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&models.FactoryWorkOrder{}).Where("factory_id = ?", factoryID).Count(&count).Error)
	return count
}

func checkKeys(checks []models.FactoryWorkOrderCheck) []string {
	keys := make([]string, 0, len(checks))
	for i := range checks {
		keys = append(keys, checks[i].Key)
	}
	return keys
}
