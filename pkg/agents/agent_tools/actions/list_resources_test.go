package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	organizationpb "github.com/superplanehq/superplane/pkg/protos/organizations"
)

func TestSerializeAgentIntegrationResources_IgnoresListNotices(t *testing.T) {
	service := func(id string) *organizationpb.IntegrationResourceRef {
		return &organizationpb.IntegrationResourceRef{Type: "service", Id: id, Name: id}
	}
	notice := &organizationpb.IntegrationResourceRef{
		Type: listNoticeResourceType,
		Id:   "issue-search-failed",
		Name: "missing",
	}

	tests := []struct {
		name      string
		resources []*organizationpb.IntegrationResourceRef
		count     int
		truncated bool
		ids       []string
	}{
		{
			name:      "notice after a full page",
			resources: []*organizationpb.IntegrationResourceRef{service("api"), notice},
			count:     1,
			ids:       []string{"api"},
		},
		{
			name:      "notice before services",
			resources: []*organizationpb.IntegrationResourceRef{notice, service("api"), service("web")},
			count:     2,
			truncated: true,
			ids:       []string{"api"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resources, count, truncated := serializeAgentIntegrationResources(test.resources, 1)

			assert.Equal(t, test.count, count)
			assert.Equal(t, test.truncated, truncated)
			require.Len(t, resources, len(test.ids))
			for i, id := range test.ids {
				assert.Equal(t, id, resources[i].ID)
			}
		})
	}
}
