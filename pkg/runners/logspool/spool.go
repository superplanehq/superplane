package logspool

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	runnerapi "github.com/superplanehq/superplane/pkg/runners/api"
)

const (
	defaultChunkBytes           = 64 * 1024
	defaultMaxBytes             = 10 * 1024 * 1024
	defaultRetryDelay           = time.Second
	defaultMinimumFlushInterval = 2500 * time.Millisecond
	defaultMaximumFlushInterval = 5 * time.Second
)

var droppedRecord = []byte(
	`{"type":"line","text":"[Runner logs were dropped because the local log spool reached its limit.]"}` + "\n",
)

type Config struct {
	Directory        string
	TaskID           string
	BaseURL          string
	AccessToken      string
	ChunkBytes       int64
	MaxBytes         int64
	RetryDelay       time.Duration
	FlushInterval    time.Duration
	FlushIntervalMin time.Duration
	FlushIntervalMax time.Duration
	HTTPClient       *http.Client
	Log              *slog.Logger
}

// Spool converts process output to NDJSON and durably stages bounded chunks.
// One uploader sends sealed chunks in sequence and deletes only acknowledged
// files.
type Spool struct {
	config Config

	mu             sync.Mutex
	current        *os.File
	currentPath    string
	currentBytes   int64
	diskBytes      int64
	peakBytes      int64
	nextSequence   int64
	uploadSequence int64
	line           []byte
	dropped        bool
	closed         bool
	stopped        bool
	terminalErr    error

	wake chan struct{}
	done chan struct{}
}

func New(config Config) (*Spool, error) {
	config.Directory = strings.TrimSpace(config.Directory)
	config.TaskID = strings.TrimSpace(config.TaskID)
	config.BaseURL = strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	config.AccessToken = strings.TrimSpace(config.AccessToken)
	if config.ChunkBytes <= 0 {
		config.ChunkBytes = defaultChunkBytes
	}
	if config.MaxBytes <= 0 {
		config.MaxBytes = defaultMaxBytes
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = defaultRetryDelay
	}
	if config.FlushInterval > 0 {
		config.FlushIntervalMin = config.FlushInterval
		config.FlushIntervalMax = config.FlushInterval
	} else {
		if config.FlushIntervalMin <= 0 {
			config.FlushIntervalMin = defaultMinimumFlushInterval
		}
		if config.FlushIntervalMax < config.FlushIntervalMin {
			config.FlushIntervalMax = config.FlushIntervalMin
		}
	}
	if config.HTTPClient == nil {
		config.HTTPClient = http.DefaultClient
	}
	switch {
	case config.Directory == "":
		return nil, errors.New("log spool directory is required")
	case config.TaskID == "":
		return nil, errors.New("log spool task ID is required")
	case config.BaseURL == "":
		return nil, errors.New("runner API URL is required")
	case config.AccessToken == "":
		return nil, errors.New("runner access token is required")
	case config.ChunkBytes > config.MaxBytes:
		return nil, errors.New("log chunk size exceeds spool size")
	case config.MaxBytes < int64(len(droppedRecord)):
		return nil, errors.New("log spool size cannot hold the dropped-log marker")
	}

	taskHash := sha256.Sum256([]byte(config.TaskID))
	taskDirectory := filepath.Join(
		config.Directory,
		hex.EncodeToString(taskHash[:]),
	)
	if err := os.MkdirAll(taskDirectory, 0o700); err != nil {
		return nil, fmt.Errorf("create log spool directory: %w", err)
	}
	config.Directory = taskDirectory

	spool := &Spool{
		config: config,
		wake:   make(chan struct{}, 1),
		done:   make(chan struct{}),
	}
	go spool.uploadLoop()
	return spool, nil
}

func (s *Spool) Write(content []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, io.ErrClosedPipe
	}
	if s.stopped {
		return len(content), nil
	}
	if s.terminalErr != nil {
		return len(content), nil
	}

	originalLength := len(content)
	for len(content) > 0 && !s.dropped {
		newline := bytes.IndexByte(content, '\n')
		if newline >= 0 {
			s.appendLineContentLocked(content[:newline])
			if len(s.line) > 0 {
				s.appendLineLocked()
				s.line = s.line[:0]
			}
			content = content[newline+1:]
			continue
		}

		s.appendLineContentLocked(content)
		content = nil
	}
	return originalLength, nil
}

