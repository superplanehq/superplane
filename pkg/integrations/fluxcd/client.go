package fluxcd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	Server    string
	Token     string
	Namespace string
	http      core.HTTPContext
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	server, err := ctx.GetConfig("server")
	if err != nil {
		return nil, fmt.Errorf("error getting server: %v", err)
	}

	token, err := ctx.GetConfig("token")
	if err != nil {
		return nil, fmt.Errorf("error getting token: %v", err)
	}

	namespace := "flux-system"
	ns, err := ctx.GetConfig("namespace")
	if err == nil && strings.TrimSpace(string(ns)) != "" {
		namespace = strings.TrimSpace(string(ns))
	}

	return &Client{
		Server:    strings.TrimRight(string(server), "/"),
		Token:     string(token),
		Namespace: namespace,
		http:      http,
	}, nil
}

func (c *Client) execRequest(method, url string, body io.Reader, contentType string) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	} else {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

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
		return nil, fmt.Errorf("request got %d: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// ValidateConnection verifies that the Kubernetes API server is reachable
// and the token is valid by checking server version.
func (c *Client) ValidateConnection() error {
	url := fmt.Sprintf("%s/version", c.Server)
	_, err := c.execRequest(http.MethodGet, url, nil, "")
	return err
}

var fluxResourceAPIs = map[string]string{
	"Kustomization":  "kustomize.toolkit.fluxcd.io/v1",
	"HelmRelease":    "helm.toolkit.fluxcd.io/v2",
	"GitRepository":  "source.toolkit.fluxcd.io/v1",
	"HelmRepository": "source.toolkit.fluxcd.io/v1",
	"OCIRepository":  "source.toolkit.fluxcd.io/v1beta2",
	"Bucket":         "source.toolkit.fluxcd.io/v1beta2",
}

var fluxResourcePlurals = map[string]string{
	"Kustomization":  "kustomizations",
	"HelmRelease":    "helmreleases",
	"GitRepository":  "gitrepositories",
	"HelmRepository": "helmrepositories",
	"OCIRepository":  "ocirepositories",
	"Bucket":         "buckets",
}

// ReconcileResource annotates a Flux resource to trigger reconciliation.
func (c *Client) ReconcileResource(kind, namespace, name string) (map[string]any, error) {
	apiVersion, ok := fluxResourceAPIs[kind]
	if !ok {
		return nil, fmt.Errorf("unsupported resource kind: %s", kind)
	}

	plural, ok := fluxResourcePlurals[kind]
	if !ok {
		return nil, fmt.Errorf("unsupported resource kind: %s", kind)
	}

	if namespace == "" {
		namespace = c.Namespace
	}

	url := fmt.Sprintf("%s/apis/%s/namespaces/%s/%s/%s",
		c.Server, apiVersion, namespace, plural, name)

	patch := map[string]any{
		"metadata": map[string]any{
			"annotations": map[string]string{
				"reconcile.fluxcd.io/requestedAt": time.Now().UTC().Format(time.RFC3339),
			},
		},
	}

	body, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("error marshaling patch: %v", err)
	}

	responseBody, err := c.execRequest(
		http.MethodPatch,
		url,
		bytes.NewReader(body),
		"application/merge-patch+json",
	)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return result, nil
}
