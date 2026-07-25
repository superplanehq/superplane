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
	"github.com/superplane/runner/shared/models"
	brokermodels "github.com/superplane/runner/task-broker/internal/models"
	"github.com/superplane/runner/task-broker/internal/store/testdb"
)

func TestCreateTaskRequiresFleetID(t *testing.T) {
	srv := &Server{}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "token"}))
	defer ts.Close()

	body, err := json.Marshal(api.BrokerCreateTaskRequest{
		CreateTaskRequest: api.CreateTaskRequest{
			Commands:   models.CommandList{{Command: "echo hi"}},
			WebhookURL: "https://example.com/hook",
		},
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
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status: %d body: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	if !strings.Contains(string(respBody), "fleet_id required") {
		t.Fatalf("body: %s", strings.TrimSpace(string(respBody)))
	}
}

func TestCreateTaskPersistsOriginLabels(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	if err := st.CreateFleet(context.Background(), &brokermodels.Fleet{
		ID:          "fleet-labels",
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
		"fleet_id": "fleet-labels",
		"webhook_url": "https://example.com/hook",
		"commands": ["echo hi"],
		"labels": {
			"canvas_name": "release-train",
			"node_name": "Run tests",
			"ignored": "nope"
		}
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
	if task.Labels[models.LabelCanvasName] != "release-train" || task.Labels[models.LabelNodeName] != "Run tests" {
		t.Fatalf("labels=%#v", task.Labels)
	}
	if _, ok := task.Labels["ignored"]; ok {
		t.Fatalf("ignored key should be dropped: %#v", task.Labels)
	}
}
