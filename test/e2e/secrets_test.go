package e2e

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"testing"

	pw "github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestSecrets(t *testing.T) {
	steps := &SecretsSteps{t: t}

	t.Run("creating a new secret", func(t *testing.T) {
		steps.start()
		steps.visitSecretsPage()
		steps.givenASecretExists("E2E Test Secret", map[string]string{"API_KEY": "test-api-key-value"})
		steps.assertSecretSavedInDB("E2E Test Secret", map[string]string{"API_KEY": "test-api-key-value"})
		steps.assertSecretVisibleInList("E2E Test Secret")
	})

	// t.Run("adding a key/value pair to a secret", func(t *testing.T) {
	// 	steps.start()
	// 	steps.visitSecretsPage()
	// 	steps.givenASecretExists("E2E Test Secret 2", map[string]string{"KEY1": "value1"})
	// 	steps.clickAddKey()
	// 	steps.fillAddKeyForm("KEY2", "value2")
	// 	steps.submitAddKey()
	// 	steps.assertSecretSavedInDB("E2E Test Secret 2", map[string]string{"KEY1": "value1", "KEY2": "value2"})
	// })

	t.Run("removing a key/value pair from a secret", func(t *testing.T) {
		steps.start()
		steps.visitSecretsPage()
		steps.givenASecretExists("E2E Test Secret 3", map[string]string{"KEY1": "value1", "KEY2": "value2"})
		steps.clickRemoveKeyOnDetail(0)
		steps.assertSecretSavedInDB("E2E Test Secret 3", map[string]string{"KEY2": "value2"})
	})

	t.Run("edit a key/value pair from a secret", func(t *testing.T) {
		steps.start()
		steps.visitSecretsPage()
		steps.givenASecretExists("E2E Test Secret 4", map[string]string{"KEY1": "old-value"})
		steps.clickEditKeyOnDetail(0)
		steps.fillEditingValue("new-value")
		steps.submitEditKey()
		steps.assertSecretSavedInDB("E2E Test Secret 4", map[string]string{"KEY1": "new-value"})
	})

	// t.Run("change the name of the secret", func(t *testing.T) {
	// 	t.Skip("secret name editing is not implemented in the UI")
	// })

	// t.Run("deleting a secret", func(t *testing.T) {
	// 	steps.start()
	// 	steps.visitSecretsPage()
	// 	steps.givenASecretExists("E2E Test Secret 6", map[string]string{"KEY1": "value1"})
	// 	steps.assertSecretVisibleInList("E2E Test Secret 6")
	// 	steps.clickDeleteSecret("E2E Test Secret 6")
	// 	steps.assertSecretDeletedFromDB("E2E Test Secret 6")
	// 	steps.assertSecretNotVisibleInList("E2E Test Secret 6")
	// })
}

type SecretsSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *SecretsSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *SecretsSteps) visitSecretsPage() {
	s.session.Visit("/" + s.session.OrgID.String() + "/settings/secrets")
	s.session.Sleep(500)
}

func (s *SecretsSteps) clickCreateSecret() {
	page := s.session.Page()
	createBtn := page.GetByTestId("secrets-create-btn")
	if err := createBtn.First().Click(); err != nil {
		s.t.Fatalf("clicking create secret button: %v", err)
	}
	s.session.Sleep(500)
}

func (s *SecretsSteps) fillSecretName(name string) {
	page := s.session.Page()
	if err := page.GetByTestId("secrets-create-name").Fill(name); err != nil {
		s.t.Fatalf("filling secret name: %v", err)
	}
	s.session.Sleep(300)
}

func (s *SecretsSteps) fillKeyValuePair(index int, key, value string) {
	page := s.session.Page()
	keyInput := page.GetByTestId("secrets-create-key").Nth(index)
	if err := keyInput.Fill(key); err != nil {
		s.t.Fatalf("filling key at index %d: %v", index, err)
	}
	s.session.Sleep(200)
	valueTextarea := page.GetByTestId("secrets-create-value").Nth(index)
	if err := valueTextarea.Fill(value); err != nil {
		s.t.Fatalf("filling value at index %d: %v", index, err)
	}
	s.session.Sleep(200)
}


func (s *SecretsSteps) clickAddPair() {
	page := s.session.Page()
	if err := page.GetByTestId("secrets-create-add-pair").Click(); err != nil {
		s.t.Fatalf("clicking Add Pair: %v", err)
	}
	s.session.Sleep(300)
}

