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

type DeleteKVValue struct{}

type DeleteKVValueSpec struct {
	AccountID string `json:"accountId"`
	Namespace string `json:"namespace"`
	KVKey     string `json:"kvKey"`
}

func (c *DeleteKVValue) Name() string {
	return "cloudflare.deleteKVValue"
}

func (c *DeleteKVValue) Label() string {
	return "Delete KV Value"
}

func (c *DeleteKVValue) Description() string {
	return "Delete a key from a Cloudflare Workers KV namespace"
}

func (c *DeleteKVValue) Documentation() string {
	return `The Delete KV Value component removes a key-value pair from a Cloudflare Workers KV namespace.

## Use Cases

- **Cleanup**: Remove stale keys as part of a teardown workflow
- **Feature flag rollback**: Delete a flag to revert to default behaviour
- **Session invalidation**: Remove session keys to force re-authentication

## Configuration

- **Namespace**: The ID of the KV namespace to delete from
- **Key**: The key to delete

## Output

Emits a confirmation with the account ID, namespace ID, and key that was deleted.

> **Warning**: This operation is irreversible. The key and its value will be permanently removed.`
}

func (c *DeleteKVValue) Icon() string {
	return "cloud"
}

func (c *DeleteKVValue) Color() string {
	return "orange"
}

func (c *DeleteKVValue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteKVValue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "namespace",
			Label:       "Namespace",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The KV namespace to delete from",
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
			Description: "The key to delete",
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

func (c *DeleteKVValue) Setup(ctx core.SetupContext) error {
	spec := DeleteKVValueSpec{}
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

func (c *DeleteKVValue) Execute(ctx core.ExecutionContext) error {
	spec := DeleteKVValueSpec{}
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

	if err := client.DeleteKVValue(accountID, spec.Namespace, spec.KVKey); err != nil {
		return fmt.Errorf("failed to delete KV value: %v", err)
	}

	result := map[string]any{
		"accountId":   accountID,
		"namespaceId": spec.Namespace,
		"key":         spec.KVKey,
		"deleted":     true,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudflare.kv.value.deleted",
		[]any{result},
	)
}

func (c *DeleteKVValue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteKVValue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteKVValue) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteKVValue) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DeleteKVValue) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *DeleteKVValue) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
