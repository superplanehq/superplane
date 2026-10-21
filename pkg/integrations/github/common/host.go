package common

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	HostKindDotCom HostKind = "dotcom"
	HostKindGHECom HostKind = "ghecom"
	HostKindGHES   HostKind = "ghes"

	// ConfigHost is the integration configuration key for a GitHub host.
	// Empty means github.com.
	ConfigHost = "host"
	// PropertyHost is the setup-wizard property for the same value.
	PropertyHost = "host"
	// PropertyGitHubLocation is the selectOwner choice between github.com and
	// GitHub Enterprise Cloud.
	PropertyGitHubLocation = "githubLocation"

	GitHubLocationDotCom          = "github.com"
	GitHubLocationEnterpriseCloud = "ghe.com"

	dotComHostname     = "github.com"
	dotComAPIHostname  = "api.github.com"
	dotComUploadHost   = "uploads.github.com"
	dotComCodeloadHost = "codeload.github.com"
	gheComSuffix       = ".ghe.com"
)

// HostKind is the GitHub deployment a Host talks to.
type HostKind string

// Host is a GitHub web and API deployment. An empty Hostname is github.com.
type Host struct {
	Kind     HostKind
	Hostname string
}

// DotCom is public GitHub.
func DotCom() Host {
	return Host{Kind: HostKindDotCom}
}

// ParseHost accepts a hostname or https URL. An empty value is github.com.
// It requires https when a scheme is present, and it rejects userinfo, path,
// query, and fragment.
func ParseHost(input string) (Host, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return DotCom(), nil
	}

	hostname, err := parseHostInput(input)
	if err != nil {
		return Host{}, err
	}

	return hostFromHostname(hostname)
}

func parseHostInput(input string) (string, error) {
	if !strings.Contains(input, "://") {
		if strings.ContainsAny(input, "/?#") || strings.Contains(input, "@") {
			return "", fmt.Errorf("GitHub host %q must be a hostname or https URL", input)
		}
		return normalizeHostname(input)
	}

	parsed, err := url.Parse(input)
	if err != nil {
		return "", fmt.Errorf("GitHub host %q is not a valid URL: %w", input, err)
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return "", fmt.Errorf("GitHub host %q must use https", input)
	}
	if parsed.User != nil {
		return "", fmt.Errorf("GitHub host %q must not include userinfo", input)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("GitHub host %q must not include a query or fragment", input)
	}
	path := strings.Trim(parsed.Path, "/")
	if path != "" {
		return "", fmt.Errorf("GitHub host %q must not include a path", input)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("GitHub host %q must include a hostname", input)
	}
	return normalizeHostname(parsed.Host)
}

func normalizeHostname(host string) (string, error) {
	host = strings.TrimSpace(strings.ToLower(host))
	host = strings.TrimSuffix(host, ".")
	if host == "" {
		return "", fmt.Errorf("GitHub host must include a hostname")
	}
	if strings.Contains(host, ":") {
		hostname, port, err := net.SplitHostPort(host)
		if err != nil {
			return "", fmt.Errorf("GitHub host %q is not a valid hostname: %w", host, err)
		}
		if port != "" && port != "443" {
			return "", fmt.Errorf("GitHub host %q must not include a non-https port", host)
		}
		host = hostname
	}
	if host == "www.github.com" {
		return dotComHostname, nil
	}
	return host, nil
}

func hostFromHostname(hostname string) (Host, error) {
	switch {
	case hostname == "" || hostname == dotComHostname:
		return DotCom(), nil
	case hostname == "ghe.com":
		return Host{}, fmt.Errorf("GitHub host %q is not a tenant hostname", hostname)
	case strings.HasSuffix(hostname, gheComSuffix):
		return Host{Kind: HostKindGHECom, Hostname: hostname}, nil
	default:
		return Host{Kind: HostKindGHES, Hostname: hostname}, nil
	}
}

// HostFromIntegration reads the host property, then configuration "host".
// A missing or empty value is github.com.
func HostFromIntegration(i core.IntegrationContext) Host {
	if i == nil {
		return DotCom()
	}
	if properties := i.Properties(); properties != nil {
		if raw, err := properties.GetString(PropertyHost); err == nil {
			if host, err := ParseHost(raw); err == nil {
				return host
			}
		}
	}
	if raw, err := i.GetConfig(ConfigHost); err == nil {
		if host, err := ParseHost(string(raw)); err == nil {
			return host
		}
	}
	return DotCom()
}

func HostFromProperties(properties core.IntegrationPropertyStorageReader) Host {
	if properties == nil {
		return DotCom()
	}
	raw, err := properties.GetString(PropertyHost)
	if err != nil {
		return DotCom()
	}
	host, err := ParseHost(raw)
	if err != nil {
		return DotCom()
	}
	return host
}

// HostFromModel reads the host property, then configuration "host".
func HostFromModel(i *models.Integration) Host {
	if i == nil {
		return DotCom()
	}
	for _, property := range i.Properties {
		if property.Name != PropertyHost {
			continue
		}
		raw, _ := property.Value.(string)
		host, err := ParseHost(raw)
		if err == nil {
			return host
		}
	}
	raw, _ := i.Configuration.Data()[ConfigHost].(string)
	host, err := ParseHost(raw)
	if err != nil {
		return DotCom()
	}
	return host
}

