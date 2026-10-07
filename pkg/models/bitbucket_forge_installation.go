package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// BitbucketForgeInstallation is one install of the SuperPlane Forge app on a
// Bitbucket workspace. The system token is ciphertext. Callers encrypt and
// decrypt it outside this package.
type BitbucketForgeInstallation struct {
	InstallationID     string `gorm:"primaryKey"`
	WorkspaceUUID      string
	WorkspaceSlug      string
	InstallerAccountID string
	APIBaseURL         string
	SystemToken        []byte
	TokenExpiresAt     time.Time
	LastDeliveryAt     time.Time
	InstalledAt        time.Time
	UninstalledAt      *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (BitbucketForgeInstallation) TableName() string {
	return "bitbucket_forge_installations"
}

// Stale reports that the cached token must not be used. A missing token, an
// uninstall, an expiry, or three hours without a Forge delivery all count.
func (row BitbucketForgeInstallation) Stale(now time.Time) bool {
	if row.UninstalledAt != nil {
		return true
	}
	if len(row.SystemToken) == 0 || row.TokenExpiresAt.IsZero() || !row.TokenExpiresAt.After(now) {
		return true
	}
	if row.LastDeliveryAt.IsZero() {
		return true
	}
	return now.Sub(row.LastDeliveryAt) > 3*time.Hour
}

// BitbucketForgeDelivery is one Forge call that carries a system token.
type BitbucketForgeDelivery struct {
	InstallationID     string
	WorkspaceUUID      string
	WorkspaceSlug      string
	InstallerAccountID string
	APIBaseURL         string
	SystemToken        []byte
	TokenExpiresAt     time.Time
	DeliveredAt        time.Time
	Uninstall          bool
}

// SaveBitbucketForgeDelivery stores a Forge delivery. A token is written only
// when it expires later than the cached token. LastDeliveryAt still moves
// forward so a short-lived token does not make a fresh installation look stale.
// Deliveries are serialized per installation, including the first delivery.
func SaveBitbucketForgeDelivery(tx *gorm.DB, delivery BitbucketForgeDelivery) (*BitbucketForgeInstallation, error) {
	delivery.InstallationID = strings.TrimSpace(delivery.InstallationID)
	if delivery.InstallationID == "" {
		return nil, errors.New("forge installation id is required")
	}
	var row *BitbucketForgeInstallation
	err := tx.Transaction(func(tx *gorm.DB) error {
		lockKey := "bitbucket-forge-delivery:" + delivery.InstallationID
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lockKey).Error; err != nil {
			return err
		}
		var err error
		row, err = saveBitbucketForgeDelivery(tx, delivery)
		return err
	})
	return row, err
}

// FindBitbucketForgeInstallation loads one Forge installation by id.
func FindBitbucketForgeInstallation(tx *gorm.DB, installationID string) (*BitbucketForgeInstallation, error) {
	var row BitbucketForgeInstallation
	err := tx.Where("installation_id = ?", strings.TrimSpace(installationID)).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// FindOrCreateBitbucketForgeIntegration binds an organization to one Forge
// installation. The integration stores no token. Runners read the cached
// system token from the installation row.
func FindOrCreateBitbucketForgeIntegration(
	tx *gorm.DB,
	organizationID uuid.UUID,
	installationID, workspaceSlug string,
) (*Integration, error) {
	installationID = strings.TrimSpace(installationID)
	if installationID == "" {
		return nil, errors.New("forge installation id is required")
	}
	workspaceSlug = strings.TrimSpace(workspaceSlug)

	var integration *Integration
	err := tx.Transaction(func(tx *gorm.DB) error {
		lockKey := fmt.Sprintf("bitbucket-forge-binding:%s:%s", organizationID, installationID)
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", lockKey).Error; err != nil {
			return err
		}

		var existing Integration
		findErr := tx.Where(
			"organization_id = ? AND app_name = ? AND metadata->>'forgeInstallationId' = ? AND deleted_at IS NULL",
			organizationID,
			ProviderBitbucket,
			installationID,
		).First(&existing).Error
		if findErr == nil {
			integration = &existing
			return nil
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}

		now := time.Now()
		name := ProviderBitbucket
		if workspaceSlug != "" {
			name = ProviderBitbucket + "-" + workspaceSlug
		}
		candidate := &Integration{
			ID:               uuid.New(),
			OrganizationID:   organizationID,
			AppName:          ProviderBitbucket,
			InstallationName: name,
			State:            IntegrationStateReady,
			Configuration: datatypes.NewJSONType(map[string]any{
				"authType":  "forgeApp",
				"workspace": workspaceSlug,
			}),
			Metadata: datatypes.NewJSONType(map[string]any{
				"hostedApp":           true,
				"authType":            "forgeApp",
				"forgeInstallationId": installationID,
				"workspace": map[string]any{
					"slug": workspaceSlug,
				},
			}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}
		if err := candidate.AssignUniqueInstallationName(tx, candidate.InstallationName); err != nil {
			return err
		}
		if err := tx.Create(candidate).Error; err != nil {
			return err
		}
		integration = candidate
		return nil
	})
	return integration, err
}

// ListActiveBitbucketForgeInstallations returns installations that still have
// a cached token and have not been uninstalled.
func ListActiveBitbucketForgeInstallations(tx *gorm.DB) ([]BitbucketForgeInstallation, error) {
	var rows []BitbucketForgeInstallation
	err := tx.Where("uninstalled_at IS NULL AND system_token IS NOT NULL").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func saveBitbucketForgeDelivery(tx *gorm.DB, delivery BitbucketForgeDelivery) (*BitbucketForgeInstallation, error) {
	installationID := delivery.InstallationID
	now := delivery.DeliveredAt
	if now.IsZero() {
		now = time.Now()
	}

	var row BitbucketForgeInstallation
	created := false
	err := tx.Where("installation_id = ?", installationID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		created = true
		row = BitbucketForgeInstallation{
			InstallationID: installationID,
			InstalledAt:    now,
			CreatedAt:      now,
		}
	} else if err != nil {
		return nil, err
	}
	// Reinstallation gets a new installation id. A delayed refresh must not
	// restore credentials for an installation that Forge already removed.
	if row.UninstalledAt != nil {
		return &row, nil
	}

	if now.After(row.LastDeliveryAt) {
		row.LastDeliveryAt = now
	}
	row.UpdatedAt = now
	if value := strings.TrimSpace(delivery.WorkspaceUUID); value != "" {
		row.WorkspaceUUID = value
	}
	if value := strings.TrimSpace(delivery.WorkspaceSlug); value != "" {
		row.WorkspaceSlug = value
	}
	if value := strings.TrimSpace(delivery.InstallerAccountID); value != "" {
		row.InstallerAccountID = value
	}
	if value := strings.TrimSpace(delivery.APIBaseURL); value != "" {
		row.APIBaseURL = value
	}
	if delivery.Uninstall {
		row.UninstalledAt = &now
		row.SystemToken = nil
		row.TokenExpiresAt = time.Time{}
	} else if delivery.TokenExpiresAt.After(row.TokenExpiresAt) && len(delivery.SystemToken) > 0 {
		row.SystemToken = delivery.SystemToken
		row.TokenExpiresAt = delivery.TokenExpiresAt
	}

	if created {
		if err := tx.Create(&row).Error; err != nil {
			return nil, err
		}
		return &row, nil
	}
	if err := tx.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}
