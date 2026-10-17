package config

import (
	"os"
	"regexp"
	"strings"
)

const (
	OwnerBootstrapEmailEnv     = "SUPERPLANE_OWNER_EMAIL"
	OwnerBootstrapPasswordEnv  = "SUPERPLANE_OWNER_PASSWORD"
	OwnerBootstrapFirstNameEnv = "SUPERPLANE_OWNER_FIRST_NAME"
	OwnerBootstrapLastNameEnv  = "SUPERPLANE_OWNER_LAST_NAME"
	OwnerBootstrapOrgNameEnv   = "SUPERPLANE_OWNER_ORG_NAME"
)

// DefaultOwnerBootstrapOrgName matches the organization name created by the
// interactive owner setup flow (pkg/public/setup_owner.go).
const DefaultOwnerBootstrapOrgName = "Demo"

var ownerBootstrapEmailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// OwnerBootstrapConfig holds the owner account fields that can be supplied
// through the environment to skip the interactive /setup wizard.
type OwnerBootstrapConfig struct {
	Email     string
	FirstName string
	LastName  string
	Password  string
	OrgName   string
}

// LoadOwnerBootstrapConfig reads the SUPERPLANE_OWNER_* environment variables.
// It returns the config, whether any bootstrap variable was set, and the list
// of required variables that are missing.
func LoadOwnerBootstrapConfig() (OwnerBootstrapConfig, bool, []string) {
	cfg := OwnerBootstrapConfig{
		Email:     strings.TrimSpace(os.Getenv(OwnerBootstrapEmailEnv)),
		FirstName: strings.TrimSpace(os.Getenv(OwnerBootstrapFirstNameEnv)),
		LastName:  strings.TrimSpace(os.Getenv(OwnerBootstrapLastNameEnv)),
		Password:  os.Getenv(OwnerBootstrapPasswordEnv),
		OrgName:   strings.TrimSpace(os.Getenv(OwnerBootstrapOrgNameEnv)),
	}

	if cfg.OrgName == "" {
		cfg.OrgName = DefaultOwnerBootstrapOrgName
	}

	hasAny := cfg.Email != "" ||
		cfg.FirstName != "" ||
		cfg.LastName != "" ||
		cfg.Password != "" ||
		strings.TrimSpace(os.Getenv(OwnerBootstrapOrgNameEnv)) != ""

	var missing []string
	if cfg.Email == "" {
		missing = append(missing, OwnerBootstrapEmailEnv)
	}
	if cfg.FirstName == "" {
		missing = append(missing, OwnerBootstrapFirstNameEnv)
	}
	if cfg.LastName == "" {
		missing = append(missing, OwnerBootstrapLastNameEnv)
	}
	if cfg.Password == "" {
		missing = append(missing, OwnerBootstrapPasswordEnv)
	}

	return cfg, hasAny, missing
}

// ValidateOwnerBootstrapConfig checks the bootstrap config with the same
// rules as the interactive owner setup form. It returns a list of human
// readable problems; an empty list means the config is valid.
func ValidateOwnerBootstrapConfig(cfg OwnerBootstrapConfig) []string {
	var problems []string

	if cfg.Email == "" {
		problems = append(problems, OwnerBootstrapEmailEnv+" is required")
	} else if !ownerBootstrapEmailPattern.MatchString(cfg.Email) {
		problems = append(problems, OwnerBootstrapEmailEnv+" must be a valid email address")
	}

	if strings.TrimSpace(cfg.FirstName) == "" {
		problems = append(problems, OwnerBootstrapFirstNameEnv+" is required")
	}

	if strings.TrimSpace(cfg.LastName) == "" {
		problems = append(problems, OwnerBootstrapLastNameEnv+" is required")
	}

	if cfg.Password == "" {
		problems = append(problems, OwnerBootstrapPasswordEnv+" is required")
	} else if !IsOwnerBootstrapPasswordValid(cfg.Password) {
		problems = append(problems, OwnerBootstrapPasswordEnv+" must be 8+ characters with at least 1 number and 1 capital letter")
	}

	if strings.TrimSpace(cfg.OrgName) == "" {
		problems = append(problems, OwnerBootstrapOrgNameEnv+" must not be blank")
	}

	return problems
}

// IsOwnerBootstrapPasswordValid mirrors the frontend owner setup validation
// (web_src/src/pages/auth/OwnerSetup.tsx).
func IsOwnerBootstrapPasswordValid(password string) bool {
	if len(password) < 8 {
		return false
	}

	var hasDigit, hasUpper bool
	for _, r := range password {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		}
		if hasDigit && hasUpper {
			return true
		}
	}

	return false
}
