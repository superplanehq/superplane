package public

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/test/support"
)

func TestCanvasReadsFailClosedOnAuthorizationError(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	server, signer := mustRunnerLiveLogServer(t, r)
	server.RegisterWebSocketRoutes()
	task := createSessionRunnerTask(t, r)
	canvas, execution := createCanvasWithComponentExecution(t, r, "runner", "auth-error", map[string]any{
		runneraction.ExecutionMetadataBrokerTaskID: task.ID.String(),
	})
	credentials := canvasReadCredentials(t, r, signer, canvas)
	server.authService = failedCanvasReadAuthorization{Authorization: r.AuthService}
	queries := observeCanvasReadQueries(t)
	for _, path := range []string{
		fmt.Sprintf("/api/v1/canvases/%s/file?path=canvas.yaml", canvas),
		fmt.Sprintf("/api/v1/canvases/%s/node-executions/%s/runner-logs", canvas, execution),
		fmt.Sprintf("/api/v1/canvases/%s/node-executions/%s/runner-live-logs/session", canvas, execution),
		"/ws/" + canvas.String(),
	} {
		response := canvasReadGET(server, credentials[0].token, path)
		require.Equal(t, http.StatusInternalServerError, response.Code, response.Body.String())
		require.NotContains(t, response.Body.String(), "private authorization failure")
		require.Empty(t, *queries)
	}
}

type failedCanvasReadAuthorization struct{ authorization.Authorization }

func (failedCanvasReadAuthorization) CheckOrganizationPermission(context.Context, string, string, string, string) (bool, error) {
	return true, errors.New("private authorization failure")
}
