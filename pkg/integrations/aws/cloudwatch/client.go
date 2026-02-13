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
	serviceName                        = "monitoring"
	apiVersion                         = "2010-08-01"
	contentType                        = "application/x-www-form-urlencoded; charset=utf-8"
	metricsQueryID                     = "q1"
	defaultMetricsInsightsPeriodSeconds = 60
)

type Client struct {
	http        core.HTTPContext
	region      string
	credentials *aws.Credentials
	signer      *v4.Signer
}

type QueryMetricsInsightsInput struct {
	Query        string
	StartTime    time.Time
	EndTime      time.Time
	ScanBy       string
	MaxDatapoints int
}

type QueryMetricsInsightsOutput struct {
	RequestID string                `json:"requestId"`
	Results   []MetricDataResult    `json:"results"`
	Messages  []MetricDataMessage   `json:"messages,omitempty"`
}

type MetricDataResult struct {
	ID         string              `json:"id" xml:"Id"`
	Label      string              `json:"label" xml:"Label"`
	StatusCode string              `json:"statusCode" xml:"StatusCode"`
	Timestamps []string            `json:"timestamps" xml:"Timestamps>member"`
	Values     []float64           `json:"values" xml:"Values>member"`
	Messages   []MetricDataMessage `json:"messages,omitempty" xml:"Messages>member"`
}

type MetricDataMessage struct {
	Code  string `json:"code" xml:"Code"`
	Value string `json:"value" xml:"Value"`
}

type getMetricDataResponse struct {
	MetricDataResults []MetricDataResult  `xml:"GetMetricDataResult>MetricDataResults>member"`
	Messages          []MetricDataMessage `xml:"GetMetricDataResult>Messages>member"`
	NextToken         string              `xml:"GetMetricDataResult>NextToken"`
	RequestID         string              `xml:"ResponseMetadata>RequestId"`
}

func NewClient(httpCtx core.HTTPContext, credentials *aws.Credentials, region string) *Client {
	return &Client{
		http:        httpCtx,
		region:      strings.TrimSpace(region),
		credentials: credentials,
		signer:      v4.NewSigner(),
	}
}

func (c *Client) QueryMetricsInsights(input QueryMetricsInsightsInput) (*QueryMetricsInsightsOutput, error) {
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	startTime := input.StartTime.UTC()
	if startTime.IsZero() {
		return nil, fmt.Errorf("start time is required")
	}

	endTime := input.EndTime.UTC()
	if endTime.IsZero() {
		return nil, fmt.Errorf("end time is required")
	}

	if !endTime.After(startTime) {
		return nil, fmt.Errorf("end time must be after start time")
	}

	if input.MaxDatapoints < 0 {
		return nil, fmt.Errorf("max datapoints must be greater than or equal to zero")
	}

	scanBy := strings.TrimSpace(input.ScanBy)
	if scanBy == "" {
		scanBy = ScanByTimestampDescending
	}

	if !isValidScanBy(scanBy) {
		return nil, fmt.Errorf("invalid scan by value: %s", scanBy)
	}

	output := &QueryMetricsInsightsOutput{}
	nextToken := ""

	for {
		values := c.getMetricDataValues(query, startTime, endTime, scanBy, input.MaxDatapoints, nextToken)
		response := getMetricDataResponse{}
		if err := c.postForm(values, &response); err != nil {
			return nil, err
		}

		output.RequestID = strings.TrimSpace(response.RequestID)
		output.Results = mergeMetricDataResults(output.Results, response.MetricDataResults)
		output.Messages = append(output.Messages, response.Messages...)

		nextToken = strings.TrimSpace(response.NextToken)
		if nextToken == "" {
			return output, nil
		}
	}
}

func (c *Client) getMetricDataValues(
	query string,
	startTime time.Time,
	endTime time.Time,
	scanBy string,
	maxDatapoints int,
	nextToken string,
) url.Values {
	values := url.Values{}
	values.Set("Action", "GetMetricData")
	values.Set("Version", apiVersion)
	values.Set("StartTime", startTime.Format(time.RFC3339))
	values.Set("EndTime", endTime.Format(time.RFC3339))
	values.Set("ScanBy", scanBy)

	if maxDatapoints > 0 {
		values.Set("MaxDatapoints", strconv.Itoa(maxDatapoints))
	}

	values.Set("MetricDataQueries.member.1.Id", metricsQueryID)
	values.Set("MetricDataQueries.member.1.Expression", query)
	values.Set("MetricDataQueries.member.1.ReturnData", "true")
	values.Set("MetricDataQueries.member.1.Period", strconv.Itoa(defaultMetricsInsightsPeriodSeconds))

	if strings.TrimSpace(nextToken) != "" {
		values.Set("NextToken", strings.TrimSpace(nextToken))
	}

	return values
}

func (c *Client) postForm(values url.Values, out any) error {
	body := values.Encode()

	req, err := http.NewRequest(http.MethodPost, c.endpoint(), strings.NewReader(body))
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
	return c.signer.SignHTTP(context.Background(), *c.credentials, req, payloadHash, serviceName, c.region, time.Now())
}

func (c *Client) endpoint() string {
	return fmt.Sprintf("https://monitoring.%s.amazonaws.com/", c.region)
}

func mergeMetricDataResults(existing []MetricDataResult, incoming []MetricDataResult) []MetricDataResult {
	if len(incoming) == 0 {
		return existing
	}

	if len(existing) == 0 {
		return incoming
	}

	indexByID := map[string]int{}
	for i, result := range existing {
		indexByID[result.ID] = i
	}

	for _, result := range incoming {
		index, ok := indexByID[result.ID]
		if !ok {
			indexByID[result.ID] = len(existing)
			existing = append(existing, result)
			continue
		}

		existing[index].Timestamps = append(existing[index].Timestamps, result.Timestamps...)
		existing[index].Values = append(existing[index].Values, result.Values...)
		existing[index].Messages = append(existing[index].Messages, result.Messages...)
		if strings.TrimSpace(result.Label) != "" {
			existing[index].Label = result.Label
		}
		if strings.TrimSpace(result.StatusCode) != "" {
			existing[index].StatusCode = result.StatusCode
		}
	}

	return existing
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

	code := strings.TrimSpace(payload.Error.Code)
	message := strings.TrimSpace(payload.Error.Message)
	if code == "" && message == "" {
		return nil
	}

	return &common.Error{
		Code:    code,
		Message: message,
	}
}
