package linear

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__IssueRefFromURL(t *testing.T) {
	ref, ok := IssueRefFromURL("https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails")
	assert.True(t, ok)
	assert.Equal(t, "ENG-142", ref.Identifier)
	assert.Equal(t, "linear.app", ref.Host)

	_, ok = IssueRefFromURL("https://linear.app/acme/team/ENG")
	assert.False(t, ok)
}

func Test__IssueRefFromEventData(t *testing.T) {
	ref, ok := IssueRefFromEventData(map[string]any{
		"type": IssuePayloadType,
		"data": map[string]any{
			"url": "https://linear.app/acme/issue/ENG-142/slug",
			"data": map[string]any{
				"id":         "issue-1",
				"identifier": "ENG-142",
			},
		},
	})
	assert.True(t, ok)
	assert.Equal(t, "issue-1", ref.ID)
	assert.Equal(t, "ENG-142", ref.Identifier)
	assert.Equal(t, "linear.app", ref.Host)
}
