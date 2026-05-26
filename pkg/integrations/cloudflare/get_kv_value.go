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

type GetKVValue struct{}

type GetKVValueSpec struct {
	AccountID string `json:"accountId"`
	Namespace string `json:"namespace"`
	KVKey     string `json:"kvKey"`
}

func (c *GetKVValue) Name() string {
	return "cloudflare.getKVValue"
}

func (c *GetKVValue) Label() string {
	return "Get KV Value"
}

func (c *GetKVValue) Description() string {
	return "Read a value by key from a Cloudflare Workers KV namespace"
}

func (c *GetKVValue) Documentation() string {
	return `The Get KV Value component reads a value by key from a Cloudflare Workers KV namespace.

## Use Cases

- **Feature flag checks**: Read a feature flag value before proceeding
- **Configuration lookup**: Retrieve dynamic configuration at workflow runtime
- **Audit**: Capture the current value of a key as part of a pipeline snapshot

## Configuration

- **Namespace**: The ID of the KV namespace to read from
- **Key**: The key to retrieve

## Output

Returns the account ID, namespace ID, key, and the retrieved value string.`
}

func (c *GetKVValue) Icon() string {
	return "cloud"
}

func (c *GetKVValue) Color() string {
	return "orange"
}

func (c *GetKVValue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetKVValue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "namespace",
			Label:       "Namespace",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The KV namespace to read from",
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
			Name:        "kvKey",
			Label:       "Key",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The key to retrieve",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "kv_key",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "namespace",
							ValueFrom: &configuration.ParameterValueFrom{Field: "namespace"},
						},
						{
							Name:      "accountId",
							ValueFrom: &configuration.ParameterValueFrom{Field: "accountId"},
						},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "namespace", Values: []string{"*"}},
			},
		},
	}
}

func (c *GetKVValue) Setup(ctx core.SetupContext) error {
	spec := GetKVValueSpec{}
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

	if spec.KVKey == "" {
		return errors.New("key is required")
	}

	meta := KVNodeMetadata{KeyName: spec.KVKey}
	namespaceName, err := resolveKVNamespaceMetadata(ctx, accountID, spec.Namespace)
	if err != nil {
		return err
	}
	meta.NamespaceName = namespaceName
	return ctx.Metadata.Set(meta)
}

func (c *GetKVValue) Execute(ctx core.ExecutionContext) error {
	spec := GetKVValueSpec{}
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

	value, err := client.GetKVValue(accountID, spec.Namespace, spec.KVKey)
	if err != nil {
		return fmt.Errorf("failed to get KV value: %v", err)
	}

	result := map[string]any{
		"accountId":   accountID,
		"namespaceId": spec.Namespace,
		"key":         spec.KVKey,
		"value":       value,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudflare.kv.value.fetched",
		[]any{result},
	)
}

func (c *GetKVValue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetKVValue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetKVValue) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetKVValue) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetKVValue) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetKVValue) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
