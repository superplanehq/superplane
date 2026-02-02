package e2e

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestSecrets(t *testing.T) {
	steps := &SecretsSteps{t: t}

	t.Run("creating a new secret", func(t *testing.T) {
		steps.start()
		steps.visitSecretsPage()
		steps.clickCreateSecret()
		steps.fillSecretName("E2E Test Secret")
		steps.fillKeyValuePair(0, "API_KEY", "test-api-key-value")
		steps.submitCreateSecret()
		steps.assertSecretSavedInDB("E2E Test Secret", map[string]string{"API_KEY": "test-api-key-value"})
		steps.assertSecretVisibleInList("E2E Test Secret")
	})

	t.Run("adding a key/value pair to a secret", func(t *testing.T) {
		steps.start()
		steps.createSecretInDB("E2E Test Secret 2", map[string]string{"KEY1": "value1"})
		steps.visitSecretsPage()
		steps.clickEditSecret("E2E Test Secret 2")
		steps.clickAddPair()
		steps.fillKeyValuePair(1, "KEY2", "value2")
		steps.submitUpdateSecret()
		steps.assertSecretSavedInDB("E2E Test Secret 2", map[string]string{"KEY1": "value1", "KEY2": "value2"})
	})

	t.Run("removing a key/value pair from a secret", func(t *testing.T) {
		steps.start()
		steps.createSecretInDB("E2E Test Secret 3", map[string]string{"KEY1": "value1", "KEY2": "value2"})
		steps.visitSecretsPage()
		steps.clickEditSecret("E2E Test Secret 3")
		steps.removeKeyValuePair(0)
		steps.submitUpdateSecret()
		steps.assertSecretSavedInDB("E2E Test Secret 3", map[string]string{"KEY2": "value2"})
	})

	t.Run("edit a key/value pair from a secret", func(t *testing.T) {
		steps.start()
		steps.createSecretInDB("E2E Test Secret 4", map[string]string{"KEY1": "old-value"})
		steps.visitSecretsPage()
		steps.clickEditSecret("E2E Test Secret 4")
		steps.fillKeyValuePair(0, "KEY1", "new-value")
		steps.submitUpdateSecret()
		steps.assertSecretSavedInDB("E2E Test Secret 4", map[string]string{"KEY1": "new-value"})
	})

	t.Run("change the name of the secret", func(t *testing.T) {
		steps.start()
		steps.createSecretInDB("E2E Test Secret 5", map[string]string{"KEY1": "value1"})
		steps.visitSecretsPage()
		steps.clickEditSecret("E2E Test Secret 5")
		steps.fillSecretName("E2E Test Secret 5 Updated")
		steps.submitUpdateSecret()
		steps.assertSecretSavedInDB("E2E Test Secret 5 Updated", map[string]string{"KEY1": "value1"})
		steps.assertSecretNotVisibleInList("E2E Test Secret 5")
		steps.assertSecretVisibleInList("E2E Test Secret 5 Updated")
	})

	t.Run("deleting a secret", func(t *testing.T) {
		steps.start()
		steps.createSecretInDB("E2E Test Secret 6", map[string]string{"KEY1": "value1"})
		steps.visitSecretsPage()
		steps.assertSecretVisibleInList("E2E Test Secret 6")
		steps.clickDeleteSecret("E2E Test Secret 6")
		steps.assertSecretDeletedFromDB("E2E Test Secret 6")
		steps.assertSecretNotVisibleInList("E2E Test Secret 6")
	})
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
	createButton := q.Text("Create Secret")
	s.session.Click(createButton)
	s.session.Sleep(500)
}

func (s *SecretsSteps) fillSecretName(name string) {
	nameInput := q.Locator(`input[placeholder*="production-api-keys"]`)
	s.session.FillIn(nameInput, name)
	s.session.Sleep(300)
}

