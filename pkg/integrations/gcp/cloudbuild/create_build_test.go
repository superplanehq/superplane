package cloudbuild

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestCreateBuildBuildRequest(t *testing.T) {
	t.Run("uses source json when provided", func(t *testing.T) {
		build, err := buildRequest(CreateBuildConfiguration{
			Steps:  `[{"name":"golang:1.22","args":["test","./..."]}]`,
			Source: `{"gitSource":{"url":"https://github.com/org/repo.git","revision":"main"}}`,
			Images: []string{"us-central1-docker.pkg.dev/demo/app/image"},
		})

		require.NoError(t, err)
		source, ok := build["source"].(map[string]any)
		require.True(t, ok)
		gitSource, ok := source["gitSource"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "https://github.com/org/repo.git", gitSource["url"])
		assert.Equal(t, "main", gitSource["revision"])
		assert.Equal(t, []string{"us-central1-docker.pkg.dev/demo/app/image"}, build["images"])
	})

	t.Run("uses git source shortcut for repository urls", func(t *testing.T) {
		build, err := buildRequest(CreateBuildConfiguration{
			Steps:      `[{"name":"golang:1.22"}]`,
			RepoName:   "https://github.com/org/repo.git",
			BranchName: "main",
		})

		require.NoError(t, err)

		source, ok := build["source"].(map[string]any)
		require.True(t, ok)

		gitSource, ok := source["gitSource"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "https://github.com/org/repo.git", gitSource["url"])
		assert.Equal(t, "main", gitSource["revision"])
	})

	t.Run("normalizes repository urls without scheme", func(t *testing.T) {
		build, err := buildRequest(CreateBuildConfiguration{
			Steps:      `[{"name":"golang:1.22"}]`,
			RepoName:   "github.com/org/repo",
			BranchName: "main",
		})

		require.NoError(t, err)

		source, ok := build["source"].(map[string]any)
		require.True(t, ok)

		gitSource, ok := source["gitSource"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "https://github.com/org/repo", gitSource["url"])
		assert.Equal(t, "main", gitSource["revision"])
	})

	t.Run("uses repo source shortcut for cloud source repositories", func(t *testing.T) {
		build, err := buildRequest(CreateBuildConfiguration{
			Steps:      `[{"name":"golang:1.22"}]`,
			RepoName:   "my-cloud-source-repo",
			BranchName: "main",
		})

		require.NoError(t, err)

		source, ok := build["source"].(map[string]any)
		require.True(t, ok)

		repoSource, ok := source["repoSource"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "my-cloud-source-repo", repoSource["repoName"])
		assert.Equal(t, "main", repoSource["branchName"])
	})

	t.Run("uses connected repository shortcut", func(t *testing.T) {
		build, err := buildRequest(CreateBuildConfiguration{
			Steps:                 `[{"name":"golang:1.22"}]`,
			ConnectedRepository:   "projects/demo-project/locations/us-central1/connections/github-main/repositories/rtlbx",
			ConnectedRevisionType: createBuildConnectedRevisionBranch,
			ConnectedBranch:       "refs/heads/main",
		})

		require.NoError(t, err)

		source, ok := build["source"].(map[string]any)
		require.True(t, ok)

		connectedRepository, ok := source["connectedRepository"].(map[string]any)
		require.True(t, ok)
		assert.Equal(
			t,
			"projects/demo-project/locations/us-central1/connections/github-main/repositories/rtlbx",
			connectedRepository["repository"],
		)
		assert.Equal(t, "refs/heads/main", connectedRepository["revision"])
	})

	t.Run("rejects invalid repo source configuration", func(t *testing.T) {
		_, err := buildRequest(CreateBuildConfiguration{
			Steps:      `[{"name":"golang:1.22"}]`,
			RepoName:   "my-repo",
			BranchName: "main",
			TagName:    "v1.0.0",
		})

		require.ErrorContains(t, err, "mutually exclusive")
	})

	t.Run("uses commit sha revision shortcut", func(t *testing.T) {
		build, err := buildRequest(CreateBuildConfiguration{
			Steps:     `[{"name":"golang:1.22"}]`,
			RepoName:  "https://github.com/org/repo.git",
			CommitSHA: "5d7363a99d19e45830e1bc9622d2e4fa72d7229f",
		})

		require.NoError(t, err)

		source, ok := build["source"].(map[string]any)
		require.True(t, ok)

		gitSource, ok := source["gitSource"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "5d7363a99d19e45830e1bc9622d2e4fa72d7229f", gitSource["revision"])
	})
}

