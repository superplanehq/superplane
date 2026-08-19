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
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	monitoringServiceName = "monitoring"
	monitoringAPIVersion  = "2010-08-01"

	// DescribeAlarms/ListMetrics are paginated. The caps keep resource pickers
	// responsive on accounts with very large numbers of alarms or metrics.
	maxAlarmPages  = 20
	maxMetricPages = 20
)

// Client provides CloudWatch alarm and metric operations through signed HTTP requests.
type Client struct {
	http        core.HTTPContext
	region      string
	endpoint    string
	credentials *aws.Credentials
	signer      *v4.Signer
}

// NewClient creates a region-scoped CloudWatch client.
func NewClient(httpCtx core.HTTPContext, credentials *aws.Credentials, region string) *Client {
	normalizedRegion := strings.TrimSpace(region)
	return &Client{
		http:        httpCtx,
		region:      normalizedRegion,
		endpoint:    fmt.Sprintf("https://%s.%s.%s/", monitoringServiceName, normalizedRegion, APIDNSSuffix(normalizedRegion)),
		credentials: credentials,
		signer:      v4.NewSigner(),
	}
}

type Dimension struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

// PutMetricAlarmInput carries the full desired alarm state. CloudWatch replaces
// the whole alarm configuration on every PutMetricAlarm call, so callers updating
// an existing alarm must seed this from the current alarm.
type PutMetricAlarmInput struct {
	AlarmName          string
	AlarmDescription   string
	Namespace          string
	MetricName         string
	Dimensions         []Dimension
	Statistic          string
	ExtendedStatistic  string
	Unit               string
	Period             int
	EvaluationPeriods  int
	DatapointsToAlarm  int
	Threshold          float64
	ComparisonOperator string
	TreatMissingData   string
	ActionsEnabled     bool

	// EvaluateLowSampleCountPercentile applies to percentile alarms only.
	// It must be carried over on update, or CloudWatch resets it to its default.
	EvaluateLowSampleCountPercentile string

	// Action ARNs executed on each state transition. A nil slice clears the actions.
	AlarmActions            []string
	OKActions               []string
	InsufficientDataActions []string

	// IncludeAlarmDescription sends AlarmDescription even when empty, to clear an existing description.
	IncludeAlarmDescription bool
}

type MetricAlarm struct {
	AlarmName                          string      `json:"alarmName" mapstructure:"alarmName"`
	AlarmArn                           string      `json:"alarmArn" mapstructure:"alarmArn"`
	AlarmDescription                   string      `json:"alarmDescription" mapstructure:"alarmDescription"`
	AlarmConfigurationUpdatedTimestamp string      `json:"alarmConfigurationUpdatedTimestamp" mapstructure:"alarmConfigurationUpdatedTimestamp"`
	ActionsEnabled                     bool        `json:"actionsEnabled" mapstructure:"actionsEnabled"`
	Namespace                          string      `json:"namespace" mapstructure:"namespace"`
	MetricName                         string      `json:"metricName" mapstructure:"metricName"`
	Dimensions                         []Dimension `json:"dimensions" mapstructure:"dimensions"`
	Statistic                          string      `json:"statistic" mapstructure:"statistic"`
	ExtendedStatistic                  string      `json:"extendedStatistic" mapstructure:"extendedStatistic"`
	Unit                               string      `json:"unit" mapstructure:"unit"`
	Period                             int         `json:"period" mapstructure:"period"`
	EvaluationPeriods                  int         `json:"evaluationPeriods" mapstructure:"evaluationPeriods"`
	DatapointsToAlarm                  int         `json:"datapointsToAlarm" mapstructure:"datapointsToAlarm"`
	Threshold                          float64     `json:"threshold" mapstructure:"threshold"`
	ThresholdMetricID                  string      `json:"thresholdMetricId" mapstructure:"thresholdMetricId"`
	ComparisonOperator                 string      `json:"comparisonOperator" mapstructure:"comparisonOperator"`
	TreatMissingData                   string      `json:"treatMissingData" mapstructure:"treatMissingData"`
	EvaluateLowSampleCountPercentile   string      `json:"evaluateLowSampleCountPercentile" mapstructure:"evaluateLowSampleCountPercentile"`
	StateValue                         string      `json:"stateValue" mapstructure:"stateValue"`
	StateReason                        string      `json:"stateReason" mapstructure:"stateReason"`
	StateReasonData                    string      `json:"stateReasonData" mapstructure:"stateReasonData"`
	StateUpdatedTimestamp              string      `json:"stateUpdatedTimestamp" mapstructure:"stateUpdatedTimestamp"`
	StateTransitionedTimestamp         string      `json:"stateTransitionedTimestamp" mapstructure:"stateTransitionedTimestamp"`
	AlarmActions                       []string    `json:"alarmActions" mapstructure:"alarmActions"`
	OKActions                          []string    `json:"okActions" mapstructure:"okActions"`
	InsufficientDataActions            []string    `json:"insufficientDataActions" mapstructure:"insufficientDataActions"`
	Region                             string      `json:"region" mapstructure:"region"`
	ConsoleURL                         string      `json:"consoleUrl" mapstructure:"consoleUrl"`

	// HasMetricQueries marks an alarm defined by a Metrics array — metric math,
	// anomaly detection or a Metrics Insights query. Those cannot be expressed
	// with the single-metric fields, so they are never rewritten.
	HasMetricQueries bool `json:"-" mapstructure:"-"`
}

