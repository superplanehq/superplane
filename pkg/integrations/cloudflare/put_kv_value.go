package cloudflare

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type PutKVValue struct{}

type PutKVValueSpec struct {
	AccountID     string `json:"accountId"`
	Namespace     string `json:"namespace"`
	Key           string `json:"key"`
	Value         string `json:"value"`
	ExpirationTTL *int   `json:"expirationTtl"`
}

func (c *PutKVValue) Name() string {
	return "cloudflare.putKVValue"
}

func (c *PutKVValue) Label() string {
	return "Put KV Value"
}

func (c *PutKVValue) Description() string {
	return "Write a key-value pair to a Cloudflare Workers KV namespace"
}

func (c *PutKVValue) Documentation() string {
	return `The Put KV Value component writes a key-value pair to a Cloudflare Workers KV namespace.

## Use Cases

- **Feature flags**: Toggle features by writing flag values into KV storage
- **Cache invalidation**: Store cache keys or version stamps
- **Dynamic configuration**: Update runtime configuration values from a workflow

## Configuration

- **Namespace**: The ID of the KV namespace to write to
- **Key**: The key name (up to 512 bytes, printable non-whitespace characters)
- **Value**: The value to store (up to 25 MiB)
- **Expiration TTL**: (Optional) Number of seconds until the key expires (minimum 60)

## Output

Emits a confirmation with the account ID, namespace ID, and key that was written.`
}

func (c *PutKVValue) Icon() string {
	return "cloud"
}

func (c *PutKVValue) Color() string {
	return "orange"
}

func (c *PutKVValue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *PutKVValue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "namespace",
			Label:       "Namespace",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The KV namespace to write to",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "namespace",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "accountId",
							ValueFrom: &configuration.ParameterValueFrom{Field: "accountId"},
						},
					},
				},
			},
		},
		{
			Name:        "key",
			Label:       "Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The key name to write (up to 512 bytes, printable non-whitespace characters)",
			Placeholder: "my-key",
		},
		{
			Name:        "value",
			Label:       "Value",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The value to store (up to 25 MiB)",
			Placeholder: "my-value",
		},
		{
			Name:        "expirationTtl",
			Label:       "Expiration TTL (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Number of seconds until the key expires. Must be at least 60. Leave empty for no expiration.",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := 60; return &min }(),
				},
			},
		},
	}
}

func (c *PutKVValue) Setup(ctx core.SetupContext) error {
	spec := PutKVValueSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	accountID := resolveAccountID(spec.AccountID, ctx.Integration)
	if accountID == "" {
		return errors.New("accountId is required")
	}

	if spec.Namespace == "" {
		return errors.New("namespace is required")
	}

	if spec.Key == "" {
		return errors.New("key is required")
	}

	if spec.Value == "" {
		return errors.New("value is required")
	}

	meta := KVNodeMetadata{KeyName: spec.Key}
	namespaceName, err := resolveKVNamespaceMetadata(ctx, accountID, spec.Namespace)
	if err != nil {
		return err
	}
	meta.NamespaceName = namespaceName
	return ctx.Metadata.Set(meta)
}

func (c *PutKVValue) Execute(ctx core.ExecutionContext) error {
	spec := PutKVValueSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	accountID := resolveAccountID(spec.AccountID, ctx.Integration)
	if accountID == "" {
		return errors.New("accountId is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	if err := client.PutKVValue(accountID, spec.Namespace, spec.Key, spec.Value, spec.ExpirationTTL); err != nil {
		return fmt.Errorf("failed to put KV value: %v", err)
	}

	result := map[string]any{
		"accountId":   accountID,
		"namespaceId": spec.Namespace,
		"key":         spec.Key,
		"written":     true,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudflare.kv.value.put",
		[]any{result},
	)
}

func (c *PutKVValue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *PutKVValue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *PutKVValue) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *PutKVValue) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *PutKVValue) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *PutKVValue) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
