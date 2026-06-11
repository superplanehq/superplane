package cloudsql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
)

// roleHintAdmin is the IAM role required to manage Cloud SQL instances.
const roleHintAdmin = "roles/cloudsql.admin (or roles/cloudsql.editor)"

// Instance models the subset of a Cloud SQL instance resource the components use.
type Instance struct {
	Name            string            `json:"name"`
	State           string            `json:"state"`
	DatabaseVersion string            `json:"databaseVersion"`
	Region          string            `json:"region"`
	ConnectionName  string            `json:"connectionName"`
	SelfLink        string            `json:"selfLink"`
	Settings        *InstanceSettings `json:"settings"`
	IPAddresses     []ipMapping       `json:"ipAddresses"`
}

type InstanceSettings struct {
	Tier           string `json:"tier"`
	DataDiskSizeGb string `json:"dataDiskSizeGb"`
	Edition        string `json:"edition"`
}

type ipMapping struct {
	Type      string `json:"type"`
	IPAddress string `json:"ipAddress"`
}

// operation models the long-running operation envelope returned by instance
// create/delete. These operations take minutes, so the components return the
// operation reference rather than waiting.
type operation struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	TargetID string `json:"targetId"`
}

// InstanceNodeMetadata is the node metadata shared by the instance components so
// the collapsed node can show what it targets.
type InstanceNodeMetadata struct {
	Instance string `json:"instance,omitempty" mapstructure:"instance"`
}

func instancesURL(project string) string {
	return fmt.Sprintf("%s/projects/%s/instances", sqlAdminBaseURL, project)
}

func instanceURL(project, instance string) string {
	return fmt.Sprintf("%s/projects/%s/instances/%s", sqlAdminBaseURL, project, instance)
}

// createInstance starts provisioning a Cloud SQL instance and returns the
// long-running operation. It does not wait: instance creation takes several
// minutes, so callers emit the operation and poll the instance (Get Instance)
// for readiness.
func createInstance(ctx context.Context, client Client, project string, body map[string]any) (*operation, error) {
	respBody, err := client.PostURL(ctx, instancesURL(project), body)
	if err != nil {
		return nil, err
	}
	return parseOperation(respBody)
}

// getInstance fetches a single instance.
func getInstance(ctx context.Context, client Client, project, name string) (*Instance, error) {
	respBody, err := client.GetURL(ctx, instanceURL(project, name))
	if err != nil {
		return nil, err
	}
	var inst Instance
	if err := json.Unmarshal(respBody, &inst); err != nil {
		return nil, fmt.Errorf("parse instance response: %w", err)
	}
	return &inst, nil
}

// deleteInstance starts deleting a Cloud SQL instance and returns the
// long-running operation (it does not wait).
func deleteInstance(ctx context.Context, client Client, project, name string) (*operation, error) {
	respBody, err := client.DeleteURL(ctx, instanceURL(project, name))
	if err != nil {
		return nil, err
	}
	return parseOperation(respBody)
}

// ListInstances lists the Cloud SQL instances in the project, following
// pagination so projects with more instances than one page are fully listed.
func ListInstances(ctx context.Context, client Client, project string) ([]Instance, error) {
	var all []Instance
	pageToken := ""
	for {
		u := instancesURL(project)
		if pageToken != "" {
			u += "?pageToken=" + url.QueryEscape(pageToken)
		}
		respBody, err := client.GetURL(ctx, u)
		if err != nil {
			return nil, err
		}
		var resp struct {
			Items         []Instance `json:"items"`
			NextPageToken string     `json:"nextPageToken"`
		}
		if err := json.Unmarshal(respBody, &resp); err != nil {
			return nil, fmt.Errorf("parse instances list: %w", err)
		}
		all = append(all, resp.Items...)
		if resp.NextPageToken == "" {
			return all, nil
		}
		pageToken = resp.NextPageToken
	}
}

func parseOperation(body []byte) (*operation, error) {
	var op operation
	if err := json.Unmarshal(body, &op); err != nil {
		return nil, fmt.Errorf("parse operation response: %w", err)
	}
	return &op, nil
}

// instancePayload converts an Instance into the component output payload.
func instancePayload(i *Instance) map[string]any {
	payload := map[string]any{
		"name":            i.Name,
		"state":           i.State,
		"databaseVersion": i.DatabaseVersion,
		"region":          i.Region,
		"connectionName":  i.ConnectionName,
		"selfLink":        i.SelfLink,
	}
	if i.Settings != nil {
		payload["tier"] = i.Settings.Tier
		payload["diskSizeGb"] = i.Settings.DataDiskSizeGb
		payload["edition"] = i.Settings.Edition
	}
	for _, ip := range i.IPAddresses {
		if ip.Type == "PRIMARY" || ip.Type == "" {
			payload["ipAddress"] = ip.IPAddress
			break
		}
	}
	return payload
}

// apiErrorMessage formats an API error for the execution state, appending an IAM
// hint on 403 since a missing Cloud SQL admin role is the most common cause.
func apiErrorMessage(action string, err error) string {
	var apiErr *gcpcommon.GCPAPIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden {
		return fmt.Sprintf("%s: %v — ensure the integration's service account has the %s IAM role", action, err, roleHintAdmin)
	}
	return fmt.Sprintf("%s: %v", action, err)
}