// HostFromFactoryOnboarding reads the GitHub host from the workspace VCS integration.
// Empty onboarding or a missing integration is github.com.
func HostFromFactoryOnboarding(tx *gorm.DB, orgID uuid.UUID, onboarding models.FactoryOnboardingConfig) (Host, error) {
	if strings.EqualFold(onboarding.EffectiveVCSProvider(), models.ProviderBitbucket) {
		return DotCom(), nil
	}
	rawID := strings.TrimSpace(onboarding.VCSIntegrationID)
	if rawID == "" || tx == nil {
		return DotCom(), nil
	}
	integrationID, err := uuid.Parse(rawID)
	if err != nil {
		return DotCom(), nil
	}
	integration, err := models.FindIntegrationInTransaction(tx, orgID, integrationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DotCom(), nil
		}
		return Host{}, err
	}
	return HostFromModel(integration), nil
}

func (h Host) IsDotCom() bool {
	return h.Kind == HostKindDotCom || h.WebHostname() == dotComHostname
}

func (h Host) IsGHECom() bool {
	return h.normalized().Kind == HostKindGHECom
}

func (h Host) IsGHES() bool {
	return h.normalized().Kind == HostKindGHES
}

func (h Host) normalized() Host {
	if h.Kind == "" && h.Hostname == "" {
		return DotCom()
	}
	host, err := hostFromHostname(strings.ToLower(strings.TrimSpace(h.Hostname)))
	if err != nil {
		return DotCom()
	}
	if h.Kind != "" {
		host.Kind = h.Kind
	}
	return host
}

// WebHostname is the browser host, such as github.com or acme.ghe.com.
func (h Host) WebHostname() string {
	host := h.normalized()
	if host.Hostname == "" {
		return dotComHostname
	}
	return host.Hostname
}

func (h Host) GitHost() string {
	return h.WebHostname()
}

func (h Host) WebURL() string {
	return "https://" + h.WebHostname()
}

// APIURL is the REST API base, including a trailing slash.
func (h Host) APIURL() string {
	host := h.normalized()
	switch host.Kind {
	case HostKindGHECom:
		return "https://api." + host.Hostname + "/"
	case HostKindGHES:
		return "https://" + host.Hostname + "/api/v3/"
	default:
		return "https://" + dotComAPIHostname + "/"
	}
}

func (h Host) GraphQLURL() string {
	host := h.normalized()
	switch host.Kind {
	case HostKindGHECom:
		return "https://api." + host.Hostname + "/graphql"
	case HostKindGHES:
		return "https://" + host.Hostname + "/api/graphql"
	default:
		return "https://" + dotComAPIHostname + "/graphql"
	}
}

func (h Host) UploadURL() string {
	host := h.normalized()
	switch host.Kind {
	case HostKindGHECom:
		return "https://uploads." + host.Hostname + "/"
	case HostKindGHES:
		return "https://" + host.Hostname + "/api/uploads/"
	default:
		return "https://" + dotComUploadHost + "/"
	}
}

func (h Host) RepoURL(fullName string) string {
	fullName = strings.Trim(strings.TrimSpace(fullName), "/")
	if fullName == "" {
		return h.WebURL()
	}
	return h.WebURL() + "/" + fullName
}

func (h Host) CloneURL(fullName string) string {
	fullName = strings.TrimSuffix(strings.Trim(strings.TrimSpace(fullName), "/"), ".git")
	if fullName == "" {
		return h.WebURL() + ".git"
	}
	return h.WebURL() + "/" + fullName + ".git"
}

func (h Host) ProfileURL(login string) string {
	login = strings.TrimSpace(login)
	if login == "" {
		return h.WebURL()
	}
	return h.WebURL() + "/" + login
}

func (h Host) TreeURL(fullName, branch string) string {
	fullName = strings.Trim(strings.TrimSpace(fullName), "/")
	branch = strings.TrimSpace(branch)
	if fullName == "" {
		return h.WebURL()
	}
	if branch == "" {
		return h.RepoURL(fullName)
	}
	return h.RepoURL(fullName) + "/tree/" + branch
}

func (h Host) CreateAppURL(ownerType, owner string) string {
	if ownerType == OwnerTypeOrganization {
		return h.WebURL() + "/organizations/" + strings.TrimSpace(owner) + "/settings/apps/new"
	}
	return h.WebURL() + "/settings/apps/new"
}

func (h Host) InstallAppURL(slug, state string) string {
	install := h.WebURL() + "/apps/" + strings.TrimSpace(slug) + "/installations/new"
	if strings.TrimSpace(state) == "" {
		return install
	}
	return install + "?state=" + url.QueryEscape(state)
}

func (h Host) ManifestConversionURL(code string) string {
	return h.APIURL() + "app-manifests/" + strings.TrimSpace(code) + "/conversions"
}

