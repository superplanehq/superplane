package sentry

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	httpContext core.HTTPContext
	baseURL     string
	userToken   string
	orgSlug     string
}

type apiError struct {
	StatusCode int
	Body       string
}

const releaseScope = "project:releases"

func (e *apiError) Error() string {
	return fmt.Sprintf("sentry API returned status %d: %s", e.StatusCode, e.Body)
}

func wrapReleaseScopeError(err error) error {
	var sentryAPIError *apiError
	if errors.As(err, &sentryAPIError) && sentryAPIError.StatusCode == http.StatusForbidden {
		return fmt.Errorf(
			"Sentry release APIs require the personal token to include `%s`: %w",
			releaseScope,
			err,
		)
	}

	return err
}

func NewClient(httpContext core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	baseURL, err := integration.GetConfig("baseUrl")
	if err != nil {
		return nil, fmt.Errorf("failed to get sentry base URL: %w", err)
	}

	userToken, err := integration.GetConfig("userToken")
	if err != nil {
		return nil, fmt.Errorf("failed to get sentry user token: %w", err)
	}

	if strings.TrimSpace(string(userToken)) == "" {
		return nil, fmt.Errorf("Sentry user token is missing")
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode sentry metadata: %w", err)
	}

	if metadata.Organization == nil || metadata.Organization.Slug == "" {
		return nil, fmt.Errorf("Sentry organization is not connected")
	}

	return &Client{
		httpContext: httpContext,
		baseURL:     normalizeBaseURL(string(baseURL)),
		userToken:   strings.TrimSpace(string(userToken)),
		orgSlug:     metadata.Organization.Slug,
	}, nil
}

func NewAPIClient(httpContext core.HTTPContext, baseURL, userToken string) *Client {
	return &Client{
		httpContext: httpContext,
		baseURL:     normalizeBaseURL(baseURL),
		userToken:   strings.TrimSpace(userToken),
	}
}

type Organization struct {
	ID   string `json:"id" mapstructure:"id"`
	Slug string `json:"slug" mapstructure:"slug"`
	Name string `json:"name" mapstructure:"name"`
}

type Project struct {
	ID   string `json:"id" mapstructure:"id"`
	Slug string `json:"slug" mapstructure:"slug"`
	Name string `json:"name" mapstructure:"name"`
}

type Team struct {
	ID   string `json:"id" mapstructure:"id"`
	Slug string `json:"slug" mapstructure:"slug"`
	Name string `json:"name" mapstructure:"name"`
}

type ProjectMember struct {
	ID    string `json:"id" mapstructure:"id"`
	Name  string `json:"name" mapstructure:"name"`
	Email string `json:"email" mapstructure:"email"`
	User  *struct {
		ID       string `json:"id" mapstructure:"id"`
		Name     string `json:"name" mapstructure:"name"`
		Username string `json:"username" mapstructure:"username"`
		Email    string `json:"email" mapstructure:"email"`
	} `json:"user" mapstructure:"user"`
}

type AssigneeResource struct {
	ID   string
	Name string
}

type Issue struct {
	ID            string         `json:"id" mapstructure:"id"`
	ShortID       string         `json:"shortId" mapstructure:"shortId"`
	Title         string         `json:"title" mapstructure:"title"`
	Count         string         `json:"count" mapstructure:"count"`
	Status        string         `json:"status" mapstructure:"status"`
	Priority      string         `json:"priority" mapstructure:"priority"`
	HasSeen       bool           `json:"hasSeen" mapstructure:"hasSeen"`
	IsPublic      bool           `json:"isPublic" mapstructure:"isPublic"`
	IsSubscribed  bool           `json:"isSubscribed" mapstructure:"isSubscribed"`
	StatusDetails any            `json:"statusDetails" mapstructure:"statusDetails"`
	NumComments   int            `json:"numComments" mapstructure:"numComments"`
	UserCount     int            `json:"userCount" mapstructure:"userCount"`
	Permalink     string         `json:"permalink" mapstructure:"permalink"`
	WebURL        string         `json:"web_url" mapstructure:"web_url"`
	Metadata      map[string]any `json:"metadata" mapstructure:"metadata"`
	Tags          []IssueTag     `json:"tags" mapstructure:"tags"`
	Stats         map[string]any `json:"stats" mapstructure:"stats"`
	Events        []IssueEvent   `json:"events,omitempty" mapstructure:"events"`
	AssignedTo    *IssueAssignee `json:"assignedTo" mapstructure:"assignedTo"`
	Project       *IssueProject  `json:"project" mapstructure:"project"`
}

