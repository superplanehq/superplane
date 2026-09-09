package workers

import (
	"context"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/usage/pricesync"
	"gorm.io/gorm"
)

func TestPriceBookSyncInterval(t *testing.T) {
	t.Setenv("PRICE_BOOK_SYNC_EVERY", "")
	assert.Equal(t, defaultPriceBookSyncEvery, priceBookSyncInterval())

	t.Setenv("PRICE_BOOK_SYNC_EVERY", "not-a-duration")
	assert.Equal(t, defaultPriceBookSyncEvery, priceBookSyncInterval())

	t.Setenv("PRICE_BOOK_SYNC_EVERY", "15m")
	assert.Equal(t, 15*time.Minute, priceBookSyncInterval())
}

func TestPriceBookSyncWorkerTick_PublishesScan(t *testing.T) {
	called := false
	worker := &PriceBookSyncWorker{
		logger:   log.NewEntry(log.New()),
		interval: time.Hour,
		scan: func(context.Context, *gorm.DB) (*pricesync.Result, error) {
			called = true
			return &pricesync.Result{Version: "2099-01-02.1"}, nil
		},
	}

	worker.tick(context.Background())
	require.True(t, called)
}
