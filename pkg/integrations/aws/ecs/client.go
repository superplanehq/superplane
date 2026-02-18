package ecs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const targetPrefix = "AmazonEC2ContainerServiceV20141113."

type Client struct {
	http        core.HTTPContext
	region      string
	credentials *aws.Credentials
	signer      *v4.Signer
}

type Cluster struct {
	ClusterArn  string `json:"clusterArn"`
	ClusterName string `json:"clusterName"`
	Status      string `json:"status"`
}

type Failure struct {
	Arn    string `json:"arn"`
	Reason string `json:"reason"`
	Detail string `json:"detail"`
}

type Service struct {
	ServiceArn      string              `json:"serviceArn"`
	ServiceName     string              `json:"serviceName"`
	ClusterArn      string              `json:"clusterArn"`
	Status          string              `json:"status"`
	TaskDefinition  string              `json:"taskDefinition"`
	DesiredCount    int                 `json:"desiredCount"`
	RunningCount    int                 `json:"runningCount"`
	PendingCount    int                 `json:"pendingCount"`
	LaunchType      string              `json:"launchType"`
	PlatformVersion string              `json:"platformVersion"`
	Scheduling      string              `json:"schedulingStrategy"`
	CreatedAt       common.FloatTime    `json:"createdAt,omitempty"`
	Deployments     []ServiceDeployment `json:"deployments"`
	Events          []ServiceEvent      `json:"events"`
	TaskSets        []ServiceTaskSet    `json:"taskSets"`
	NetworkConfig   any                 `json:"networkConfiguration,omitempty"`
	PropagateTags   string              `json:"propagateTags"`
	EnableExec      bool                `json:"enableExecuteCommand"`
}

type ServiceDeployment struct {
	ID             string           `json:"id"`
	Status         string           `json:"status"`
	TaskDefinition string           `json:"taskDefinition"`
	DesiredCount   int              `json:"desiredCount"`
	PendingCount   int              `json:"pendingCount"`
	RunningCount   int              `json:"runningCount"`
	CreatedAt      common.FloatTime `json:"createdAt,omitempty"`
	UpdatedAt      common.FloatTime `json:"updatedAt,omitempty"`
}

type ServiceEvent struct {
	ID        string           `json:"id"`
	CreatedAt common.FloatTime `json:"createdAt,omitempty"`
	Message   string           `json:"message"`
}

type ServiceTaskSet struct {
	ID                   string           `json:"id"`
	TaskSetArn           string           `json:"taskSetArn"`
	Status               string           `json:"status"`
	TaskDefinition       string           `json:"taskDefinition"`
	ServiceArn           string           `json:"serviceArn"`
	ClusterArn           string           `json:"clusterArn"`
	LaunchType           string           `json:"launchType"`
	PlatformVersion      string           `json:"platformVersion"`
	ComputedDesiredCount int              `json:"computedDesiredCount"`
	PendingCount         int              `json:"pendingCount"`
	RunningCount         int              `json:"runningCount"`
	CreatedAt            common.FloatTime `json:"createdAt,omitempty"`
	UpdatedAt            common.FloatTime `json:"updatedAt,omitempty"`
}

type DescribeServicesResponse struct {
	Services []Service `json:"services"`
	Failures []Failure `json:"failures"`
}

type DescribeTasksResponse struct {
	Tasks    []Task    `json:"tasks"`
	Failures []Failure `json:"failures"`
}

type Task struct {
	TaskArn           string           `json:"taskArn"`
	ClusterArn        string           `json:"clusterArn"`
	TaskDefinitionArn string           `json:"taskDefinitionArn"`
	LastStatus        string           `json:"lastStatus"`
	DesiredStatus     string           `json:"desiredStatus"`
	StoppedReason     string           `json:"stoppedReason"`
	LaunchType        string           `json:"launchType"`
	PlatformVersion   string           `json:"platformVersion"`
	Group             string           `json:"group"`
	StartedBy         string           `json:"startedBy"`
	CreatedAt         common.FloatTime `json:"createdAt,omitempty"`
}

type RunTaskInput struct {
	Cluster              string
	TaskDefinition       string
	Count                int
	LaunchType           string
	CapacityProvider     []RunTaskCapacityProviderStrategyItem
	Group                string
	StartedBy            string
	PlatformVersion      string
	EnableExecuteCommand bool
	NetworkConfiguration RunTaskNetworkConfiguration
	Overrides            RunTaskOverrides
}

type RunTaskResponse struct {
	Tasks    []Task    `json:"tasks"`
	Failures []Failure `json:"failures"`
}

