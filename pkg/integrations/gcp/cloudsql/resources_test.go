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
			// The first request has no pageToken; the second carries the token
			// returned by the first page.
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

const tiersResponse = `{"items":[
	{"tier":"db-f1-micro","RAM":"655360000","region":["us-central1","europe-west1"]},
	{"tier":"db-custom-2-7680","RAM":"8053063680","region":["us-central1"]},
	{"tier":"db-n1-standard-1","RAM":"3840000000","region":["europe-west1"]}
]}`

func tiersClient() *mockClient {
	return &mockClient{
		projectID: "my-project",
		getFunc: func(ctx context.Context, u string) ([]byte, error) {
			return []byte(tiersResponse), nil
		},
	}
}

func Test__ListRegionResources__UnionOfTierRegions(t *testing.T) {
	mc := &mockClient{
		projectID: "my-project",
		getFunc: func(ctx context.Context, u string) ([]byte, error) {
			assert.Contains(t, u, "/projects/my-project/tiers")
			return []byte(tiersResponse), nil
		},
	}

	regions, err := ListRegionResources(context.Background(), mc)
	require.NoError(t, err)
	ids := make([]string, 0, len(regions))
	for _, r := range regions {
		ids = append(ids, r.ID)
	}
	// Deduplicated and sorted union of the regions each tier is offered in.
	assert.Equal(t, []string{"europe-west1", "us-central1"}, ids)
}

func Test__ListTierResources__FiltersByRegion(t *testing.T) {
	mc := tiersClient()

	t.Run("filters to the selected region", func(t *testing.T) {
		tiers, err := ListTierResources(context.Background(), mc, "us-central1")
		require.NoError(t, err)
		ids := make([]string, 0, len(tiers))
		for _, tr := range tiers {
			ids = append(ids, tr.ID)
		}
		// db-n1-standard-1 is europe-west1 only, so it is excluded.
		assert.Equal(t, []string{"db-custom-2-7680", "db-f1-micro"}, ids)
		// Memory is surfaced in the label.
		assert.Contains(t, tiers[1].Name, "GB RAM")
	})

	t.Run("returns empty until a region is chosen", func(t *testing.T) {
		tiers, err := ListTierResources(context.Background(), mc, "")
		require.NoError(t, err)
		assert.Empty(t, tiers)
	})
}
