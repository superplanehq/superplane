package cloudwatch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	cloudWatchServiceName = "monitoring"
	cloudWatchAPIVersion  = "2010-08-01"
	cloudWatchContentType = "application/x-www-form-urlencoded; charset=utf-8"
)

type Client struct {
	http        core.HTTPContext
	region      string
	credentials *aws.Credentials
	signer      *v4.Signer
}

type PutMetricDataResponse struct {
	RequestID string `xml:"ResponseMetadata>RequestId"`
}

type MetricDatum struct {
	MetricName        string
	Value             float64
	Unit              string
	Timestamp         *time.Time
	StorageResolution *int
	Dimensions        []Dimension
}

type Dimension struct {
	Name  string
	Value string
}

func NewClient(httpCtx core.HTTPContext, credentials *aws.Credentials, region string) *Client {
	return &Client{
		http:        httpCtx,
		region:      region,
		credentials: credentials,
		signer:      v4.NewSigner(),
	}
}

func (c *Client) PutMetricData(namespace string, metricData []MetricDatum) (*PutMetricDataResponse, error) {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	if len(metricData) == 0 {
		return nil, fmt.Errorf("at least one metric datum is required")
	}

	values := url.Values{}
	values.Set("Action", "PutMetricData")
	values.Set("Version", cloudWatchAPIVersion)
	values.Set("Namespace", namespace)

	for i, metric := range metricData {
		metricIndex := i + 1
		metricName := strings.TrimSpace(metric.MetricName)
		if metricName == "" {
			return nil, fmt.Errorf("metricData[%d].metricName is required", i)
		}

		values.Set(fmt.Sprintf("MetricData.member.%d.MetricName", metricIndex), metricName)
		values.Set(fmt.Sprintf("MetricData.member.%d.Value", metricIndex), strconv.FormatFloat(metric.Value, 'f', -1, 64))

		unit := strings.TrimSpace(metric.Unit)
		if unit != "" {
			values.Set(fmt.Sprintf("MetricData.member.%d.Unit", metricIndex), unit)
		}

		if metric.Timestamp != nil {
			values.Set(
				fmt.Sprintf("MetricData.member.%d.Timestamp", metricIndex),
				metric.Timestamp.UTC().Format(time.RFC3339),
			)
		}

		if metric.StorageResolution != nil {
			values.Set(
				fmt.Sprintf("MetricData.member.%d.StorageResolution", metricIndex),
				strconv.Itoa(*metric.StorageResolution),
			)
		}

		for j, dimension := range metric.Dimensions {
			dimensionIndex := j + 1
			name := strings.TrimSpace(dimension.Name)
			value := strings.TrimSpace(dimension.Value)
			if name == "" {
				return nil, fmt.Errorf("metricData[%d].dimensions[%d].name is required", i, j)
			}
			if value == "" {
				return nil, fmt.Errorf("metricData[%d].dimensions[%d].value is required", i, j)
			}

			values.Set(fmt.Sprintf("MetricData.member.%d.Dimensions.member.%d.Name", metricIndex, dimensionIndex), name)
			values.Set(fmt.Sprintf("MetricData.member.%d.Dimensions.member.%d.Value", metricIndex, dimensionIndex), value)
		}
	}

	response := PutMetricDataResponse{}
	if err := c.postForm(values, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) postForm(values url.Values, out any) error {
	body := values.Encode()
	endpoint := fmt.Sprintf("https://monitoring.%s.amazonaws.com/", c.region)
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", cloudWatchContentType)
	req.Header.Set("Accept", "application/xml")

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

		return fmt.Errorf("CloudWatch API request failed with %d: %s", res.StatusCode, string(responseBody))
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
	return c.signer.SignHTTP(context.Background(), *c.credentials, req, payloadHash, cloudWatchServiceName, c.region, time.Now())
}

func parseError(body []byte) *common.Error {
	var payload struct {
		Error struct {
			Code    string `xml:"Code"`
			Message string `xml:"Message"`
		} `xml:"Error"`
		Errors struct {
			Error struct {
				Code    string `xml:"Code"`
				Message string `xml:"Message"`
			} `xml:"Error"`
		} `xml:"Errors"`
	}

	if err := xml.Unmarshal(body, &payload); err != nil {
		return nil
	}

	code := strings.TrimSpace(payload.Error.Code)
	message := strings.TrimSpace(payload.Error.Message)
	if code == "" && message == "" {
		code = strings.TrimSpace(payload.Errors.Error.Code)
		message = strings.TrimSpace(payload.Errors.Error.Message)
	}

	if code == "" && message == "" {
		return nil
	}

	return &common.Error{
		Code:    code,
		Message: message,
	}
}
