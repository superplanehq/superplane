package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const setupTestOrgID = "org-1"

func TestSetupInitCreatesIntegrationAndRendersFirstStep(t *testing.T) {
	var createBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/me":
			writeSetupMeResponse(w)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/organizations/org-1/integrations":
			payload, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(payload, &createBody))
			writeJSON(w, `{
				"integration": {
					"metadata": {"id": "int-1", "name": "rtx", "integrationName": "semaphore"},
					"status": {
						"state": "pending",
						"setupState": {
							"currentStep": {
								"type": "INPUTS",
								"name": "selectOrganization",
								"label": "What is your Semaphore Organization URL?",
								"instructions": "Enter your organization URL.",
								"inputs": [
									{"name": "organizationUrl", "label": "Semaphore Organization URL", "type": "string", "required": true}
								]
							},
							"previousSteps": []
						}
					}
				}
			}`)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newSetupTestContext(t, server, "text", nil)

	name := "rtx"
	integration := "semaphore"
	interactive := false
	cmd := &setupInitCommand{
		setupTarget: setupTarget{name: &name, integration: &integration},
		interactive: &interactive,
	}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	require.Equal(t, "rtx", createBody["name"])
	require.Equal(t, "semaphore", createBody["integrationName"])
	configuration, ok := createBody["configuration"].(map[string]interface{})
	require.True(t, ok)
	require.Empty(t, configuration)

	raw := stdout.String()
	require.Contains(t, raw, "Integration ID: int-1")
	require.Contains(t, raw, "Next Step: selectOrganization")
	require.Contains(t, raw, "Instructions:")
	require.Contains(t, raw, "organizationUrl")
}

