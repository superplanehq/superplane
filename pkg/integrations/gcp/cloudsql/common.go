package cloudsql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
)

// Cloud SQL instance lifecycle states the poll loop branches on.
const (
	instanceStateRunnable = "RUNNABLE"
	instanceStateFailed   = "FAILED"
)

// Instance create/delete are long-running (minutes), so the components poll via
// scheduled internal "poll" hooks instead of blocking a single execution.
const (
	pollHookName            = "poll"
	instancePollInterval    = 15 * time.Second
	instanceMaxPollAttempts = 80 // ~20 minutes at the 15s interval
	maxPollErrors           = 10 // consecutive fetch errors before giving up
)

// Database operations are short-lived, so they wait inline for the operation.
const (
	operationPollInterval = 2 * time.Second
	operationWaitTimeout  = 5 * time.Minute
	operationStatusDone   = "DONE"
)

// instanceExecMetadata is the per-execution state the poll hook reads to track a
// long-running instance operation across scheduled invocations.
type instanceExecMetadata struct {
	Instance     string `json:"instance" mapstructure:"instance"`
	PollAttempts int    `json:"pollAttempts" mapstructure:"pollAttempts"`
	PollErrors   int    `json:"pollErrors" mapstructure:"pollErrors"`
}

// roleHintAdmin is the IAM role required to manage (create/delete) Cloud SQL
// instances and databases; roleHintViewer is the read-only role sufficient for
// the get components.
const (
	roleHintAdmin  = "roles/cloudsql.admin (or roles/cloudsql.editor)"
	roleHintViewer = "roles/cloudsql.viewer (or roles/cloudsql.admin)"
)

// Database models a Cloud SQL database resource.
type Database struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Instance  string `json:"instance"`
	Project   string `json:"project"`
	SelfLink  string `json:"selfLink"`
	Charset   string `json:"charset"`
	Collation string `json:"collation"`
	Etag      string `json:"etag"`
}

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

// Tier models a Cloud SQL machine tier from the tiers.list endpoint. RAM and
// DiskQuota are returned as int64-formatted strings; Region lists the regions
// where the tier is offered. Note the API does not enumerate custom machine
// types (db-custom-*) here.
type Tier struct {
	Tier      string   `json:"tier"`
	RAM       int64    `json:"RAM,string"`
	DiskQuota int64    `json:"DiskQuota,string"`
	Region    []string `json:"region"`
}

// DatabaseNodeMetadata is the node metadata shared by the create/get/delete
// database components so the collapsed node can show what it targets.
type DatabaseNodeMetadata struct {
	Instance string `json:"instance,omitempty" mapstructure:"instance"`
	Database string `json:"database,omitempty" mapstructure:"database"`
}

// InstanceNodeMetadata is the node metadata shared by the instance components so
// the collapsed node can show what it targets.
type InstanceNodeMetadata struct {
	Instance string `json:"instance,omitempty" mapstructure:"instance"`
}

// operation models the long-running operation envelope returned by the Cloud SQL
// Admin API. Database operations wait on Status/Error; instance operations are
// long-running and return the operation reference (TargetID) without waiting.
type operation struct {
	Name          string          `json:"name"`
	Status        string          `json:"status"`
	OperationType string          `json:"operationType"`
	TargetID      string          `json:"targetId"`
	Error         *operationError `json:"error"`
}

