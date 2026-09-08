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
