package workers

import (
	"context"
	"io"
	"sync/atomic"

	"github.com/superplanehq/superplane/pkg/blob"
)

type blockingFinalPutProvider struct {
	blob.Provider
	finalKey  string
	started   chan struct{}
	release   chan struct{}
	finalPuts atomic.Int32
}

func newBlockingFinalPutProvider(
	provider blob.Provider,
	finalKey string,
) *blockingFinalPutProvider {
	return &blockingFinalPutProvider{
		Provider: provider,
		finalKey: finalKey,
		started:  make(chan struct{}),
		release:  make(chan struct{}, 1),
	}
}

func (p *blockingFinalPutProvider) Put(
	ctx context.Context,
	key string,
	reader io.Reader,
	options blob.PutOptions,
) error {
	if key == p.finalKey {
		if p.finalPuts.Add(1) == 1 {
			close(p.started)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-p.release:
			}
		}
	}
	return p.Provider.Put(ctx, key, reader, options)
}
