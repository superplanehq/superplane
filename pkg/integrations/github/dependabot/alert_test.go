package dependabot

import (
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskCopyFromAlert(t *testing.T) {
	alert := &github.DependabotAlert{
		HTMLURL: github.Ptr("https://github.com/acme/payments/security/dependabot/7"),
		Dependency: &github.Dependency{
			Package:      &github.VulnerabilityPackage{Name: github.Ptr("lodash"), Ecosystem: github.Ptr("npm")},
			ManifestPath: github.Ptr("package.json"),
		},
		SecurityAdvisory: &github.DependabotSecurityAdvisory{
			Summary:  github.Ptr("Prototype pollution in lodash"),
			Severity: github.Ptr("high"),
		},
		SecurityVulnerability: &github.AdvisoryVulnerability{
			VulnerableVersionRange: github.Ptr("< 4.17.21"),
			FirstPatchedVersion:    &github.FirstPatchedVersion{Identifier: github.Ptr("4.17.21")},
		},
	}

	copy := TaskCopyFromAlert(alert)
	assert.Equal(t, "Bump lodash in package.json", copy.Title)
	assert.Contains(t, copy.Description, "Prototype pollution in lodash")
	assert.Contains(t, copy.Description, "Package: lodash (npm)")
	assert.Contains(t, copy.Description, "Manifest: package.json")
	assert.Contains(t, copy.Description, "Vulnerable versions: < 4.17.21")
	assert.Contains(t, copy.Description, "Patched version: 4.17.21")
	assert.Contains(t, copy.Description, "Severity: high")
	assert.Contains(t, copy.Description, alert.GetHTMLURL())
}

func TestAlertRefFromURL(t *testing.T) {
	ref, ok := AlertRefFromURL("https://github.com/acme/payments/security/dependabot/7")
	require.True(t, ok)
	assert.Equal(t, AlertRef{Repository: "acme/payments", Number: 7}, ref)

	_, ok = AlertRefFromURL("https://github.com/acme/payments/issues/7")
	assert.False(t, ok)
}

func TestAlertRefFromEventData(t *testing.T) {
	ref, ok := AlertRefFromEventData(map[string]any{
		"type": AlertPayloadType,
		"data": map[string]any{
			"action": "created",
			"alert": map[string]any{
				"html_url": "https://github.com/acme/payments/security/dependabot/7",
			},
		},
	})
	require.True(t, ok)
	assert.Equal(t, 7, ref.Number)
	assert.Equal(t, "acme/payments", ref.Repository)
}
