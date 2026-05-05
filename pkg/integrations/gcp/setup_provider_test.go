package gcp

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
	"github.com/superplanehq/superplane/test/support/logger"
)

var testIntegrationID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// testSAKeyJSON generates a real service account JSON with a valid RSA key for testing.
func testSAKeyJSON(t *testing.T) string {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pkcs8Bytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)
	keyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8Bytes}))

	m := map[string]string{
		"type":                        "service_account",
		"project_id":                  "my-project",
		"private_key_id":              "key-id",
		"private_key":                 keyPEM,
		"client_email":                "sa@my-project.iam.gserviceaccount.com",
		"client_id":                   "123456789",
		"auth_uri":                    "https://accounts.google.com/o/oauth2/auth",
		"token_uri":                   "https://oauth2.googleapis.com/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
	}
	b, err := json.Marshal(m)
	require.NoError(t, err)
	return string(b)
}

func newSetupCtx(props *contexts.IntegrationPropertyStorage, intCtx *contexts.IntegrationContext, httpCtx *contexts.HTTPContext, caps *contexts.CapabilityContext) core.SetupStepContext {
	return core.SetupStepContext{
		IntegrationID: testIntegrationID,
		Logger:        logger.DiscardLogger(),
		HTTP:          httpCtx,
		Properties:    props,
		Secrets:       intCtx.Secrets(),
		Capabilities:  caps,
	}
}

func crmOKResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"projectId":"my-project"}`)),
	}
}

// Test_GCP_SetupProvider_OnCapabilityUpdate -------------------------------------------------------

func Test_GCP_SetupProvider_OnCapabilityUpdate(t *testing.T) {
	s := &SetupProvider{}

	t.Run("returns error when no requested entry", func(t *testing.T) {
		_, err := s.OnCapabilityUpdate(core.CapabilityUpdateContext{
			Logger:       logger.DiscardLogger(),
			Changes:      map[core.IntegrationCapabilityState][]string{},
			Capabilities: &contexts.CapabilityContext{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no requested capabilities")
	})

	t.Run("enables requested capabilities", func(t *testing.T) {
		caps := &contexts.CapabilityContext{}
		_, err := s.OnCapabilityUpdate(core.CapabilityUpdateContext{
			Logger: logger.DiscardLogger(),
			Changes: map[core.IntegrationCapabilityState][]string{
				core.IntegrationCapabilityStateRequested: {"gcp.createVM", "gcp.cloudbuild.createBuild"},
			},
			Capabilities: caps,
		})
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"gcp.createVM", "gcp.cloudbuild.createBuild"}, caps.EnabledCapabilities)
	})
}

// Test_GCP_SetupProvider_OnStepSubmit -------------------------------------------------------------

func Test_GCP_SetupProvider_OnStepSubmit(t *testing.T) {
	s := &SetupProvider{}

	t.Run("unknown step returns error", func(t *testing.T) {
		ctx := newSetupCtx(contexts.NewIntegrationPropertyStorage(), &contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: "nope"}
		_, err := s.OnStepSubmit(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown step")
	})

	// --- capabilitySelection ---

	t.Run("capabilitySelection moves capabilities to requested/available and returns connection method step", func(t *testing.T) {
		caps := &contexts.CapabilityContext{}
		ctx := newSetupCtx(contexts.NewIntegrationPropertyStorage(), &contexts.IntegrationContext{}, &contexts.HTTPContext{}, caps)
		ctx.Step = core.StepInfo{
			Name:         SetupStepCapabilitySelection,
			Capabilities: []string{"gcp.createVM"},
		}
		next, err := s.OnStepSubmit(ctx)
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, SetupStepSelectConnectionMethod, next.Name)
		assert.Equal(t, core.SetupStepTypeInputs, next.Type)
		assert.Equal(t, []string{"gcp.createVM"}, caps.RequestedCapabilties)
		assert.NotEmpty(t, caps.AvailableCapabilities)
	})

	// --- selectConnectionMethod ---

	t.Run("selectConnectionMethod with invalid input returns error", func(t *testing.T) {
		ctx := newSetupCtx(contexts.NewIntegrationPropertyStorage(), &contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepSelectConnectionMethod, Inputs: "bad"}
		_, err := s.OnStepSubmit(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid input")
	})

	t.Run("selectConnectionMethod with empty method returns error", func(t *testing.T) {
		ctx := newSetupCtx(contexts.NewIntegrationPropertyStorage(), &contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepSelectConnectionMethod, Inputs: map[string]any{PropertyConnectionMethod: ""}}
		_, err := s.OnStepSubmit(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "connection method is required")
	})

	t.Run("selectConnectionMethod SAK routes to enterServiceAccountKey", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		ctx := newSetupCtx(props, &contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepSelectConnectionMethod, Inputs: map[string]any{PropertyConnectionMethod: ConnectionMethodServiceAccountKey}}
		next, err := s.OnStepSubmit(ctx)
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, SetupStepServiceAccountKey, next.Name)
		assert.Equal(t, core.SetupStepTypeInputs, next.Type)
		stored, _ := props.GetString(PropertyConnectionMethod)
		assert.Equal(t, ConnectionMethodServiceAccountKey, stored)
	})

	t.Run("selectConnectionMethod WIF routes to enterWIFProvider", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		ctx := newSetupCtx(props, &contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.WebhooksBaseURL = "https://superplane.example"
		ctx.Step = core.StepInfo{Name: SetupStepSelectConnectionMethod, Inputs: map[string]any{PropertyConnectionMethod: ConnectionMethodWIF}}
		next, err := s.OnStepSubmit(ctx)
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, SetupStepWIFProvider, next.Name)
		assert.Equal(t, core.SetupStepTypeInputs, next.Type)
		assert.NotEmpty(t, next.Instructions)
		assert.Contains(t, next.Instructions, "https://superplane.example/.well-known/openid-configuration")
		stored, _ := props.GetString(PropertyConnectionMethod)
		assert.Equal(t, ConnectionMethodWIF, stored)
	})

	// --- enterServiceAccountKey ---

	t.Run("enterServiceAccountKey with invalid input returns error", func(t *testing.T) {
		ctx := newSetupCtx(contexts.NewIntegrationPropertyStorage(), &contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepServiceAccountKey, Inputs: "bad"}
		_, err := s.OnStepSubmit(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid input")
	})

	t.Run("enterServiceAccountKey with empty key returns error", func(t *testing.T) {
		ctx := newSetupCtx(contexts.NewIntegrationPropertyStorage(), &contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepServiceAccountKey, Inputs: map[string]any{"serviceAccountKey": ""}}
		_, err := s.OnStepSubmit(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service account key is required")
	})

	t.Run("enterServiceAccountKey with invalid JSON returns error", func(t *testing.T) {
		ctx := newSetupCtx(contexts.NewIntegrationPropertyStorage(), &contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepServiceAccountKey, Inputs: map[string]any{"serviceAccountKey": "{not json"}}
		_, err := s.OnStepSubmit(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid service account key")
	})

	t.Run("enterServiceAccountKey CRM call fails returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				// google credential library calls token endpoint first
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"access_token":"tok","expires_in":3600,"token_type":"Bearer"}`))},
				// then our CRM call fails
				{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"denied"}}`))},
			},
		}
		ctx := newSetupCtx(contexts.NewIntegrationPropertyStorage(), &contexts.IntegrationContext{}, httpCtx, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepServiceAccountKey, Inputs: map[string]any{"serviceAccountKey": testSAKeyJSON(t)}}
		_, err := s.OnStepSubmit(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "connection failed")
	})

	t.Run("enterServiceAccountKey success stores secret and properties, returns done", func(t *testing.T) {
		keyJSON := testSAKeyJSON(t)
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"access_token":"tok","expires_in":3600,"token_type":"Bearer"}`))},
				crmOKResponse(),
			},
		}
		props := contexts.NewIntegrationPropertyStorage()
		intCtx := &contexts.IntegrationContext{}
		caps := &contexts.CapabilityContext{RequestedCapabilties: []string{"gcp.createVM"}}
		ctx := newSetupCtx(props, intCtx, httpCtx, caps)
		ctx.Step = core.StepInfo{Name: SetupStepServiceAccountKey, Inputs: map[string]any{"serviceAccountKey": keyJSON}}

		next, err := s.OnStepSubmit(ctx)
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, core.SetupStepTypeDone, next.Type)
		assert.Equal(t, "done", next.Name)
		assert.Contains(t, next.Instructions, "my-project")
		assert.Contains(t, next.Instructions, "sa@my-project.iam.gserviceaccount.com")

		projectID, _ := props.GetString(PropertyProjectID)
		assert.Equal(t, "my-project", projectID)
		clientEmail, _ := props.GetString(PropertyClientEmail)
		assert.Equal(t, "sa@my-project.iam.gserviceaccount.com", clientEmail)

		key, err := intCtx.Secrets().Get("serviceAccountKey")
		require.NoError(t, err)
		assert.Equal(t, keyJSON, key)

		assert.Equal(t, []string{"gcp.createVM"}, caps.EnabledCapabilities)
	})

	// --- enterWIFProvider ---

	t.Run("enterWIFProvider with invalid input returns error", func(t *testing.T) {
		ctx := newSetupCtx(contexts.NewIntegrationPropertyStorage(), &contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepWIFProvider, Inputs: 123}
		_, err := s.OnStepSubmit(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid input")
	})

	t.Run("enterWIFProvider with empty provider returns error", func(t *testing.T) {
		ctx := newSetupCtx(contexts.NewIntegrationPropertyStorage(), &contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepWIFProvider, Inputs: map[string]any{PropertyWIFProvider: "", PropertyProjectID: "my-project"}}
		_, err := s.OnStepSubmit(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pool provider resource name is required")
	})

	t.Run("enterWIFProvider with empty project ID returns error", func(t *testing.T) {
		ctx := newSetupCtx(contexts.NewIntegrationPropertyStorage(), &contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepWIFProvider, Inputs: map[string]any{
			PropertyWIFProvider: "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/pool/providers/sp",
			PropertyProjectID:   "",
		}}
		_, err := s.OnStepSubmit(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project ID is required")
	})

	t.Run("enterWIFProvider success stores properties and returns enterWIFServiceAccount", func(t *testing.T) {
		provider := "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/my-pool/providers/superplane"
		props := contexts.NewIntegrationPropertyStorage()
		ctx := newSetupCtx(props, &contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepWIFProvider, Inputs: map[string]any{
			PropertyWIFProvider: provider,
			PropertyProjectID:   "my-project",
		}}

		next, err := s.OnStepSubmit(ctx)
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, SetupStepWIFServiceAccount, next.Name)
		assert.Equal(t, core.SetupStepTypeInputs, next.Type)
		assert.Contains(t, next.Instructions, testIntegrationID.String())
		assert.Contains(t, next.Instructions, "app-installation:"+testIntegrationID.String())

		storedProvider, _ := props.GetString(PropertyWIFProvider)
		assert.Equal(t, provider, storedProvider)
		storedProject, _ := props.GetString(PropertyProjectID)
		assert.Equal(t, "my-project", storedProject)
	})

	// --- enterWIFServiceAccount ---

	t.Run("enterWIFServiceAccount with invalid input returns error", func(t *testing.T) {
		ctx := newSetupCtx(contexts.NewIntegrationPropertyStorage(), &contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepWIFServiceAccount, Inputs: "bad"}
		_, err := s.OnStepSubmit(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid input")
	})

	t.Run("enterWIFServiceAccount with empty email returns error", func(t *testing.T) {
		ctx := newSetupCtx(contexts.NewIntegrationPropertyStorage(), &contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepWIFServiceAccount, Inputs: map[string]any{PropertyServiceAccountEmail: ""}}
		_, err := s.OnStepSubmit(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service account email is required")
	})

	t.Run("enterWIFServiceAccount success stores email, enables capabilities, returns done", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{Name: PropertyProjectID, Value: "my-project"}))
		caps := &contexts.CapabilityContext{RequestedCapabilties: []string{"gcp.cloudbuild.onBuildComplete"}}
		ctx := newSetupCtx(props, &contexts.IntegrationContext{}, &contexts.HTTPContext{}, caps)
		ctx.Step = core.StepInfo{Name: SetupStepWIFServiceAccount, Inputs: map[string]any{PropertyServiceAccountEmail: "sp@my-project.iam.gserviceaccount.com"}}

		next, err := s.OnStepSubmit(ctx)
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, core.SetupStepTypeDone, next.Type)
		assert.Equal(t, "done", next.Name)
		assert.Contains(t, next.Instructions, "my-project")
		assert.Contains(t, next.Instructions, "sp@my-project.iam.gserviceaccount.com")
		assert.Contains(t, next.Instructions, testIntegrationID.String())

		storedEmail, _ := props.GetString(PropertyServiceAccountEmail)
		assert.Equal(t, "sp@my-project.iam.gserviceaccount.com", storedEmail)
		assert.Equal(t, []string{"gcp.cloudbuild.onBuildComplete"}, caps.EnabledCapabilities)
	})
}

