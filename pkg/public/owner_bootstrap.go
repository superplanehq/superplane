package public

import (
	"errors"
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

// errOwnerSetupAlreadyCompleted is returned when another API pod (or a
// concurrent interactive setup request) finished owner setup while this
// process waited on the owner setup lock.
var errOwnerSetupAlreadyCompleted = errors.New("owner setup already completed")

// lockOwnerSetup serializes the first-owner check and creation across API
// pods. The advisory lock is held until the transaction commits or rolls
// back, so a second pod blocks here until the first pod finishes and then
// sees the new user instead of failing on the accounts.email unique
// constraint.
func lockOwnerSetup(tx *gorm.DB) error {
	return tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "owner-setup").Error
}

func ownerSetupRequiredInTransaction(tx *gorm.DB) (bool, error) {
	var count int64
	if err := tx.Model(&models.User{}).Limit(1).Count(&count).Error; err != nil {
		return false, err
	}

	return count == 0, nil
}

// createOwnerAccountInTransaction creates the first account, its password
// auth, organization, and membership in a single transaction. It backs both
// the interactive POST /api/v1/setup-owner handler and the .env bootstrap.
//
// The transaction takes the owner setup lock and re-checks whether setup is
// still required before inserting anything. It returns
// errOwnerSetupAlreadyCompleted when another process finished setup first.
func createOwnerAccountInTransaction(
	tx *gorm.DB,
	authService authorization.Authorization,
	email, firstName, lastName, password, orgName string,
) (*models.Organization, *models.Account, *models.User, error) {
	if err := lockOwnerSetup(tx); err != nil {
		return nil, nil, nil, err
	}

	setupRequired, err := ownerSetupRequiredInTransaction(tx)
	if err != nil {
		return nil, nil, nil, err
	}

	if !setupRequired {
		return nil, nil, nil, errOwnerSetupAlreadyCompleted
	}

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

	if errors.Is(err, errOwnerSetupAlreadyCompleted) {
		// Another API pod created the owner while this one waited on the
		// setup lock. Sync the cached setup state so this pod stops
		// redirecting requests to /setup.
		middleware.MarkOwnerSetupCompleted()
		log.Info("Owner setup already completed by another API pod; /setup wizard skipped")
		return true
	}

	if err != nil {
		// Another pod may have completed setup after this transaction
		// failed. Re-read the setup state from the database instead of
		// trusting the local cache before falling back to the wizard.
		if !middleware.RefreshOwnerSetupState() {
			log.Info("Owner setup already completed by another API pod; /setup wizard skipped")
			return true
		}

		log.Errorf("Owner bootstrap failed, falling back to /setup wizard: %v", err)
		return false
	}

	middleware.MarkOwnerSetupCompleted()
	log.Infof("Owner account %s created from environment (organization %q); /setup wizard skipped", cfg.Email, organization.Name)
	return true
}
