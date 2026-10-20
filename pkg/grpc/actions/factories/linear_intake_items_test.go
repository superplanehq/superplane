package factories

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/linear"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__LinearIntakeItemSource__SearchScope(t *testing.T) {
	for _, tc := range []struct{ name, fields, reason string }{
		{"allowed", `"project":{"id":"p1"},"labels":{"nodes":[{"name":"bug"}]}`, ""},
		{"wrong project", `"project":{"id":"p2"},"labels":{"nodes":[{"name":"bug"}]}`, "This issue is not in the selected projects."},
		{"no project", `"project":null,"labels":{"nodes":[{"name":"bug"}]}`, "This issue has no project."},
		{"completed", `"project":{"id":"p1"},"state":{"type":"completed"},"labels":{"nodes":[{"name":"bug"}]}`, "This issue is closed or canceled."},
		{"canceled", `"project":{"id":"p1"},"state":{"type":"canceled"},"labels":{"nodes":[{"name":"bug"}]}`, "This issue is closed or canceled."},
		{"labels", `"project":{"id":"p1"},"labels":{"nodes":[{"name":"chore"}]}`, "This issue has no matching intake labels."},
		{"multiple restrictions", `"project":null,"state":{"type":"completed"},"labels":{"nodes":[]}`, "This issue has no project. This issue is closed or canceled. This issue has no matching intake labels."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			exact := `{"id":"exact","identifier":"ENG-123","title":"Fix checkout","description":"Details",` + tc.fields + `}`
			response := func(body string) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}
			}
			httpContext := &contexts.HTTPContext{Responses: []*http.Response{
				response(`{"data":{"issues":{"nodes":[{"id":"unlabeled","project":{"id":"p1"}},{"id":"allowed-text","project":{"id":"p1"},"labels":{"nodes":[{"name":"bug"}]}}]}}}`),
				response(`{"data":{"issues":{"nodes":[` + exact + `]}}}`),
				response(`{"data":{"issue":` + exact + `}}`),
			}}
			client, err := linear.NewClient(httpContext, &contexts.IntegrationContext{CurrentSecrets: map[string]core.IntegrationSecret{
				linear.OAuthAccessToken: {Name: linear.OAuthAccessToken, Value: []byte("token")},
			}})
			require.NoError(t, err)
			source := &linearIntakeItemSource{linear: client, projectIDs: []string{"p1"}, labels: []string{"bug"}}
			items, err := source.Search(context.Background(), "eng-123", 5)
			require.NoError(t, err)
			require.Len(t, items, 2)
			assert.Equal(t, "exact", items[0].ID)
			assert.Equal(t, tc.reason, items[0].DisabledReason)
			assert.Equal(t, "allowed-text", items[1].ID)
			// Preserve the issue body and carry restrictions through the API boundary.
			serialized := serializeFactoryIntakeItem(items[0])
			assert.Equal(t, "Details", serialized.GetBody())
			assert.Equal(t, tc.reason, serialized.GetDisabledReason())
			item, err := source.Get(context.Background(), "exact")
			if tc.reason != "" {
				require.ErrorIs(t, err, errIntakeItemNotFound)
				assert.Nil(t, item)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, item)
			assert.Equal(t, "exact", item.ID)
		})
	}
}
