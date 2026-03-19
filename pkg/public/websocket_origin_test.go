package public

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test__newWebSocketCheckOrigin__ProductionComparisonIsCaseInsensitive(t *testing.T) {
	checkOrigin := newWebSocketCheckOrigin("production", "HTTPS://App.Example.com")

	request, err := http.NewRequest(http.MethodGet, "/ws", nil)
	require.NoError(t, err)
	request.Header.Set("Origin", "https://app.example.com")

	require.True(t, checkOrigin(request))
}

func Test__newWebSocketCheckOrigin__ProductionRejectsEmptyOrigin(t *testing.T) {
	checkOrigin := newWebSocketCheckOrigin("production", "https://app.example.com")

	request, err := http.NewRequest(http.MethodGet, "/ws", nil)
	require.NoError(t, err)

	require.False(t, checkOrigin(request))
}
