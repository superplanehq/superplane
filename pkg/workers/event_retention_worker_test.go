package workers

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/usage"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

type fakeEventRetentionUsageService struct {
	enabled bool
}

func (s *fakeEventRetentionUsageService) Enabled() bool {
	return s.enabled
}

func (s *fakeEventRetentionUsageService) SetupAccount(context.Context, string) (*pb.SetupAccountResponse, error) {
	return &pb.SetupAccountResponse{}, nil
}

func (s *fakeEventRetentionUsageService) SetupOrganization(context.Context, string, string) (*pb.SetupOrganizationResponse, error) {
	return &pb.SetupOrganizationResponse{}, nil
}

func (s *fakeEventRetentionUsageService) DescribeAccountLimits(context.Context, string) (*pb.DescribeAccountLimitsResponse, error) {
	return &pb.DescribeAccountLimitsResponse{}, nil
}

func (s *fakeEventRetentionUsageService) DescribeOrganizationLimits(context.Context, string) (*pb.DescribeOrganizationLimitsResponse, error) {
	return &pb.DescribeOrganizationLimitsResponse{}, nil
}

func (s *fakeEventRetentionUsageService) DescribeOrganizationUsage(context.Context, string) (*pb.DescribeOrganizationUsageResponse, error) {
	return &pb.DescribeOrganizationUsageResponse{}, nil
}

func (s *fakeEventRetentionUsageService) CheckAccountLimits(context.Context, string, *pb.AccountState) (*pb.CheckAccountLimitsResponse, error) {
	return &pb.CheckAccountLimitsResponse{Allowed: true}, nil
}

func (s *fakeEventRetentionUsageService) CheckOrganizationLimits(context.Context, string, *pb.OrganizationState, *pb.CanvasState) (*pb.CheckOrganizationLimitsResponse, error) {
	return &pb.CheckOrganizationLimitsResponse{Allowed: true}, nil
}

var _ usage.Service = (*fakeEventRetentionUsageService)(nil)

func cacheOrganizationRetentionWindowDays(t *testing.T, orgID uuid.UUID, retentionWindowDays int32) {
	t.Helper()

	require.NoError(
		t,
		models.MarkOrganizationUsageLimitsSynced(orgID.String(), &retentionWindowDays, time.Now()),
	)
}

func Test__EventRetentionWorker_SkipsRootEventWithinRetentionWindow(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewEventRetentionWorker(&fakeEventRetentionUsageService{enabled: true})
	cacheOrganizationRetentionWindowDays(t, r.Organization.ID, 30)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "trigger",
				Type:   models.NodeTypeTrigger,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Trigger: &models.TriggerRef{Name: "start"},
				}),
			},
		},
		[]models.Edge{},
	)

	rootEventRecord := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	require.NoError(t, database.Conn().Model(&models.CanvasEvent{}).Where("id = ?", rootEventRecord.ID).Updates(map[string]any{
		"state":      models.CanvasEventStateRouted,
		"created_at": time.Now().AddDate(0, 0, -29),
	}).Error)

	rootEventInDB, err := models.FindCanvasEvent(rootEventRecord.ID)
	require.NoError(t, err)

	require.NoError(t, worker.LockAndProcessRootEvent(*rootEventInDB, time.Now()))

	support.VerifyCanvasEventsCount(t, canvas.ID, 1)
}

func Test__EventRetentionWorker_SkipsRootEventWithoutCachedRetentionWindow(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewEventRetentionWorker(&fakeEventRetentionUsageService{enabled: true})

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "trigger",
				Type:   models.NodeTypeTrigger,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Trigger: &models.TriggerRef{Name: "start"},
				}),
			},
		},
		[]models.Edge{},
	)

	rootEventRecord := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	require.NoError(t, database.Conn().Model(&models.CanvasEvent{}).Where("id = ?", rootEventRecord.ID).Updates(map[string]any{
		"state":      models.CanvasEventStateRouted,
		"created_at": time.Now().AddDate(0, 0, -31),
	}).Error)

	rootEventInDB, err := models.FindCanvasEvent(rootEventRecord.ID)
	require.NoError(t, err)

	require.NoError(t, worker.LockAndProcessRootEvent(*rootEventInDB, time.Now()))

	support.VerifyCanvasEventsCount(t, canvas.ID, 1)
}

