package cloudbuild

import (
	"context"
	"fmt"
	"sync"

	"github.com/superplanehq/superplane/pkg/core"
)

const cloudBuildBaseURL = "https://cloudbuild.googleapis.com/v1"

// Client is the interface used by Cloud Build components to call the API.
type Client interface {
	GetURL(ctx context.Context, fullURL string) ([]byte, error)
	PostURL(ctx context.Context, fullURL string, body any) ([]byte, error)
	ProjectID() string
}

var (
	clientFactoryMu sync.RWMutex
	clientFactory   func(httpCtx core.HTTPContext, integration core.IntegrationContext) (Client, error)
)

func SetClientFactory(fn func(httpCtx core.HTTPContext, integration core.IntegrationContext) (Client, error)) {
	clientFactoryMu.Lock()
	defer clientFactoryMu.Unlock()
	clientFactory = fn
}

func getClient(httpCtx core.HTTPContext, integration core.IntegrationContext) (Client, error) {
	clientFactoryMu.RLock()
	fn := clientFactory
	clientFactoryMu.RUnlock()
	if fn == nil {
		panic("gcp cloudbuild: SetClientFactory was not called by the gcp integration")
	}
	return fn(httpCtx, integration)
}

func buildGetURL(projectID string, buildID string, buildName string) string {
	nameProjectID, location, nameBuildID := parseCloudBuildBuildName(buildName)
	if location != "" && location != "global" {
		return fmt.Sprintf("%s/%s", cloudBuildBaseURL, buildName)
	}
	if nameProjectID != "" {
		projectID = nameProjectID
	}
	if nameBuildID != "" {
		buildID = nameBuildID
	}

	return fmt.Sprintf("%s/projects/%s/builds/%s", cloudBuildBaseURL, projectID, buildID)
}

func buildRunTriggerURL(projectID, triggerID string) string {
	return fmt.Sprintf("%s/projects/%s/triggers/%s:run", cloudBuildBaseURL, projectID, triggerID)
}

func buildGetTriggerURL(projectID, triggerID string) string {
	return fmt.Sprintf("%s/projects/%s/triggers/%s", cloudBuildBaseURL, projectID, triggerID)
}

func buildCancelURL(projectID string, buildID string, buildName string) string {
	nameProjectID, location, nameBuildID := parseCloudBuildBuildName(buildName)
	if location != "" && location != "global" {
		return fmt.Sprintf("%s/%s:cancel", cloudBuildBaseURL, buildName)
	}
	if nameProjectID != "" {
		projectID = nameProjectID
	}
	if nameBuildID != "" {
		buildID = nameBuildID
	}

	return fmt.Sprintf("%s/projects/%s/builds/%s:cancel", cloudBuildBaseURL, projectID, buildID)
}

func buildCreateTarget(
	projectOverride string,
	integrationProjectID string,
	build map[string]any,
) (string, string, error) {

	connectedRepository := connectedRepositoryNameFromBuild(build)
	if connectedRepository != "" {
		projectID, location, _, _ := parseCloudBuildRepositoryName(connectedRepository)
		if projectID == "" || location == "" {
			return "", "", fmt.Errorf("connectedRepository must be a valid Cloud Build repository resource name")
		}
		if projectOverride != "" && projectOverride != projectID {
			return "", "", fmt.Errorf("projectId override must match the connected repository project")
		}

		return projectID, fmt.Sprintf("%s/projects/%s/locations/%s/builds", cloudBuildBaseURL, projectID, location), nil
	}

	projectID := projectOverride
	if projectID == "" {
		projectID = integrationProjectID
	}
	if projectID == "" {
		return "", "", fmt.Errorf("projectId is required")
	}

	return projectID, fmt.Sprintf("%s/projects/%s/builds", cloudBuildBaseURL, projectID), nil
}

func connectedRepositoryNameFromBuild(build map[string]any) string {
	source, ok := build["source"].(map[string]any)
	if !ok {
		return ""
	}

	connectedRepository, ok := source["connectedRepository"].(map[string]any)
	if !ok {
		return ""
	}

	repository, _ := connectedRepository["repository"].(string)
	return repository
}