func TestCreateBuildSetup(t *testing.T) {
	component := &CreateBuild{}

	t.Run("creates subscription on first setup", func(t *testing.T) {
		integrationCtx := &testcontexts.IntegrationContext{}
		metadataCtx := &testcontexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"steps": `[{"name":"golang:1.22","args":["test","./..."]}]`,
			},
			Integration: integrationCtx,
			Metadata:    metadataCtx,
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 1)
		require.Len(t, integrationCtx.ActionRequests, 1)
		assert.Equal(t, gcpcommon.ActionNameEnsureCloudBuild, integrationCtx.ActionRequests[0].ActionName)

		metadata := CreateBuildNodeMetadata{}
		require.NoError(t, mapstructure.Decode(metadataCtx.Get(), &metadata))
		assert.NotEmpty(t, metadata.SubscriptionID)
	})

	t.Run("ensures subscription exists even when metadata already has a subscription id", func(t *testing.T) {
		integrationCtx := &testcontexts.IntegrationContext{
			Metadata: gcpcommon.Metadata{
				ProjectID:              "demo-project",
				CloudBuildSubscription: "sp-cb-sub-existing",
			},
		}
		metadataCtx := &testcontexts.MetadataContext{
			Metadata: CreateBuildNodeMetadata{SubscriptionID: "existing-id"},
		}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"steps": `[{"name":"golang:1.22","args":["test","./..."]}]`,
			},
			Integration: integrationCtx,
			Metadata:    metadataCtx,
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 1)
		require.Len(t, integrationCtx.ActionRequests, 1)
		assert.Equal(t, gcpcommon.ActionNameEnsureCloudBuild, integrationCtx.ActionRequests[0].ActionName)

		metadata := CreateBuildNodeMetadata{}
		require.NoError(t, mapstructure.Decode(metadataCtx.Get(), &metadata))
		assert.NotEmpty(t, metadata.SubscriptionID)
	})
}

func TestCreateBuildExecuteSchedulesPolling(t *testing.T) {
	component := &CreateBuild{}
	client := &mockClient{
		projectID: "demo-project",
		postURL: func(_ context.Context, fullURL string, body any) ([]byte, error) {
			assert.Equal(t, "https://cloudbuild.googleapis.com/v1/projects/demo-project/builds", fullURL)

			request, ok := body.(map[string]any)
			require.True(t, ok)
			assert.Equal(t, []string{"us-central1-docker.pkg.dev/demo/app/image"}, request["images"])

			return []byte(`{
				"metadata": {
					"build": {
						"id": "build-123",
						"name": "projects/demo-project/locations/global/builds/build-123",
						"projectId": "demo-project",
						"status": "WORKING",
						"logUrl": "https://console.cloud.google.com/cloud-build/builds/build-123"
					}
				}
			}`), nil
		},
	}

	setTestClientFactory(t, func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
		return client, nil
	})

	metadataCtx := &testcontexts.MetadataContext{}
	executionStateCtx := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
	requestCtx := &testcontexts.RequestContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"steps":  `[{"name":"golang:1.22","args":["test","./..."]}]`,
			"images": []string{"us-central1-docker.pkg.dev/demo/app/image"},
			"source": `{"gitSource":{"url":"https://github.com/org/repo.git","revision":"main"}}`,
		},
		Integration:    &testcontexts.IntegrationContext{},
		Metadata:       metadataCtx,
		ExecutionState: executionStateCtx,
		Requests:       requestCtx,
	})

	require.NoError(t, err)
	assert.Equal(t, createBuildPollAction, requestCtx.Action)
	assert.Equal(t, "build-123", executionStateCtx.KVs[createBuildExecutionKV])

	metadata := CreateBuildExecutionMetadata{}
	require.NoError(t, mapstructure.Decode(metadataCtx.Get(), &metadata))
	assert.Equal(t, "WORKING", metadata.Build["status"])
	assert.Equal(t, "demo-project", metadata.Build["projectId"])
}

