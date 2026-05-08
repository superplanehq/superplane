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

type CreateKVNamespace struct{}

type CreateKVNamespaceSpec struct {
	AccountID string `json:"accountId"`
	Title     string `json:"title"`
}

func (c *CreateKVNamespace) Name() string {
	return "cloudflare.createKVNamespace"
}

func (c *CreateKVNamespace) Label() string {
	return "Create KV Namespace"
}

func (c *CreateKVNamespace) Description() string {
	return "Create a Cloudflare Workers KV namespace"
}

func (c *CreateKVNamespace) Documentation() string {
	return `The Create KV Namespace component creates a new Cloudflare Workers KV namespace.

## Use Cases

- **Feature flags**: Provision a dedicated namespace to store feature flag state
- **Session storage**: Create a namespace for application session data
- **Configuration store**: Set up a namespace for dynamic configuration values

## Configuration

- **Title**: A human-readable name for the namespace (must be unique per account)

## Output

Returns the created namespace with its assigned ID and title.`
}

func (c *CreateKVNamespace) Icon() string {
	return "cloud"
}

func (c *CreateKVNamespace) Color() string {
	return "orange"
}

func (c *CreateKVNamespace) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateKVNamespace) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "title",
			Label:       "Namespace Title",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "A human-readable name for the KV namespace. Must be unique within the account.",
			Placeholder: "my-kv-namespace",
		},
	}
}

func (c *CreateKVNamespace) Setup(ctx core.SetupContext) error {
	spec := CreateKVNamespaceSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	accountID := resolveAccountID(spec.AccountID, ctx.Integration)
	if accountID == "" {
		return errors.New("accountId is required")
	}

	if spec.Title == "" {
		return errors.New("title is required")
	}

	return nil
}

func (c *CreateKVNamespace) Execute(ctx core.ExecutionContext) error {
	spec := CreateKVNamespaceSpec{}
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

	ns, err := client.CreateKVNamespace(accountID, CreateKVNamespaceRequest{Title: spec.Title})
	if err != nil {
		return fmt.Errorf("failed to create KV namespace: %v", err)
	}

	result := map[string]any{
		"namespace": ns,
		"accountId": accountID,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"cloudflare.kv.namespace.created",
		[]any{result},
	)
}

func (c *CreateKVNamespace) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateKVNamespace) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateKVNamespace) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateKVNamespace) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateKVNamespace) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateKVNamespace) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
