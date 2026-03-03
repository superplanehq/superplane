package cloudbuild

import "strings"

func parseCloudBuildRepositoryName(name string) (projectID string, location string, connectionID string, repositoryID string) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 8 {
		return "", "", "", ""
	}

	if parts[0] != "projects" || parts[2] != "locations" || parts[4] != "connections" || parts[6] != "repositories" {
		return "", "", "", ""
	}
	if parts[1] == "" || parts[3] == "" || parts[5] == "" || parts[7] == "" {
		return "", "", "", ""
	}

	return parts[1], parts[3], parts[5], parts[7]
}

func parseCloudBuildBuildName(name string) (projectID string, location string, buildID string) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 6 {
		return "", "", ""
	}

	if parts[0] != "projects" || parts[2] != "locations" || parts[4] != "builds" {
		return "", "", ""
	}
	if parts[1] == "" || parts[3] == "" || parts[5] == "" {
		return "", "", ""
	}

	return parts[1], parts[3], parts[5]
}