// Metric identifies a CloudWatch metric: a namespace, a name and a dimension set.
type Metric struct {
	Namespace  string      `json:"namespace" mapstructure:"namespace"`
	MetricName string      `json:"metricName" mapstructure:"metricName"`
	Dimensions []Dimension `json:"dimensions" mapstructure:"dimensions"`
}

func (c *Client) PutMetricAlarm(input PutMetricAlarmInput) error {
	params := url.Values{}
	params.Set("AlarmName", strings.TrimSpace(input.AlarmName))
	params.Set("Namespace", strings.TrimSpace(input.Namespace))
	params.Set("MetricName", strings.TrimSpace(input.MetricName))
	params.Set("ComparisonOperator", strings.TrimSpace(input.ComparisonOperator))
	params.Set("Threshold", strconv.FormatFloat(input.Threshold, 'f', -1, 64))
	params.Set("ActionsEnabled", strconv.FormatBool(input.ActionsEnabled))

	for index, dimension := range input.Dimensions {
		name := strings.TrimSpace(dimension.Name)
		if name == "" {
			continue
		}

		params.Set(fmt.Sprintf("Dimensions.member.%d.Name", index+1), name)
		params.Set(fmt.Sprintf("Dimensions.member.%d.Value", index+1), strings.TrimSpace(dimension.Value))
	}

	// Statistic and ExtendedStatistic are mutually exclusive in the API.
	if extended := strings.TrimSpace(input.ExtendedStatistic); extended != "" {
		params.Set("ExtendedStatistic", extended)
	} else {
		statistic := strings.TrimSpace(input.Statistic)
		if statistic == "" {
			statistic = StatisticAverage
		}
		params.Set("Statistic", statistic)
	}

	period := input.Period
	if period <= 0 {
		period = defaultAlarmPeriod
	}
	params.Set("Period", strconv.Itoa(period))

	evaluationPeriods := input.EvaluationPeriods
	if evaluationPeriods <= 0 {
		evaluationPeriods = defaultAlarmEvaluationPeriods
	}
	params.Set("EvaluationPeriods", strconv.Itoa(evaluationPeriods))

	if input.DatapointsToAlarm > 0 {
		params.Set("DatapointsToAlarm", strconv.Itoa(input.DatapointsToAlarm))
	}

	description := strings.TrimSpace(input.AlarmDescription)
	if input.IncludeAlarmDescription || description != "" {
		params.Set("AlarmDescription", description)
	}

	if treatMissingData := strings.TrimSpace(input.TreatMissingData); treatMissingData != "" {
		params.Set("TreatMissingData", treatMissingData)
	}

	if unit := strings.TrimSpace(input.Unit); unit != "" {
		params.Set("Unit", unit)
	}

	if percentile := strings.TrimSpace(input.EvaluateLowSampleCountPercentile); percentile != "" {
		params.Set("EvaluateLowSampleCountPercentile", percentile)
	}

	setActionMembers(params, "AlarmActions", input.AlarmActions)
	setActionMembers(params, "OKActions", input.OKActions)
	setActionMembers(params, "InsufficientDataActions", input.InsufficientDataActions)

	return c.postSignedForm("PutMetricAlarm", params, nil)
}

