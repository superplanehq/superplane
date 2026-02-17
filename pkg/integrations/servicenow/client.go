package servicenow

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	AuthType    string
	InstanceURL string
	Username    string
	Password    string
	Token       string
	http        core.HTTPContext
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	instanceURL, err := ctx.GetConfig("instanceUrl")
	if err != nil {
		return nil, fmt.Errorf("error getting instanceUrl: %v", err)
	}

	authType, err := ctx.GetConfig("authType")
	if err != nil {
		return nil, fmt.Errorf("error getting authType: %v", err)
	}

	switch string(authType) {
	case AuthTypeBasicAuth:
		username, err := ctx.GetConfig("username")
		if err != nil {
			return nil, fmt.Errorf("error getting username: %v", err)
		}

		password, err := ctx.GetConfig("password")
		if err != nil {
			return nil, fmt.Errorf("error getting password: %v", err)
		}

		return &Client{
			AuthType:    AuthTypeBasicAuth,
			InstanceURL: string(instanceURL),
			Username:    string(username),
			Password:    string(password),
			http:        http,
		}, nil

	case AuthTypeOAuth:
		secrets, err := ctx.GetSecrets()
		if err != nil {
			return nil, fmt.Errorf("failed to get secrets: %v", err)
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
			AuthType:    AuthTypeOAuth,
			InstanceURL: string(instanceURL),
			Token:       accessToken,
			http:        http,
		}, nil
	}

	return nil, fmt.Errorf("unknown auth type %s", authType)
}

func (c *Client) authHeader() string {
	if c.AuthType == AuthTypeOAuth {
		return fmt.Sprintf("Bearer %s", c.Token)
	}

	credentials := fmt.Sprintf("%s:%s", c.Username, c.Password)
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(credentials))
}

func (c *Client) execRequest(method, path string, body io.Reader) ([]byte, error) {
	url := fmt.Sprintf("%s%s", strings.TrimRight(c.InstanceURL, "/"), path)

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.authHeader())

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
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
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, "/api/now/table/incident", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response map[string]any
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response, nil
}

func (c *Client) GetIncident(sysID string) (map[string]any, error) {
	path := fmt.Sprintf("/api/now/table/incident/%s", sysID)
	responseBody, err := c.execRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response map[string]any
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response, nil
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
		return nil, fmt.Errorf("error parsing response: %v", err)
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
		return nil, fmt.Errorf("error parsing response: %v", err)
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
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Result, nil
}

type AssignmentGroupRecord struct {
	SysID string `json:"sys_id"`
	Name  string `json:"name"`
}

func (c *Client) ListGroupMembers(groupSysID string) ([]UserRecord, error) {
	path := fmt.Sprintf("/api/now/table/sys_user_grmember?sysparm_query=group=%s&sysparm_fields=user&sysparm_limit=200", groupSysID)
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
		return nil, fmt.Errorf("error parsing response: %v", err)
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

	query := "sys_idIN" + strings.Join(userIDs, ",") + "^active=true"
	usersPath := fmt.Sprintf("/api/now/table/sys_user?sysparm_query=%s&sysparm_fields=sys_id,name,email&sysparm_limit=200", query)
	usersBody, err := c.execRequest(http.MethodGet, usersPath, nil)
	if err != nil {
		return nil, err
	}

	var usersResponse struct {
		Result []UserRecord `json:"result"`
	}

	err = json.Unmarshal(usersBody, &usersResponse)
	if err != nil {
		return nil, fmt.Errorf("error parsing users response: %v", err)
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
		return nil, fmt.Errorf("error parsing response: %v", err)
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
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Result, nil
}

func (c *Client) ListSubcategories(category string) ([]ChoiceRecord, error) {
	query := "name=incident^element=subcategory"
	if category != "" {
		query += "^dependent_value=" + url.QueryEscape(category)
	}

	path := fmt.Sprintf("/api/now/table/sys_choice?sysparm_query=%s&sysparm_fields=label,value&sysparm_limit=200", query)
	responseBody, err := c.execRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Result []ChoiceRecord `json:"result"`
	}

	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return response.Result, nil
}
