package render

import (
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type RenderServiceNodeMetadata struct {
	Service *RenderServiceMetadata `json:"service,omitempty" mapstructure:"service"`
}

type RenderServiceMetadata struct {
	ID   string `json:"id" mapstructure:"id"`
	Name string `json:"name" mapstructure:"name"`
}

func setServiceNodeMetadata(ctx core.SetupContext, serviceID string) error {
	serviceID = strings.TrimSpace(serviceID)
	if serviceID == "" || strings.Contains(serviceID, "{{") || ctx.Metadata == nil || ctx.HTTP == nil || ctx.Integration == nil {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	service, err := client.GetService(serviceID)
	if err != nil {
		return err
	}

	return ctx.Metadata.Set(RenderServiceNodeMetadata{
		Service: &RenderServiceMetadata{
			ID:   service.ID,
			Name: service.Name,
		},
	})
}
