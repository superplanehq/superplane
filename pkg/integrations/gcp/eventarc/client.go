package eventarc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/integrations/gcp/common"
)

const baseURL = "https://eventarc.googleapis.com/v1"

// --- MessageBus ---

func CreateMessageBus(ctx context.Context, client *common.Client, projectID, region, busID string) (string, error) {
	url := fmt.Sprintf("%s/projects/%s/locations/%s/messageBuses?messageBusId=%s", baseURL, projectID, region, busID)
	raw, err := json.Marshal(map[string]string{"displayName": busID})
	if err != nil {
		return "", fmt.Errorf("marshal message bus body: %w", err)
	}
	resp, err := client.ExecRequest(ctx, "POST", url, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	return operationName(resp)
}

func GetMessageBus(ctx context.Context, client *common.Client, projectID, region, busID string) error {
	url := fmt.Sprintf("%s/projects/%s/locations/%s/messageBuses/%s", baseURL, projectID, region, busID)
	_, err := client.ExecRequest(ctx, "GET", url, nil)
	return err
}

func DeleteMessageBus(ctx context.Context, client *common.Client, projectID, region, busID string) error {
	url := fmt.Sprintf("%s/projects/%s/locations/%s/messageBuses/%s", baseURL, projectID, region, busID)
	_, err := client.ExecRequest(ctx, "DELETE", url, nil)
	return err
}

// --- GoogleApiSource ---

func CreateGoogleAPISource(ctx context.Context, client *common.Client, projectID, region, sourceID, busFullName string) (string, error) {
	url := fmt.Sprintf("%s/projects/%s/locations/%s/googleApiSources?googleApiSourceId=%s", baseURL, projectID, region, sourceID)
	raw, err := json.Marshal(map[string]string{
		"destination": busFullName,
		"displayName": sourceID,
	})
	if err != nil {
		return "", fmt.Errorf("marshal google api source body: %w", err)
	}
	resp, err := client.ExecRequest(ctx, "POST", url, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	return operationName(resp)
}

func GetGoogleAPISource(ctx context.Context, client *common.Client, projectID, region, sourceID string) error {
	url := fmt.Sprintf("%s/projects/%s/locations/%s/googleApiSources/%s", baseURL, projectID, region, sourceID)
	_, err := client.ExecRequest(ctx, "GET", url, nil)
	return err
}

func DeleteGoogleAPISource(ctx context.Context, client *common.Client, projectID, region, sourceID string) error {
	url := fmt.Sprintf("%s/projects/%s/locations/%s/googleApiSources/%s", baseURL, projectID, region, sourceID)
	_, err := client.ExecRequest(ctx, "DELETE", url, nil)
	return err
}

// --- Pipeline ---

type pipelineRequest struct {
	DisplayName        string                `json:"displayName"`
	Destinations       []pipelineDestination `json:"destinations"`
	InputPayloadFormat *payloadFormat        `json:"inputPayloadFormat,omitempty"`
	RetryPolicy        *retryPolicy          `json:"retryPolicy,omitempty"`
}

type pipelineDestination struct {
	HTTPEndpoint         *httpEndpoint         `json:"httpEndpoint,omitempty"`
	AuthenticationConfig *authenticationConfig `json:"authenticationConfig,omitempty"`
}

type httpEndpoint struct {
	URI string `json:"uri"`
}

type authenticationConfig struct {
	GoogleOidc *oidcToken `json:"googleOidc,omitempty"`
}

type oidcToken struct {
	ServiceAccount string `json:"serviceAccount"`
	Audience       string `json:"audience,omitempty"`
}

type payloadFormat struct {
	JSON *struct{} `json:"json,omitempty"`
}

type retryPolicy struct {
	MaxAttempts   int    `json:"maxAttempts"`
	MinRetryDelay string `json:"minRetryDelay"`
	MaxRetryDelay string `json:"maxRetryDelay"`
}

func CreatePipeline(ctx context.Context, client *common.Client, projectID, region, pipelineID, webhookURL, serviceAccountEmail string) (string, error) {
	url := fmt.Sprintf("%s/projects/%s/locations/%s/pipelines?pipelineId=%s", baseURL, projectID, region, pipelineID)
	req := pipelineRequest{
		DisplayName: pipelineID,
		Destinations: []pipelineDestination{
			{
				HTTPEndpoint: &httpEndpoint{URI: webhookURL},
				AuthenticationConfig: &authenticationConfig{
					GoogleOidc: &oidcToken{
						ServiceAccount: serviceAccountEmail,
						Audience:       webhookURL,
					},
				},
			},
		},
		InputPayloadFormat: &payloadFormat{JSON: &struct{}{}},
		RetryPolicy: &retryPolicy{
			MaxAttempts:   5,
			MinRetryDelay: "5s",
			MaxRetryDelay: "60s",
		},
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal pipeline body: %w", err)
	}
	resp, err := client.ExecRequest(ctx, "POST", url, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	return operationName(resp)
}

func DeletePipeline(ctx context.Context, client *common.Client, projectID, region, pipelineID string) error {
	url := fmt.Sprintf("%s/projects/%s/locations/%s/pipelines/%s", baseURL, projectID, region, pipelineID)
	_, err := client.ExecRequest(ctx, "DELETE", url, nil)
	return err
}

// --- Enrollment ---

type enrollmentRequest struct {
	DisplayName string `json:"displayName"`
	CelMatch    string `json:"celMatch"`
	MessageBus  string `json:"messageBus"`
	Destination string `json:"destination"`
}

func CreateEnrollment(ctx context.Context, client *common.Client, projectID, region, enrollmentID, busFullName, pipelineFullName, celFilter string) (string, error) {
	url := fmt.Sprintf("%s/projects/%s/locations/%s/enrollments?enrollmentId=%s", baseURL, projectID, region, enrollmentID)
	req := enrollmentRequest{
		DisplayName: enrollmentID,
		CelMatch:    celFilter,
		MessageBus:  busFullName,
		Destination: pipelineFullName,
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal enrollment body: %w", err)
	}
	resp, err := client.ExecRequest(ctx, "POST", url, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	return operationName(resp)
}

func DeleteEnrollment(ctx context.Context, client *common.Client, projectID, region, enrollmentID string) error {
	url := fmt.Sprintf("%s/projects/%s/locations/%s/enrollments/%s", baseURL, projectID, region, enrollmentID)
	_, err := client.ExecRequest(ctx, "DELETE", url, nil)
	return err
}

// --- Operations ---

type operationResponse struct {
	Name  string          `json:"name"`
	Done  bool            `json:"done"`
	Error *operationError `json:"error,omitempty"`
}

type operationError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func PollOperation(ctx context.Context, client *common.Client, opName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	interval := 3 * time.Second

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("operation %s timed out after %s", opName, timeout)
		}

		url := fmt.Sprintf("%s/%s", baseURL, opName)
		resp, err := client.ExecRequest(ctx, "GET", url, nil)
		if err != nil {
			return fmt.Errorf("poll operation %s: %w", opName, err)
		}

		var op operationResponse
		if err := json.Unmarshal(resp, &op); err != nil {
			return fmt.Errorf("parse operation response: %w", err)
		}

		if op.Done {
			if op.Error != nil {
				return fmt.Errorf("operation %s failed: %s", opName, op.Error.Message)
			}
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}

// --- Helpers ---

func operationName(resp []byte) (string, error) {
	var op struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(resp, &op); err != nil {
		return "", fmt.Errorf("parse operation response: %w", err)
	}
	if op.Name == "" {
		return "", fmt.Errorf("operation response missing name")
	}
	return op.Name, nil
}

func MessageBusFullName(projectID, region, busID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/messageBuses/%s", projectID, region, busID)
}

func PipelineFullName(projectID, region, pipelineID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/pipelines/%s", projectID, region, pipelineID)
}

func IsAlreadyExistsError(err error) bool {
	return common.IsAlreadyExistsError(err)
}

func IsNotFoundError(err error) bool {
	return common.IsNotFoundError(err)
}
