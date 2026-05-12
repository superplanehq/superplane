package gcp

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// workloadIdentityProviderPathRE matches the GCP IAM workload identity pool provider
// resource path (projects/…/locations/global/workloadIdentityPools/…/providers/…),
// including when embedded in https://iam.googleapis.com/v1/… URLs or console links.
var workloadIdentityProviderPathRE = regexp.MustCompile(`(?i)projects/([^/\s]+)/locations/global/workloadIdentityPools/([^/\s]+)/providers/([^/\s?&#:]+)`)

// NormalizeWorkloadIdentityProviderResourceName parses user input (canonical resource name,
// IAM REST URL, or any text containing the provider path) and returns the canonical form:
//
//	//iam.googleapis.com/projects/N/locations/global/workloadIdentityPools/POOL/providers/PROVIDER
func NormalizeWorkloadIdentityProviderResourceName(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	s = strings.Trim(s, "\u200b\uFEFF")
	s = strings.Trim(s, `"'`)
	if s == "" {
		return "", errors.New("pool provider resource name is required")
	}

	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.TrimSpace(s)

	if dec, err := url.PathUnescape(s); err == nil {
		s = dec
	}
	if dec, err := url.QueryUnescape(s); err == nil {
		s = dec
	}

	sub := workloadIdentityProviderPathRE.FindStringSubmatch(s)
	if len(sub) != 4 {
		return "", fmt.Errorf("could not parse workload identity provider; paste the provider resource name (//iam.googleapis.com/projects/…) or the https://iam.googleapis.com/v1/projects/… URL from Google Cloud Console")
	}

	canonical := fmt.Sprintf("//iam.googleapis.com/projects/%s/locations/global/workloadIdentityPools/%s/providers/%s", sub[1], sub[2], sub[3])
	if err := validateWorkloadIdentityProviderCanonical(canonical); err != nil {
		return "", err
	}
	return canonical, nil
}

func validateWorkloadIdentityProviderCanonical(s string) error {
	const prefix = "//iam.googleapis.com/projects/"
	if !strings.HasPrefix(s, prefix) {
		return fmt.Errorf("invalid workload identity provider resource name")
	}
	rest := strings.TrimPrefix(s, prefix)
	parts := strings.Split(rest, "/")
	if len(parts) != 7 {
		return fmt.Errorf("invalid workload identity provider resource name")
	}
	if parts[1] != "locations" || parts[2] != "global" || parts[3] != "workloadIdentityPools" || parts[5] != "providers" {
		return fmt.Errorf("invalid workload identity provider resource name")
	}
	for _, idx := range []int{0, 4, 6} {
		if parts[idx] == "" {
			return fmt.Errorf("invalid workload identity provider resource name")
		}
	}
	return nil
}
