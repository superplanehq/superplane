package agent

import (
	"encoding/json"
	"io"
	"strings"
	"time"
)

type liveLogCommandStatus string

const (
	liveLogCommandPassed liveLogCommandStatus = "passed"
	liveLogCommandFailed liveLogCommandStatus = "failed"
)

type liveLogCommandStartRecord struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Text  string `json:"text"`
}

type liveLogCommandEndRecord struct {
	Type       string               `json:"type"`
	Index      int                  `json:"index"`
	Status     liveLogCommandStatus `json:"status"`
	DurationMS int64                `json:"duration_ms"`
}

func writeLiveLogCommandStart(live io.Writer, index int, text string) {
	if live == nil || index < 0 {
		return
	}
	rec := liveLogCommandStartRecord{
		Type:  "cmd_start",
		Index: index,
		Text:  strings.TrimSpace(text),
	}
	writeLiveLogRecord(live, rec)
}

func writeLiveLogCommandEnd(live io.Writer, index int, exitCode int, duration time.Duration) {
	if live == nil || index < 0 {
		return
	}
	status := liveLogCommandPassed
	if exitCode != 0 {
		status = liveLogCommandFailed
	}
	if duration < 0 {
		duration = 0
	}
	rec := liveLogCommandEndRecord{
		Type:       "cmd_end",
		Index:      index,
		Status:     status,
		DurationMS: duration.Milliseconds(),
	}
	writeLiveLogRecord(live, rec)
}

func writeLiveLogRecord(live io.Writer, rec any) {
	b, err := json.Marshal(rec)
	if err != nil {
		return
	}
	_, _ = live.Write(append(b, '\n'))
}
