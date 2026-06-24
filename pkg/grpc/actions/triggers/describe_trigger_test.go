package triggers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"google.golang.org/grpc/codes"
)

func Test__DescribeTrigger(t *testing.T) {
	r := support.Setup(t)
	r.Registry.Triggers["test"] = impl.NewDummyTrigger(impl.DummyTriggerOptions{})

	t.Run("trigger does not exist -> error", func(t *testing.T) {
		_, err := DescribeTrigger(context.Background(), r.Registry, "nope")
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
		assert.Equal(t, "trigger nope not found", msg)
	})

	t.Run("describe existing trigger", func(t *testing.T) {
		response, err := DescribeTrigger(context.Background(), r.Registry, "test")
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Trigger)
		require.Equal(t, "dummy", response.Trigger.Name)
		require.Equal(t, "dummy", response.Trigger.Label)
		require.Equal(t, "Just a dummy trigger used in unit tests", response.Trigger.Description)
	})
}
