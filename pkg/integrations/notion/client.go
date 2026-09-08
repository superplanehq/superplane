package notion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	// databasesPageSize is how many databases each page requests when
	// listing the databases shared with the integration.
	databasesPageSize = 100

	// maxDatabasePages bounds how many pages ListDatabases walks, so a
	// misbehaving pagination cursor cannot loop forever.
	maxDatabasePages = 20

	// maxContentBlocks bounds how many blocks of a page are read for its
	// content. A page longer than that is truncated rather than fully read on
	// every poll.
	maxContentBlocks = 100

	// searchPageSize is how many of a database's newest pages ListPages reads
	// before filtering by title. Notion's search API cannot filter by parent
	// database, so matching happens after the read.
	searchPageSize = 100

	sortTimestampCreated = "created_time"
	sortDirectionNewest  = "descending"
	sortDirectionOldest  = "ascending"
)

// Client talks to Notion's REST API using an internal integration token sent
// as a bearer token on every request.
type Client struct {
	APIToken string
	BaseURL  string
	http     core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	apiToken, err := ctx.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("error getting apiToken: %v", err)
	}

	if strings.TrimSpace(string(apiToken)) == "" {
		return nil, fmt.Errorf("missing Notion internal integration token")
	}

	if httpCtx == nil {
		return nil, fmt.Errorf("missing HTTP context")
	}

	return &Client{
		APIToken: strings.TrimSpace(string(apiToken)),
		BaseURL:  BaseURL,
		http:     httpCtx,
	}, nil
}

func (c *Client) execRequest(method, requestURL string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIToken)
	req.Header.Set("Notion-Version", APIVersion)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

func (c *Client) execJSONRequest(method, requestURL string, body map[string]any) ([]byte, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	return c.execRequest(method, requestURL, bytes.NewReader(encoded))
}

// ValidateCredentials confirms the token is accepted by Notion. Reading the
// integration's own bot user is the cheapest authenticated endpoint, and
// needs no capability beyond what every internal integration already has.
func (c *Client) ValidateCredentials() error {
	_, err := c.execRequest(http.MethodGet, fmt.Sprintf("%s/users/me", c.BaseURL), nil)
	return err
}

// Database is a Notion database shared with the integration, used by the
// onPageAdded trigger's database picker and to scope a poll to one database.
type Database struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func databaseFromObject(object map[string]any) Database {
	id, _ := object["id"].(string)
	return Database{ID: id, Name: titleFromRichText(object["title"])}
}

// titleFromRichText joins the plain text of a Notion rich text array, the
// shape Notion uses for a database's title and for a page property's title.
func titleFromRichText(value any) string {
	segments, ok := value.([]any)
	if !ok {
		return ""
	}

	var builder strings.Builder
	for _, segment := range segments {
		part, ok := segment.(map[string]any)
		if !ok {
			continue
		}
		plainText, _ := part["plain_text"].(string)
		builder.WriteString(plainText)
	}

	return builder.String()
}

// titleFromProperties finds the title property of a page - the one property
// every database page has exactly one of - and returns its plain text.
// Databases name that property differently, so the type is what is matched
// on, not the name.
func titleFromProperties(properties map[string]any) string {
	for _, property := range properties {
		propertyMap, ok := property.(map[string]any)
		if !ok {
			continue
		}
		if propertyMap["type"] != "title" {
			continue
		}
		return titleFromRichText(propertyMap["title"])
	}

	return ""
}

// PageTitle reads the plain text of a page's title property, whatever that
// property is named in the source database.
func PageTitle(page map[string]any) string {
	properties, _ := page["properties"].(map[string]any)
	return titleFromProperties(properties)
}

// ListDatabases returns every database shared with the integration. Notion
// paginates responses, so pages are walked until a short page ends them.
func (c *Client) ListDatabases() ([]Database, error) {
	databases := []Database{}
	startCursor := ""

	for page := 0; page < maxDatabasePages; page++ {
		body := map[string]any{
			"filter":    map[string]any{"property": "object", "value": "database"},
			"page_size": databasesPageSize,
		}
		if startCursor != "" {
			body["start_cursor"] = startCursor
		}

		responseBody, err := c.execJSONRequest(http.MethodPost, fmt.Sprintf("%s/search", c.BaseURL), body)
		if err != nil {
			return nil, err
		}

		response := searchResponse{}
		if err := json.Unmarshal(responseBody, &response); err != nil {
			return nil, fmt.Errorf("error parsing databases: %v", err)
		}

		for _, object := range response.Results {
			databases = append(databases, databaseFromObject(object))
		}

		if !response.HasMore || response.NextCursor == "" {
			return databases, nil
		}
		startCursor = response.NextCursor
	}

	return databases, nil
}

