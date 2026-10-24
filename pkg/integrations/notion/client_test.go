package notion

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestSearchAPIErrors(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 429, 500, 529} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			r := response(status, `{"code":"validation_error","message":"Do not disclose test-token"}`)
			r.Header.Set("Retry-After", "12")
			h := &contexts.HTTPContext{Responses: []*http.Response{r}}
			state := &contexts.ExecutionStateContext{}
			err := (&Search{}).Execute(core.ExecutionContext{HTTP: h, Integration: integration(), ExecutionState: state, Configuration: map[string]any{}})
			var apiError *APIError
			require.ErrorAs(t, err, &apiError)
			assert.Equal(t, status, apiError.StatusCode)
			assert.Equal(t, "validation_error", apiError.Code)
			assert.Equal(t, "12", apiError.RetryAfter)
			assert.NotContains(t, err.Error(), "test-token")
			if status == 429 || status == 529 {
				assert.Contains(t, err.Error(), "12")
			}
			assert.Len(t, h.Requests, 1)
			assert.Empty(t, state.Payloads)
		})
	}
}

func TestSearchRejectsInvalidResponse(t *testing.T) {
	for _, body := range []string{`not json`, `null`, `[]`, `{}`} {
		h := &contexts.HTTPContext{Responses: []*http.Response{response(200, body)}}
		state := &contexts.ExecutionStateContext{}
		err := (&Search{}).Execute(core.ExecutionContext{HTTP: h, Integration: integration(), ExecutionState: state, Configuration: map[string]any{}})
		require.Error(t, err)
		assert.Empty(t, state.Payloads)
	}
}
