package public

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestFactoryWebSocketRequiresFactoriesFeature(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	server, _, token := setupTestServer(r, t)

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, "ws-factory", "desc", "")
	require.NoError(t, err)
	require.NoError(t, models.DisableExperimentalFeature(r.Organization.ID, features.FeatureFactories))

	response := execRequest(server, requestParams{
		method:     http.MethodGet,
		path:       "/ws/factories/" + factory.ID.String() + "?organization_id=" + r.Organization.ID.String(),
		authCookie: token,
	})

	assert.Equal(t, http.StatusForbidden, response.Code)
}

func TestFactoryWebSocketRequiresExistingFactory(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	server, _, token := setupTestServer(r, t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))

	response := execRequest(server, requestParams{
		method:     http.MethodGet,
		path:       "/ws/factories/11111111-1111-1111-1111-111111111111?organization_id=" + r.Organization.ID.String(),
		authCookie: token,
	})

	assert.Equal(t, http.StatusNotFound, response.Code)
}

func TestFactoryWebSocketRejectsInvalidFactoryID(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	server, _, token := setupTestServer(r, t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))

	response := execRequest(server, requestParams{
		method:     http.MethodGet,
		path:       "/ws/factories/not-a-uuid?organization_id=" + r.Organization.ID.String(),
		authCookie: token,
	})

	assert.Equal(t, http.StatusBadRequest, response.Code)
}
