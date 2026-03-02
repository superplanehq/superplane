package firehydrant

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func newListResourcesContext(responses []*http.Response) core.ListResourcesContext {
	return core.ListResourcesContext{
		HTTP: &contexts.HTTPContext{Responses: responses},
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		},
	}
}

func newListResourcesContextWithoutKey() core.ListResourcesContext {
	return core.ListResourcesContext{
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{},
		},
	}
}

func Test__ListResources__Severity(t *testing.T) {
	fh := &FireHydrant{}

	t.Run("returns severity resources", func(t *testing.T) {
		ctx := newListResourcesContext([]*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"data": [
						{"slug": "SEV1", "description": "Critical", "type": "severity"},
						{"slug": "SEV2", "description": "High", "type": "severity"}
					]
				}`)),
			},
		})

		resources, err := fh.ListResources("severity", ctx)

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "severity", resources[0].Type)
		assert.Equal(t, "SEV1", resources[0].Name)
		assert.Equal(t, "SEV1", resources[0].ID)
		assert.Equal(t, "severity", resources[1].Type)
		assert.Equal(t, "SEV2", resources[1].Name)
		assert.Equal(t, "SEV2", resources[1].ID)
	})

	t.Run("empty list -> returns empty slice", func(t *testing.T) {
		ctx := newListResourcesContext([]*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data": []}`)),
			},
		})

		resources, err := fh.ListResources("severity", ctx)

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		ctx := newListResourcesContext([]*http.Response{
			{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader(`{"error": "unauthorized"}`)),
			},
		})

		_, err := fh.ListResources("severity", ctx)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to list severities")
	})

	t.Run("missing API key -> returns error", func(t *testing.T) {
		ctx := newListResourcesContextWithoutKey()

		_, err := fh.ListResources("severity", ctx)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create client")
	})
}

func Test__ListResources__Priority(t *testing.T) {
	fh := &FireHydrant{}

	t.Run("returns priority resources", func(t *testing.T) {
		ctx := newListResourcesContext([]*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"data": [
						{"slug": "P1", "description": "Highest"},
						{"slug": "P2", "description": "High"},
						{"slug": "P3", "description": "Medium"}
					]
				}`)),
			},
		})

		resources, err := fh.ListResources("priority", ctx)

		require.NoError(t, err)
		require.Len(t, resources, 3)
		assert.Equal(t, "priority", resources[0].Type)
		assert.Equal(t, "P1", resources[0].Name)
		assert.Equal(t, "P1", resources[0].ID)
		assert.Equal(t, "P2", resources[1].Name)
		assert.Equal(t, "P3", resources[2].Name)
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		ctx := newListResourcesContext([]*http.Response{
			{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader(`{"error": "server error"}`)),
			},
		})

		_, err := fh.ListResources("priority", ctx)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to list priorities")
	})

	t.Run("missing API key -> returns error", func(t *testing.T) {
		ctx := newListResourcesContextWithoutKey()

		_, err := fh.ListResources("priority", ctx)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create client")
	})
}

func Test__ListResources__Service(t *testing.T) {
	fh := &FireHydrant{}

	t.Run("returns service resources", func(t *testing.T) {
		ctx := newListResourcesContext([]*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"data": [
						{"id": "svc-001", "name": "API Gateway", "description": "Main API"},
						{"id": "svc-002", "name": "Auth Service", "description": "Authentication"}
					]
				}`)),
			},
		})

		resources, err := fh.ListResources("service", ctx)

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "service", resources[0].Type)
		assert.Equal(t, "API Gateway", resources[0].Name)
		assert.Equal(t, "svc-001", resources[0].ID)
		assert.Equal(t, "service", resources[1].Type)
		assert.Equal(t, "Auth Service", resources[1].Name)
		assert.Equal(t, "svc-002", resources[1].ID)
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		ctx := newListResourcesContext([]*http.Response{
			{
				StatusCode: http.StatusForbidden,
				Body:       io.NopCloser(strings.NewReader(`{"error": "forbidden"}`)),
			},
		})

		_, err := fh.ListResources("service", ctx)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to list services")
	})

	t.Run("missing API key -> returns error", func(t *testing.T) {
		ctx := newListResourcesContextWithoutKey()

		_, err := fh.ListResources("service", ctx)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create client")
	})
}

func Test__ListResources__Team(t *testing.T) {
	fh := &FireHydrant{}

	t.Run("returns team resources", func(t *testing.T) {
		ctx := newListResourcesContext([]*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"data": [
						{"id": "team-001", "name": "Platform"},
						{"id": "team-002", "name": "Infrastructure"}
					]
				}`)),
			},
		})

		resources, err := fh.ListResources("team", ctx)

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "team", resources[0].Type)
		assert.Equal(t, "Platform", resources[0].Name)
		assert.Equal(t, "team-001", resources[0].ID)
		assert.Equal(t, "team", resources[1].Type)
		assert.Equal(t, "Infrastructure", resources[1].Name)
		assert.Equal(t, "team-002", resources[1].ID)
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		ctx := newListResourcesContext([]*http.Response{
			{
				StatusCode: http.StatusBadGateway,
				Body:       io.NopCloser(strings.NewReader(`{"error": "bad gateway"}`)),
			},
		})

		_, err := fh.ListResources("team", ctx)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to list teams")
	})

	t.Run("missing API key -> returns error", func(t *testing.T) {
		ctx := newListResourcesContextWithoutKey()

		_, err := fh.ListResources("team", ctx)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create client")
	})
}

func Test__ListResources__UnknownType(t *testing.T) {
	fh := &FireHydrant{}

	t.Run("unknown resource type -> returns empty slice", func(t *testing.T) {
		resources, err := fh.ListResources("unknown", core.ListResourcesContext{})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})
}
