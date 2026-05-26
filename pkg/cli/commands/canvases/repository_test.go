package canvases

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const repositoryTestCanvasID = "441a9b61-3bce-417d-9e82-f1a1017dc398"

func TestRepositoryGitURLCommandPrintsGitURL(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + repositoryTestCanvasID + "/repository",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"repository":{"canvasId":"` + repositoryTestCanvasID + `","gitUrl":"https://acme.code.storage/orgs/org/canvases/` + repositoryTestCanvasID + `.git"}}`))
			},
		},
	)
	ctx, stdout := newCreateCommandContextForTest(t, server.server, "text")
	ctx.Args = []string{repositoryTestCanvasID}

	err := (&repositoryGitURLCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, "https://acme.code.storage/orgs/org/canvases/"+repositoryTestCanvasID+".git\n", stdout.String())
}

func TestRepositoryCredentialHelperPrintsGitCredentials(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodPost,
			path:   "/api/v1/canvases/" + repositoryTestCanvasID + "/repository/credentials",
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, false, body["readOnly"])
				require.Equal(t, "3600", body["ttlSeconds"])

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"username":"t","password":"secret"}`))
			},
		},
	)
	ctx, stdout := newCreateCommandContextForTest(t, server.server, "text")
	ctx.Args = []string{"get"}
	ctx.Cmd.SetIn(strings.NewReader("protocol=https\nhost=acme.code.storage\npath=orgs/org/canvases/" + repositoryTestCanvasID + ".git\n\n"))

	readOnly := false
	ttlSeconds := int64(3600)
	allowForcePush := false
	err := (&repositoryCredentialHelperCommand{
		readOnly:       &readOnly,
		ttlSeconds:     &ttlSeconds,
		allowForcePush: &allowForcePush,
	}).Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, "username=t\npassword=secret\n\n", stdout.String())
}

func TestRepositoryCredentialHelperIgnoresStoreAndErase(t *testing.T) {
	ctx, stdout := newCreateCommandContextForTest(t, nil, "text")
	ctx.Args = []string{"store"}

	err := (&repositoryCredentialHelperCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Empty(t, stdout.String())
}

func TestCanvasIDFromGitCredentialInputRequiresPath(t *testing.T) {
	_, err := canvasIDFromGitCredentialInput(strings.NewReader("protocol=https\nhost=acme.code.storage\n\n"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "credential.useHttpPath=true")
}
