package gitlab

import (
	"context"
	"fmt"
)

const (
	ReleaseStrategySpecific = "specific"
	ReleaseStrategyLatest   = "latest"
)

// resolveReleaseTagName identifies which release to act on: the configured tag, or the project's latest release.
func resolveReleaseTagName(client *Client, projectID, releaseStrategy, tagName string) (string, error) {
	if releaseStrategy != ReleaseStrategyLatest {
		return tagName, nil
	}

	latest, err := client.GetLatestRelease(context.Background(), projectID)
	if err != nil {
		return "", fmt.Errorf("failed to find latest release: %w", err)
	}

	return latest.TagName, nil
}
