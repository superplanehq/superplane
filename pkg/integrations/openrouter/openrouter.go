package openrouter

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
)

const connectDescription = `Connect your OpenRouter account.

You will be sent to OpenRouter to approve the connection, and an API key will be issued back to SuperPlane automatically. Nothing needs to be copied by hand.`

func init() {
	registry.RegisterIntegration("openrouter", &OpenRouter{})
}

const (
	ResourceTypeModel    = "model"
	ResourceTypeProvider = "provider"
)

type OpenRouter struct{}

type Configuration struct {
	ManagementKey string `json:"managementKey"`
}

func (o *OpenRouter) Name() string {
	return "openrouter"
}

func (o *OpenRouter) Label() string {
	return "OpenRouter"
}

func (o *OpenRouter) Icon() string {
	return "openrouter"
}

func (o *OpenRouter) Description() string {
	return "Run prompts across hundreds of models through OpenRouter"
}

func (o *OpenRouter) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "managementKey",
			Label:       "Provisioning API Key",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "Only needed for Get Credits. OpenRouter's OAuth cannot issue a provisioning key, so this one is entered by hand.",
		},
	}
}

func (o *OpenRouter) Actions() []core.Action {
	return []core.Action{
		&ChatCompletion{},
		&GetCredits{},
	}
}

func (o *OpenRouter) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (o *OpenRouter) Instructions() string {
	return `## Connecting

Connecting sends you to OpenRouter to approve access. OpenRouter issues an API key back to SuperPlane automatically, so there is no key to copy for **Chat Completion**.

- You need an OpenRouter account with credits available.
- The connection uses OAuth with PKCE, so there is no app to register.
- To disconnect, delete the key under [Keys](https://openrouter.ai/settings/keys) and re-connect here.

## Provisioning API Key (optional)

Only required by the **Get Credits** component. OpenRouter's OAuth cannot issue this kind of key, so it is entered by hand.

Create one at [Provisioning Keys](https://openrouter.ai/settings/provisioning-keys) and paste it below.

- Provisioning keys read account credits and manage keys, but cannot call model endpoints.
- Leave it empty if you only use Chat Completion.

## Credits

Chat Completion spends credits from your OpenRouter account. Free model variants (IDs ending in ` + "`:free`" + `) draw from a shared upstream pool and are rate limited independently of your balance.

> **Note:** The provisioning key is shown only once — store it somewhere safe before continuing.`
}

func (o *OpenRouter) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (o *OpenRouter) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	callbackURL := fmt.Sprintf("%s/api/v1/integrations/%s/callback", ctx.BaseURL, ctx.Integration.ID())

	//
	// Without a key, send the user through OpenRouter's OAuth consent screen.
	// The key is issued there rather than pasted in.
	//
	apiKey, _ := findSecret(ctx.Integration, SecretAPIKey)
	if apiKey == "" {
		return o.requestAuthorization(ctx, callbackURL)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.Verify(); err != nil {
		return err
	}

	// The provisioning key is only used by Get Credits, so a failed verification
	// must not block Chat Completion from becoming ready.
	if config.ManagementKey != "" {
		if err := client.VerifyManagement(); err != nil && ctx.Logger != nil {
			ctx.Logger.Warnf("provisioning key verification failed: %v", err)
		}
	}

	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.Ready()
	return nil
}

// requestAuthorization starts the PKCE flow: it mints a verifier, stores it, and
// points the user at OpenRouter's consent screen.
func (o *OpenRouter) requestAuthorization(ctx core.SyncContext, callbackURL string) error {
	verifier, err := newCodeVerifier()
	if err != nil {
		return fmt.Errorf("failed to generate code verifier: %v", err)
	}

	if err := ctx.Integration.SetSecret(SecretCodeVerifier, []byte(verifier)); err != nil {
		return fmt.Errorf("failed to store code verifier: %v", err)
	}

	state, err := crypto.Base64String(32)
	if err != nil {
		return fmt.Errorf("failed to generate state: %v", err)
	}
	ctx.Integration.SetMetadata(Metadata{State: state})

	ctx.Integration.NewBrowserAction(core.BrowserAction{
		Description: connectDescription,
		URL:         authorizeURL(callbackURL, state, verifier),
		Method:      "GET",
	})

	return nil
}

