package logging

import (
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
)

func ForStageEvent(event *models.StageEvent) *log.Entry {
	if event == nil {
		return log.WithFields(log.Fields{})
	}

	return log.WithFields(
		log.Fields{
			"id":          event.EventID,
			"source_name": event.SourceName,
			"stage_id":    event.StageID,
		},
	)
}

func ForStage(stage *models.Stage) *log.Entry {
	if stage == nil {
		return log.WithFields(log.Fields{})
	}

	return log.WithFields(
		log.Fields{
			"organization_id": stage.OrganizationID,
			"canvas_id":       stage.CanvasID,
			"name":            stage.Name,
		},
	)
}

func ForCanvas(canvas *models.Canvas) *log.Entry {
	if canvas == nil {
		return log.WithFields(log.Fields{})
	}

	return log.WithFields(
		log.Fields{
			"id":              canvas.ID,
			"name":            canvas.Name,
			"organization_id": canvas.OrganizationID,
		},
	)
}

func ForExecution(execution *models.StageExecution) *log.Entry {
	if execution == nil {
		return log.WithFields(log.Fields{})
	}

	return log.WithFields(
		log.Fields{
			"id":             execution.ID,
			"stage_id":       execution.StageID,
			"stage_event_id": execution.StageEventID,
		},
	)
}

func ForEvent(event *models.Event) *log.Entry {
	if event == nil {
		return log.WithFields(log.Fields{})
	}

	return log.WithFields(
		log.Fields{
			"id":          event.ID,
			"source_id":   event.SourceID,
			"source_type": event.SourceType,
		},
	)
}