type IssueTag struct {
	Key   string `json:"key" mapstructure:"key"`
	Value string `json:"value" mapstructure:"value"`
}

type IssueEvent struct {
	ID          string         `json:"id" mapstructure:"id"`
	EventID     string         `json:"eventID" mapstructure:"eventID"`
	Title       string         `json:"title" mapstructure:"title"`
	Message     string         `json:"message" mapstructure:"message"`
	DateCreated string         `json:"dateCreated" mapstructure:"dateCreated"`
	Platform    string         `json:"platform" mapstructure:"platform"`
	Location    string         `json:"location" mapstructure:"location"`
	Culprit     string         `json:"culprit" mapstructure:"culprit"`
	Tags        []IssueTag     `json:"tags" mapstructure:"tags"`
	User        map[string]any `json:"user" mapstructure:"user"`
}

type IssueAssignee struct {
	Type  string `json:"type" mapstructure:"type"`
	ID    string `json:"id" mapstructure:"id"`
	Name  string `json:"name" mapstructure:"name"`
	Email string `json:"email" mapstructure:"email"`
}

type IssueProject struct {
	ID   string `json:"id" mapstructure:"id"`
	Name string `json:"name" mapstructure:"name"`
	Slug string `json:"slug" mapstructure:"slug"`
}

type AuthIdentity struct {
	ID       string `json:"id" mapstructure:"id"`
	Name     string `json:"name" mapstructure:"name"`
	Username string `json:"username" mapstructure:"username"`
	Email    string `json:"email" mapstructure:"email"`
}

type SentryApp struct {
	Name           string   `json:"name" mapstructure:"name"`
	Slug           string   `json:"slug" mapstructure:"slug"`
	Scopes         []string `json:"scopes" mapstructure:"scopes"`
	Events         []string `json:"events" mapstructure:"events"`
	WebhookURL     string   `json:"webhookUrl" mapstructure:"webhookUrl"`
	RedirectURL    *string  `json:"redirectUrl" mapstructure:"redirectUrl"`
	IsInternal     bool     `json:"isInternal" mapstructure:"isInternal"`
	IsAlertable    bool     `json:"isAlertable" mapstructure:"isAlertable"`
	Overview       *string  `json:"overview" mapstructure:"overview"`
	VerifyInstall  bool     `json:"verifyInstall" mapstructure:"verifyInstall"`
	AllowedOrigins []string `json:"allowedOrigins" mapstructure:"allowedOrigins"`
	Author         string   `json:"author" mapstructure:"author"`
	Schema         any      `json:"schema" mapstructure:"schema"`
	ClientSecret   string   `json:"clientSecret" mapstructure:"clientSecret"`
}

type UpdateSentryAppRequest struct {
	Name           string   `json:"name"`
	Scopes         []string `json:"scopes"`
	Events         []string `json:"events,omitempty"`
	WebhookURL     string   `json:"webhookUrl,omitempty"`
	RedirectURL    *string  `json:"redirectUrl,omitempty"`
	IsInternal     bool     `json:"isInternal"`
	IsAlertable    bool     `json:"isAlertable"`
	Overview       *string  `json:"overview,omitempty"`
	VerifyInstall  bool     `json:"verifyInstall"`
	AllowedOrigins []string `json:"allowedOrigins,omitempty"`
	Author         string   `json:"author,omitempty"`
	Schema         any      `json:"schema,omitempty"`
}