func (c *Client) DescribeAlarm(alarmName string) (*MetricAlarm, error) {
	params := url.Values{}
	params.Set("AlarmNames.member.1", strings.TrimSpace(alarmName))
	params.Set("AlarmTypes.member.1", "MetricAlarm")

	response := describeAlarmsResponse{}
	if err := c.postSignedForm("DescribeAlarms", params, &response); err != nil {
		return nil, err
	}

	if len(response.Result.MetricAlarms) == 0 {
		return nil, fmt.Errorf("alarm not found: %s", alarmName)
	}

	return alarmFromXML(response.Result.MetricAlarms[0], c.region), nil
}

func (c *Client) ListAlarms() ([]MetricAlarm, error) {
	alarms := []MetricAlarm{}
	nextToken := ""

	for page := 0; page < maxAlarmPages; page++ {
		params := url.Values{}
		params.Set("MaxRecords", "100")
		params.Set("AlarmTypes.member.1", "MetricAlarm")
		if nextToken != "" {
			params.Set("NextToken", nextToken)
		}

		response := describeAlarmsResponse{}
		if err := c.postSignedForm("DescribeAlarms", params, &response); err != nil {
			return nil, err
		}

		for _, xmlAlarm := range response.Result.MetricAlarms {
			alarms = append(alarms, *alarmFromXML(xmlAlarm, c.region))
		}

		nextToken = strings.TrimSpace(response.Result.NextToken)
		if nextToken == "" {
			break
		}
	}

	return alarms, nil
}

// ListMetrics returns the metrics available in the region, optionally narrowed
// down to a namespace and a metric name.
func (c *Client) ListMetrics(namespace, metricName string) ([]Metric, error) {
	metrics := []Metric{}
	nextToken := ""

	for page := 0; page < maxMetricPages; page++ {
		params := url.Values{}
		if trimmed := strings.TrimSpace(namespace); trimmed != "" {
			params.Set("Namespace", trimmed)
		}
		if trimmed := strings.TrimSpace(metricName); trimmed != "" {
			params.Set("MetricName", trimmed)
		}
		if nextToken != "" {
			params.Set("NextToken", nextToken)
		}

		response := listMetricsResponse{}
		if err := c.postSignedForm("ListMetrics", params, &response); err != nil {
			return nil, err
		}

		for _, xmlMetric := range response.Result.Metrics {
			metrics = append(metrics, metricFromXML(xmlMetric))
		}

		nextToken = strings.TrimSpace(response.Result.NextToken)
		if nextToken == "" {
			break
		}
	}

	return metrics, nil
}

// AlarmMuteRuleSchedule mirrors CloudWatch's Schedule shape: a recurring
// cron(...) or one-time at(...) expression, how long each activation mutes
// alarms for, and the timezone the expression is evaluated in.
type AlarmMuteRuleSchedule struct {
	Expression string
	Duration   string
	Timezone   string
}

type PutAlarmMuteRuleInput struct {
	Name        string
	Description string
	Schedule    AlarmMuteRuleSchedule
	AlarmNames  []string
	StartDate   *time.Time
	ExpireDate  *time.Time
}

type AlarmMuteRule struct {
	Name                 string
	AlarmMuteRuleArn     string
	Description          string
	Schedule             AlarmMuteRuleSchedule
	AlarmNames           []string
	StartDate            string
	ExpireDate           string
	Status               string
	MuteType             string
	LastUpdatedTimestamp string
}

func (c *Client) PutAlarmMuteRule(input PutAlarmMuteRuleInput) error {
	params := url.Values{}
	params.Set("Name", strings.TrimSpace(input.Name))
	params.Set("Rule.Schedule.Expression", strings.TrimSpace(input.Schedule.Expression))
	params.Set("Rule.Schedule.Duration", strings.TrimSpace(input.Schedule.Duration))

	if description := strings.TrimSpace(input.Description); description != "" {
		params.Set("Description", description)
	}

	if timezone := strings.TrimSpace(input.Schedule.Timezone); timezone != "" {
		params.Set("Rule.Schedule.Timezone", timezone)
	}

	setActionMembers(params, "MuteTargets.AlarmNames", input.AlarmNames)

	if input.StartDate != nil {
		params.Set("StartDate", input.StartDate.UTC().Format(time.RFC3339))
	}

	if input.ExpireDate != nil {
		params.Set("ExpireDate", input.ExpireDate.UTC().Format(time.RFC3339))
	}

	return c.postSignedForm("PutAlarmMuteRule", params, nil)
}

