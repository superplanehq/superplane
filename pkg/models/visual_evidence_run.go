package models

import (
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const visualEvidenceRunTitle = "Capture screenshots or a short video"

const customAutomationRunTitle = "Custom automation"

// MaybeLinkVisualEvidenceRun attaches a Visual Evidence canvas run to the
// factory pull request in the root event. The task Verify list shows a column
// app only after that link exists.
func MaybeLinkVisualEvidenceRun(tx *gorm.DB, canvas *Canvas, event *CanvasEvent, run *CanvasRun) error {
	return maybeLinkFactoryColumnAppRun(tx, canvas, event, run, FactoryAppTemplateVisualEvidenceID, visualEvidenceRunTitle)
}

// MaybeLinkCustomAutomationRun attaches a custom Verify automation run to
// the factory pull request in the root event. The task log shows that
// automation only after that link exists.
func MaybeLinkCustomAutomationRun(tx *gorm.DB, canvas *Canvas, event *CanvasEvent, run *CanvasRun) error {
	title := customAutomationRunTitle
	if canvas != nil {
		if name := strings.TrimSpace(canvas.Name); name != "" {
			title = name
		}
	}
	return maybeLinkFactoryColumnAppRun(tx, canvas, event, run, FactoryAppTemplateCustomAutomationID, title)
}

// MaybeLinkMergeConfidenceRun records task authority before agent admission.
func MaybeLinkMergeConfidenceRun(tx *gorm.DB, canvas *Canvas, event *CanvasEvent, run *CanvasRun) error {
	return maybeLinkFactoryColumnAppRun(tx, canvas, event, run, "risk-score", "Merge confidence")
}

func maybeLinkFactoryColumnAppRun(
	tx *gorm.DB,
	canvas *Canvas,
	event *CanvasEvent,
	run *CanvasRun,
	templateID, title string,
) error {
	if canvas == nil || event == nil || run == nil || canvas.FactoryID == nil {
		return nil
	}
	if event.WorkflowID != canvas.ID || run.WorkflowID != canvas.ID {
		return nil
	}

	matches, err := CanvasHasFactoryAppTemplate(tx, canvas.ID, templateID)
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

	order, err := factoryModel.FindWorkOrder(tx, pullRequest.WorkOrderID)
	if err != nil {
		return err
	}
	if err := order.RequireExecutionOwner(tx, true); err != nil {
		return recordColumnAppRejection(tx, order, title, err)
	}
	if err := order.ValidateSubscriptionCanvasModels(tx, canvas.ID, ""); err != nil {
		return recordColumnAppRejection(tx, order, title, err)
	}
	_, err = pullRequest.CreateActivity(tx, FactoryPullRequestActivityParams{
		RunID:  run.ID,
		Title:  title,
		Access: FactoryPullRequestAccessConcurrent,
	})
	return err
}

// recordColumnAppRejection shows members why an automatic app run cannot
// start. Only permanent subscription errors leave a note. Temporary
// contention and database errors pass through unchanged, so the caller can
// retry. The original error is always returned.
func recordColumnAppRejection(tx *gorm.DB, order *FactoryWorkOrder, title string, cause error) error {
	if !IsPermanentSubscriptionAdmissionError(cause) || !order.IsOpen() {
		return cause
	}
	_, err := order.SetStatusNote(tx, FactoryWorkOrderStatusNoteParams{
		Key:      subscriptionAdmissionStatusNoteKey,
		Headline: "Automation could not start",
		Body:     title + ": " + cause.Error(),
	})
	if err != nil {
		return err
	}
	return cause
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