func (s *Spool) appendLineContentLocked(content []byte) {
	lineLimit := int(s.config.ChunkBytes)
	for len(content) > 0 && !s.dropped {
		remaining := lineLimit - len(s.line)
		if remaining > len(content) {
			remaining = len(content)
		}
		s.line = append(s.line, content[:remaining]...)
		content = content[remaining:]
		if len(s.line) == lineLimit {
			s.appendLineLocked()
			s.line = s.line[:0]
		}
	}
}

func (s *Spool) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return s.terminalErr
	}
	if s.stopped {
		s.closed = true
		return s.terminalErr
	}
	if len(s.line) > 0 && !s.dropped {
		s.appendLineLocked()
		s.line = nil
	}
	if s.current != nil {
		s.sealCurrentLocked()
	}
	s.closed = true
	s.signalUploaderLocked()
	s.finishIfDoneLocked()
	return s.terminalErr
}

func (s *Spool) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.done:
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.terminalErr
	}
}

func (s *Spool) PeakBytes() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.peakBytes
}

func (s *Spool) appendLineLocked() {
	line := s.line
	maxUploadBytes := runnerapi.LogUploadPolicy{
		TargetChunkBytes: s.config.ChunkBytes,
	}.MaxUploadBytes()
	for len(line) > 0 && !s.dropped {
		record := encodeRecord(line)
		if len(record) == 0 {
			return
		}
		if int64(len(record)) <= maxUploadBytes {
			s.appendRecordLocked(record, false)
			return
		}
		prefix := largestEncodedPrefixWithin(line, maxUploadBytes)
		if prefix == 0 {
			s.failLocked(errors.New("log upload limit cannot hold one encoded log record"))
			return
		}
		prefixRecord := encodeRecord(line[:prefix])
		if len(prefixRecord) > 0 {
			s.appendRecordLocked(prefixRecord, false)
		}
		line = line[prefix:]
	}
}

func largestEncodedPrefixWithin(line []byte, limit int64) int {
	best := 0
	for low, high := 1, len(line); low <= high; {
		middle := low + (high-low)/2
		if int64(len(encodeRecord(line[:middle]))) <= limit {
			best = middle
			low = middle + 1
			continue
		}
		high = middle - 1
	}
	return best
}

func (s *Spool) appendRecordLocked(record []byte, isDroppedMarker bool) {
	if s.terminalErr != nil {
		return
	}
	limit := s.config.MaxBytes
	if !isDroppedMarker {
		limit -= int64(len(droppedRecord))
	}
	if s.diskBytes+int64(len(record)) > limit {
		if !s.dropped {
			s.dropped = true
			s.appendRecordLocked(droppedRecord, true)
		}
		return
	}
	if s.current != nil &&
		s.currentBytes > 0 &&
		s.currentBytes+int64(len(record)) > s.config.ChunkBytes {
		s.sealCurrentLocked()
	}
	if s.terminalErr != nil {
		return
	}
	if s.current == nil {
		s.openCurrentLocked()
	}
	if s.terminalErr != nil {
		return
	}
	if _, err := s.current.Write(record); err != nil {
		s.failLocked(fmt.Errorf("write log spool chunk: %w", err))
		return
	}
	s.currentBytes += int64(len(record))
	s.diskBytes += int64(len(record))
	if s.diskBytes > s.peakBytes {
		s.peakBytes = s.diskBytes
	}
}

func (s *Spool) openCurrentLocked() {
	path := filepath.Join(
		s.config.Directory,
		fmt.Sprintf("%020d.ndjson.tmp", s.nextSequence),
	)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		s.failLocked(fmt.Errorf("create log spool chunk: %w", err))
		return
	}
	s.current = file
	s.currentPath = path
	s.currentBytes = 0
}

