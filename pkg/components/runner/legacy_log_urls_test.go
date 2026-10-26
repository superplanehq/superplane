package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBrowserTaskBrokerBaseURL(t *testing.T) {
	t.Setenv("TASK_BROKER_HOST_PORT", "")

	assert.Equal(t, "http://localhost:8091", browserTaskBrokerBaseURL("http://host.docker.internal:8091"))
	assert.Equal(t, "http://localhost:8091", browserTaskBrokerBaseURL("http://task-broker:8081"))
	assert.Equal(t, "https://broker.example", browserTaskBrokerBaseURL("https://broker.example"))
	assert.Equal(t, "http://localhost:8091", browserTaskBrokerBaseURL("http://localhost:8091"))

	t.Setenv("TASK_BROKER_HOST_PORT", "9091")
	assert.Equal(t, "http://localhost:9091", browserTaskBrokerBaseURL("http://task-broker:8081"))
}
