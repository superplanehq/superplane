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

func TestPoolMarshalJSON_IncludesMinimumOriginsWhenZero(t *testing.T) {
	pool := Pool{Name: "pool-a", Enabled: true, MinimumOrigins: 0}

	raw, err := json.Marshal(pool)
	require.NoError(t, err)

	assert.Contains(t, string(raw), `"minimum_origins":0`)
}

func TestCreatePoolRequestMarshalJSON_IncludesMinimumOriginsWhenZero(t *testing.T) {
	z := 0
	req := CreatePoolRequest{Name: "pool-a", Enabled: true, Origins: []Origin{}, MinimumOrigins: &z}

	raw, err := json.Marshal(req)
	require.NoError(t, err)

	assert.Contains(t, string(raw), `"minimum_origins":0`)
}
