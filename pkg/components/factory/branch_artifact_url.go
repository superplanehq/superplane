package factory

import (
	"fmt"
	"net/url"
	"strings"
)

// branchTreeURL builds a browseable tree URL from a repository reference
// and a branch name. `repository` is either `owner/repo` (GitHub.com) or
// a repository http(s) URL (GitHub.com or GitHub Enterprise).
//
// GitHub's create-ref API does not return a browse URL. After the
// branch exists we already have owner/repo + name, so we persist this
// at attach time instead of waiting for a pull request.
func branchTreeURL(repository, name string) string {
	repo := strings.TrimRight(strings.TrimSpace(repository), "/")
	branch := strings.TrimSpace(name)
	if repo == "" || branch == "" {
		return ""
	}

	if parsed := parseHTTPRepositoryURL(repo); parsed != nil {
		// Assign the raw branch name to Path. String() encodes reserved
		// characters. Pre-escaping here would double-encode `#` as `%2523`.
		parsed.Path = parsed.Path + "/tree/" + branch
		return parsed.String()
	}

	if strings.Contains(repo, "://") || strings.HasPrefix(repo, "/") {
		return ""
	}

	owner, rest, ok := strings.Cut(repo, "/")
	if !ok || owner == "" || rest == "" || strings.Contains(rest, "/") {
		return ""
	}

	return "https://github.com/" + owner + "/" + rest + "/tree/" + encodeBranchPath(branch)
}

func encodeBranchPath(name string) string {
	parts := strings.Split(name, "/")
	encoded := make([]string, len(parts))
	for i, part := range parts {
		encoded[i] = url.PathEscape(part)
	}
	return strings.Join(encoded, "/")
}

func applyBranchTreeURL(config AddWorkOrderArtifactConfiguration, data map[string]any) map[string]any {
	if config.ArtifactType != "branch" {
		return data
	}

	repository := config.Repository
	if repository == "" {
		repository = artifactString(data, "repository")
	}
	data = storeSanitizedRepositoryField(data, "repository", repository)
	data = storeSanitizedRepositoryField(data, "repo", artifactString(data, "repo"))

	if artifactString(data, "url") != "" || artifactString(data, "html_url") != "" {
		return data
	}

	repo := artifactString(data, "repository")
	if repo == "" {
		repo = artifactString(data, "repo")
	}

	name := config.Name
	if name == "" {
		name = artifactString(data, "name")
	}

	tree := branchTreeURL(repo, name)
	if tree == "" {
		return data
	}

	data = ensureArtifactData(data)
	data["url"] = tree
	return data
}

func branchArtifactHasURLOrRepository(config AddWorkOrderArtifactConfiguration) bool {
	if strings.TrimSpace(config.URL) != "" || strings.TrimSpace(config.Repository) != "" {
		return true
	}
	for _, entry := range config.Data {
		switch strings.TrimSpace(entry.Name) {
		case "url", "html_url", "repository", "repo":
			if strings.TrimSpace(entry.Value) != "" {
				return true
			}
		}
	}
	return false
}

func validateBranchArtifactConfiguration(config AddWorkOrderArtifactConfiguration) error {
	if config.ArtifactType != "branch" {
		return nil
	}
	if branchArtifactHasURLOrRepository(config) {
		return nil
	}
	return fmt.Errorf("branch artifact requires a url or a repository that can produce a tree URL")
}

// requireReachableBranchURL rejects a branch attach that still has no
// browse URL after applyBranchTreeURL. A remote branch is reachable as
// soon as it is pushed; SuperPlane does not wait for a pull request.
func requireReachableBranchURL(artifactType string, data map[string]any) error {
	if artifactType != "branch" {
		return nil
	}
	if artifactString(data, "url") != "" || artifactString(data, "html_url") != "" {
		return nil
	}
	return fmt.Errorf("branch artifact requires a url or a repository that can produce a tree URL")
}

// parseHTTPRepositoryURL returns a credential-free repository URL.
// Query and fragment are dropped so later path joins stay on the path.
func hasHTTPScheme(repository string) bool {
	lower := strings.ToLower(repository)
	return strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://")
}

func parseHTTPRepositoryURL(repository string) *url.URL {
	if !hasHTTPScheme(repository) {
		return nil
	}

	parsed, err := url.Parse(repository)
	if err != nil || parsed.Host == "" {
		return nil
	}
	if !strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https") {
		return nil
	}

	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed
}

// sanitizeRepositoryRef strips userinfo, query, and fragment from a
// repository http(s) URL. owner/repo values pass through unchanged.
func sanitizeRepositoryRef(repository string) string {
	repo := strings.TrimRight(strings.TrimSpace(repository), "/")
	if repo == "" {
		return ""
	}
	if hasHTTPScheme(repo) {
		if parsed := parseHTTPRepositoryURL(repo); parsed != nil {
			return parsed.String()
		}
		return ""
	}
	return repo
}

func storeSanitizedRepositoryField(data map[string]any, key, value string) map[string]any {
	if value == "" {
		return data
	}

	sanitized := sanitizeRepositoryRef(value)
	if sanitized == "" {
		if data != nil {
			delete(data, key)
		}
		return data
	}

	data = ensureArtifactData(data)
	data[key] = sanitized
	return data
}

func ensureArtifactData(data map[string]any) map[string]any {
	if data == nil {
		return map[string]any{}
	}
	return data
}

func artifactString(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	value, ok := data[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
