package components

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__DescribeComponent(t *testing.T) {
	r := support.Setup(t)
	r.Registry.Components["test"] = support.NewDummyComponent(support.DummyComponentOptions{})

	t.Run("component does not exist -> error", func(t *testing.T) {
		_, err := DescribeComponent(context.Background(), r.Registry, "nope")
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "component nope not found", s.Message())
	})

	t.Run("describe existing component", func(t *testing.T) {
		response, err := DescribeComponent(context.Background(), r.Registry, "test")
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Component)
		require.Equal(t, "dummy", response.Component.Name)
		require.Equal(t, "dummy", response.Component.Label)
		require.Equal(t, "Just a dummy component used in unit tests", response.Component.Description)
	})
}
