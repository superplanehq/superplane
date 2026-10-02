package hetzner

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestHetzner_ListResources_FiltersLocationsByAServerTypeOnALaterPage(t *testing.T) {
	integration := &Hetzner{}

	// The Location picker narrows to the locations that sell the chosen server
	// type. Resolving that type reads the server type listing, which used to end
	// at the first page and leave the filter unapplied.
	resources, err := integration.ListResources("location", core.ListResourcesContext{
		HTTP: &contexts.HTTPContext{Responses: []*http.Response{
			hetznerResponse(200, `{"locations": [{"id": 1, "name": "fsn1", "city": "Falkenstein", "country": "DE"}, {"id": 2, "name": "hil", "city": "Hillsboro", "country": "US"}], "meta": {"pagination": {"next_page": null}}}`),
			hetznerResponse(200, `{"server_types": [{"id": 1, "name": "cx22", "prices": [{"location": "fsn1"}, {"location": "hil"}]}], "meta": {"pagination": {"next_page": 2}}}`),
			hetznerResponse(200, `{"server_types": [{"id": 2, "name": "ccx63", "prices": [{"location": "fsn1"}]}], "meta": {"pagination": {"next_page": null}}}`),
		}},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "token"}},
		Parameters:  map[string]string{"serverType": "ccx63"},
	})

	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "fsn1", resources[0].ID)
	assert.Equal(t, "Falkenstein, DE (fsn1)", resources[0].Name)
}
