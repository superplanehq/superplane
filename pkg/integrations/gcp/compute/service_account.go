package compute

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

// ResourceTypeServiceAccount is the integration-resource type backing the
// service-account dropdowns on the firewall rule components. It lists the
// project's IAM service accounts so users pick a real identity instead of
// typing an email (GCP silently accepts non-existent service accounts on a
// firewall rule, producing a rule that matches nothing).
const ResourceTypeServiceAccount = "serviceAccount"

// serviceAccountEmailSuffix is the common suffix of every Google service
// account email (e.g. ...@<project>.iam.gserviceaccount.com,
// ...@<project>.appspot.gserviceaccount.com, the default compute SA, etc.).
const serviceAccountEmailSuffix = "gserviceaccount.com"

// iamBaseURL is the IAM API host (distinct from the Compute host) used to list
// service accounts via the integration client's GetURL.
const iamBaseURL = "https://iam.googleapis.com/v1"

type iamServiceAccount struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Disabled    bool   `json:"disabled"`
}

type iamServiceAccountList struct {
	Accounts      []iamServiceAccount `json:"accounts"`
	NextPageToken string              `json:"nextPageToken"`
}

// ListServiceAccountResources lists the project's IAM service accounts for the
// firewall service-account dropdowns. Each resource's ID is the SA email, which
// is exactly what the firewalls API's source/target service-account fields
// expect. Requires the iam.serviceAccounts.list permission.
func ListServiceAccountResources(ctx context.Context, c Client, project string) ([]core.IntegrationResource, error) {
	if project == "" {
		project = c.ProjectID()
	}

	out := make([]core.IntegrationResource, 0)
	pageToken := ""
	// Bound the pagination loop; projects rarely exceed a few pages of SAs.
	for page := 0; page < 50; page++ {
		reqURL := fmt.Sprintf("%s/projects/%s/serviceAccounts?pageSize=100", iamBaseURL, url.PathEscape(project))
		if pageToken != "" {
			reqURL += "&pageToken=" + url.QueryEscape(pageToken)
		}
		body, err := c.GetURL(ctx, reqURL)
		if err != nil {
			return nil, fmt.Errorf("failed to list service accounts (the integration's service account needs the iam.serviceAccounts.list permission): %w", err)
		}
		var list iamServiceAccountList
		if err := json.Unmarshal(body, &list); err != nil {
			return nil, fmt.Errorf("failed to parse service accounts response: %w", err)
		}
		for _, sa := range list.Accounts {
			if strings.TrimSpace(sa.Email) == "" {
				continue
			}
			label := sa.Email
			if name := strings.TrimSpace(sa.DisplayName); name != "" {
				label = fmt.Sprintf("%s (%s)", name, sa.Email)
			}
			if sa.Disabled {
				label += " — disabled"
			}
			out = append(out, core.IntegrationResource{Type: ResourceTypeServiceAccount, Name: label, ID: sa.Email})
		}
		if list.NextPageToken == "" {
			break
		}
		pageToken = list.NextPageToken
	}
	return out, nil
}

// validateServiceAccountEmails rejects values that aren't service-account
// emails. GCP accepts arbitrary strings here and silently creates a rule that
// matches nothing, so this guard catches the common footgun (a user email or a
// typo) for free-text entries before the rule is created.
func validateServiceAccountEmails(emails []string) error {
	for _, e := range emails {
		v := strings.ToLower(strings.TrimSpace(e))
		if v == "" {
			continue
		}
		if !strings.HasSuffix(v, serviceAccountEmailSuffix) {
			return fmt.Errorf("%q is not a service account email (expected an address ending in %s)", e, serviceAccountEmailSuffix)
		}
	}
	return nil
}