// Test_GCP_SetupProvider_OnStepRevert -------------------------------------------------------------

func Test_GCP_SetupProvider_OnStepRevert(t *testing.T) {
	s := &SetupProvider{}

	t.Run("unknown step returns error", func(t *testing.T) {
		ctx := newSetupCtx(contexts.NewIntegrationPropertyStorage(), &contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: "nope"}
		err := s.OnStepRevert(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown step")
	})

	t.Run("capabilitySelection clears capabilities", func(t *testing.T) {
		caps := &contexts.CapabilityContext{
			RequestedCapabilties:  []string{"gcp.createVM"},
			AvailableCapabilities: []string{"gcp.cloudbuild.createBuild"},
		}
		ctx := newSetupCtx(contexts.NewIntegrationPropertyStorage(), &contexts.IntegrationContext{}, &contexts.HTTPContext{}, caps)
		ctx.Step = core.StepInfo{Name: SetupStepCapabilitySelection}
		require.NoError(t, s.OnStepRevert(ctx))
		assert.Empty(t, caps.RequestedCapabilties)
		assert.Empty(t, caps.AvailableCapabilities)
	})

	t.Run("selectConnectionMethod deletes connectionMethod property", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{Name: PropertyConnectionMethod, Value: ConnectionMethodServiceAccountKey}))
		ctx := newSetupCtx(props, &contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepSelectConnectionMethod}
		require.NoError(t, s.OnStepRevert(ctx))
		_, err := props.GetString(PropertyConnectionMethod)
		require.Error(t, err)
	})

	t.Run("enterServiceAccountKey deletes key secret and properties", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{Name: PropertyProjectID, Value: "my-project"}))
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{Name: PropertyClientEmail, Value: "sa@my-project.iam.gserviceaccount.com"}))
		intCtx := &contexts.IntegrationContext{}
		require.NoError(t, intCtx.SetSecret("serviceAccountKey", []byte("key-json")))

		ctx := newSetupCtx(props, intCtx, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepServiceAccountKey}
		require.NoError(t, s.OnStepRevert(ctx))

		_, err := props.GetString(PropertyProjectID)
		require.Error(t, err)
		_, err = props.GetString(PropertyClientEmail)
		require.Error(t, err)
		_, err = intCtx.Secrets().Get("serviceAccountKey")
		require.Error(t, err)
	})

	t.Run("enterWIFProvider deletes provider and projectId properties", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{Name: PropertyWIFProvider, Value: "//iam.googleapis.com/projects/123/..."}))
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{Name: PropertyProjectID, Value: "my-project"}))

		ctx := newSetupCtx(props, &contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepWIFProvider}
		require.NoError(t, s.OnStepRevert(ctx))

		_, err := props.GetString(PropertyWIFProvider)
		require.Error(t, err)
		_, err = props.GetString(PropertyProjectID)
		require.Error(t, err)
	})

	t.Run("enterWIFServiceAccount deletes service account email", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{Name: PropertyServiceAccountEmail, Value: "sa@proj.iam.gserviceaccount.com"}))

		ctx := newSetupCtx(props, &contexts.IntegrationContext{}, &contexts.HTTPContext{}, &contexts.CapabilityContext{})
		ctx.Step = core.StepInfo{Name: SetupStepWIFServiceAccount}
		require.NoError(t, s.OnStepRevert(ctx))

		_, err := props.GetString(PropertyServiceAccountEmail)
		require.Error(t, err)
	})
}