func (o *OpenRouter) HandleRequest(ctx core.HTTPRequestContext) {
	if !strings.HasSuffix(ctx.Request.URL.Path, "/callback") {
		ctx.Response.WriteHeader(http.StatusNotFound)
		return
	}

	settingsURL := fmt.Sprintf("%s/%s/settings/integrations/%s", ctx.BaseURL, ctx.OrganizationID, ctx.Integration.ID())

	code := ctx.Request.URL.Query().Get("code")
	if code == "" {
		ctx.Logger.Error("Callback error: missing code")
		http.Redirect(ctx.Response, ctx.Request, settingsURL, http.StatusSeeOther)
		return
	}

	//
	// This endpoint is unauthenticated, so the state guards against a forged
	// callback planting someone else's key. OpenRouter's authorize endpoint takes
	// no state parameter of its own, so it rides along in callback_url and only
	// comes back if OpenRouter preserves the query string.
	//
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		ctx.Logger.Errorf("Callback error: failed to decode metadata: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	state := ctx.Request.URL.Query().Get("state")
	if state != "" && state != metadata.State {
		ctx.Logger.Error("Callback error: invalid state")
		http.Redirect(ctx.Response, ctx.Request, settingsURL, http.StatusSeeOther)
		return
	}
	if state == "" {
		ctx.Logger.Warn("Callback did not carry the state parameter; it could not be verified")
	}

	verifier, err := findSecret(ctx.Integration, SecretCodeVerifier)
	if err != nil || verifier == "" {
		ctx.Logger.Error("Callback error: no code verifier stored")
		http.Redirect(ctx.Response, ctx.Request, settingsURL, http.StatusSeeOther)
		return
	}

	apiKey, err := NewAuth(ctx.HTTP).ExchangeCode(code, verifier)
	if err != nil {
		ctx.Logger.Errorf("Callback error: %v", err)
		http.Redirect(ctx.Response, ctx.Request, settingsURL, http.StatusSeeOther)
		return
	}

	if err := ctx.Integration.SetSecret(SecretAPIKey, []byte(apiKey)); err != nil {
		ctx.Logger.Errorf("Callback error: failed to store key: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	// The verifier and state are single-use.
	_ = ctx.Integration.SetSecret(SecretCodeVerifier, []byte(""))
	ctx.Integration.SetMetadata(Metadata{})

	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.Ready()

	http.Redirect(ctx.Response, ctx.Request, settingsURL, http.StatusSeeOther)
}

func (o *OpenRouter) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case ResourceTypeModel:
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, err
		}

		models, err := client.ListModels()
		if err != nil {
			return nil, err
		}

		resources := make([]core.IntegrationResource, 0, len(models))
		for _, model := range models {
			if model.ID == "" {
				continue
			}

			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: model.ID,
				ID:   model.ID,
			})
		}
		return resources, nil

	case ResourceTypeProvider:
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return nil, err
		}

		// Narrow the list to the providers actually serving the selected model,
		// since routing to one that does not serve it fails the request.
		if model := ctx.Parameters["model"]; model != "" {
			endpoints, err := client.ListModelEndpoints(model)
			if err != nil {
				return nil, err
			}
			return providerResourcesFromEndpoints(endpoints), nil
		}

		providers, err := client.ListProviders()
		if err != nil {
			return nil, err
		}

		resources := make([]core.IntegrationResource, 0, len(providers))
		for _, provider := range providers {
			if provider.Slug == "" {
				continue
			}

			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: provider.Name,
				ID:   provider.Slug,
			})
		}
		return resources, nil
	}

	return []core.IntegrationResource{}, nil
}

// providerResourcesFromEndpoints reduces a model's endpoints to the distinct
// provider slugs routing accepts. A provider can serve the same model from
// several regions, which carry a region suffix on the tag (e.g.
// "azure/swedencentral") that routing does not take.
func providerResourcesFromEndpoints(endpoints []ModelEndpoint) []core.IntegrationResource {
	resources := make([]core.IntegrationResource, 0, len(endpoints))
	seen := map[string]bool{}

	for _, endpoint := range endpoints {
		slug, _, _ := strings.Cut(endpoint.Tag, "/")
		if slug == "" || seen[slug] {
			continue
		}
		seen[slug] = true

		name := endpoint.ProviderName
		if name == "" {
			name = slug
		}

		resources = append(resources, core.IntegrationResource{
			Type: ResourceTypeProvider,
			Name: name,
			ID:   slug,
		})
	}

	return resources
}

func (o *OpenRouter) Hooks() []core.Hook {
	return []core.Hook{}
}

func (o *OpenRouter) HandleHook(ctx core.IntegrationHookContext) error {
	return nil
}
