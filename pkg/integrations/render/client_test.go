package render

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func newTestClient(httpCtx *contexts.HTTPContext) *Client {
	return &Client{
		APIKey:  "rnd_test",
		BaseURL: defaultRenderBaseURL,
		http:    httpCtx,
	}
}

// ---------------------------------------------------------------------------
// GetService
// ---------------------------------------------------------------------------

func Test__Client__GetService(t *testing.T) {
	tests := []struct {
		name       string
		serviceID  string
		statusCode int
		body       string
		wantErr    bool
		errContains string
		validate   func(t *testing.T, svc *ServiceDetail)
	}{
		{
			name:       "200 OK -> returns service detail",
			serviceID:  "srv-abc123",
			statusCode: http.StatusOK,
			body:       `{"id":"srv-abc123","name":"my-api","type":"web_service","suspended":"not_suspended","autoDeploy":"yes","repo":"https://github.com/org/repo","branch":"main","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-02-01T00:00:00Z"}`,
			wantErr:    false,
			validate: func(t *testing.T, svc *ServiceDetail) {
				assert.Equal(t, "srv-abc123", svc.ID)
				assert.Equal(t, "my-api", svc.Name)
				assert.Equal(t, "web_service", svc.Type)
				assert.Equal(t, "not_suspended", svc.Suspended)
				assert.False(t, svc.IsSuspended())
				assert.Equal(t, "yes", svc.AutoDeploy)
				assert.Equal(t, "main", svc.Branch)
			},
		},
		{
			name:       "200 OK -> suspended service returns IsSuspended true",
			serviceID:  "srv-suspended",
			statusCode: http.StatusOK,
			body:       `{"id":"srv-suspended","name":"paused","type":"web_service","suspended":"suspended","autoDeploy":"no","repo":"","branch":"","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-02-01T00:00:00Z"}`,
			wantErr:    false,
			validate: func(t *testing.T, svc *ServiceDetail) {
				assert.Equal(t, "suspended", svc.Suspended)
				assert.True(t, svc.IsSuspended())
			},
		},
		{
			name:        "404 Not Found -> returns API error",
			serviceID:   "srv-missing",
			statusCode:  http.StatusNotFound,
			body:        `{"message":"service not found"}`,
			wantErr:     true,
			errContains: "404",
		},
		{
			name:        "429 Rate Limit -> returns API error",
			serviceID:   "srv-abc123",
			statusCode:  http.StatusTooManyRequests,
			body:        `{"message":"rate limit exceeded"}`,
			wantErr:     true,
			errContains: "429",
		},
		{
			name:        "empty serviceID -> returns validation error",
			serviceID:   "",
			statusCode:  0,
			body:        "",
			wantErr:     true,
			errContains: "serviceID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpCtx := &contexts.HTTPContext{}
			if tt.statusCode > 0 {
				httpCtx.Responses = []*http.Response{
					{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(strings.NewReader(tt.body)),
					},
				}
			}

			client := newTestClient(httpCtx)
			svc, err := client.GetService(tt.serviceID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, svc)
			if tt.validate != nil {
				tt.validate(t, svc)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetDeploy
// ---------------------------------------------------------------------------

func Test__Client__GetDeploy(t *testing.T) {
	tests := []struct {
		name        string
		serviceID   string
		deployID    string
		statusCode  int
		body        string
		wantErr     bool
		errContains string
		validate    func(t *testing.T, deploy DeployResponse)
	}{
		{
			name:       "200 OK -> returns deploy details",
			serviceID:  "srv-abc123",
			deployID:   "dep-xyz789",
			statusCode: http.StatusOK,
			body:       `{"id":"dep-xyz789","status":"live","createdAt":"2026-02-05T16:10:00Z","finishedAt":"2026-02-05T16:15:00Z"}`,
			wantErr:    false,
			validate: func(t *testing.T, deploy DeployResponse) {
				assert.Equal(t, "dep-xyz789", deploy.ID)
				assert.Equal(t, "live", deploy.Status)
				assert.Equal(t, "2026-02-05T16:10:00Z", deploy.CreatedAt)
				assert.Equal(t, "2026-02-05T16:15:00Z", deploy.FinishedAt)
			},
		},
		{
			name:        "404 Not Found -> returns API error",
			serviceID:   "srv-abc123",
			deployID:    "dep-missing",
			statusCode:  http.StatusNotFound,
			body:        `{"message":"deploy not found"}`,
			wantErr:     true,
			errContains: "404",
		},
		{
			name:        "429 Rate Limit -> returns API error",
			serviceID:   "srv-abc123",
			deployID:    "dep-xyz789",
			statusCode:  http.StatusTooManyRequests,
			body:        `{"message":"rate limit exceeded"}`,
			wantErr:     true,
			errContains: "429",
		},
		{
			name:        "empty serviceID -> returns validation error",
			serviceID:   "",
			deployID:    "dep-xyz789",
			wantErr:     true,
			errContains: "serviceID is required",
		},
		{
			name:        "empty deployID -> returns validation error",
			serviceID:   "srv-abc123",
			deployID:    "",
			wantErr:     true,
			errContains: "deployID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpCtx := &contexts.HTTPContext{}
			if tt.statusCode > 0 {
				httpCtx.Responses = []*http.Response{
					{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(strings.NewReader(tt.body)),
					},
				}
			}

			client := newTestClient(httpCtx)
			deploy, err := client.GetDeploy(tt.serviceID, tt.deployID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, deploy)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CancelDeploy
// ---------------------------------------------------------------------------

func Test__Client__CancelDeploy(t *testing.T) {
	tests := []struct {
		name        string
		serviceID   string
		deployID    string
		statusCode  int
		body        string
		wantErr     bool
		errContains string
		validate    func(t *testing.T, deploy DeployResponse, req *http.Request)
	}{
		{
			name:       "200 OK -> returns canceled deploy",
			serviceID:  "srv-abc123",
			deployID:   "dep-xyz789",
			statusCode: http.StatusOK,
			body:       `{"id":"dep-xyz789","status":"canceled","createdAt":"2026-02-05T16:10:00Z","finishedAt":"2026-02-05T16:12:00Z"}`,
			wantErr:    false,
			validate: func(t *testing.T, deploy DeployResponse, req *http.Request) {
				assert.Equal(t, "dep-xyz789", deploy.ID)
				assert.Equal(t, "canceled", deploy.Status)
				assert.Equal(t, http.MethodPost, req.Method)
				assert.Contains(t, req.URL.Path, "/services/srv-abc123/deploys/dep-xyz789/cancel")
			},
		},
		{
			name:        "404 Not Found -> returns API error",
			serviceID:   "srv-abc123",
			deployID:    "dep-missing",
			statusCode:  http.StatusNotFound,
			body:        `{"message":"deploy not found"}`,
			wantErr:     true,
			errContains: "404",
		},
		{
			name:        "429 Rate Limit -> returns API error",
			serviceID:   "srv-abc123",
			deployID:    "dep-xyz789",
			statusCode:  http.StatusTooManyRequests,
			body:        `{"message":"rate limit exceeded"}`,
			wantErr:     true,
			errContains: "429",
		},
		{
			name:        "empty serviceID -> returns validation error",
			serviceID:   "",
			deployID:    "dep-xyz789",
			wantErr:     true,
			errContains: "serviceID is required",
		},
		{
			name:        "empty deployID -> returns validation error",
			serviceID:   "srv-abc123",
			deployID:    "",
			wantErr:     true,
			errContains: "deployID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpCtx := &contexts.HTTPContext{}
			if tt.statusCode > 0 {
				httpCtx.Responses = []*http.Response{
					{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(strings.NewReader(tt.body)),
					},
				}
			}

			client := newTestClient(httpCtx)
			deploy, err := client.CancelDeploy(tt.serviceID, tt.deployID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.Len(t, httpCtx.Requests, 1)
			if tt.validate != nil {
				tt.validate(t, deploy, httpCtx.Requests[0])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Rollback
// ---------------------------------------------------------------------------

func Test__Client__Rollback(t *testing.T) {
	tests := []struct {
		name        string
		serviceID   string
		deployID    string
		statusCode  int
		body        string
		wantErr     bool
		errContains string
		validate    func(t *testing.T, deploy DeployResponse, req *http.Request)
	}{
		{
			name:       "200 OK -> returns new deploy from rollback",
			serviceID:  "srv-abc123",
			deployID:   "dep-old456",
			statusCode: http.StatusOK,
			body:       `{"id":"dep-new789","status":"created","createdAt":"2026-02-05T16:20:00Z","finishedAt":""}`,
			wantErr:    false,
			validate: func(t *testing.T, deploy DeployResponse, req *http.Request) {
				assert.Equal(t, "dep-new789", deploy.ID)
				assert.Equal(t, "created", deploy.Status)
				assert.Equal(t, http.MethodPost, req.Method)
				assert.Contains(t, req.URL.Path, "/services/srv-abc123/rollbacks")

				reqBody, readErr := io.ReadAll(req.Body)
				require.NoError(t, readErr)
				payload := map[string]any{}
				require.NoError(t, json.Unmarshal(reqBody, &payload))
				assert.Equal(t, "dep-old456", payload["deployId"])
			},
		},
		{
			name:        "404 Not Found -> returns API error",
			serviceID:   "srv-abc123",
			deployID:    "dep-missing",
			statusCode:  http.StatusNotFound,
			body:        `{"message":"deploy not found"}`,
			wantErr:     true,
			errContains: "404",
		},
		{
			name:        "429 Rate Limit -> returns API error",
			serviceID:   "srv-abc123",
			deployID:    "dep-old456",
			statusCode:  http.StatusTooManyRequests,
			body:        `{"message":"rate limit exceeded"}`,
			wantErr:     true,
			errContains: "429",
		},
		{
			name:        "empty serviceID -> returns validation error",
			serviceID:   "",
			deployID:    "dep-old456",
			wantErr:     true,
			errContains: "serviceID is required",
		},
		{
			name:        "empty deployID -> returns validation error",
			serviceID:   "srv-abc123",
			deployID:    "",
			wantErr:     true,
			errContains: "deployID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpCtx := &contexts.HTTPContext{}
			if tt.statusCode > 0 {
				httpCtx.Responses = []*http.Response{
					{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(strings.NewReader(tt.body)),
					},
				}
			}

			client := newTestClient(httpCtx)
			deploy, err := client.Rollback(tt.serviceID, tt.deployID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.Len(t, httpCtx.Requests, 1)
			if tt.validate != nil {
				tt.validate(t, deploy, httpCtx.Requests[0])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// PurgeCache
// ---------------------------------------------------------------------------

func Test__Client__PurgeCache(t *testing.T) {
	tests := []struct {
		name        string
		serviceID   string
		statusCode  int
		body        string
		wantErr     bool
		errContains string
		validate    func(t *testing.T, req *http.Request)
	}{
		{
			name:       "204 No Content -> success",
			serviceID:  "srv-abc123",
			statusCode: http.StatusNoContent,
			body:       "",
			wantErr:    false,
			validate: func(t *testing.T, req *http.Request) {
				assert.Equal(t, http.MethodDelete, req.Method)
				assert.Contains(t, req.URL.Path, "/services/srv-abc123/cache")
			},
		},
		{
			name:       "200 OK -> also success",
			serviceID:  "srv-abc123",
			statusCode: http.StatusOK,
			body:       `{}`,
			wantErr:    false,
		},
		{
			name:        "404 Not Found -> returns API error",
			serviceID:   "srv-missing",
			statusCode:  http.StatusNotFound,
			body:        `{"message":"service not found"}`,
			wantErr:     true,
			errContains: "404",
		},
		{
			name:        "429 Rate Limit -> returns API error",
			serviceID:   "srv-abc123",
			statusCode:  http.StatusTooManyRequests,
			body:        `{"message":"rate limit exceeded"}`,
			wantErr:     true,
			errContains: "429",
		},
		{
			name:        "empty serviceID -> returns validation error",
			serviceID:   "",
			wantErr:     true,
			errContains: "serviceID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpCtx := &contexts.HTTPContext{}
			if tt.statusCode > 0 {
				httpCtx.Responses = []*http.Response{
					{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(strings.NewReader(tt.body)),
					},
				}
			}

			client := newTestClient(httpCtx)
			err := client.PurgeCache(tt.serviceID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.Len(t, httpCtx.Requests, 1)
			if tt.validate != nil {
				tt.validate(t, httpCtx.Requests[0])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// UpdateEnvVars
// ---------------------------------------------------------------------------

func Test__Client__UpdateEnvVars(t *testing.T) {
	tests := []struct {
		name        string
		serviceID   string
		envVars     []EnvVar
		statusCode  int
		body        string
		wantErr     bool
		errContains string
		validate    func(t *testing.T, result []EnvVar, req *http.Request)
	}{
		{
			name:      "200 OK -> returns updated env vars (flat format)",
			serviceID: "srv-abc123",
			envVars: []EnvVar{
				{Key: "NODE_ENV", Value: "production"},
				{Key: "API_URL", Value: "https://api.example.com"},
			},
			statusCode: http.StatusOK,
			body:       `[{"key":"NODE_ENV","value":"production"},{"key":"API_URL","value":"https://api.example.com"}]`,
			wantErr:    false,
			validate: func(t *testing.T, result []EnvVar, req *http.Request) {
				require.Len(t, result, 2)
				assert.Equal(t, "NODE_ENV", result[0].Key)
				assert.Equal(t, "production", result[0].Value)
				assert.Equal(t, "API_URL", result[1].Key)

				assert.Equal(t, http.MethodPut, req.Method)
				assert.Contains(t, req.URL.Path, "/services/srv-abc123/env-vars")

				reqBody, readErr := io.ReadAll(req.Body)
				require.NoError(t, readErr)
				var sent []EnvVar
				require.NoError(t, json.Unmarshal(reqBody, &sent))
				require.Len(t, sent, 2)
				assert.Equal(t, "NODE_ENV", sent[0].Key)
			},
		},
		{
			name:      "200 OK -> returns updated env vars (cursor format)",
			serviceID: "srv-abc123",
			envVars: []EnvVar{
				{Key: "DB_HOST", Value: "localhost"},
			},
			statusCode: http.StatusOK,
			body:       `[{"cursor":"a","envVar":{"key":"DB_HOST","value":"localhost"}}]`,
			wantErr:    false,
			validate: func(t *testing.T, result []EnvVar, req *http.Request) {
				require.Len(t, result, 1)
				assert.Equal(t, "DB_HOST", result[0].Key)
				assert.Equal(t, "localhost", result[0].Value)
			},
		},
		{
			name:        "404 Not Found -> returns API error",
			serviceID:   "srv-missing",
			envVars:     []EnvVar{{Key: "K", Value: "V"}},
			statusCode:  http.StatusNotFound,
			body:        `{"message":"service not found"}`,
			wantErr:     true,
			errContains: "404",
		},
		{
			name:        "429 Rate Limit -> returns API error",
			serviceID:   "srv-abc123",
			envVars:     []EnvVar{{Key: "K", Value: "V"}},
			statusCode:  http.StatusTooManyRequests,
			body:        `{"message":"rate limit exceeded"}`,
			wantErr:     true,
			errContains: "429",
		},
		{
			name:        "empty serviceID -> returns validation error",
			serviceID:   "",
			envVars:     []EnvVar{{Key: "K", Value: "V"}},
			wantErr:     true,
			errContains: "serviceID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpCtx := &contexts.HTTPContext{}
			if tt.statusCode > 0 {
				httpCtx.Responses = []*http.Response{
					{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(strings.NewReader(tt.body)),
					},
				}
			}

			client := newTestClient(httpCtx)
			result, err := client.UpdateEnvVars(tt.serviceID, tt.envVars)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.Len(t, httpCtx.Requests, 1)
			if tt.validate != nil {
				tt.validate(t, result, httpCtx.Requests[0])
			}
		})
	}
}