type operationError struct {
	Errors []struct {
		Kind    string `json:"kind"`
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

func databasesURL(project, instance string) string {
	return fmt.Sprintf("%s/projects/%s/instances/%s/databases", sqlAdminBaseURL, project, instance)
}

func databaseURL(project, instance, database string) string {
	return fmt.Sprintf("%s/projects/%s/instances/%s/databases/%s", sqlAdminBaseURL, project, instance, database)
}

func instancesURL(project string) string {
	return fmt.Sprintf("%s/projects/%s/instances", sqlAdminBaseURL, project)
}

func instanceURL(project, instance string) string {
	return fmt.Sprintf("%s/projects/%s/instances/%s", sqlAdminBaseURL, project, instance)
}

func tiersURL(project string) string {
	return fmt.Sprintf("%s/projects/%s/tiers", sqlAdminBaseURL, project)
}

func operationURL(project, operationName string) string {
	return fmt.Sprintf("%s/projects/%s/operations/%s", sqlAdminBaseURL, project, operationName)
}

// createDatabase creates a logical database in the instance, waits for the
// returned operation to finish, and returns the created database.
func createDatabase(ctx context.Context, client Client, project, instance, name string) (*Database, error) {
	body := map[string]any{"name": name, "project": project, "instance": instance}
	respBody, err := client.PostURL(ctx, databasesURL(project, instance), body)
	if err != nil {
		return nil, err
	}
	if err := waitForOperation(ctx, client, project, respBody); err != nil {
		return nil, err
	}
	return getDatabase(ctx, client, project, instance, name)
}

// getDatabase fetches a single logical database.
func getDatabase(ctx context.Context, client Client, project, instance, name string) (*Database, error) {
	respBody, err := client.GetURL(ctx, databaseURL(project, instance, name))
	if err != nil {
		return nil, err
	}
	var db Database
	if err := json.Unmarshal(respBody, &db); err != nil {
		return nil, fmt.Errorf("parse database response: %w", err)
	}
	return &db, nil
}

// deleteDatabase deletes a logical database and waits for the operation to finish.
func deleteDatabase(ctx context.Context, client Client, project, instance, name string) error {
	respBody, err := client.DeleteURL(ctx, databaseURL(project, instance, name))
	if err != nil {
		return err
	}
	return waitForOperation(ctx, client, project, respBody)
}

// ListDatabases lists the databases in an instance.
func ListDatabases(ctx context.Context, client Client, project, instance string) ([]Database, error) {
	respBody, err := client.GetURL(ctx, databasesURL(project, instance))
	if err != nil {
		return nil, err
	}
	var resp struct {
		Items []Database `json:"items"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("parse databases list: %w", err)
	}
	return resp.Items, nil
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

// ListTiers lists the predefined machine tiers available to the project. The
// tiers.list endpoint is not paginated and does not include custom machine
// types (db-custom-*).
func ListTiers(ctx context.Context, client Client, project string) ([]Tier, error) {
	respBody, err := client.GetURL(ctx, tiersURL(project))
	if err != nil {
		return nil, err
	}
	var resp struct {
		Items []Tier `json:"items"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("parse tiers list: %w", err)
	}
	return resp.Items, nil
}

func parseOperation(body []byte) (*operation, error) {
	var op operation
	if err := json.Unmarshal(body, &op); err != nil {
		return nil, fmt.Errorf("parse operation response: %w", err)
	}
	return &op, nil
}

// databasePayload converts a Database into the component output payload.
func databasePayload(db *Database) map[string]any {
	return map[string]any{
		"name":      db.Name,
		"instance":  db.Instance,
		"project":   db.Project,
		"charset":   db.Charset,
		"collation": db.Collation,
		"selfLink":  db.SelfLink,
	}
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

// waitForOperation polls the operation referenced by the API response until it
// reaches DONE (or the timeout elapses). Cloud SQL database create/delete are
// asynchronous, so a caller that didn't wait could observe a not-yet-applied
// state.
func waitForOperation(ctx context.Context, client Client, project string, opBody []byte) error {
	var op operation
	if err := json.Unmarshal(opBody, &op); err != nil {
		return fmt.Errorf("parse operation response: %w", err)
	}
	// Some responses are not long-running operations; nothing to wait on.
	if op.Name == "" {
		return nil
	}

	deadline := time.Now().Add(operationWaitTimeout)
	ticker := time.NewTicker(operationPollInterval)
	defer ticker.Stop()
	for {
		if op.Status == operationStatusDone {
			return operationResultError(op)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for Cloud SQL operation %s", op.Name)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		body, err := client.GetURL(ctx, operationURL(project, op.Name))
		if err != nil {
			return err
		}
		if err := json.Unmarshal(body, &op); err != nil {
			return fmt.Errorf("parse operation poll response: %w", err)
		}
	}
}

func operationResultError(op operation) error {
	if op.Error == nil || len(op.Error.Errors) == 0 {
		return nil
	}
	e := op.Error.Errors[0]
	msg := e.Message
	if msg == "" {
		msg = e.Code
	}
	return fmt.Errorf("Cloud SQL operation failed: %s", msg)
}

// apiErrorMessage formats an API error for the execution state, appending the
// IAM role the component needs on a 403 (a missing role is the most common
// cause). Callers pass the role appropriate to the operation (read vs. write).
func apiErrorMessage(action string, err error, roleHint string) string {
	var apiErr *gcpcommon.GCPAPIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden {
		return fmt.Sprintf("%s: %v — ensure the integration's service account has the %s IAM role", action, err, roleHint)
	}
	return fmt.Sprintf("%s: %v", action, err)
}