func (c *Client) GetAlarmMuteRule(name string) (*AlarmMuteRule, error) {
	params := url.Values{}
	params.Set("AlarmMuteRuleName", strings.TrimSpace(name))

	response := getAlarmMuteRuleResponse{}
	if err := c.postSignedForm("GetAlarmMuteRule", params, &response); err != nil {
		return nil, err
	}

	return alarmMuteRuleFromXML(response.Result), nil
}

func setActionMembers(params url.Values, key string, arns []string) {
	index := 0
	for _, arn := range arns {
		arn = strings.TrimSpace(arn)
		if arn == "" {
			continue
		}

		index++
		params.Set(fmt.Sprintf("%s.member.%d", key, index), arn)
	}
}

// ─── XML response shapes ────────────────────────────────────────────────────

type describeAlarmsResponse struct {
	XMLName xml.Name             `xml:"DescribeAlarmsResponse"`
	Result  describeAlarmsResult `xml:"DescribeAlarmsResult"`
}

type describeAlarmsResult struct {
	MetricAlarms []xmlMetricAlarm `xml:"MetricAlarms>member"`
	NextToken    string           `xml:"NextToken"`
}

type xmlMetricAlarm struct {
	AlarmName                          string          `xml:"AlarmName"`
	AlarmArn                           string          `xml:"AlarmArn"`
	AlarmDescription                   string          `xml:"AlarmDescription"`
	AlarmConfigurationUpdatedTimestamp string          `xml:"AlarmConfigurationUpdatedTimestamp"`
	ActionsEnabled                     bool            `xml:"ActionsEnabled"`
	Namespace                          string          `xml:"Namespace"`
	MetricName                         string          `xml:"MetricName"`
	Dimensions                         []xmlDimension  `xml:"Dimensions>member"`
	Statistic                          string          `xml:"Statistic"`
	ExtendedStatistic                  string          `xml:"ExtendedStatistic"`
	Unit                               string          `xml:"Unit"`
	Period                             int             `xml:"Period"`
	EvaluationPeriods                  int             `xml:"EvaluationPeriods"`
	DatapointsToAlarm                  int             `xml:"DatapointsToAlarm"`
	Threshold                          float64         `xml:"Threshold"`
	ThresholdMetricID                  string          `xml:"ThresholdMetricId"`
	ComparisonOperator                 string          `xml:"ComparisonOperator"`
	TreatMissingData                   string          `xml:"TreatMissingData"`
	EvaluateLowSampleCountPercentile   string          `xml:"EvaluateLowSampleCountPercentile"`
	StateValue                         string          `xml:"StateValue"`
	StateReason                        string          `xml:"StateReason"`
	StateReasonData                    string          `xml:"StateReasonData"`
	StateUpdatedTimestamp              string          `xml:"StateUpdatedTimestamp"`
	StateTransitionedTimestamp         string          `xml:"StateTransitionedTimestamp"`
	AlarmActions                       []string        `xml:"AlarmActions>member"`
	OKActions                          []string        `xml:"OKActions>member"`
	InsufficientDataActions            []string        `xml:"InsufficientDataActions>member"`
	Metrics                            []xmlMetricItem `xml:"Metrics>member"`
}

// xmlMetricItem only needs the id: its presence marks a metric math alarm,
// whose configuration cannot be expressed with the single-metric fields.
type xmlMetricItem struct {
	ID string `xml:"Id"`
}

type xmlDimension struct {
	Name  string `xml:"Name"`
	Value string `xml:"Value"`
}

type listMetricsResponse struct {
	XMLName xml.Name          `xml:"ListMetricsResponse"`
	Result  listMetricsResult `xml:"ListMetricsResult"`
}

type listMetricsResult struct {
	Metrics   []xmlMetric `xml:"Metrics>member"`
	NextToken string      `xml:"NextToken"`
}

type xmlMetric struct {
	Namespace  string         `xml:"Namespace"`
	MetricName string         `xml:"MetricName"`
	Dimensions []xmlDimension `xml:"Dimensions>member"`
}