type UpdateIssueRequest struct {
	Status       string `json:"status,omitempty"`
	AssignedTo   string `json:"assignedTo,omitempty"`
	Priority     string `json:"priority,omitempty"`
	HasSeen      *bool  `json:"hasSeen,omitempty"`
	IsPublic     *bool  `json:"isPublic,omitempty"`
	IsSubscribed *bool  `json:"isSubscribed,omitempty"`
}

type MetricAlertRule struct {
	ID               string                    `json:"id" mapstructure:"id"`
	Name             string                    `json:"name" mapstructure:"name"`
	OrganizationID   string                    `json:"organizationId" mapstructure:"organizationId"`
	QueryType        int                       `json:"queryType" mapstructure:"queryType"`
	Dataset          string                    `json:"dataset" mapstructure:"dataset"`
	Query            string                    `json:"query" mapstructure:"query"`
	Aggregate        string                    `json:"aggregate" mapstructure:"aggregate"`
	ThresholdType    int                       `json:"thresholdType" mapstructure:"thresholdType"`
	ResolveThreshold any                       `json:"resolveThreshold" mapstructure:"resolveThreshold"`
	TimeWindow       float64                   `json:"timeWindow" mapstructure:"timeWindow"`
	Environment      any                       `json:"environment" mapstructure:"environment"`
	Projects         []string                  `json:"projects" mapstructure:"projects"`
	Owner            *string                   `json:"owner" mapstructure:"owner"`
	OriginalRuleID   any                       `json:"originalAlertRuleId" mapstructure:"originalAlertRuleId"`
	ComparisonDelta  any                       `json:"comparisonDelta" mapstructure:"comparisonDelta"`
	DateModified     string                    `json:"dateModified" mapstructure:"dateModified"`
	DateCreated      string                    `json:"dateCreated" mapstructure:"dateCreated"`
	CreatedBy        *MetricAlertRuleCreatedBy `json:"createdBy" mapstructure:"createdBy"`
	EventTypes       []string                  `json:"eventTypes" mapstructure:"eventTypes"`
	Triggers         []MetricAlertTrigger      `json:"triggers" mapstructure:"triggers"`
}

type MetricAlertTrigger struct {
	ID               string              `json:"id" mapstructure:"id"`
	AlertRuleID      string              `json:"alertRuleId" mapstructure:"alertRuleId"`
	Label            string              `json:"label" mapstructure:"label"`
	ThresholdType    int                 `json:"thresholdType" mapstructure:"thresholdType"`
	AlertThreshold   any                 `json:"alertThreshold" mapstructure:"alertThreshold"`
	ResolveThreshold any                 `json:"resolveThreshold" mapstructure:"resolveThreshold"`
	DateCreated      string              `json:"dateCreated" mapstructure:"dateCreated"`
	Actions          []MetricAlertAction `json:"actions" mapstructure:"actions"`
}

type MetricAlertAction struct {
	ID                 string  `json:"id" mapstructure:"id"`
	AlertRuleTriggerID string  `json:"alertRuleTriggerId" mapstructure:"alertRuleTriggerId"`
	Type               string  `json:"type" mapstructure:"type"`
	TargetType         string  `json:"targetType" mapstructure:"targetType"`
	TargetIdentifier   string  `json:"targetIdentifier" mapstructure:"targetIdentifier"`
	InputChannelID     *string `json:"inputChannelId" mapstructure:"inputChannelId"`
	IntegrationID      *string `json:"integrationId" mapstructure:"integrationId"`
	SentryAppID        any     `json:"sentryAppId" mapstructure:"sentryAppId"`
	Priority           *string `json:"priority" mapstructure:"priority"`
	DateCreated        string  `json:"dateCreated" mapstructure:"dateCreated"`
}

