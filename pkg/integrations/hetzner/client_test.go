package hetzner

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func hetznerResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func testClient(responses ...*http.Response) (*Client, *contexts.HTTPContext) {
	httpCtx := &contexts.HTTPContext{Responses: responses}
	return &Client{Token: "token", BaseURL: defaultHetznerBaseURL, http: httpCtx}, httpCtx
}

func requestPaths(t *testing.T, httpCtx *contexts.HTTPContext) []string {
	t.Helper()

	paths := make([]string, 0, len(httpCtx.Requests))
	for _, request := range httpCtx.Requests {
		paths = append(paths, request.URL.Path+"?"+request.URL.RawQuery)
	}
	return paths
}

func TestClient_ListServers_FollowsEveryPage(t *testing.T) {
	t.Run("collects servers from all pages", func(t *testing.T) {
		client, httpCtx := testClient(
			hetznerResponse(200, `{
				"servers": [{"id": 1, "name": "server-1"}, {"id": 2, "name": "server-2"}],
				"meta": {"pagination": {"page": 1, "per_page": 50, "next_page": 2, "last_page": 2, "total_entries": 3}}
			}`),
			hetznerResponse(200, `{
				"servers": [{"id": 3, "name": "server-3"}],
				"meta": {"pagination": {"page": 2, "per_page": 50, "next_page": null, "last_page": 2, "total_entries": 3}}
			}`),
		)

		servers, err := client.ListServers()

		require.NoError(t, err)
		require.Len(t, servers, 3)
		assert.Equal(t, []string{"server-1", "server-2", "server-3"}, []string{servers[0].Name, servers[1].Name, servers[2].Name})
		assert.Equal(t, "3", servers[2].ID)
		assert.Equal(t, []string{
			"/v1/servers?per_page=50&page=1",
			"/v1/servers?per_page=50&page=2",
		}, requestPaths(t, httpCtx))
	})

	t.Run("stops after a page without a next one", func(t *testing.T) {
		client, httpCtx := testClient(
			hetznerResponse(200, `{
				"servers": [{"id": 1, "name": "server-1"}],
				"meta": {"pagination": {"page": 1, "per_page": 50, "next_page": null}}
			}`),
		)

		servers, err := client.ListServers()

		require.NoError(t, err)
		require.Len(t, servers, 1)
		assert.Len(t, requestPaths(t, httpCtx), 1)
	})

	t.Run("stops when next_page does not advance", func(t *testing.T) {
		client, httpCtx := testClient(
			hetznerResponse(200, `{
				"servers": [{"id": 1, "name": "server-1"}],
				"meta": {"pagination": {"page": 1, "per_page": 50, "next_page": 1}}
			}`),
		)

		servers, err := client.ListServers()

		require.NoError(t, err)
		require.Len(t, servers, 1)
		assert.Len(t, requestPaths(t, httpCtx), 1)
	})

	t.Run("surfaces an API error raised on a later page", func(t *testing.T) {
		client, _ := testClient(
			hetznerResponse(200, `{
				"servers": [{"id": 1, "name": "server-1"}],
				"meta": {"pagination": {"page": 1, "per_page": 50, "next_page": 2}}
			}`),
			hetznerResponse(503, `{"error": {"code": "unavailable", "message": "service unavailable"}}`),
		)

		servers, err := client.ListServers()

		require.Error(t, err)
		assert.Nil(t, servers)
		assert.ErrorContains(t, err, "service unavailable")
	})
}

