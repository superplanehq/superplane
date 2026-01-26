package lambda

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	http   core.HTTPContext
	region string
	creds  aws.Credentials
	signer *v4.Signer
}

type InvokeResult struct {
	FunctionError string
	LogResult     string
	RequestID     string
	Payload       []byte
}

type FunctionSummary struct {
	FunctionName string `json:"FunctionName"`
	FunctionArn  string `json:"FunctionArn"`
}

type listFunctionsResponse struct {
	Functions  []FunctionSummary `json:"Functions"`
	NextMarker string            `json:"NextMarker"`
}

func NewClient(httpCtx core.HTTPContext, creds aws.Credentials, region string) *Client {
	return &Client{
		http:   httpCtx,
		region: region,
		creds:  creds,
		signer: v4.NewSigner(),
	}
}

func (c *Client) Invoke(functionArn string, payload []byte) (*InvokeResult, error) {
	endpoint := fmt.Sprintf("https://lambda.%s.amazonaws.com/2015-03-31/functions/%s/invocations", c.region, url.PathEscape(functionArn))
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to build invoke request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Amz-Invocation-Type", "RequestResponse")
	req.Header.Set("X-Amz-Log-Type", "Tail")

	if err := c.signRequest(req, payload); err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("invoke request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read invoke response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("invoke failed with %d: %s", res.StatusCode, string(body))
	}
	return &InvokeResult{
		RequestID:     res.Header.Get("X-Amzn-Requestid"),
		LogResult:     res.Header.Get("X-Amz-Log-Result"),
		FunctionError: res.Header.Get("X-Amz-Function-Error"),
		Payload:       body,
	}, nil
}

func (c *Client) ListFunctions() ([]FunctionSummary, error) {
	var (
		functions []FunctionSummary
		marker    string
	)

	for {
		endpoint := fmt.Sprintf("https://lambda.%s.amazonaws.com/2015-03-31/functions", c.region)
		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to build list functions request: %w", err)
		}

		query := req.URL.Query()
		query.Set("MaxItems", "50")
		if strings.TrimSpace(marker) != "" {
			query.Set("Marker", marker)
		}
		req.URL.RawQuery = query.Encode()

		if err := c.signRequest(req, []byte{}); err != nil {
			return nil, err
		}

		res, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("list functions request failed: %w", err)
		}

		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read list functions response: %w", err)
		}

		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return nil, fmt.Errorf("list functions failed with %d: %s", res.StatusCode, string(body))
		}

		var response listFunctionsResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to decode list functions response: %w", err)
		}

		functions = append(functions, response.Functions...)

		if strings.TrimSpace(response.NextMarker) == "" {
			break
		}
		marker = response.NextMarker
	}

	return functions, nil
}

func (c *Client) signRequest(req *http.Request, payload []byte) error {
	hash := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(hash[:])
	return c.signer.SignHTTP(context.Background(), c.creds, req, payloadHash, "lambda", c.region, time.Now())
}
