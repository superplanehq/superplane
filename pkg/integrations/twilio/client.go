package twilio

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const twilioAPIBase = "https://api.twilio.com/2010-04-01"

type Client struct {
	AccountSID string
	AuthToken  string
	FromNumber string
}

func NewClient(ctx core.IntegrationContext) (*Client, error) {
	accountSID, err := ctx.GetConfig("accountSid")
	if err != nil {
		return nil, fmt.Errorf("failed to get account SID: %w", err)
	}
	authToken, err := ctx.GetConfig("authToken")
	if err != nil {
		return nil, fmt.Errorf("failed to get auth token: %w", err)
	}
	fromNumber, err := ctx.GetConfig("fromNumber")
	if err != nil {
		return nil, fmt.Errorf("failed to get from number: %w", err)
	}

	return &Client{
		AccountSID: string(accountSID),
		AuthToken:  string(authToken),
		FromNumber: string(fromNumber),
	}, nil
}

// CallResponse is the response from creating a call.
type CallResponse struct {
	SID         string `json:"sid"`
	Status      string `json:"status"`
	To          string `json:"to"`
	From        string `json:"from"`
	DateCreated string `json:"date_created"`
	Duration    string `json:"duration"`
}

// MessageResponse is the response from sending an SMS.
type MessageResponse struct {
	SID         string `json:"sid"`
	Status      string `json:"status"`
	To          string `json:"to"`
	From        string `json:"from"`
	Body        string `json:"body"`
	DateCreated string `json:"date_created"`
}

// AccountResponse is used to verify credentials.
type AccountResponse struct {
	SID          string `json:"sid"`
	FriendlyName string `json:"friendly_name"`
	Status       string `json:"status"`
}

// MakeCall places an outbound call with a TTS message.
func (c *Client) MakeCall(to, message string, timeout int) (*CallResponse, error) {
	twiml := fmt.Sprintf("<Response><Say>%s</Say></Response>", xmlEscape(message))

	data := url.Values{}
	data.Set("To", to)
	data.Set("From", c.FromNumber)
	data.Set("Twiml", twiml)
	if timeout > 0 {
		data.Set("Timeout", fmt.Sprintf("%d", timeout))
	}

	endpoint := fmt.Sprintf("%s/Accounts/%s/Calls.json", twilioAPIBase, c.AccountSID)
	body, err := c.doPost(endpoint, data)
	if err != nil {
		return nil, fmt.Errorf("make call: %w", err)
	}

	var resp CallResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode call response: %w", err)
	}
	return &resp, nil
}

// SendSMS sends an outbound SMS message.
func (c *Client) SendSMS(to, messageBody string) (*MessageResponse, error) {
	data := url.Values{}
	data.Set("To", to)
	data.Set("From", c.FromNumber)
	data.Set("Body", messageBody)

	endpoint := fmt.Sprintf("%s/Accounts/%s/Messages.json", twilioAPIBase, c.AccountSID)
	body, err := c.doPost(endpoint, data)
	if err != nil {
		return nil, fmt.Errorf("send SMS: %w", err)
	}

	var resp MessageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode SMS response: %w", err)
	}
	return &resp, nil
}

// GetAccount verifies the credentials by fetching account info.
func (c *Client) GetAccount() (*AccountResponse, error) {
	endpoint := fmt.Sprintf("%s/Accounts/%s.json", twilioAPIBase, c.AccountSID)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.AccountSID, c.AuthToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("twilio API error (%d): %s", resp.StatusCode, string(body))
	}

	var account AccountResponse
	if err := json.Unmarshal(body, &account); err != nil {
		return nil, err
	}
	return &account, nil
}

func (c *Client) doPost(endpoint string, data url.Values) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.AccountSID, c.AuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("twilio API error (%d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
