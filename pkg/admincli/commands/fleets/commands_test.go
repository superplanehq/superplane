package fleets

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/openapi_client"
	clitest "github.com/superplanehq/superplane/test/support/cli"
)

func TestCreateFleetReadsDefinitionFromStdin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/admin/api/installation/fleets", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"fleet":{"id":"aws-large-amd64","enabled":true}}`)
	}))
	defer server.Close()

	ctx, stdout := clitest.NewCommandContext(t, server, "json")
	ctx.Cmd.SetIn(strings.NewReader(`
fleetId: aws-large-amd64
runnerVersion: v0.0.1
enabled: true
spec:
  operatingSystem: linux
  architecture: amd64
  cpuMillicores: 8000
  memoryMb: 32768
  diskGb: 30
`))

	err := (&createCommand{fileCommand: fileCommand{File: "-"}}).Execute(ctx)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), `"id": "aws-large-amd64"`)
}

func TestCreateFleetRejectsUnknownFields(t *testing.T) {
	ctx, _ := clitest.NewCommandContext(t, nil, "json")
	ctx.Cmd.SetIn(strings.NewReader(`
fleetId: aws-large-amd64
runnerVersion: v0.0.1
spec:
  architecture: amd64
unknownField: true
`))

	err := (&createCommand{fileCommand: fileCommand{File: "-"}}).Execute(ctx)
	require.ErrorContains(t, err, "unknownField")
}

func TestDecodeFleetFileReadsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fleet.yml")
	require.NoError(t, os.WriteFile(path, []byte("fleetId: aws-large-amd64\n"), 0600))

	ctx, _ := clitest.NewCommandContext(t, nil, "json")
	var request openapi_client.RunnersCreateFleetRequest
	require.NoError(t, decodeFleetFile(ctx, path, &request))
	assert.Equal(t, "aws-large-amd64", request.GetFleetId())
}

func TestUpdateFleetRequiresMutableFields(t *testing.T) {
	ctx, _ := clitest.NewCommandContext(t, nil, "json")
	ctx.Cmd.SetIn(strings.NewReader("fleetId: aws-large-amd64\n"))

	err := (&updateCommand{fileCommand: fileCommand{File: "-"}}).Execute(ctx)
	require.EqualError(t, err, "at least one of runnerVersion, enabled, or spec is required")
}

func TestDescribeFleetGetsCapacityWithoutLongPolling(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/admin/api/installation/fleets/aws-large-amd64":
			_, _ = fmt.Fprint(w, `{"fleet":{"id":"aws-large-amd64"}}`)
		case "/admin/api/installation/fleets/aws-large-amd64/capacity":
			_, _ = fmt.Fprint(w, `{"runnableTasks":"2","generation":"3"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx, stdout := clitest.NewCommandContext(t, server, "json")
	err := (&describeCommand{FleetID: "aws-large-amd64"}).Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"/admin/api/installation/fleets/aws-large-amd64",
		"/admin/api/installation/fleets/aws-large-amd64/capacity",
	}, paths)
	assert.Contains(t, stdout.String(), `"runnableTasks": "2"`)
}

func TestDescribeFleetTextUsesRelativeTimes(t *testing.T) {
	now := time.Now().UTC()
	created := now.Add(-2 * time.Hour).Truncate(time.Second)
	updated := now.Add(-72 * time.Hour).Truncate(time.Second)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/admin/api/installation/fleets/aws-large-amd64":
			_, _ = fmt.Fprintf(
				w,
				`{"fleet":{"id":"aws-large-amd64","createdAt":%q,"updatedAt":%q}}`,
				created.Format(time.RFC3339),
				updated.Format(time.RFC3339),
			)
		case "/admin/api/installation/fleets/aws-large-amd64/capacity":
			_, _ = fmt.Fprint(w, `{"runnableTasks":"2","generation":"3"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx, stdout := clitest.NewCommandContext(t, server, "text")
	err := (&describeCommand{FleetID: "aws-large-amd64"}).Execute(ctx)

	require.NoError(t, err)
	output := stdout.String()
	assert.Contains(t, output, "Created: 2h ago\n")
	assert.Contains(t, output, "Updated: 3d ago\n")
	assert.NotContains(t, output, created.Format(time.RFC3339))
	assert.NotContains(t, output, updated.Format(time.RFC3339))
}

func TestDescribeYAMLCanBeEditedAndUsedForUpdate(t *testing.T) {
	var update updateFile
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet &&
			r.URL.Path == "/admin/api/installation/fleets/aws-large-amd64":
			_, _ = fmt.Fprint(w, `{
				"fleet":{
					"id":"aws-large-amd64",
					"runnerVersion":"v0.0.1",
					"enabled":true,
					"spec":{"operatingSystem":"linux","architecture":"amd64"}
				}
			}`)
		case r.Method == http.MethodGet &&
			r.URL.Path == "/admin/api/installation/fleets/aws-large-amd64/capacity":
			_, _ = fmt.Fprint(w, `{"runnableTasks":"0","idleRunners":"1","generation":"3"}`)
		case r.Method == http.MethodPatch &&
			r.URL.Path == "/admin/api/installation/fleets/aws-large-amd64":
			require.NoError(t, json.NewDecoder(r.Body).Decode(&update))
			_, _ = fmt.Fprint(w, `{"fleet":{"id":"aws-large-amd64","runnerVersion":"v0.0.2"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	describeContext, described := clitest.NewCommandContext(t, server, "yaml")
	require.NoError(
		t,
		(&describeCommand{FleetID: "aws-large-amd64"}).Execute(describeContext),
	)
	definition := strings.Replace(described.String(), "v0.0.1", "v0.0.2", 1)

	updateContext, _ := clitest.NewCommandContext(t, server, "yaml")
	updateContext.Cmd.SetIn(strings.NewReader(definition))
	require.NoError(
		t,
		(&updateCommand{fileCommand: fileCommand{File: "-"}}).Execute(updateContext),
	)

	require.NotNil(t, update.RunnerVersion)
	assert.Equal(t, "v0.0.2", *update.RunnerVersion)
	assert.Equal(t, "linux", update.Spec.GetOperatingSystem())
	assert.Nil(t, update.Capacity)
}
