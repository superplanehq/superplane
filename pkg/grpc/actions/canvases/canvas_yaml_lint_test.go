package canvases

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"
)

func TestCanvasFromYAMLTextRejectsSnakeCaseConfigurationFields(t *testing.T) {
	_, err := canvasFromYAMLText(`apiVersion: v1
kind: Canvas
metadata:
  name: test
spec:
  nodes:
    - id: wait-1
      name: Wait
      component: wait
      configuration:
        duration_seconds: 30
  edges: []
`)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Contains(t, st.Message(), "duration_seconds")
	require.Contains(t, st.Message(), "durationSeconds")
}