// Test_GCP_SetupProvider_OnSecretUpdate -----------------------------------------------------------

func Test_GCP_SetupProvider_OnSecretUpdate(t *testing.T) {
	s := &SetupProvider{}

	t.Run("unknown secret returns error", func(t *testing.T) {
		_, err := s.OnSecretUpdate(core.SecretUpdateContext{
			Logger:     logger.DiscardLogger(),
			SecretName: "other",
			Value:      "x",
			HTTP:       &contexts.HTTPContext{},
			Properties: contexts.NewIntegrationPropertyStorage(),
			Secrets:    (&contexts.IntegrationContext{}).Secrets(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown secret")
	})

	t.Run("empty key returns error", func(t *testing.T) {
		_, err := s.OnSecretUpdate(core.SecretUpdateContext{
			Logger:     logger.DiscardLogger(),
			SecretName: "serviceAccountKey",
			Value:      "   ",
			HTTP:       &contexts.HTTPContext{},
			Properties: contexts.NewIntegrationPropertyStorage(),
			Secrets:    (&contexts.IntegrationContext{}).Secrets(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service account key is required")
	})

	t.Run("key for wrong project returns error", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{Name: PropertyProjectID, Value: "other-project"}))
		_, err := s.OnSecretUpdate(core.SecretUpdateContext{
			Logger:     logger.DiscardLogger(),
			SecretName: "serviceAccountKey",
			Value:      testSAKeyJSON(t),
			HTTP:       &contexts.HTTPContext{},
			Properties: props,
			Secrets:    (&contexts.IntegrationContext{}).Secrets(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "other-project")
	})

	t.Run("CRM call fails returns error", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{Name: PropertyProjectID, Value: "my-project"}))
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"access_token":"tok","expires_in":3600,"token_type":"Bearer"}`))},
				{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"denied"}}`))},
			},
		}
		_, err := s.OnSecretUpdate(core.SecretUpdateContext{
			Logger:     logger.DiscardLogger(),
			SecretName: "serviceAccountKey",
			Value:      testSAKeyJSON(t),
			HTTP:       httpCtx,
			Properties: props,
			Secrets:    (&contexts.IntegrationContext{}).Secrets(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "connection failed")
	})

	t.Run("success updates the stored secret", func(t *testing.T) {
		keyJSON := testSAKeyJSON(t)
		props := contexts.NewIntegrationPropertyStorage()
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{Name: PropertyProjectID, Value: "my-project"}))
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"access_token":"tok","expires_in":3600,"token_type":"Bearer"}`))},
				crmOKResponse(),
			},
		}
		intCtx := &contexts.IntegrationContext{}
		require.NoError(t, intCtx.SetSecret("serviceAccountKey", []byte("old-key")))

		_, err := s.OnSecretUpdate(core.SecretUpdateContext{
			Logger:     logger.DiscardLogger(),
			SecretName: "serviceAccountKey",
			Value:      keyJSON,
			HTTP:       httpCtx,
			Properties: props,
			Secrets:    intCtx.Secrets(),
		})
		require.NoError(t, err)
		v, getErr := intCtx.Secrets().Get("serviceAccountKey")
		require.NoError(t, getErr)
		assert.Equal(t, keyJSON, v)
	})
}

// Test_GCP_SetupProvider_wifPrincipal -------------------------------------------------------------

func Test_GCP_SetupProvider_wifPrincipal(t *testing.T) {
	provider := "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/my-pool/providers/superplane"
	id := "abc-def"
	principal := wifPrincipal(provider, id)
	assert.Equal(t,
		"principal://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/my-pool/subject/app-installation:abc-def",
		principal,
	)
}

// Test_GCP_SetupProvider_CapabilityGroups ---------------------------------------------------------

func Test_GCP_SetupProvider_CapabilityGroups(t *testing.T) {
	s := &SetupProvider{}
	groups := s.CapabilityGroups()
	require.NotEmpty(t, groups)

	var allNames []string
	for _, g := range groups {
		for _, c := range g.Capabilities {
			allNames = append(allNames, c.Name)
			assert.NotEmpty(t, c.Label, "capability %s has no label", c.Name)
		}
	}

	assert.Contains(t, allNames, "gcp.createVM")
	assert.Contains(t, allNames, "gcp.compute.onVMInstance")
	assert.Contains(t, allNames, "gcp.cloudbuild.createBuild")
	assert.Contains(t, allNames, "gcp.cloudbuild.onBuildComplete")
	assert.Contains(t, allNames, "gcp.artifactregistry.getArtifact")
	assert.Contains(t, allNames, "gcp.artifactregistry.onArtifactPush")
	assert.Contains(t, allNames, "gcp.pubsub.publishMessage")
	assert.Contains(t, allNames, "gcp.pubsub.onMessage")
	assert.Contains(t, allNames, "gcp.cloudfunctions.invokeFunction")
	assert.Contains(t, allNames, "gcp.clouddns.createRecord")
}

