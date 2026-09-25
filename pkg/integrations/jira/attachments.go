package jira

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
)

// IssueAttachment is a file linked to a Jira issue. Webhook payloads may
// include these records under issue.fields.attachment, but a fresh API read
// is still used so late uploads are not missed.
type IssueAttachment struct {
	ID           string
	Filename     string
	MIMEType     string
	ContentURL   string
	ThumbnailURL string
}

// IssueFile is a downloaded Jira file ready to store on a work order.
// ReplaceURLs are the exact description strings that should become the stored
// file reference. An empty list means the file is appended.
type IssueFile struct {
	Name        string
	ContentType string
	Body        []byte
	ReplaceURLs []string
}

// IsJiraAttachmentURL reports whether raw points at a Jira attachment download.
func IsJiraAttachmentURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}

	host := strings.ToLower(parsed.Hostname())
	if host != "api.atlassian.com" && !strings.HasSuffix(host, ".atlassian.net") {
		return false
	}

	lowerPath := strings.ToLower(parsed.Path)
	return strings.Contains(lowerPath, "/rest/api/3/attachment/content/") ||
		strings.Contains(lowerPath, "/rest/api/3/attachment/thumbnail/")
}

// IssueFiles downloads the files attached to an issue, plus Jira attachment
// URLs already written in description. OAuth credentials stay on the request.
func (c *Client) IssueFiles(ctx context.Context, issueKey, description string) ([]IssueFile, error) {
	attachments, err := c.ListIssueAttachments(issueKey)
	if err != nil {
		return nil, err
	}

	planned := planIssueDownloads(description, attachments)
	if len(planned) == 0 {
		return nil, nil
	}

	files := make([]IssueFile, 0, len(planned))
	for _, item := range planned {
		file, ok := c.downloadIssueFile(ctx, item)
		if !ok {
			continue
		}
		files = append(files, file)
	}
	return files, nil
}

// ListIssueAttachments returns files linked to one issue.
func (c *Client) ListIssueAttachments(issueKey string) ([]IssueAttachment, error) {
	issueKey = strings.TrimSpace(issueKey)
	if issueKey == "" {
		return nil, fmt.Errorf("issue key is required")
	}

	issue, err := c.GetIssueWithOptions(issueKey, GetIssueOptions{Fields: "attachment"})
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, nil
	}
	return attachmentsFromFields(issue.Fields), nil
}

func attachmentsFromFields(fields map[string]any) []IssueAttachment {
	raw, ok := fields["attachment"].([]any)
	if !ok {
		return nil
	}

	attachments := make([]IssueAttachment, 0, len(raw))
	for _, entry := range raw {
		document, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		attachment, ok := attachmentFromDocument(document)
		if !ok {
			continue
		}
		attachments = append(attachments, attachment)
	}
	return attachments
}

func attachmentFromDocument(document map[string]any) (IssueAttachment, bool) {
	id := stringAttribute(document["id"])
	if id == "" {
		return IssueAttachment{}, false
	}

	filename := stringAttribute(document["filename"])
	if filename == "" {
		filename = stringAttribute(document["name"])
	}
	if filename == "" {
		filename = "attachment-" + id
	}

	return IssueAttachment{
		ID:           id,
		Filename:     filename,
		MIMEType:     stringAttribute(document["mimeType"]),
		ContentURL:   stringAttribute(document["content"]),
		ThumbnailURL: stringAttribute(document["thumbnail"]),
	}, true
}