type MetricAlertRuleCreatedBy struct {
	ID    any    `json:"id" mapstructure:"id"`
	Name  string `json:"name" mapstructure:"name"`
	Email string `json:"email" mapstructure:"email"`
}

type CreateOrUpdateMetricAlertRuleRequest struct {
	Name             string                    `json:"name"`
	Aggregate        string                    `json:"aggregate"`
	TimeWindow       int                       `json:"timeWindow"`
	Projects         []string                  `json:"projects"`
	Query            string                    `json:"query"`
	ThresholdType    int                       `json:"thresholdType"`
	Triggers         []MetricAlertTriggerInput `json:"triggers"`
	Environment      string                    `json:"environment,omitempty"`
	Dataset          string                    `json:"dataset,omitempty"`
	QueryType        *int                      `json:"queryType,omitempty"`
	EventTypes       []string                  `json:"eventTypes,omitempty"`
	ComparisonDelta  *int                      `json:"comparisonDelta,omitempty"`
	ResolveThreshold *float64                  `json:"resolveThreshold,omitempty"`
	Owner            string                    `json:"owner,omitempty"`
}

type MetricAlertTriggerInput struct {
	Label            string                   `json:"label"`
	AlertThreshold   float64                  `json:"alertThreshold"`
	ResolveThreshold *float64                 `json:"resolveThreshold,omitempty"`
	Actions          []MetricAlertActionInput `json:"actions"`
}

type MetricAlertActionInput struct {
	Type             string  `json:"type"`
	TargetType       string  `json:"targetType"`
	TargetIdentifier string  `json:"targetIdentifier"`
	InputChannelID   *string `json:"inputChannelId,omitempty"`
	IntegrationID    *string `json:"integrationId,omitempty"`
	SentryAppID      *string `json:"sentryAppId,omitempty"`
	Priority         *string `json:"priority,omitempty"`
}

type Release struct {
	ID           int              `json:"id" mapstructure:"id"`
	Version      string           `json:"version" mapstructure:"version"`
	ShortVersion string           `json:"shortVersion" mapstructure:"shortVersion"`
	Ref          string           `json:"ref" mapstructure:"ref"`
	URL          string           `json:"url" mapstructure:"url"`
	DateCreated  string           `json:"dateCreated" mapstructure:"dateCreated"`
	DateReleased string           `json:"dateReleased" mapstructure:"dateReleased"`
	CommitCount  int              `json:"commitCount" mapstructure:"commitCount"`
	DeployCount  int              `json:"deployCount" mapstructure:"deployCount"`
	NewGroups    int              `json:"newGroups" mapstructure:"newGroups"`
	Projects     []ReleaseProject `json:"projects" mapstructure:"projects"`
	LastDeploy   *Deploy          `json:"lastDeploy,omitempty" mapstructure:"lastDeploy"`
}

type ReleaseProject struct {
	Name string `json:"name" mapstructure:"name"`
	Slug string `json:"slug" mapstructure:"slug"`
}

type ReleaseCommit struct {
	ID          string `json:"id" mapstructure:"id"`
	Repository  string `json:"repository,omitempty" mapstructure:"repository"`
	Message     string `json:"message,omitempty" mapstructure:"message"`
	AuthorName  string `json:"author_name,omitempty" mapstructure:"author_name"`
	AuthorEmail string `json:"author_email,omitempty" mapstructure:"author_email"`
	Timestamp   string `json:"timestamp,omitempty" mapstructure:"timestamp"`
}

type ReleaseRef struct {
	Repository     string `json:"repository" mapstructure:"repository"`
	Commit         string `json:"commit" mapstructure:"commit"`
	PreviousCommit string `json:"previousCommit,omitempty" mapstructure:"previousCommit"`
}

