package notion

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support/contexts"
)

const parentID = "18f7101a-6f38-4c52-bf59-71a68469daf1"

func createConfiguration(parentType string) map[string]any {
	return map[string]any{"parentType": parentType, "parentId": parentID, "title": "Incident runbook"}
}

func TestCreatePageExecute(t *testing.T) {
	for _, parentType := range []string{"page", "data_source"} {
		t.Run(parentType, func(t *testing.T) {
			config := createConfiguration(parentType)
			config["children"] = `[{"object":"block","type":"paragraph","paragraph":{"rich_text":[{"type":"text","text":{"content":"Restart the service."}}]}}]`
			h := &contexts.HTTPContext{}
			if parentType == "data_source" {
				config["properties"] = `{"Status":{"select":{"name":"Open"}}}`
				h.Responses = append(h.Responses, response(200, `{"object":"data_source","id":"`+parentID+`","properties":{"Runbook name":{"id":"title","type":"title","title":{}},"Status":{"id":"status","type":"select","select":{}}}}`))
			}
			h.Responses = append(h.Responses, response(200, `{"object":"page","id":"created-page","url":"https://www.notion.so/created-page","properties":{"title":{"type":"title","title":[]}}}`))
			state := &contexts.ExecutionStateContext{}
			err := (&CreatePage{}).Execute(core.ExecutionContext{HTTP: h, Integration: integration(), ExecutionState: state, Configuration: config})
			require.NoError(t, err)
			if parentType == "data_source" {
				require.Len(t, h.Requests, 2)
				assert.Equal(t, http.MethodGet, h.Requests[0].Method)
				assert.Equal(t, "/v1/data_sources/"+parentID, h.Requests[0].URL.Path)
			} else {
				require.Len(t, h.Requests, 1)
			}
			request := h.Requests[len(h.Requests)-1]
			assert.Equal(t, http.MethodPost, request.Method)
			assert.Equal(t, "https://api.notion.com/v1/pages", request.URL.String())
			var body map[string]any
			require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
			assert.Equal(t, parentID, body["parent"].(map[string]any)[parentType+"_id"])
			properties := body["properties"].(map[string]any)
			title := properties["title"].(map[string]any)["title"].([]any)[0].(map[string]any)
			assert.Equal(t, "Incident runbook", title["text"].(map[string]any)["content"])
			if parentType == "data_source" {
				assert.Contains(t, properties, "Status")
			}
			assert.Len(t, body["children"], 1)
			assert.Equal(t, "default", state.Channel)
			assert.Equal(t, "notion.page", state.Type)
			require.Len(t, state.Payloads, 1)
			assert.Equal(t, "created-page", state.Payloads[0].(map[string]any)["data"].(map[string]any)["id"])
		})
	}
}

func TestCreatePageValidation(t *testing.T) {
	for _, test := range []struct {
		name  string
		field string
		value any
	}{
		{"parent type", "parentType", "database"},
		{"parent ID", "parentId", "https://notion.so/page"},
		{"empty title", "title", " "},
		{"long title", "title", strings.Repeat("x", 2001)},
		{"properties array", "properties", "[]"},
		{"null properties", "properties", "null"},
		{"children object", "children", "{}"},
		{"null children", "children", "null"},
		{"invalid block", "children", "[null]"},
		{"invalid JSON", "children", "["},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := createConfiguration("data_source")
			config[test.field] = test.value
			require.Error(t, (&CreatePage{}).Setup(core.SetupContext{Configuration: config}))
			h := &contexts.HTTPContext{}
			err := (&CreatePage{}).Execute(core.ExecutionContext{Configuration: config, HTTP: h, Integration: integration()})
			require.Error(t, err)
			assert.Empty(t, h.Requests)
		})
	}

	t.Run("setup preserves expressions", func(t *testing.T) {
		require.NoError(t, (&CreatePage{}).Setup(core.SetupContext{Configuration: map[string]any{
			"parentType": "data_source", "parentId": "{{ $['Input'].data.parent }}", "title": "{{ $['Input'].data.title }}",
			"properties": "{{ $['Input'].data.properties }}", "children": "{{ $['Input'].data.children }}",
		}}))
	})
}

func TestCreatePageRejectsTitleConflict(t *testing.T) {
	for _, property := range []string{"Runbook name", "title"} {
		config := createConfiguration("data_source")
		config["properties"] = `{"` + property + `":{"title":[]}}`
		h := &contexts.HTTPContext{Responses: []*http.Response{response(200, `{"object":"data_source","properties":{"Runbook name":{"id":"title","type":"title"}}}`)}}
		err := (&CreatePage{}).Execute(core.ExecutionContext{Configuration: config, HTTP: h, Integration: integration()})
		require.ErrorContains(t, err, "Title field")
		require.Len(t, h.Requests, 1)
		assert.Equal(t, http.MethodGet, h.Requests[0].Method)
	}
}

