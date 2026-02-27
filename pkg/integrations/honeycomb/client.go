package honeycomb

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	secretNameIngestKey        = "honeycomb_ingest_key"
	secretNameConfigurationKey = "honeycomb_configuration_key"
)

type Client struct {
	BaseURL        string
	ManagementKey  string
	http           core.HTTPContext
	integrationCtx core.IntegrationContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	siteAny, err := ctx.GetConfig("site")
	if err != nil {
		siteAny = []byte("api.honeycomb.io")
	}
	site := strings.TrimSpace(string(siteAny))
	if site == "" {
		site = "api.honeycomb.io"
	}

	baseURL := site
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}

	mkAny, err := ctx.GetConfig("managementKey")
	if err != nil {
		return nil, fmt.Errorf("managementKey is required")
	}
	mk := strings.TrimSpace(string(mkAny))
	if mk == "" {
		return nil, fmt.Errorf("managementKey is required")
	}

	return &Client{
		BaseURL:        baseURL,
		ManagementKey:  mk,
		http:           httpCtx,
		integrationCtx: ctx,
	}, nil
}

// bearerFromManagementKey normalizes the management key into "keyID:secret" format
// required by the Honeycomb v2 API Authorization header.
func (c *Client) bearerFromManagementKey() (string, error) {
	mk := strings.TrimSpace(c.ManagementKey)
	if mk == "" {
		return "", fmt.Errorf("managementKey is empty")
	}

	if strings.HasPrefix(strings.ToLower(mk), "bearer ") {
		mk = strings.TrimSpace(mk[len("bearer "):])
	}

	parts := strings.SplitN(mk, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("managementKey must be in format <keyID>:<secret>")
	}

	id := strings.TrimSpace(parts[0])
	sec := strings.TrimSpace(parts[1])

	if id == "" || sec == "" {
		return "", fmt.Errorf("managementKey must be in format <keyID>:<secret> (both parts required)")
	}

	return id + ":" + sec, nil
}

// newReqV1 builds a request for the Honeycomb /1 API using the configuration key secret.
func (c *Client) newReqV1(method, path string, body io.Reader) (*http.Request, error) {
	u, _ := url.Parse(c.BaseURL)
	u.Path = path

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	cfgKey, err := c.getSecretValue(secretNameConfigurationKey)
	if err != nil {
		return nil, fmt.Errorf("missing configuration key secret %q: %w", secretNameConfigurationKey, err)
	}
	cfgKey = strings.TrimSpace(cfgKey)
	if cfgKey == "" {
		return nil, fmt.Errorf("configuration key secret %q is empty", secretNameConfigurationKey)
	}

	req.Header.Set("X-Honeycomb-Team", cfgKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return req, nil
}

// newReqV2 builds a request for the Honeycomb /2 API using the management key.
func (c *Client) newReqV2(method, path string, body io.Reader) (*http.Request, error) {
	u, _ := url.Parse(c.BaseURL)
	u.Path = path

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	bearer, err := c.bearerFromManagementKey()
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("Content-Type", "application/vnd.api+json")
	return req, nil
}

func (c *Client) do(req *http.Request) ([]byte, int, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	return b, resp.StatusCode, nil
}

func (c *Client) ValidateManagementKey(teamSlug string) error {
	teamSlug = strings.TrimSpace(teamSlug)
	if teamSlug == "" {
		return fmt.Errorf("teamSlug is required")
	}

	req, err := c.newReqV2(
		http.MethodGet,
		fmt.Sprintf("/2/teams/%s/environments", url.PathEscape(teamSlug)),
		nil,
	)
	if err != nil {
		return err
	}

	body, code, err := c.do(req)
	if err != nil {
		return fmt.Errorf("failed to validate management key: %w", err)
	}

	switch code {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return fmt.Errorf("invalid management key (401): check keyID:secret and that you're using the correct site (US vs EU). body=%s", string(body))
	case http.StatusForbidden:
		return fmt.Errorf("management key forbidden (403): missing permissions/scopes. body=%s", string(body))
	default:
		return fmt.Errorf("management key validation failed (http %d): %s", code, string(body))
	}
}

func (c *Client) pingV1WithConfigKey() (int, []byte, error) {
	req, err := c.newReqV1(http.MethodGet, "/1/auth", nil)
	if err != nil {
		return 0, nil, err
	}

	b, code, err := c.do(req)
	return code, b, err
}

// pingV1WithKey pings /1/auth with the given API key to validate it works.
// Honeycomb accepts both configuration and ingest keys for this endpoint.
func (c *Client) pingV1WithKey(key string) (int, []byte, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return 0, nil, fmt.Errorf("key is empty")
	}
	u, _ := url.Parse(c.BaseURL)
	u.Path = "/1/auth"
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("X-Honeycomb-Team", key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	b, code, err := c.do(req)
	return code, b, err
}

func (c *Client) pingV1WithIngestKey() (int, []byte, error) {
	ingestKey, err := c.getSecretValue(secretNameIngestKey)
	if err != nil {
		return 0, nil, err
	}
	return c.pingV1WithKey(ingestKey)
}

type listEnvironmentsResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Name string `json:"name"`
			Slug string `json:"slug"`
		} `json:"attributes"`
	} `json:"data"`
}