// clickAddKey clicks "Add key" on the secret detail page to show the add-key form.
func (s *SecretsSteps) clickAddKey() {
	page := s.session.Page()
	addKeyBtn := page.GetByTestId("secret-detail-add-key")
	if err := addKeyBtn.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible}); err != nil {
		s.t.Fatalf("waiting for Add key button: %v", err)
	}
	if err := addKeyBtn.Click(); err != nil {
		s.t.Fatalf("clicking Add key button: %v", err)
	}
	s.session.Sleep(300)
}

// fillAddKeyForm fills the key name and value in the add-key form on the secret detail page.
func (s *SecretsSteps) fillAddKeyForm(key, value string) {
	page := s.session.Page()
	if err := page.GetByTestId("secret-detail-add-key-name").Fill(key); err != nil {
		s.t.Fatalf("filling add-key form key: %v", err)
	}
	s.session.Sleep(200)
	if err := page.GetByTestId("secret-detail-add-value").Fill(value); err != nil {
		s.t.Fatalf("filling add-key form value: %v", err)
	}
	s.session.Sleep(200)
}

// submitAddKey clicks Save in the add-key form on the secret detail page.
func (s *SecretsSteps) submitAddKey() {
	page := s.session.Page()
	if err := page.GetByTestId("secret-detail-add-save").Click(); err != nil {
		s.t.Fatalf("clicking Save in add-key form: %v", err)
	}
	s.session.Sleep(500)
}

// clickRemoveKeyOnDetail clicks the Nth "Remove key" button on the secret detail page (removes that key immediately).
func (s *SecretsSteps) clickRemoveKeyOnDetail(index int) {
	page := s.session.Page()
	removeBtn := page.GetByTestId("secret-detail-remove-key").Nth(index)
	if err := removeBtn.Click(); err != nil {
		s.t.Fatalf("clicking Remove key at index %d: %v", index, err)
	}
	s.session.Sleep(500)
}

// clickEditKeyOnDetail clicks the Nth "Edit value" button on the secret detail page to expand the edit form.
func (s *SecretsSteps) clickEditKeyOnDetail(index int) {
	page := s.session.Page()
	editBtn := page.GetByTestId("secret-detail-edit-key").Nth(index)
	if err := editBtn.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible}); err != nil {
		s.t.Fatalf("waiting for Edit value button at index %d: %v", index, err)
	}
	if err := editBtn.Click(); err != nil {
		s.t.Fatalf("clicking Edit value at index %d: %v", index, err)
	}
	s.session.Sleep(300)
}

// fillEditingValue fills the value textarea in the edit form on the secret detail page.
func (s *SecretsSteps) fillEditingValue(value string) {
	page := s.session.Page()
	if err := page.GetByTestId("secret-detail-edit-value").Fill(value); err != nil {
		s.t.Fatalf("filling editing value: %v", err)
	}
	s.session.Sleep(200)
}

// submitEditKey clicks Save in the edit form on the secret detail page.
func (s *SecretsSteps) submitEditKey() {
	page := s.session.Page()
	if err := page.GetByTestId("secret-detail-edit-save").Click(); err != nil {
		s.t.Fatalf("clicking Save in edit form: %v", err)
	}
	s.session.Sleep(500)
}

func (s *SecretsSteps) removeKeyValuePair(index int) {
	page := s.session.Page()
	if err := page.GetByTestId("secrets-create-remove-pair").Nth(index).Click(); err != nil {
		s.t.Fatalf("clicking remove pair at index %d: %v", index, err)
	}
	s.session.Sleep(300)
}

func (s *SecretsSteps) submitCreateSecret() {
	page := s.session.Page()
	createBtn := page.GetByTestId("secrets-create-submit")
	if err := createBtn.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible}); err != nil {
		s.t.Fatalf("waiting for create secret button: %v", err)
	}
	s.session.Sleep(300)
	if err := createBtn.Click(); err != nil {
		s.t.Fatalf("clicking submit create secret button: %v", err)
	}

	// Wait for the modal to close (indicating success) or for an error message
	// The modal has class "fixed inset-0", so we wait for it to disappear
	modal := page.Locator(".fixed.inset-0")
	if err := modal.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateHidden, Timeout: pw.Float(5000)}); err != nil {
		// Modal didn't close - check for error message
		errorMsg := page.Locator("text=/Failed to create secret/")
		if count, _ := errorMsg.Count(); count > 0 {
			s.t.Fatalf("secret creation failed with error message")
		}
		s.t.Fatalf("modal did not close after submitting: %v", err)
	}

	s.session.Sleep(500)
}

