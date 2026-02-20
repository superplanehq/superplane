package harness

import (
	"fmt"
	"slices"
	"strings"
)

func validateHarnessScopeSelection(client *Client, orgID, projectID string) error {
	orgID = strings.TrimSpace(orgID)
	projectID = strings.TrimSpace(projectID)

	organizations, err := client.ListOrganizations()
	if err != nil {
		return fmt.Errorf("failed to validate organization: %w", err)
	}

	if !slices.ContainsFunc(organizations, func(org Organization) bool {
		return strings.TrimSpace(org.Identifier) == orgID
	}) {
		return fmt.Errorf("organization %q not found or inaccessible", orgID)
	}

	projects, err := client.ListProjects(orgID)
	if err != nil {
		return fmt.Errorf("failed to validate project in organization %q: %w", orgID, err)
	}

	if !slices.ContainsFunc(projects, func(project Project) bool {
		return strings.TrimSpace(project.Identifier) == projectID
	}) {
		return fmt.Errorf("project %q not found or inaccessible in organization %q", projectID, orgID)
	}

	return nil
}

func validateHarnessPipelineSelection(client *Client, orgID, projectID, pipelineIdentifier string) error {
	pipelineIdentifier = strings.TrimSpace(pipelineIdentifier)
	if pipelineIdentifier == "" {
		return nil
	}

	_, err := client.withScope(orgID, projectID).GetPipelineYAML(pipelineIdentifier)
	if err != nil {
		return fmt.Errorf(
			"pipeline %q not found or inaccessible in organization %q project %q: %s",
			pipelineIdentifier,
			strings.TrimSpace(orgID),
			strings.TrimSpace(projectID),
			summarizeVerificationError(err),
		)
	}

	return nil
}
