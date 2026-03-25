package logfire

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
			readTokenSecretName: {Name: readTokenSecretName, Value: []byte("read_token_123")},
		},
	}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"sql":       "SELECT message FROM records",
			"projectId": "proj_123",
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

func TestQueryLogfire_Setup_ValidatesProjectSelection(t *testing.T) {
	t.Parallel()

	component := &QueryLogfire{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[{"id":"proj_123","project_name":"Project 123"}]`)),
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
			"sql":         "SELECT message FROM records LIMIT 1",
			"projectId":   "proj_123",
			"limit":       10,
			"rowOriented": false,
		},
		HTTP:        httpCtx,
		Integration: integrationCtx,
	})

	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 1)
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
			"sql":       "SELECT message FROM records LIMIT 1",
			"projectId": "proj_123",
		},
		HTTP:        httpCtx,
		Integration: integrationCtx,
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "invalid Logfire project selection")
}
