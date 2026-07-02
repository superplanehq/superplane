package compute

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__ListServiceAccountResources(t *testing.T) {
	t.Run("maps IAM accounts to resources keyed by email", func(t *testing.T) {
		var calledURL string
		mc := &mockFirewallClient{
			projectID: "my-project",
			getURLFunc: func(ctx context.Context, fullURL string) ([]byte, error) {
				calledURL = fullURL
				return []byte(`{"accounts":[
					{"email":"build@my-project.iam.gserviceaccount.com","displayName":"Build"},
					{"email":"run@my-project.iam.gserviceaccount.com","disabled":true}
				]}`), nil
			},
		}

		res, err := ListServiceAccountResources(context.Background(), mc, "")
		require.NoError(t, err)
		require.Len(t, res, 2)
		// Project defaulted from the client, hits the IAM host.
		assert.Contains(t, calledURL, "iam.googleapis.com")
		assert.Contains(t, calledURL, "projects/my-project/serviceAccounts")
		// ID is the email (what the firewall API expects); label shows display name.
		assert.Equal(t, "build@my-project.iam.gserviceaccount.com", res[0].ID)
		assert.Equal(t, "Build (build@my-project.iam.gserviceaccount.com)", res[0].Name)
		assert.Equal(t, ResourceTypeServiceAccount, res[0].Type)
		assert.Contains(t, res[1].Name, "disabled")
	})

	t.Run("surfaces a helpful error mentioning the permission", func(t *testing.T) {
		mc := &mockFirewallClient{
			projectID: "my-project",
			getURLFunc: func(ctx context.Context, fullURL string) ([]byte, error) {
				return nil, assertPermissionErr
			},
		}
		_, err := ListServiceAccountResources(context.Background(), mc, "my-project")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "iam.serviceAccounts.list")
	})
}

var assertPermissionErr = &gcpTestErr{"403 forbidden"}

type gcpTestErr struct{ msg string }

func (e *gcpTestErr) Error() string { return e.msg }

func Test__validateServiceAccountEmails(t *testing.T) {
	t.Run("accepts service account emails", func(t *testing.T) {
		require.NoError(t, validateServiceAccountEmails([]string{
			"build@my-project.iam.gserviceaccount.com",
			"123-compute@developer.gserviceaccount.com",
		}))
	})
	t.Run("rejects a non-service-account email", func(t *testing.T) {
		err := validateServiceAccountEmails([]string{"someone@example.com"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a service account email")
	})
	t.Run("ignores blanks", func(t *testing.T) {
		require.NoError(t, validateServiceAccountEmails([]string{"  ", ""}))
	})
}

func Test__mergeDedup(t *testing.T) {
	got := mergeDedup([]string{"a", "b"}, []string{"b", "c"})
	assert.Equal(t, []string{"a", "b", "c"}, got)
	assert.True(t, strings.Join(got, ",") == "a,b,c")
}
