package daytona

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	PreviewURLPayloadType             = "daytona.preview.response"
	defaultPreviewURLPort             = 22222
	minPreviewURLPort                 = 1
	maxPreviewURLPort                 = 65535
	defaultPreviewURLExpiresInSeconds = 3600
	minPreviewURLExpiresInSeconds     = 1
	maxPreviewURLExpiresInSeconds     = 86400
)

type GetPreviewURLComponent struct{}

type PreviewURLSpec struct {
	Sandbox          string `json:"sandbox"`
	Port             int    `json:"port,omitempty"`
	Signed           *bool  `json:"signed,omitempty"`
	ExpiresInSeconds int    `json:"expiresInSeconds,omitempty"`
}

type PreviewURLPayload struct {
	Sandbox          string `json:"sandbox"`
	Port             int    `json:"port"`
	Signed           bool   `json:"signed"`
	URL              string `json:"url"`
	Token            string `json:"token"`
	ExpiresInSeconds int    `json:"expiresInSeconds,omitempty"`
}

func (p *GetPreviewURLComponent) Name() string {
	return "daytona.getPreviewUrl"
}

func (p *GetPreviewURLComponent) Label() string {
	return "Get Preview URL"
}

func (p *GetPreviewURLComponent) Description() string {
	return "Generate a preview URL for a sandbox port"
}

func (p *GetPreviewURLComponent) Documentation() string {
	return `The Get Preview URL component generates a Daytona preview URL for a specific sandbox port.

## Use Cases

- **Open sandbox web apps**: Access a web app running in a sandbox from a browser
- **Share previews**: Generate signed URLs that can be opened without custom headers
- **Automation**: Generate preview links for downstream steps and notifications

## Configuration

- **Sandbox**: Sandbox to generate the preview URL for
- **Port**: Sandbox port to preview (default: 3000)
- **Signed URL**: Whether to generate a signed preview URL (default: true)
- **Expires In Seconds**: Signed URL expiration in seconds (default: 60, max: 86400)

## URL Formats

- **Standard URL** (` + "`" + `signed=false` + "`" + `): ` + "`" + `https://{port}-{sandboxId}.{daytonaProxyDomain}` + "`" + `
  Requires ` + "`" + `x-daytona-preview-token` + "`" + ` header with the returned token
- **Signed URL** (` + "`" + `signed=true` + "`" + `): ` + "`" + `https://{port}-{token}.{daytonaProxyDomain}` + "`" + `
  Authentication is embedded in the URL, no custom header required

## Output

Returns preview URL information including:
- **sandbox**: The sandbox ID used in the request
- **port**: The target sandbox port
- **signed**: Whether the generated URL is signed
- **url**: Generated preview URL
- **token**: Preview token (embedded token for signed URLs, header token for standard URLs)
- **expiresInSeconds**: Expiration for signed URLs

## Notes

- The target port must be serving HTTP traffic in the sandbox, otherwise preview access may fail
- Signed URLs can be opened directly in browsers
- Standard URLs require ` + "`" + `x-daytona-preview-token` + "`" + ` header for private sandboxes`
}

func (p *GetPreviewURLComponent) Icon() string {
	return "daytona"
}

func (p *GetPreviewURLComponent) Color() string {
	return "orange"
}

func (p *GetPreviewURLComponent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (p *GetPreviewURLComponent) Configuration() []configuration.Field {
	portMin := minPreviewURLPort
	portMax := maxPreviewURLPort
	expiresMin := minPreviewURLExpiresInSeconds
	expiresMax := maxPreviewURLExpiresInSeconds

	return []configuration.Field{
		{
			Name:        "sandbox",
			Label:       "Sandbox",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Sandbox to generate the preview URL for",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "sandbox",
				},
			},
		},
		{
			Name:        "port",
			Label:       "Port",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     defaultPreviewURLPort,
			Description: "Sandbox port to preview",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: &portMin,
					Max: &portMax,
				},
			},
		},
		{
			Name:        "signed",
			Label:       "Signed URL",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Generate a signed preview URL that can be opened without custom headers",
		},
		{
			Name:        "expiresInSeconds",
			Label:       "Expires In Seconds",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     defaultPreviewURLExpiresInSeconds,
			Description: "Expiration time in seconds for signed URLs",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: &expiresMin,
					Max: &expiresMax,
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "signed",
					Values: []string{"true"},
				},
			},
		},
	}
}