func TestClient_ListResources_PaginateEveryCollection(t *testing.T) {
	t.Run("load balancers", func(t *testing.T) {
		client, httpCtx := testClient(
			hetznerResponse(200, `{"load_balancers": [{"id": 1, "name": "lb-1"}], "meta": {"pagination": {"next_page": 2}}}`),
			hetznerResponse(200, `{"load_balancers": [{"id": 2, "name": "lb-2"}], "meta": {"pagination": {"next_page": null}}}`),
		)

		loadBalancers, err := client.ListLoadBalancers()

		require.NoError(t, err)
		require.Len(t, loadBalancers, 2)
		assert.Equal(t, "/v1/load_balancers?per_page=50&page=2", requestPaths(t, httpCtx)[1])
	})

	t.Run("firewalls", func(t *testing.T) {
		client, httpCtx := testClient(
			hetznerResponse(200, `{"firewalls": [{"id": 1, "name": "firewall-1"}], "meta": {"pagination": {"next_page": 2}}}`),
			hetznerResponse(200, `{"firewalls": [{"id": 2, "name": "firewall-2"}], "meta": {"pagination": {"next_page": null}}}`),
		)

		firewalls, err := client.ListFirewalls()

		require.NoError(t, err)
		require.Len(t, firewalls, 2)
		assert.Equal(t, 2, firewalls[1].ID)
		assert.Equal(t, "/v1/firewalls?per_page=50&page=2", requestPaths(t, httpCtx)[1])
	})

	t.Run("server types", func(t *testing.T) {
		client, httpCtx := testClient(
			hetznerResponse(200, `{"server_types": [{"id": 1, "name": "cx22", "cores": 2}], "meta": {"pagination": {"next_page": 2}}}`),
			hetznerResponse(200, `{"server_types": [{"id": 2, "name": "cpx51", "cores": 16}], "meta": {"pagination": {"next_page": null}}}`),
		)

		serverTypes, err := client.ListServerTypes()

		require.NoError(t, err)
		require.Len(t, serverTypes, 2)
		assert.Equal(t, "cpx51", serverTypes[1].Name)
		assert.Equal(t, "/v1/server_types?per_page=50&page=2", requestPaths(t, httpCtx)[1])
	})

	t.Run("load balancer types", func(t *testing.T) {
		client, httpCtx := testClient(
			hetznerResponse(200, `{"load_balancer_types": [{"id": 1, "name": "lb11"}], "meta": {"pagination": {"next_page": 2}}}`),
			hetznerResponse(200, `{"load_balancer_types": [{"id": 2, "name": "lb21"}], "meta": {"pagination": {"next_page": null}}}`),
		)

		types, err := client.ListLoadBalancerTypes()

		require.NoError(t, err)
		require.Len(t, types, 2)
		assert.Equal(t, "lb21", types[1].Name)
		assert.Equal(t, "/v1/load_balancer_types?per_page=50&page=2", requestPaths(t, httpCtx)[1])
	})

	t.Run("locations", func(t *testing.T) {
		client, httpCtx := testClient(
			hetznerResponse(200, `{"locations": [{"id": 1, "name": "fsn1", "city": "Falkenstein", "country": "DE"}], "meta": {"pagination": {"next_page": 2}}}`),
			hetznerResponse(200, `{"locations": [{"id": 2, "name": "nbg1", "city": "Nuremberg", "country": "DE"}], "meta": {"pagination": {"next_page": null}}}`),
		)

		locations, err := client.ListLocations()

		require.NoError(t, err)
		require.Len(t, locations, 2)
		assert.Equal(t, "nbg1", locations[1].Name)
		assert.Equal(t, "/v1/locations?per_page=50&page=2", requestPaths(t, httpCtx)[1])
	})
}

func TestClient_ListImages_DropsDuplicatesAcrossPages(t *testing.T) {
	client, _ := testClient(
		hetznerResponse(200, `{"images": [{"id": 1, "name": "ubuntu-24.04"}, {"id": 2, "name": "debian-12"}], "meta": {"pagination": {"next_page": 2}}}`),
		hetznerResponse(200, `{"images": [{"id": 2, "name": "debian-12"}, {"id": 3, "name": "snapshot"}], "meta": {"pagination": {"next_page": null}}}`),
	)

	images, err := client.ListImages()

	require.NoError(t, err)
	require.Len(t, images, 3)
	assert.Equal(t, []int{1, 2, 3}, []int{images[0].ID, images[1].ID, images[2].ID})
}

func TestClient_ListServers_ReadsEveryPageOfAFullAccount(t *testing.T) {
	// A single page tops out at 50 entries, which is where the listings used to
	// stop and hide the rest of the account.
	first := make([]string, 0, listPerPage)
	for i := 1; i <= listPerPage; i++ {
		first = append(first, fmt.Sprintf(`{"id": %d, "name": "server-%d"}`, i, i))
	}

	client, _ := testClient(
		hetznerResponse(200, fmt.Sprintf(`{"servers": [%s], "meta": {"pagination": {"page": 1, "next_page": 2}}}`, strings.Join(first, ","))),
		hetznerResponse(200, `{"servers": [{"id": 51, "name": "server-51"}], "meta": {"pagination": {"page": 2, "next_page": null}}}`),
	)

	servers, err := client.ListServers()

	require.NoError(t, err)
	require.Len(t, servers, listPerPage+1)
	assert.Equal(t, "server-51", servers[listPerPage].Name)
}
