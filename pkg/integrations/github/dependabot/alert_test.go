package dependabot

import (
	"strconv"
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/factory"
)

func lodashAlert(number int, manifest string) *github.DependabotAlert {
	return &github.DependabotAlert{
		Number:  github.Ptr(number),
		State:   github.Ptr("open"),
		HTMLURL: github.Ptr("https://github.com/acme/payments/security/dependabot/" + strconv.Itoa(number)),
		Dependency: &github.Dependency{
			Package:      &github.VulnerabilityPackage{Name: github.Ptr("lodash"), Ecosystem: github.Ptr("npm")},
			ManifestPath: github.Ptr(manifest),
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
}

func TestTaskCopyFromAlerts(t *testing.T) {
	ref, ok := PackageRefFromAlert("acme/payments", lodashAlert(7, "package.json"))
	require.True(t, ok)

	copy := TaskCopyFromAlerts(ref, []*github.DependabotAlert{
		lodashAlert(7, "package.json"),
		lodashAlert(8, "package-lock.json"),
	})

	assert.Equal(t, "Fix Dependabot alerts for lodash (npm)", copy.Title)
	assert.Contains(t, copy.Description, "Fix every open Dependabot alert for lodash (npm).")
	assert.Contains(t, copy.Description, "## Alerts")
	assert.Contains(t, copy.Description, "### #7 Prototype pollution in lodash\nSeverity: high\nManifest: package.json\nVulnerable versions: < 4.17.21\nPatched version: 4.17.21\nhttps://github.com/acme/payments/security/dependabot/7")
	assert.Contains(t, copy.Description, "### #8 Prototype pollution in lodash\nSeverity: high\nManifest: package-lock.json")
	assert.NotContains(t, copy.Description, "Relationship:")
}

func TestAlertSection_NamesTheRelationship(t *testing.T) {
	section := AlertSection(map[string]any{
		"number":   float64(7),
		"html_url": "https://github.com/acme/payments/security/dependabot/7",
		"dependency": map[string]any{
			"manifest_path": "package.json",
			"relationship":  "transitive",
		},
		"security_advisory": map[string]any{"summary": "Prototype pollution", "severity": "high"},
	})

	assert.Contains(t, section, "### #7 Prototype pollution")
	assert.Contains(t, section, "Relationship: transitive\nhttps://github.com/acme/payments/security/dependabot/7")

	unknown := AlertSection(map[string]any{"dependency": map[string]any{"relationship": "unknown"}})
	assert.NotContains(t, unknown, "Relationship:")
}

func TestMergeAlertSection(t *testing.T) {
	first := "Fix every open Dependabot alert for lodash (npm).\n\n## Alerts\n\n### #7 Prototype pollution\nhttps://github.com/acme/payments/security/dependabot/7"
	second := "### #8 Prototype pollution\nhttps://github.com/acme/payments/security/dependabot/8"

	t.Run("appends a new alert", func(t *testing.T) {
		merged := MergeAlertSection(first, second)
		assert.Equal(t, first+"\n\n"+second, merged)
	})

	t.Run("keeps the alert before the instructions", func(t *testing.T) {
		withInstructions := first + "\n\n" + factory.InstructionsHeading + "\nUpdate the direct dependency."
		merged := MergeAlertSection(withInstructions, second)
		assert.Equal(t, first+"\n\n"+second+"\n\n"+factory.InstructionsHeading+"\nUpdate the direct dependency.", merged)
	})

	t.Run("does not add the same alert twice", func(t *testing.T) {
		merged := MergeAlertSection(first, second)
		assert.Equal(t, merged, MergeAlertSection(merged, second))
	})

	t.Run("ignores an empty section", func(t *testing.T) {
		assert.Equal(t, first, MergeAlertSection(first, "  \n"))
	})
}

func TestPackageRefOriginURL_RoundTrips(t *testing.T) {
	ref := PackageRef{Repository: "acme/payments", Ecosystem: "npm", Name: "@babel/core"}

	originURL := ref.OriginURL()
	assert.Equal(t, "https://github.com/acme/payments/security/dependabot?q=is%3Aopen+package%3A%40babel%2Fcore+ecosystem%3Anpm", originURL)
	assert.Equal(t, "Dependabot: @babel/core", ref.OriginLabel())

	parsed, ok := PackageRefFromURL(originURL)
	require.True(t, ok)
	assert.Equal(t, ref, parsed)

	_, ok = PackageRefFromURL("https://github.com/acme/payments/security/dependabot/7")
	assert.False(t, ok)
	_, ok = PackageRefFromURL("https://github.com/acme/payments/issues/7")
	assert.False(t, ok)
}

func TestPackageRefFromEventData(t *testing.T) {
	ref, ok := PackageRefFromEventData(map[string]any{
		"type": AlertPayloadType,
		"data": map[string]any{
			"action": "created",
			"alert": map[string]any{
				"html_url": "https://github.com/Acme/Payments/security/dependabot/7",
				"dependency": map[string]any{
					"package": map[string]any{"name": "lodash", "ecosystem": "NPM"},
				},
			},
		},
	})
	require.True(t, ok)
	assert.Equal(t, PackageRef{Repository: "Acme/Payments", Ecosystem: "npm", Name: "lodash"}, ref)
	assert.True(t, ref.Matches(PackageRef{Repository: "acme/payments", Ecosystem: "npm", Name: "lodash"}))
	assert.False(t, ref.Matches(PackageRef{Repository: "acme/payments", Ecosystem: "npm", Name: "Lodash"}))

	_, ok = PackageRefFromEventData(map[string]any{
		"type": AlertPayloadType,
		"data": map[string]any{"alert": map[string]any{"html_url": "https://github.com/acme/payments/security/dependabot/7"}},
	})
	assert.False(t, ok)
}
