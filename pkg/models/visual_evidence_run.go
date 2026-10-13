package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const visualEvidenceRunTitle = "Capture screenshots or a short video"

// MaybeLinkVisualEvidenceRun attaches a Visual Evidence canvas run to the
// factory pull request in the root event. The task Verify list shows a column
// app only after that link exists.
func MaybeLinkVisualEvidenceRun(tx *gorm.DB, canvas *Canvas, event *CanvasEvent, run *CanvasRun) error {
	if canvas == nil || event == nil || run == nil || canvas.FactoryID == nil {
		return nil
	}
	if event.WorkflowID != canvas.ID || run.WorkflowID != canvas.ID {
		return nil
	}

	matches, err := CanvasHasFactoryAppTemplate(tx, canvas.ID, FactoryAppTemplateVisualEvidenceID)
	if err != nil || !matches {
		return err
	}

	pullRequestID := visualEvidencePullRequestID(event.Data.Data())
	if pullRequestID == uuid.Nil {
		return nil
	}

	factoryModel, err := FindFactory(tx, canvas.OrganizationID, *canvas.FactoryID)
	if err != nil {
		return err
	}
	pullRequest, err := factoryModel.FindPullRequest(tx, FactoryPullRequestLookup{ID: pullRequestID})
	if err != nil {
		return err
	}

	_, err = pullRequest.CreateActivity(tx, FactoryPullRequestActivityParams{
		RunID:  run.ID,
		Title:  visualEvidenceRunTitle,
		Access: FactoryPullRequestAccessConcurrent,
	})
	return err
}

func visualEvidencePullRequestID(eventData any) uuid.UUID {
	payload, ok := objectMap(RootEventSourcePayload(eventData))
	if !ok {
		return uuid.Nil
	}
	pullRequest, ok := objectMap(payload["pullRequest"])
	if !ok {
		return uuid.Nil
	}
	raw, ok := pullRequest["id"].(string)
	if !ok || raw == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil
	}
	return id
}
