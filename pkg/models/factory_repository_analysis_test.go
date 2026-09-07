package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildFactoryRepositoryAnalysis(t *testing.T) {
	npm := "npm ci"
	cargo := "cargo fetch"
	document := `{
	  "schemaVersion": "1",
	  "projects": [
	    {
	      "path": "web",
	      "languages": [{"name": "typescript"}],
	      "frameworks": [{"name": "react"}],
	      "packageManagers": [{"name": "npm"}],
	      "requirements": [],
	      "preparation": [
	        {"name": "Install dependencies", "run": "` + npm + `", "directory": ".", "interpretations": []},
	        {"name": "Fetch Rust", "run": "` + cargo + `", "directory": ".", "interpretations": []}
	      ],
	      "commands": [
	        {"name": "Test", "run": "npm test", "directory": ".", "interpretations": [{"capability": "test.run"}]}
	      ],
	      "ambiguities": [],
	      "conflicts": []
	    }
	  ]
	}`

	analysis, err := BuildFactoryRepositoryAnalysis(document, "abc123")
	require.NoError(t, err)
	assert.Equal(t, FactoryRepositoryAnalysisStatusReady, analysis.Status)
	assert.Equal(t, []string{"typescript"}, analysis.Languages)
	assert.Equal(t, []FactoryRepositorySetupStep{{
		Name: "Install dependencies", Command: npm, Directory: "web",
	}}, analysis.SetupSteps)
	assert.Contains(t, analysis.Context, `"capability":"test.run"`)
	assert.NotContains(t, analysis.Context, cargo)
}

func TestBuildFactoryRepositoryAnalysisRejectsUnknownSchema(t *testing.T) {
	_, err := BuildFactoryRepositoryAnalysis(`{"schemaVersion":"2","projects":[]}`, "")
	assert.ErrorIs(t, err, ErrFactoryRepositoryAnalysisInvalid)
}

func TestBuildFactoryRepositoryAnalysisSkipsUnsafeDirectory(t *testing.T) {
	document := `{
	  "schemaVersion": "1",
	  "projects": [{
	    "path": ".",
	    "languages": [],
	    "frameworks": [],
	    "packageManagers": [],
	    "requirements": [],
	    "preparation": [{"name":"Install","run":"npm ci","directory":"../outside","interpretations":[]}],
	    "commands": [],
	    "ambiguities": [],
	    "conflicts": []
	  }]
	}`

	analysis, err := BuildFactoryRepositoryAnalysis(document, "")
	require.NoError(t, err)
	assert.Empty(t, analysis.SetupSteps)
}
