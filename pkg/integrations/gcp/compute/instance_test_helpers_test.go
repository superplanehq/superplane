package compute

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// mockInstanceClient is a shared Client mock for the instance-targeting
// components (manage power, update machine type, get metrics).
type mockInstanceClient struct {
	projectID  string
	getFunc    func(ctx context.Context, path string) ([]byte, error)
	postFunc   func(ctx context.Context, path string, body any) ([]byte, error)
	getURLFunc func(ctx context.Context, fullURL string) ([]byte, error)
}

func (m *mockInstanceClient) Get(ctx context.Context, path string) ([]byte, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, path)
	}
	return nil, fmt.Errorf("Get not implemented")
}

func (m *mockInstanceClient) Post(ctx context.Context, path string, body any) ([]byte, error) {
	if m.postFunc != nil {
		return m.postFunc(ctx, path, body)
	}
	return nil, fmt.Errorf("Post not implemented")
}

func (m *mockInstanceClient) Delete(ctx context.Context, path string) ([]byte, error) {
	return nil, fmt.Errorf("Delete not implemented")
}

func (m *mockInstanceClient) GetURL(ctx context.Context, fullURL string) ([]byte, error) {
	if m.getURLFunc != nil {
		return m.getURLFunc(ctx, fullURL)
	}
	return nil, fmt.Errorf("GetURL not implemented")
}

func (m *mockInstanceClient) ProjectID() string {
	return m.projectID
}

// instanceGetJSON builds an instances.get response body matching instanceGetResp.
func instanceGetJSON(id, name, zone, status, machineType string) []byte {
	b, _ := json.Marshal(map[string]any{
		"id":          id,
		"name":        name,
		"selfLink":    fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/my-project/zones/%s/instances/%s", zone, name),
		"status":      status,
		"zone":        fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/my-project/zones/%s", zone),
		"machineType": fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/my-project/zones/%s/machineTypes/%s", zone, machineType),
		"networkInterfaces": []map[string]any{
			{
				"networkIP":     "10.0.0.2",
				"accessConfigs": []map[string]any{{"natIP": "34.1.2.3"}},
			},
		},
	})
	return b
}

// isOperationPath reports whether a Get path targets a zone operation rather
// than an instance.
func isOperationPath(path string) bool {
	return strings.Contains(path, "/operations/")
}