type searchResponse struct {
	Results    []map[string]any `json:"results"`
	HasMore    bool             `json:"has_more"`
	NextCursor string           `json:"next_cursor"`
}

// GetDatabase fetches a single database by id, for resolving the database a
// trigger was configured with into the name shown on its canvas card.
func (c *Client) GetDatabase(id string) (*Database, error) {
	body, err := c.execRequest(http.MethodGet, fmt.Sprintf("%s/databases/%s", c.BaseURL, url.PathEscape(id)), nil)
	if err != nil {
		return nil, err
	}

	var object map[string]any
	if err := json.Unmarshal(body, &object); err != nil {
		return nil, fmt.Errorf("error parsing database: %v", err)
	}

	if databaseID, _ := object["id"].(string); databaseID == "" {
		return nil, fmt.Errorf("database %s not found", id)
	}

	database := databaseFromObject(object)
	return &database, nil
}

// queryOptions describes one page of a database's pages.
type queryOptions struct {
	startCursor string
	pageSize    int

	// oldestFirst sorts the pages by creation time ascending rather than the
	// default descending. A poll reads oldest first so it can advance its
	// cursor as it goes and never skip a page when more were added than a
	// single poll can read.
	oldestFirst bool

	// createdAfter, when set, asks Notion to return only pages created strictly
	// after this RFC3339 timestamp. Filtering server-side keeps a poll bounded
	// to the pages that are actually new.
	createdAfter string
}

// queryDatabase reads one page of a database's pages, newest created first by
// default, or oldest first when the options ask for it.
func (c *Client) queryDatabase(databaseID string, options queryOptions) ([]map[string]any, bool, string, error) {
	direction := sortDirectionNewest
	if options.oldestFirst {
		direction = sortDirectionOldest
	}

	body := map[string]any{
		"sorts":     []any{map[string]any{"timestamp": sortTimestampCreated, "direction": direction}},
		"page_size": options.pageSize,
	}
	if options.startCursor != "" {
		body["start_cursor"] = options.startCursor
	}
	if options.createdAfter != "" {
		body["filter"] = map[string]any{
			"timestamp":          sortTimestampCreated,
			sortTimestampCreated: map[string]any{"after": options.createdAfter},
		}
	}

	responseBody, err := c.execJSONRequest(
		http.MethodPost,
		fmt.Sprintf("%s/databases/%s/query", c.BaseURL, url.PathEscape(databaseID)),
		body,
	)
	if err != nil {
		return nil, false, "", err
	}

	response := searchResponse{}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, false, "", fmt.Errorf("error parsing pages: %v", err)
	}

	return response.Results, response.HasMore, response.NextCursor, nil
}

// ListNewestPages returns the newest pages of the database, most recently
// created first. Seeding an intake replays these through the graph the
// trigger feeds.
func (c *Client) ListNewestPages(databaseID string, limit int) ([]map[string]any, error) {
	results, _, _, err := c.queryDatabase(databaseID, queryOptions{pageSize: limit})
	return results, err
}

// ListChangedPageDocuments returns one page of the database's pages created
// after createdAfter, oldest created first, starting at cursor. The onPageAdded
// trigger walks these oldest first so a burst larger than one poll can read is
// caught up over several polls instead of skipping the overflow.
func (c *Client) ListChangedPageDocuments(databaseID, cursor, createdAfter string, pageSize int) ([]map[string]any, bool, string, error) {
	return c.queryDatabase(databaseID, queryOptions{
		startCursor:  cursor,
		pageSize:     pageSize,
		oldestFirst:  true,
		createdAfter: createdAfter,
	})
}

