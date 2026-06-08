package compute

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStaticIPClient is a configurable Client mock used by the static IP
// component tests. Each test wires only the funcs it needs.
type mockStaticIPClient struct {
	projectID  string
	getFunc    func(ctx context.Context, path string) ([]byte, error)
	postFunc   func(ctx context.Context, path string, body any) ([]byte, error)
	deleteFunc func(ctx context.Context, path string) ([]byte, error)
}

func (m *mockStaticIPClient) Get(ctx context.Context, path string) ([]byte, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, path)
	}
	return nil, fmt.Errorf("unexpected Get(%s)", path)
}

func (m *mockStaticIPClient) Post(ctx context.Context, path string, body any) ([]byte, error) {
	if m.postFunc != nil {
		return m.postFunc(ctx, path, body)
	}
	return nil, fmt.Errorf("unexpected Post(%s)", path)
}

func (m *mockStaticIPClient) Delete(ctx context.Context, path string) ([]byte, error) {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, path)
	}
	return nil, fmt.Errorf("unexpected Delete(%s)", path)
}

func (m *mockStaticIPClient) GetURL(ctx context.Context, fullURL string) ([]byte, error) {
	return nil, fmt.Errorf("unexpected GetURL(%s)", fullURL)
}

func (m *mockStaticIPClient) ProjectID() string {
	return m.projectID
}

// addressJSON serializes a regional address resource as the API would return it.
func addressJSON(name, address, region, status, addressType, tier string) []byte {
	selfLink := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/my-project/regions/%s/addresses/%s", region, name)
	b, _ := json.Marshal(map[string]any{
		"name":        name,
		"address":     address,
		"region":      fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/my-project/regions/%s", region),
		"selfLink":    selfLink,
		"status":      status,
		"addressType": addressType,
		"networkTier": tier,
	})
	return b
}

func Test__ParseAddressPath(t *testing.T) {
	t.Run("relative path", func(t *testing.T) {
		project, region, name, err := parseAddressPath("regions/us-central1/addresses/web-ip")
		require.NoError(t, err)
		assert.Equal(t, "", project)
		assert.Equal(t, "us-central1", region)
		assert.Equal(t, "web-ip", name)
	})

	t.Run("full selfLink URL", func(t *testing.T) {
		selfLink := "https://www.googleapis.com/compute/v1/projects/elffie/regions/europe-west1/addresses/db-ip"
		project, region, name, err := parseAddressPath(selfLink)
		require.NoError(t, err)
		assert.Equal(t, "elffie", project)
		assert.Equal(t, "europe-west1", region)
		assert.Equal(t, "db-ip", name)
	})

	t.Run("project-qualified relative path", func(t *testing.T) {
		project, region, name, err := parseAddressPath("projects/elffie/regions/us-east1/addresses/api-ip")
		require.NoError(t, err)
		assert.Equal(t, "elffie", project)
		assert.Equal(t, "us-east1", region)
		assert.Equal(t, "api-ip", name)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		_, region, name, err := parseAddressPath("  regions/us-central1/addresses/web-ip  ")
		require.NoError(t, err)
		assert.Equal(t, "us-central1", region)
		assert.Equal(t, "web-ip", name)
	})

	t.Run("plain name is rejected", func(t *testing.T) {
		_, _, _, err := parseAddressPath("just-a-name")
		require.Error(t, err)
	})

	t.Run("empty value is rejected", func(t *testing.T) {
		_, _, _, err := parseAddressPath("")
		require.Error(t, err)
	})

	t.Run("missing addresses segment is rejected", func(t *testing.T) {
		_, _, _, err := parseAddressPath("regions/us-central1/foo/web-ip")
		require.Error(t, err)
	})
}

func Test__ListStaticIPResources(t *testing.T) {
	t.Run("returns external addresses across all regions keyed by selfLink", func(t *testing.T) {
		mc := &mockStaticIPClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				assert.True(t, strings.Contains(path, "/aggregated/addresses"))
				b, _ := json.Marshal(map[string]any{
					"items": map[string]any{
						"regions/us-central1": map[string]any{
							"addresses": []map[string]any{
								{
									"name":        "web-ip",
									"address":     "34.1.2.3",
									"region":      "https://www.googleapis.com/compute/v1/projects/my-project/regions/us-central1",
									"selfLink":    "https://www.googleapis.com/compute/v1/projects/my-project/regions/us-central1/addresses/web-ip",
									"addressType": "EXTERNAL",
								},
								{
									"name":        "internal-ip",
									"address":     "10.0.0.5",
									"selfLink":    "https://www.googleapis.com/compute/v1/projects/my-project/regions/us-central1/addresses/internal-ip",
									"addressType": "INTERNAL",
								},
							},
						},
						"regions/europe-west1": map[string]any{
							"addresses": []map[string]any{
								{
									"name":        "eu-ip",
									"address":     "35.5.5.5",
									"region":      "https://www.googleapis.com/compute/v1/projects/my-project/regions/europe-west1",
									"selfLink":    "https://www.googleapis.com/compute/v1/projects/my-project/regions/europe-west1/addresses/eu-ip",
									"addressType": "EXTERNAL",
								},
							},
						},
						"regions/us-east1": map[string]any{
							"warning": map[string]any{"code": "NO_RESULTS_ON_PAGE"},
						},
					},
				})
				return b, nil
			},
		}

		out, err := ListStaticIPResources(context.Background(), mc, "", "")
		require.NoError(t, err)
		require.Len(t, out, 2) // two EXTERNAL across regions; INTERNAL skipped

		labels := map[string]string{}
		for _, r := range out {
			assert.Equal(t, ResourceTypeStaticIP, r.Type)
			labels[r.ID] = r.Name
		}
		assert.Equal(t, "web-ip (34.1.2.3, us-central1)", labels["https://www.googleapis.com/compute/v1/projects/my-project/regions/us-central1/addresses/web-ip"])
		assert.Equal(t, "eu-ip (35.5.5.5, europe-west1)", labels["https://www.googleapis.com/compute/v1/projects/my-project/regions/europe-west1/addresses/eu-ip"])
	})

	t.Run("filters to the selected VM's region", func(t *testing.T) {
		mc := &mockStaticIPClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				b, _ := json.Marshal(map[string]any{
					"items": map[string]any{
						"regions/us-central1": map[string]any{
							"addresses": []map[string]any{
								{
									"name":        "web-ip",
									"address":     "34.1.2.3",
									"region":      "https://www.googleapis.com/compute/v1/projects/my-project/regions/us-central1",
									"selfLink":    "https://www.googleapis.com/compute/v1/projects/my-project/regions/us-central1/addresses/web-ip",
									"addressType": "EXTERNAL",
								},
							},
						},
						"regions/europe-west1": map[string]any{
							"addresses": []map[string]any{
								{
									"name":        "eu-ip",
									"address":     "35.5.5.5",
									"region":      "https://www.googleapis.com/compute/v1/projects/my-project/regions/europe-west1",
									"selfLink":    "https://www.googleapis.com/compute/v1/projects/my-project/regions/europe-west1/addresses/eu-ip",
									"addressType": "EXTERNAL",
								},
							},
						},
					},
				})
				return b, nil
			},
		}

		// Instance in us-central1-a -> only the us-central1 IP should remain.
		out, err := ListStaticIPResources(context.Background(), mc, "", "zones/us-central1-a/instances/my-vm")
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.Equal(t, "web-ip (34.1.2.3, us-central1)", out[0].Name)
	})
}