func (s *Spool) sealCurrentLocked() {
	if s.current == nil {
		return
	}
	if err := s.current.Sync(); err != nil {
		s.failLocked(fmt.Errorf("sync log spool chunk: %w", err))
		return
	}
	if err := s.current.Close(); err != nil {
		s.failLocked(fmt.Errorf("close log spool chunk: %w", err))
		return
	}
	sealedPath := strings.TrimSuffix(s.currentPath, ".tmp")
	if err := os.Rename(s.currentPath, sealedPath); err != nil {
		s.failLocked(fmt.Errorf("seal log spool chunk: %w", err))
		return
	}
	s.current = nil
	s.currentPath = ""
	s.currentBytes = 0
	s.nextSequence++
	s.signalUploaderLocked()
}

func (s *Spool) uploadLoop() {
	defer close(s.done)
	flushTimer := time.NewTimer(s.nextFlushInterval())
	defer flushTimer.Stop()
	for {
		path, size, complete := s.nextUpload()
		if complete {
			return
		}
		if path == "" {
			select {
			case <-s.wake:
			case <-flushTimer.C:
				s.sealOpenChunk()
				flushTimer.Reset(s.nextFlushInterval())
			}
			continue
		}
		outcome, err := s.upload(path)
		if err != nil {
			delay := s.retryDelay()
			var retryErr *uploadRetryError
			if errors.As(err, &retryErr) && retryErr.delay > 0 {
				delay = retryErr.delay
			}
			time.Sleep(delay)
			continue
		}
		if s.config.Log != nil {
			s.config.Log.Info(
				"task log chunk uploaded",
				slog.String("task_id", s.config.TaskID),
				slog.Int64("sequence", s.uploadSequence),
				slog.Int64("bytes", size),
			)
		}
		if err := os.Remove(path); err != nil {
			s.mu.Lock()
			s.failLocked(fmt.Errorf("remove acknowledged log spool chunk: %w", err))
			s.mu.Unlock()
			return
		}
		s.mu.Lock()
		s.diskBytes -= size
		s.uploadSequence++
		if outcome.policy != nil {
			s.applyPolicyLocked(*outcome.policy)
		}
		if outcome.stop {
			s.stopLocked()
			s.mu.Unlock()
			return
		}
		s.mu.Unlock()
		if outcome.policy != nil {
			if !flushTimer.Stop() {
				select {
				case <-flushTimer.C:
				default:
				}
			}
			flushTimer.Reset(s.nextFlushInterval())
		}
	}
}

func (s *Spool) nextFlushInterval() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	minimum := s.config.FlushIntervalMin
	maximum := s.config.FlushIntervalMax
	if minimum <= 0 {
		minimum = defaultMinimumFlushInterval
	}
	if maximum < minimum {
		maximum = defaultMaximumFlushInterval
	}
	if maximum < minimum {
		maximum = minimum
	}
	span := maximum - minimum
	return minimum +
		time.Duration(rand.Int64N(int64(span)+1))
}

func (s *Spool) retryDelay() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.config.RetryDelay
}

func (s *Spool) sealOpenChunk() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current != nil && s.currentBytes > 0 {
		s.sealCurrentLocked()
	}
}

func (s *Spool) nextUpload() (string, int64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.terminalErr != nil || s.stopped {
		return "", 0, true
	}
	path := filepath.Join(
		s.config.Directory,
		fmt.Sprintf("%020d.ndjson", s.uploadSequence),
	)
	info, err := os.Stat(path)
	if err == nil {
		return path, info.Size(), false
	}
	if !errors.Is(err, os.ErrNotExist) {
		s.failLocked(fmt.Errorf("inspect log spool chunk: %w", err))
		return "", 0, true
	}
	if s.closed && s.uploadSequence == s.nextSequence {
		return "", 0, true
	}
	return "", 0, false
}

type uploadOutcome struct {
	stop   bool
	policy *runnerapi.LogUploadPolicy
}

type uploadRetryError struct {
	delay time.Duration
	err   error
}

func (e *uploadRetryError) Error() string {
	return e.err.Error()
}

func (e *uploadRetryError) Unwrap() error {
	return e.err
}

