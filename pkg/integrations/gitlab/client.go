package gitlab

import (
	"fmt"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const apiVersion = "v4"
const DefaultBaseURL = "https://gitlab.com"

type Client struct {
	baseURL    string
	token      string
	authType   string
	groupID    string
	httpClient core.HTTPContext
}

func NewClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	config, err := ctx.GetConfig("authType")
	if err != nil {
		return nil, fmt.Errorf("failed to get authType: %v", err)
	}
	authType := string(config)

	baseURLBytes, _ := ctx.GetConfig("baseUrl")
	baseURL := normalizeBaseURL(string(baseURLBytes))

	var token string
	
	groupIDBytes, err := ctx.GetConfig("groupId")
	if err != nil || len(groupIDBytes) == 0 {
		return nil, fmt.Errorf("groupId is required")
	}
	groupID := string(groupIDBytes)

	switch authType {
	case AuthTypePersonalAccessToken:
		tokenBytes, err := ctx.GetConfig("personalAccessToken")
		if err != nil {
			return nil, err
		}
		token = string(tokenBytes)
		if token == "" {
			return nil, fmt.Errorf("personal access token not found")
		}

	case AuthTypeAppOAuth:
		secrets, err := ctx.GetSecrets()
		if err != nil {
			return nil, err
		}
		for _, secret := range secrets {
			if secret.Name == OAuthAccessToken {
				token = string(secret.Value)
				break
			}
		}
		if token == "" {
			return nil, fmt.Errorf("OAuth access token not found")
		}

	default:
		return nil, fmt.Errorf("unknown auth type: %s", authType)
	}

	return &Client{
		baseURL:    baseURL,
		token:      token,
		authType:   authType,
		groupID:    groupID,
		httpClient: httpClient,
	}, nil
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	if c.authType == AuthTypePersonalAccessToken {
		req.Header.Set("PRIVATE-TOKEN", c.token)
	} else {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return c.httpClient.Do(req)
}

func (c *Client) Verify() error {
	if c.groupID == "" {
		return fmt.Errorf("groupID is missing") 	
	}

	url := fmt.Sprintf("%s/api/%s/groups/%s", c.baseURL, apiVersion, c.groupID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("access validation failed for group %s: status %d", c.groupID, resp.StatusCode)
	}
	return nil
}