func TestCreateBuildExecuteUsesRegionalEndpointForConnectedRepositories(t *testing.T) {
	component := &CreateBuild{}
	client := &mockClient{
		projectID: "integration-project",
		postURL: func(_ context.Context, fullURL string, body any) ([]byte, error) {
			assert.Equal(
				t,
				"https://cloudbuild.googleapis.com/v1/projects/demo-project/locations/us-central1/builds",
				fullURL,
			)

			request, ok := body.(map[string]any)
			require.True(t, ok)
			source, ok := request["source"].(map[string]any)
			require.True(t, ok)
			connectedRepository, ok := source["connectedRepository"].(map[string]any)
			require.True(t, ok)
			assert.Equal(
				t,
				"projects/demo-project/locations/us-central1/connections/github-main/repositories/rtlbx",
				connectedRepository["repository"],
			)

			return []byte(`{
				"metadata": {
					"build": {
						"id": "build-123",
						"name": "projects/demo-project/locations/us-central1/builds/build-123",
						"projectId": "demo-project",
						"status": "WORKING"
					}
				}
			}`), nil
		},
	}

	setTestClientFactory(t, func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
		return client, nil
	})

	metadataCtx := &testcontexts.MetadataContext{}
	executionStateCtx := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
	requestCtx := &testcontexts.RequestContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"steps":                  `[{"name":"golang:1.22","args":["test","./..."]}]`,
			"useConnectedRepository": true,
			"connectionLocation":     "us-central1",
			"connection":             "projects/demo-project/locations/us-central1/connections/github-main",
			"connectedRepository":    "projects/demo-project/locations/us-central1/connections/github-main/repositories/rtlbx",
			"connectedRevisionType":  createBuildConnectedRevisionBranch,
			"connectedBranch":        "refs/heads/main",
		},
		Integration:    &testcontexts.IntegrationContext{},
		Metadata:       metadataCtx,
		ExecutionState: executionStateCtx,
		Requests:       requestCtx,
	})

	require.NoError(t, err)
	assert.Equal(t, createBuildPollAction, requestCtx.Action)
	assert.Equal(t, "build-123", executionStateCtx.KVs[createBuildExecutionKV])
}

func TestCreateBuildPollSuccess(t *testing.T) {
	component := &CreateBuild{}
	client := &mockClient{
		projectID: "demo-project",
		getURL: func(_ context.Context, fullURL string) ([]byte, error) {
			assert.Equal(t, "https://cloudbuild.googleapis.com/v1/projects/demo-project/builds/build-123", fullURL)
			return []byte(`{
				"id": "build-123",
				"name": "projects/demo-project/locations/global/builds/build-123",
				"projectId": "demo-project",
				"status": "SUCCESS",
				"logUrl": "https://console.cloud.google.com/cloud-build/builds/build-123"
			}`), nil
		},
	}

	setTestClientFactory(t, func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
		return client, nil
	})

	metadataCtx := &testcontexts.MetadataContext{Metadata: CreateBuildExecutionMetadata{
		Build: map[string]any{
			"id":        "build-123",
			"projectId": "demo-project",
			"status":    "WORKING",
		},
	}}
	executionStateCtx := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}

	err := component.HandleAction(core.ActionContext{
		Name:           createBuildPollAction,
		Configuration:  map[string]any{"steps": `[{"name":"golang:1.22"}]`},
		Metadata:       metadataCtx,
		ExecutionState: executionStateCtx,
		Integration:    &testcontexts.IntegrationContext{},
		Requests:       &testcontexts.RequestContext{},
	})

	require.NoError(t, err)
	assert.True(t, executionStateCtx.Passed)
	assert.Equal(t, createBuildPassedOutputChannel, executionStateCtx.Channel)
	assert.Equal(t, createBuildPayloadType, executionStateCtx.Type)
}

