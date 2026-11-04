package prbody

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
)

type Description struct {
	Body    string
	Open    bool
	HeadSHA string
}

type Client interface {
	Read(context.Context, string, int64) (Description, error)
	Write(context.Context, string, int64, string) error
}

type Outcome struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func Skipped(reason string) Outcome { return Outcome{Status: "skipped", Reason: reason} }

// Publish rechecks eligibility after the read so cancellation and newer runs can stop a pending write.
func Publish(ctx context.Context, client Client, config Configuration, eligible func() (bool, error)) (Outcome, error) {
	if err := config.Validate(); err != nil {
		return Outcome{}, err
	}
	number, err := config.Number()
	if err != nil {
		return Outcome{}, err
	}
	if !config.IsEnabled() {
		return Skipped("Publication is disabled for this run."), nil
	}
	canPublish := func() (bool, error) {
		if eligible == nil {
			return true, nil
		}
		return eligible()
	}
	active, err := canPublish()
	if err != nil {
		return Outcome{}, err
	}
	if !active {
		return Skipped("Publication is disabled, cancelled, or superseded."), nil
	}
	latest, err := client.Read(ctx, config.Repository, number)
	if err != nil {
		return Outcome{}, err
	}
	if !latest.Open || (config.ExpectedHeadSHA != "" && latest.HeadSHA != config.ExpectedHeadSHA) {
		return Skipped("Pull request is closed or its expected commit changed."), nil
	}
	body, err := UpdateBodySection(latest.Body, config.SectionKey, config.Body)
	if err != nil {
		return Skipped("Description contains invalid section markers."), nil
	}
	if body == latest.Body {
		return Skipped("Section is already current."), nil
	}
	active, err = canPublish()
	if err != nil {
		return Outcome{}, err
	}
	if !active {
		return Skipped("Publication was disabled, cancelled, or superseded before writing."), nil
	}
	if err := client.Write(ctx, config.Repository, number, body); err != nil {
		return Outcome{}, err
	}
	return Outcome{Status: "published"}, nil
}

func HandleHook(ctx core.ActionHookContext, client Client) error {
	if ctx.Name != HookName {
		return fmt.Errorf("unknown hook: %s", ctx.Name)
	}
	config, err := Decode(ctx.Configuration)
	if err != nil {
		return err
	}
	outcome, err := Publish(context.Background(), client, config, nil)
	if err != nil {
		return err
	}
	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, PayloadType, []any{outcome})
}
