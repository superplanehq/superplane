package graphql

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func createExecutionContext(config map[string]any) (core.ExecutionContext, *contexts.ExecutionStateContext) {
	if _, ok := config["timeoutSeconds"]; !ok {
		config["timeoutSeconds"] = 1
	}

	stateCtx := &contexts.ExecutionStateContext{}
	metadataCtx := &contexts.MetadataContext{}
	return core.ExecutionContext{
		Logger:         log.NewEntry(log.StandardLogger()),
		Configuration:  config,
		ExecutionState: stateCtx,
		Metadata:       metadataCtx,
		HTTP:           &http.Client{},
		Secrets:        &contexts.SecretsContext{Values: map[string][]byte{}},
	}, stateCtx
}

func responsePayload(t *testing.T, stateCtx *contexts.ExecutionStateContext) map[string]any {
	t.Helper()

	require.Len(t, stateCtx.Payloads, 1)

	payload, ok := stateCtx.Payloads[0].(map[string]any)
	require.True(t, ok)

	data, ok := payload["data"].(map[string]any)
	require.True(t, ok)

	return data
}

func TestGraphQL__Setup__Valid(t *testing.T) {
	g := &GraphQL{}
	err := g.Setup(core.SetupContext{
		Configuration: map[string]any{
			"url":   "https://api.github.com/graphql",
			"query": "{ __typename }",
		},
	})
	assert.NoError(t, err)
}

func TestGraphQL__Setup__MissingURL(t *testing.T) {
	g := &GraphQL{}
	err := g.Setup(core.SetupContext{Configuration: map[string]any{"query": "query { x }"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "url is required")
}

func TestGraphQL__Setup__MissingQuery(t *testing.T) {
	g := &GraphQL{}
	err := g.Setup(core.SetupContext{Configuration: map[string]any{"url": "https://api.example.com/graphql"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query is required")
}

func TestGraphQL__Setup__BearerAuthorizationRequiresToken(t *testing.T) {
	g := &GraphQL{}
	err := g.Setup(core.SetupContext{
		Configuration: map[string]any{
			"url":   "https://api.example.com/graphql",
			"query": "{ __typename }",
			"authorization": map[string]any{
				"type": AuthorizationTypeBearer,
			},
		},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bearer token credential is required")
}

func TestGraphQL__Execute__BuildsBodyAndSuccess(t *testing.T) {
	g := &GraphQL{}

	const q = "query { repo { name } }"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, q, payload["query"])
		assert.NotContains(t, payload, "operationName")

		vars, ok := payload["variables"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "acme", vars["org"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"ok":true}}`))
	}))
	t.Cleanup(server.Close)

	ctx, stateCtx := createExecutionContext(map[string]any{
		"url":            server.URL,
		"query":          q,
		"timeoutSeconds": 1,
		"variables": []map[string]any{
			{"key": "org", "value": "acme"},
		},
	})

	err := g.Execute(ctx)
	require.NoError(t, err)

	assert.True(t, stateCtx.Passed)
	assert.Equal(t, SuccessOutputChannel, stateCtx.Channel)
	assert.Equal(t, "graphql.request.finished", stateCtx.Type)

	data := responsePayload(t, stateCtx)
	assert.Equal(t, http.StatusOK, data["status"])
	body := data["body"].(map[string]any)
	inner, ok := body["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, inner["ok"])
}

func TestGraphQL__Execute__AddsBearerAuthorizationHeader(t *testing.T) {
	g := &GraphQL{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer secret-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"ok":true}}`))
	}))
	t.Cleanup(server.Close)

	ctx, stateCtx := createExecutionContext(map[string]any{
		"url":            server.URL,
		"query":          "{ ok }",
		"timeoutSeconds": 1,
		"authorization": map[string]any{
			"type": AuthorizationTypeBearer,
			"token": map[string]any{
				"secret": "graphql",
				"key":    "token",
			},
		},
	})
	ctx.Secrets = &contexts.SecretsContext{
		Values: map[string][]byte{
			"graphql/token": []byte(" secret-token "),
		},
	}

	err := g.Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, SuccessOutputChannel, stateCtx.Channel)
}

func TestGraphQL__Execute__OmitEmptyVariables(t *testing.T) {
	g := &GraphQL{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		_, hasVars := payload["variables"]
		assert.False(t, hasVars)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)

	ctx, _ := createExecutionContext(map[string]any{
		"url":            server.URL,
		"query":          "{ a }",
		"timeoutSeconds": 1,
	})

	err := g.Execute(ctx)
	require.NoError(t, err)
}

func TestGraphQL__Execute__FailureChannelOnStatus(t *testing.T) {
	g := &GraphQL{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"x"}`))
	}))
	t.Cleanup(server.Close)

	ctx, stateCtx := createExecutionContext(map[string]any{
		"url":            server.URL,
		"query":          "query { a }",
		"timeoutSeconds": 1,
	})

	err := g.Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, FailureOutputChannel, stateCtx.Channel)
	assert.Equal(t, "graphql.request.failed", stateCtx.Type)

	data := responsePayload(t, stateCtx)
	assert.Equal(t, http.StatusBadGateway, data["status"])
}

func TestGraphQL__Execute__FailureChannelOnGraphQLErrors(t *testing.T) {
	g := &GraphQL{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errors":[{"message":"Expected one of SCHEMA, SCALAR"}]}`))
	}))
	t.Cleanup(server.Close)

	ctx, stateCtx := createExecutionContext(map[string]any{
		"url":            server.URL,
		"query":          "query { a }",
		"timeoutSeconds": 1,
	})

	err := g.Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, FailureOutputChannel, stateCtx.Channel)
	assert.Equal(t, "graphql.request.failed", stateCtx.Type)

	data := responsePayload(t, stateCtx)
	assert.Equal(t, http.StatusOK, data["status"])
	body := data["body"].(map[string]any)
	errors, ok := body["errors"].([]any)
	require.True(t, ok)
	require.Len(t, errors, 1)
}

func TestGraphQL__Execute__SuccessChannelOnEmptyGraphQLErrors(t *testing.T) {
	g := &GraphQL{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"ok":true},"errors":[]}`))
	}))
	t.Cleanup(server.Close)

	ctx, stateCtx := createExecutionContext(map[string]any{
		"url":            server.URL,
		"query":          "query { a }",
		"timeoutSeconds": 1,
	})

	err := g.Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, SuccessOutputChannel, stateCtx.Channel)
	assert.Equal(t, "graphql.request.finished", stateCtx.Type)
}

func TestGraphQL__BuildRequestBody__SkipsEmptyVariableKeys(t *testing.T) {
	g := &GraphQL{}
	spec := Spec{
		URL:   "https://example.com/g",
		Query: "query { x }",
		Variables: &[]KeyValue{
			{Key: "", Value: "ignored"},
			{Key: "k", Value: "v"},
		},
	}

	b, err := g.buildRequestBody(spec)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	vars := m["variables"].(map[string]any)
	assert.Equal(t, "v", vars["k"])
}