func (s *SecretsSteps) submitUpdateSecret() {
	page := s.session.Page()
	s.session.Click(q.Text("Update Secret"))
	modal := page.Locator(".fixed.inset-0")
	if err := modal.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateHidden, Timeout: pw.Float(5000)}); err != nil {
		errorMsg := page.Locator("text=/Failed to update secret/")
		if count, _ := errorMsg.Count(); count > 0 {
			s.t.Fatalf("secret update failed with error message")
		}
		s.t.Fatalf("modal did not close after update: %v", err)
	}
	s.session.Sleep(500)
}

func (s *SecretsSteps) clickEditSecret(secretName string) {
	s.visitSecretsPage()
	page := s.session.Page()
	link := page.GetByTestId("secrets-secret-link").GetByText(secretName, pw.LocatorGetByTextOptions{Exact: pw.Bool(true)})
	if err := link.Click(); err != nil {
		s.t.Fatalf("clicking secret link for %q: %v", secretName, err)
	}
	s.session.Sleep(500)
}

func (s *SecretsSteps) clickDeleteSecret(secretName string) {
	s.clickEditSecret(secretName)
	page := s.session.Page()
	if err := page.GetByTestId("secret-detail-delete").Click(); err != nil {
		s.t.Fatalf("clicking delete button for secret %q: %v", secretName, err)
	}
	s.session.Sleep(500)
}

func (s *SecretsSteps) assertSecretSavedInDB(name string, expectedData map[string]string) {
	secret, err := models.FindSecretByName(models.DomainTypeOrganization, s.session.OrgID, name)
	require.NoError(s.t, err)
	require.Equal(s.t, name, secret.Name)
	require.Equal(s.t, models.DomainTypeOrganization, secret.DomainType)
	require.Equal(s.t, s.session.OrgID.String(), secret.DomainID.String())

	// Secrets created via UI are encrypted; decrypt before comparing
	encryptor := encryptorFromEnv()
	decrypted, err := encryptor.Decrypt(context.Background(), secret.Data, []byte(secret.Name))
	require.NoError(s.t, err)
	var secretData map[string]string
	err = json.Unmarshal(decrypted, &secretData)
	require.NoError(s.t, err)
	require.Equal(s.t, expectedData, secretData)
}

func (s *SecretsSteps) assertSecretDeletedFromDB(name string) {
	_, err := models.FindSecretByName(models.DomainTypeOrganization, s.session.OrgID, name)
	require.Error(s.t, err)
	require.Contains(s.t, err.Error(), "record not found")
}

func (s *SecretsSteps) assertSecretVisibleInList(name string) {
	s.visitSecretsPage()
	s.session.AssertText(name)
}

func (s *SecretsSteps) assertSecretNotVisibleInList(name string) {
	s.visitSecretsPage()
	s.session.Sleep(500)
	locator := s.session.Page().Locator("text=" + name)
	count, err := locator.Count()
	require.NoError(s.t, err)
	require.Equal(s.t, 0, count, "secret %q should not be visible in the list", name)
}

// encryptorFromEnv returns the same encryptor the app uses (from NO_ENCRYPTION / ENCRYPTION_KEY),
// used to decrypt secret data when asserting DB state after UI-created secrets.
func encryptorFromEnv() crypto.Encryptor {
	if os.Getenv("NO_ENCRYPTION") == "yes" {
		return crypto.NewNoOpEncryptor()
	}
	key := os.Getenv("ENCRYPTION_KEY")
	if key == "" {
		panic("ENCRYPTION_KEY must be set when NO_ENCRYPTION is not yes")
	}
	return crypto.NewAESGCMEncryptor([]byte(key))
}

// givenASecretExists creates a secret via the UI: Create Secret modal, name, key/value pairs, submit.
// Call after visitSecretsPage(). Keys are filled in sorted order so row indices are deterministic.
func (s *SecretsSteps) givenASecretExists(name string, data map[string]string) {
	s.clickCreateSecret()
	s.fillSecretName(name)
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		if i > 0 {
			s.clickAddPair()
		}
		s.fillKeyValuePair(i, k, data[k])
	}
	s.submitCreateSecret()
}
