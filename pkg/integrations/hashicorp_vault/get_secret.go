package hashicorp_vault

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const SecretPayloadType = "hashicorp_vault.secret"

type getSecret struct{}

type getSecretSpec struct {
	Mount string `mapstructure:"mount"`
	Path  string `mapstructure:"path"`
	Key   string `mapstructure:"key"`
}

type secretPayload struct {
	Mount    string           `json:"mount"`
	Path     string           `json:"path"`
	Data     map[string]any   `json:"data"`
	Value    string           `json:"value,omitempty"`
	Metadata KVSecretMetadata `json:"metadata"`
}

func (c *getSecret) Name() string        { return "hashicorp_vault.getSecret" }
func (c *getSecret) Label() string       { return "Get Secret" }
func (c *getSecret) Description() string { return "Read a secret from HashiCorp Vault KV v2" }
func (c *getSecret) Icon() string        { return "lock" }
func (c *getSecret) Color() string       { return "gray" }

func (c *getSecret) Documentation() string {
	return `The Get Secret component reads a secret from a HashiCorp Vault KV v2 secrets engine.

## Use Cases

- **Inject secrets into workflows**: Read credentials, API keys, or certificates at runtime
- **Secret rotation checks**: Read the latest version of a secret after rotation
- **Conditional workflows**: Branch based on secret values

## Configuration

- **Mount**: The KV v2 secrets engine mount path (default: "secret")
- **Path**: The secret path within the mount, e.g. "myapp/db"
- **Key**: Optional. If set, extracts a single key from the secret data. Available as "value" in the output.

## Output

Returns the full secret data map, optional extracted value, and version metadata.`
}

func (c *getSecret) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *getSecret) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "mount",
			Label:       "Mount",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "secret",
			Description: "KV v2 mount path",
		},
		{
			Name:        "path",
			Label:       "Secret Path",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "myapp/db",
			Description: "Path to the secret, e.g. myapp/db",
		},
		{
			Name:        "key",
			Label:       "Key",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "password",
			Description: "Optional. Extract a specific key from the secret data.",
		},
	}
}

func (c *getSecret) Setup(ctx core.SetupContext) error {
	spec := getSecretSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.Path == "" {
		return fmt.Errorf("path is required")
	}

	return nil
}

func (c *getSecret) Execute(ctx core.ExecutionContext) error {
	spec := getSecretSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.Path == "" {
		return fmt.Errorf("path is required")
	}

	mount := spec.Mount
	if mount == "" {
		mount = "secret"
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	secret, err := client.GetKVSecret(mount, spec.Path)
	if err != nil {
		return err
	}

	payload := secretPayload{
		Mount:    mount,
		Path:     spec.Path,
		Data:     secret.Data,
		Metadata: secret.Metadata,
	}

	if spec.Key != "" {
		val, ok := secret.Data[spec.Key]
		if !ok {
			return fmt.Errorf("key %q not found in secret data", spec.Key)
		}

		strVal, ok := val.(string)
		if !ok {
			return fmt.Errorf("key %q has non-string value", spec.Key)
		}

		payload.Value = strVal
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		SecretPayloadType,
		[]any{payload},
	)
}

func (c *getSecret) Cancel(ctx core.ExecutionContext) error  { return nil }
func (c *getSecret) Cleanup(ctx core.SetupContext) error     { return nil }

func (c *getSecret) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *getSecret) Actions() []core.Action                   { return []core.Action{} }
func (c *getSecret) HandleAction(ctx core.ActionContext) error { return nil }

func (c *getSecret) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
