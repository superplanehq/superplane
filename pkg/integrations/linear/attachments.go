package linear

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/blob"
)

const (
	// maxIssueFileBytes matches the work-order attachment limit. A larger
	// download is skipped so one file cannot exhaust memory.
	maxIssueFileBytes = 10 << 20

	issueFileFetchTimeout = 30 * time.Second

	linearUploadHost = "uploads.linear.app"
)

// IssueFile is a downloaded Linear file ready to store on a work order.
// ReplaceURLs are the description strings that should become the stored
// file reference. An empty list means the file is appended.
type IssueFile struct {
	Name        string
	ContentType string
	Body        []byte
	ReplaceURLs []string
}

// IssueLink is a Linear attachment that points at another service. It is
// not a file, so the task lists it as a link.
type IssueLink struct {
	Title string
	URL   string
}

// IsLinearUploadURL reports whether raw points at Linear's private file storage.
func IsLinearUploadURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return strings.EqualFold(parsed.Hostname(), linearUploadHost)
}

// IssueFiles downloads files on uploads.linear.app that are embedded in the
// description or listed as issue attachments. Link attachments are returned
// separately. OAuth credentials stay on the request.
func (c *Client) IssueFiles(ctx context.Context, issueID, description string) ([]IssueFile, []IssueLink, error) {
	attachments, err := c.ListIssueAttachments(issueID)
	if err != nil {
		return nil, nil, err
	}

	links := linkAttachments(attachments)
	planned := planLinearDownloads(description, attachments)
	if len(planned) == 0 {
		return nil, links, nil
	}

	files := make([]IssueFile, 0, len(planned))
	for _, item := range planned {
		file, ok := c.downloadLinearFile(ctx, item)
		if !ok {
			continue
		}
		files = append(files, file)
	}
	return files, links, nil
}

func linkAttachments(attachments []Attachment) []IssueLink {
	links := make([]IssueLink, 0)
	for _, attachment := range attachments {
		if IsLinearUploadURL(attachment.URL) {
			continue
		}
		rawURL := strings.TrimSpace(attachment.URL)
		if rawURL == "" {
			continue
		}
		title := strings.TrimSpace(attachment.Title)
		if title == "" {
			title = rawURL
		}
		links = append(links, IssueLink{Title: title, URL: rawURL})
	}
	return links
}

// LinkSectionMarkdown renders link attachments as a markdown list. An empty
// list returns an empty string.
func LinkSectionMarkdown(links []IssueLink) string {
	if len(links) == 0 {
		return ""
	}
	lines := make([]string, 0, len(links)+2)
	lines = append(lines, "## Attachments", "")
	for _, link := range links {
		lines = append(lines, fmt.Sprintf("- [%s](%s)", link.Title, link.URL))
	}
	return strings.Join(lines, "\n")
}

type plannedLinearDownload struct {
	name        string
	contentType string
	rawURL      string
	replaceURLs []string
}

func planLinearDownloads(description string, attachments []Attachment) []plannedLinearDownload {
	byURL := map[string]*plannedLinearDownload{}
	var planned []*plannedLinearDownload

	add := func(rawURL, name string, replace bool) {
		rawURL = strings.TrimSpace(rawURL)
		if !IsLinearUploadURL(rawURL) {
			return
		}
		key := linearUploadKey(rawURL)
		if key == "" {
			return
		}
		item, exists := byURL[key]
		if !exists {
			item = &plannedLinearDownload{
				name:   name,
				rawURL: rawURL,
			}
			byURL[key] = item
			planned = append(planned, item)
		}
		if item.name == "" {
			item.name = name
		}
		if replace && !slices.Contains(item.replaceURLs, rawURL) {
			item.replaceURLs = append(item.replaceURLs, rawURL)
		}
	}

	for _, attachment := range attachments {
		replace := strings.Contains(description, attachment.URL)
		name := strings.TrimSpace(attachment.Title)
		if name == "" {
			name = path.Base(attachment.URL)
		}
		add(attachment.URL, name, replace)
	}
	for _, rawURL := range blob.HTTPResourceURLs(description) {
		add(rawURL, path.Base(rawURL), true)
	}

	result := make([]plannedLinearDownload, 0, len(planned))
	for _, item := range planned {
		result = append(result, *item)
	}
	return result
}

func linearUploadKey(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" {
		return ""
	}
	return strings.ToLower(parsed.Hostname()) + parsed.EscapedPath()
}

func (c *Client) downloadLinearFile(ctx context.Context, item plannedLinearDownload) (IssueFile, bool) {
	body, contentType, ok := c.fetchLinearUpload(ctx, item.rawURL)
	if !ok {
		return IssueFile{}, false
	}
	name := strings.TrimSpace(item.name)
	if name == "" || name == "." || name == "/" {
		name = "attachment"
	}
	return IssueFile{
		Name:        name,
		ContentType: contentType,
		Body:        body,
		ReplaceURLs: item.replaceURLs,
	}, true
}

func (c *Client) fetchLinearUpload(ctx context.Context, rawURL string) ([]byte, string, bool) {
	if !IsLinearUploadURL(rawURL) {
		return nil, "", false
	}
	fetchCtx, cancel := context.WithTimeout(ctx, issueFileFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(fetchCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", false
	}
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Accept", "*/*")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, "", false
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, maxIssueFileBytes+1))
	if err != nil || res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, "", false
	}
	if int64(len(body)) > maxIssueFileBytes {
		return nil, "", false
	}
	return body, res.Header.Get("Content-Type"), true
}
