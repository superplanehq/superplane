package cloudbuild

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCloudBuildRepositoryName(t *testing.T) {
	t.Run("parses valid repository name", func(t *testing.T) {
		projectID, location, connectionID, repositoryID := parseCloudBuildRepositoryName(
			"projects/my-project/locations/us-central1/connections/github-main/repositories/my-repo",
		)
		assert.Equal(t, "my-project", projectID)
		assert.Equal(t, "us-central1", location)
		assert.Equal(t, "github-main", connectionID)
		assert.Equal(t, "my-repo", repositoryID)
	})

	t.Run("trims whitespace before parsing", func(t *testing.T) {
		projectID, location, connectionID, repositoryID := parseCloudBuildRepositoryName(
			"  projects/p/locations/us-central1/connections/c/repositories/r  ",
		)
		assert.Equal(t, "p", projectID)
		assert.Equal(t, "us-central1", location)
		assert.Equal(t, "c", connectionID)
		assert.Equal(t, "r", repositoryID)
	})

	t.Run("returns empty strings for empty input", func(t *testing.T) {
		projectID, location, connectionID, repositoryID := parseCloudBuildRepositoryName("")
		assert.Empty(t, projectID)
		assert.Empty(t, location)
		assert.Empty(t, connectionID)
		assert.Empty(t, repositoryID)
	})

	t.Run("returns empty strings for plain build ID", func(t *testing.T) {
		projectID, location, connectionID, repositoryID := parseCloudBuildRepositoryName(
			"9f7d716f-a898-424e-8bda-ac2dc2bf8247",
		)
		assert.Empty(t, projectID)
		assert.Empty(t, location)
		assert.Empty(t, connectionID)
		assert.Empty(t, repositoryID)
	})

	t.Run("returns empty strings when segment names are wrong", func(t *testing.T) {
		projectID, location, connectionID, repositoryID := parseCloudBuildRepositoryName(
			"wrong/my-project/locations/us-central1/connections/c/repositories/r",
		)
		assert.Empty(t, projectID)
		assert.Empty(t, location)
		assert.Empty(t, connectionID)
		assert.Empty(t, repositoryID)
	})

	t.Run("returns empty strings when too few parts", func(t *testing.T) {
		projectID, location, connectionID, repositoryID := parseCloudBuildRepositoryName(
			"projects/p/locations/us-central1",
		)
		assert.Empty(t, projectID)
		assert.Empty(t, location)
		assert.Empty(t, connectionID)
		assert.Empty(t, repositoryID)
	})

	t.Run("returns empty strings when identifier segments are empty", func(t *testing.T) {
		projectID, location, connectionID, repositoryID := parseCloudBuildRepositoryName(
			"projects//locations/us-central1/connections/conn/repositories/repo",
		)
		assert.Empty(t, projectID)
		assert.Empty(t, location)
		assert.Empty(t, connectionID)
		assert.Empty(t, repositoryID)

		projectID, location, connectionID, repositoryID = parseCloudBuildRepositoryName(
			"projects/p/locations/us-central1/connections/conn/repositories/",
		)
		assert.Empty(t, projectID)
		assert.Empty(t, location)
		assert.Empty(t, connectionID)
		assert.Empty(t, repositoryID)
	})
}

func TestParseCloudBuildBuildName(t *testing.T) {
	t.Run("parses valid build name", func(t *testing.T) {
		projectID, location, buildID := parseCloudBuildBuildName(
			"projects/my-project/locations/us-central1/builds/build-123",
		)
		assert.Equal(t, "my-project", projectID)
		assert.Equal(t, "us-central1", location)
		assert.Equal(t, "build-123", buildID)
	})

	t.Run("parses global location", func(t *testing.T) {
		projectID, location, buildID := parseCloudBuildBuildName(
			"projects/demo-project/locations/global/builds/9f7d716f-a898-424e-8bda-ac2dc2bf8247",
		)
		assert.Equal(t, "demo-project", projectID)
		assert.Equal(t, "global", location)
		assert.Equal(t, "9f7d716f-a898-424e-8bda-ac2dc2bf8247", buildID)
	})

	t.Run("trims whitespace before parsing", func(t *testing.T) {
		projectID, location, buildID := parseCloudBuildBuildName(
			"  projects/p/locations/us-central1/builds/b  ",
		)
		assert.Equal(t, "p", projectID)
		assert.Equal(t, "us-central1", location)
		assert.Equal(t, "b", buildID)
	})

	t.Run("returns empty strings for plain UUID", func(t *testing.T) {
		projectID, location, buildID := parseCloudBuildBuildName(
			"9f7d716f-a898-424e-8bda-ac2dc2bf8247",
		)
		assert.Empty(t, projectID)
		assert.Empty(t, location)
		assert.Empty(t, buildID)
	})

	t.Run("returns empty strings for empty input", func(t *testing.T) {
		projectID, location, buildID := parseCloudBuildBuildName("")
		assert.Empty(t, projectID)
		assert.Empty(t, location)
		assert.Empty(t, buildID)
	})

	t.Run("returns empty strings when segment names are wrong", func(t *testing.T) {
		projectID, location, buildID := parseCloudBuildBuildName(
			"projects/my-project/locations/us-central1/wrong/build-123",
		)
		assert.Empty(t, projectID)
		assert.Empty(t, location)
		assert.Empty(t, buildID)
	})

	t.Run("returns empty strings when too few parts", func(t *testing.T) {
		projectID, location, buildID := parseCloudBuildBuildName(
			"projects/my-project/locations/us-central1",
		)
		assert.Empty(t, projectID)
		assert.Empty(t, location)
		assert.Empty(t, buildID)
	})

	t.Run("returns empty strings when identifier segments are empty", func(t *testing.T) {
		projectID, location, buildID := parseCloudBuildBuildName(
			"projects//locations/us-central1/builds/123",
		)
		assert.Empty(t, projectID)
		assert.Empty(t, location)
		assert.Empty(t, buildID)

		projectID, location, buildID = parseCloudBuildBuildName(
			"projects/p/locations/us-central1/builds/",
		)
		assert.Empty(t, projectID)
		assert.Empty(t, location)
		assert.Empty(t, buildID)
	})
}
