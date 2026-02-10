package snyk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnykIntegration(t *testing.T) {
	snyk := &Snyk{}

	assert.Equal(t, "snyk", snyk.Name())
	assert.Equal(t, "Snyk", snyk.Label())
	assert.Equal(t, "shield", snyk.Icon())
	assert.Equal(t, "Security workflow integration with Snyk", snyk.Description())

	components := snyk.Components()
	assert.Len(t, components, 1)
	assert.Equal(t, "snyk.ignoreIssue", components[0].Name())

	triggers := snyk.Triggers()
	assert.Len(t, triggers, 1)
	assert.Equal(t, "snyk.onNewIssueDetected", triggers[0].Name())
}

func TestSnykConfiguration(t *testing.T) {
	snyk := &Snyk{}
	configFields := snyk.Configuration()

	assert.Len(t, configFields, 2)

	fieldNames := make(map[string]bool)
	for _, field := range configFields {
		fieldNames[field.Name] = true
	}

	assert.True(t, fieldNames["apiToken"])
	assert.True(t, fieldNames["organizationId"])
}

func Test__Snyk__CompareWebhookConfig(t *testing.T) {
	s := &Snyk{}

	testCases := []struct {
		name        string
		configA     any
		configB     any
		expectEqual bool
		expectError bool
	}{
		{
			name: "identical configurations",
			configA: WebhookConfiguration{
				EventType: "issue.detected",
				OrgID:     "org-123",
				ProjectID: "project-123",
			},
			configB: WebhookConfiguration{
				EventType: "issue.detected",
				OrgID:     "org-123",
				ProjectID: "project-123",
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "different event types",
			configA: WebhookConfiguration{
				EventType: "issue.detected",
				OrgID:     "org-123",
			},
			configB: WebhookConfiguration{
				EventType: "issue.resolved",
				OrgID:     "org-123",
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "different org IDs",
			configA: WebhookConfiguration{
				EventType: "issue.detected",
				OrgID:     "org-123",
			},
			configB: WebhookConfiguration{
				EventType: "issue.detected",
				OrgID:     "org-456",
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "different project IDs",
			configA: WebhookConfiguration{
				EventType: "issue.detected",
				OrgID:     "org-123",
				ProjectID: "project-123",
			},
			configB: WebhookConfiguration{
				EventType: "issue.detected",
				OrgID:     "org-123",
				ProjectID: "project-456",
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "all fields different",
			configA: WebhookConfiguration{
				EventType: "issue.detected",
				OrgID:     "org-123",
				ProjectID: "project-123",
			},
			configB: WebhookConfiguration{
				EventType: "issue.resolved",
				OrgID:     "org-456",
				ProjectID: "project-456",
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "comparing map representations",
			configA: map[string]any{
				"eventType": "issue.detected",
				"orgId":     "org-123",
				"projectId": "project-123",
			},
			configB: map[string]any{
				"eventType": "issue.detected",
				"orgId":     "org-123",
				"projectId": "project-123",
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name:    "invalid first configuration",
			configA: "invalid",
			configB: WebhookConfiguration{
				EventType: "issue.detected",
				OrgID:     "org-123",
			},
			expectEqual: false,
			expectError: true,
		},
		{
			name: "invalid second configuration",
			configA: WebhookConfiguration{
				EventType: "issue.detected",
				OrgID:     "org-123",
			},
			configB:     "invalid",
			expectEqual: false,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			equal, err := s.CompareWebhookConfig(tc.configA, tc.configB)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.expectEqual, equal)
		})
	}
}