func (c *Client) getEnvironmentID(teamSlug, envSlug string) (string, error) {
	envSlug = strings.TrimSpace(envSlug)
	if envSlug == "" {
		return "", fmt.Errorf("environmentSlug is required")
	}

	req, err := c.newReqV2(http.MethodGet,
		fmt.Sprintf("/2/teams/%s/environments", url.PathEscape(teamSlug)),
		nil,
	)
	if err != nil {
		return "", err
	}

	body, code, err := c.do(req)
	if err != nil {
		return "", err
	}
	if code < 200 || code >= 300 {
		return "", fmt.Errorf("list environments failed (http %d): %s", code, string(body))
	}

	var parsed listEnvironmentsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("failed to parse environments: %w", err)
	}

	for _, e := range parsed.Data {
		if strings.EqualFold(strings.TrimSpace(e.Attributes.Slug), envSlug) {
			id := strings.TrimSpace(e.ID)
			if id != "" {
				return id, nil
			}
		}
	}

	return "", fmt.Errorf("environmentSlug %q not found in team %q", envSlug, teamSlug)
}

// EnsureConfigurationKey creates a configuration API key via the /2 API and stores
// its secret for use in /1 API requests. If a valid key already exists, it is reused.
func (c *Client) EnsureConfigurationKey(teamSlug string) error {
	teamSlug = strings.TrimSpace(teamSlug)
	if teamSlug == "" {
		return fmt.Errorf("teamSlug is required")
	}

	if c.hasSecret(secretNameConfigurationKey) {
		code, body, err := c.pingV1WithConfigKey()
		if err == nil && code >= 200 && code < 300 {
			return nil
		}

		if err != nil {
			return fmt.Errorf("configuration key v1 ping failed: %w", err)
		}
		if code == http.StatusUnauthorized || code == http.StatusForbidden {
			_ = c.integrationCtx.SetSecret(secretNameConfigurationKey, []byte{})
		} else {
			return fmt.Errorf("existing configuration key failed v1 ping (http %d): %s", code, string(body))
		}
	}

	envSlugAny, err := c.integrationCtx.GetConfig("environmentSlug")
	if err != nil || strings.TrimSpace(string(envSlugAny)) == "" {
		return fmt.Errorf("environmentSlug is required")
	}
	envSlug := strings.TrimSpace(string(envSlugAny))

	envID, err := c.getEnvironmentID(teamSlug, envSlug)
	if err != nil {
		return fmt.Errorf("failed to resolve environment ID for slug %q: %w", envSlug, err)
	}

	payload := map[string]any{
		"data": map[string]any{
			"type": "api-keys",
			"attributes": map[string]any{
				"key_type": "configuration",
				"name":     "SuperPlane Configuration Key",
				"disabled": false,
				"permissions": map[string]any{
					"manage_triggers":   true,
					"manage_recipients": true,
					"send_events":       false,
				},
			},
			"relationships": map[string]any{
				"environment": map[string]any{
					"data": map[string]any{
						"id":   envID,
						"type": "environments",
					},
				},
			},
		},
	}

	body, _ := json.Marshal(payload)

	req, err := c.newReqV2(
		http.MethodPost,
		fmt.Sprintf("/2/teams/%s/api-keys", url.PathEscape(teamSlug)),
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}

	respBody, code, err := c.do(req)
	if err != nil {
		return err
	}
	if code < 200 || code >= 300 {
		return fmt.Errorf("create configuration key failed (http %d): %s", code, string(respBody))
	}

	keySecret, err := parseCreatedEnvKeyValue(respBody)
	if err != nil {
		return err
	}

	if err := c.integrationCtx.SetSecret(secretNameConfigurationKey, []byte(keySecret)); err != nil {
		return fmt.Errorf("failed to store configuration key: %w", err)
	}

	code2, body2, err2 := c.pingV1WithConfigKey()
	if err2 != nil {
		return fmt.Errorf("v1 ping failed after creating config key: %w", err2)
	}
	if code2 < 200 || code2 >= 300 {
		return fmt.Errorf("created configuration key but v1 ping failed (http %d): %s", code2, string(body2))
	}

	return nil
}

