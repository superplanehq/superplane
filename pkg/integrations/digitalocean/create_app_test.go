package digitalocean

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateApp__Setup(t *testing.T) {
	component := &CreateApp{}

	t.Run("missing name returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":         "nyc",
				"componentType":  "service",
				"sourceProvider": "github",
				"gitHubRepo":     "owner/repo",
			},
		})

		require.ErrorContains(t, err, "name is required")
	})

	t.Run("missing region returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":           "my-app",
				"componentType":  "service",
				"sourceProvider": "github",
				"gitHubRepo":     "owner/repo",
			},
		})

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing componentType returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":           "my-app",
				"region":         "nyc",
				"sourceProvider": "github",
				"gitHubRepo":     "owner/repo",
			},
		})

		require.ErrorContains(t, err, "componentType is required")
	})

	t.Run("unsupported component type returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":           "my-app",
				"region":         "nyc",
				"componentType":  "function",
				"sourceProvider": "github",
				"gitHubRepo":     "owner/repo",
			},
		})

		require.ErrorContains(t, err, "unsupported component type: function")
	})

	t.Run("missing sourceProvider returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":          "my-app",
				"region":        "nyc",
				"componentType": "service",
				"gitHubRepo":    "owner/repo",
			},
		})

		require.ErrorContains(t, err, "sourceProvider is required")
	})

	t.Run("unsupported source provider returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":           "my-app",
				"region":         "nyc",
				"componentType":  "service",
				"sourceProvider": "unknown",
			},
		})

		require.ErrorContains(t, err, "unsupported source provider: unknown")
	})

	t.Run("github provider without repo returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":           "my-app",
				"region":         "nyc",
				"componentType":  "service",
				"sourceProvider": "github",
			},
		})

		require.ErrorContains(t, err, "gitHubRepo is required when using GitHub as source provider")
	})

	t.Run("gitlab provider without repo returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":           "my-app",
				"region":         "nyc",
				"componentType":  "service",
				"sourceProvider": "gitlab",
			},
		})

		require.ErrorContains(t, err, "gitLabRepo is required when using GitLab as source provider")
	})

	t.Run("bitbucket provider without repo returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":           "my-app",
				"region":         "nyc",
				"componentType":  "service",
				"sourceProvider": "bitbucket",
			},
		})

		require.ErrorContains(t, err, "bitbucketRepo is required when using Bitbucket as source provider")
	})

	t.Run("valid github service configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":           "my-app",
				"region":         "nyc",
				"componentType":  "service",
				"sourceProvider": "github",
				"gitHubRepo":     "owner/repo",
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid static-site configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":           "my-site",
				"region":         "nyc",
				"componentType":  "static-site",
				"sourceProvider": "github",
				"gitHubRepo":     "owner/repo",
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid worker configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":           "my-worker",
				"region":         "nyc",
				"componentType":  "worker",
				"sourceProvider": "gitlab",
				"gitLabRepo":     "owner/repo",
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid job configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":           "my-job",
				"region":         "nyc",
				"componentType":  "job",
				"sourceProvider": "bitbucket",
				"bitbucketRepo":  "owner/repo",
			},
		})

		require.NoError(t, err)
	})

	t.Run("database enabled without name returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":           "my-app",
				"region":         "nyc",
				"componentType":  "service",
				"sourceProvider": "github",
				"gitHubRepo":     "owner/repo",
				"addDatabase":    true,
				"databaseEngine": "PG",
			},
		})

		require.ErrorContains(t, err, "databaseName is required when addDatabase is enabled")
	})

	t.Run("database enabled without engine returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":           "my-app",
				"region":         "nyc",
				"componentType":  "service",
				"sourceProvider": "github",
				"gitHubRepo":     "owner/repo",
				"addDatabase":    true,
				"databaseName":   "db",
			},
		})

		require.ErrorContains(t, err, "databaseEngine is required when addDatabase is enabled")
	})

	t.Run("database with unsupported engine returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":           "my-app",
				"region":         "nyc",
				"componentType":  "service",
				"sourceProvider": "github",
				"gitHubRepo":     "owner/repo",
				"addDatabase":    true,
				"databaseName":   "db",
				"databaseEngine": "ORACLE",
			},
		})

		require.ErrorContains(t, err, "unsupported database engine: ORACLE")
	})

	t.Run("managed database without cluster name returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":               "my-app",
				"region":             "nyc",
				"componentType":      "service",
				"sourceProvider":     "github",
				"gitHubRepo":         "owner/repo",
				"addDatabase":        true,
				"databaseName":       "db",
				"databaseEngine":     "PG",
				"databaseProduction": true,
			},
		})

		require.ErrorContains(t, err, "databaseClusterName is required when using a managed database")
	})

	t.Run("valid dev database configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":           "my-app",
				"region":         "nyc",
				"componentType":  "service",
				"sourceProvider": "github",
				"gitHubRepo":     "owner/repo",
				"addDatabase":    true,
				"databaseName":   "db",
				"databaseEngine": "PG",
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid managed database configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":                "my-app",
				"region":              "nyc",
				"componentType":       "service",
				"sourceProvider":      "github",
				"gitHubRepo":          "owner/repo",
				"addDatabase":         true,
				"databaseName":        "db",
				"databaseEngine":      "PG",
				"databaseProduction":  true,
				"databaseClusterName": "my-cluster",
			},
		})

		require.NoError(t, err)
	})

	t.Run("expression name is accepted at setup time", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":           "{{ $.trigger.data.appName }}",
				"region":         "nyc",
				"componentType":  "service",
				"sourceProvider": "github",
				"gitHubRepo":     "owner/repo",
			},
		})

		require.NoError(t, err)
	})
}

