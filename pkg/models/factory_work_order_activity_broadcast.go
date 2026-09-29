package models

import (
	"errors"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
)

const (
	MaxFactoryWorkOrderActivityBroadcastTitleBytes = 255
	MaxFactoryWorkOrderActivityBroadcastBodyBytes  = 16 * 1024
	MaxFactoryWorkOrderActivityBroadcastURLBytes   = 2048
)

var ErrFactoryWorkOrderActivityBroadcastInvalid = errors.New("invalid work order activity broadcast")

type FactoryWorkOrderActivityBroadcastParams struct {
	Title      string
	Body       string
	URL        string
	Automation *factory.AutomationRef
	Run        *factory.RunRef
}

func (o *FactoryWorkOrder) RecordActivityBroadcast(tx *gorm.DB, params FactoryWorkOrderActivityBroadcastParams) error {
	broadcast, err := normalizeActivityBroadcast(params)
	if err != nil {
		return err
	}
	broadcast.Order = o.Ref()
	return o.recordEvent(tx, factory.EventTypeOrderActivityBroadcast, broadcast)
}

func normalizeActivityBroadcast(params FactoryWorkOrderActivityBroadcastParams) (factory.WorkOrderActivityBroadcast, error) {
	title := strings.TrimSpace(params.Title)
	body := strings.TrimSpace(params.Body)
	rawURL := strings.TrimSpace(params.URL)

	if title == "" {
		return factory.WorkOrderActivityBroadcast{}, fmt.Errorf("%w: title is required", ErrFactoryWorkOrderActivityBroadcastInvalid)
	}
	if len(title) > MaxFactoryWorkOrderActivityBroadcastTitleBytes {
		return factory.WorkOrderActivityBroadcast{}, fmt.Errorf(
			"%w: title exceeds %d bytes",
			ErrFactoryWorkOrderActivityBroadcastInvalid,
			MaxFactoryWorkOrderActivityBroadcastTitleBytes,
		)
	}
	if body == "" && rawURL == "" {
		return factory.WorkOrderActivityBroadcast{}, fmt.Errorf(
			"%w: body or url is required",
			ErrFactoryWorkOrderActivityBroadcastInvalid,
		)
	}
	if len(body) > MaxFactoryWorkOrderActivityBroadcastBodyBytes {
		return factory.WorkOrderActivityBroadcast{}, fmt.Errorf(
			"%w: body exceeds %d bytes",
			ErrFactoryWorkOrderActivityBroadcastInvalid,
			MaxFactoryWorkOrderActivityBroadcastBodyBytes,
		)
	}
	if rawURL != "" {
		if len(rawURL) > MaxFactoryWorkOrderActivityBroadcastURLBytes {
			return factory.WorkOrderActivityBroadcast{}, fmt.Errorf(
				"%w: url exceeds %d bytes",
				ErrFactoryWorkOrderActivityBroadcastInvalid,
				MaxFactoryWorkOrderActivityBroadcastURLBytes,
			)
		}
		if !isSafeArtifactURL(rawURL) {
			return factory.WorkOrderActivityBroadcast{}, fmt.Errorf(
				"%w: url must be http(s)",
				ErrFactoryWorkOrderActivityBroadcastInvalid,
			)
		}
	}

	return factory.WorkOrderActivityBroadcast{
		Title:      title,
		Body:       body,
		URL:        rawURL,
		Automation: params.Automation,
		Run:        params.Run,
	}, nil
}
