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
)

const (
	defaultChunkBytes       = 64 * 1024
	defaultMaxBytes         = 64 * 1024 * 1024
	defaultRetryDelay       = time.Second
	defaultFlushIntervalMin = 2500 * time.Millisecond
	defaultFlushIntervalMax = 5 * time.Second
	maxUploadBytes          = 4 * 1024 * 1024
)

var droppedRecord = []byte(
	`{"type":"line","text":"[Runner logs were dropped because the local log spool reached its limit.]"}` + "\n",
)

type Config struct {
	Directory     string
	TaskID        string
	BaseURL       string
	AccessToken   string
	ChunkBytes    int64
	MaxBytes      int64
	RetryDelay    time.Duration
	FlushInterval time.Duration
	HTTPClient    *http.Client
	Log           *slog.Logger
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
	case config.ChunkBytes > maxUploadBytes:
		return nil, errors.New("log chunk size exceeds the runner API limit")
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
	if s.terminalErr != nil {
		return len(content), nil
	}

	originalLength := len(content)
	for len(content) > 0 && !s.dropped {
		newline := bytes.IndexByte(content, '\n')
		if newline >= 0 {
			s.line = append(s.line, content[:newline]...)
			s.appendLineLocked()
			s.line = s.line[:0]
			content = content[newline+1:]
			continue
		}

		// JSON escaping can expand one input byte to six output bytes.
		lineLimit := int(s.config.ChunkBytes / 8)
		if lineLimit < 1 {
			lineLimit = 1
		}
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
	return originalLength, nil
}

func (s *Spool) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
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
	record := encodeRecord(s.line)
	if len(record) == 0 {
		return
	}
	s.appendRecordLocked(record, false)
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
		if err := s.upload(path); err != nil {
			time.Sleep(s.config.RetryDelay)
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
		s.mu.Unlock()
	}
}

func (s *Spool) nextFlushInterval() time.Duration {
	if s.config.FlushInterval > 0 {
		return s.config.FlushInterval
	}
	span := defaultFlushIntervalMax - defaultFlushIntervalMin
	return defaultFlushIntervalMin +
		time.Duration(rand.Int64N(int64(span)+1))
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
	if s.terminalErr != nil {
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

func (s *Spool) upload(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open log spool chunk: %w", err)
	}
	defer file.Close()

	endpoint := s.config.BaseURL +
		"/runner/v1/tasks/" + url.PathEscape(s.config.TaskID) +
		"/logs/chunks/" + strconv.FormatInt(s.uploadSequence, 10)
	request, err := http.NewRequest(http.MethodPut, endpoint, file)
	if err != nil {
		return fmt.Errorf("create log upload request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+s.config.AccessToken)
	request.Header.Set("Content-Type", "application/x-ndjson")

	response, err := s.config.HTTPClient.Do(request)
	if err != nil {
		return fmt.Errorf("upload log chunk %d: %w", s.uploadSequence, err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4*1024))
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf(
			"upload log chunk %d: status %d",
			s.uploadSequence,
			response.StatusCode,
		)
	}
	return nil
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
