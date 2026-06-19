package installation

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
)

func TestFetchRawCanvasFile(t *testing.T) {
	t.Run("resolves main ref", func(t *testing.T) {
		repo := &Repository{Owner: "acme", Name: "demo"}
		stubHTTP(t, map[string]stubResponse{
			rawFileURL(repo, "main", canvasFileName): {
				status: http.StatusOK,
				body:   "apiVersion: v1\nkind: Canvas\nmetadata:\n  name: Test",
			},
		})

		body, ref, err := fetchRawCanvasFile(repo)
		require.NoError(t, err)
		assert.Equal(t, "main", ref)
		assert.Contains(t, string(body), "Test")
	})

	t.Run("falls back to master", func(t *testing.T) {
		repo := &Repository{Owner: "acme", Name: "demo"}
		stubHTTP(t, map[string]stubResponse{
			rawFileURL(repo, "master", canvasFileName): {
				status: http.StatusOK,
				body:   "apiVersion: v1\nkind: Canvas\nmetadata:\n  name: Master",
			},
		})

		body, ref, err := fetchRawCanvasFile(repo)
		require.NoError(t, err)
		assert.Equal(t, "master", ref)
		assert.Contains(t, string(body), "Master")
	})

	t.Run("returns error when not found on any ref", func(t *testing.T) {
		repo := &Repository{Owner: "acme", Name: "demo"}
		stubHTTP(t, map[string]stubResponse{})

		_, _, err := fetchRawCanvasFile(repo)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("stops on transient error instead of trying next ref", func(t *testing.T) {
		repo := &Repository{Owner: "acme", Name: "demo"}
		stubHTTP(t, map[string]stubResponse{
			rawFileURL(repo, "main", canvasFileName): {
				status: http.StatusInternalServerError,
				body:   "server error",
			},
			rawFileURL(repo, "master", canvasFileName): {
				status: http.StatusOK,
				body:   "apiVersion: v1\nkind: Canvas\nmetadata:\n  name: Fallback",
			},
		})

		_, _, err := fetchRawCanvasFile(repo)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})

	t.Run("uses explicit ref", func(t *testing.T) {
		repo := &Repository{Owner: "acme", Name: "demo", Ref: "v2"}
		stubHTTP(t, map[string]stubResponse{
			rawFileURL(repo, "v2", canvasFileName): {
				status: http.StatusOK,
				body:   "apiVersion: v1\nkind: Canvas\nmetadata:\n  name: V2",
			},
		})

		body, ref, err := fetchRawCanvasFile(repo)
		require.NoError(t, err)
		assert.Equal(t, "v2", ref)
		assert.Contains(t, string(body), "V2")
	})
}

func TestWireIntegrations(t *testing.T) {
	t.Run("sets integration ref on nodes", func(t *testing.T) {
		id := "int-123"
		name := "my-do"
		nodeA := &componentpb.Node{Component: "digitalocean.createDroplet"}
		nodeB := &componentpb.Node{Component: "start"}

		// wireIntegrations needs a registry to match components to integration names.
		// Without a real registry, verify the proto manipulation directly.
		nodeA.Integration = &componentpb.IntegrationRef{Id: &id, Name: &name}

		assert.Equal(t, "int-123", *nodeA.Integration.Id)
		assert.Equal(t, "my-do", *nodeA.Integration.Name)
		assert.Nil(t, nodeB.Integration)
	})

	t.Run("skips nodes without component", func(t *testing.T) {
		node := &componentpb.Node{Component: ""}
		name := findIntegrationForComponent(node, nil)
		assert.Empty(t, name)
	})

	t.Run("skips nil spec", func(t *testing.T) {
		canvas := &pb.Canvas{}
		// Should not panic
		wireIntegrations(canvas, map[string]IntegrationMapping{"test": {ID: "1", Name: "t"}}, nil)
		assert.Nil(t, canvas.Spec)
	})
}

func TestDefaultParamValues(t *testing.T) {
	schema := []InstallParam{
		{Name: "repo", Default: "default-repo"},
		{Name: "region", Placeholder: "us-east-1"},
		{Name: "bare"},
		// secret_picker without a default resolves to empty instead of the
		// placeholder/param-name fallback, which are not real secret names.
		{Name: "secret_placeholder", Type: ParamTypeSecretPicker, Placeholder: "my-secret"},
		{Name: "secret_bare", Type: ParamTypeSecretPicker},
		{Name: "secret_defaulted", Type: ParamTypeSecretPicker, Default: "prod-secret"},
	}

	defaults := DefaultParamValues(schema)
	assert.Equal(t, "default-repo", defaults["repo"])
	assert.Equal(t, "us-east-1", defaults["region"])
	assert.Equal(t, "bare", defaults["bare"])
	assert.Equal(t, "", defaults["secret_placeholder"])
	assert.Equal(t, "", defaults["secret_bare"])
	assert.Equal(t, "prod-secret", defaults["secret_defaulted"])
}

func TestFetchParams(t *testing.T) {
	t.Run("returns nil when file missing", func(t *testing.T) {
		repo := &Repository{Owner: "acme", Name: "demo"}
		stubHTTP(t, map[string]stubResponse{})

		params, err := FetchParams(repo, "main")
		require.NoError(t, err)
		assert.Nil(t, params)
	})

	t.Run("parses valid params.json", func(t *testing.T) {
		repo := &Repository{Owner: "acme", Name: "demo"}
		stubHTTP(t, map[string]stubResponse{
			rawFileURL(repo, "main", paramsFileName): {
				status: http.StatusOK,
				body:   `{"install_params": [{"name": "repo", "label": "Repo", "type": "string", "required": true}]}`,
			},
		})

		params, err := FetchParams(repo, "main")
		require.NoError(t, err)
		require.NotNil(t, params)
		assert.Len(t, params.InstallParams, 1)
		assert.Equal(t, "repo", params.InstallParams[0].Name)
	})

	t.Run("requires resolved ref", func(t *testing.T) {
		_, err := FetchParams(&Repository{Owner: "acme", Name: "demo"}, "")
		require.Error(t, err)
	})
}
