package aws

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/superplanehq/superplane/pkg/core"
)

type lambdaClient struct {
	http   core.HTTPContext
	region string
	creds  aws.Credentials
	signer *v4.Signer
}

type invokeResult struct {
	StatusCode    int
	FunctionError string
	LogResult     string
	Payload       []byte
}

type createFunctionRequest struct {
	FunctionName   string
	Runtime        string
	Handler        string
	RoleArn        string
	Code           string
	TimeoutSeconds int
	MemoryMB       int
	Description    string
}

type createFunctionResponse struct {
	FunctionArn string `json:"FunctionArn"`
}

func newLambdaClient(httpCtx core.HTTPContext, creds aws.Credentials, region string) *lambdaClient {
	return &lambdaClient{
		http:   httpCtx,
		region: region,
		creds:  creds,
		signer: v4.NewSigner(),
	}
}

func (c *lambdaClient) Invoke(functionArn string, payload []byte, invocationType string) (invokeResult, error) {
	endpoint := fmt.Sprintf("https://lambda.%s.amazonaws.com/2015-03-31/functions/%s/invocations", c.region, url.PathEscape(functionArn))
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return invokeResult{}, fmt.Errorf("failed to build invoke request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Amz-Invocation-Type", invocationType)
	req.Header.Set("X-Amz-Log-Type", "Tail")

	if err := c.signRequest(req, payload); err != nil {
		return invokeResult{}, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return invokeResult{}, fmt.Errorf("invoke request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return invokeResult{}, fmt.Errorf("failed to read invoke response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return invokeResult{}, fmt.Errorf("invoke failed with %d: %s", res.StatusCode, string(body))
	}

	return invokeResult{
		StatusCode:    res.StatusCode,
		FunctionError: res.Header.Get("X-Amz-Function-Error"),
		LogResult:     res.Header.Get("X-Amz-Log-Result"),
		Payload:       body,
	}, nil
}

func (c *lambdaClient) CreateFunction(request createFunctionRequest) (string, error) {
	zipBytes, err := zipFunctionCode(request.Code)
	if err != nil {
		return "", err
	}

	payload := map[string]any{
		"FunctionName": request.FunctionName,
		"Runtime":      request.Runtime,
		"Handler":      request.Handler,
		"Role":         request.RoleArn,
		"Code": map[string]any{
			"ZipFile": base64.StdEncoding.EncodeToString(zipBytes),
		},
		"Timeout":    request.TimeoutSeconds,
		"MemorySize": request.MemoryMB,
	}

	if request.Description != "" {
		payload["Description"] = request.Description
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal create function payload: %w", err)
	}

	endpoint := fmt.Sprintf("https://lambda.%s.amazonaws.com/2015-03-31/functions", c.region)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to build create function request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if err := c.signRequest(req, body); err != nil {
		return "", err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("create function request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read create function response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("create function failed with %d: %s", res.StatusCode, string(responseBody))
	}

	var response createFunctionResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", fmt.Errorf("failed to parse create function response: %w", err)
	}

	if response.FunctionArn == "" {
		return "", fmt.Errorf("create function response missing function ARN")
	}

	return response.FunctionArn, nil
}

func (c *lambdaClient) signRequest(req *http.Request, payload []byte) error {
	hash := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(hash[:])
	return c.signer.SignHTTP(context.Background(), c.creds, req, payloadHash, "lambda", c.region, time.Now())
}

func zipFunctionCode(code string) ([]byte, error) {
	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)

	fileWriter, err := zipWriter.Create("index.js")
	if err != nil {
		return nil, fmt.Errorf("failed to create zip entry: %w", err)
	}

	if _, err := fileWriter.Write([]byte(code)); err != nil {
		return nil, fmt.Errorf("failed to write code to zip: %w", err)
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize zip: %w", err)
	}

	return buffer.Bytes(), nil
}