func TestCreateBuildPollFailureEmitsFailedChannel(t *testing.T) {
	component := &CreateBuild{}
	client := &mockClient{
		projectID: "demo-project",
		getURL: func(_ context.Context, fullURL string) ([]byte, error) {
			assert.Equal(t, "https://cloudbuild.googleapis.com/v1/projects/demo-project/builds/build-123", fullURL)
			return []byte(`{
				"id": "build-123",
				"name": "projects/demo-project/locations/global/builds/build-123",
				"projectId": "demo-project",
				"status": "FAILURE",
				"logUrl": "https://console.cloud.google.com/cloud-build/builds/build-123"
			}`), nil
		},
	}

	setTestClientFactory(t, func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
		return client, nil
	})

	metadataCtx := &testcontexts.MetadataContext{Metadata: CreateBuildExecutionMetadata{
		Build: map[string]any{
			"id":        "build-123",
			"projectId": "demo-project",
			"status":    "WORKING",
		},
	}}
	executionStateCtx := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}

	err := component.HandleAction(core.ActionContext{
		Name:           createBuildPollAction,
		Configuration:  map[string]any{"steps": `[{"name":"golang:1.22"}]`},
		Metadata:       metadataCtx,
		ExecutionState: executionStateCtx,
		Integration:    &testcontexts.IntegrationContext{},
		Requests:       &testcontexts.RequestContext{},
	})

	require.NoError(t, err)
	assert.True(t, executionStateCtx.Passed)
	assert.Equal(t, createBuildFailedOutputChannel, executionStateCtx.Channel)
	assert.Equal(t, createBuildPayloadType, executionStateCtx.Type)
}

func TestCreateBuildOnIntegrationMessageEmitsFailedChannel(t *testing.T) {
	component := &CreateBuild{}
	executionMetadataCtx := &testcontexts.MetadataContext{}
	executionStateCtx := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}

	err := component.OnIntegrationMessage(core.IntegrationMessageContext{
		Message: map[string]any{
			"id":        "build-123",
			"projectId": "demo-project",
			"status":    "FAILURE",
			"logUrl":    "https://console.cloud.google.com/cloud-build/builds/build-123",
		},
		FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
			assert.Equal(t, createBuildExecutionKV, key)
			assert.Equal(t, "build-123", value)
			return &core.ExecutionContext{
				Metadata:       executionMetadataCtx,
				ExecutionState: executionStateCtx,
			}, nil
		},
	})

	require.NoError(t, err)
	assert.True(t, executionStateCtx.Passed)
	assert.Equal(t, createBuildFailedOutputChannel, executionStateCtx.Channel)
	assert.Equal(t, createBuildPayloadType, executionStateCtx.Type)

	metadata := CreateBuildExecutionMetadata{}
	require.NoError(t, mapstructure.Decode(executionMetadataCtx.Get(), &metadata))
	assert.Equal(t, "FAILURE", metadata.Build["status"])
}

func TestCreateBuildCancelPostsCancelRequest(t *testing.T) {
	component := &CreateBuild{}
	client := &mockClient{
		projectID: "demo-project",
		postURL: func(_ context.Context, fullURL string, body any) ([]byte, error) {
			assert.Equal(t, "https://cloudbuild.googleapis.com/v1/projects/demo-project/builds/build-123:cancel", fullURL)

			payload, ok := body.(map[string]any)
			require.True(t, ok)
			assert.Empty(t, payload)

			return []byte(`{}`), nil
		},
	}

	setTestClientFactory(t, func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
		return client, nil
	})

	metadataCtx := &testcontexts.MetadataContext{Metadata: CreateBuildExecutionMetadata{
		Build: map[string]any{
			"id":        "build-123",
			"projectId": "demo-project",
			"status":    "WORKING",
		},
	}}

	err := component.Cancel(core.ExecutionContext{
		Metadata:    metadataCtx,
		Integration: &testcontexts.IntegrationContext{},
	})

	require.NoError(t, err)

	metadata := CreateBuildExecutionMetadata{}
	require.NoError(t, mapstructure.Decode(metadataCtx.Get(), &metadata))
	assert.Equal(t, "CANCELLED", metadata.Build["status"])
}