// EnsureIngestKey creates an ingest API key via the /2 API and stores it for use
// when sending events. If a valid key already exists, it is reused.
func (c *Client) EnsureIngestKey(teamSlug string) error {
	if c.hasSecret(secretNameIngestKey) {
		code, body, err := c.pingV1WithIngestKey()
		if err == nil && code >= 200 && code < 300 {
			return nil
		}

		if err != nil {
			return fmt.Errorf("ingest key v1 ping failed: %w", err)
		}
		if code == http.StatusUnauthorized || code == http.StatusForbidden {
			_ = c.integrationCtx.SetSecret(secretNameIngestKey, []byte{})
		} else {
			return fmt.Errorf("existing ingest key failed v1 ping (http %d): %s", code, string(body))
		}
	}

	teamSlug = strings.TrimSpace(teamSlug)
	if teamSlug == "" {
		return fmt.Errorf("teamSlug is required")
	}

	envSlugAny, err := c.integrationCtx.GetConfig("environmentSlug")
	if err != nil || strings.TrimSpace(string(envSlugAny)) == "" {
		return fmt.Errorf("environmentSlug is required")
	}
	envSlug := strings.TrimSpace(string(envSlugAny))

	envID, err := c.getEnvironmentID(teamSlug, envSlug)
	if err != nil {
		return fmt.Errorf("failed to resolve environment ID for slug %q: %w", envSlug, err)
	}

	payload := map[string]any{
		"data": map[string]any{
			"type": "api-keys",
			"attributes": map[string]any{
				"key_type": "ingest",
				"name":     "SuperPlane Ingest Key",
				"disabled": false,
				"permissions": map[string]any{
					"create_datasets": true,
				},
			},
			"relationships": map[string]any{
				"environment": map[string]any{
					"data": map[string]any{
						"id":   envID,
						"type": "environments",
					},
				},
			},
		},
	}

	body, _ := json.Marshal(payload)

	req, err := c.newReqV2(
		http.MethodPost,
		fmt.Sprintf("/2/teams/%s/api-keys", url.PathEscape(teamSlug)),
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}

	respBody, code, err := c.do(req)
	if err != nil {
		return err
	}
	if code < 200 || code >= 300 {
		return fmt.Errorf("create ingest key failed (http %d): %s", code, string(respBody))
	}

	keyValue, err := parseCreatedIngestKeyValue(respBody)
	if err != nil {
		return err
	}

	if err := c.integrationCtx.SetSecret(secretNameIngestKey, []byte(keyValue)); err != nil {
		return fmt.Errorf("failed to store ingest key secret: %w", err)
	}

	code2, body2, err2 := c.pingV1WithIngestKey()
	if err2 != nil {
		return fmt.Errorf("v1 ping failed after creating ingest key: %w", err2)
	}
	if code2 < 200 || code2 >= 300 {
		return fmt.Errorf("created ingest key but v1 ping failed (http %d): %s", code2, string(body2))
	}

	return nil
}

