package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueURLFromEventData_NormalizesOwnerAndRepositoryCase(t *testing.T) {
	issueURL, ok := IssueURLFromEventData(map[string]any{
		"type": IssueEventPayloadType,
		"data": map[string]any{
			"issue": map[string]any{
				"html_url": "https://GitHub.com/Acme/Payments/issues/12/",
			},
		},
	})
	require.True(t, ok)
	assert.Equal(t, "https://github.com/acme/payments/issues/12", issueURL)
}
