package blob

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

const FileRefScheme = "sp-file"

var (
	markdownLinkPattern = regexp.MustCompile(`(!?\[[^\]]*]\()([^)\s]+)(\))`)
	htmlSrcPattern      = regexp.MustCompile(`(?i)(<img\b[^>]*?\bsrc\s*=\s*["'])([^"']+)(["'])`)
)

func FileRef(id uuid.UUID) string {
	return FileRefScheme + "://" + id.String()
}

func ParseFileID(raw string) (uuid.UUID, bool) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, FileRefScheme+"://") {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(strings.TrimPrefix(trimmed, FileRefScheme+"://"))
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

func FileIDsInMarkdown(markdown string) []uuid.UUID {
	seen := map[uuid.UUID]struct{}{}
	var ids []uuid.UUID
	collect := func(raw string) {
		id, ok := ParseFileID(raw)
		if !ok {
			return
		}
		if _, exists := seen[id]; exists {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	for _, match := range markdownLinkPattern.FindAllStringSubmatch(markdown, -1) {
		collect(match[2])
	}
	for _, match := range htmlSrcPattern.FindAllStringSubmatch(markdown, -1) {
		collect(match[2])
	}
	return ids
}

func RewriteFileRefs(markdown string, urls map[uuid.UUID]string) string {
	replace := func(raw string) string {
		id, ok := ParseFileID(raw)
		if !ok {
			return raw
		}
		next, exists := urls[id]
		if !exists || strings.TrimSpace(next) == "" {
			return raw
		}
		return next
	}
	out := markdownLinkPattern.ReplaceAllStringFunc(markdown, func(match string) string {
		parts := markdownLinkPattern.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}
		return parts[1] + replace(parts[2]) + parts[3]
	})
	return htmlSrcPattern.ReplaceAllStringFunc(out, func(match string) string {
		parts := htmlSrcPattern.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}
		return parts[1] + replace(parts[2]) + parts[3]
	})
}

func ReplaceURL(markdown, from, to string) string {
	if from == "" || to == "" {
		return markdown
	}
	return strings.ReplaceAll(markdown, from, to)
}

func HTTPImageURLs(markdown string) []string {
	seen := map[string]struct{}{}
	var urls []string
	collect := func(raw string) {
		parsed, err := url.Parse(strings.TrimSpace(raw))
		if err != nil || parsed.Host == "" {
			return
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return
		}
		if _, exists := seen[raw]; exists {
			return
		}
		seen[raw] = struct{}{}
		urls = append(urls, raw)
	}
	for _, match := range markdownLinkPattern.FindAllStringSubmatch(markdown, -1) {
		if strings.HasPrefix(match[1], "!") {
			collect(match[2])
		}
	}
	for _, match := range htmlSrcPattern.FindAllStringSubmatch(markdown, -1) {
		collect(match[2])
	}
	return urls
}

func IsGitHubImageHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	switch host {
	case "user-images.githubusercontent.com",
		"private-user-images.githubusercontent.com",
		"objects.githubusercontent.com",
		"github.com":
		return true
	default:
		return strings.HasSuffix(host, ".githubusercontent.com")
	}
}

func IsGitHubImageURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	if !IsGitHubImageHost(parsed.Host) {
		return false
	}
	if parsed.Host != "github.com" {
		return true
	}
	path := strings.ToLower(parsed.Path)
	return strings.Contains(path, "/assets/") || strings.Contains(path, "/user-attachments/")
}
