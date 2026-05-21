// Package cloudwatchlog streams task stdout/stderr to Amazon CloudWatch Logs (PutLogEvents).
package cloudwatchlog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

// StreamConfig selects log group and stream for a single task run.
type StreamConfig struct {
	LogGroup   string
	StreamName string
	Region     string // optional; default chain (e.g. AWS_REGION on EC2)
}

// StreamWriter batches task log bytes into CloudWatch log events.
type StreamWriter struct {
	client *cloudwatchlogs.Client
	group  string
	stream string
	mu        sync.Mutex
	buf       []byte
	token     *string
	lastTS    int64
	closed    bool
	stopFlush chan struct{}
	flushDone chan struct{}
}

var errClosed = errors.New("cloudwatchlog: writer closed")

// NewStreamWriter creates the log stream (and log group when permitted) and starts periodic flush.
func NewStreamWriter(ctx context.Context, cfg StreamConfig) (*StreamWriter, error) {
	group := strings.TrimSpace(cfg.LogGroup)
	stream := strings.TrimSpace(cfg.StreamName)
	if group == "" || stream == "" {
		return nil, fmt.Errorf("cloudwatchlog: log group and stream name required")
	}

	opts := []func(*awsconfig.LoadOptions) error{}
	if r := strings.TrimSpace(cfg.Region); r != "" {
		opts = append(opts, awsconfig.WithRegion(r))
	}
	awscfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("cloudwatchlog: aws config: %w", err)
	}
	c := cloudwatchlogs.NewFromConfig(awscfg)

	w := &StreamWriter{
		client:    c,
		group:     group,
		stream:    stream,
		stopFlush: make(chan struct{}),
		flushDone: make(chan struct{}),
	}
	if err := w.ensureStream(ctx); err != nil {
		return nil, err
	}

	go w.flushLoop()
	return w, nil
}

func (w *StreamWriter) ensureStream(ctx context.Context) error {
	_, err := w.client.CreateLogGroup(ctx, &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(w.group),
	})
	if err != nil {
		var dup *types.ResourceAlreadyExistsException
		if !errors.As(err, &dup) {
			// Group may already exist or creation may be denied; continue to stream creation.
		}
	}

	_, err = w.client.CreateLogStream(ctx, &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(w.group),
		LogStreamName: aws.String(w.stream),
	})
	if err != nil {
		var dup *types.ResourceAlreadyExistsException
		if errors.As(err, &dup) {
			return w.refreshSequenceToken(ctx)
		}
		return fmt.Errorf("cloudwatchlog: create log stream: %w", err)
	}
	return nil
}

func (w *StreamWriter) refreshSequenceToken(ctx context.Context) error {
	out, err := w.client.DescribeLogStreams(ctx, &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        aws.String(w.group),
		LogStreamNamePrefix: aws.String(w.stream),
		Limit:               aws.Int32(50),
	})
	if err != nil {
		return fmt.Errorf("cloudwatchlog: describe log streams: %w", err)
	}
	for i := range out.LogStreams {
		ls := &out.LogStreams[i]
		if aws.ToString(ls.LogStreamName) == w.stream {
			w.mu.Lock()
			w.token = ls.UploadSequenceToken
			w.mu.Unlock()
			return nil
		}
	}
	return fmt.Errorf("cloudwatchlog: log stream %q not found", w.stream)
}

func (w *StreamWriter) flushLoop() {
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-w.stopFlush:
			_ = w.flush(context.Background())
			close(w.flushDone)
			return
		case <-t.C:
			_ = w.flush(context.Background())
		}
	}
}

// Write buffers data for PutLogEvents. Safe for concurrent use with Close.
func (w *StreamWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return 0, errClosed
	}
	w.buf = append(w.buf, p...)
	return len(p), nil
}

// Close stops the flush ticker and sends remaining bytes.
func (w *StreamWriter) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	w.mu.Unlock()

	close(w.stopFlush)
	<-w.flushDone
	return nil
}

const maxCWMessageRunes = 240_000 // below CloudWatch 262144-byte limit with UTF-8 headroom

const maxEventsPerPut = 900

