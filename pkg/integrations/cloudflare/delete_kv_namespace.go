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

type DeleteKVNamespace struct{}

type DeleteKVNamespaceSpec struct {
	AccountID string `json:"accountId"`
	Namespace string `json:"namespace"`
}

func (c *DeleteKVNamespace) Name() string {
	return "cloudflare.deleteKVNamespace"
}

func (c *DeleteKVNamespace) Label() string {
	return "Delete KV Namespace"
}

func (c *DeleteKVNamespace) Description() string {
	return "Delete a Cloudflare Workers KV namespace"
}

func (c *DeleteKVNamespace) Documentation() string {
	return `The Delete KV Namespace component permanently removes a Cloudflare Workers KV namespace and all of its key-value pairs.

## Use Cases

- **Environment teardown**: Remove namespaces as part of infrastructure cleanup
- **Pipeline cleanup**: Delete temporary namespaces created during a workflow

## Configuration

- **Namespace**: The KV namespace to delete

## Output

Emits a confirmation with the account ID, namespace ID, and a deleted flag.

> **Warning**: This operation is irreversible. All keys stored in the namespace will be permanently deleted.`
}

func (c *DeleteKVNamespace) Icon() string {
	return "cloud"
}

func (c *DeleteKVNamespace) Color() string {
	return "orange"
}

func (c *DeleteKVNamespace) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteKVNamespace) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "namespace",
			Label:       "Namespace",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The KV namespace to delete",
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
	}
}

func (c *DeleteKVNamespace) Setup(ctx core.SetupContext) error {
	spec := DeleteKVNamespaceSpec{}
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

	namespaceName, err := resolveKVNamespaceMetadata(ctx, accountID, spec.Namespace)
	if err != nil {
		return err
	}
	return ctx.Metadata.Set(KVNodeMetadata{NamespaceName: namespaceName})
}

func (c *DeleteKVNamespace) Execute(ctx core.ExecutionContext) error {
	spec := DeleteKVNamespaceSpec{}
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

	if err := client.DeleteKVNamespace(accountID, spec.Namespace); err != nil {
		return fmt.Errorf("failed to delete KV namespace: %v", err)
	}

	namespaceName := ""
	if ctx.NodeMetadata != nil {
		meta := KVNodeMetadata{}
		mapstructure.Decode(ctx.NodeMetadata.Get(), &meta)
		namespaceName = meta.NamespaceName
	}

	result := map[string]any{
		"accountId": accountID,
		"namespace": map[string]any{
			"id":    spec.Namespace,
			"title": namespaceName,
		},
		"deleted": true,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudflare.kv.namespace.deleted",
		[]any{result},
	)
}

func (c *DeleteKVNamespace) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteKVNamespace) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteKVNamespace) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteKVNamespace) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DeleteKVNamespace) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *DeleteKVNamespace) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
