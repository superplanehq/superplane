package github

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
)

const maxOrganizationNameLength = 39

// A GitHub login accepts letters, numbers, and single hyphens between them.
var organizationNamePattern = regexp.MustCompile(`^[A-Za-z0-9]+(-[A-Za-z0-9]+)*$`)

/*
 * validateOrganizationName rejects input that GitHub cannot accept as an
 * organization login.
 *
 * The setup sends the user to github.com/organizations/<name>/settings/apps/new.
 * GitHub answers with a 404 page when the name is wrong, and that page does not
 * tell the user what to correct, so the check must happen before the user leaves
 * SuperPlane.
 */
func validateOrganizationName(organization string) error {
	if strings.Contains(organization, "/") || strings.Contains(organization, "github.com") {
		return errors.New("enter only the organization name, for example superplanehq, and not a URL or a repository path")
	}

	if len(organization) > maxOrganizationNameLength {
		return fmt.Errorf("the organization name must have %d characters or less", maxOrganizationNameLength)
	}

	if !organizationNamePattern.MatchString(organization) {
		return errors.New("the organization name can contain only letters, numbers, and single hyphens between them")
	}

	return nil
}

/*
 * checkOrganizationExists asks GitHub if the organization is real.
 *
 * Only a definitive "not found" answer stops the setup. Any other failure lets
 * the setup continue, because a rate limit or an outage on GitHub must not block
 * a configuration that is correct.
 */
func checkOrganizationExists(httpCtx core.HTTPContext, logger *logrus.Entry, organization string) error {
	if httpCtx == nil {
		return nil
	}

	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger())
	}

	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.github.com/orgs/%s", organization), nil)
	if err != nil {
		return nil
	}

	request.Header.Set("Accept", "application/vnd.github+json")

	response, err := httpCtx.Do(request)
	if err != nil {
		logger.Warnf("could not verify GitHub organization %s: %v", organization, err)
		return nil
	}

	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return fmt.Errorf("no GitHub organization named %s exists. Check the spelling. You must also be an owner of the organization, because SuperPlane creates the GitHub App in it", organization)
	}

	if response.StatusCode != http.StatusOK {
		logger.Warnf("could not verify GitHub organization %s: status %d", organization, response.StatusCode)
	}

	return nil
}
