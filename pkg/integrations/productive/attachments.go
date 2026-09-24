package productive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	// maxTaskFileBytes matches the work-order attachment limit. A larger
	// download is skipped so one file cannot exhaust memory.
	maxTaskFileBytes = 10 << 20

	taskFileFetchTimeout = 30 * time.Second
)

// Attachment is a file linked to a Productive.io task. The webhook payload
// does not include these records. They come from GET /attachments.
type Attachment struct {
	Name        string
	ContentType string
	URL         string
}

// TaskFile is a downloaded Productive.io file ready to store on a work order.
// ReplaceURLs are the exact description strings that should become the stored
// file reference. An empty list means the file is appended.
type TaskFile struct {
	Name        string
	ContentType string
	Body        []byte
	ReplaceURLs []string
}

// TaskIDFromEventData reads the Productive.io task id from a canvas root
// event. The envelope looks like:
//
//	{ "type": "productive.task", "data": { "data": { "id": "20305431" } } }
func TaskIDFromEventData(eventData any) (string, bool) {
	envelope, ok := eventData.(map[string]any)
	if !ok {
		return "", false
	}
	if typeName, _ := envelope["type"].(string); typeName != TaskPayloadType {
		return "", false
	}

	outer, ok := envelope["data"].(map[string]any)
	if !ok {
		return "", false
	}
	document, ok := outer["data"].(map[string]any)
	if !ok {
		return "", false
	}
	taskID := strings.TrimSpace(stringAttribute(document["id"]))
	if taskID == "" {
		return "", false
	}
	return taskID, true
}

// IsProductiveFileURL reports whether raw points at a Productive.io file
// download. API resources under /api/v2/attachments are not file downloads.
func IsProductiveFileURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return false
	}
	if parsed.Scheme != "https" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "files.productive.io" && !strings.HasSuffix(host, ".productive.io") {
		return false
	}
	return strings.Contains(strings.ToLower(parsed.Path), "/attachments/files/")
}

// TaskFiles downloads the files attached to a task, plus Productive.io file
// URLs already written in description. The API token is added only on the
// download request. It is not returned.
func (c *Client) TaskFiles(ctx context.Context, taskID, description string) ([]TaskFile, error) {
	attachments, err := c.ListTaskAttachments(taskID)
	if err != nil {
		return nil, err
	}

	planned := planTaskDownloads(description, attachments)
	if len(planned) == 0 {
		return nil, nil
	}

	files := make([]TaskFile, 0, len(planned))
	for _, item := range planned {
		if len(files) >= models.MaxFilesPerWorkOrder {
			break
		}
		file, ok := c.downloadTaskFile(ctx, item)
		if !ok {
			continue
		}
		files = append(files, file)
	}
	return files, nil
}

// ListTaskAttachments returns files linked to one task. Deleted rows are omitted.
func (c *Client) ListTaskAttachments(taskID string) ([]Attachment, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, fmt.Errorf("task id is required")
	}

	params := url.Values{}
	params.Set("filter[task_id]", taskID)
	params.Set("page[size]", "30")
	body, err := c.execRequest(http.MethodGet, fmt.Sprintf("%s/attachments?%s", c.BaseURL, params.Encode()), nil)
	if err != nil {
		return nil, err
	}

	response := resourceListResponse{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error parsing attachments: %v", err)
	}

	attachments := make([]Attachment, 0, len(response.Data))
	for _, doc := range response.Data {
		attachment, ok := attachmentFromDocument(doc)
		if !ok {
			continue
		}
		attachments = append(attachments, attachment)
	}
	return attachments, nil
}

