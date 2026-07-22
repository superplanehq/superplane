package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/superplane/runner/shared/api"
	brokermodels "github.com/superplane/runner/task-broker/internal/models"
	"github.com/superplane/runner/task-broker/internal/store/testdb"
)

func TestCreateTaskAcceptsNamedCommands(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	if err := st.CreateFleet(context.Background(), &brokermodels.Fleet{
		ID:          "fleet-named",
		Provisioner: "local",
		Arch:        "amd64",
		Size:        "local",
		CreatedAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	srv := &Server{Store: st, Log: slog.Default()}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "token"}))
	defer ts.Close()

	body := []byte(`{
		"fleet_id": "fleet-named",
		"webhook_url": "https://example.com/hook",
		"commands": [
			{"name": "Clone", "command": "git clone repo"},
			{"command": "echo unnamed-object"}
		]
	}`)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/tasks", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var created api.BrokerCreateTaskResponse
	if err := json.Unmarshal(respBody, &created); err != nil {
		t.Fatal(err)
	}
	task, err := st.GetTask(context.Background(), created.ID)
	if err != nil || task == nil {
		t.Fatalf("get task: %v %#v", err, task)
	}
	if len(task.Commands) != 2 {
		t.Fatalf("commands=%#v", task.Commands)
	}
	if task.Commands[0].Name != "Clone" || task.Commands[0].Command != "git clone repo" {
		t.Fatalf("cmd0=%#v", task.Commands[0])
	}
	if task.Commands[1].Command != "echo unnamed-object" || task.Commands[1].Name != "" {
		t.Fatalf("cmd1=%#v", task.Commands[1])
	}
}

func TestCreateTaskRejectsPlainStringCommands(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	if err := st.CreateFleet(context.Background(), &brokermodels.Fleet{
		ID:          "fleet-string-cmds",
		Provisioner: "local",
		Arch:        "amd64",
		Size:        "local",
		CreatedAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	srv := &Server{Store: st, Log: slog.Default()}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "token"}))
	defer ts.Close()

	body := []byte(`{
		"fleet_id": "fleet-string-cmds",
		"webhook_url": "https://example.com/hook",
		"commands": ["echo plain"]
	}`)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/tasks", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var errBody struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(respBody, &errBody); err != nil {
		t.Fatalf("body=%s", strings.TrimSpace(string(respBody)))
	}
	if !strings.Contains(errBody.Error, `must be an object with "command"`) {
		t.Fatalf("error=%q", errBody.Error)
	}
}

func TestCreateTaskRejectsNamedCommandWithoutCommand(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	if err := st.CreateFleet(context.Background(), &brokermodels.Fleet{
		ID:          "fleet-named-bad",
		Provisioner: "local",
		Arch:        "amd64",
		Size:        "local",
		CreatedAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	srv := &Server{Store: st, Log: slog.Default()}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "token"}))
	defer ts.Close()

	body := []byte(`{
		"fleet_id": "fleet-named-bad",
		"webhook_url": "https://example.com/hook",
		"commands": [{"name": "Nope"}]
	}`)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/tasks", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusCreated {
		t.Fatal("expected rejection")
	}
}
