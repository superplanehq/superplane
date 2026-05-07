package triggers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__DescribeTrigger(t *testing.T) {
	r := support.Setup(t)
	r.Registry.Triggers["test"] = impl.NewDummyTrigger(impl.DummyTriggerOptions{
		Configuration: []configuration.Field{
			{
				Name: "field",
				Type: configuration.FieldTypeString,
			},
		},
		ExampleData: map[string]any{
			"message": "hello",
		},
		DefaultRunTitle: "{{ root().data.message }}",
	})

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
		require.Equal(t, "{{ root().data.message }}", response.Trigger.DefaultRunTitle)
		require.Len(t, response.Trigger.Configuration, 1)
		require.Equal(t, "field", response.Trigger.Configuration[0].Name)
		require.Equal(t, "hello", response.Trigger.ExampleData.Fields["message"].GetStringValue())
	})
}

func Test__ListTriggers(t *testing.T) {
	r := support.Setup(t)
	r.Registry.Triggers = map[string]core.Trigger{
		"test": impl.NewDummyTrigger(impl.DummyTriggerOptions{
			DefaultRunTitle: "{{ root().data.message }}",
		}),
	}

	response, err := ListTriggers(context.Background(), r.Registry)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Triggers, 1)
	assert.Equal(t, "dummy", response.Triggers[0].Name)
	assert.Equal(t, "{{ root().data.message }}", response.Triggers[0].DefaultRunTitle)
}
