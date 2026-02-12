package codeartifact

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestCreateRepository_Setup(t *testing.T) {
	component := &CreateRepository{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: "invalid",
		})
		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"domain": "my-domain", "repository": "my-repo"},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing domain -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"region": "us-east-1", "repository": "my-repo"},
		})
		require.ErrorContains(t, err, "domain is required")
	})

	t.Run("missing repository -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"region": "us-east-1", "domain": "my-domain"},
		})
		require.ErrorContains(t, err, "repository name is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"region":      "us-east-1",
				"domain":      "my-domain",
				"repository":  "my-repo",
				"description": "optional desc",
			},
		})
		require.NoError(t, err)
	})
}

func TestCreateRepository_Execute(t *testing.T) {
	component := &CreateRepository{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration:  "invalid",
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Integration:    &contexts.IntegrationContext{},
		})
		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing credentials -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region": "us-east-1", "domain": "my-domain", "repository": "my-repo",
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Integration:    &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{}},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "credentials")
	})

	t.Run("success -> emits repository", func(t *testing.T) {
		repoResp := map[string]any{
			"repository": map[string]any{
				"arn":  "arn:aws:codeartifact:us-east-1:123:repository/my-domain/my-repo",
				"name": "my-repo", "domainName": "my-domain", "domainOwner": "123",
			},
		}
		body, _ := json.Marshal(repoResp)
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(string(body))),
			}},
		}
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"region": "us-east-1", "domain": "my-domain", "repository": "my-repo"},
			ExecutionState: execState,
			HTTP:           httpContext,
			Integration: &contexts.IntegrationContext{
				Secrets: map[string]core.IntegrationSecret{
					"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
					"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
					"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
				},
			},
		})
		require.NoError(t, err)
		require.Len(t, execState.Payloads, 1)
		require.True(t, execState.Passed)
		payload := execState.Payloads[0].(map[string]any)
		require.Equal(t, "aws.codeartifact.repository", execState.Type)
		data := payload["data"].(map[string]any)
		repo, ok := data["repository"].(*RepositoryDescription)
		require.True(t, ok)
		require.Equal(t, "my-repo", repo.Name)
		require.Equal(t, "my-domain", repo.DomainName)
	})
}
