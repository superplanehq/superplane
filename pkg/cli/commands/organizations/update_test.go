package organizations

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type updateBody struct {
	Organization struct {
		Metadata struct {
			Name        *string `json:"name"`
			Description *string `json:"description"`
		} `json:"metadata"`
	} `json:"organization"`
}

func newUpdateServer(t *testing.T, seen *updateBody) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/me":
			writeMeResponse(w)
		case "/api/v1/organizations/" + testOrgID:
			payload, _ := io.ReadAll(r.Body)
			require.NoError(t, json.Unmarshal(payload, seen))
			writeOrganizationResponse(w)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestUpdateCommandRequiresFlag(t *testing.T) {
	ctx, _ := newTestContext(t, nil, "text")
	name := ""
	description := ""
	err := (&updateCommand{name: &name, description: &description}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one flag")
}

func TestUpdateCommandSendsOnlyName(t *testing.T) {
	var seen updateBody
	server := newUpdateServer(t, &seen)

	ctx, stdout := newTestContext(t, server, "text")
	name := "Acme"
	description := ""
	cmd := &updateCommand{name: &name, description: &description}

	ctx.Cmd.Flags().String("name", "", "")
	ctx.Cmd.Flags().String("description", "", "")
	require.NoError(t, ctx.Cmd.Flags().Set("name", name))

	require.NoError(t, cmd.Execute(ctx))
	require.NotNil(t, seen.Organization.Metadata.Name)
	require.Equal(t, "Acme", *seen.Organization.Metadata.Name)
	require.Nil(t, seen.Organization.Metadata.Description)
	require.Contains(t, stdout.String(), "Name: Acme")
}

func TestUpdateCommandSendsOnlyDescription(t *testing.T) {
	var seen updateBody
	server := newUpdateServer(t, &seen)

	ctx, stdout := newTestContext(t, server, "text")
	name := ""
	description := "New description"
	cmd := &updateCommand{name: &name, description: &description}

	ctx.Cmd.Flags().String("name", "", "")
	ctx.Cmd.Flags().String("description", "", "")
	require.NoError(t, ctx.Cmd.Flags().Set("description", description))

	require.NoError(t, cmd.Execute(ctx))
	require.Nil(t, seen.Organization.Metadata.Name)
	require.NotNil(t, seen.Organization.Metadata.Description)
	require.Equal(t, "New description", *seen.Organization.Metadata.Description)
	require.Contains(t, stdout.String(), "Description: Acme corp")
}
