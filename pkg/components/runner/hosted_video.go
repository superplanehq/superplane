package runner

import (
	"encoding/json"
	"net/url"
	"path"
	"regexp"
	"strings"
	"sync"

	_ "embed"
)

const HostedVideoKind = "hosted_video"

//go:embed hosted_video_hosts.json
var hostedVideoHostsJSON []byte

type hostedVideoCatalog struct {
	MaxDurationSeconds int               `json:"maxDurationSeconds"`
	MaxBytes           int64             `json:"maxBytes"`
	Hosts              []hostedVideoHost `json:"hosts"`
}

type hostedVideoHost struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Suffixes []string `json:"suffixes"`
}

type HostedVideo struct {
	ProviderID   string
	ProviderName string
	ID           string
	PageURL      string
}

var (
	hostedVideoCatalogOnce sync.Once
	hostedVideoCatalogData hostedVideoCatalog
	markdownImagePattern   = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+)\)`)
	youtubeIDPattern       = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
	vimeoIDPattern         = regexp.MustCompile(`^[0-9]{6,12}$`)
	loomIDPattern          = regexp.MustCompile(`^[A-Za-z0-9]{16,64}$`)
)

func hostedVideoCatalogValue() hostedVideoCatalog {
	hostedVideoCatalogOnce.Do(func() {
		if err := json.Unmarshal(hostedVideoHostsJSON, &hostedVideoCatalogData); err != nil {
			panic(err)
		}
	})
	return hostedVideoCatalogData
}

func HostedVideoMaxDurationSeconds() int {
	if seconds := hostedVideoCatalogValue().MaxDurationSeconds; seconds > 0 {
		return seconds
	}
	return 300
}

func HostedVideoMaxBytes() int64 {
	if bytes := hostedVideoCatalogValue().MaxBytes; bytes > 0 {
		return bytes
	}
	return 256 << 20
}

func HostedVideoAttachments(texts ...string) []TaskAttachment {
	seen := map[string]struct{}{}
	var attachments []TaskAttachment
	for _, text := range texts {
		for _, match := range markdownImagePattern.FindAllStringSubmatch(text, -1) {
			video, ok := ParseHostedVideoURL(match[2])
			if !ok {
				continue
			}
			if _, exists := seen[video.PageURL]; exists {
				continue
			}
			seen[video.PageURL] = struct{}{}
			attachments = append(attachments, TaskAttachment{
				URL:      video.PageURL,
				Filename: hostedVideoFilename(video),
				Kind:     HostedVideoKind,
			})
		}
	}
	return attachments
}

func (a TaskAttachment) RunnerFile() map[string]any {
	out := map[string]any{
		"filename": a.Filename,
		"url":      a.URL,
		"kind":     attachmentKind(a),
	}
	if a.ID != "" {
		out["id"] = a.ID
	}
	if a.ContentType != "" {
		out["content_type"] = a.ContentType
	}
	if a.SizeBytes > 0 {
		out["size_bytes"] = a.SizeBytes
	}
	if a.Checksum != "" {
		out["checksum"] = a.Checksum
	}
	return out
}

func ParseHostedVideoURL(raw string) (HostedVideo, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return HostedVideo{}, false
	}
	if strings.ContainsAny(parsed.String(), " \t\r\n") {
		return HostedVideo{}, false
	}
	host := hostedVideoHostFor(parsed.Hostname())
	if host.ID == "" {
		return HostedVideo{}, false
	}
	id, ok := hostedVideoID(host.ID, parsed)
	if !ok {
		return HostedVideo{}, false
	}
	return HostedVideo{
		ProviderID:   host.ID,
		ProviderName: host.Name,
		ID:           id,
		PageURL:      parsed.String(),
	}, true
}

func hostedVideoHostFor(hostname string) hostedVideoHost {
	host := strings.ToLower(strings.TrimSuffix(hostname, "."))
	for _, candidate := range hostedVideoCatalogValue().Hosts {
		for _, suffix := range candidate.Suffixes {
			if host == suffix || strings.HasSuffix(host, "."+suffix) {
				return candidate
			}
		}
	}
	return hostedVideoHost{}
}

func hostedVideoID(provider string, parsed *url.URL) (string, bool) {
	switch provider {
	case "youtube":
		return youtubeVideoID(parsed)
	case "vimeo":
		return vimeoVideoID(parsed)
	case "loom":
		return loomVideoID(parsed)
	default:
		return genericHostedVideoID(parsed)
	}
}

func youtubeVideoID(parsed *url.URL) (string, bool) {
	host := strings.ToLower(parsed.Hostname())
	if host == "youtu.be" || strings.HasSuffix(host, ".youtu.be") {
		id := firstPathSegment(parsed.Path)
		if youtubeIDPattern.MatchString(id) {
			return id, true
		}
		return "", false
	}
	if id := parsed.Query().Get("v"); youtubeIDPattern.MatchString(id) && pathIs(parsed.Path, "watch") {
		return id, true
	}
	parts := pathSegments(parsed.Path)
	if len(parts) >= 2 && (parts[0] == "shorts" || parts[0] == "embed" || parts[0] == "live" || parts[0] == "v") {
		if youtubeIDPattern.MatchString(parts[1]) {
			return parts[1], true
		}
	}
	return "", false
}

func vimeoVideoID(parsed *url.URL) (string, bool) {
	parts := pathSegments(parsed.Path)
	if len(parts) >= 2 && parts[0] == "video" && vimeoIDPattern.MatchString(parts[1]) {
		return parts[1], true
	}
	if len(parts) >= 3 && parts[len(parts)-2] == "videos" && vimeoIDPattern.MatchString(parts[len(parts)-1]) {
		return parts[len(parts)-1], true
	}
	if len(parts) >= 3 && parts[len(parts)-2] == "video" && vimeoIDPattern.MatchString(parts[len(parts)-1]) {
		return parts[len(parts)-1], true
	}
	if len(parts) >= 1 && vimeoIDPattern.MatchString(parts[0]) {
		return parts[0], true
	}
	return "", false
}

func loomVideoID(parsed *url.URL) (string, bool) {
	parts := pathSegments(parsed.Path)
	if len(parts) >= 2 && (parts[0] == "share" || parts[0] == "embed") && loomIDPattern.MatchString(parts[1]) {
		return parts[1], true
	}
	return "", false
}

func genericHostedVideoID(parsed *url.URL) (string, bool) {
	parts := pathSegments(parsed.Path)
	if len(parts) == 0 {
		return "", false
	}
	id := parts[len(parts)-1]
	if len(id) < 4 {
		return "", false
	}
	return id, true
}

func hostedVideoFilename(video HostedVideo) string {
	name := video.ProviderID
	if video.ID != "" {
		name += "-" + video.ID
	}
	return sanitizeAttachmentName(name)
}

func pathSegments(raw string) []string {
	cleaned := strings.Trim(path.Clean("/"+raw), "/")
	if cleaned == "" || cleaned == "." {
		return nil
	}
	return strings.Split(cleaned, "/")
}

func firstPathSegment(raw string) string {
	parts := pathSegments(raw)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func pathIs(raw, name string) bool {
	parts := pathSegments(raw)
	return len(parts) == 1 && parts[0] == name
}
