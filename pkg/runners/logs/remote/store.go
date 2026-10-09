package remote

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/runners/logs"
)

const BaseURLEnv = "RUNNER_ACTIVE_LOG_BASE_URL"

var errReadOnly = errors.New("remote active log store is read-only")

var httpClient = &http.Client{Timeout: 15 * time.Second}

type Store struct {
	baseURL string
	secret  string
}

func FromEnvironment(secret string) logs.Store {
	baseURL := strings.TrimSpace(os.Getenv(BaseURLEnv))
	if baseURL == "" || strings.TrimSpace(secret) == "" {
		return nil
	}
	return New(baseURL, secret)
}

func New(baseURL, secret string) *Store {
	return &Store{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		secret:  secret,
	}
}

func (s *Store) Name() string {
	return logs.StoreFS
}

func (s *Store) Setup(logs.SetupContext) error {
	return nil
}

func (s *Store) Initialize(context.Context, uuid.UUID) error {
	return errReadOnly
}

func (s *Store) Append(context.Context, uuid.UUID, int64, []byte) (logs.AppendResult, error) {
	return logs.AppendResult{}, errReadOnly
}

func (s *Store) Delete(context.Context, uuid.UUID) error {
	return errReadOnly
}

func (s *Store) ReadAfter(ctx context.Context, taskID uuid.UUID, cursor string) (*logs.ReadResult, error) {
	token, err := logs.SignActiveLogRead(s.secret, taskID, cursor, time.Now())
	if err != nil {
		return nil, err
	}
	endpoint := s.baseURL + "/internal/v1/tasks/" + taskID.String() + "/logs"
	if cursor != "" {
		endpoint += "?after=" + url.QueryEscape(cursor)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("read active logs: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		content, err := io.ReadAll(io.LimitReader(resp.Body, logs.MaxRetainedBytes+1))
		if err != nil {
			return nil, fmt.Errorf("read active logs: %w", err)
		}
		if int64(len(content)) > logs.MaxRetainedBytes {
			return nil, errors.New("active log read exceeded the retained limit")
		}
		return &logs.ReadResult{
			Content: io.NopCloser(bytes.NewReader(content)),
			Cursor:  resp.Header.Get(logs.HeaderCursor),
		}, nil
	case http.StatusNotFound:
		return nil, logs.ErrNotFound
	case http.StatusBadRequest:
		return nil, logs.ErrInvalidCursor
	default:
		return nil, fmt.Errorf("read active logs: status %d", resp.StatusCode)
	}
}
