package ecr

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const targetPrefix = "AmazonEC2ContainerRegistry_V20150921."

type Client struct {
	http        core.HTTPContext
	region      string
	credentials *aws.Credentials
	signer      *v4.Signer
}

func NewClient(httpCtx core.HTTPContext, credentials *aws.Credentials, region string) *Client {
	return &Client{
		http:        httpCtx,
		region:      region,
		credentials: credentials,
		signer:      v4.NewSigner(),
	}
}

func (c *Client) DescribeRepository(name string) (*Repository, error) {
	payload := map[string]any{
		"repositoryNames": []string{name},
	}

	var response struct {
		Repositories []Repository `json:"repositories"`
	}

	if err := c.postJSON("DescribeRepositories", payload, &response); err != nil {
		return nil, err
	}

	if len(response.Repositories) == 0 {
		return nil, fmt.Errorf("repository not found")
	}

	return &response.Repositories[0], nil
}

func (c *Client) ListRepositories() ([]Repository, error) {
	repositories := []Repository{}
	nextToken := ""

	for {
		payload := map[string]any{
			"maxResults": 100,
		}
		if nextToken != "" {
			payload["nextToken"] = nextToken
		}

		var response struct {
			Repositories []Repository `json:"repositories"`
			NextToken    string       `json:"nextToken"`
		}

		if err := c.postJSON("DescribeRepositories", payload, &response); err != nil {
			return nil, err
		}

		repositories = append(repositories, response.Repositories...)
		if response.NextToken == "" {
			break
		}
		nextToken = response.NextToken
	}

	return repositories, nil
}

func (c *Client) postJSON(action string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("https://api.ecr.%s.amazonaws.com/", c.region)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", targetPrefix+action)

	if err := c.signRequest(req, body); err != nil {
		return err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if awsErr := common.ParseError(responseBody); awsErr != nil {
			return awsErr
		}
		return fmt.Errorf("ECR API request failed with %d: %s", res.StatusCode, string(responseBody))
	}

	if out == nil {
		return nil
	}

	if err := json.Unmarshal(responseBody, out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

func (c *Client) signRequest(req *http.Request, payload []byte) error {
	hash := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(hash[:])
	return c.signer.SignHTTP(context.Background(), *c.credentials, req, payloadHash, "ecr", c.region, time.Now())
}
