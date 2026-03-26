package azure

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

func TestInvokeFunctionComponent_Metadata(t *testing.T) {
	component := &InvokeFunctionComponent{}

	assert.Equal(t, "azure.invokeFunction", component.Name())
	assert.Equal(t, "Invoke Function", component.Label())
	assert.Equal(t, "azure", component.Icon())
	assert.Equal(t, "blue", component.Color())
	assert.NotEmpty(t, component.Description())
	assert.NotEmpty(t, component.Documentation())
}

func TestInvokeFunctionComponent_Configuration(t *testing.T) {
	component := &InvokeFunctionComponent{}
	fields := component.Configuration()

	require.Len(t, fields, 5)

	assert.Equal(t, "resourceGroup", fields[0].Name)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, fields[0].Type)
	assert.True(t, fields[0].Required)

	assert.Equal(t, "functionApp", fields[1].Name)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, fields[1].Type)
	assert.True(t, fields[1].Required)

	assert.Equal(t, "functionName", fields[2].Name)
	assert.Equal(t, configuration.FieldTypeIntegrationResource, fields[2].Type)
	assert.True(t, fields[2].Required)

	assert.Equal(t, "httpMethod", fields[3].Name)
	assert.Equal(t, configuration.FieldTypeSelect, fields[3].Type)
	assert.True(t, fields[3].Required)

	assert.Equal(t, "payload", fields[4].Name)
	assert.Equal(t, configuration.FieldTypeExpression, fields[4].Type)
	assert.False(t, fields[4].Required)
}

func TestInvokeFunctionComponent_ExampleOutput(t *testing.T) {
	component := &InvokeFunctionComponent{}
	example := component.ExampleOutput()

	require.NotNil(t, example)
	assert.Contains(t, example, "statusCode")
	assert.Contains(t, example, "body")
	assert.Contains(t, example, "functionApp")
	assert.Contains(t, example, "functionName")
}

func TestInvokeFunctionComponent_OutputChannels(t *testing.T) {
	component := &InvokeFunctionComponent{}
	channels := component.OutputChannels(nil)

	require.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}

func TestInvokeFunctionComponent_Actions(t *testing.T) {
	component := &InvokeFunctionComponent{}
	assert.Empty(t, component.Actions())
}

func TestInvokeFunctionComponent_HandleAction(t *testing.T) {
	component := &InvokeFunctionComponent{}
	err := component.HandleAction(core.ActionContext{
		Name:   "test",
		Logger: logrus.NewEntry(logrus.New()),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no actions defined")
}

func TestInvokeFunctionComponent_Setup_Valid(t *testing.T) {
	component := &InvokeFunctionComponent{}
	ctx := newSetupContext(map[string]any{
		"resourceGroup": "my-rg",
		"functionApp":   "my-function-app",
		"functionName":  "my-function",
		"httpMethod":    "POST",
	})
	assert.NoError(t, component.Setup(ctx))
}

func TestInvokeFunctionComponent_Setup_MissingResourceGroup(t *testing.T) {
	component := &InvokeFunctionComponent{}
	ctx := newSetupContext(map[string]any{
		"functionApp":  "my-function-app",
		"functionName": "my-function",
	})
	err := component.Setup(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resource group is required")
}

func TestInvokeFunctionComponent_Setup_MissingFunctionApp(t *testing.T) {
	component := &InvokeFunctionComponent{}
	ctx := newSetupContext(map[string]any{
		"resourceGroup": "my-rg",
		"functionName":  "my-function",
	})
	err := component.Setup(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "function app is required")
}

func TestInvokeFunctionComponent_Setup_MissingFunctionName(t *testing.T) {
	component := &InvokeFunctionComponent{}
	ctx := newSetupContext(map[string]any{
		"resourceGroup": "my-rg",
		"functionApp":   "my-function-app",
	})
	err := component.Setup(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "function name is required")
}

func TestInvokeFunction_Success(t *testing.T) {
	responseBody := map[string]any{"result": "ok", "value": 42}

	provider, server := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/my-function", r.URL.Path)
		assert.Equal(t, "Bearer mock-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(responseBody)
	})
	defer server.Close()

	functionURL := server.URL + "/api/my-function"
	output, err := invokeFunction(context.Background(), provider.getClient(), functionURL, http.MethodPost, `{"input":"test"}`)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, output["statusCode"])

	body, ok := output["body"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ok", body["result"])
}

func TestInvokeFunction_NonJSONResponse(t *testing.T) {
	provider, server := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("plain text response"))
	})
	defer server.Close()

	functionURL := server.URL + "/api/my-function"
	output, err := invokeFunction(context.Background(), provider.getClient(), functionURL, http.MethodGet, "")

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, output["statusCode"])
	assert.Equal(t, "plain text response", output["body"])
}

func TestInvokeFunction_ErrorStatusCode(t *testing.T) {
	provider, server := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"something went wrong"}`))
	})
	defer server.Close()

	functionURL := server.URL + "/api/my-function"
	output, err := invokeFunction(context.Background(), provider.getClient(), functionURL, http.MethodPost, "")

	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, output["statusCode"])
}
