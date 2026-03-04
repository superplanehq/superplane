package cloudbuild

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestGetBuildSetupRejectsMissingBuildID(t *testing.T) {
	component := &GetBuild{}

	err := component.Setup(core.SetupContext{
		Configuration: map[string]any{},
		Integration:   &testcontexts.IntegrationContext{},
		Metadata:      &testcontexts.MetadataContext{},
	})

	require.ErrorContains(t, err, "buildId is required")
}

func TestGetBuildExecuteUsesGlobalEndpointForPlainBuildID(t *testing.T) {
	component := &GetBuild{}
	client := &mockClient{
		projectID: "demo-project",
		getURL: func(_ context.Context, fullURL string) ([]byte, error) {
			assert.Equal(
				t,
				"https://cloudbuild.googleapis.com/v1/projects/demo-project/builds/build-123",
				fullURL,
			)
			return []byte(`{"id":"build-123","status":"SUCCESS"}`), nil
		},
	}

	setTestClientFactory(t, func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
		return client, nil
	})

	executionState := &testcontexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"buildId": "build-123",
		},
		Integration:    &testcontexts.IntegrationContext{},
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	assert.True(t, executionState.Passed)
	assert.Equal(t, getBuildOutputChannel, executionState.Channel)
}

func TestGetBuildExecuteUsesFullResourceName(t *testing.T) {
	component := &GetBuild{}
	client := &mockClient{
		projectID: "demo-project",
		getURL: func(_ context.Context, fullURL string) ([]byte, error) {
			assert.Equal(
				t,
				"https://cloudbuild.googleapis.com/v1/projects/demo-project/locations/us-central1/builds/build-123",
				fullURL,
			)
			return []byte(`{"id":"build-123","status":"SUCCESS"}`), nil
		},
	}

	setTestClientFactory(t, func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
		return client, nil
	})

	executionState := &testcontexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"buildId": "projects/demo-project/locations/us-central1/builds/build-123",
		},
		Integration:    &testcontexts.IntegrationContext{},
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	assert.True(t, executionState.Passed)
	assert.Equal(t, getBuildOutputChannel, executionState.Channel)
}

func TestGetBuildExecuteFullGlobalResourceNameIgnoresProjectOverride(t *testing.T) {
	component := &GetBuild{}
	client := &mockClient{
		projectID: "integration-project",
		getURL: func(_ context.Context, fullURL string) ([]byte, error) {
			assert.Equal(
				t,
				"https://cloudbuild.googleapis.com/v1/projects/demo-project/builds/build-123",
				fullURL,
			)
			return []byte(`{"id":"build-123","status":"SUCCESS"}`), nil
		},
	}

	setTestClientFactory(t, func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
		return client, nil
	})

	executionState := &testcontexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"buildId":   "projects/demo-project/locations/global/builds/build-123",
			"projectId": "other-project",
		},
		Integration:    &testcontexts.IntegrationContext{},
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	assert.True(t, executionState.Passed)
	assert.Equal(t, getBuildOutputChannel, executionState.Channel)
}