// PageContent returns the plain text content of a page: its blocks, joined by
// blank lines. Only the first maxContentBlocks blocks are read.
func (c *Client) PageContent(pageID string) (string, error) {
	requestURL := fmt.Sprintf("%s/blocks/%s/children?page_size=%d", c.BaseURL, url.PathEscape(pageID), maxContentBlocks)
	body, err := c.execRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return "", err
	}

	response := struct {
		Results []map[string]any `json:"results"`
	}{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("error parsing blocks: %v", err)
	}

	lines := make([]string, 0, len(response.Results))
	for _, block := range response.Results {
		if text := blockPlainText(block); text != "" {
			lines = append(lines, text)
		}
	}

	return strings.Join(lines, "\n\n"), nil
}

// blockPlainText reads the visible text of one Notion block. Blocks that
// carry no rich text (dividers, images, and similar) contribute nothing.
func blockPlainText(block map[string]any) string {
	blockType, _ := block["type"].(string)
	body, _ := block[blockType].(map[string]any)
	text := titleFromRichText(body["rich_text"])
	if text == "" {
		return ""
	}

	switch blockType {
	case "bulleted_list_item", "numbered_list_item":
		return "- " + text
	case "to_do":
		if checked, _ := body["checked"].(bool); checked {
			return "[x] " + text
		}
		return "[ ] " + text
	case "quote":
		return "> " + text
	default:
		return text
	}
}

// Page is a Notion page that can seed a factory intake or be imported by hand.
type Page struct {
	ID      string
	Title   string
	Content string
	URL     string

	// ParentDatabaseID is the database the page belongs to, when the page is a
	// database entry. It lets a caller confirm an imported page belongs to the
	// database the intake is scoped to, rather than trusting a caller-supplied
	// id alone.
	ParentDatabaseID string
}

// GetPage fetches a single page by id, with its content, for importing an
// item found through Search.
func (c *Client) GetPage(id string) (*Page, error) {
	body, err := c.execRequest(http.MethodGet, fmt.Sprintf("%s/pages/%s", c.BaseURL, url.PathEscape(id)), nil)
	if err != nil {
		return nil, err
	}

	var object map[string]any
	if err := json.Unmarshal(body, &object); err != nil {
		return nil, fmt.Errorf("error parsing page: %v", err)
	}

	pageID, _ := object["id"].(string)
	if pageID == "" {
		return nil, fmt.Errorf("page %s not found", id)
	}

	content, err := c.PageContent(pageID)
	if err != nil {
		return nil, err
	}

	pageURL, _ := object["url"].(string)
	return &Page{
		ID:               pageID,
		Title:            PageTitle(object),
		Content:          content,
		URL:              pageURL,
		ParentDatabaseID: parentDatabaseID(object),
	}, nil
}

// parentDatabaseID reads the id of the database a page belongs to. Pages that
// live outside a database (or whose parent Notion did not return) yield an
// empty id, which callers treat as out of scope.
func parentDatabaseID(object map[string]any) string {
	parent, ok := object["parent"].(map[string]any)
	if !ok {
		return ""
	}
	id, _ := parent["database_id"].(string)
	return id
}

// SameDatabase reports whether two Notion database ids refer to the same
// database. Notion returns ids both with and without the dashes of their UUID
// form, so the comparison ignores dashes and case. An empty id never matches,
// so a page whose parent database could not be determined is out of scope.
func SameDatabase(a, b string) bool {
	na, nb := normalizeID(a), normalizeID(b)
	return na != "" && na == nb
}

func normalizeID(id string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(id), "-", ""))
}

// ListPages returns the pages of the database whose title contains query
// (case-insensitively), newest created first. Query results carry no page
// content: fetching it for every match would cost one request each, so
// callers that need it call GetPage on the chosen page instead.
func (c *Client) ListPages(databaseID, query string, limit int) ([]Page, error) {
	results, err := c.ListNewestPages(databaseID, searchPageSize)
	if err != nil {
		return nil, err
	}

	needle := strings.ToLower(strings.TrimSpace(query))
	pages := make([]Page, 0, limit)
	for _, object := range results {
		if len(pages) == limit {
			break
		}

		title := PageTitle(object)
		if needle != "" && !strings.Contains(strings.ToLower(title), needle) {
			continue
		}

		id, _ := object["id"].(string)
		pageURL, _ := object["url"].(string)
		pages = append(pages, Page{ID: id, Title: title, URL: pageURL})
	}

	return pages, nil
}
