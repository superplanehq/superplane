package config

import (
	"os"
	"strings"
)

const EnvGitHubEnterpriseAllowedHosts = "SUPERPLANE_GITHUB_ENTERPRISE_ALLOWED_HOSTS"

// GitHubEnterpriseAllowedHosts is the optional hostname allowlist for GitHub
// Enterprise. An empty list means any hostname that ParseHost accepts.
// github.com is always allowed.
func GitHubEnterpriseAllowedHosts() []string {
	raw := strings.TrimSpace(os.Getenv(EnvGitHubEnterpriseAllowedHosts))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		host := strings.ToLower(strings.TrimSpace(part))
		if host == "" {
			continue
		}
		out = append(out, host)
	}
	return out
}
