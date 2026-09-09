package pricesync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUSDPerTokenToCentsPerMillion(t *testing.T) {
	assert.Equal(t, int64(300), USDPerTokenToCentsPerMillion(0.000003))
	assert.Equal(t, int64(1500), USDPerTokenToCentsPerMillion(0.000015))
	assert.Equal(t, int64(30), USDPerTokenToCentsPerMillion(0.0000003))
	assert.Equal(t, int64(0), USDPerTokenToCentsPerMillion(0))
	assert.Equal(t, int64(0), USDPerTokenToCentsPerMillion(-1))
}

func TestParseUSDPerToken(t *testing.T) {
	assert.Equal(t, int64(300), parseUSDPerToken("0.000003"))
	assert.Equal(t, int64(0), parseUSDPerToken(""))
	assert.Equal(t, int64(0), parseUSDPerToken("not-a-number"))
}
