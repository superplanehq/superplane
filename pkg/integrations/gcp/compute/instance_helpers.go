package compute

import (
	"context"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
)

// resolveInstanceNodeMetadata stores the instance name and zone on the node so
// the collapsed UI can display something meaningful. It best-effort resolves
// the canonical instance name via the API, falling back to the parsed values
// when credentials or the API are unavailable. Components that target an
// existing VM instance (manage power, update machine type, get metrics) share
// this behaviour.
func resolveInstanceNodeMetadata(ctx core.SetupContext, instanceValue string) error {
	// Expressions are resolved at execution time. Store the raw value so the UI
	// can still display something meaningful in the collapsed node.
	if strings.Contains(instanceValue, "{{") {
		return ctx.Metadata.Set(VMInstanceNodeMetadata{
			InstanceName: instanceValue,
		})
	}

	_, zone, name, err := parseInstancePath(instanceValue)
	if err != nil {
		return err
	}

	// If metadata is already set for the same instance, skip the API call.
	var existing VMInstanceNodeMetadata
	if decErr := mapstructure.Decode(ctx.Metadata.Get(), &existing); decErr == nil &&
		existing.InstanceName == name && existing.Zone == zone {
		return nil
	}

	fallback := VMInstanceNodeMetadata{InstanceName: name, Zone: zone}

	if ctx.Integration == nil {
		return ctx.Metadata.Set(fallback)
	}

	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.Metadata.Set(fallback)
	}

	body, err := GetInstance(context.Background(), client, client.ProjectID(), zone, name)
	if err != nil {
		return ctx.Metadata.Set(fallback)
	}

	payload, err := InstancePayloadFromGetResponse(body, zone)
	if err != nil {
		return ctx.Metadata.Set(fallback)
	}

	resolvedName, _ := payload["name"].(string)
	if resolvedName == "" {
		resolvedName = name
	}

	return ctx.Metadata.Set(VMInstanceNodeMetadata{
		InstanceName: resolvedName,
		Zone:         zone,
	})
}