func (p *GetPreviewURLComponent) Setup(ctx core.SetupContext) error {
	spec := PreviewURLSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.Sandbox == "" {
		return fmt.Errorf("sandbox is required")
	}

	port := resolvePreviewURLPort(spec.Port)
	if !isValidPreviewURLPort(port) {
		return fmt.Errorf("port must be between %d and %d", minPreviewURLPort, maxPreviewURLPort)
	}

	if resolveSignedPreview(spec.Signed) {
		expiresInSeconds := resolvePreviewURLExpiresInSeconds(spec.ExpiresInSeconds)
		if !isValidPreviewURLExpiresInSeconds(expiresInSeconds) {
			return fmt.Errorf(
				"expiresInSeconds must be between %d and %d",
				minPreviewURLExpiresInSeconds,
				maxPreviewURLExpiresInSeconds,
			)
		}
	}

	return nil
}

func (p *GetPreviewURLComponent) Execute(ctx core.ExecutionContext) error {
	spec := PreviewURLSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	port := resolvePreviewURLPort(spec.Port)
	signed := resolveSignedPreview(spec.Signed)

	payload := PreviewURLPayload{
		Sandbox: spec.Sandbox,
		Port:    port,
		Signed:  signed,
	}

	if signed {
		expiresInSeconds := resolvePreviewURLExpiresInSeconds(spec.ExpiresInSeconds)
		signedPreviewURL, err := client.GetSignedPreviewURL(spec.Sandbox, port, expiresInSeconds)
		if err != nil {
			return fmt.Errorf("failed to generate signed preview URL: %v", err)
		}

		payload.Sandbox = signedPreviewURL.SandboxID
		if signedPreviewURL.Port > 0 {
			payload.Port = signedPreviewURL.Port
		}
		payload.URL = signedPreviewURL.URL
		payload.Token = signedPreviewURL.Token
		payload.ExpiresInSeconds = expiresInSeconds
	} else {
		previewURL, err := client.GetPreviewURL(spec.Sandbox, port)
		if err != nil {
			return fmt.Errorf("failed to generate preview URL: %v", err)
		}

		payload.Sandbox = previewURL.SandboxID
		payload.URL = previewURL.URL
		payload.Token = previewURL.Token
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		PreviewURLPayloadType,
		[]any{payload},
	)
}

func (p *GetPreviewURLComponent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (p *GetPreviewURLComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (p *GetPreviewURLComponent) Actions() []core.Action {
	return []core.Action{}
}

func (p *GetPreviewURLComponent) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (p *GetPreviewURLComponent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (p *GetPreviewURLComponent) Cleanup(ctx core.SetupContext) error {
	return nil
}

func resolvePreviewURLPort(port int) int {
	if port <= 0 {
		return defaultPreviewURLPort
	}

	return port
}

func isValidPreviewURLPort(port int) bool {
	return port >= minPreviewURLPort && port <= maxPreviewURLPort
}

func resolvePreviewURLExpiresInSeconds(seconds int) int {
	if seconds <= 0 {
		return defaultPreviewURLExpiresInSeconds
	}

	return seconds
}

func isValidPreviewURLExpiresInSeconds(seconds int) bool {
	return seconds >= minPreviewURLExpiresInSeconds && seconds <= maxPreviewURLExpiresInSeconds
}

func resolveSignedPreview(signed *bool) bool {
	if signed == nil {
		return true
	}

	return *signed
}
