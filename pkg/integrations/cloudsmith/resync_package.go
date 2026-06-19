package cloudsmith

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ResyncPackage struct{}

func (r *ResyncPackage) Name() string {
	return "cloudsmith.resyncPackage"
}

func (r *ResyncPackage) Label() string {
	return "Resync Package"
}

func (r *ResyncPackage) Description() string {
	return "Schedule a Cloudsmith package for resynchronization"
}

func (r *ResyncPackage) Documentation() string {
	return `The Resync Package component schedules a Cloudsmith package for resynchronization.

## Configuration

- **Repository**: The repository that contains the package.
- **Package Identifier**: The Cloudsmith package identifier (` + "`slug_perm`" + `).

## Output

Emits the package returned by Cloudsmith on the default channel.`
}

func (r *ResyncPackage) Icon() string {
	return "refresh-cw"
}

func (r *ResyncPackage) Color() string {
	return "gray"
}

func (r *ResyncPackage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (r *ResyncPackage) Configuration() []configuration.Field {
	return packageConfigurationFields()
}

func (r *ResyncPackage) Setup(ctx core.SetupContext) error {
	return setupPackageComponent(ctx)
}

func (r *ResyncPackage) Execute(ctx core.ExecutionContext) error {
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

	pkg, err := client.ResyncPackage(owner, repository, identifier)
	if err != nil {
		return fmt.Errorf("failed to resync package: %v", err)
	}
	if pkg == nil {
		return fmt.Errorf("resync returned empty response")
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, PackageResyncedPayloadType, []any{pkg})
}

func (r *ResyncPackage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (r *ResyncPackage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return defaultProcessQueueItem(ctx)
}

func (r *ResyncPackage) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return defaultHandleWebhook(ctx)
}

func (r *ResyncPackage) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (r *ResyncPackage) Hooks() []core.Hook {
	return []core.Hook{}
}

func (r *ResyncPackage) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