func (h Host) AppPermissionsURL(ownerType, owner, appSlug string) string {
	appSlug = strings.TrimSpace(appSlug)
	if ownerType == OwnerTypeOrganization {
		return h.WebURL() + "/organizations/" + strings.TrimSpace(owner) + "/settings/apps/" + appSlug + "/permissions"
	}
	return h.WebURL() + "/settings/apps/" + appSlug + "/permissions"
}

func (h Host) InstallationSettingsURL(ownerType, owner, installationID string) string {
	installationID = strings.TrimSpace(installationID)
	if ownerType == OwnerTypeOrganization {
		return h.WebURL() + "/organizations/" + strings.TrimSpace(owner) + "/settings/installations/" + installationID
	}
	return h.WebURL() + "/settings/installations/" + installationID
}

func (h Host) DependabotOriginURL(repository, query string) string {
	u := h.RepoURL(repository) + "/security/dependabot"
	if strings.TrimSpace(query) == "" {
		return u
	}
	return u + "?" + query
}

// IsWebHostname reports whether hostname is this tenant's browser host.
func (h Host) IsWebHostname(hostname string) bool {
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	if hostname == "www.github.com" && h.IsDotCom() {
		return true
	}
	return hostname == strings.ToLower(h.WebHostname())
}

// Matches reports whether hostname belongs to this tenant, including API,
// upload, and codeload hosts.
func (h Host) Matches(hostname string) bool {
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	if hostname == "" {
		return false
	}
	host := h.normalized()
	web := strings.ToLower(host.WebHostname())
	if hostname == web {
		return true
	}
	switch host.Kind {
	case HostKindGHECom:
		return hostname == "api."+web ||
			hostname == "uploads."+web ||
			hostname == "codeload."+web ||
			strings.HasSuffix(hostname, "."+web)
	case HostKindGHES:
		return false
	default:
		switch hostname {
		case "www.github.com", dotComAPIHostname, dotComUploadHost, dotComCodeloadHost:
			return true
		default:
			return strings.HasSuffix(hostname, ".githubusercontent.com")
		}
	}
}

// ParseRepoURL reads owner and repo from a web or clone URL on this host.
func (h Host) ParseRepoURL(raw string) (owner, repo string, ok bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return "", "", false
	}
	if !h.IsWebHostname(parsed.Hostname()) && !h.Matches(parsed.Hostname()) {
		return "", "", false
	}
	path := strings.Trim(parsed.Path, "/")
	path = strings.TrimSuffix(path, ".git")
	owner, repo, found := strings.Cut(path, "/")
	if !found || owner == "" || repo == "" {
		return "", "", false
	}
	if rest, _, extra := strings.Cut(repo, "/"); extra {
		repo = rest
	}
	return owner, repo, true
}

func (h Host) AllowedHosts() []string {
	host := h.normalized()
	web := host.WebHostname()
	switch host.Kind {
	case HostKindGHECom:
		return []string{web, "*." + web}
	case HostKindGHES:
		return []string{web}
	default:
		return []string{
			dotComHostname,
			dotComAPIHostname,
			dotComCodeloadHost,
			"*.githubusercontent.com",
			"objects.githubusercontent.com",
		}
	}
}

func (h Host) ServerURLEnv() string {
	return h.WebURL()
}

func (h Host) APIURLEnv() string {
	return strings.TrimSuffix(h.APIURL(), "/")
}

func (h Host) GraphQLURLEnv() string {
	return h.GraphQLURL()
}

func (h Host) SetupGitHostname() string {
	return h.GitHost()
}

func (h Host) SecretUsage() string {
	usage := `A GitHub token is available in the GITHUB_TOKEN environment variable.
The gh CLI is already installed on this runner. Use gh. Do not download or install gh.
gh reads GITHUB_TOKEN.`
	if !h.IsDotCom() {
		usage += `
GH_HOST and GH_ENTERPRISE_TOKEN are set for this GitHub Enterprise host.`
	}
	return usage + `
Use normal HTTPS repository URLs such as ` + h.CloneURL("<owner>/<repo>") + `.
Do not put the token in URLs, commands, Git configuration, or output.`
}

func (h Host) SetupGitScript() string {
	return `set -euo pipefail
: "${GITHUB_TOKEN:?GITHUB_TOKEN is required}"
gh auth setup-git --hostname ` + h.SetupGitHostname() + ` --force
`
}

// IsKnownGitHubHostname reports whether hostname is github.com, a github.com
// API host, or a GHE.com tenant. It does not look up organization integrations.
func IsKnownGitHubHostname(hostname string) bool {
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	if hostname == "" {
		return false
	}
	if DotCom().Matches(hostname) || DotCom().IsWebHostname(hostname) {
		return true
	}
	return strings.HasSuffix(hostname, gheComSuffix) && hostname != "ghe.com"
}

// IsKnownGitHubWebHostname reports whether hostname is a GitHub browser host.
func IsKnownGitHubWebHostname(hostname string) bool {
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	if hostname == "" {
		return false
	}
	if DotCom().IsWebHostname(hostname) {
		return true
	}
	return strings.HasSuffix(hostname, gheComSuffix) && hostname != "ghe.com"
}
