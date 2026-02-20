package servicenow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	InstanceURL string
	Token       string
	http        core.HTTPContext
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	instanceURL, err := ctx.GetConfig("instanceUrl")
	if err != nil {
		return nil, fmt.Errorf("error getting instanceUrl: %w", err)
	}

	secrets, err := ctx.GetSecrets()
	if err != nil {
		return nil, fmt.Errorf("failed to get secrets: %w", err)
	}

	var accessToken string
	for _, secret := range secrets {
		if secret.Name == OAuthAccessToken {
			accessToken = string(secret.Value)
			break
		}
	}

	if accessToken == "" {
		return nil, fmt.Errorf("OAuth access token not found")
	}

	return &Client{
		InstanceURL: string(instanceURL),
		Token:       accessToken,
		http:        http,
	}, nil
}

func (c *Client) authHeader() string {
	return fmt.Sprintf("Bearer %s", c.Token)
}

func (c *Client) execRequest(method, path string, body io.Reader) ([]byte, error) {
	url := fmt.Sprintf("%s%s", strings.TrimRight(c.InstanceURL, "/"), path)

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.authHeader())

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

type CreateIncidentParams struct {
	ShortDescription string `json:"short_description"`
	Description      string `json:"description,omitempty"`
	State            string `json:"state,omitempty"`
	Urgency          string `json:"urgency,omitempty"`
	Impact           string `json:"impact,omitempty"`
	Category         string `json:"category,omitempty"`
	Subcategory      string `json:"subcategory,omitempty"`
	AssignmentGroup  string `json:"assignment_group,omitempty"`
	AssignedTo       string `json:"assigned_to,omitempty"`
	Caller           string `json:"caller_id,omitempty"`
	ResolutionCode   string `json:"close_code,omitempty"`
	ResolutionNotes  string `json:"close_notes,omitempty"`
	OnHoldReason     string `json:"hold_reason,omitempty"`
}

func (c *Client) CreateIncident(params CreateIncidentParams) (map[string]any, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, "/api/now/table/incident", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Result map[string]any `json:"result"`
	}
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return response.Result, nil
}

func (c *Client) GetIncident(sysID string) (*IncidentRecord, error) {
	path := fmt.Sprintf("/api/now/table/incident/%s?sysparm_fields=sys_id,number,short_description,state,urgency,impact,priority,category,subcategory,sys_created_on,sys_updated_on", sysID)
	responseBody, err := c.execRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var response struct {
		Result IncidentRecord `json:"result"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}
	return &response.Result, nil
}

func (c *Client) ListIncidents(limit int) ([]IncidentRecord, error) {
	params := url.Values{}
	params.Set("sysparm_fields", "sys_id,number,short_description")
	params.Set("sysparm_limit", fmt.Sprintf("%d", limit))
	params.Set("sysparm_query", "active=true")
	path := "/api/now/table/incident?" + params.Encode()
	responseBody, err := c.execRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var response struct {
		Result []IncidentRecord `json:"result"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}
	return response.Result, nil
}

func (c *Client) ValidateConnection() error {
	_, err := c.execRequest(http.MethodGet, "/api/now/table/incident?sysparm_limit=1", nil)
	return err
}

func (c *Client) GetUser(sysID string) (*UserRecord, error) {
	path := fmt.Sprintf("/api/now/table/sys_user/%s?sysparm_fields=sys_id,name,email", sysID)
	responseBody, err := c.execRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Result UserRecord `json:"result"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return &response.Result, nil
}

func (c *Client) GetAssignmentGroup(sysID string) (*AssignmentGroupRecord, error) {
	path := fmt.Sprintf("/api/now/table/sys_user_group/%s?sysparm_fields=sys_id,name", sysID)
	responseBody, err := c.execRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Result AssignmentGroupRecord `json:"result"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return &response.Result, nil
}

type UserRecord struct {
	SysID string `json:"sys_id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (c *Client) ListUsers() ([]UserRecord, error) {
	path := "/api/now/table/sys_user?sysparm_query=active=true&sysparm_fields=sys_id,name,email&sysparm_limit=200"
	responseBody, err := c.execRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Result []UserRecord `json:"result"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return response.Result, nil
}

type AssignmentGroupRecord struct {
	SysID string `json:"sys_id" mapstructure:"sys_id"`
	Name  string `json:"name" mapstructure:"name"`
}

func (c *Client) ListGroupMembers(groupSysID string) ([]UserRecord, error) {
	params := url.Values{}
	params.Set("sysparm_query", "group="+groupSysID)
	params.Set("sysparm_fields", "user")
	params.Set("sysparm_limit", "200")
	path := "/api/now/table/sys_user_grmember?" + params.Encode()
	responseBody, err := c.execRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Result []struct {
			User struct {
				Value string `json:"value"`
			} `json:"user"`
		} `json:"result"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	if len(response.Result) == 0 {
		return []UserRecord{}, nil
	}

	userIDs := make([]string, 0, len(response.Result))
	for _, member := range response.Result {
		if member.User.Value != "" {
			userIDs = append(userIDs, member.User.Value)
		}
	}

	if len(userIDs) == 0 {
		return []UserRecord{}, nil
	}

	userParams := url.Values{}
	userParams.Set("sysparm_query", "sys_idIN"+strings.Join(userIDs, ",")+"^active=true")
	userParams.Set("sysparm_fields", "sys_id,name,email")
	userParams.Set("sysparm_limit", "200")
	usersPath := "/api/now/table/sys_user?" + userParams.Encode()
	usersBody, err := c.execRequest(http.MethodGet, usersPath, nil)
	if err != nil {
		return nil, err
	}

	var usersResponse struct {
		Result []UserRecord `json:"result"`
	}

	err = json.Unmarshal(usersBody, &usersResponse)
	if err != nil {
		return nil, fmt.Errorf("error parsing users response: %w", err)
	}

	return usersResponse.Result, nil
}

func (c *Client) ListAssignmentGroups() ([]AssignmentGroupRecord, error) {
	path := "/api/now/table/sys_user_group?sysparm_query=active=true&sysparm_fields=sys_id,name&sysparm_limit=200"
	responseBody, err := c.execRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Result []AssignmentGroupRecord `json:"result"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return response.Result, nil
}

type ChoiceRecord struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

func (c *Client) ListCategories() ([]ChoiceRecord, error) {
	path := "/api/now/table/sys_choice?sysparm_query=name=incident^element=category&sysparm_fields=label,value&sysparm_limit=200"
	responseBody, err := c.execRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Result []ChoiceRecord `json:"result"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return response.Result, nil
}

func (c *Client) ListSubcategories(category string) ([]ChoiceRecord, error) {
	query := "name=incident^element=subcategory"
	if category != "" {
		query += "^dependent_value=" + category
	}

	params := url.Values{}
	params.Set("sysparm_query", query)
	params.Set("sysparm_fields", "label,value")
	params.Set("sysparm_limit", "200")
	path := "/api/now/table/sys_choice?" + params.Encode()
	responseBody, err := c.execRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Result []ChoiceRecord `json:"result"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	return response.Result, nil
}
