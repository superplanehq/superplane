package cloudflare

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolMarshalJSON_IncludesEnabledWhenFalse(t *testing.T) {
	pool := Pool{Name: "disabled-pool", Enabled: false}

	raw, err := json.Marshal(pool)
	require.NoError(t, err)

	assert.Contains(t, string(raw), `"enabled":false`)
}