type HoneycombTrigger struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Raw  map[string]any `json:"-"`
}

func (c *Client) ListTriggers(datasetSlug string) ([]HoneycombTrigger, error) {
	req, err := c.newReqV1(http.MethodGet, fmt.Sprintf("/1/triggers/%s", url.PathEscape(datasetSlug)), nil)
	if err != nil {
		return nil, err
	}
	respBody, code, err := c.do(req)
	if err != nil {
		return nil, err
	}
	if code < 200 || code >= 300 {
		return nil, fmt.Errorf("list triggers failed (http %d): %s", code, string(respBody))
	}

	var arr []map[string]any
	if err := json.Unmarshal(respBody, &arr); err != nil {
		return nil, fmt.Errorf("failed to parse triggers list: %w", err)
	}

	out := make([]HoneycombTrigger, 0, len(arr))
	for _, m := range arr {
		id, _ := m["id"].(string)
		name, _ := m["name"].(string)
		out = append(out, HoneycombTrigger{ID: id, Name: name, Raw: m})
	}
	return out, nil
}

func (c *Client) GetTrigger(datasetSlug, triggerID string) (map[string]any, error) {
	req, err := c.newReqV1(http.MethodGet, fmt.Sprintf("/1/triggers/%s/%s", url.PathEscape(datasetSlug), url.PathEscape(triggerID)), nil)
	if err != nil {
		return nil, err
	}
	respBody, code, err := c.do(req)
	if err != nil {
		return nil, err
	}
	if code < 200 || code >= 300 {
		return nil, fmt.Errorf("get trigger failed (http %d): %s", code, string(respBody))
	}

	var obj map[string]any
	if err := json.Unmarshal(respBody, &obj); err != nil {
		return nil, fmt.Errorf("failed to parse trigger: %w", err)
	}
	return obj, nil
}

// stripTriggerForUpdate removes read-only and conflicting fields from a trigger
// payload so it can be sent to the Honeycomb update API.
func stripTriggerForUpdate(trigger map[string]any) {
	if _, hasQueryID := trigger["query_id"]; hasQueryID {
		delete(trigger, "query")
	}
	delete(trigger, "id")
	delete(trigger, "dataset_slug")
	delete(trigger, "created_at")
	delete(trigger, "updated_at")
	delete(trigger, "triggered")
}

func (c *Client) UpdateTrigger(datasetSlug, triggerID string, trigger map[string]any) error {
	body, _ := json.Marshal(trigger)
	req, err := c.newReqV1(http.MethodPut, fmt.Sprintf("/1/triggers/%s/%s", url.PathEscape(datasetSlug), url.PathEscape(triggerID)), bytes.NewReader(body))
	if err != nil {
		return err
	}
	respBody, code, err := c.do(req)
	if err != nil {
		return err
	}
	if code < 200 || code >= 300 {
		return fmt.Errorf("update trigger failed (http %d): %s", code, string(respBody))
	}
	return nil
}

