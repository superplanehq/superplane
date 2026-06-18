package installation

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support"
)

func TestValidateSecretPickerParams(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	secret, err := support.CreateSecret(t, r, map[string]string{"password": "s3cret"})
	require.NoError(t, err)

	schema := []InstallParam{
		{Name: "ssh_password_secret", Type: ParamTypeSecretPicker, Required: true},
		// Non-secret_picker params must be ignored even if the value is
		// not a real secret name, since the picker validator is the only
		// place that performs an existence check.
		{Name: "region", Type: ParamTypeString, Required: false},
	}

	t.Run("accepts an existing secret name", func(t *testing.T) {
		err := ValidateSecretPickerParams(schema, map[string]string{
			"ssh_password_secret": secret.Name,
			"region":              "us-east-1",
		}, r.Organization.ID)
		require.NoError(t, err)
	})

	t.Run("rejects a missing secret", func(t *testing.T) {
		err := ValidateSecretPickerParams(schema, map[string]string{
			"ssh_password_secret": "does-not-exist",
		}, r.Organization.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("skips empty values", func(t *testing.T) {
		// Empty values are handled by ValidateInstallParams (the required
		// check); the picker validator should not double-count.
		err := ValidateSecretPickerParams(schema, map[string]string{
			"ssh_password_secret": "",
		}, r.Organization.ID)
		require.NoError(t, err)
	})

	t.Run("does not validate when no secret_picker params exist", func(t *testing.T) {
		err := ValidateSecretPickerParams(
			[]InstallParam{{Name: "region", Type: ParamTypeString}},
			map[string]string{"region": "us-east-1"},
			uuid.New(),
		)
		require.NoError(t, err)
	})
}