func Test__EventRetentionWorker_CleansExpiredCompletedRootEventChain(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewEventRetentionWorker(&fakeEventRetentionUsageService{enabled: true})
	cacheOrganizationRetentionWindowDays(t, r.Organization.ID, 30)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "trigger",
				Type:   models.NodeTypeTrigger,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Trigger: &models.TriggerRef{Name: "start"},
				}),
			},
			{
				NodeID: "component",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	require.NoError(t, database.Conn().Model(&models.CanvasEvent{}).Where("id = ?", rootEvent.ID).Updates(map[string]any{
		"state":      models.CanvasEventStateRouted,
		"created_at": time.Now().AddDate(0, 0, -31),
	}).Error)

	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "component", rootEvent.ID, rootEvent.ID, nil)
	require.NoError(t, database.Conn().Model(&models.CanvasNodeExecution{}).Where("id = ?", execution.ID).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultPassed,
		"created_at": time.Now().AddDate(0, 0, -31),
		"updated_at": time.Now().AddDate(0, 0, -31),
	}).Error)

	childEvent := support.EmitCanvasEventForNode(t, canvas.ID, "component", "default", &execution.ID)
	require.NoError(t, database.Conn().Model(&models.CanvasEvent{}).Where("id = ?", childEvent.ID).Updates(map[string]any{
		"state":      models.CanvasEventStateRouted,
		"created_at": time.Now().AddDate(0, 0, -31),
	}).Error)

	require.NoError(t, models.CreateNodeExecutionKVInTransaction(database.Conn(), canvas.ID, "component", execution.ID, "test-key", "test-value"))

	request := models.CanvasNodeRequest{
		ID:          uuid.New(),
		WorkflowID:  canvas.ID,
		NodeID:      "component",
		ExecutionID: &execution.ID,
		State:       models.NodeExecutionRequestStateCompleted,
		Type:        models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{ActionName: "test", Parameters: map[string]any{}},
		}),
		RunAt:     time.Now().AddDate(0, 0, -31),
		CreatedAt: time.Now().AddDate(0, 0, -31),
		UpdatedAt: time.Now().AddDate(0, 0, -31),
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	rootEventInDB, err := models.FindCanvasEvent(rootEvent.ID)
	require.NoError(t, err)

	require.NoError(t, worker.LockAndProcessRootEvent(*rootEventInDB, time.Now()))

	support.VerifyCanvasEventsCount(t, canvas.ID, 0)
	support.VerifyNodeExecutionsCount(t, canvas.ID, 0)
	support.VerifyNodeExecutionKVCount(t, canvas.ID, 0)
	support.VerifyNodeRequestCount(t, canvas.ID, 0)
}

func Test__EventRetentionWorker_SkipsRootEventWithQueuedWork(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewEventRetentionWorker(&fakeEventRetentionUsageService{enabled: true})
	cacheOrganizationRetentionWindowDays(t, r.Organization.ID, 30)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "trigger",
				Type:   models.NodeTypeTrigger,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Trigger: &models.TriggerRef{Name: "start"},
				}),
			},
			{
				NodeID: "component",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	require.NoError(t, database.Conn().Model(&models.CanvasEvent{}).Where("id = ?", rootEvent.ID).Updates(map[string]any{
		"state":      models.CanvasEventStateRouted,
		"created_at": time.Now().AddDate(0, 0, -31),
	}).Error)
	support.CreateQueueItem(t, canvas.ID, "component", rootEvent.ID, rootEvent.ID)

	rootEventInDB, err := models.FindCanvasEvent(rootEvent.ID)
	require.NoError(t, err)

	require.NoError(t, worker.LockAndProcessRootEvent(*rootEventInDB, time.Now()))

	support.VerifyCanvasEventsCount(t, canvas.ID, 1)
	support.VerifyNodeQueueCount(t, canvas.ID, 1)
}

func Test__EventRetentionWorker_SkipsRootEventWithPendingRequest(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewEventRetentionWorker(&fakeEventRetentionUsageService{enabled: true})
	cacheOrganizationRetentionWindowDays(t, r.Organization.ID, 30)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "trigger",
				Type:   models.NodeTypeTrigger,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Trigger: &models.TriggerRef{Name: "start"},
				}),
			},
			{
				NodeID: "component",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	require.NoError(t, database.Conn().Model(&models.CanvasEvent{}).Where("id = ?", rootEvent.ID).Updates(map[string]any{
		"state":      models.CanvasEventStateRouted,
		"created_at": time.Now().AddDate(0, 0, -31),
	}).Error)

	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "component", rootEvent.ID, rootEvent.ID, nil)
	require.NoError(t, database.Conn().Model(&models.CanvasNodeExecution{}).Where("id = ?", execution.ID).Updates(map[string]any{
		"state":      models.CanvasNodeExecutionStateFinished,
		"result":     models.CanvasNodeExecutionResultPassed,
		"created_at": time.Now().AddDate(0, 0, -31),
		"updated_at": time.Now().AddDate(0, 0, -31),
	}).Error)

	request := models.CanvasNodeRequest{
		ID:          uuid.New(),
		WorkflowID:  canvas.ID,
		NodeID:      "component",
		ExecutionID: &execution.ID,
		State:       models.NodeExecutionRequestStatePending,
		Type:        models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{ActionName: "test", Parameters: map[string]any{}},
		}),
		RunAt:     time.Now().AddDate(0, 0, -31),
		CreatedAt: time.Now().AddDate(0, 0, -31),
		UpdatedAt: time.Now().AddDate(0, 0, -31),
	}
	require.NoError(t, database.Conn().Create(&request).Error)

	rootEventInDB, err := models.FindCanvasEvent(rootEvent.ID)
	require.NoError(t, err)

	require.NoError(t, worker.LockAndProcessRootEvent(*rootEventInDB, time.Now()))

	support.VerifyCanvasEventsCount(t, canvas.ID, 1)
	support.VerifyNodeExecutionsCount(t, canvas.ID, 1)
	support.VerifyNodeRequestCount(t, canvas.ID, 1)
}