func (s *SecretsSteps) fillKeyValuePair(index int, key, value string) {
	// Find all key inputs and value textareas
	keyInputs := s.session.Page().Locator(`input[placeholder="Key"]`)
	valueTextareas := s.session.Page().Locator(`textarea[placeholder="Value"]`)

	// Fill in the key at the specified index
	keyInput := keyInputs.Nth(index)
	if err := keyInput.Fill(key); err != nil {
		s.t.Fatalf("filling key at index %d: %v", index, err)
	}
	s.session.Sleep(200)

	// Fill in the value at the specified index
	valueTextarea := valueTextareas.Nth(index)
	if err := valueTextarea.Fill(value); err != nil {
		s.t.Fatalf("filling value at index %d: %v", index, err)
	}
	s.session.Sleep(200)
}

func (s *SecretsSteps) clickAddPair() {
	addPairButton := q.Text("Add Pair")
	s.session.Click(addPairButton)
	s.session.Sleep(300)
}

func (s *SecretsSteps) removeKeyValuePair(index int) {
	// Find all delete buttons (trash icons)
	deleteButtons := s.session.Page().Locator(`button[title="Remove pair"]`)

	// Click the delete button at the specified index
	deleteButton := deleteButtons.Nth(index)
	if err := deleteButton.Click(); err != nil {
		s.t.Fatalf("clicking delete button at index %d: %v", index, err)
	}
	s.session.Sleep(300)
}

func (s *SecretsSteps) submitCreateSecret() {
	createButton := q.Text("Create Secret")
	s.session.Click(createButton)
	s.session.Sleep(1000)
}

func (s *SecretsSteps) submitUpdateSecret() {
	updateButton := q.Text("Update Secret")
	s.session.Click(updateButton)
	s.session.Sleep(1000)
}

func (s *SecretsSteps) clickEditSecret(secretName string) {
	// Find the secret card by name, then find the edit button within it
	// The secret name is in a heading, we need to find the parent card and then the edit button
	secretNameLocator := s.session.Page().Locator("text=" + secretName)
	secretCard := secretNameLocator.Locator("xpath=ancestor::div[contains(@class, 'rounded-md')]")
	editButton := secretCard.Locator(`button[title="Edit secret"]`)

	if err := editButton.Click(); err != nil {
		s.t.Fatalf("clicking edit button for secret %q: %v", secretName, err)
	}
	s.session.Sleep(500)
}

func (s *SecretsSteps) clickDeleteSecret(secretName string) {
	// Find the secret card by name, then find the delete button within it
	secretNameLocator := s.session.Page().Locator("text=" + secretName)
	secretCard := secretNameLocator.Locator("xpath=ancestor::div[contains(@class, 'rounded-md')]")
	deleteButton := secretCard.Locator(`button[title="Delete secret"]`)

	if err := deleteButton.Click(); err != nil {
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

	// Parse the data
	var secretData models.SecretData
	err = json.Unmarshal(secret.Data, &secretData)
	require.NoError(s.t, err)
	require.Equal(s.t, expectedData, secretData.Local)
}

func (s *SecretsSteps) assertSecretDeletedFromDB(name string) {
	_, err := models.FindSecretByName(models.DomainTypeOrganization, s.session.OrgID, name)
	require.Error(s.t, err)
	require.Contains(s.t, err.Error(), "record not found")
}

func (s *SecretsSteps) assertSecretVisibleInList(name string) {
	s.session.AssertText(name)
}

func (s *SecretsSteps) assertSecretNotVisibleInList(name string) {
	// Wait a bit for the UI to update
	s.session.Sleep(500)

	// Check that the text is not visible
	locator := s.session.Page().Locator("text=" + name)
	count, err := locator.Count()
	require.NoError(s.t, err)
	require.Equal(s.t, 0, count, "secret %q should not be visible in the list", name)
}

func (s *SecretsSteps) createSecretInDB(name string, data map[string]string) {
	secretData := models.SecretData{
		Local: data,
	}

	dataBytes, err := json.Marshal(secretData)
	require.NoError(s.t, err)

	secret, err := models.CreateSecret(
		name,
		"PROVIDER_LOCAL",
		s.session.Account.ID.String(),
		models.DomainTypeOrganization,
		s.session.OrgID,
		dataBytes,
	)
	require.NoError(s.t, err)
	require.NotNil(s.t, secret)
}
