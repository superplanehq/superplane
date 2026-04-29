package workers

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
)

func init() {
	registry.RegisterAction(testHookProbeComponentName, impl.NewDummyAction(impl.DummyActionOptions{
		Name: testHookProbeComponentName,
		Hooks: []core.Hook{
			{Name: "probeHook", Type: core.HookTypeInternal},
		},
		ProcessQueueFunc: func(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
			return ctx.DefaultProcessing()
		},
		ExecuteFunc: func(ctx core.ExecutionContext) error {
			return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "hook_probe.done", []any{map[string]any{}})
		},
		HandleHookFunc: func(ctx core.ActionHookContext) error {
			testHookProbeConfigMu.Lock()
			testHookProbeLastCfg = ctx.Configuration
			testHookProbeConfigMu.Unlock()

			return nil
		},
	}))
}

const testHookProbeComponentName = "hook_probe_cfg_action"

var (
	testHookProbeConfigMu sync.Mutex
	testHookProbeLastCfg  any
)

func resetTestHookProbeCapture() {
	testHookProbeConfigMu.Lock()
	defer testHookProbeConfigMu.Unlock()
	testHookProbeLastCfg = nil
}

func testHookProbeLastConfiguration() any {
	testHookProbeConfigMu.Lock()
	defer testHookProbeConfigMu.Unlock()
	return testHookProbeLastCfg
}

/*
Internal component hooks for workflow-level nodes must receive execution.Configuration (resolved snapshot),
not node.Configuration (raw canvas templates). Regression for issue #4441 (HTTP retries saw raw URLs).
*/
func Test_NodeRequestWorker_InternalHookUsesExecutionSnapshotConfiguration(t *testing.T) {
	resetTestHookProbeCapture()

	r := support.Setup(t)
	defer r.Close()

	amqpURL, _ := config.RabbitMQURL()
	executionConsumer := testconsumer.New(amqpURL, messages.CanvasExecutionRoutingKey)
	executionConsumer.Start()
	defer executionConsumer.Stop()

	rawURL := `http://{{ $['Create Droplet'].data.networks.v4[0].ip_address }}.sslip.io/health`
	resolvedURL := "http://45.55.209.135.sslip.io/health"

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{
			{
				NodeID:        "probe-node",
				Name:          "Probe",
				Type:          models.NodeTypeComponent,
				Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: testHookProbeComponentName}}),
				Configuration: datatypes.NewJSONType(map[string]any{"url": rawURL}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "probe-node", "default", nil)
	execution := support.CreateNodeExecutionWithConfiguration(t, canvas.ID, "probe-node", rootEvent.ID, rootEvent.ID, nil,
		map[string]any{"url": resolvedURL},
	)

	req := models.CanvasNodeRequest{
		ID:          uuid.New(),
		WorkflowID:  canvas.ID,
		NodeID:      "probe-node",
		ExecutionID: &execution.ID,
		Type:        models.NodeRequestTypeInvokeAction,
		Spec: datatypes.NewJSONType(models.NodeExecutionRequestSpec{
			InvokeAction: &models.InvokeAction{
				ActionName: "probeHook",
				Parameters: map[string]any{},
			},
		}),
		State: models.NodeExecutionRequestStatePending,
	}
	require.NoError(t, database.Conn().Create(&req).Error)

	worker := NewNodeRequestWorker(r.Encryptor, r.Registry, "", r.AuthService)
	err := worker.LockAndProcessRequest(req)
	require.NoError(t, err)

	cfg, ok := testHookProbeLastConfiguration().(map[string]any)
	require.True(t, ok, "hook should receive configuration map")
	assert.Equal(t, resolvedURL, cfg["url"], "internal hooks must receive resolved execution snapshot, not raw canvas templates")

	assert.False(t, executionConsumer.HasReceivedMessage())
}