type StopTaskResponse struct {
	Task Task `json:"task"`
}

func NewClient(httpCtx core.HTTPContext, credentials *aws.Credentials, region string) *Client {
	return &Client{
		http:        httpCtx,
		region:      region,
		credentials: credentials,
		signer:      v4.NewSigner(),
	}
}

func (c *Client) ListClusters() ([]Cluster, error) {
	clusterArns, err := c.listPaginatedStrings("ListClusters", map[string]any{}, "clusterArns")
	if err != nil {
		return nil, err
	}

	clusters := make([]Cluster, 0, len(clusterArns))
	for _, arn := range clusterArns {
		clusterName := clusterNameFromArn(arn)
		clusters = append(clusters, Cluster{
			ClusterArn:  arn,
			ClusterName: clusterName,
		})
	}

	return clusters, nil
}

func (c *Client) ListServices(cluster string) ([]string, error) {
	return c.listPaginatedStrings(
		"ListServices",
		map[string]any{
			"cluster": cluster,
		},
		"serviceArns",
	)
}

func (c *Client) ListTaskDefinitions() ([]string, error) {
	return c.listPaginatedStrings(
		"ListTaskDefinitions",
		map[string]any{
			"status": "ACTIVE",
		},
		"taskDefinitionArns",
	)
}

func (c *Client) ListTasks(cluster string) ([]string, error) {
	return c.listPaginatedStrings(
		"ListTasks",
		map[string]any{
			"cluster":       cluster,
			"desiredStatus": "RUNNING",
		},
		"taskArns",
	)
}