func stringAttribute(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

type plannedIssueDownload struct {
	attachmentID string
	name         string
	contentType  string
	replaceURLs  []string
}

func planIssueDownloads(description string, attachments []IssueAttachment) []plannedIssueDownload {
	byID := map[string]*plannedIssueDownload{}
	var planned []*plannedIssueDownload

	add := func(attachmentID, rawURL, name, contentType string, replace bool) {
		key := strings.TrimSpace(attachmentID)
		if key == "" {
			key = attachmentURLKey(rawURL)
		}
		if key == "" {
			return
		}

		item, exists := byID[key]
		if !exists {
			item = &plannedIssueDownload{
				attachmentID: strings.TrimSpace(attachmentID),
				name:         name,
				contentType:  contentType,
			}
			byID[key] = item
			planned = append(planned, item)
		}
		if item.name == "" {
			item.name = name
		}
		if item.contentType == "" {
			item.contentType = contentType
		}
		if item.attachmentID == "" {
			item.attachmentID = attachmentIDFromURL(rawURL)
		}
		if replace && rawURL != "" && !slices.Contains(item.replaceURLs, rawURL) {
			item.replaceURLs = append(item.replaceURLs, rawURL)
		}
	}

	for _, attachment := range attachments {
		replace := descriptionReferencesAttachment(description, attachment)
		add(attachment.ID, attachment.ContentURL, attachment.Filename, attachment.MIMEType, replace)
		if attachment.ThumbnailURL != "" && strings.Contains(description, attachment.ThumbnailURL) {
			add(attachment.ID, attachment.ThumbnailURL, attachment.Filename, attachment.MIMEType, true)
		}
	}
	for _, rawURL := range blob.HTTPResourceURLs(description) {
		if !IsJiraAttachmentURL(rawURL) {
			continue
		}
		add(attachmentIDFromURL(rawURL), rawURL, path.Base(rawURL), "", true)
	}

	result := make([]plannedIssueDownload, 0, len(planned))
	for _, item := range planned {
		result = append(result, *item)
	}
	return result
}

func descriptionReferencesAttachment(description string, attachment IssueAttachment) bool {
	if strings.Contains(description, attachment.ContentURL) {
		return true
	}
	if attachment.ThumbnailURL != "" && strings.Contains(description, attachment.ThumbnailURL) {
		return true
	}
	return strings.Contains(description, attachment.ID)
}

func attachmentIDFromURL(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == "content" || parts[i] == "thumbnail" {
			if i+1 < len(parts) {
				return parts[i+1]
			}
			return ""
		}
	}
	return ""
}

func attachmentURLKey(rawURL string) string {
	id := attachmentIDFromURL(rawURL)
	if id != "" {
		return "id:" + id
	}
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" {
		return ""
	}
	return strings.ToLower(parsed.Hostname()) + parsed.EscapedPath()
}

func (c *Client) downloadIssueFile(ctx context.Context, item plannedIssueDownload) (IssueFile, bool) {
	attachmentID := strings.TrimSpace(item.attachmentID)
	if attachmentID == "" {
		return IssueFile{}, false
	}

	body, contentType, ok := c.fetchAttachmentContent(ctx, attachmentID)
	if !ok {
		return IssueFile{}, false
	}

	if strings.TrimSpace(contentType) == "" {
		contentType = item.contentType
	}
	name := item.name
	if strings.TrimSpace(name) == "" {
		name = "attachment-" + attachmentID
	}
	return IssueFile{
		Name:        name,
		ContentType: contentType,
		Body:        body,
		ReplaceURLs: item.replaceURLs,
	}, true
}

func (c *Client) fetchAttachmentContent(ctx context.Context, attachmentID string) ([]byte, string, bool) {
	fetchCtx, cancel := context.WithTimeout(ctx, issueFileFetchTimeout)
	defer cancel()

	endpoint := c.apiURL("/rest/api/3/attachment/content/" + url.PathEscape(strings.TrimSpace(attachmentID)))
	body, contentType, status, err := c.fetchBinary(fetchCtx, endpoint)
	if err != nil {
		return nil, "", false
	}
	if status == http.StatusUnauthorized {
		if err := c.recoverFromUnauthorized(); err != nil {
			return nil, "", false
		}
		body, contentType, status, err = c.fetchBinary(fetchCtx, endpoint)
		if err != nil || status < 200 || status >= 300 {
			return nil, "", false
		}
	}
	if status < 200 || status >= 300 {
		return nil, "", false
	}
	if int64(len(body)) > maxIssueFileBytes {
		return nil, "", false
	}
	return body, contentType, true
}

func (c *Client) fetchBinary(ctx context.Context, requestURL string) ([]byte, string, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, "", 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Accept", "*/*")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, "", 0, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, maxIssueFileBytes+1))
	if err != nil {
		return nil, "", res.StatusCode, err
	}
	return body, res.Header.Get("Content-Type"), res.StatusCode, nil
}
