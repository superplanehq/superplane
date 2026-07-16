package runner

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	gojwt "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	LiveLogStreamTokenPurpose  = "runner_live_logs"
	LiveLogStreamTokenAudience = "task_broker"
	liveLogStreamTokenTTL      = 5 * time.Minute
)

var (
	ErrLiveLogCanvasNotFound    = errors.New("canvas not found")
	ErrLiveLogExecutionNotFound = errors.New("execution not found")
	ErrLiveLogNodeNotFound      = errors.New("node not found")
	ErrLiveLogNotRunner         = errors.New("not a runner component")
	ErrLiveLogBrokerTaskMissing = errors.New("broker task id missing")
	ErrLiveLogNotConfigured     = errors.New("live logs not configured")
)

// LiveLogStreamTokenClaims is the JWT payload SuperPlane mints for browser → task-broker streaming.
type LiveLogStreamTokenClaims struct {
	TaskID  string `json:"task_id"`
	Purpose string `json:"purpose"`
	gojwt.RegisteredClaims
}

// LiveLogSession is returned to the browser after SuperPlane authorizes log access.
type LiveLogSession struct {
	StreamURL string    `json:"stream_url"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// LiveLogAccessContext is the resolved runner execution context for live log access.
type LiveLogAccessContext struct {
	BrokerTaskID string
}

func IsRunnerComponent(name string) bool {
	switch strings.TrimSpace(name) {
	case ComponentName, RunJSComponentName, RunPythonComponentName, RunBashComponentName, RunClaudeCodeComponentName:
		return true
	default:
		return false
	}
}

func BrokerTaskIDFromExecutionMetadata(meta map[string]any) string {
	if meta == nil {
		return ""
	}
	v, ok := meta[ExecutionMetadataBrokerTaskID]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func ResolveLiveLogAccess(orgID uuid.UUID, canvasID uuid.UUID, executionID uuid.UUID) (*LiveLogAccessContext, error) {
	if _, err := models.FindCanvas(orgID, canvasID); err != nil {
		return nil, ErrLiveLogCanvasNotFound
	}

	execution, err := models.FindNodeExecution(canvasID, executionID)
	if err != nil {
		return nil, ErrLiveLogExecutionNotFound
	}

	node, err := models.FindCanvasNode(database.Conn(), canvasID, execution.NodeID)
	if err != nil {
		return nil, ErrLiveLogNodeNotFound
	}

	ref := node.Ref.Data()
	if ref.Component == nil || !IsRunnerComponent(ref.Component.Name) {
		return nil, ErrLiveLogNotRunner
	}

	brokerTaskID := BrokerTaskIDFromExecutionMetadata(execution.Metadata.Data())
	if brokerTaskID == "" {
		return nil, ErrLiveLogBrokerTaskMissing
	}

	return &LiveLogAccessContext{BrokerTaskID: brokerTaskID}, nil
}

func taskBrokerBaseURL() (string, error) {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_BROKER_BASE_URL")), "/")
	if base == "" {
		return "", fmt.Errorf("%w: TASK_BROKER_BASE_URL is not set", ErrLiveLogNotConfigured)
	}
	return base, nil
}

func LiveLogStreamURL(brokerTaskID string) (string, error) {
	base, err := taskBrokerBaseURL()
	if err != nil {
		return "", err
	}
	brokerTaskID = strings.TrimSpace(brokerTaskID)
	if brokerTaskID == "" {
		return "", fmt.Errorf("broker task id is empty")
	}
	return base + "/v1/tasks/" + brokerTaskID + "/live-logs", nil
}

func taskBrokerAuthToken() (string, error) {
	token := strings.TrimSpace(os.Getenv("TASK_BROKER_AUTH_TOKEN"))
	if token == "" {
		return "", fmt.Errorf("%w: TASK_BROKER_AUTH_TOKEN is not set", ErrLiveLogNotConfigured)
	}
	return token, nil
}

func MintLiveLogStreamToken(brokerTaskID string, now time.Time) (string, time.Time, error) {
	brokerTaskID = strings.TrimSpace(brokerTaskID)
	if brokerTaskID == "" {
		return "", time.Time{}, fmt.Errorf("broker task id is empty")
	}

	secret, err := taskBrokerAuthToken()
	if err != nil {
		return "", time.Time{}, err
	}

	expiresAt := now.Add(liveLogStreamTokenTTL)
	claims := LiveLogStreamTokenClaims{
		TaskID:  brokerTaskID,
		Purpose: LiveLogStreamTokenPurpose,
		RegisteredClaims: gojwt.RegisteredClaims{
			Audience:  gojwt.ClaimStrings{LiveLogStreamTokenAudience},
			ExpiresAt: gojwt.NewNumericDate(expiresAt),
			IssuedAt:  gojwt.NewNumericDate(now),
			NotBefore: gojwt.NewNumericDate(now.Add(-time.Minute)),
		},
	}

	token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign live log token: %w", err)
	}

	return tokenString, expiresAt, nil
}

func NewLiveLogSession(brokerTaskID string, now time.Time) (*LiveLogSession, error) {
	streamURL, err := LiveLogStreamURL(brokerTaskID)
	if err != nil {
		return nil, err
	}

	token, expiresAt, err := MintLiveLogStreamToken(brokerTaskID, now)
	if err != nil {
		return nil, err
	}

	return &LiveLogSession{
		StreamURL: streamURL,
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

// ValidateLiveLogStreamToken validates a browser stream token and ensures it matches the requested task.
func ValidateLiveLogStreamToken(tokenString, brokerTaskID, secret string) error {
	brokerTaskID = strings.TrimSpace(brokerTaskID)
	if brokerTaskID == "" {
		return fmt.Errorf("broker task id is empty")
	}
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return fmt.Errorf("jwt secret is empty")
	}

	claims := &LiveLogStreamTokenClaims{}
	token, err := gojwt.ParseWithClaims(tokenString, claims, func(token *gojwt.Token) (any, error) {
		if _, ok := token.Method.(*gojwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return err
	}
	if !token.Valid {
		return fmt.Errorf("invalid token")
	}
	if claims.Purpose != LiveLogStreamTokenPurpose {
		return fmt.Errorf("invalid purpose")
	}
	if !claims.VerifyAudience(LiveLogStreamTokenAudience, true) {
		return fmt.Errorf("invalid audience")
	}
	if strings.TrimSpace(claims.TaskID) != brokerTaskID {
		return fmt.Errorf("task id mismatch")
	}
	return nil
}
