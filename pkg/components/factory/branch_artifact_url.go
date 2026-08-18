package factory

import (
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

	if strings.HasPrefix(repo, "https://") || strings.HasPrefix(repo, "http://") {
		parsed, err := url.Parse(repo)
		if err != nil || parsed.Host == "" {
			return ""
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return ""
		}
		parsed.User = nil
		parsed.RawQuery = ""
		parsed.Fragment = ""
		// Assign the raw branch name to Path. String() encodes reserved
		// characters. Pre-escaping here would double-encode `#` as `%2523`.
		parsed.Path = strings.TrimRight(parsed.Path, "/") + "/tree/" + branch
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

	if config.Repository != "" {
		data = ensureArtifactData(data)
		data["repository"] = config.Repository
	}

	if artifactString(data, "url") != "" || artifactString(data, "html_url") != "" {
		return data
	}

	repo := config.Repository
	if repo == "" {
		repo = artifactString(data, "repository")
	}
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
