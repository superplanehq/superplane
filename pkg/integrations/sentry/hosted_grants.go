package sentry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

// hostedInstallGrantTTL is how long an install grant waits for the browser
// callback that claims it.
const hostedInstallGrantTTL = 20 * time.Minute

// hostedInstallGrantStore keeps public Sentry app install grants until the
// setup callback claims one. Sentry sends the installation webhook and the
// browser redirect at the same time, and each one can reach a different
// SuperPlane process, so every process reads the grants from one place.
type hostedInstallGrantStore interface {
	Remember(install hostedSentryInstall) error
	Take(installationUUID, code string) (*hostedSentryInstall, error)
	Forget(installationUUID string) error
}

var hostedInstallGrants hostedInstallGrantStore = databaseHostedInstallGrants{}

type databaseHostedInstallGrants struct{}

func (databaseHostedInstallGrants) Remember(install hostedSentryInstall) error {
	installationUUID := strings.TrimSpace(install.InstallationUUID)
	code := strings.TrimSpace(install.Code)
	if installationUUID == "" || code == "" {
		return nil
	}

	encryptor, err := hostedInstallGrantEncryptor()
	if err != nil {
		return err
	}

	accessToken, err := sealHostedGrantToken(encryptor, installationUUID, install.AccessToken)
	if err != nil {
		return err
	}
	refreshToken, err := sealHostedGrantToken(encryptor, installationUUID, install.RefreshToken)
	if err != nil {
		return err
	}

	expiresAt := install.ExpiresAt
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(hostedInstallGrantTTL)
	}

	organizationSlug := ""
	if install.Organization != nil {
		organizationSlug = install.Organization.Slug
	}

	tx := database.Conn()
	if err := models.DeleteExpiredSentryAppInstallGrants(tx, time.Now()); err != nil {
		return err
	}

	return models.UpsertSentryAppInstallGrant(tx, models.SentryAppInstallGrant{
		InstallationUUID: installationUUID,
		CodeDigest:       hostedInstallCodeDigest(code),
		OrganizationSlug: organizationSlug,
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		TokenExpiresAt:   install.TokenExpiresAt,
		ExpiresAt:        expiresAt,
	})
}

func (databaseHostedInstallGrants) Take(installationUUID, code string) (*hostedSentryInstall, error) {
	installationUUID = strings.TrimSpace(installationUUID)
	code = strings.TrimSpace(code)
	if installationUUID == "" || code == "" {
		return nil, nil
	}

	encryptor, err := hostedInstallGrantEncryptor()
	if err != nil {
		return nil, err
	}

	grant, err := models.TakeSentryAppInstallGrant(
		database.Conn(),
		installationUUID,
		hostedInstallCodeDigest(code),
		time.Now(),
	)
	if err != nil || grant == nil {
		return nil, err
	}

	accessToken, err := openHostedGrantToken(encryptor, installationUUID, grant.AccessToken)
	if err != nil {
		return nil, err
	}
	refreshToken, err := openHostedGrantToken(encryptor, installationUUID, grant.RefreshToken)
	if err != nil {
		return nil, err
	}

	install := &hostedSentryInstall{
		InstallationUUID: grant.InstallationUUID,
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		TokenExpiresAt:   grant.TokenExpiresAt,
		Code:             code,
		ExpiresAt:        grant.ExpiresAt,
	}
	if grant.OrganizationSlug != "" {
		install.Organization = &OrganizationSummary{Slug: grant.OrganizationSlug}
	}
	return install, nil
}

func (databaseHostedInstallGrants) Forget(installationUUID string) error {
	return models.DeleteSentryAppInstallGrant(database.Conn(), installationUUID)
}

// hostedInstallCodeDigest hides the one-time install code, which stays a
// secret even though the grant it unlocks is short lived.
func hostedInstallCodeDigest(code string) string {
	digest := sha256.Sum256([]byte(code))
	return hex.EncodeToString(digest[:])
}

func hostedInstallGrantEncryptor() (crypto.Encryptor, error) {
	if hostedInstallEncryptor == nil {
		return nil, fmt.Errorf("Sentry install grants need an encryptor")
	}
	return hostedInstallEncryptor, nil
}

func sealHostedGrantToken(encryptor crypto.Encryptor, installationUUID, token string) ([]byte, error) {
	if strings.TrimSpace(token) == "" {
		return nil, nil
	}
	return encryptor.Encrypt(context.Background(), []byte(token), []byte(installationUUID))
}

func openHostedGrantToken(encryptor crypto.Encryptor, installationUUID string, sealed []byte) (string, error) {
	if len(sealed) == 0 {
		return "", nil
	}
	plain, err := encryptor.Decrypt(context.Background(), sealed, []byte(installationUUID))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(plain)), nil
}
