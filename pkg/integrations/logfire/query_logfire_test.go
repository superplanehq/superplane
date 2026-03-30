package logfire

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestQueryLogfire_Setup(t *testing.T) {
	t.Parallel()

	component := &QueryLogfire{}

	t.Run("missing sql", func(t *testing.T) {
		t.Parallel()

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
		})
		require.ErrorContains(t, err, "sql is required")
	})

	t.Run("write query is rejected", func(t *testing.T) {
		t.Parallel()

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"sql": "DELETE FROM records",
			},
		})
		require.ErrorContains(t, err, "only read-only queries are allowed")
	})

	t.Run("negative limit is rejected", func(t *testing.T) {
		t.Parallel()

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"sql":   "SELECT * FROM records",
				"limit": -1,
			},
		})
		require.ErrorContains(t, err, "limit must be greater than or equal to 0")
	})

	t.Run("missing project", func(t *testing.T) {
		t.Parallel()

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"sql": "SELECT * FROM records",
			},
		})
		require.ErrorContains(t, err, "project is required")
	})
}

func TestQueryLogfire_Setup_ValidatesProjectSelection(t *testing.T) {
	t.Parallel()

	component := &QueryLogfire{}

	// Mock responses:
	// 1. ListProjects -> returns project proj_123
	// 2. CreateReadToken -> 201 with token
	// 3. ValidateReadToken -> 200 (executeQuery with validate SQL)
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[{"id":"proj_123","project_name":"Project 123"}]`)),
			},
			{
				StatusCode: http.StatusCreated,
				Body:       io.NopCloser(strings.NewReader(`{"id":"tok_1","token":"read_token_new"}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"columns":[{"name":"start_timestamp"}],"rows":[["2026-01-01T00:00:00Z"]]}`)),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiKey": "lf_api_us_123",
		},
		Secrets: map[string]core.IntegrationSecret{},
	}

	metadataCtx := &contexts.MetadataContext{}

	err := component.Setup(core.SetupContext{
		Configuration: map[string]any{
			"sql":         "SELECT message FROM records LIMIT 1",
			"project":     "proj_123",
			"limit":       10,
			"rowOriented": false,
		},
		HTTP:        httpCtx,
		Integration: integrationCtx,
		Metadata:    metadataCtx,
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 3)

	// Verify metadata was set with project info.
	var meta QueryLogfireNodeMetadata
	require.NoError(t, mapstructure.Decode(metadataCtx.Metadata, &meta))
	require.NotNil(t, meta.Project)
	assert.Equal(t, "proj_123", meta.Project.ID)
	assert.Equal(t, "Project 123", meta.Project.Name)

	// Verify per-project secret was stored.
	secret, ok := integrationCtx.Secrets[readTokenSecretNameForProject("proj_123")]
	require.True(t, ok)
	assert.Equal(t, "read_token_new", string(secret.Value))
}

func TestQueryLogfire_Setup_InvalidProject_ReturnsError(t *testing.T) {
	t.Parallel()

	component := &QueryLogfire{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[{"id":"proj_other","project_name":"Other"}]`)),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiKey": "lf_api_us_123",
		},
		Secrets: map[string]core.IntegrationSecret{},
	}

	err := component.Setup(core.SetupContext{
		Configuration: map[string]any{
			"sql":     "SELECT message FROM records LIMIT 1",
			"project": "proj_123",
		},
		HTTP:        httpCtx,
		Integration: integrationCtx,
		Metadata:    &contexts.MetadataContext{},
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "invalid Logfire project selection")
}

func TestQueryLogfire_Setup_ReusesExistingToken(t *testing.T) {
	t.Parallel()

	component := &QueryLogfire{}

	// Mock responses:
	// 1. ListProjects -> returns proj_123
	// 2. ValidateReadToken -> 200 (existing token is valid)
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[{"id":"proj_123","project_name":"Project 123"}]`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"columns":[{"name":"start_timestamp"}],"rows":[["2026-01-01T00:00:00Z"]]}`)),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiKey": "lf_api_us_123",
		},
		Secrets: map[string]core.IntegrationSecret{
			readTokenSecretNameForProject("proj_123"): {
				Name:  readTokenSecretNameForProject("proj_123"),
				Value: []byte("existing_read_token"),
			},
		},
	}

	metadataCtx := &contexts.MetadataContext{}

	err := component.Setup(core.SetupContext{
		Configuration: map[string]any{
			"sql":     "SELECT message FROM records LIMIT 1",
			"project": "proj_123",
		},
		HTTP:        httpCtx,
		Integration: integrationCtx,
		Metadata:    metadataCtx,
	})

	require.NoError(t, err)

	// Only 2 requests: ListProjects + ValidateReadToken.
	// No CreateReadToken call should have been made.
	require.Len(t, httpCtx.Requests, 2)

	// Verify metadata was set.
	var meta QueryLogfireNodeMetadata
	require.NoError(t, mapstructure.Decode(metadataCtx.Metadata, &meta))
	require.NotNil(t, meta.Project)
	assert.Equal(t, "proj_123", meta.Project.ID)
	assert.Equal(t, "Project 123", meta.Project.Name)
}