func TestCreatePageRejectsCombinedPayloadBeforeWrite(t *testing.T) {
	richText := make([]map[string]any, 100)
	for i := range richText {
		richText[i] = map[string]any{"type": "text", "text": map[string]any{"content": strings.Repeat("x", 2000)}}
	}
	properties := map[string]any{"Description": map[string]any{"rich_text": richText}}
	children := make([]map[string]any, 100)
	for i := range children {
		children[i] = map[string]any{"object": "block", "type": "paragraph", "paragraph": map[string]any{"rich_text": richText[:2]}}
	}
	for _, value := range []any{properties, children} {
		encoded, err := json.Marshal(value)
		require.NoError(t, err)
		require.Less(t, len(encoded), maxRequestBytes)
	}
	configuration := createConfiguration("data_source")
	configuration["properties"] = properties
	configuration["children"] = children
	require.NoError(t, (&CreatePage{}).Setup(core.SetupContext{Configuration: configuration}))
	h := &contexts.HTTPContext{Responses: []*http.Response{
		response(200, `{"object":"data_source","properties":{"Name":{"id":"title","type":"title"}}}`),
	}}
	state := &contexts.ExecutionStateContext{}
	err := (&CreatePage{}).Execute(core.ExecutionContext{Configuration: configuration, HTTP: h, Integration: integration(), ExecutionState: state})
	require.ErrorContains(t, err, "500 KB limit")
	assert.NotContains(t, err.Error(), "page may exist")
	assert.NotContains(t, err.Error(), "Check Notion before")
	require.Len(t, h.Requests, 1)
	assert.Equal(t, http.MethodGet, h.Requests[0].Method)
	assert.Empty(t, state.Payloads)
}

func TestCreatePageDoesNotRetryUncertainWrites(t *testing.T) {
	t.Run("transport error", func(t *testing.T) {
		h := &failingHTTP{}
		state := &contexts.ExecutionStateContext{}
		err := (&CreatePage{}).Execute(core.ExecutionContext{Configuration: createConfiguration("page"), HTTP: h, Integration: integration(), ExecutionState: state})
		require.ErrorContains(t, err, "Check Notion before")
		assert.Equal(t, 1, h.requests)
		assert.Empty(t, state.Payloads)
	})
	for _, status := range []int{500, 502, 529} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			h := &contexts.HTTPContext{Responses: []*http.Response{response(status, `{"code":"internal_server_error"}`)}}
			state := &contexts.ExecutionStateContext{}
			err := (&CreatePage{}).Execute(core.ExecutionContext{Configuration: createConfiguration("page"), HTTP: h, Integration: integration(), ExecutionState: state})
			require.Error(t, err)
			assert.Len(t, h.Requests, 1)
			assert.Empty(t, state.Payloads)
		})
	}
}

func TestCreatePageResolvesStructuredExpressions(t *testing.T) {
	component := &CreatePage{}
	configuration := createConfiguration("data_source")
	configuration["properties"] = "{{ previous().data.properties }}"
	configuration["children"] = "{{ previous().data.children }}"
	require.NoError(t, component.Setup(core.SetupContext{Configuration: configuration}))

	builder := workercontexts.NewNodeConfigurationBuilder(nil, uuid.Nil).
		WithInput(map[string]any{"Input": map[string]any{
			"type": "input.data", "timestamp": time.Now().Format(time.RFC3339),
			"data": map[string]any{
				"properties": map[string]any{"Count": map[string]any{"number": 7}, "Ready": map[string]any{"checkbox": true}},
				"children": []any{map[string]any{
					"object": "block", "type": "paragraph",
					"paragraph": map[string]any{"rich_text": []any{map[string]any{"type": "text", "text": map[string]any{"content": "Keep {{ template }} literal"}}}},
				}},
			},
		}}).
		WithConfigurationFields(component.Configuration())
	resolved, err := builder.Build(configuration)
	require.NoError(t, err)

	h := &contexts.HTTPContext{Responses: []*http.Response{
		response(200, `{"object":"data_source","properties":{"Runbook":{"id":"title","type":"title"}}}`),
		response(200, `{"object":"page","id":"created-page"}`),
	}}
	state := &contexts.ExecutionStateContext{}
	require.NoError(t, component.Execute(core.ExecutionContext{HTTP: h, Integration: integration(), ExecutionState: state, Configuration: resolved}))
	require.Len(t, h.Requests, 2)
	var body map[string]any
	require.NoError(t, json.NewDecoder(h.Requests[1].Body).Decode(&body))
	properties := body["properties"].(map[string]any)
	assert.Equal(t, float64(7), properties["Count"].(map[string]any)["number"])
	assert.Equal(t, true, properties["Ready"].(map[string]any)["checkbox"])
	paragraph := body["children"].([]any)[0].(map[string]any)["paragraph"].(map[string]any)
	text := paragraph["rich_text"].([]any)[0].(map[string]any)["text"].(map[string]any)
	assert.Equal(t, "Keep {{ template }} literal", text["content"])
	require.Len(t, state.Payloads, 1)
}
