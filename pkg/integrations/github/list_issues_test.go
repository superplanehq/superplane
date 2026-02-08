package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListIssues_Name(t *testing.T) {
	component := &ListIssues{}
	assert.Equal(t, "github.listIssues", component.Name())
}

func TestListIssues_Label(t *testing.T) {
	component := &ListIssues{}
	assert.Equal(t, "Get Repository Issues", component.Label())
}

func TestListIssues_Description(t *testing.T) {
	component := &ListIssues{}
	assert.Equal(t, "List issues from a GitHub repository with search and filter options", component.Description())
}

func TestListIssues_Configuration(t *testing.T) {
	component := &ListIssues{}
	config := component.Configuration()

	// Should have all the filter fields
	assert.True(t, len(config) >= 10)

	// Check required fields
	fieldNames := make(map[string]bool)
	for _, field := range config {
		fieldNames[field.Name] = true
	}

	assert.True(t, fieldNames["repository"])
	assert.True(t, fieldNames["searchQuery"])
	assert.True(t, fieldNames["state"])
	assert.True(t, fieldNames["labels"])
	assert.True(t, fieldNames["assignee"])
	assert.True(t, fieldNames["creator"])
	assert.True(t, fieldNames["sort"])
	assert.True(t, fieldNames["direction"])
	assert.True(t, fieldNames["perPage"])
	assert.True(t, fieldNames["page"])
}

func TestListIssues_OutputChannels(t *testing.T) {
	component := &ListIssues{}
	channels := component.OutputChannels(nil)

	assert.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}

func TestListIssues_ExampleOutput(t *testing.T) {
	component := &ListIssues{}
	example := component.ExampleOutput()

	assert.NotNil(t, example)
	assert.Contains(t, example, "issues")
	assert.Contains(t, example, "total_count")
}

func TestListIssues_Icon(t *testing.T) {
	component := &ListIssues{}
	assert.Equal(t, "github", component.Icon())
}

func TestListIssues_Color(t *testing.T) {
	component := &ListIssues{}
	assert.Equal(t, "gray", component.Color())
}

func TestListIssues_Actions(t *testing.T) {
	component := &ListIssues{}
	actions := component.Actions()
	assert.Empty(t, actions)
}

func TestListIssuesConfiguration_Defaults(t *testing.T) {
	component := &ListIssues{}
	config := component.Configuration()

	// Check default values
	for _, field := range config {
		switch field.Name {
		case "state":
			assert.Equal(t, "open", field.Default)
		case "sort":
			assert.Equal(t, "created", field.Default)
		case "direction":
			assert.Equal(t, "desc", field.Default)
		case "perPage":
			assert.Equal(t, "30", field.Default)
		case "page":
			assert.Equal(t, "1", field.Default)
		}
	}
}