func TestQueryLogfire_Setup_MigratesLegacyToken(t *testing.T) {
	t.Parallel()

	component := &QueryLogfire{}

	// Mock responses:
	// 1. ListProjects -> returns proj_123
	// 2. ValidateReadToken -> 200 (legacy token is valid)
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[{"id":"proj_123","project_name":"Project 123"}]`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"columns":[{"name":"start_timestamp"}],"rows":[["2026-01-01T00:00:00Z"]]}`)),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiKey": "lf_api_us_123",
		},
		Secrets: map[string]core.IntegrationSecret{
			readTokenSecretName: {
				Name:  readTokenSecretName,
				Value: []byte("legacy_read_token"),
			},
		},
	}

	metadataCtx := &contexts.MetadataContext{}

	err := component.Setup(core.SetupContext{
		Configuration: map[string]any{
			"sql":     "SELECT message FROM records LIMIT 1",
			"project": "proj_123",
		},
		HTTP:        httpCtx,
		Integration: integrationCtx,
		Metadata:    metadataCtx,
	})

	require.NoError(t, err)

	// Only 2 requests: ListProjects + ValidateReadToken.
	require.Len(t, httpCtx.Requests, 2)

	// Verify per-project secret was stored (migration from legacy).
	secret, ok := integrationCtx.Secrets[readTokenSecretNameForProject("proj_123")]
	require.True(t, ok)
	assert.Equal(t, "legacy_read_token", string(secret.Value))

	// Verify metadata was set.
	var meta QueryLogfireNodeMetadata
	require.NoError(t, mapstructure.Decode(metadataCtx.Metadata, &meta))
	require.NotNil(t, meta.Project)
	assert.Equal(t, "proj_123", meta.Project.ID)
	assert.Equal(t, "Project 123", meta.Project.Name)
}

func TestQueryLogfire_Execute_Success(t *testing.T) {
	t.Parallel()

	component := &QueryLogfire{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"columns":[{"name":"message"}],"rows":[["ok"]]}`)),
			},
		},
	}
	executionState := &contexts.ExecutionStateContext{
		KVs: map[string]string{},
	}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiKey": "lf_api_us_123",
		},
		Secrets: map[string]core.IntegrationSecret{
			readTokenSecretNameForProject("proj_123"): {
				Name:  readTokenSecretNameForProject("proj_123"),
				Value: []byte("read_token_123"),
			},
		},
	}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"sql":     "SELECT message FROM records",
			"project": "proj_123",
		},
		HTTP:           httpCtx,
		ExecutionState: executionState,
		Integration:    integrationCtx,
	})
	require.NoError(t, err)
	require.Len(t, executionState.Payloads, 1)

	emitted, ok := executionState.Payloads[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "logfire.query", emitted["type"])

	data, ok := emitted["data"].(*QueryResponse)
	require.True(t, ok)
	assert.NotNil(t, data.Columns)
	assert.NotNil(t, data.Rows)
}

func TestQueryLogfire_Execute_FallsBackToLegacyToken(t *testing.T) {
	t.Parallel()

	component := &QueryLogfire{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"columns":[{"name":"message"}],"rows":[["ok"]]}`)),
			},
		},
	}
	executionState := &contexts.ExecutionStateContext{
		KVs: map[string]string{},
	}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiKey": "lf_api_us_123",
		},
		Secrets: map[string]core.IntegrationSecret{
			readTokenSecretName: {
				Name:  readTokenSecretName,
				Value: []byte("legacy_read_token"),
			},
		},
	}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"sql":     "SELECT message FROM records",
			"project": "proj_123",
		},
		HTTP:           httpCtx,
		ExecutionState: executionState,
		Integration:    integrationCtx,
	})
	require.NoError(t, err)
	require.Len(t, executionState.Payloads, 1)

	emitted, ok := executionState.Payloads[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "logfire.query", emitted["type"])

	data, ok := emitted["data"].(*QueryResponse)
	require.True(t, ok)
	assert.NotNil(t, data.Columns)
	assert.NotNil(t, data.Rows)
}

func TestQueryLogfire_Execute_NoTokenAvailable(t *testing.T) {
	t.Parallel()

	component := &QueryLogfire{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{},
	}
	executionState := &contexts.ExecutionStateContext{
		KVs: map[string]string{},
	}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiKey": "lf_api_us_123",
		},
		Secrets: map[string]core.IntegrationSecret{},
	}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"sql":     "SELECT message FROM records",
			"project": "proj_123",
		},
		HTTP:           httpCtx,
		ExecutionState: executionState,
		Integration:    integrationCtx,
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "no read token available")
}

func TestValidateReadOnlySQL(t *testing.T) {
	t.Parallel()

	require.NoError(t, validateReadOnlySQL("SELECT * FROM records"))
	require.ErrorContains(t, validateReadOnlySQL("update records set message = 'x'"), "only read-only queries are allowed")

	destructiveQueries := []string{
		"INSERT INTO records (message) VALUES ('x')",
		"DELETE FROM records",
		"DROP TABLE records",
		"ALTER TABLE records ADD COLUMN foo TEXT",
		"TRUNCATE TABLE records",
		"CREATE TABLE records_copy AS SELECT * FROM records",
		"GRANT SELECT ON records TO role_reader",
	}

	for _, query := range destructiveQueries {
		require.ErrorContains(t, validateReadOnlySQL(query), "only read-only queries are allowed")
	}
}