func TestCreateBuildCancelUsesRegionalEndpointWhenBuildIsRegional(t *testing.T) {
	component := &CreateBuild{}
	client := &mockClient{
		projectID: "demo-project",
		postURL: func(_ context.Context, fullURL string, body any) ([]byte, error) {
			assert.Equal(
				t,
				"https://cloudbuild.googleapis.com/v1/projects/demo-project/locations/us-central1/builds/build-123:cancel",
				fullURL,
			)

			payload, ok := body.(map[string]any)
			require.True(t, ok)
			assert.Empty(t, payload)

			return []byte(`{}`), nil
		},
	}

	setTestClientFactory(t, func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
		return client, nil
	})

	metadataCtx := &testcontexts.MetadataContext{Metadata: CreateBuildExecutionMetadata{
		Build: map[string]any{
			"id":        "build-123",
			"name":      "projects/demo-project/locations/us-central1/builds/build-123",
			"projectId": "demo-project",
			"status":    "WORKING",
		},
	}}

	err := component.Cancel(core.ExecutionContext{
		Metadata:    metadataCtx,
		Integration: &testcontexts.IntegrationContext{},
	})

	require.NoError(t, err)

	metadata := CreateBuildExecutionMetadata{}
	require.NoError(t, mapstructure.Decode(metadataCtx.Get(), &metadata))
	assert.Equal(t, "CANCELLED", metadata.Build["status"])
}

func TestCreateBuildCancelFullGlobalResourceNameIgnoresProjectOverride(t *testing.T) {
	component := &CreateBuild{}
	client := &mockClient{
		projectID: "integration-project",
		postURL: func(_ context.Context, fullURL string, body any) ([]byte, error) {
			assert.Equal(
				t,
				"https://cloudbuild.googleapis.com/v1/projects/demo-project/builds/build-123:cancel",
				fullURL,
			)

			payload, ok := body.(map[string]any)
			require.True(t, ok)
			assert.Empty(t, payload)

			return []byte(`{}`), nil
		},
	}

	setTestClientFactory(t, func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
		return client, nil
	})

	metadataCtx := &testcontexts.MetadataContext{Metadata: CreateBuildExecutionMetadata{
		Build: map[string]any{
			"id":        "build-123",
			"name":      "projects/demo-project/locations/global/builds/build-123",
			"projectId": "demo-project",
			"status":    "WORKING",
		},
	}}

	err := component.Cancel(core.ExecutionContext{
		Configuration: map[string]any{
			"projectId": "other-project",
		},
		Metadata:    metadataCtx,
		Integration: &testcontexts.IntegrationContext{},
	})

	require.NoError(t, err)

	metadata := CreateBuildExecutionMetadata{}
	require.NoError(t, mapstructure.Decode(metadataCtx.Get(), &metadata))
	assert.Equal(t, "CANCELLED", metadata.Build["status"])
}

func TestCreateBuildCancelPropagatesCancelRequestError(t *testing.T) {
	component := &CreateBuild{}
	setTestClientFactory(t, func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
		return &mockClient{
			projectID: "demo-project",
			postURL: func(_ context.Context, _ string, _ any) ([]byte, error) {
				return nil, errors.New("boom")
			},
		}, nil
	})

	err := component.Cancel(core.ExecutionContext{
		Metadata: &testcontexts.MetadataContext{Metadata: CreateBuildExecutionMetadata{
			Build: map[string]any{
				"id":        "build-123",
				"projectId": "demo-project",
				"status":    "WORKING",
			},
		}},
		Integration: &testcontexts.IntegrationContext{},
	})

	require.ErrorContains(t, err, "cancel Cloud Build build build-123")
	require.ErrorContains(t, err, "boom")
}

