package runagent

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__setAuthHeader(t *testing.T) {
	tests := []struct {
		name              string
		apiKey            string
		expectedAuthValue string
		expectedAPIKeySet bool
	}{
		{name: "regular API key uses x-api-key", apiKey: "sk-ant-api03-abc", expectedAPIKeySet: true},
		{name: "OAuth token uses bearer auth", apiKey: "sk-ant-oat01-abc", expectedAuthValue: "Bearer sk-ant-oat01-abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
			require.NoError(t, err)

			setAuthHeader(req, tt.apiKey)

			if tt.expectedAPIKeySet {
				assert.Equal(t, tt.apiKey, req.Header.Get("x-api-key"))
				assert.Empty(t, req.Header.Get("Authorization"))
			} else {
				assert.Equal(t, tt.expectedAuthValue, req.Header.Get("Authorization"))
				assert.Empty(t, req.Header.Get("x-api-key"))
			}
		})
	}
}