func (s *Spool) upload(path string) (uploadOutcome, error) {
	file, err := os.Open(path)
	if err != nil {
		return uploadOutcome{}, fmt.Errorf("open log spool chunk: %w", err)
	}
	defer file.Close()

	endpoint := s.config.BaseURL +
		"/runner/v1/tasks/" + url.PathEscape(s.config.TaskID) +
		"/logs/chunks/" + strconv.FormatInt(s.uploadSequence, 10)
	request, err := http.NewRequest(http.MethodPut, endpoint, file)
	if err != nil {
		return uploadOutcome{}, fmt.Errorf("create log upload request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+s.config.AccessToken)
	request.Header.Set("Content-Type", "application/x-ndjson")

	response, err := s.config.HTTPClient.Do(request)
	if err != nil {
		return uploadOutcome{}, fmt.Errorf("upload log chunk %d: %w", s.uploadSequence, err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4*1024))
	if response.StatusCode != http.StatusNoContent {
		responseErr := fmt.Errorf(
			"upload log chunk %d: status %d",
			s.uploadSequence,
			response.StatusCode,
		)
		return uploadOutcome{}, &uploadRetryError{
			delay: retryAfter(response.Header.Get("Retry-After")),
			err:   responseErr,
		}
	}
	return uploadOutcome{
		stop:   strings.EqualFold(response.Header.Get(runnerapi.HeaderLogUploadAction), runnerapi.LogUploadActionStop),
		policy: policyFromHeaders(response.Header),
	}, nil
}

func policyFromHeaders(header http.Header) *runnerapi.LogUploadPolicy {
	target, targetErr := strconv.ParseInt(header.Get(runnerapi.HeaderLogTargetChunkBytes), 10, 64)
	minimum, minimumErr := strconv.ParseInt(header.Get(runnerapi.HeaderLogFlushMinimumMS), 10, 64)
	maximum, maximumErr := strconv.ParseInt(header.Get(runnerapi.HeaderLogFlushMaximumMS), 10, 64)
	if targetErr != nil || minimumErr != nil || maximumErr != nil {
		return nil
	}
	policy := runnerapi.NormalizeLogUploadPolicy(&runnerapi.LogUploadPolicy{
		TargetChunkBytes:      target,
		PartialFlushMinimumMS: minimum,
		PartialFlushMaximumMS: maximum,
	})
	return &policy
}

func retryAfter(value string) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(value); err == nil {
		if delay := time.Until(at); delay > 0 {
			return delay
		}
	}
	return 0
}

func (s *Spool) applyPolicyLocked(policy runnerapi.LogUploadPolicy) {
	s.config.ChunkBytes = policy.TargetChunkBytes
	s.config.FlushIntervalMin = time.Duration(policy.PartialFlushMinimumMS) * time.Millisecond
	s.config.FlushIntervalMax = time.Duration(policy.PartialFlushMaximumMS) * time.Millisecond
	if s.current != nil && s.currentBytes >= s.config.ChunkBytes {
		s.sealCurrentLocked()
	}
}

func (s *Spool) stopLocked() {
	s.stopped = true
	s.line = nil
	if s.current != nil {
		_ = s.current.Close()
		_ = os.Remove(s.currentPath)
		s.current = nil
		s.currentPath = ""
		s.currentBytes = 0
	}
	entries, err := os.ReadDir(s.config.Directory)
	if err != nil {
		s.failLocked(fmt.Errorf("read stopped log spool: %w", err))
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if err := os.Remove(filepath.Join(s.config.Directory, entry.Name())); err != nil &&
			!errors.Is(err, os.ErrNotExist) {
			s.failLocked(fmt.Errorf("remove stopped log spool chunk: %w", err))
			return
		}
	}
	s.diskBytes = 0
}

func (s *Spool) signalUploaderLocked() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *Spool) failLocked(err error) {
	if s.terminalErr == nil {
		s.terminalErr = err
	}
	if s.current != nil {
		_ = s.current.Close()
		s.current = nil
	}
	s.signalUploaderLocked()
}

func (s *Spool) finishIfDoneLocked() {
	if s.terminalErr != nil {
		s.signalUploaderLocked()
	}
}
