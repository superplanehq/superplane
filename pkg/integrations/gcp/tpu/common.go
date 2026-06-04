package tpu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	operationWaitTimeout  = 15 * time.Minute
	operationPollInterval = 5 * time.Second
)

// LabelEntry is a single key/value pair from a labels list field.
type LabelEntry struct {
	Key   string `json:"key" mapstructure:"key"`
	Value string `json:"value" mapstructure:"value"`
}

// TPUNodeMetadata is persisted on the node so the collapsed UI can show the
// targeted TPU node name. Create/Get/Delete share it.
type TPUNodeMetadata struct {
	NodeName string `json:"nodeName" mapstructure:"nodeName"`
}

// Node is the Cloud TPU v2 node resource sent on create.
type Node struct {
	AcceleratorType  string            `json:"acceleratorType,omitempty"`
	RuntimeVersion   string            `json:"runtimeVersion,omitempty"`
	Description      string            `json:"description,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
	NetworkConfig    *NetworkConfig    `json:"networkConfig,omitempty"`
	SchedulingConfig *SchedulingConfig `json:"schedulingConfig,omitempty"`
}

type NetworkConfig struct {
	Network           string `json:"network,omitempty"`
	Subnetwork        string `json:"subnetwork,omitempty"`
	EnableExternalIps bool   `json:"enableExternalIps,omitempty"`
}

type SchedulingConfig struct {
	Preemptible bool `json:"preemptible,omitempty"`
}

// nodeGetResp parses a Cloud TPU node from a get response or operation result.
type nodeGetResp struct {
	Name             string            `json:"name"`
	AcceleratorType  string            `json:"acceleratorType"`
	RuntimeVersion   string            `json:"runtimeVersion"`
	State            string            `json:"state"`
	Health           string            `json:"health"`
	Description      string            `json:"description"`
	CreateTime       string            `json:"createTime"`
	Labels           map[string]string `json:"labels"`
	NetworkEndpoints []struct {
		IPAddress string `json:"ipAddress"`
		Port      int    `json:"port"`
	} `json:"networkEndpoints"`
}

// operationResponse is the standard google.longrunning.Operation envelope the
// Cloud TPU v2 API returns from create and delete calls.
type operationResponse struct {
	Name  string `json:"name"`
	Done  bool   `json:"done"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func nodesBaseURL(project, location string) string {
	return fmt.Sprintf("%s/projects/%s/locations/%s/nodes", tpuBaseURL, project, location)
}

func nodeURL(project, location, nodeID string) string {
	return fmt.Sprintf("%s/%s", nodesBaseURL(project, location), nodeID)
}

func createNode(ctx context.Context, client Client, project, location, nodeID string, node *Node) ([]byte, error) {
	reqURL := fmt.Sprintf("%s?nodeId=%s", nodesBaseURL(project, location), url.QueryEscape(nodeID))
	return client.PostURL(ctx, reqURL, node)
}

func getNode(ctx context.Context, client Client, project, location, nodeID string) ([]byte, error) {
	return client.GetURL(ctx, nodeURL(project, location, nodeID))
}

func deleteNode(ctx context.Context, client Client, project, location, nodeID string) ([]byte, error) {
	return client.DeleteURL(ctx, nodeURL(project, location, nodeID))
}

// waitForOperation polls a long-running operation until it is done, returning
// the final operation body. The operation name is the full resource name
// (projects/.../locations/.../operations/...) returned by the mutating call.
func waitForOperation(ctx context.Context, client Client, operationName string) ([]byte, error) {
	reqURL := fmt.Sprintf("%s/%s", tpuBaseURL, strings.TrimPrefix(operationName, "/"))
	deadline := time.Now().Add(operationWaitTimeout)
	ticker := time.NewTicker(operationPollInterval)
	defer ticker.Stop()
	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for operation %s", operationName)
		}
		body, err := client.GetURL(ctx, reqURL)
		if err != nil {
			return nil, err
		}
		var op operationResponse
		if err := json.Unmarshal(body, &op); err != nil {
			return nil, fmt.Errorf("parse operation response: %w", err)
		}
		if op.Done {
			if op.Error != nil {
				msg := op.Error.Message
				if msg == "" {
					msg = fmt.Sprintf("code %d", op.Error.Code)
				}
				return nil, fmt.Errorf("operation failed: %s", msg)
			}
			return body, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

// operationNameFromBody extracts the operation name from a mutating call's
// response, returning a descriptive error when it is absent.
func operationNameFromBody(body []byte, what string) (string, error) {
	var op operationResponse
	if err := json.Unmarshal(body, &op); err != nil {
		return "", fmt.Errorf("parse %s operation response: %w", what, err)
	}
	if op.Name == "" {
		return "", fmt.Errorf("%s operation response missing operation name", what)
	}
	return op.Name, nil
}

// nodePayloadFromResponse converts a node get response into the flat payload
// emitted by the Create and Get components.
func nodePayloadFromResponse(body []byte) (map[string]any, error) {
	var n nodeGetResp
	if err := json.Unmarshal(body, &n); err != nil {
		return nil, fmt.Errorf("parse node response: %w", err)
	}
	payload := map[string]any{
		"name":            lastSegment(n.Name),
		"resourceName":    n.Name,
		"acceleratorType": n.AcceleratorType,
		"runtimeVersion":  n.RuntimeVersion,
		"state":           n.State,
	}
	if _, location, _, err := parseNodeName(n.Name); err == nil && location != "" {
		payload["location"] = location
	}
	if n.Health != "" {
		payload["health"] = n.Health
	}
	if n.Description != "" {
		payload["description"] = n.Description
	}
	if n.CreateTime != "" {
		payload["createTime"] = n.CreateTime
	}
	if len(n.Labels) > 0 {
		payload["labels"] = n.Labels
	}
	if len(n.NetworkEndpoints) > 0 {
		ips := make([]string, 0, len(n.NetworkEndpoints))
		for _, e := range n.NetworkEndpoints {
			if e.IPAddress != "" {
				ips = append(ips, e.IPAddress)
			}
		}
		if len(ips) > 0 {
			payload["ipAddresses"] = ips
		}
	}
	return payload, nil
}

// parseNodeName extracts (project, location, node) from a TPU node resource name
// of the form projects/{project}/locations/{location}/nodes/{node}.
func parseNodeName(name string) (project, location, node string, err error) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	for i := 0; i+1 < len(parts); i += 2 {
		switch parts[i] {
		case "projects":
			project = parts[i+1]
		case "locations":
			location = parts[i+1]
		case "nodes":
			node = parts[i+1]
		}
	}
	if node == "" {
		return "", "", "", fmt.Errorf("could not parse TPU node name from %q", name)
	}
	return project, location, node, nil
}

// resolveNodeSelection parses the selected TPU node value into its location and
// node ID. The node picker returns the node's full resource name
// (projects/{project}/locations/{location}/nodes/{node}), so the location is
// derived from the selection itself. The project is validated against the
// integration's bound project.
func resolveNodeSelection(nodeValue, boundProject string) (location, nodeID string, err error) {
	v := strings.TrimSpace(nodeValue)
	if v == "" {
		return "", "", errors.New("node is required")
	}
	proj, loc, n, perr := parseNodeName(v)
	if perr != nil {
		return "", "", fmt.Errorf("select a TPU node from the list: %w", perr)
	}
	if proj != "" && boundProject != "" && proj != boundProject {
		return "", "", fmt.Errorf(
			"node belongs to project %q but this GCP integration is bound to project %q; cross-project operations are not supported",
			proj, boundProject,
		)
	}
	if loc == "" {
		return "", "", fmt.Errorf("could not determine the location of node %q", v)
	}
	return loc, n, nil
}

// labelsFromEntries converts the labels list field into the GCP labels map,
// dropping entries with empty keys and de-duplicating on first write.
func labelsFromEntries(entries []LabelEntry) map[string]string {
	if len(entries) == 0 {
		return nil
	}
	out := make(map[string]string)
	for _, e := range entries {
		k := strings.TrimSpace(e.Key)
		if k == "" {
			continue
		}
		if _, exists := out[k]; exists {
			continue
		}
		out[k] = strings.TrimSpace(e.Value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func lastSegment(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return s[i+1:]
	}
	return s
}
