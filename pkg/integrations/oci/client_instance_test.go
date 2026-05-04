package oci

import (
	"net/http"
	neturl "net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Client__EscapesInstanceIDInPath(t *testing.T) {
	instanceID := "ocid1.instance.oc1.test/with slash"
	escapedPath := "/20160918/instances/" + neturl.PathEscape(instanceID)
	displayName := "renamed"

	tests := []struct {
		name      string
		response  *http.Response
		call      func(*Client) error
		queryName string
	}{
		{
			name:     "get",
			response: ociMockResponse(http.StatusOK, ociInstanceBody(instanceStateRunning)),
			call: func(client *Client) error {
				_, err := client.GetInstance(instanceID)
				return err
			},
		},
		{
			name:     "update",
			response: ociMockResponse(http.StatusOK, ociInstanceBody(instanceStateRunning)),
			call: func(client *Client) error {
				_, err := client.UpdateInstance(instanceID, UpdateInstanceRequest{DisplayName: &displayName})
				return err
			},
		},
		{
			name:     "action",
			response: ociMockResponse(http.StatusOK, ociInstanceBody(instanceStateStopped)),
			call: func(client *Client) error {
				_, err := client.InstanceAction(instanceID, "STOP")
				return err
			},
			queryName: "action",
		},
		{
			name:     "terminate",
			response: ociMockResponse(http.StatusNoContent, ``),
			call: func(client *Client) error {
				return client.TerminateInstance(instanceID, true)
			},
			queryName: "preserveBootVolume",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpCtx := &contexts.HTTPContext{
				Responses: []*http.Response{tt.response},
			}
			client, err := NewClient(httpCtx, ociIntegrationContext())
			require.NoError(t, err)

			require.NoError(t, tt.call(client))
			require.Len(t, httpCtx.Requests, 1)
			assert.Equal(t, escapedPath, httpCtx.Requests[0].URL.EscapedPath())
			if tt.queryName != "" {
				assert.NotEmpty(t, httpCtx.Requests[0].URL.Query().Get(tt.queryName))
			}
		})
	}
}
