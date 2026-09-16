package blob

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDispatchDownloadTTL(t *testing.T) {
	assert.Equal(t, time.Hour+30*time.Minute, DispatchDownloadTTL(0))
	assert.Equal(t, time.Hour+30*time.Minute, DispatchDownloadTTL(60))
	assert.Equal(t, 2*time.Hour+30*time.Minute, DispatchDownloadTTL(7200))
}

func TestStableExpiryStaysInTheSameBucket(t *testing.T) {
	first := StableExpiry(time.Hour)
	second := StableExpiry(time.Hour)
	assert.True(t, first.Equal(second))
	assert.True(t, first.After(time.Now().Add(time.Hour-time.Second)))
}
