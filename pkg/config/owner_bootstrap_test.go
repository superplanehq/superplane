package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func setOwnerBootstrapEnv(t *testing.T, email, firstName, lastName, password, orgName string) {
	t.Helper()
	t.Setenv(OwnerBootstrapEmailEnv, email)
	t.Setenv(OwnerBootstrapPasswordEnv, password)
	t.Setenv(OwnerBootstrapFirstNameEnv, firstName)
	t.Setenv(OwnerBootstrapLastNameEnv, lastName)
	t.Setenv(OwnerBootstrapOrgNameEnv, orgName)
}

func TestLoadOwnerBootstrapConfigUnset(t *testing.T) {
	setOwnerBootstrapEnv(t, "", "", "", "", "")
	_, hasAny, missing := LoadOwnerBootstrapConfig()
	assert.False(t, hasAny)
	assert.Len(t, missing, 4)
}

func TestLoadOwnerBootstrapConfigComplete(t *testing.T) {
	setOwnerBootstrapEnv(t, "owner@example.com", "Owner", "User", "Password1", "")
	cfg, hasAny, missing := LoadOwnerBootstrapConfig()
	assert.True(t, hasAny)
	assert.Empty(t, missing)
	assert.Equal(t, "owner@example.com", cfg.Email)
	assert.Equal(t, DefaultOwnerBootstrapOrgName, cfg.OrgName)
	assert.Empty(t, ValidateOwnerBootstrapConfig(cfg))
}

func TestLoadOwnerBootstrapConfigPartial(t *testing.T) {
	setOwnerBootstrapEnv(t, "owner@example.com", "", "", "", "")
	_, hasAny, missing := LoadOwnerBootstrapConfig()
	assert.True(t, hasAny)
	assert.Contains(t, missing, OwnerBootstrapPasswordEnv)
	assert.Contains(t, missing, OwnerBootstrapFirstNameEnv)
	assert.Contains(t, missing, OwnerBootstrapLastNameEnv)
}

func TestValidateOwnerBootstrapConfig(t *testing.T) {
	setOwnerBootstrapEnv(t, "not-an-email", "Owner", "User", "weak", "")
	cfg, _, _ := LoadOwnerBootstrapConfig()
	problems := ValidateOwnerBootstrapConfig(cfg)
	assert.Len(t, problems, 2)

	setOwnerBootstrapEnv(t, "owner@example.com", "Owner", "User", "Password1", "Acme")
	cfg, _, _ = LoadOwnerBootstrapConfig()
	assert.Empty(t, ValidateOwnerBootstrapConfig(cfg))
	assert.Equal(t, "Acme", cfg.OrgName)
}

func TestIsOwnerBootstrapPasswordValid(t *testing.T) {
	assert.False(t, IsOwnerBootstrapPasswordValid("short1A"))
	assert.False(t, IsOwnerBootstrapPasswordValid("nonumbercap"))
	assert.False(t, IsOwnerBootstrapPasswordValid("nonumberAA"))
	assert.False(t, IsOwnerBootstrapPasswordValid("nocapital1"))
	assert.True(t, IsOwnerBootstrapPasswordValid("Password1"))
}
