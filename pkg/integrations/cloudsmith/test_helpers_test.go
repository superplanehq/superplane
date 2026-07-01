package cloudsmith

import (
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func cloudsmithPackageExecutionContext(httpContext *contexts.HTTPContext, executionState *contexts.ExecutionStateContext) core.ExecutionContext {
	return cloudsmithPackageExecutionContextWithConfiguration(
		httpContext,
		executionState,
		map[string]any{
			"repository": "acme/production",
			"package":    "pkg_123",
		},
	)
}

func cloudsmithPackageExecutionContextWithConfiguration(
	httpContext *contexts.HTTPContext,
	executionState *contexts.ExecutionStateContext,
	configuration map[string]any,
) core.ExecutionContext {
	return core.ExecutionContext{
		Configuration: configuration,
		HTTP:          httpContext,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-key",
			},
		},
		ExecutionState: executionState,
	}
}
