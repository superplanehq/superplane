package store

import (
	"testing"
	"time"

	"github.com/superplane/runner/shared/models"
)

func TestTaskMapRoundTripFiles(t *testing.T) {
	task := &models.Task{
		ID:            "task-files",
		FleetID:       "fleet-1",
		RunMode:       models.RunModeCommandList,
		Commands:      models.CommandList{{Name: "Run", Command: "echo hi"}},
		Files:         []models.TaskFile{{Path: "hi.txt", Content: "hello", Mode: "0644"}},
		Labels:        map[string]string{models.LabelCanvasName: "c1", models.LabelNodeName: "n1"},
		WebhookURL:    "https://example/hook",
		Status:        models.StatusQueued,
		CreatedAt:     time.Now().UTC(),
		ExecutionMode: models.ExecutionHost,
	}
	row, err := taskRowFromModel(task)
	if err != nil {
		t.Fatal(err)
	}
	if row.FilesJSON == "" {
		t.Fatal("expected files_json")
	}
	if row.LabelsJSON == "" {
		t.Fatal("expected labels_json")
	}
	got, err := taskModelFromRow(row)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Files) != 1 || got.Files[0].Path != "hi.txt" || got.Files[0].Content != "hello" {
		t.Fatalf("files: %#v", got.Files)
	}
	if got.Labels[models.LabelCanvasName] != "c1" || got.Labels[models.LabelNodeName] != "n1" {
		t.Fatalf("labels: %#v", got.Labels)
	}
}