type CreateReleaseRequest struct {
	Version      string          `json:"version"`
	Projects     []string        `json:"projects"`
	Ref          string          `json:"ref,omitempty"`
	URL          string          `json:"url,omitempty"`
	DateReleased string          `json:"dateReleased,omitempty"`
	Commits      []ReleaseCommit `json:"commits,omitempty"`
	Refs         []ReleaseRef    `json:"refs,omitempty"`
}

type Deploy struct {
	ID             string   `json:"id" mapstructure:"id"`
	Environment    string   `json:"environment" mapstructure:"environment"`
	Name           string   `json:"name" mapstructure:"name"`
	URL            string   `json:"url" mapstructure:"url"`
	DateStarted    string   `json:"dateStarted" mapstructure:"dateStarted"`
	DateFinished   string   `json:"dateFinished" mapstructure:"dateFinished"`
	ReleaseVersion string   `json:"releaseVersion,omitempty" mapstructure:"releaseVersion"`
	Projects       []string `json:"projects,omitempty" mapstructure:"projects"`
}

type CreateDeployRequest struct {
	Environment  string   `json:"environment"`
	Name         string   `json:"name,omitempty"`
	URL          string   `json:"url,omitempty"`
	Projects     []string `json:"projects,omitempty"`
	DateStarted  string   `json:"dateStarted,omitempty"`
	DateFinished string   `json:"dateFinished,omitempty"`
}

func (c *Client) ListOrganizations() ([]Organization, error) {
	responseBody, err := c.doJSON(http.MethodGet, "/api/0/organizations/", nil)
	if err != nil {
		return nil, err
	}

	organizations := []Organization{}
	if err := json.Unmarshal(responseBody, &organizations); err != nil {
		return nil, err
	}

	return organizations, nil
}

func (c *Client) GetAuthIdentity() (*AuthIdentity, error) {
	responseBody, err := c.doJSON(http.MethodGet, "/api/0/auth/", nil)
	if err != nil {
		return nil, err
	}

	identity := AuthIdentity{}
	if err := json.Unmarshal(responseBody, &identity); err != nil {
		return nil, err
	}

	return &identity, nil
}

func (c *Client) GetOrganization() (*Organization, error) {
	responseBody, err := c.doJSON(http.MethodGet, fmt.Sprintf("/api/0/organizations/%s/", c.orgSlug), nil)
	if err != nil {
		return nil, err
	}

	organization := Organization{}
	if err := json.Unmarshal(responseBody, &organization); err != nil {
		return nil, err
	}

	return &organization, nil
}

func (c *Client) ListProjects() ([]ProjectSummary, error) {
	responseBody, err := c.doJSON(http.MethodGet, fmt.Sprintf("/api/0/organizations/%s/projects/", c.orgSlug), nil)
	if err != nil {
		return nil, err
	}

	projects := []Project{}
	if err := json.Unmarshal(responseBody, &projects); err != nil {
		return nil, err
	}

	result := make([]ProjectSummary, 0, len(projects))
	for _, project := range projects {
		result = append(result, ProjectSummary{
			ID:   project.ID,
			Slug: project.Slug,
			Name: project.Name,
		})
	}

	return result, nil
}

func (c *Client) ListTeams() ([]TeamSummary, error) {
	responseBody, err := c.doJSON(http.MethodGet, fmt.Sprintf("/api/0/organizations/%s/teams/", c.orgSlug), nil)
	if err != nil {
		return nil, err
	}

	teams := []Team{}
	if err := json.Unmarshal(responseBody, &teams); err != nil {
		return nil, err
	}

	result := make([]TeamSummary, 0, len(teams))
	for _, team := range teams {
		result = append(result, TeamSummary{
			ID:   team.ID,
			Slug: team.Slug,
			Name: team.Name,
		})
	}

	return result, nil
}