func (c *Client) DescribeServices(cluster string, services []string) (*DescribeServicesResponse, error) {
	payload := map[string]any{
		"cluster":  cluster,
		"services": services,
	}

	response := DescribeServicesResponse{}
	if err := c.postJSON("DescribeServices", payload, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) DescribeTasks(cluster string, tasks []string) (*DescribeTasksResponse, error) {
	payload := map[string]any{
		"cluster": cluster,
		"tasks":   tasks,
	}

	response := DescribeTasksResponse{}
	if err := c.postJSON("DescribeTasks", payload, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) RunTask(input RunTaskInput) (*RunTaskResponse, error) {
	count := input.Count
	if count <= 0 {
		count = 1
	}

	payload := map[string]any{
		"cluster":        input.Cluster,
		"taskDefinition": input.TaskDefinition,
		"count":          count,
	}

	if input.LaunchType != "" {
		payload["launchType"] = input.LaunchType
	}
	if len(input.CapacityProvider) > 0 {
		payload["capacityProviderStrategy"] = input.CapacityProvider
	}
	if input.Group != "" {
		payload["group"] = input.Group
	}
	if input.StartedBy != "" {
		payload["startedBy"] = input.StartedBy
	}
	if input.PlatformVersion != "" {
		payload["platformVersion"] = input.PlatformVersion
	}
	if input.EnableExecuteCommand {
		payload["enableExecuteCommand"] = true
	}
	networkConfiguration := input.NetworkConfiguration.ToMap()
	if !isEmptyObject(networkConfiguration) && !isNetworkConfigurationTemplate(networkConfiguration) {
		payload["networkConfiguration"] = networkConfiguration
	}
	overrides := input.Overrides.ToMap()
	if !isEmptyObject(overrides) && !isOverridesTemplate(overrides) {
		payload["overrides"] = overrides
	}

	response := RunTaskResponse{}
	if err := c.postJSON("RunTask", payload, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) StopTask(cluster string, task string, reason string) (*StopTaskResponse, error) {
	payload := map[string]any{
		"cluster": cluster,
		"task":    task,
	}

	if reason != "" {
		payload["reason"] = reason
	}

	response := StopTaskResponse{}
	if err := c.postJSON("StopTask", payload, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *Client) postJSON(action string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("https://ecs.%s.amazonaws.com/", c.region)
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
		return fmt.Errorf("ECS API request failed with %d: %s", res.StatusCode, string(responseBody))
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
	return c.signer.SignHTTP(context.Background(), *c.credentials, req, payloadHash, "ecs", c.region, time.Now())
}

func (c *Client) listPaginatedStrings(action string, basePayload map[string]any, responseField string) ([]string, error) {
	results := []string{}
	nextToken := ""

	for {
		payload := clonePayload(basePayload)
		payload["maxResults"] = 100
		if nextToken != "" {
			payload["nextToken"] = nextToken
		}

		response := map[string]json.RawMessage{}
		if err := c.postJSON(action, payload, &response); err != nil {
			return nil, err
		}

		rawItems, ok := response[responseField]
		if !ok {
			return nil, fmt.Errorf("missing %s in %s response", responseField, action)
		}

		pageItems := []string{}
		if err := json.Unmarshal(rawItems, &pageItems); err != nil {
			return nil, fmt.Errorf("failed to decode %s in %s response: %w", responseField, action, err)
		}
		results = append(results, pageItems...)

		rawNextToken, ok := response["nextToken"]
		if !ok {
			break
		}

		pageNextToken := ""
		if err := json.Unmarshal(rawNextToken, &pageNextToken); err != nil {
			return nil, fmt.Errorf("failed to decode nextToken in %s response: %w", action, err)
		}
		if pageNextToken == "" {
			break
		}

		nextToken = pageNextToken
	}

	return results, nil
}

func clonePayload(base map[string]any) map[string]any {
	cloned := make(map[string]any, len(base)+2)
	for key, value := range base {
		cloned[key] = value
	}

	return cloned
}

func isEmptyObject(value any) bool {
	if value == nil {
		return true
	}

	parsedValue := reflect.ValueOf(value)
	if parsedValue.Kind() != reflect.Map {
		return false
	}

	if parsedValue.IsNil() {
		return true
	}

	return parsedValue.Len() == 0
}

func isNetworkConfigurationTemplate(value any) bool {
	object, ok := toStringAnyMap(value)
	if !ok || len(object) != 1 {
		return false
	}

	rawConfig, ok := object["awsvpcConfiguration"]
	if !ok {
		return false
	}

	config, ok := toStringAnyMap(rawConfig)
	if !ok {
		return false
	}
	if !hasOnlyKeys(config, "assignPublicIp", "subnets", "securityGroups") {
		return false
	}

	assignPublicIP, _ := config["assignPublicIp"].(string)
	if strings.ToUpper(assignPublicIP) != "DISABLED" {
		return false
	}

	subnets, hasSubnets := config["subnets"]
	if hasSubnets && !isEmptyArray(subnets) {
		return false
	}

	securityGroups, hasSecurityGroups := config["securityGroups"]
	if hasSecurityGroups && !isEmptyArray(securityGroups) {
		return false
	}

	return true
}

func isOverridesTemplate(value any) bool {
	object, ok := toStringAnyMap(value)
	if !ok || len(object) != 1 {
		return false
	}

	containerOverrides, ok := object["containerOverrides"]
	if !ok {
		return false
	}

	return isEmptyArray(containerOverrides)
}

func isEmptyArray(value any) bool {
	if value == nil {
		return false
	}

	parsedValue := reflect.ValueOf(value)
	switch parsedValue.Kind() {
	case reflect.Slice, reflect.Array:
		return parsedValue.Len() == 0
	default:
		return false
	}
}

func toStringAnyMap(value any) (map[string]any, bool) {
	parsedValue := reflect.ValueOf(value)
	if parsedValue.Kind() != reflect.Map {
		return nil, false
	}

	if parsedValue.IsNil() {
		return nil, false
	}

	converted := make(map[string]any, parsedValue.Len())
	iterator := parsedValue.MapRange()
	for iterator.Next() {
		converted[fmt.Sprintf("%v", iterator.Key().Interface())] = iterator.Value().Interface()
	}

	return converted, true
}

func hasOnlyKeys(object map[string]any, allowedKeys ...string) bool {
	allowed := make(map[string]struct{}, len(allowedKeys))
	for _, key := range allowedKeys {
		allowed[key] = struct{}{}
	}

	for key := range object {
		if _, ok := allowed[key]; !ok {
			return false
		}
	}

	return true
}

func clusterNameFromArn(arn string) string {
	parts := strings.SplitN(arn, "cluster/", 2)
	if len(parts) != 2 {
		return arn
	}
	return parts[1]
}

func serviceNameFromArn(arn string) string {
	parts := strings.SplitN(arn, "service/", 2)
	if len(parts) != 2 {
		return arn
	}

	segments := strings.Split(parts[1], "/")
	if len(segments) == 0 {
		return arn
	}
	return segments[len(segments)-1]
}

func taskDefinitionNameFromArn(arn string) string {
	parts := strings.SplitN(arn, "task-definition/", 2)
	if len(parts) != 2 {
		return arn
	}
	return parts[1]
}

func taskIDFromArn(arn string) string {
	parts := strings.SplitN(arn, "task/", 2)
	if len(parts) != 2 {
		return arn
	}

	taskIDWithClusterName := parts[1]
	parts = strings.SplitN(taskIDWithClusterName, "/", 2)
	if len(parts) != 2 {
		return arn
	}

	return parts[1]
}