// EnsureRecipientOnTrigger attaches a webhook recipient to a Honeycomb trigger if not already attached.
func (c *Client) EnsureRecipientOnTrigger(datasetSlug, triggerID, recipientID string) error {
	trigger, err := c.GetTrigger(datasetSlug, triggerID)
	if err != nil {
		return err
	}

	recipientsAny, ok := trigger["recipients"]
	if !ok || recipientsAny == nil {
		recipientsAny = []any{}
	}
	recipientsSlice, _ := recipientsAny.([]any)

	for _, r := range recipientsSlice {
		if rm, ok := r.(map[string]any); ok {
			if id, _ := rm["id"].(string); strings.TrimSpace(id) == recipientID {
				return nil // already attached
			}
		}
	}

	recipientsSlice = append(recipientsSlice, map[string]any{
		"id":     recipientID,
		"type":   "webhook",
		"target": "SuperPlane",
	})
	trigger["recipients"] = recipientsSlice
	stripTriggerForUpdate(trigger)
	return c.UpdateTrigger(datasetSlug, triggerID, trigger)
}

type Recipient struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Target  string         `json:"target"`
	Details map[string]any `json:"details,omitempty"`
}

func (c *Client) CreateWebhookRecipient(webhookURL, secret string) (Recipient, error) {
	payload := map[string]any{
		"type": "webhook",
		"details": map[string]any{
			"webhook_name":   "SuperPlane",
			"webhook_url":    webhookURL,
			"webhook_secret": secret,
		},
	}

	body, _ := json.Marshal(payload)
	req, err := c.newReqV1(http.MethodPost, "/1/recipients", bytes.NewReader(body))
	if err != nil {
		return Recipient{}, err
	}

	respBody, code, err := c.do(req)
	if err != nil {
		return Recipient{}, err
	}

	if code == http.StatusConflict {
		return Recipient{}, fmt.Errorf("recipient already exists in Honeycomb but cannot be retrieved. Delete old SuperPlane recipients in Honeycomb UI under Team Settings > Recipients, then retry")
	}
	if code < 200 || code >= 300 {
		return Recipient{}, fmt.Errorf("create recipient failed (http %d): %s", code, string(respBody))
	}

	var obj map[string]any
	if err := json.Unmarshal(respBody, &obj); err != nil {
		return Recipient{}, fmt.Errorf("failed to parse recipient response: %w", err)
	}

	id, _ := obj["id"].(string)
	typ, _ := obj["type"].(string)
	details, _ := obj["details"].(map[string]any)

	return Recipient{ID: id, Type: typ, Target: webhookURL, Details: details}, nil
}

func (c *Client) DeleteRecipient(recipientID string, datasetSlug string) error {
	// First, remove the recipient from all associated triggers
	req, err := c.newReqV1(http.MethodGet, fmt.Sprintf("/1/recipients/%s/triggers", url.PathEscape(recipientID)), nil)
	if err != nil {
		return err
	}
	body, code, err := c.do(req)
	if err != nil {
		return err
	}
	if code == http.StatusOK {
		var triggers []map[string]any
		if json.Unmarshal(body, &triggers) == nil {
			for _, tr := range triggers {
				triggerID, _ := tr["id"].(string)
				if datasetSlug != "" && triggerID != "" {
					if err := c.RemoveRecipientFromTrigger(datasetSlug, triggerID, recipientID); err != nil {
						return err
					}
				}
			}
		}
	}

	req, err = c.newReqV1(http.MethodDelete, fmt.Sprintf("/1/recipients/%s", url.PathEscape(recipientID)), nil)
	if err != nil {
		return err
	}
	_, code, err = c.do(req)
	if err != nil {
		return err
	}
	if code == http.StatusNotFound {
		return nil
	}
	if code < 200 || code >= 300 {
		return fmt.Errorf("delete recipient failed (http %d)", code)
	}
	return nil
}

func (c *Client) RemoveRecipientFromTrigger(datasetSlug, triggerID, recipientID string) error {
	trigger, err := c.GetTrigger(datasetSlug, triggerID)
	if err != nil {
		return err
	}

	recipientsAny, _ := trigger["recipients"].([]any)
	filtered := make([]any, 0)
	for _, r := range recipientsAny {
		if rm, ok := r.(map[string]any); ok {
			if id, _ := rm["id"].(string); id != recipientID {
				filtered = append(filtered, rm)
			}
		}
	}
	trigger["recipients"] = filtered
	stripTriggerForUpdate(trigger)
	return c.UpdateTrigger(datasetSlug, triggerID, trigger)
}