func TestSetupStatusReturnsCurrentStep(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/me":
			writeSetupMeResponse(w)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/organizations/org-1/integrations":
			writeJSON(w, `{
				"integrations": [
					{"metadata": {"id": "int-1", "name": "rtx", "integrationName": "semaphore"}, "status": {"state": "pending"}}
				]
			}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/organizations/org-1/integrations/int-1":
			writeJSON(w, `{
				"integration": {
					"metadata": {"id": "int-1", "name": "rtx", "integrationName": "semaphore"},
					"status": {
						"state": "pending",
						"setupState": {
							"currentStep": {
								"type": "INPUTS",
								"name": "enterAPIToken",
								"inputs": [{"name": "apiToken", "type": "string", "required": true, "sensitive": true}]
							},
							"previousSteps": [{"type":"INPUTS","name":"selectOrganization"}]
						}
					}
				}
			}`)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newSetupTestContext(t, server, "text", nil)

	name := "rtx"
	integration := "semaphore"
	cmd := &setupStatusCommand{setupTarget: setupTarget{name: &name, integration: &integration}}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	raw := stdout.String()
	require.Contains(t, raw, "State: pending")
	require.Contains(t, raw, "Next Step: enterAPIToken")
}

func TestSetupNextParsesJSONInputsAndSubmitsStep(t *testing.T) {
	var nextBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/me":
			writeSetupMeResponse(w)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/organizations/org-1/integrations":
			writeJSON(w, `{
				"integrations": [
					{"metadata": {"id": "int-1", "name": "rtx", "integrationName": "semaphore"}, "status": {"state": "pending"}}
				]
			}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/organizations/org-1/integrations/int-1":
			writeJSON(w, `{
				"integration": {
					"metadata": {"id": "int-1", "name": "rtx", "integrationName": "semaphore"},
					"status": {
						"state": "pending",
						"setupState": {
							"currentStep": {
								"type": "INPUTS",
								"name": "enterAPIToken",
								"inputs": [{"name": "apiToken", "type": "string", "required": true, "sensitive": true}]
							},
							"previousSteps": [{"type":"INPUTS","name":"selectOrganization"}]
						}
					}
				}
			}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/organizations/org-1/integrations/int-1/next":
			payload, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(payload, &nextBody))
			writeJSON(w, `{
				"integration": {
					"metadata": {"id": "int-1", "name": "rtx", "integrationName": "semaphore"},
					"status": {"state": "ready"}
				}
			}`)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newSetupTestContext(t, server, "text", nil)

	name := "rtx"
	integration := "semaphore"
	stepInputs := `{"apiToken":"token-123","enabled":true}`
	cmd := &setupNextCommand{
		setupTarget: setupTarget{name: &name, integration: &integration},
		stepInputs:  &stepInputs,
	}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	inputs, ok := nextBody["inputs"].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "token-123", inputs["apiToken"])
	require.Equal(t, true, inputs["enabled"])

	raw := stdout.String()
	require.Contains(t, raw, "State: ready")
	require.Contains(t, raw, "Next Step: none")
}

func TestSetupPreviousRevertsToPreviousStep(t *testing.T) {
	var previousBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/me":
			writeSetupMeResponse(w)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/organizations/org-1/integrations":
			writeJSON(w, `{
				"integrations": [
					{"metadata": {"id": "int-1", "name": "rtx", "integrationName": "semaphore"}, "status": {"state": "pending"}}
				]
			}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/organizations/org-1/integrations/int-1":
			writeJSON(w, `{
				"integration": {
					"metadata": {"id": "int-1", "name": "rtx", "integrationName": "semaphore"},
					"status": {
						"state": "pending",
						"setupState": {
							"currentStep": {"type":"INPUTS","name":"enterAPIToken"},
							"previousSteps": [{"type":"INPUTS","name":"selectOrganization"}]
						}
					}
				}
			}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/organizations/org-1/integrations/int-1/previous":
			payload, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(payload, &previousBody))
			writeJSON(w, `{
				"integration": {
					"metadata": {"id": "int-1", "name": "rtx", "integrationName": "semaphore"},
					"status": {
						"state": "pending",
						"setupState": {
							"currentStep": {"type":"INPUTS","name":"selectOrganization"},
							"previousSteps": []
						}
					}
				}
			}`)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newSetupTestContext(t, server, "text", nil)

	name := "rtx"
	integration := "semaphore"
	cmd := &setupPreviousCommand{
		setupTarget: setupTarget{name: &name, integration: &integration},
	}

	err := cmd.Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{}, previousBody)

	raw := stdout.String()
	require.Contains(t, raw, "State: pending")
	require.Contains(t, raw, "Next Step: selectOrganization")
}

func TestSetupInitInteractiveCompletesFlow(t *testing.T) {
	nextCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/me":
			writeSetupMeResponse(w)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/organizations/org-1/integrations":
			writeJSON(w, `{
				"integration": {
					"metadata": {"id": "int-1", "name": "rtx", "integrationName": "semaphore"},
					"status": {
						"state": "pending",
						"setupState": {
							"currentStep": {
								"type": "INPUTS",
								"label": "What is your Semaphore Organization URL?",
								"instructions": "Provide your Semaphore organization URL.",
								"name": "selectOrganization",
								"inputs": [
									{"name": "organizationUrl", "label": "Semaphore Organization URL", "type": "string", "required": true}
								]
							},
							"previousSteps": []
						}
					}
				}
			}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/organizations/org-1/integrations/int-1/next":
			nextCalls++
			payload := map[string]interface{}{}
			rawBody, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(rawBody, &payload))

			switch nextCalls {
			case 1:
				inputs := payload["inputs"].(map[string]interface{})
				require.Equal(t, "https://acme.semaphoreci.com", inputs["organizationUrl"])
				writeJSON(w, `{
					"integration": {
						"metadata": {"id": "int-1", "name": "rtx", "integrationName": "semaphore"},
						"status": {
							"state": "pending",
							"setupState": {
								"currentStep": {
									"type": "INPUTS",
									"label": "Enter Semaphore API token",
									"instructions": "Provide your API token.",
									"name": "enterAPIToken",
									"inputs": [
										{"name": "apiToken", "label": "API Token", "type": "string", "required": true, "sensitive": true}
									]
								},
								"previousSteps": [{"type":"INPUTS","name":"selectOrganization"}]
							}
						}
					}
				}`)
			case 2:
				inputs := payload["inputs"].(map[string]interface{})
				require.Equal(t, "secret-token", inputs["apiToken"])
				writeJSON(w, `{
					"integration": {
						"metadata": {"id": "int-1", "name": "rtx", "integrationName": "semaphore"},
						"status": {"state": "ready"}
					}
				}`)
			default:
				t.Fatalf("unexpected next call count %d", nextCalls)
			}
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	input := strings.NewReader("https://acme.semaphoreci.com\nsecret-token\n")
	ctx, stdout := newSetupTestContext(t, server, "text", input)

	name := "rtx"
	integration := "semaphore"
	interactive := true
	cmd := &setupInitCommand{
		setupTarget: setupTarget{name: &name, integration: &integration},
		interactive: &interactive,
	}

	err := cmd.Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, nextCalls)
	require.Contains(t, stdout.String(), "New integration 'rtx' (int-1) created")
	require.Contains(t, stdout.String(), "Next step: What is your Semaphore Organization URL?")
	require.Contains(t, stdout.String(), "Inputs required: Semaphore Organization URL")
	require.Contains(t, stdout.String(), "Next step: Enter Semaphore API token")
	require.Contains(t, stdout.String(), "Inputs required: Semaphore API token")
	require.NotContains(t, stdout.String(), "Instructions:")
	require.NotContains(t, stdout.String(), "Integration ID:")
	require.Contains(t, stdout.String(), "Setup finished.")
}

func TestParseSetupStepInputsKeyValue(t *testing.T) {
	raw := "apiToken=token-123,retry=2,enabled=true"
	parsed, err := parseSetupStepInputs(&raw)
	require.NoError(t, err)

	require.Equal(t, "token-123", parsed["apiToken"])
	require.EqualValues(t, int64(2), parsed["retry"])
	require.Equal(t, true, parsed["enabled"])
}

func writeSetupMeResponse(w http.ResponseWriter) {
	writeJSON(w, `{"user":{"id":"me","email":"me@example.com","organizationId":"`+setupTestOrgID+`"}}`)
}

func writeJSON(w http.ResponseWriter, payload string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(payload))
}

func newSetupTestContext(
	t *testing.T,
	server *httptest.Server,
	outputFormat string,
	stdin io.Reader,
) (core.CommandContext, *bytes.Buffer) {
	t.Helper()

	stdout := bytes.NewBuffer(nil)
	renderer, err := core.NewRenderer(outputFormat, stdout)
	require.NoError(t, err)

	cobraCmd := &cobra.Command{}
	cobraCmd.SetOut(stdout)
	if stdin != nil {
		cobraCmd.SetIn(stdin)
	}

	config := openapi_client.NewConfiguration()
	config.Servers = openapi_client.ServerConfigurations{{URL: server.URL}}

	return core.CommandContext{
		Context:  context.Background(),
		Cmd:      cobraCmd,
		API:      openapi_client.NewAPIClient(config),
		Renderer: renderer,
	}, stdout
}
