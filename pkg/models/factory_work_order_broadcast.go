package models

import (
	"errors"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
)

const (
	MaxFactoryWorkOrderBroadcastSummaryBytes  = 512
	MaxFactoryWorkOrderBroadcastBodyBytes     = 16 * 1024
	MaxFactoryWorkOrderBroadcastURLBytes      = 2048
	MaxFactoryWorkOrderBroadcastURLLabelBytes = 255
)

var ErrFactoryWorkOrderBroadcastInvalid = errors.New("invalid work order broadcast")

// FactoryWorkOrderContentBroadcastParams carries one activity-log update.
// Summary is required. Body or URL is required so the line can expand.
type FactoryWorkOrderContentBroadcastParams struct {
	Summary    string
	Body       string
	URL        string
	URLLabel   string
	Automation *factory.AutomationRef
	Run        *factory.RunRef
}

func (o *FactoryWorkOrder) RecordContentBroadcast(
	tx *gorm.DB,
	params FactoryWorkOrderContentBroadcastParams,
) error {
	normalized, err := normalizeContentBroadcast(params)
	if err != nil {
		return err
	}

	return o.recordEvent(tx, factory.EventTypeOrderContentBroadcast, factory.WorkOrderContentBroadcast{
		Order:      o.Ref(),
		Summary:    normalized.Summary,
		Body:       normalized.Body,
		URL:        normalized.URL,
		URLLabel:   normalized.URLLabel,
		Automation: params.Automation,
		Run:        params.Run,
	})
}

func normalizeContentBroadcast(
	params FactoryWorkOrderContentBroadcastParams,
) (FactoryWorkOrderContentBroadcastParams, error) {
	summary := strings.TrimSpace(params.Summary)
	body := strings.TrimSpace(params.Body)
	rawURL := strings.TrimSpace(params.URL)
	urlLabel := strings.TrimSpace(params.URLLabel)

	if summary == "" {
		return FactoryWorkOrderContentBroadcastParams{}, fmt.Errorf(
			"%w: summary is required",
			ErrFactoryWorkOrderBroadcastInvalid,
		)
	}
	if len(summary) > MaxFactoryWorkOrderBroadcastSummaryBytes {
		return FactoryWorkOrderContentBroadcastParams{}, fmt.Errorf(
			"%w: summary exceeds %d bytes",
			ErrFactoryWorkOrderBroadcastInvalid,
			MaxFactoryWorkOrderBroadcastSummaryBytes,
		)
	}
	if body == "" && rawURL == "" {
		return FactoryWorkOrderContentBroadcastParams{}, fmt.Errorf(
			"%w: content or link URL is required",
			ErrFactoryWorkOrderBroadcastInvalid,
		)
	}
	if len(body) > MaxFactoryWorkOrderBroadcastBodyBytes {
		return FactoryWorkOrderContentBroadcastParams{}, fmt.Errorf(
			"%w: content exceeds %d bytes",
			ErrFactoryWorkOrderBroadcastInvalid,
			MaxFactoryWorkOrderBroadcastBodyBytes,
		)
	}
	if rawURL != "" {
		if len(rawURL) > MaxFactoryWorkOrderBroadcastURLBytes {
			return FactoryWorkOrderContentBroadcastParams{}, fmt.Errorf(
				"%w: link URL exceeds %d bytes",
				ErrFactoryWorkOrderBroadcastInvalid,
				MaxFactoryWorkOrderBroadcastURLBytes,
			)
		}
		if !isSafeArtifactURL(rawURL) {
			return FactoryWorkOrderContentBroadcastParams{}, fmt.Errorf(
				"%w: link URL must be an absolute http or https URL",
				ErrFactoryWorkOrderBroadcastInvalid,
			)
		}
	} else {
		urlLabel = ""
	}
	if len(urlLabel) > MaxFactoryWorkOrderBroadcastURLLabelBytes {
		return FactoryWorkOrderContentBroadcastParams{}, fmt.Errorf(
			"%w: link label exceeds %d bytes",
			ErrFactoryWorkOrderBroadcastInvalid,
			MaxFactoryWorkOrderBroadcastURLLabelBytes,
		)
	}

	return FactoryWorkOrderContentBroadcastParams{
		Summary:  summary,
		Body:     body,
		URL:      rawURL,
		URLLabel: urlLabel,
	}, nil
}