func alarmFromXML(x xmlMetricAlarm, region string) *MetricAlarm {
	dimensions := make([]Dimension, 0, len(x.Dimensions))
	for _, dimension := range x.Dimensions {
		dimensions = append(dimensions, Dimension{Name: dimension.Name, Value: dimension.Value})
	}

	return &MetricAlarm{
		AlarmName:                          x.AlarmName,
		AlarmArn:                           x.AlarmArn,
		AlarmDescription:                   x.AlarmDescription,
		AlarmConfigurationUpdatedTimestamp: x.AlarmConfigurationUpdatedTimestamp,
		ActionsEnabled:                     x.ActionsEnabled,
		Namespace:                          x.Namespace,
		MetricName:                         x.MetricName,
		Dimensions:                         dimensions,
		Statistic:                          x.Statistic,
		ExtendedStatistic:                  x.ExtendedStatistic,
		Unit:                               x.Unit,
		Period:                             x.Period,
		EvaluationPeriods:                  x.EvaluationPeriods,
		DatapointsToAlarm:                  x.DatapointsToAlarm,
		Threshold:                          x.Threshold,
		ThresholdMetricID:                  x.ThresholdMetricID,
		ComparisonOperator:                 x.ComparisonOperator,
		TreatMissingData:                   x.TreatMissingData,
		EvaluateLowSampleCountPercentile:   x.EvaluateLowSampleCountPercentile,
		StateValue:                         x.StateValue,
		StateReason:                        x.StateReason,
		StateReasonData:                    x.StateReasonData,
		StateUpdatedTimestamp:              x.StateUpdatedTimestamp,
		StateTransitionedTimestamp:         x.StateTransitionedTimestamp,
		AlarmActions:                       x.AlarmActions,
		OKActions:                          x.OKActions,
		InsufficientDataActions:            x.InsufficientDataActions,
		Region:                             region,
		ConsoleURL:                         AlarmConsoleURL(region, x.AlarmName),
		HasMetricQueries:                   len(x.Metrics) > 0,
	}
}

type getAlarmMuteRuleResponse struct {
	XMLName xml.Name               `xml:"GetAlarmMuteRuleResponse"`
	Result  getAlarmMuteRuleResult `xml:"GetAlarmMuteRuleResult"`
}

type getAlarmMuteRuleResult struct {
	Name                 string         `xml:"Name"`
	AlarmMuteRuleArn     string         `xml:"AlarmMuteRuleArn"`
	Description          string         `xml:"Description"`
	Rule                 xmlMuteRule    `xml:"Rule"`
	MuteTargets          xmlMuteTargets `xml:"MuteTargets"`
	StartDate            string         `xml:"StartDate"`
	ExpireDate           string         `xml:"ExpireDate"`
	Status               string         `xml:"Status"`
	LastUpdatedTimestamp string         `xml:"LastUpdatedTimestamp"`
	MuteType             string         `xml:"MuteType"`
}

type xmlMuteRule struct {
	Schedule xmlMuteSchedule `xml:"Schedule"`
}

type xmlMuteSchedule struct {
	Expression string `xml:"Expression"`
	Duration   string `xml:"Duration"`
	Timezone   string `xml:"Timezone"`
}

type xmlMuteTargets struct {
	AlarmNames []string `xml:"AlarmNames>member"`
}

func alarmMuteRuleFromXML(x getAlarmMuteRuleResult) *AlarmMuteRule {
	return &AlarmMuteRule{
		Name:             x.Name,
		AlarmMuteRuleArn: x.AlarmMuteRuleArn,
		Description:      x.Description,
		Schedule: AlarmMuteRuleSchedule{
			Expression: x.Rule.Schedule.Expression,
			Duration:   x.Rule.Schedule.Duration,
			Timezone:   x.Rule.Schedule.Timezone,
		},
		AlarmNames:           x.MuteTargets.AlarmNames,
		StartDate:            x.StartDate,
		ExpireDate:           x.ExpireDate,
		Status:               x.Status,
		MuteType:             x.MuteType,
		LastUpdatedTimestamp: x.LastUpdatedTimestamp,
	}
}

func alarmMuteRuleToMap(rule *AlarmMuteRule) map[string]any {
	if rule == nil {
		return map[string]any{}
	}

	return map[string]any{
		"name":                 rule.Name,
		"alarmMuteRuleArn":     rule.AlarmMuteRuleArn,
		"description":          rule.Description,
		"scheduleExpression":   rule.Schedule.Expression,
		"scheduleDuration":     rule.Schedule.Duration,
		"scheduleTimezone":     rule.Schedule.Timezone,
		"alarmNames":           rule.AlarmNames,
		"startDate":            rule.StartDate,
		"expireDate":           rule.ExpireDate,
		"status":               rule.Status,
		"muteType":             rule.MuteType,
		"lastUpdatedTimestamp": rule.LastUpdatedTimestamp,
	}
}