func attachmentFromDocument(doc resourceDocument) (Attachment, bool) {
	if deleted, exists := doc.Attributes["deleted_at"]; exists && deleted != nil {
		if text, ok := deleted.(string); !ok || strings.TrimSpace(text) != "" {
			return Attachment{}, false
		}
	}

	fileURL := stringAttribute(doc.Attributes["url"])
	if fileURL == "" {
		fileURL = stringAttribute(doc.Attributes["temp_url"])
	}
	if !IsProductiveFileURL(fileURL) {
		return Attachment{}, false
	}

	name := stringAttribute(doc.Attributes["name"])
	if name == "" {
		name = path.Base(fileURL)
	}
	return Attachment{
		Name:        name,
		ContentType: stringAttribute(doc.Attributes["content_type"]),
		URL:         fileURL,
	}, true
}

type plannedDownload struct {
	downloadURL string
	name        string
	contentType string
	replaceURLs []string
}

func planTaskDownloads(description string, attachments []Attachment) []plannedDownload {
	byPath := map[string]*plannedDownload{}
	var planned []*plannedDownload

	add := func(rawURL, name, contentType string, replaceURLs []string) {
		key := filePathKey(rawURL)
		if key == "" {
			return
		}
		item, exists := byPath[key]
		if !exists {
			item = &plannedDownload{
				downloadURL: rawURL,
				name:        name,
				contentType: contentType,
			}
			byPath[key] = item
			planned = append(planned, item)
		}
		if item.name == "" {
			item.name = name
		}
		if item.contentType == "" {
			item.contentType = contentType
		}
		for _, replaceURL := range replaceURLs {
			if replaceURL == "" || slices.Contains(item.replaceURLs, replaceURL) {
				continue
			}
			item.replaceURLs = append(item.replaceURLs, replaceURL)
		}
	}

	for _, attachment := range attachments {
		key := filePathKey(attachment.URL)
		add(attachment.URL, attachment.Name, attachment.ContentType, descriptionURLsWithPathKey(description, key))
	}
	for _, rawURL := range blob.HTTPResourceURLs(description) {
		if !IsProductiveFileURL(rawURL) {
			continue
		}
		add(rawURL, path.Base(rawURL), "", []string{rawURL})
	}

	result := make([]plannedDownload, 0, len(planned))
	for _, item := range planned {
		result = append(result, *item)
	}
	return result
}

func (c *Client) downloadTaskFile(ctx context.Context, item plannedDownload) (TaskFile, bool) {
	fetchCtx, cancel := context.WithTimeout(ctx, taskFileFetchTimeout)
	defer cancel()

	downloadURL, err := withAccessToken(item.downloadURL, c.APIToken)
	if err != nil {
		return TaskFile{}, false
	}

	req, err := http.NewRequestWithContext(fetchCtx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return TaskFile{}, false
	}
	req.Header.Set(AuthTokenHeader, c.APIToken)
	req.Header.Set(OrganizationIDHeader, c.OrganizationID)

	res, err := c.http.Do(req)
	if err != nil {
		return TaskFile{}, false
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return TaskFile{}, false
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, maxTaskFileBytes+1))
	if err != nil || int64(len(body)) > maxTaskFileBytes {
		return TaskFile{}, false
	}

	contentType := res.Header.Get("Content-Type")
	if strings.TrimSpace(contentType) == "" {
		contentType = item.contentType
	}
	name := item.name
	if strings.TrimSpace(name) == "" {
		name = path.Base(item.downloadURL)
	}
	return TaskFile{
		Name:        name,
		ContentType: contentType,
		Body:        body,
		ReplaceURLs: item.replaceURLs,
	}, true
}

func withAccessToken(rawURL, token string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	if strings.TrimSpace(query.Get("token")) == "" && strings.TrimSpace(token) != "" {
		query.Set("token", token)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func filePathKey(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" {
		return ""
	}
	return strings.ToLower(parsed.Hostname()) + parsed.EscapedPath()
}

func descriptionURLsWithPathKey(description, key string) []string {
	if key == "" {
		return nil
	}
	matches := make([]string, 0, 1)
	for _, rawURL := range blob.HTTPResourceURLs(description) {
		if filePathKey(rawURL) != key {
			continue
		}
		matches = append(matches, rawURL)
	}
	return matches
}
