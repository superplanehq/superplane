package bitbucket

import (
	"io"
	"net/http"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnCommitStatus__Setup(t *testing.T) {
	trigger := &OnCommitStatus{}
	metadata := Metadata{
		AuthType:  AuthTypeWorkspaceAccessToken,
		Workspace: &WorkspaceMetadata{Slug: "superplane"},
	}

	integrationContext := func() *contexts.IntegrationContext {
		return &contexts.IntegrationContext{
			Configuration: map[string]any{"token": "token"},
			Metadata:      metadata,
		}
	}

	t.Run("at least one state is required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: integrationContext(),
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"states":     []string{},
			},
		})

		require.ErrorContains(t, err, "at least one state is required")
	})

	t.Run("unsupported state is rejected", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: integrationContext(),
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"states":     []string{"GREEN"},
			},
		})

		require.ErrorContains(t, err, `unsupported build state "GREEN"`)
	})

	t.Run("both commit status events are requested", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"values":[{"uuid":"{hello}","name":"hello","full_name":"superplane/hello","slug":"hello"}]}`,
					)),
				},
			},
		}
		integrationCtx := integrationContext()

		require.NoError(t, trigger.Setup(core.TriggerContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"states":     []string{StateSuccessful},
			},
		}))

		require.Len(t, integrationCtx.WebhookRequests, 1)
		webhookRequest, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, []string{"repo:commit_status_created", "repo:commit_status_updated"}, webhookRequest.EventTypes)
	})
}

func Test__OnCommitStatus__HandleWebhook(t *testing.T) {
	trigger := &OnCommitStatus{}

	handle := func(eventKey string, body []byte, config map[string]any) (int, *contexts.EventContext, error) {
		headers := http.Header{}
		headers.Set("X-Event-Key", eventKey)
		headers.Set("X-Hub-Signature", "sha256="+signBitbucketPayload("test-secret", body))

		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Logger:        log.NewEntry(log.New()),
			Body:          body,
			Headers:       headers,
			Webhook:       &contexts.NodeWebhookContext{Secret: "test-secret"},
			Configuration: config,
			Events:        eventContext,
		})

		return code, eventContext, err
	}

	successOnly := map[string]any{
		"repository": "hello",
		"states":     []string{StateSuccessful},
	}

	t.Run("selected state -> event is emitted", func(t *testing.T) {
		body := []byte(`{"commit_status":{"key":"BITBUCKETPIPELINE","state":"SUCCESSFUL","refname":"main"}}`)

		code, eventContext, err := handle("repo:commit_status_updated", body, successOnly)

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "bitbucket.commitStatus", eventContext.Payloads[0].Type)
	})

	t.Run("state that was not selected -> event is not emitted", func(t *testing.T) {
		body := []byte(`{"commit_status":{"key":"BITBUCKETPIPELINE","state":"INPROGRESS","refname":"main"}}`)

		code, eventContext, err := handle("repo:commit_status_updated", body, successOnly)

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("payload without a commit status -> event is not emitted", func(t *testing.T) {
		code, eventContext, err := handle("repo:commit_status_created", []byte(`{}`), successOnly)

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("key filter is applied", func(t *testing.T) {
		config := map[string]any{
			"repository": "hello",
			"states":     []string{StateSuccessful},
			"keys": []configuration.Predicate{
				{Type: configuration.PredicateTypeEquals, Value: "integration-tests"},
			},
		}

		body := []byte(`{"commit_status":{"key":"BITBUCKETPIPELINE","state":"SUCCESSFUL","refname":"main"}}`)
		code, eventContext, err := handle("repo:commit_status_updated", body, config)

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())

		body = []byte(`{"commit_status":{"key":"integration-tests","state":"SUCCESSFUL","refname":"main"}}`)
		code, eventContext, err = handle("repo:commit_status_updated", body, config)

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("ref filter is applied", func(t *testing.T) {
		config := map[string]any{
			"repository": "hello",
			"states":     []string{StateSuccessful},
			"refs": []configuration.Predicate{
				{Type: configuration.PredicateTypeEquals, Value: "main"},
			},
		}

		body := []byte(`{"commit_status":{"key":"BITBUCKETPIPELINE","state":"SUCCESSFUL","refname":"feature-1"}}`)
		code, eventContext, err := handle("repo:commit_status_updated", body, config)

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})
}

func Test__CombineCommitStatusStates(t *testing.T) {
	t.Run("no statuses is not a pass", func(t *testing.T) {
		assert.Equal(t, StateNoStatus, combineCommitStatusStates(nil))
	})

	t.Run("all successful", func(t *testing.T) {
		assert.Equal(t, StateSuccessful, combineCommitStatusStates([]CommitStatus{
			{State: StateSuccessful},
			{State: StateSuccessful},
		}))
	})

	t.Run("a failure outranks everything", func(t *testing.T) {
		assert.Equal(t, StateFailed, combineCommitStatusStates([]CommitStatus{
			{State: StateSuccessful},
			{State: StateInProgress},
			{State: StateStopped},
			{State: StateFailed},
		}))
	})

	t.Run("a stopped build outranks a running one", func(t *testing.T) {
		assert.Equal(t, StateStopped, combineCommitStatusStates([]CommitStatus{
			{State: StateSuccessful},
			{State: StateInProgress},
			{State: StateStopped},
		}))
	})

	t.Run("a running build outranks a successful one", func(t *testing.T) {
		assert.Equal(t, StateInProgress, combineCommitStatusStates([]CommitStatus{
			{State: StateSuccessful},
			{State: StateInProgress},
		}))
	})

	// An unknown state must never let a deploy gate through.
	t.Run("an unrecognized state is treated as a failure", func(t *testing.T) {
		assert.Equal(t, StateFailed, combineCommitStatusStates([]CommitStatus{
			{State: StateSuccessful},
			{State: "SOMETHING_NEW"},
		}))
	})
}
