package public

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveHostedListModelsBaseURL(t *testing.T) {
	proxy := "https://proxy.example/v1"
	stored := "https://saved.example/v1"

	assert.Equal(t, proxy, resolveHostedListModelsBaseURL(&proxy, stored))
	empty := ""
	assert.Equal(t, "", resolveHostedListModelsBaseURL(&empty, stored))
	assert.Equal(t, stored, resolveHostedListModelsBaseURL(nil, stored))
	assert.Equal(t, "", resolveHostedListModelsBaseURL(nil, ""))
}
