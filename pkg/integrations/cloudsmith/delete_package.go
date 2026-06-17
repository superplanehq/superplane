package cloudsmith

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeletePackage struct{}

func (d *DeletePackage) Name() string {
	return "cloudsmith.deletePackage"
}

func (d *DeletePackage) Label() string {
	return "Delete Package"
}

func (d *DeletePackage) Description() string {
	return "Delete a package from a Cloudsmith repository"
}

func (d *DeletePackage) Documentation() string {
	return `The Delete Package component deletes a package from a Cloudsmith repository.

## Configuration

- **Repository**: The repository that contains the package.
- **Package Identifier**: The Cloudsmith package identifier (` + "`slug_perm`" + `).

## Output

Emits the deleted package identifier on the default channel.`
}

func (d *DeletePackage) Icon() string {
	return "trash"
}

func (d *DeletePackage) Color() string {
	return "red"
}

func (d *DeletePackage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeletePackage) Configuration() []configuration.Field {
	return packageConfigurationFields()
}

func (d *DeletePackage) Setup(ctx core.SetupContext) error {
	return setupPackageComponent(ctx)
}

func (d *DeletePackage) Execute(ctx core.ExecutionContext) error {
	spec, err := decodePackageSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	owner, repository, identifier, err := packageRequestParts(spec)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	if err := client.DeletePackage(owner, repository, identifier); err != nil {
		return fmt.Errorf("failed to delete package: %v", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, PackageDeletedPayloadType, []any{packageResult(spec, nil)})
}

func (d *DeletePackage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeletePackage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return defaultProcessQueueItem(ctx)
}

func (d *DeletePackage) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return defaultHandleWebhook(ctx)
}

func (d *DeletePackage) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (d *DeletePackage) Hooks() []core.Hook {
	return []core.Hook{}
}

func (d *DeletePackage) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
