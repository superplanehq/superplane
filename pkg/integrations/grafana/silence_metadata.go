package grafana

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type SilenceNodeMetadata struct {
	Comment string `json:"comment,omitempty" mapstructure:"comment"`
	State   string `json:"state,omitempty" mapstructure:"state"`
	Label   string `json:"label,omitempty" mapstructure:"label"`
}

func resolveSilenceNodeMetadata(ctx core.SetupContext, silenceID string) error {
	silenceID = strings.TrimSpace(silenceID)
	if silenceID == "" || isTemplateExpression(silenceID) {
		return ctx.Metadata.Set(SilenceNodeMetadata{})
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client during setup: %w", err)
	}

	silence, err := client.GetSilence(silenceID)
	if err != nil {
		return fmt.Errorf("error getting silence during setup: %w", err)
	}

	if silence == nil {
		return ctx.Metadata.Set(SilenceNodeMetadata{})
	}

	return ctx.Metadata.Set(SilenceNodeMetadata{
		Comment: strings.TrimSpace(silence.Comment),
		State:   strings.TrimSpace(silence.Status.State),
		Label:   strings.TrimSpace(formatSilenceResourceLabel(*silence)),
	})
}

func isTemplateExpression(value string) bool {
	trimmed := strings.TrimSpace(value)
	return strings.Contains(trimmed, "{{") && strings.Contains(trimmed, "}}")
}
