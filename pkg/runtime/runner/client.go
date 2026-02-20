package runner

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	TransportHTTP = "http"
	TransportGRPC = "grpc"

	DefaultHTTPBaseURL = "http://127.0.0.1:7761"
	DefaultGRPCAddress = "127.0.0.1:7762"
	DefaultTimeout     = 30 * time.Second
	DefaultVersion     = "v1"

	RunnerTransportEnv   = "TYPESCRIPT_RUNNER_TRANSPORT"
	RunnerHTTPBaseURLEnv = "TYPESCRIPT_RUNNER_HTTP_BASE_URL"
	RunnerGRPCAddressEnv = "TYPESCRIPT_RUNNER_GRPC_ADDRESS"
	RunnerTimeoutEnv     = "TYPESCRIPT_RUNNER_TIMEOUT"
	RunnerAuthTokenEnv   = "TYPESCRIPT_RUNNER_AUTH_TOKEN"
	RunnerVersionEnv     = "TYPESCRIPT_RUNNER_VERSION"
)

type Config struct {
	Transport string
	HTTPURL   string
	GRPCAddr  string
	Timeout   time.Duration
	AuthToken string
	Version   string
}

type RequestEnvelope struct {
	RequestID string `json:"request_id"`
	Version   string `json:"version"`
	TimeoutMS int64  `json:"timeout_ms"`
}

type RuntimeContext struct {
	OrganizationID string         `json:"organization_id,omitempty"`
	WorkspaceID    string         `json:"workspace_id,omitempty"`
	UserID         string         `json:"user_id,omitempty"`
	CanvasID       string         `json:"canvas_id,omitempty"`
	NodeID         string         `json:"node_id,omitempty"`
	Labels         map[string]any `json:"labels,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type OperationRequest struct {
	Request RequestEnvelope `json:"request"`
	Context RuntimeContext  `json:"context"`
	Input   any             `json:"input"`
}

type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type Log struct {
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields,omitempty"`
}

type OperationResponse struct {
	OK      bool               `json:"ok"`
	Output  map[string]any     `json:"output,omitempty"`
	Logs    []Log              `json:"logs,omitempty"`
	Error   *Error             `json:"error,omitempty"`
	Metrics map[string]float64 `json:"metrics,omitempty"`
}

type Capability struct {
	Kind       string   `json:"kind"`
	Name       string   `json:"name"`
	Operations []string `json:"operations,omitempty"`
	SchemaHash string   `json:"schema_hash,omitempty"`
}

type Client interface {
	SetupTrigger(ctx context.Context, name string, req OperationRequest) (*OperationResponse, error)
	SetupComponent(ctx context.Context, name string, req OperationRequest) (*OperationResponse, error)
	ExecuteComponent(ctx context.Context, name string, req OperationRequest) (*OperationResponse, error)
	SyncIntegration(ctx context.Context, name string, req OperationRequest) (*OperationResponse, error)
	CleanupIntegration(ctx context.Context, name string, req OperationRequest) (*OperationResponse, error)
	ListCapabilities(ctx context.Context) ([]Capability, error)
}

func LoadConfigFromEnv() (Config, error) {
	transport := strings.ToLower(strings.TrimSpace(os.Getenv(RunnerTransportEnv)))
	if transport == "" {
		transport = TransportHTTP
	}

	timeout := DefaultTimeout
	timeoutValue := strings.TrimSpace(os.Getenv(RunnerTimeoutEnv))
	if timeoutValue != "" {
		parsed, err := time.ParseDuration(timeoutValue)
		if err != nil {
			return Config{}, fmt.Errorf("invalid %s: %w", RunnerTimeoutEnv, err)
		}
		if parsed <= 0 {
			return Config{}, fmt.Errorf("%s must be > 0", RunnerTimeoutEnv)
		}
		timeout = parsed
	}

	version := strings.TrimSpace(os.Getenv(RunnerVersionEnv))
	if version == "" {
		version = DefaultVersion
	}

	cfg := Config{
		Transport: transport,
		HTTPURL:   strings.TrimSpace(os.Getenv(RunnerHTTPBaseURLEnv)),
		GRPCAddr:  strings.TrimSpace(os.Getenv(RunnerGRPCAddressEnv)),
		Timeout:   timeout,
		AuthToken: strings.TrimSpace(os.Getenv(RunnerAuthTokenEnv)),
		Version:   version,
	}

	if cfg.HTTPURL == "" {
		cfg.HTTPURL = DefaultHTTPBaseURL
	}
	if cfg.GRPCAddr == "" {
		cfg.GRPCAddr = DefaultGRPCAddress
	}

	switch cfg.Transport {
	case TransportHTTP, TransportGRPC:
		return cfg, nil
	default:
		return Config{}, fmt.Errorf("unsupported runtime runner transport %q", cfg.Transport)
	}
}

func NewClient(cfg Config) (Client, error) {
	switch cfg.Transport {
	case TransportHTTP:
		return newHTTPClient(cfg), nil
	case TransportGRPC:
		return newGRPCClient(cfg)
	default:
		return nil, fmt.Errorf("unsupported runtime runner transport %q", cfg.Transport)
	}
}
