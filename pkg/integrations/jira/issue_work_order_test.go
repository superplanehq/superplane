package jira

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test__jiraIssueLockKey_includesSite(t *testing.T) {
	factoryID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	acme := jiraIssueLockKey(factoryID, IssueRef{Host: "acme.atlassian.net", Key: "ENG-5"})
	other := jiraIssueLockKey(factoryID, IssueRef{Host: "other.atlassian.net", Key: "ENG-5"})
	sameKeyOtherIssue := jiraIssueLockKey(factoryID, IssueRef{Host: "acme.atlassian.net", Key: "ENG-50"})

	assert.NotEqual(t, acme, other)
	assert.NotEqual(t, acme, sameKeyOtherIssue)
	assert.Equal(t, acme, jiraIssueLockKey(factoryID, IssueRef{Host: "acme.atlassian.net", Key: "ENG-5"}))
}
