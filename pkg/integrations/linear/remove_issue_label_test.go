package linear

import (
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func removeLabelConfig() map[string]any {
	return map[string]any{"team": "t1", "issue": "ENG-142", "labels": []string{"l1"}}
}

func issueWithLabelsResponse() string {
	return `{"data":{"issueRemoveLabel":{"success":true,"issue":{"id":"i1","identifier":"ENG-142","title":"Boom","labels":{"nodes":[{"id":"l2","name":"backend"}]}}}}}`
}

func labelNotOnIssueResponse() string {
	return `{"errors":[{"message":"Label not on issue"}]}`
}

func Test__RemoveIssueLabel__Setup(t *testing.T) {
	component := RemoveIssueLabel{}

	t.Run("missing team -> error", func(t *testing.T) {
		config := removeLabelConfig()
		delete(config, "team")

		err := component.Setup(core.SetupContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: config,
		})

		require.ErrorContains(t, err, "team is required")
	})

	t.Run("unknown team -> error", func(t *testing.T) {
		config := removeLabelConfig()
		config["team"] = "other"

		err := component.Setup(core.SetupContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: config,
		})

		require.ErrorContains(t, err, "team other not found")
	})

	t.Run("missing issue -> error", func(t *testing.T) {
		config := removeLabelConfig()
		delete(config, "issue")

		err := component.Setup(core.SetupContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: config,
		})

		require.ErrorContains(t, err, "issue is required")
	})

	t.Run("missing labels -> error", func(t *testing.T) {
		config := removeLabelConfig()
		delete(config, "labels")

		err := component.Setup(core.SetupContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: config,
		})

		require.ErrorContains(t, err, "at least one label is required")
	})

	t.Run("blank labels -> error", func(t *testing.T) {
		config := removeLabelConfig()
		config["labels"] = []string{"  ", ""}

		err := component.Setup(core.SetupContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: config,
		})

		require.ErrorContains(t, err, "at least one label is required")
	})

	//
	// The canvas card reads the team off node metadata, so a successful setup has
	// to store it rather than leaving the card to fall back to the raw team id.
	//
	t.Run("stores the resolved team on the node", func(t *testing.T) {
		metadataContext := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Integration:   integrationWithTeam(),
			Metadata:      metadataContext,
			Configuration: removeLabelConfig(),
		})

		require.NoError(t, err)

		metadata, ok := metadataContext.Metadata.(NodeMetadata)
		require.True(t, ok)
		require.NotNil(t, metadata.Team)
		assert.Equal(t, "ENG", metadata.Team.Key)
	})
}

func Test__RemoveIssueLabel__Execute(t *testing.T) {
	component := RemoveIssueLabel{}

	t.Run("emits the updated issue", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{jsonResponse(issueWithLabelsResponse())},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: executionState,
			Configuration:  removeLabelConfig(),
		})

		require.NoError(t, err)
		assert.Equal(t, IssuePayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		issue, ok := wrapped["data"].(*Issue)
		require.True(t, ok)
		assert.Equal(t, "ENG-142", issue.Identifier)
		require.Len(t, issue.Labels, 1)
		assert.Equal(t, "backend", issue.Labels[0].Name)

		variables := variablesFromRequest(t, httpContext, 0)
		assert.Equal(t, "ENG-142", variables["id"])
		assert.Equal(t, "l1", variables["labelId"])
	})

	t.Run("removes each label in turn and dedupes", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(issueWithLabelsResponse()),
				jsonResponse(issueWithLabelsResponse()),
			},
		}

		config := removeLabelConfig()
		config["labels"] = []string{"l1", "l2", "l1"}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  config,
		})

		require.NoError(t, err)
		assert.Equal(t, "l1", variablesFromRequest(t, httpContext, 0)["labelId"])
		assert.Equal(t, "l2", variablesFromRequest(t, httpContext, 1)["labelId"])
	})

	//
	// The re-run case: Linear rejects removing a label the issue does not carry,
	// which must not fail the execution unless the user opted in.
	//
	t.Run("a label already off the issue is skipped", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(labelNotOnIssueResponse()),
				jsonResponse(`{"data":{"issue":{"id":"i1","identifier":"ENG-142","title":"Boom"}}}`),
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: executionState,
			Configuration:  removeLabelConfig(),
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)

		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		issue, ok := wrapped["data"].(*Issue)
		require.True(t, ok)
		assert.Equal(t, "ENG-142", issue.Identifier)
	})

	t.Run("failIfNotFound surfaces the error instead", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{jsonResponse(labelNotOnIssueResponse())},
		}

		config := removeLabelConfig()
		config["failIfNotFound"] = true

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  config,
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.ErrorContains(t, err, "Label not on issue")
	})

	//
	// Unlike GitHub's equivalent, an unrelated failure must still surface even
	// when failIfNotFound is off.
	//
	t.Run("unrelated errors are surfaced even when lenient", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{jsonResponse(`{"errors":[{"message":"Entity not found"}]}`)},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  removeLabelConfig(),
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.ErrorContains(t, err, "Entity not found")
	})

	t.Run("blank issue -> error", func(t *testing.T) {
		config := removeLabelConfig()
		config["issue"] = "   "

		err := component.Execute(core.ExecutionContext{
			HTTP:           &contexts.HTTPContext{},
			Integration:    newAuthorizedIntegration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  config,
		})

		require.ErrorContains(t, err, "issue is required")
	})
}