func (c *Client) ListProjectMembers(projectSlug string) ([]ProjectMember, error) {
	responseBody, err := c.doJSON(
		http.MethodGet,
		fmt.Sprintf("/api/0/projects/%s/%s/members/", c.orgSlug, projectSlug),
		nil,
	)
	if err != nil {
		return nil, err
	}

	members := []ProjectMember{}
	if err := json.Unmarshal(responseBody, &members); err != nil {
		return nil, err
	}

	return members, nil
}

func (c *Client) ListProjectTeams(projectSlug string) ([]Team, error) {
	responseBody, err := c.doJSON(
		http.MethodGet,
		fmt.Sprintf("/api/0/projects/%s/%s/teams/", c.orgSlug, projectSlug),
		nil,
	)
	if err != nil {
		return nil, err
	}

	teams := []Team{}
	if err := json.Unmarshal(responseBody, &teams); err != nil {
		return nil, err
	}

	return teams, nil
}

func (c *Client) ListProjectAssignees(projectSlug string) ([]AssigneeResource, error) {
	members, err := c.ListProjectMembers(projectSlug)
	if err != nil {
		return nil, err
	}

	teams, err := c.ListProjectTeams(projectSlug)
	if err != nil {
		return nil, err
	}

	resources := make([]AssigneeResource, 0, len(members)+len(teams))
	for _, member := range members {
		value := ""
		label := ""

		if member.User != nil {
			userID := strings.TrimSpace(member.User.ID)
			switch {
			case userID != "":
				value = "user:" + userID
			case strings.TrimSpace(member.User.Username) != "":
				value = strings.TrimSpace(member.User.Username)
			case strings.TrimSpace(member.User.Email) != "":
				value = strings.TrimSpace(member.User.Email)
			}

			switch {
			case strings.TrimSpace(member.User.Name) != "":
				label = strings.TrimSpace(member.User.Name)
			case strings.TrimSpace(member.User.Email) != "":
				label = strings.TrimSpace(member.User.Email)
			case strings.TrimSpace(member.User.Username) != "":
				label = strings.TrimSpace(member.User.Username)
			}
		}

		if value == "" {
			switch {
			case strings.TrimSpace(member.ID) != "":
				value = "user:" + strings.TrimSpace(member.ID)
			case strings.TrimSpace(member.Email) != "":
				value = strings.TrimSpace(member.Email)
			}
		}

		if label == "" {
			switch {
			case strings.TrimSpace(member.Name) != "":
				label = strings.TrimSpace(member.Name)
			case strings.TrimSpace(member.Email) != "":
				label = strings.TrimSpace(member.Email)
			}
		}

		if value == "" || label == "" {
			continue
		}

		resources = append(resources, AssigneeResource{
			ID:   value,
			Name: "User · " + label,
		})
	}

	for _, team := range teams {
		teamID := strings.TrimSpace(team.ID)
		teamName := strings.TrimSpace(team.Name)
		if teamID == "" || teamName == "" {
			continue
		}

		resources = append(resources, AssigneeResource{
			ID:   "team:" + teamID,
			Name: "Team · " + teamName,
		})
	}

	return resources, nil
}

func (c *Client) ListIssues() ([]Issue, error) {
	responseBody, err := c.doJSON(
		http.MethodGet,
		fmt.Sprintf("/api/0/organizations/%s/issues/?query=&limit=100", c.orgSlug),
		nil,
	)
	if err != nil {
		return nil, err
	}

	issues := []Issue{}
	if err := json.Unmarshal(responseBody, &issues); err != nil {
		return nil, err
	}

	return issues, nil
}

func (c *Client) ListReleases() ([]Release, error) {
	responseBody, err := c.doJSON(
		http.MethodGet,
		fmt.Sprintf("/api/0/organizations/%s/releases/", c.orgSlug),
		nil,
	)
	if err != nil {
		return nil, wrapReleaseScopeError(err)
	}

	releases := []Release{}
	if err := json.Unmarshal(responseBody, &releases); err != nil {
		return nil, err
	}

	return releases, nil
}

