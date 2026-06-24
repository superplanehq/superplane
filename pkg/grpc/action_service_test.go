package grpc

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	pb "github.com/superplanehq/superplane/pkg/protos/actions"
	"github.com/superplanehq/superplane/pkg/registry"
	_ "github.com/superplanehq/superplane/pkg/registryimports"
)

func newTestRegistry(t *testing.T) *registry.Registry {
	t.Helper()
	reg, err := registry.NewRegistry(nil, registry.HTTPOptions{})
	require.NoError(t, err)
	return reg
}

func Test__ActionService__ListActions(t *testing.T) {
	t.Run("returns serialized actions registered in the registry", func(t *testing.T) {
		reg := newTestRegistry(t)
		svc := NewActionService(reg)

		resp, err := svc.ListActions(context.Background(), &pb.ListActionsRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.Actions, "expected at least one registered action")

		names := map[string]bool{}
		for _, action := range resp.Actions {
			assert.NotEmpty(t, action.Name)
			names[action.Name] = true
		}
		assert.True(t, names["wait"], "expected the 'wait' core action to be present")
	})

	t.Run("skips actions that panic during serialization instead of failing the whole request", func(t *testing.T) {
		reg := newTestRegistry(t)
		// Inject an action that panics on every metadata method. The
		// service must keep enumerating the remaining actions instead
		// of returning HTTP 500.
		reg.Actions["__panicking__"] = &panickingAction{}

		svc := NewActionService(reg)
		resp, err := svc.ListActions(context.Background(), &pb.ListActionsRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.Actions)

		for _, action := range resp.Actions {
			assert.NotEqual(t, "__panicking__", action.Name)
		}
	})
}

func Test__ActionService__DescribeAction(t *testing.T) {
	t.Run("returns the action when present", func(t *testing.T) {
		reg := newTestRegistry(t)
		svc := NewActionService(reg)

		resp, err := svc.DescribeAction(context.Background(), &pb.DescribeActionRequest{Name: "wait"})
		require.NoError(t, err)
		require.NotNil(t, resp.Action)
		assert.Equal(t, "wait", resp.Action.Name)
	})

	t.Run("returns NotFound for unknown action", func(t *testing.T) {
		reg := newTestRegistry(t)
		svc := NewActionService(reg)

		_, err := svc.DescribeAction(context.Background(), &pb.DescribeActionRequest{Name: "missing"})
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, grpcerrors.Code(err))
	})

	t.Run("returns Internal when the action panics during serialization", func(t *testing.T) {
		reg := newTestRegistry(t)
		reg.Actions["__panicking__"] = &panickingAction{}

		svc := NewActionService(reg)
		_, err := svc.DescribeAction(context.Background(), &pb.DescribeActionRequest{Name: "__panicking__"})
		require.Error(t, err)
		assert.Equal(t, codes.Internal, grpcerrors.Code(err))
	})
}

// panickingAction is a test double whose definition methods panic. It
// validates that the action service tolerates misbehaving actions instead
// of bubbling a panic up as an HTTP 500.
type panickingAction struct{}

func (panickingAction) Name() string                            { return "__panicking__" }
func (panickingAction) Label() string                           { panic("Label panicked") }
func (panickingAction) Description() string                     { panic("Description panicked") }
func (panickingAction) Documentation() string                   { panic("Documentation panicked") }
func (panickingAction) Icon() string                            { panic("Icon panicked") }
func (panickingAction) Color() string                           { panic("Color panicked") }
func (panickingAction) ExampleOutput() map[string]any           { panic("ExampleOutput panicked") }
func (panickingAction) OutputChannels(any) []core.OutputChannel { panic("OutputChannels panicked") }
func (panickingAction) Configuration() []configuration.Field    { panic("Configuration panicked") }
func (panickingAction) Setup(core.SetupContext) error           { panic("Setup panicked") }
func (panickingAction) ProcessQueueItem(core.ProcessQueueContext) (*uuid.UUID, error) {
	panic("ProcessQueueItem panicked")
}
func (panickingAction) Execute(core.ExecutionContext) error     { panic("Execute panicked") }
func (panickingAction) Hooks() []core.Hook                      { panic("Hooks panicked") }
func (panickingAction) HandleHook(core.ActionHookContext) error { panic("HandleHook panicked") }
func (panickingAction) HandleWebhook(core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	panic("HandleWebhook panicked")
}
func (panickingAction) Cancel(core.ExecutionContext) error { panic("Cancel panicked") }
func (panickingAction) Cleanup(core.SetupContext) error    { panic("Cleanup panicked") }
