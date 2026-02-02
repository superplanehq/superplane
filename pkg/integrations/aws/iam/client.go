package iam

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	serviceName   = "iam"
	defaultRegion = "us-east-1"
	apiVersion    = "2010-05-08"
	endpoint      = "https://iam.amazonaws.com/"
	contentType   = "application/x-www-form-urlencoded; charset=utf-8"
)

type Client struct {
	http   core.HTTPContext
	region string
	creds  aws.Credentials
	signer *v4.Signer
}

func NewClient(httpCtx core.HTTPContext, creds aws.Credentials) *Client {
	return &Client{
		http:   httpCtx,
		region: defaultRegion,
		creds:  creds,
		signer: v4.NewSigner(),
	}
}

func IsEntityAlreadyExistsErr(err error) bool {
	var awsErr *common.Error
	if errors.As(err, &awsErr) {
		return strings.Contains(awsErr.Code, "EntityAlreadyExists")
	}
	return false
}

func (c *Client) CreateRole(name, assumeRolePolicy string, tags []common.Tag) (string, error) {
	params := map[string]string{
		"RoleName":                 name,
		"AssumeRolePolicyDocument": assumeRolePolicy,
	}

	if len(tags) > 0 {
		for i, tag := range tags {
			index := i + 1
			params[fmt.Sprintf("Tags.member.%d.Key", index)] = tag.Key
			params[fmt.Sprintf("Tags.member.%d.Value", index)] = tag.Value
		}
	}

	var response struct {
		Arn string `xml:"CreateRoleResult>Role>Arn"`
	}

	if err := c.postForm("CreateRole", params, &response); err != nil {
		return "", err
	}

	return response.Arn, nil
}

func (c *Client) GetRole(name string) (string, error) {
	params := map[string]string{
		"RoleName": name,
	}

	var response struct {
		Arn string `xml:"GetRoleResult>Role>Arn"`
	}

	if err := c.postForm("GetRole", params, &response); err != nil {
		return "", err
	}

	return response.Arn, nil
}

func (c *Client) PutRolePolicy(roleName, policyName, policyDocument string) error {
	params := map[string]string{
		"RoleName":       roleName,
		"PolicyName":     policyName,
		"PolicyDocument": policyDocument,
	}

	return c.postForm("PutRolePolicy", params, nil)
}

func (c *Client) postForm(action string, params map[string]string, out any) error {
	values := url.Values{}
	values.Set("Action", action)
	values.Set("Version", apiVersion)
	for key, value := range params {
		values.Set(key, value)
	}

	body := values.Encode()
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)

	if err := c.signRequest(req, []byte(body)); err != nil {
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
		if awsErr := parseError(responseBody); awsErr != nil {
			return awsErr
		}
		return fmt.Errorf("IAM API request failed with %d: %s", res.StatusCode, string(responseBody))
	}

	if out == nil {
		return nil
	}

	if err := xml.Unmarshal(responseBody, out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

func (c *Client) signRequest(req *http.Request, payload []byte) error {
	hash := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(hash[:])
	return c.signer.SignHTTP(context.Background(), c.creds, req, payloadHash, serviceName, c.region, time.Now())
}

func parseError(body []byte) *common.Error {
	var payload struct {
		Error struct {
			Code    string `xml:"Code"`
			Message string `xml:"Message"`
		} `xml:"Error"`
	}

	if err := xml.Unmarshal(body, &payload); err != nil {
		return nil
	}

	if payload.Error.Code == "" && payload.Error.Message == "" {
		return nil
	}

	return &common.Error{
		Code:    strings.TrimSpace(payload.Error.Code),
		Message: strings.TrimSpace(payload.Error.Message),
	}
}
