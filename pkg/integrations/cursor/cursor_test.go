package cursor

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Cursor__Sync(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	t.Run("cloudAgentsApiKey required", func(t *testing.T) {
		i := &Cursor{}
		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{}}
		httpCtx := &contexts.HTTPContext{}

		err := i.Sync(core.SyncContext{
			Logger:          logger,
			Configuration:   map[string]any{"cloudAgentsApiKey": ""},
			HTTP:            httpCtx,
			Integration:     integrationCtx,
			WebhooksBaseURL: "http://localhost:8000",
		})

		assert.Error(t, err)
	})

	t.Run("cloudAgentsApiKey verified -> ready", func(t *testing.T) {
		i := &Cursor{}
		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"cloudAgentsApiKey": "test"}}

		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"id":"me"}`)),
			},
		}}

		err := i.Sync(core.SyncContext{
			Logger:          logger,
			Configuration:   map[string]any{"cloudAgentsApiKey": "test"},
			HTTP:            httpCtx,
			Integration:     integrationCtx,
			WebhooksBaseURL: "http://localhost:8000",
		})

		assert.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
	})
}
