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

// roleHintAdmin is the IAM role required to manage Cloud SQL databases.
const roleHintAdmin = "roles/cloudsql.admin (or roles/cloudsql.editor)"

const (
	operationPollInterval = 2 * time.Second
	operationWaitTimeout  = 5 * time.Minute
	operationStatusDone   = "DONE"
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

// Instance is the subset of a Cloud SQL instance used to populate the dropdown.
type Instance struct {
	Name            string `json:"name"`
	ConnectionName  string `json:"connectionName"`
	Region          string `json:"region"`
	DatabaseVersion string `json:"databaseVersion"`
	State           string `json:"state"`
}

// DatabaseNodeMetadata is the node metadata shared by the create/get/delete
// database components so the collapsed node can show what it targets.
type DatabaseNodeMetadata struct {
	Instance string `json:"instance,omitempty" mapstructure:"instance"`
	Database string `json:"database,omitempty" mapstructure:"database"`
}

type operation struct {
	Name          string          `json:"name"`
	Status        string          `json:"status"`
	OperationType string          `json:"operationType"`
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

func operationURL(project, operationName string) string {
	return fmt.Sprintf("%s/projects/%s/operations/%s", sqlAdminBaseURL, project, operationName)
}

// createDatabase creates a logical database in the instance, waits for the
// returned operation to finish, and returns the created database. Charset and
// collation are optional; when blank the database engine's defaults apply.
func createDatabase(ctx context.Context, client Client, project, instance, name, charset, collation string) (*Database, error) {
	body := map[string]any{"name": name, "project": project, "instance": instance}
	if charset != "" {
		body["charset"] = charset
	}
	if collation != "" {
		body["collation"] = collation
	}
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

// apiErrorMessage formats an API error for the execution state, appending an IAM
// hint on 403 since a missing Cloud SQL admin role is the most common cause.
func apiErrorMessage(action string, err error) string {
	var apiErr *gcpcommon.GCPAPIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden {
		return fmt.Sprintf("%s: %v — ensure the integration's service account has the %s IAM role", action, err, roleHintAdmin)
	}
	return fmt.Sprintf("%s: %v", action, err)
}
