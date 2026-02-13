package linear

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateIssue__Setup(t *testing.T) {
	component := &CreateIssue{}

	t.Run("missing team -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "x"}},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"title": "Fix bug"},
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "team is required")
	})

	t.Run("missing title -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "x"}},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "t1"},
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "title is required")
	})

	t.Run("team not found -> error", func(t *testing.T) {
		teamsResp := `{"data":{"teams":{"nodes":[{"id":"other","name":"Other","key":"O"}]}}}`
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(teamsResp))},
			},
		}
		appCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "key"}}
		err := component.Setup(core.SetupContext{
			HTTP:          httpCtx,
			Integration:   appCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "t1", "title": "Task"},
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "not found")
	})

	t.Run("success", func(t *testing.T) {
		teamsResp := `{"data":{"teams":{"nodes":[{"id":"t1","name":"Eng","key":"ENG"}]}}}`
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(teamsResp))},
			},
		}
		appCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "key"}}
		metaCtx := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			HTTP:          httpCtx,
			Integration:   appCtx,
			Metadata:      metaCtx,
			Configuration: map[string]any{"team": "t1", "title": "Task"},
		})
		require.NoError(t, err)
		md, _ := metaCtx.Get().(NodeMetadata)
		require.NotNil(t, md.Team)
		assert.Equal(t, "t1", md.Team.ID)
	})
}

func Test__CreateIssue__Execute(t *testing.T) {
	component := &CreateIssue{}

	t.Run("success", func(t *testing.T) {
		createResp := `{"data":{"issueCreate":{"success":true,"issue":{"id":"i1","identifier":"ENG-1","title":"Task","team":{"id":"t1"},"state":{"id":"s1"}}}}}`
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(createResp))},
			},
		}
		appCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "key"}}
		execState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    appCtx,
			Configuration:  map[string]any{"team": "t1", "title": "Task"},
			ExecutionState: execState,
		})
		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
	})
}