func (c *Client) ValidateReleaseAccess() error {
	_, err := c.doJSON(
		http.MethodGet,
		fmt.Sprintf("/api/0/organizations/%s/releases/?per_page=1", c.orgSlug),
		nil,
	)
	if err != nil {
		return wrapReleaseScopeError(err)
	}

	return nil
}

func (c *Client) GetIssue(issueID string) (*Issue, error) {
	responseBody, err := c.doJSON(
		http.MethodGet,
		fmt.Sprintf("/api/0/organizations/%s/issues/%s/", c.orgSlug, url.PathEscape(issueID)),
		nil,
	)
	if err != nil {
		return nil, err
	}

	issue := Issue{}
	if err := json.Unmarshal(responseBody, &issue); err != nil {
		return nil, err
	}

	return &issue, nil
}

func (c *Client) ListIssueEvents(issueID string) ([]IssueEvent, error) {
	responseBody, err := c.doJSON(
		http.MethodGet,
		fmt.Sprintf("/api/0/organizations/%s/issues/%s/events/?limit=10", c.orgSlug, url.PathEscape(issueID)),
		nil,
	)
	if err != nil {
		return nil, err
	}

	events := []IssueEvent{}
	if err := json.Unmarshal(responseBody, &events); err != nil {
		return nil, err
	}

	return events, nil
}

func (c *Client) ListSentryApps(orgSlug string) ([]SentryApp, error) {
	responseBody, err := c.doJSON(http.MethodGet, fmt.Sprintf("/api/0/organizations/%s/sentry-apps/", orgSlug), nil)
	if err != nil {
		return nil, err
	}

	apps := []SentryApp{}
	if err := json.Unmarshal(responseBody, &apps); err != nil {
		return nil, err
	}

	return apps, nil
}

func (c *Client) UpdateSentryApp(appSlug string, request UpdateSentryAppRequest) (*SentryApp, error) {
	responseBody, err := c.doJSON(http.MethodPut, fmt.Sprintf("/api/0/sentry-apps/%s/", appSlug), request)
	if err != nil {
		return nil, err
	}

	app := SentryApp{}
	if err := json.Unmarshal(responseBody, &app); err != nil {
		return nil, err
	}

	return &app, nil
}

func (c *Client) UpdateIssue(issueID string, request UpdateIssueRequest) (*Issue, error) {
	_, err := c.doJSON(
		http.MethodPut,
		fmt.Sprintf("/api/0/organizations/%s/issues/?id=%s", c.orgSlug, url.QueryEscape(issueID)),
		request,
	)
	if err != nil {
		return nil, err
	}

	return c.GetIssue(issueID)
}

