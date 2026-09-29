package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/storedfiles"
)

func TestRewriteLoopbackTaskFileURLsUsesDockerHostForLocalBroker(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "http://task-broker:8081")
	t.Setenv("BASE_URL", "http://localhost:8000")
	t.Setenv("PUBLIC_API_PORT", "8000")

	raw := "http://localhost:8000/api/v1/public/files/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa?expires=1&sig=abc&sp_file=1"
	text, files := RewriteLoopbackTaskFileURLs("See "+raw, []storedfiles.DispatchFile{{URL: raw}})

	want := "http://host.docker.internal:8000/api/v1/public/files/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa?expires=1&sig=abc&sp_file=1"
	assert.Equal(t, "See "+want, text)
	assert.Equal(t, want, files[0].URL)
}

func TestRewriteLoopbackTaskFileURLsLeavesPublicHosts(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "http://task-broker:8081")
	t.Setenv("BASE_URL", "https://app.example")

	raw := "https://app.example/api/v1/public/files/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa?expires=1&sig=abc&sp_file=1"
	gcs := "https://storage.googleapis.com/bucket/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa?sp_file=1"
	text, files := RewriteLoopbackTaskFileURLs(raw+" "+gcs, []storedfiles.DispatchFile{{URL: raw}, {URL: gcs}})

	assert.Equal(t, raw+" "+gcs, text)
	assert.Equal(t, raw, files[0].URL)
	assert.Equal(t, gcs, files[1].URL)
}

func TestRewriteLoopbackTaskFileURLsLeavesRemoteBrokers(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("BASE_URL", "http://localhost:8000")

	raw := "http://localhost:8000/api/v1/public/files/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa?expires=1&sig=abc&sp_file=1"
	text, files := RewriteLoopbackTaskFileURLs("See "+raw, []storedfiles.DispatchFile{{URL: raw}})

	assert.Equal(t, "See "+raw, text)
	assert.Equal(t, raw, files[0].URL)
}
