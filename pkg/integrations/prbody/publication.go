package prbody

import (
	"context"
	"fmt"
	"strings"
	"time"

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

type Status string

const (
	StatusPublished Status = "published"
	StatusUnchanged Status = "unchanged"
	StatusSkipped   Status = "skipped"
)

type Outcome struct {
	Status Status `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func Skipped(reason string) Outcome { return Outcome{Status: StatusSkipped, Reason: reason} }

// Publish rechecks eligibility after the read so excluded publication cannot complete or write.
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
	active, err = canPublish()
	if err != nil {
		return Outcome{}, err
	}
	if !active {
		return Skipped("Publication was disabled, cancelled, or superseded."), nil
	}
	if body == latest.Body {
		return Outcome{Status: StatusUnchanged}, nil
	}
	if err := client.Write(ctx, config.Repository, number, body); err != nil {
		return Outcome{}, err
	}
	return Outcome{Status: StatusPublished}, nil
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

// DescriptionKey is shared by every section of one pull request description.
func (c Configuration) DescriptionKey() string {
	return fmt.Sprintf("pull-request-description/%s/%s", strings.ToLower(c.Repository), c.PullNumber)
}

// PublishSerialized holds the description lease from the read to the write.
// Writers of different sections build their body from the same description,
// so without the lease the last write removes the update of the other section.
// It returns core.ErrPublicationBusy while another section write is in progress.
func PublishSerialized(ctx context.Context, client Client, store core.PublicationStore, config Configuration, eligible func() (bool, error)) (Outcome, error) {
	key := config.DescriptionKey()
	lease, err := core.BeginPublicationLease(store, key, orderedWriteLease)
	if err != nil {
		return Outcome{}, err
	}
	publishCtx, cancel := context.WithDeadline(ctx, lease.ExpiresAt.Add(-orderedWriteMargin))
	defer cancel()
	outcome, err := Publish(publishCtx, client, config, eligible)
	if releaseErr := core.FinishPublicationLease(store, key, lease); releaseErr != nil && err == nil {
		return outcome, releaseErr
	}
	return outcome, err
}

const orderedWriteLease = time.Minute
const orderedWriteMargin = 15 * time.Second

type orderedClient struct {
	Client
	store core.PublicationStore
	write core.OrderedWrite
}

// OrderedClient claims the publication boundary for each description write.
// Write returns core.ErrPublicationSuperseded after a newer source or attempt was accepted.
func OrderedClient(client Client, store core.PublicationStore, write core.OrderedWrite) Client {
	return orderedClient{Client: client, store: store, write: write}
}

func (c orderedClient) Write(ctx context.Context, repository string, number int64, body string) error {
	lease, _, err := core.BeginOrderedWrite(c.store, c.write, orderedWriteLease)
	if err != nil {
		return err
	}
	writeCtx, cancel := context.WithDeadline(ctx, lease.ExpiresAt.Add(-orderedWriteMargin))
	defer cancel()
	writeErr := c.Client.Write(writeCtx, repository, number, body)
	if err := core.FinishOrderedWrite(c.store, c.write, lease, nil); err != nil && writeErr == nil {
		return err
	}
	return writeErr
}