func TestCreateBuildCancelPropagatesMetadataStoreError(t *testing.T) {
	component := &CreateBuild{}
	setTestClientFactory(t, func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
		return &mockClient{
			projectID: "demo-project",
			postURL: func(_ context.Context, _ string, _ any) ([]byte, error) {
				return []byte(`{}`), nil
			},
		}, nil
	})

	err := component.Cancel(core.ExecutionContext{
		Metadata: &failingMetadataContext{
			metadata: CreateBuildExecutionMetadata{
				Build: map[string]any{
					"id":        "build-123",
					"projectId": "demo-project",
					"status":    "WORKING",
				},
			},
			err: errors.New("set failed"),
		},
		Integration: &testcontexts.IntegrationContext{},
	})

	require.ErrorContains(t, err, "store cancelled build metadata")
	require.ErrorContains(t, err, "set failed")
}

func TestDecodeCreateBuildConfigurationRejectsConflictingSourceFields(t *testing.T) {
	_, err := decodeCreateBuildConfiguration(map[string]any{
		"steps":    `[{"name":"golang:1.22"}]`,
		"source":   `{"gitSource":{"url":"https://github.com/org/repo.git"}}`,
		"repoName": "my-repo",
	})

	require.ErrorContains(t, err, "cannot be combined")
}

func TestDecodeCreateBuildConfigurationRejectsMultipleRevisionShortcuts(t *testing.T) {
	_, err := decodeCreateBuildConfiguration(map[string]any{
		"steps":      `[{"name":"golang:1.22"}]`,
		"repoName":   "https://github.com/org/repo.git",
		"branchName": "main",
		"commitSha":  "5d7363a99d19e45830e1bc9622d2e4fa72d7229f",
	})

	require.ErrorContains(t, err, "mutually exclusive")
}

func TestDecodeCreateBuildConfigurationRejectsMixedConnectedRepositoryAndManualSource(t *testing.T) {
	_, err := decodeCreateBuildConfiguration(map[string]any{
		"steps":                  `[{"name":"golang:1.22"}]`,
		"useConnectedRepository": true,
		"connectionLocation":     "us-central1",
		"connection":             "projects/demo-project/locations/us-central1/connections/github-main",
		"connectedRepository":    "projects/demo-project/locations/us-central1/connections/github-main/repositories/rtlbx",
		"connectedRevisionType":  createBuildConnectedRevisionBranch,
		"connectedBranch":        "refs/heads/main",
		"repoName":               "https://github.com/org/repo.git",
	})

	require.ErrorContains(t, err, "cannot be combined")
}

func TestDecodeCreateBuildConfigurationRejectsConnectedRepositoryMismatch(t *testing.T) {
	_, err := decodeCreateBuildConfiguration(map[string]any{
		"steps":                  `[{"name":"golang:1.22"}]`,
		"useConnectedRepository": true,
		"connectionLocation":     "europe-west1",
		"connection":             "projects/demo-project/locations/europe-west1/connections/github-main",
		"connectedRepository":    "projects/demo-project/locations/us-central1/connections/github-main/repositories/rtlbx",
		"connectedRevisionType":  createBuildConnectedRevisionBranch,
		"connectedBranch":        "refs/heads/main",
	})

	require.ErrorContains(t, err, "connectionLocation must match")
}

func TestCreateBuildFailureMetadataStaysJSONCompatible(t *testing.T) {
	build := map[string]any{
		"id":        "build-123",
		"status":    "FAILURE",
		"projectId": "demo-project",
	}

	metadataCtx := &testcontexts.MetadataContext{}
	require.NoError(t, storeCreateBuildMetadata(metadataCtx, build, "demo-project"))

	encoded, err := json.Marshal(metadataCtx.Get())
	require.NoError(t, err)
	assert.Contains(t, string(encoded), "build-123")
}

type failingMetadataContext struct {
	metadata any
	err      error
}

func (m *failingMetadataContext) Get() any {
	return m.metadata
}

func (m *failingMetadataContext) Set(_ any) error {
	return m.err
}
