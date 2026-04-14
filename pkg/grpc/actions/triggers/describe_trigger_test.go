package triggers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__DescribeTrigger(t *testing.T) {
	r := support.Setup(t)
	r.Registry.Triggers["test"] = support.NewDummyTrigger(support.DummyTriggerOptions{})

	t.Run("trigger does not exist -> error", func(t *testing.T) {
		_, err := DescribeTrigger(context.Background(), r.Registry, "nope")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "trigger nope not found", s.Message())
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
