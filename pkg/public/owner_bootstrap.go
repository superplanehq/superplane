package public

import (
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"gorm.io/gorm"
)

// createOwnerAccountInTransaction creates the first account, its password
// auth, organization, and membership in a single transaction. It backs both
// the interactive POST /api/v1/setup-owner handler and the .env bootstrap.
func createOwnerAccountInTransaction(
	tx *gorm.DB,
	authService authorization.Authorization,
	email, firstName, lastName, password, orgName string,
) (*models.Organization, *models.Account, *models.User, error) {
	fullName := strings.TrimSpace(firstName) + " " + strings.TrimSpace(lastName)

	account, err := models.CreateAccountInTransaction(tx, fullName, email)
	if err != nil {
		return nil, nil, nil, err
	}

	// The first account is automatically an installation admin
	if err := tx.Model(account).Update("installation_admin", true).Error; err != nil {
		return nil, nil, nil, err
	}
	account.InstallationAdmin = true

	passwordHash, err := crypto.HashPassword(password)
	if err != nil {
		return nil, nil, nil, err
	}

	if _, err := models.CreateAccountPasswordAuthInTransaction(tx, account.ID, passwordHash); err != nil {
		return nil, nil, nil, err
	}

	organization, err := models.CreateOrganizationInTransaction(tx, orgName, "")
	if err != nil {
		return nil, nil, nil, err
	}

	user, err := models.CreateUserInTransaction(tx, organization.ID, account.ID, email, fullName)
	if err != nil {
		return nil, nil, nil, err
	}

	if err := authService.SetupOrganization(tx, organization.ID.String(), user.ID.String()); err != nil {
		return nil, nil, nil, err
	}

	if err := models.SetOrganizationCreatedByAccount(tx, organization.ID, account.ID); err != nil {
		return nil, nil, nil, err
	}

	if err := models.GrantWelcomeCredit(tx, organization.ID, account.ID); err != nil {
		return nil, nil, nil, err
	}

	return organization, account, user, nil
}

// TryBootstrapOwnerFromEnv creates the owner account from SUPERPLANE_OWNER_*
// environment variables when owner setup is still required. It returns true
// when the owner was created and the /setup wizard can be skipped.
//
// When no bootstrap variables are set it returns false silently. When they
// are partially set or invalid it logs the problem and returns false so the
// server falls back to the interactive /setup wizard.
func TryBootstrapOwnerFromEnv(authService authorization.Authorization) bool {
	if !middleware.OwnerSetupEnabled() {
		return false
	}

	if !middleware.IsOwnerSetupRequired() {
		return false
	}

	cfg, hasAny, missing := config.LoadOwnerBootstrapConfig()
	if !hasAny {
		return false
	}

	if len(missing) > 0 {
		log.Warnf(
			"Owner bootstrap skipped: missing %s; complete them or finish setup at /setup",
			strings.Join(missing, ", "),
		)
		return false
	}

	if problems := config.ValidateOwnerBootstrapConfig(cfg); len(problems) > 0 {
		log.Warnf(
			"Owner bootstrap skipped: %s; fix them or finish setup at /setup",
			strings.Join(problems, "; "),
		)
		return false
	}

	var organization *models.Organization
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		organization, _, _, err = createOwnerAccountInTransaction(
			tx,
			authService,
			cfg.Email,
			cfg.FirstName,
			cfg.LastName,
			cfg.Password,
			cfg.OrgName,
		)
		return err
	})

	if err != nil {
		log.Errorf("Owner bootstrap failed, falling back to /setup wizard: %v", err)
		return false
	}

	middleware.MarkOwnerSetupCompleted()
	log.Infof("Owner account %s created from environment (organization %q); /setup wizard skipped", cfg.Email, organization.Name)
	return true
}