func Test__CreateApp__Execute(t *testing.T) {
	component := &CreateApp{}

	t.Run("github service -> creates app with github source and schedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"app": {
							"id": "b6bdf840-2854-4f87-a9f6-6a0c4dbf3a48",
							"pending_deployment": {"id": "dep-001"},
							"spec": {"name": "my-app", "region": "nyc"},
							"region": {"slug": "nyc"}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":           "my-app",
				"region":         "nyc",
				"componentType":  "service",
				"sourceProvider": "github",
				"gitHubRepo":     "owner/repo",
				"gitHubBranch":   "develop",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		// Should store metadata and schedule poll
		assert.False(t, executionState.Passed)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, appPollInterval, requestCtx.Duration)

		metadata, ok := metadataCtx.Metadata.(appDeploymentMetadata)
		require.True(t, ok)
		assert.Equal(t, "b6bdf840-2854-4f87-a9f6-6a0c4dbf3a48", metadata.AppID)
		assert.Equal(t, "dep-001", metadata.DeploymentID)
	})

	t.Run("github service with all options -> creates app and schedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"app": {
							"id": "full-config-app-id",
							"pending_deployment": {"id": "dep-full"},
							"spec": {"name": "full-app", "region": "nyc"},
							"region": {"slug": "nyc"}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":             "full-app",
				"region":           "nyc",
				"componentType":    "service",
				"sourceProvider":   "github",
				"gitHubRepo":       "owner/repo",
				"gitHubBranch":     "main",
				"environmentSlug":  "node-js",
				"buildCommand":     "npm install && npm run build",
				"runCommand":       "npm start",
				"sourceDir":        "/app",
				"httpPort":         int64(8080),
				"instanceSizeSlug": "apps-s-1vcpu-1gb",
				"instanceCount":    int64(2),
				"envVars":          []string{"NODE_ENV=production"},
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		// Should store metadata and schedule poll
		assert.False(t, executionState.Passed)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, appPollInterval, requestCtx.Duration)
	})

	t.Run("gitlab source -> creates app with gitlab source and schedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"app": {
							"id": "c7cef951-3965-5g98-b0g7-7b1d9d5b7d59",
							"pending_deployment": {"id": "dep-gitlab"},
							"spec": {"name": "my-gitlab-app", "region": "ams"},
							"region": {"slug": "ams"}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":           "my-gitlab-app",
				"region":         "ams",
				"componentType":  "service",
				"sourceProvider": "gitlab",
				"gitLabRepo":     "group/project",
				"gitLabBranch":   "main",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		// Should store metadata and schedule poll
		assert.False(t, executionState.Passed)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, appPollInterval, requestCtx.Duration)
	})

	t.Run("bitbucket source -> creates app with bitbucket source and schedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"app": {
							"id": "d8dfg062-4076-6h09-c1h8-8c2eae6c8e60",
							"pending_deployment": {"id": "dep-bb"},
							"spec": {"name": "my-bb-app", "region": "sfo"},
							"region": {"slug": "sfo"}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":            "my-bb-app",
				"region":          "sfo",
				"componentType":   "service",
				"sourceProvider":  "bitbucket",
				"bitbucketRepo":   "team/project",
				"bitbucketBranch": "release",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		// Should store metadata and schedule poll
		assert.False(t, executionState.Passed)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, appPollInterval, requestCtx.Duration)
	})

	t.Run("static-site -> creates app with static site component", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"app": {
							"id": "static-site-app-id",
							"pending_deployment": {"id": "dep-static"},
							"spec": {"name": "my-site", "region": "nyc"},
							"region": {"slug": "nyc"}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":             "my-site",
				"region":           "nyc",
				"componentType":    "static-site",
				"sourceProvider":   "github",
				"gitHubRepo":       "owner/website",
				"gitHubBranch":     "main",
				"environmentSlug":  "html",
				"buildCommand":     "npm run build",
				"sourceDir":        "/frontend",
				"outputDir":        "dist",
				"indexDocument":    "index.html",
				"errorDocument":    "404.html",
				"catchallDocument": "index.html",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		// Should store metadata and schedule poll
		assert.False(t, executionState.Passed)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, appPollInterval, requestCtx.Duration)
	})

	t.Run("worker -> creates app with worker component", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"app": {
							"id": "worker-app-id",
							"pending_deployment": {"id": "dep-worker"},
							"spec": {"name": "my-worker", "region": "ams"},
							"region": {"slug": "ams"}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":             "my-worker",
				"region":           "ams",
				"componentType":    "worker",
				"sourceProvider":   "gitlab",
				"gitLabRepo":       "group/worker-project",
				"environmentSlug":  "python",
				"runCommand":       "python worker.py",
				"instanceSizeSlug": "apps-s-1vcpu-1gb",
				"instanceCount":    int64(3),
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		// Should store metadata and schedule poll
		assert.False(t, executionState.Passed)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, appPollInterval, requestCtx.Duration)
	})

	t.Run("job -> creates app with job component", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"app": {
							"id": "job-app-id",
							"pending_deployment": {"id": "dep-job"},
							"spec": {"name": "my-job", "region": "sfo"},
							"region": {"slug": "sfo"}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":            "my-job",
				"region":          "sfo",
				"componentType":   "job",
				"sourceProvider":  "bitbucket",
				"bitbucketRepo":   "team/migration-scripts",
				"runCommand":      "python migrate.py",
				"environmentSlug": "python",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		// Should store metadata and schedule poll
		assert.False(t, executionState.Passed)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, appPollInterval, requestCtx.Duration)
	})

	t.Run("github source with default branch", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"app": {
							"id": "test-app-id",
							"pending_deployment": {"id": "dep-default-branch"},
							"spec": {"name": "my-app", "region": "nyc"},
							"region": {"slug": "nyc"}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":           "my-app",
				"region":         "nyc",
				"componentType":  "service",
				"sourceProvider": "github",
				"gitHubRepo":     "owner/repo",
				// No branch specified - should default to "main"
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		// Should store metadata and schedule poll
		assert.False(t, executionState.Passed)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, appPollInterval, requestCtx.Duration)
	})

	t.Run("default component type is service", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"app": {
							"id": "default-type-id",
							"pending_deployment": {"id": "dep-default-type"},
							"spec": {"name": "my-app", "region": "nyc"},
							"region": {"slug": "nyc"}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":           "my-app",
				"region":         "nyc",
				"sourceProvider": "github",
				"gitHubRepo":     "owner/repo",
				// No componentType - should default to "service"
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		// Should store metadata and schedule poll
		assert.False(t, executionState.Passed)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, appPollInterval, requestCtx.Duration)
	})

	t.Run("deploy on push disabled -> creates app with deploy_on_push false", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"app": {
							"id": "deploy-push-off-id",
							"pending_deployment": {"id": "dep-push-off"},
							"spec": {"name": "my-app", "region": "nyc"},
							"region": {"slug": "nyc"}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":           "my-app",
				"region":         "nyc",
				"componentType":  "service",
				"sourceProvider": "github",
				"gitHubRepo":     "owner/repo",
				"deployOnPush":   false,
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		// Should store metadata and schedule poll
		assert.False(t, executionState.Passed)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, appPollInterval, requestCtx.Duration)
	})

	t.Run("with dev database -> creates app with database component", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"app": {
							"id": "db-app-id",
							"pending_deployment": {"id": "dep-db"},
							"spec": {"name": "my-app", "region": "nyc"},
							"region": {"slug": "nyc"}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":            "my-app",
				"region":          "nyc",
				"componentType":   "service",
				"sourceProvider":  "github",
				"gitHubRepo":      "owner/repo",
				"addDatabase":     true,
				"databaseName":    "db",
				"databaseEngine":  "PG",
				"databaseVersion": "16",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		// Should store metadata and schedule poll
		assert.False(t, executionState.Passed)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, appPollInterval, requestCtx.Duration)
	})

	t.Run("with managed database -> creates app with production database", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"app": {
							"id": "managed-db-app-id",
							"pending_deployment": {"id": "dep-managed-db"},
							"spec": {"name": "my-app", "region": "nyc"},
							"region": {"slug": "nyc"}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":                "my-app",
				"region":              "nyc",
				"componentType":       "service",
				"sourceProvider":      "github",
				"gitHubRepo":          "owner/repo",
				"addDatabase":         true,
				"databaseName":        "db",
				"databaseEngine":      "PG",
				"databaseVersion":     "16",
				"databaseProduction":  true,
				"databaseClusterName": "my-cluster",
				"databaseDBName":      "mydb",
				"databaseDBUser":      "app_user",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		// Should store metadata and schedule poll
		assert.False(t, executionState.Passed)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, appPollInterval, requestCtx.Duration)
	})

	t.Run("with ingress path and CORS -> creates app with ingress rules", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"app": {
							"id": "ingress-app-id",
							"pending_deployment": {"id": "dep-ingress"},
							"spec": {"name": "my-api", "region": "nyc"},
							"region": {"slug": "nyc"}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":             "my-api",
				"region":           "nyc",
				"componentType":    "service",
				"sourceProvider":   "github",
				"gitHubRepo":       "owner/repo",
				"ingressPath":      "/api",
				"corsAllowOrigins": []string{"https://example.com", "https://app.example.com"},
				"corsAllowMethods": []string{"GET", "POST"},
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		// Should store metadata and schedule poll
		assert.False(t, executionState.Passed)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, appPollInterval, requestCtx.Duration)
	})

	t.Run("with VPC -> creates app with VPC configuration", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"app": {
							"id": "vpc-app-id",
							"pending_deployment": {"id": "dep-vpc"},
							"spec": {"name": "my-app", "region": "nyc"},
							"region": {"slug": "nyc"}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":           "my-app",
				"region":         "nyc",
				"componentType":  "service",
				"sourceProvider": "github",
				"gitHubRepo":     "owner/repo",
				"vpcID":          "5218b393-8cef-41a3-a436-72a20de7cba4",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		// Should store metadata and schedule poll
		assert.False(t, executionState.Passed)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, appPollInterval, requestCtx.Duration)
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnprocessableEntity,
					Body:       io.NopCloser(strings.NewReader(`{"id":"unprocessable_entity","message":"Name is already in use"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":           "my-app",
				"region":         "nyc",
				"componentType":  "service",
				"sourceProvider": "github",
				"gitHubRepo":     "owner/repo",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create app")
	})
}

func Test__CreateApp__Configuration(t *testing.T) {
	component := &CreateApp{}
	fields := component.Configuration()

	findField := func(name string) *configuration.Field {
		for _, f := range fields {
			if f.Name == name {
				return &f
			}
		}
		return nil
	}

	t.Run("has component type select field", func(t *testing.T) {
		field := findField("componentType")
		require.NotNil(t, field, "componentType field must exist")
		assert.Equal(t, "select", field.Type)
		assert.True(t, field.Required)
		assert.Equal(t, "service", field.Default)

		require.NotNil(t, field.TypeOptions)
		require.NotNil(t, field.TypeOptions.Select)
		assert.Len(t, field.TypeOptions.Select.Options, 4)

		values := make([]string, len(field.TypeOptions.Select.Options))
		for i, opt := range field.TypeOptions.Select.Options {
			values[i] = opt.Value
		}
		assert.Contains(t, values, "service")
		assert.Contains(t, values, "static-site")
		assert.Contains(t, values, "worker")
		assert.Contains(t, values, "job")
	})

	t.Run("has source provider select field", func(t *testing.T) {
		field := findField("sourceProvider")
		require.NotNil(t, field, "sourceProvider field must exist")
		assert.Equal(t, "select", field.Type)
		assert.True(t, field.Required)
		assert.Equal(t, "github", field.Default)

		require.NotNil(t, field.TypeOptions)
		require.NotNil(t, field.TypeOptions.Select)
		assert.Len(t, field.TypeOptions.Select.Options, 3)
	})

	t.Run("github fields have visibility conditions", func(t *testing.T) {
		field := findField("gitHubRepo")
		require.NotNil(t, field)
		require.Len(t, field.VisibilityConditions, 1)
		assert.Equal(t, "sourceProvider", field.VisibilityConditions[0].Field)
		assert.Equal(t, []string{"github"}, field.VisibilityConditions[0].Values)
		require.Len(t, field.RequiredConditions, 1)
		assert.Equal(t, "sourceProvider", field.RequiredConditions[0].Field)
	})

	t.Run("gitlab fields have visibility conditions", func(t *testing.T) {
		field := findField("gitLabRepo")
		require.NotNil(t, field)
		require.Len(t, field.VisibilityConditions, 1)
		assert.Equal(t, "sourceProvider", field.VisibilityConditions[0].Field)
		assert.Equal(t, []string{"gitlab"}, field.VisibilityConditions[0].Values)
		require.Len(t, field.RequiredConditions, 1)
		assert.Equal(t, "sourceProvider", field.RequiredConditions[0].Field)
	})

	t.Run("bitbucket fields have visibility conditions", func(t *testing.T) {
		field := findField("bitbucketRepo")
		require.NotNil(t, field)
		require.Len(t, field.VisibilityConditions, 1)
		assert.Equal(t, "sourceProvider", field.VisibilityConditions[0].Field)
		assert.Equal(t, []string{"bitbucket"}, field.VisibilityConditions[0].Values)
		require.Len(t, field.RequiredConditions, 1)
		assert.Equal(t, "sourceProvider", field.RequiredConditions[0].Field)
	})

	t.Run("service-specific fields have correct visibility", func(t *testing.T) {
		httpPortField := findField("httpPort")
		require.NotNil(t, httpPortField)
		require.Len(t, httpPortField.VisibilityConditions, 1)
		assert.Equal(t, "componentType", httpPortField.VisibilityConditions[0].Field)
		assert.Equal(t, []string{"service"}, httpPortField.VisibilityConditions[0].Values)
	})

	t.Run("runCommand visible for service, worker, job but not static-site", func(t *testing.T) {
		field := findField("runCommand")
		require.NotNil(t, field)
		require.Len(t, field.VisibilityConditions, 1)
		assert.Equal(t, "componentType", field.VisibilityConditions[0].Field)
		assert.Contains(t, field.VisibilityConditions[0].Values, "service")
		assert.Contains(t, field.VisibilityConditions[0].Values, "worker")
		assert.Contains(t, field.VisibilityConditions[0].Values, "job")
		assert.NotContains(t, field.VisibilityConditions[0].Values, "static-site")
	})

	t.Run("instance size visible for service, worker, job but not static-site", func(t *testing.T) {
		field := findField("instanceSizeSlug")
		require.NotNil(t, field)
		require.Len(t, field.VisibilityConditions, 1)
		assert.Equal(t, "componentType", field.VisibilityConditions[0].Field)
		assert.Contains(t, field.VisibilityConditions[0].Values, "service")
		assert.Contains(t, field.VisibilityConditions[0].Values, "worker")
		assert.Contains(t, field.VisibilityConditions[0].Values, "job")
		assert.NotContains(t, field.VisibilityConditions[0].Values, "static-site")
	})

	t.Run("static-site-specific fields have correct visibility", func(t *testing.T) {
		for _, name := range []string{"outputDir", "indexDocument", "errorDocument", "catchallDocument"} {
			field := findField(name)
			require.NotNil(t, field, "%s field must exist", name)
			require.Len(t, field.VisibilityConditions, 1, "%s must have visibility conditions", name)
			assert.Equal(t, "componentType", field.VisibilityConditions[0].Field)
			assert.Equal(t, []string{"static-site"}, field.VisibilityConditions[0].Values, "%s should only be visible for static-site", name)
		}
	})

	t.Run("common fields have no component type visibility conditions", func(t *testing.T) {
		for _, name := range []string{"environmentSlug", "buildCommand", "sourceDir"} {
			field := findField(name)
			require.NotNil(t, field, "%s field must exist", name)

			hasComponentTypeCondition := false
			for _, vc := range field.VisibilityConditions {
				if vc.Field == "componentType" {
					hasComponentTypeCondition = true
				}
			}
			assert.False(t, hasComponentTypeCondition, "%s should not have componentType visibility condition", name)
		}
	})

	t.Run("has deploy on push boolean field", func(t *testing.T) {
		field := findField("deployOnPush")
		require.NotNil(t, field, "deployOnPush field must exist")
		assert.Equal(t, "boolean", field.Type)
		assert.False(t, field.Togglable)
		assert.Equal(t, true, field.Default)
	})

	t.Run("ingress fields visible for service and static-site only", func(t *testing.T) {
		for _, name := range []string{"ingressPath", "corsAllowOrigins", "corsAllowMethods"} {
			field := findField(name)
			require.NotNil(t, field, "%s field must exist", name)
			require.Len(t, field.VisibilityConditions, 1, "%s must have visibility conditions", name)
			assert.Equal(t, "componentType", field.VisibilityConditions[0].Field)
			assert.Contains(t, field.VisibilityConditions[0].Values, "service")
			assert.Contains(t, field.VisibilityConditions[0].Values, "static-site")
			assert.NotContains(t, field.VisibilityConditions[0].Values, "worker")
			assert.NotContains(t, field.VisibilityConditions[0].Values, "job")
		}
	})

	t.Run("has CORS allow methods multi-select field", func(t *testing.T) {
		field := findField("corsAllowMethods")
		require.NotNil(t, field)
		assert.Equal(t, "multi-select", field.Type)
		require.NotNil(t, field.TypeOptions)
		require.NotNil(t, field.TypeOptions.MultiSelect)
		assert.GreaterOrEqual(t, len(field.TypeOptions.MultiSelect.Options), 5)
	})

	t.Run("has addDatabase boolean field", func(t *testing.T) {
		field := findField("addDatabase")
		require.NotNil(t, field, "addDatabase field must exist")
		assert.Equal(t, "boolean", field.Type)
		assert.False(t, field.Togglable)
		assert.Equal(t, false, field.Default)
	})

	t.Run("database fields visible when addDatabase is true", func(t *testing.T) {
		for _, name := range []string{"databaseName", "databaseEngine", "databaseVersion", "databaseProduction"} {
			field := findField(name)
			require.NotNil(t, field, "%s field must exist", name)

			hasAddDBCondition := false
			for _, vc := range field.VisibilityConditions {
				if vc.Field == "addDatabase" && len(vc.Values) == 1 && vc.Values[0] == "true" {
					hasAddDBCondition = true
				}
			}
			assert.True(t, hasAddDBCondition, "%s should be visible when addDatabase is true", name)
		}
	})

	t.Run("database engine select has correct options", func(t *testing.T) {
		field := findField("databaseEngine")
		require.NotNil(t, field)
		assert.Equal(t, "select", field.Type)
		require.NotNil(t, field.TypeOptions)
		require.NotNil(t, field.TypeOptions.Select)
		assert.Len(t, field.TypeOptions.Select.Options, 4)

		values := make([]string, len(field.TypeOptions.Select.Options))
		for i, opt := range field.TypeOptions.Select.Options {
			values[i] = opt.Value
		}
		assert.Contains(t, values, "PG")
		assert.Contains(t, values, "MYSQL")
		assert.Contains(t, values, "REDIS")
		assert.Contains(t, values, "MONGODB")
	})

	t.Run("managed database fields visible when databaseProduction is true", func(t *testing.T) {
		for _, name := range []string{"databaseClusterName", "databaseDBName", "databaseDBUser"} {
			field := findField(name)
			require.NotNil(t, field, "%s field must exist", name)

			hasProductionCondition := false
			for _, vc := range field.VisibilityConditions {
				if vc.Field == "databaseProduction" && len(vc.Values) == 1 && vc.Values[0] == "true" {
					hasProductionCondition = true
				}
			}
			assert.True(t, hasProductionCondition, "%s should be visible when databaseProduction is true", name)
		}
	})

	t.Run("databaseClusterName required when production is true", func(t *testing.T) {
		field := findField("databaseClusterName")
		require.NotNil(t, field)
		require.Len(t, field.RequiredConditions, 1)
		assert.Equal(t, "databaseProduction", field.RequiredConditions[0].Field)
		assert.Equal(t, []string{"true"}, field.RequiredConditions[0].Values)
	})

	t.Run("has VPC field", func(t *testing.T) {
		field := findField("vpcID")
		require.NotNil(t, field, "vpcID field must exist")
		assert.Equal(t, "string", field.Type)
		assert.True(t, field.Togglable)
	})
}

func Test__CreateApp__HandleAction(t *testing.T) {
	component := &CreateApp{}

	t.Run("deployment active -> emits app output", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// GetDeployment response
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"deployment": {
							"id": "dep-001",
							"phase": "ACTIVE"
						}
					}`)),
				},
				// GetApp response
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"app": {
							"id": "app-001",
							"spec": {"name": "my-app", "region": "nyc"},
							"region": {"slug": "nyc"},
							"live_url": "https://my-app.ondigitalocean.app",
							"default_ingress": "https://my-app.ondigitalocean.app"
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"appID":        "app-001",
				"deploymentID": "dep-001",
			},
		}

		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Integration:    integrationCtx,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "digitalocean.app.created", executionState.Type)
	})

	t.Run("deployment error -> fails execution with details", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"deployment": {
							"id": "dep-002",
							"phase": "ERROR",
							"cause": "build failed",
							"progress": {
								"error_steps": 1,
								"total_steps": 3,
								"steps": [
									{"name": "build", "status": "ERROR"},
									{"name": "deploy", "status": "PENDING"}
								]
							}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"appID":        "app-002",
				"deploymentID": "dep-002",
			},
		}

		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Integration:    integrationCtx,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.False(t, executionState.Passed)
		assert.Equal(t, "deployment_failed", executionState.FailureReason)
		assert.Contains(t, executionState.FailureMessage, "build failed")
		assert.Contains(t, executionState.FailureMessage, "build")
	})

	t.Run("deployment building -> reschedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"deployment": {
							"id": "dep-003",
							"phase": "BUILDING"
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"appID":        "app-003",
				"deploymentID": "dep-003",
			},
		}

		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Integration:    integrationCtx,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.False(t, executionState.Passed)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, appPollInterval, requestCtx.Duration)
	})

	t.Run("deployment pending build -> reschedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"deployment": {
							"id": "dep-004",
							"phase": "PENDING_BUILD"
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"appID":        "app-004",
				"deploymentID": "dep-004",
			},
		}

		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Integration:    integrationCtx,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.False(t, executionState.Passed)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, appPollInterval, requestCtx.Duration)
	})

	t.Run("deployment deploying -> reschedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"deployment": {
							"id": "dep-005",
							"phase": "DEPLOYING"
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"appID":        "app-005",
				"deploymentID": "dep-005",
			},
		}

		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Integration:    integrationCtx,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.False(t, executionState.Passed)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, appPollInterval, requestCtx.Duration)
	})

	t.Run("already finished -> no-op", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{
			Finished: true,
			KVs:      map[string]string{},
		}

		requestCtx := &contexts.RequestContext{}

		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			ExecutionState: executionState,
			Requests:       requestCtx,
		})

		require.NoError(t, err)
		assert.Empty(t, requestCtx.Action)
	})

	t.Run("unknown action -> returns error", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{
			Name: "unknown",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown action: unknown")
	})
}