func (c *Client) CreateEvent(datasetSlug string, fields map[string]any) error {
	datasetSlug = strings.TrimSpace(datasetSlug)
	if datasetSlug == "" {
		return fmt.Errorf("dataset is required")
	}

	ingestHeader, err := c.getSecretValue(secretNameIngestKey)
	if err != nil || strings.TrimSpace(ingestHeader) == "" {
		return fmt.Errorf("ingest key not found (expected secret %q)", secretNameIngestKey)
	}

	u, _ := url.Parse(c.BaseURL)
	u.Path = fmt.Sprintf("/1/events/%s", url.PathEscape(datasetSlug))

	body, err := json.Marshal(fields)
	if err != nil {
		return fmt.Errorf("failed to marshal fields: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Honeycomb-Team", ingestHeader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// If the event does not include a time field, set it automatically
	if _, hasTimeField := fields["time"]; !hasTimeField {
		req.Header.Set("X-Honeycomb-Event-Time", time.Now().UTC().Format(time.RFC3339Nano))
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	b, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("honeycomb create event failed (status %d): %s", resp.StatusCode, string(b))
}

func (c *Client) getSecretValue(name string) (string, error) {
	secrets, err := c.integrationCtx.GetSecrets()
	if err != nil {
		return "", err
	}
	for _, s := range secrets {
		if s.Name == name {
			v := strings.TrimSpace(string(s.Value))
			if v != "" {
				return v, nil
			}
		}
	}
	return "", fmt.Errorf("secret %q not found", name)
}

func (c *Client) hasSecret(name string) bool {
	secrets, err := c.integrationCtx.GetSecrets()
	if err != nil {
		return false
	}
	for _, s := range secrets {
		if s.Name == name && strings.TrimSpace(string(s.Value)) != "" {
			return true
		}
	}
	return false
}

func generateTokenHex(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func parseCreatedIngestKeyValue(respBody []byte) (string, error) {
	type createKeyResp struct {
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Secret string `json:"secret"`
			} `json:"attributes"`
		} `json:"data"`
	}

	var parsed createKeyResp
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("failed to parse create key response: %w", err)
	}

	id := strings.TrimSpace(parsed.Data.ID)
	secret := strings.TrimSpace(parsed.Data.Attributes.Secret)

	if id == "" {
		return "", fmt.Errorf("create key response missing data.id")
	}
	if secret == "" {
		return "", fmt.Errorf("create key response missing data.attributes.secret")
	}

	// Ingest key value is ID concatenated with secret
	return id + secret, nil
}

func parseCreatedEnvKeyValue(respBody []byte) (string, error) {
	type createKeyResp struct {
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Secret string `json:"secret"`
			} `json:"attributes"`
		} `json:"data"`
	}

	var parsed createKeyResp
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("failed to parse create key response: %w", err)
	}

	secret := strings.TrimSpace(parsed.Data.Attributes.Secret)
	if secret == "" {
		return "", fmt.Errorf("create key response missing data.attributes.secret: %s", string(respBody))
	}

	return secret, nil
}

type Dataset struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (c *Client) ListDatasets() ([]Dataset, error) {
	req, err := c.newReqV1(http.MethodGet, "/1/datasets", nil)
	if err != nil {
		return nil, err
	}

	body, code, err := c.do(req)
	if err != nil {
		return nil, err
	}
	if code < 200 || code >= 300 {
		return nil, fmt.Errorf("list datasets failed (http %d): %s", code, string(body))
	}

	var datasets []Dataset
	if err := json.Unmarshal(body, &datasets); err != nil {
		return nil, fmt.Errorf("failed to parse datasets: %w", err)
	}

	return datasets, nil
}