func metricFromXML(x xmlMetric) Metric {
	dimensions := make([]Dimension, 0, len(x.Dimensions))
	for _, dimension := range x.Dimensions {
		dimensions = append(dimensions, Dimension{Name: dimension.Name, Value: dimension.Value})
	}

	sort.Slice(dimensions, func(i, j int) bool { return dimensions[i].Name < dimensions[j].Name })

	return Metric{
		Namespace:  x.Namespace,
		MetricName: x.MetricName,
		Dimensions: dimensions,
	}
}

// AlarmConsoleURL builds the CloudWatch console link for an alarm.
func AlarmConsoleURL(region, alarmName string) string {
	region = strings.TrimSpace(region)
	alarmName = strings.TrimSpace(alarmName)
	if region == "" || alarmName == "" {
		return ""
	}

	return fmt.Sprintf(
		"https://%s.%s/cloudwatch/home?region=%s#alarmsV2:alarm/%s",
		region, ConsoleHost(region), region, url.PathEscape(alarmName),
	)
}

func alarmToMap(alarm *MetricAlarm) map[string]any {
	if alarm == nil {
		return map[string]any{}
	}

	dimensions := make([]map[string]any, 0, len(alarm.Dimensions))
	for _, dimension := range alarm.Dimensions {
		dimensions = append(dimensions, map[string]any{"name": dimension.Name, "value": dimension.Value})
	}

	return map[string]any{
		"alarmName":                          alarm.AlarmName,
		"alarmArn":                           alarm.AlarmArn,
		"alarmDescription":                   alarm.AlarmDescription,
		"alarmConfigurationUpdatedTimestamp": alarm.AlarmConfigurationUpdatedTimestamp,
		"actionsEnabled":                     alarm.ActionsEnabled,
		"namespace":                          alarm.Namespace,
		"metricName":                         alarm.MetricName,
		"dimensions":                         dimensions,
		"statistic":                          alarm.Statistic,
		"extendedStatistic":                  alarm.ExtendedStatistic,
		"unit":                               alarm.Unit,
		"period":                             alarm.Period,
		"evaluationPeriods":                  alarm.EvaluationPeriods,
		"datapointsToAlarm":                  alarm.DatapointsToAlarm,
		"threshold":                          alarm.Threshold,
		"thresholdMetricId":                  alarm.ThresholdMetricID,
		"comparisonOperator":                 alarm.ComparisonOperator,
		"treatMissingData":                   alarm.TreatMissingData,
		"evaluateLowSampleCountPercentile":   alarm.EvaluateLowSampleCountPercentile,
		"stateValue":                         alarm.StateValue,
		"stateReason":                        alarm.StateReason,
		"stateReasonData":                    alarm.StateReasonData,
		"stateUpdatedTimestamp":              alarm.StateUpdatedTimestamp,
		"stateTransitionedTimestamp":         alarm.StateTransitionedTimestamp,
		"alarmActions":                       alarm.AlarmActions,
		"okActions":                          alarm.OKActions,
		"insufficientDataActions":            alarm.InsufficientDataActions,
		"region":                             alarm.Region,
		"consoleUrl":                         alarm.ConsoleURL,
	}
}

// parseError reads the CloudWatch query-protocol error envelope:
// <ErrorResponse><Error><Code>/<Message></Error></ErrorResponse>.
func parseError(body []byte) *common.Error {
	var errResp struct {
		Error struct {
			Code    string `xml:"Code"`
			Message string `xml:"Message"`
		} `xml:"Error"`
	}

	if err := xml.Unmarshal(body, &errResp); err != nil {
		return nil
	}

	code := strings.TrimSpace(errResp.Error.Code)
	message := strings.TrimSpace(errResp.Error.Message)
	if code == "" && message == "" {
		return nil
	}

	return &common.Error{Code: code, Message: message}
}

func (c *Client) postSignedForm(action string, params url.Values, out any) error {
	if params == nil {
		params = url.Values{}
	}

	params.Set("Action", action)
	params.Set("Version", monitoringAPIVersion)

	body := []byte(params.Encode())
	req, err := http.NewRequest(http.MethodPost, c.endpoint, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	hash := sha256.Sum256(body)
	err = c.signer.SignHTTP(
		context.Background(), *c.credentials, req,
		hex.EncodeToString(hash[:]), monitoringServiceName, c.region, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
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
