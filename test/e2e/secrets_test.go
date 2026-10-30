package e2e

import (
	"context"
	"encoding/json"
	"testing"

	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/e2e/session"
	"gorm.io/gorm"
)

func TestSecrets(t *testing.T) {
	t.Run("creating editing and deleting a secret preserves its values", func(t *testing.T) {
		steps := &secretsSteps{t: t}
		steps.start()
		steps.createSecret("E2E Test Secret")
		steps.assertSecretSavedInDB("E2E Test Secret", map[string]string{
			"API_KEY": "initial-api-key",
			"TOKEN":   "unchanged-token",
		})
		steps.editValue("API_KEY", "updated-api-key")
		steps.assertSecretSavedInDB("E2E Test Secret", map[string]string{
			"API_KEY": "updated-api-key",
			"TOKEN":   "unchanged-token",
		})
		steps.deleteSecret()
		steps.assertSecretDeletedFromDB("E2E Test Secret")
		steps.assertSecretNotVisibleInList("E2E Test Secret")
	})
}

type secretsSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *secretsSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	require.NoError(s.t, models.EnableExperimentalFeature(s.session.OrgID, features.FeatureFactories))
	s.session.Login()
	s.session.Visit("/" + s.session.OrgID.String() + "/settings/secrets")
}

func (s *secretsSteps) createSecret(name string) {
	page := s.session.Page()
	require.NoError(s.t, page.GetByTestId("secrets-create-btn").First().Click())
	require.NoError(s.t, page.GetByTestId("secrets-create-name").Fill(name))
	s.fillKeyValuePair(0, "API_KEY", "initial-api-key")
	require.NoError(s.t, page.GetByTestId("secrets-create-add-pair").Click())
	s.fillKeyValuePair(1, "TOKEN", "unchanged-token")
	require.NoError(s.t, page.GetByTestId("secrets-create-submit").Click())
	require.NoError(s.t, page.GetByTestId("secrets-create-form").WaitFor(pw.LocatorWaitForOptions{
		State: pw.WaitForSelectorStateHidden,
	}))
}

func (s *secretsSteps) fillKeyValuePair(index int, key, value string) {
	page := s.session.Page()
	require.NoError(s.t, page.GetByTestId("secrets-create-key").Nth(index).Fill(key))
	require.NoError(s.t, page.GetByTestId("secrets-create-value").Nth(index).Fill(value))
}

func (s *secretsSteps) editValue(keyName, value string) {
	page := s.session.Page()
	editButtons := page.GetByTestId("secret-detail-edit-key")
	require.NoError(s.t, editButtons.First().WaitFor(pw.LocatorWaitForOptions{
		State: pw.WaitForSelectorStateVisible,
	}))
	count, err := editButtons.Count()
	require.NoError(s.t, err)
	for index := 0; index < count; index++ {
		require.NoError(s.t, editButtons.Nth(index).Click())
		key, err := page.GetByTestId("secret-detail-edit-key-name").InputValue()
		require.NoError(s.t, err)
		if key != keyName {
			require.NoError(s.t, page.GetByRole("button", pw.PageGetByRoleOptions{Name: "Cancel", Exact: pw.Bool(true)}).Click())
			continue
		}
		require.NoError(s.t, page.GetByTestId("secret-detail-edit-value").Fill(value))
		require.NoError(s.t, page.GetByTestId("secret-detail-edit-save").Click())
		require.NoError(s.t, page.GetByTestId("secret-detail-edit-value").WaitFor(pw.LocatorWaitForOptions{
			State: pw.WaitForSelectorStateHidden,
		}))
		return
	}
	s.t.Fatalf("secret key %q was not available to edit", keyName)
}

func (s *secretsSteps) deleteSecret() {
	page := s.session.Page()
	require.NoError(s.t, page.GetByTestId("secret-detail-delete").Click())
	// Returning to the list follows the completed delete request.
	require.NoError(s.t, page.GetByTestId("secrets-create-btn").First().WaitFor(pw.LocatorWaitForOptions{
		State: pw.WaitForSelectorStateVisible,
	}))
}

func (s *secretsSteps) assertSecretSavedInDB(name string, expectedData map[string]string) {
	secret, err := models.FindSecretByName(models.DomainTypeOrganization, s.session.OrgID, name)
	require.NoError(s.t, err)
	require.Equal(s.t, name, secret.Name)
	require.Equal(s.t, models.DomainTypeOrganization, secret.DomainType)
	require.Equal(s.t, s.session.OrgID, secret.DomainID)

	encryptor, err := crypto.ProcessEncryptor()
	require.NoError(s.t, err)
	decrypted, err := encryptor.Decrypt(context.Background(), secret.Data, []byte(secret.Name))
	require.NoError(s.t, err)
	var secretData map[string]string
	require.NoError(s.t, json.Unmarshal(decrypted, &secretData))
	require.Equal(s.t, expectedData, secretData)
}

func (s *secretsSteps) assertSecretDeletedFromDB(name string) {
	_, err := models.FindSecretByName(models.DomainTypeOrganization, s.session.OrgID, name)
	require.ErrorIs(s.t, err, gorm.ErrRecordNotFound)
}

func (s *secretsSteps) assertSecretNotVisibleInList(name string) {
	link := s.session.Page().GetByTestId("secrets-secret-link").GetByText(name, pw.LocatorGetByTextOptions{Exact: pw.Bool(true)})
	require.NoError(s.t, link.WaitFor(pw.LocatorWaitForOptions{
		State: pw.WaitForSelectorStateHidden,
	}))
}
