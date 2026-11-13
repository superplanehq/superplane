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
	Type      string `json:"type"`
	RecordID  string `json:"diagnostic_id,omitempty"`
	Index     int    `json:"index"`
	Text      string `json:"text"`
	Kind      string `json:"kind,omitempty"`
	Preview   string `json:"preview,omitempty"`
	StartedAt int64  `json:"started_at"`
}

type liveLogToolStartRecord struct {
	Type      string `json:"type"`
	Kind      string `json:"kind,omitempty"`
	Text      string `json:"text,omitempty"`
	StartedAt int64  `json:"started_at,omitempty"`
}

type liveLogToolEndRecord struct {
	Type       string               `json:"type"`
	Kind       string               `json:"kind,omitempty"`
	Status     liveLogCommandStatus `json:"status"`
	DurationMS int64                `json:"duration_ms"`
}

type liveLogCommandEndRecord struct {
	Type       string               `json:"type"`
	RecordID   string               `json:"diagnostic_id,omitempty"`
	Index      int                  `json:"index"`
	Status     liveLogCommandStatus `json:"status"`
	DurationMS int64                `json:"duration_ms"`
}

func writeLiveLogCommandStart(live io.Writer, index int, text, kind, preview string, startedAt time.Time) {
	if live == nil || index < 0 {
		return
	}
	rec := liveLogCommandStartRecord{
		Type:      "cmd_start",
		RecordID:  diagnosticRecordID(live),
		Index:     index,
		Text:      strings.TrimSpace(text),
		Kind:      strings.TrimSpace(kind),
		Preview:   strings.TrimSpace(preview),
		StartedAt: startedAt.UnixMilli(),
	}
	writeLiveLogRecord(live, rec)
}

func liveLogPreview(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		runes := []rune(line)
		if len(runes) > 80 {
			return string(runes[:80])
		}
		return line
	}
	return ""
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
		RecordID:   diagnosticRecordID(live),
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
	// Leading newline keeps control JSON on its own log record when the
	// previous stdout chunk omitted a trailing newline (otherwise cmd_end sticks
	// to that chunk and live-log UI never sees the command finish).
	payload := make([]byte, 0, 1+len(b)+1)
	payload = append(payload, '\n')
	payload = append(payload, b...)
	payload = append(payload, '\n')
	_, _ = live.Write(payload)
}

// The identifier distinguishes generated records from JSON printed or replayed
// by commands. It correlates records; it is not a task security boundary.
func diagnosticRecordID(live io.Writer) string {
	if capture, ok := live.(*failureDiagnostics); ok {
		return capture.recordID
	}
	return ""
}

func diagnosticDirectives(directives []shellDirective, recordID string) []shellDirective {
	for i := range directives {
		directives[i].RecordID = recordID
	}
	return directives
}