func eventsToBytes(evts []types.InputLogEvent) []byte {
	var b strings.Builder
	for _, e := range evts {
		b.WriteString(aws.ToString(e.Message))
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

func (w *StreamWriter) flush(ctx context.Context) error {
	w.mu.Lock()
	if len(w.buf) == 0 {
		w.mu.Unlock()
		return nil
	}
	data := append([]byte(nil), w.buf...)
	w.buf = w.buf[:0]
	w.mu.Unlock()

	data = normalizeTerminalOutput(data)
	lastTS := w.takeLastTS()
	allEvents := bytesToEvents(data, &lastTS)
	if len(allEvents) == 0 {
		w.mergeLastTS(lastTS)
		return nil
	}

	start := 0
	for start < len(allEvents) {
		end := start + maxEventsPerPut
		if end > len(allEvents) {
			end = len(allEvents)
		}
		part := allEvents[start:end]

		token := w.takeToken()
		ctx2, cancel := context.WithTimeout(ctx, 45*time.Second)
		out, err := w.client.PutLogEvents(ctx2, &cloudwatchlogs.PutLogEventsInput{
			LogGroupName:  aws.String(w.group),
			LogStreamName: aws.String(w.stream),
			LogEvents:     part,
			SequenceToken: token,
		})
		cancel()
		if err != nil {
			var inv *types.InvalidSequenceTokenException
			if errors.As(err, &inv) && inv.ExpectedSequenceToken != nil {
				w.setToken(inv.ExpectedSequenceToken)
				token = inv.ExpectedSequenceToken
				ctx3, cancel3 := context.WithTimeout(ctx, 45*time.Second)
				out, err = w.client.PutLogEvents(ctx3, &cloudwatchlogs.PutLogEventsInput{
					LogGroupName:  aws.String(w.group),
					LogStreamName: aws.String(w.stream),
					LogEvents:     part,
					SequenceToken: token,
				})
				cancel3()
			}
			if err != nil {
				rest := append([]types.InputLogEvent(nil), allEvents[start:]...)
				w.prependBytes(eventsToBytes(rest))
				w.mergeLastTS(lastTS)
				return err
			}
		}
		if out != nil && out.NextSequenceToken != nil {
			w.setToken(out.NextSequenceToken)
		}
		start = end
	}
	w.mergeLastTS(lastTS)
	return nil
}

func (w *StreamWriter) takeLastTS() int64 {
	w.mu.Lock()
	v := w.lastTS
	w.mu.Unlock()
	return v
}

func (w *StreamWriter) mergeLastTS(v int64) {
	w.mu.Lock()
	if v > w.lastTS {
		w.lastTS = v
	}
	w.mu.Unlock()
}

func (w *StreamWriter) takeToken() *string {
	w.mu.Lock()
	t := w.token
	w.mu.Unlock()
	return t
}

func (w *StreamWriter) setToken(t *string) {
	w.mu.Lock()
	w.token = t
	w.mu.Unlock()
}

func (w *StreamWriter) prependBytes(b []byte) {
	if len(b) == 0 {
		return
	}
	w.mu.Lock()
	w.buf = append(b, w.buf...)
	w.mu.Unlock()
}

// ansiEscape matches CSI color/cursor sequences (e.g. apt progress bars use \033[33m … \033[0m).
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

func stripANSI(data []byte) []byte {
	return ansiEscape.ReplaceAll(data, nil)
}

// normalizeTerminalOutput makes TTY-oriented task stdout/stderr readable in CloudWatch.
func normalizeTerminalOutput(data []byte) []byte {
	return collapseCarriageReturns(stripANSI(data))
}

// collapseCarriageReturns applies terminal-style \r handling: bytes before the last \r
// on each line are dropped so progress output (e.g. git "Counting objects") becomes one
// readable line instead of garbled columns in CloudWatch.
func collapseCarriageReturns(data []byte) []byte {
	if !bytes.Contains(data, []byte{'\r'}) {
		return data
	}
	out := make([]byte, 0, len(data))
	lineStart := 0
	for i := 0; i < len(data); i++ {
		switch data[i] {
		case '\r':
			if i+1 < len(data) && data[i+1] == '\n' {
				out = append(out, data[lineStart:i]...)
				out = append(out, '\n')
				i++
				lineStart = i + 1
				continue
			}
			lineStart = i + 1
		case '\n':
			out = append(out, data[lineStart:i]...)
			out = append(out, '\n')
			lineStart = i + 1
		}
	}
	return append(out, data[lineStart:]...)
}

func bytesToEvents(data []byte, lastTS *int64) []types.InputLogEvent {
	var events []types.InputLogEvent
	for len(data) > 0 {
		idx := bytes.IndexByte(data, '\n')
		var line []byte
		if idx < 0 {
			line = data
			data = nil
		} else {
			line = data[:idx]
			data = data[idx+1:]
		}
		line = bytes.TrimSuffix(line, []byte("\r"))
		msg := string(line)
		msg = strings.ToValidUTF8(msg, "\uFFFD")
		if msg == "" && idx >= 0 {
			msg = "\n"
		}
		for len(msg) > 0 {
			chunk := msg
			if utf8.RuneCountInString(chunk) > maxCWMessageRunes {
				chunk = string([]rune(chunk)[:maxCWMessageRunes])
				msg = msg[len(chunk):]
			} else {
				msg = ""
			}
			ts := time.Now().UnixMilli()
			if ts <= *lastTS {
				ts = *lastTS + 1
			}
			*lastTS = ts
			events = append(events, types.InputLogEvent{
				Message:   aws.String(chunk),
				Timestamp: aws.Int64(ts),
			})
		}
	}
	return events
}
