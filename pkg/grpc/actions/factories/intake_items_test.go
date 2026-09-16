package factories

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/integrations/jira"
)

func TestGitHubIssueItem_UsesNumberKeyAndHTMLURL(t *testing.T) {
	number := 12
	title := "Handle duplicate refunds"
	body := "Retrying a refund posts twice."
	url := "https://github.com/acme/payments/issues/12"
	issue := &github.Issue{
		Number:  &number,
		Title:   &title,
		Body:    &body,
		HTMLURL: &url,
	}

	assert.Equal(t, IntakeItem{
		ID:    "12",
		Key:   "#12",
		Title: title,
		Body:  body,
		URL:   url,
	}, gitHubIssueItem(issue))
}

func TestIntakeItemLimit(t *testing.T) {
	assert.Equal(t, defaultLatestIntakeItems, intakeItemLimit("", 0))
	assert.Equal(t, defaultSearchIntakeItems, intakeItemLimit("refund", 0))
	assert.Equal(t, 3, intakeItemLimit("", 3))
	assert.Equal(t, maxIntakeItems, intakeItemLimit("refund", 100))
}

func TestGitHubIssueSearchQuery_QuotesTheOperatorTerm(t *testing.T) {
	assert.Equal(t, "repo:acme/pay is:issue is:open", gitHubIssueSearchQuery("acme/pay", ""))
	assert.Equal(t, `repo:acme/pay is:issue is:open "refund"`, gitHubIssueSearchQuery("acme/pay", "refund"))
	assert.Equal(
		t,
		`repo:acme/pay is:issue is:open "repo:other/repo org:evil"`,
		gitHubIssueSearchQuery("acme/pay", "repo:other/repo org:evil"),
	)
}

func TestComposeImportedDescription(t *testing.T) {
	now := time.Now().UTC()
	earlier := now.Add(-2 * time.Hour)
	later := now.Add(-1 * time.Hour)

	issueComment := func(login, body string, createdAt time.Time) *github.IssueComment {
		comment := &github.IssueComment{
			Body:      github.Ptr(body),
			CreatedAt: &github.Timestamp{Time: createdAt},
		}
		if login != "" {
			comment.User = &github.User{Login: github.Ptr(login)}
		}
		return comment
	}

	t.Run("body and comments oldest first, separated by a blank line", func(t *testing.T) {
		body := "A retried refund charges the customer twice."
		comments := []*github.IssueComment{
			issueComment("ana", "Confirmed on staging.", earlier),
			issueComment("bruno", "Let's cap retries at 3.", later),
		}

		expected := fmt.Sprintf(
			"%s\n%s\n%s\nana %s\nConfirmed on staging.\n\nbruno %s\nLet's cap retries at 3.",
			body,
			importedCommentsSeparator,
			importedCommentsHeader,
			earlier.Format(time.RFC3339),
			later.Format(time.RFC3339),
		)

		assert.Equal(t, expected, composeImportedDescription(body, comments))
	})

	t.Run("no comments returns the body unchanged", func(t *testing.T) {
		body := "A retried refund charges the customer twice."

		result := composeImportedDescription(body, nil)

		assert.Equal(t, body, result)
		assert.NotContains(t, result, importedCommentsSeparator)
		assert.NotContains(t, result, importedCommentsHeader)
	})

	t.Run("empty body starts directly with the separator", func(t *testing.T) {
		comments := []*github.IssueComment{issueComment("ana", "Confirmed on staging.", earlier)}

		result := composeImportedDescription("", comments)

		expected := fmt.Sprintf(
			"%s\n%s\nana %s\nConfirmed on staging.",
			importedCommentsSeparator,
			importedCommentsHeader,
			earlier.Format(time.RFC3339),
		)
		assert.Equal(t, expected, result)
		assert.False(t, strings.HasPrefix(result, "\n"))
	})

	t.Run("comment with no user falls back to a placeholder author", func(t *testing.T) {
		comments := []*github.IssueComment{issueComment("", "Anonymous comment.", earlier)}

		result := composeImportedDescription("Body text", comments)

		assert.Contains(t, result, fmt.Sprintf("%s %s\nAnonymous comment.", importedCommentsUnknownAuthor, earlier.Format(time.RFC3339)))
	})

	t.Run("nil comments are ignored", func(t *testing.T) {
		body := "Body text"
		comments := []*github.IssueComment{nil}

		assert.Equal(t, body, composeImportedDescription(body, comments))
	})
}

func TestJiraIntakeItemSource_StaysInsideItsProject(t *testing.T) {
	source := &jiraIntakeItemSource{projectKey: "ENG", siteURL: "https://acme.atlassian.net"}

	t.Run("an origin URL of another project of the same site is not this intake's", func(t *testing.T) {
		id, ok := source.ItemIDFromOriginURL("https://acme.atlassian.net/browse/ENG-42")
		assert.True(t, ok)
		assert.Equal(t, "ENG-42", id)

		_, ok = source.ItemIDFromOriginURL("https://acme.atlassian.net/browse/OPS-42")
		assert.False(t, ok)

		_, ok = source.ItemIDFromOriginURL("https://other.atlassian.net/browse/ENG-42")
		assert.False(t, ok)
	})

	t.Run("an issue reports as owned only with a matching project key", func(t *testing.T) {
		assert.True(t, source.ownsIssue(map[string]any{"project": map[string]any{"key": "eng"}}))
		assert.False(t, source.ownsIssue(map[string]any{"project": map[string]any{"key": "OPS"}}))
		assert.False(t, source.ownsIssue(map[string]any{"project": map[string]any{}}))
		assert.False(t, source.ownsIssue(nil))
	})
}

func TestJiraIssueProjectKey(t *testing.T) {
	assert.Equal(t, "ENG", jiraIssueProjectKey("ENG-42"))
	assert.Equal(t, "ENG-SUB", jiraIssueProjectKey("ENG-SUB-42"))
	assert.Equal(t, "", jiraIssueProjectKey("ENG42"))
	assert.Equal(t, "", jiraIssueProjectKey("-42"))
}

func TestJiraIssueFromFullIssue_ReadsTheDescriptionAsText(t *testing.T) {
	issue := &jira.Issue{
		Key: "ENG-42",
		Fields: map[string]any{
			"summary": "Refund retries charge twice",
			"description": map[string]any{
				"type":    "doc",
				"version": float64(1),
				"content": []any{
					map[string]any{
						"type":    "paragraph",
						"content": []any{map[string]any{"type": "text", "text": "A retried refund charges twice."}},
					},
				},
			},
		},
	}

	assert.Equal(t, IntakeItem{
		ID:    "ENG-42",
		Key:   "ENG-42",
		Title: "Refund retries charge twice",
		Body:  "A retried refund charges twice.",
		URL:   "https://acme.atlassian.net/browse/ENG-42",
	}, jiraIssueFromFullIssue(issue, "https://acme.atlassian.net"))
}

func TestUnsupportedIntakeItemSource_DoesNotSearch(t *testing.T) {
	source := unsupportedIntakeItemSource{}
	_, err := source.Search(t.Context(), "refund", 5)
	assert.ErrorIs(t, err, errIntakeSearchUnsupported)
	_, err = source.Get(t.Context(), "1")
	assert.ErrorIs(t, err, errIntakeSearchUnsupported)
}
