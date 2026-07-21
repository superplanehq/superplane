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

	brokermodels "github.com/superplanehq/superplane/pkg/taskbroker/models"
	"github.com/superplanehq/superplane/pkg/taskbroker/shared/api"
	"github.com/superplanehq/superplane/pkg/taskbroker/shared/models"
)

func TestCreateBrokerTaskPersistsEnvironment(t *testing.T) {
	st := openStore(t)
	if err := st.CreateFleet(context.Background(), &brokermodels.Fleet{
		ID:          "fleet-1",
		Provisioner: "local",
		Arch:        "amd64",
		Size:        "local",
		CreatedAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	srv := &Server{
		Store: st,
		Log:   slog.Default(),
	}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "token"}))
	defer ts.Close()

	body, err := json.Marshal(api.BrokerCreateTaskRequest{
		CreateTaskRequest: api.CreateTaskRequest{
			Commands:   models.CommandList{{Command: "echo \"$COMMIT_AUTHOR\""}},
			WebhookURL: "https://example.com/hook",
			Environment: []api.EnvironmentVariable{
				{Name: "COMMIT_AUTHOR", Value: "alice@example.com"},
				{Name: "SPECIAL", Value: "line one\nline two=ok"},
			},
		},
		FleetID: "fleet-1",
	})
	if err != nil {
		t.Fatal(err)
	}

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
	respBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create broker task: %d %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var created api.BrokerCreateTaskResponse
	if err := json.Unmarshal(respBody, &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatal("empty task id")
	}

	task, err := st.GetTask(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if task == nil {
		t.Fatal("task not found")
	}
	if len(task.Environment) != 2 {
		t.Fatalf("environment: %#v", task.Environment)
	}
	if task.Environment[0].Name != "COMMIT_AUTHOR" || task.Environment[0].Value != "alice@example.com" {
		t.Fatalf("first env: %#v", task.Environment[0])
	}
	if task.Environment[1].Name != "SPECIAL" || task.Environment[1].Value != "line one\nline two=ok" {
		t.Fatalf("second env: %#v", task.Environment[1])
	}
	if task.WebhookURL != "https://example.com/hook" {
		t.Fatalf("webhook: %q", task.WebhookURL)
	}
}