func (c *Client) ListAlertRules() ([]MetricAlertRule, error) {
	alertRules := []MetricAlertRule{}
	path := fmt.Sprintf("/api/0/organizations/%s/alert-rules/", c.orgSlug)

	for {
		responseBody, headers, err := c.doJSONResponse(http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		page := []MetricAlertRule{}
		if err := json.Unmarshal(responseBody, &page); err != nil {
			return nil, err
		}

		alertRules = append(alertRules, page...)

		nextPath := nextCursorPath(headers.Get("Link"))
		if nextPath == "" {
			return alertRules, nil
		}

		path = nextPath
	}
}

func (c *Client) GetAlertRule(alertRuleID string) (*MetricAlertRule, error) {
	responseBody, err := c.doJSON(
		http.MethodGet,
		fmt.Sprintf("/api/0/organizations/%s/alert-rules/%s/", c.orgSlug, url.PathEscape(alertRuleID)),
		nil,
	)
	if err != nil {
		return nil, err
	}

	alertRule := MetricAlertRule{}
	if err := json.Unmarshal(responseBody, &alertRule); err != nil {
		return nil, err
	}

	return &alertRule, nil
}

func (c *Client) CreateAlertRule(request CreateOrUpdateMetricAlertRuleRequest) (*MetricAlertRule, error) {
	responseBody, err := c.doJSON(
		http.MethodPost,
		fmt.Sprintf("/api/0/organizations/%s/alert-rules/", c.orgSlug),
		request,
	)
	if err != nil {
		return nil, err
	}

	alertRule := MetricAlertRule{}
	if err := json.Unmarshal(responseBody, &alertRule); err != nil {
		return nil, err
	}

	return &alertRule, nil
}

func (c *Client) UpdateAlertRule(alertRuleID string, request CreateOrUpdateMetricAlertRuleRequest) (*MetricAlertRule, error) {
	responseBody, err := c.doJSON(
		http.MethodPut,
		fmt.Sprintf("/api/0/organizations/%s/alert-rules/%s/", c.orgSlug, url.PathEscape(alertRuleID)),
		request,
	)
	if err != nil {
		return nil, err
	}

	alertRule := MetricAlertRule{}
	if err := json.Unmarshal(responseBody, &alertRule); err != nil {
		return nil, err
	}

	return &alertRule, nil
}

func (c *Client) DeleteAlertRule(alertRuleID string) error {
	_, err := c.doJSON(
		http.MethodDelete,
		fmt.Sprintf("/api/0/organizations/%s/alert-rules/%s/", c.orgSlug, url.PathEscape(alertRuleID)),
		nil,
	)
	return err
}

func (c *Client) CreateRelease(request CreateReleaseRequest) (*Release, error) {
	responseBody, err := c.doJSON(
		http.MethodPost,
		fmt.Sprintf("/api/0/organizations/%s/releases/", c.orgSlug),
		request,
	)
	if err != nil {
		return nil, wrapReleaseScopeError(err)
	}

	release := Release{}
	if err := json.Unmarshal(responseBody, &release); err != nil {
		return nil, err
	}

	return &release, nil
}

func (c *Client) CreateDeploy(version string, request CreateDeployRequest) (*Deploy, error) {
	responseBody, err := c.doJSON(
		http.MethodPost,
		fmt.Sprintf("/api/0/organizations/%s/releases/%s/deploys/", c.orgSlug, url.PathEscape(version)),
		request,
	)
	if err != nil {
		return nil, wrapReleaseScopeError(err)
	}

	deploy := Deploy{}
	if err := json.Unmarshal(responseBody, &deploy); err != nil {
		return nil, err
	}

	deploy.ReleaseVersion = version
	if len(request.Projects) > 0 {
		deploy.Projects = append([]string(nil), request.Projects...)
	}

	return &deploy, nil
}

func (c *Client) doJSON(method, path string, payload any) ([]byte, error) {
	responseBody, _, err := c.doJSONResponse(method, path, payload)
	return responseBody, err
}

func (c *Client) doJSONResponse(method, path string, payload any) ([]byte, http.Header, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, err
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.userToken)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpContext.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, nil, &apiError{StatusCode: resp.StatusCode, Body: string(responseBody)}
	}

	return responseBody, resp.Header, nil
}

func nextCursorPath(linkHeader string) string {
	for _, section := range strings.Split(linkHeader, ",") {
		section = strings.TrimSpace(section)
		if section == "" || !strings.Contains(section, `rel="next"`) || !strings.Contains(section, `results="true"`) {
			continue
		}

		start := strings.Index(section, "<")
		end := strings.Index(section, ">")
		if start == -1 || end == -1 || end <= start+1 {
			continue
		}

		nextURL, err := url.Parse(section[start+1 : end])
		if err != nil {
			continue
		}

		if nextURL.RawPath != "" {
			if nextURL.RawQuery == "" {
				return nextURL.RawPath
			}
			return nextURL.RawPath + "?" + nextURL.RawQuery
		}

		if nextURL.Path == "" {
			continue
		}

		if nextURL.RawQuery == "" {
			return nextURL.Path
		}

		return nextURL.Path + "?" + nextURL.RawQuery
	}

	return ""
}
