package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLocalTaskBrokerURL(t *testing.T) {
	t.Parallel()

	assert.True(t, isLocalTaskBrokerURL("http://host.docker.internal:8091"))
	assert.True(t, isLocalTaskBrokerURL("http://localhost:8091"))
	assert.True(t, isLocalTaskBrokerURL("http://127.0.0.1:8091"))
	assert.False(t, isLocalTaskBrokerURL("https://broker.example"))
	assert.False(t, isLocalTaskBrokerURL(""))
}

func TestBrowserTaskBrokerBaseURL(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "http://localhost:8091", browserTaskBrokerBaseURL("http://host.docker.internal:8091"))
	assert.Equal(t, "https://broker.example", browserTaskBrokerBaseURL("https://broker.example"))
	assert.Equal(t, "http://localhost:8091", browserTaskBrokerBaseURL("http://localhost:8091"))
}
