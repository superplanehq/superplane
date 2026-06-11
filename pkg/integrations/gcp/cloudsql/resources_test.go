package cloudsql

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__ListInstances__FollowsPagination(t *testing.T) {
	calls := 0
	mc := &mockClient{
		projectID: "my-project",
		getFunc: func(ctx context.Context, u string) ([]byte, error) {
			calls++
			if !strings.Contains(u, "pageToken=") {
				return []byte(`{"items":[{"name":"instance-a"}],"nextPageToken":"tok2"}`), nil
			}
			assert.Contains(t, u, "pageToken=tok2")
			return []byte(`{"items":[{"name":"instance-b"}]}`), nil
		},
	}

	instances, err := ListInstances(context.Background(), mc, "my-project")
	require.NoError(t, err)
	assert.Equal(t, 2, calls, "should fetch both pages")
	require.Len(t, instances, 2)
	assert.Equal(t, "instance-a", instances[0].Name)
	assert.Equal(t, "instance-b", instances[1].Name)
}
