package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ApplicationEncryptionMigration struct {
	ID              uuid.UUID `gorm:"primaryKey"`
	TargetKeyID     uuid.UUID
	State           string
	RegistryVersion int
	StorageLocation string
	Cursor          string
	Pass            int
	ErrorMessage    string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	CompletedAt     *time.Time
}

type EncryptedStorageLocation struct {
	Name          string
	Table         string
	KeyColumn     string
	KeyType       string
	ValueColumn   string
	AADExpression string
	AppExpression string
	Encoding      string
}

type EncryptedStorageRow struct {
	Cursor         string
	Value          []byte
	AssociatedData string
	AppName        string
}

// ApplicationEncryptedStorage is the reviewed inventory of ciphertext stores.
// SQL identifiers and expressions must remain literals in this registry.
func ApplicationEncryptedStorage() []EncryptedStorageLocation {
	return []EncryptedStorageLocation{
		{"secrets.data", "secrets", "id", "uuid", "data", "name", "''", "bytes"},
		{"app_installation_secrets.value", "app_installation_secrets", "id", "uuid", "value", "installation_id::text", "''", "bytes"},
		{"app_installations.configuration", "app_installations", "id", "uuid", "configuration", "id::text", "app_name", "integration_json"},
		{"webhooks.secret", "webhooks", "id", "uuid", "secret", "id::text", "''", "bytes"},
		{"runner_tasks.payload_ciphertext", "runner_tasks", "id", "uuid", "payload_ciphertext", "id::text", "''", "bytes"},
		{"factory_agent_resource_secrets.value", "factory_agent_resource_secrets", "id", "uuid", "value", "resource_id::text", "''", "bytes"},
		{"hosted_llm_providers.api_key", "hosted_llm_providers", "id", "uuid", "api_key", "'hosted_llm_key:' || provider", "''", "bytes"},
		{"hosted_llm_providers.management_key", "hosted_llm_providers", "id", "uuid", "management_key", "'hosted_llm_mgmt_key:' || provider", "''", "bytes"},
		{"installation_github_apps.encrypted_private_key", "installation_github_apps", "id", "integer", "encrypted_private_key", "'installation_github_app_private_key'", "''", "bytes"},
		{"installation_github_apps.encrypted_webhook_secret", "installation_github_apps", "id", "integer", "encrypted_webhook_secret", "'installation_github_app_webhook_secret'", "''", "bytes"},
		{"installation_licenses.encrypted_license", "installation_licenses", "id", "integer", "encrypted_license", "'superplane:installation-license:v1'", "''", "bytes"},
		{"email_settings.smtp_password", "email_settings", "id", "uuid", "smtp_password", "'smtp_password'", "''", "bytes"},
		{"bitbucket_forge_installations.system_token", "bitbucket_forge_installations", "installation_id", "text", "system_token", "installation_id", "''", "bytes"},
		{"sentry_app_install_grants.access_token", "sentry_app_install_grants", "installation_uuid", "text", "access_token", "installation_uuid", "''", "bytes"},
		{"sentry_app_install_grants.refresh_token", "sentry_app_install_grants", "installation_uuid", "text", "refresh_token", "installation_uuid", "''", "bytes"},
		{"account_providers.access_token", "account_providers", "id", "uuid", "access_token", "COALESCE(email, '')", "''", "base64"},
	}
}

func (ApplicationEncryptionMigration) TableName() string {
	return "application_encryption_migrations"
}

func CreateApplicationEncryptionMigration(tx *gorm.DB, migration *ApplicationEncryptionMigration) error {
	return tx.Create(migration).Error
}

func LoadApplicationEncryptionMigration(tx *gorm.DB, id uuid.UUID) (*ApplicationEncryptionMigration, error) {
	var migration ApplicationEncryptionMigration
	if err := tx.First(&migration, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &migration, nil
}

func LockNextApplicationEncryptionMigration(tx *gorm.DB) (*ApplicationEncryptionMigration, error) {
	var migration ApplicationEncryptionMigration
	err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("state IN ?", []string{"pending", "running"}).
		Order("created_at, id").First(&migration).Error
	if err != nil {
		return nil, err
	}
	return &migration, nil
}

func SaveApplicationEncryptionMigration(tx *gorm.DB, migration *ApplicationEncryptionMigration) error {
	migration.UpdatedAt = time.Now()
	return tx.Save(migration).Error
}

func ReadEncryptedStoragePage(tx *gorm.DB, location EncryptedStorageLocation, cursor string, limit int) ([]EncryptedStorageRow, error) {
	app := location.AppExpression
	if app == "" {
		app = "''"
	}
	query := fmt.Sprintf("SELECT %s::text, %s, %s, %s FROM %s WHERE %s IS NOT NULL", location.KeyColumn, location.ValueColumn, location.AADExpression, app, location.Table, location.ValueColumn)
	args := []any{}
	if cursor != "" {
		query += fmt.Sprintf(" AND %s > CAST(? AS %s)", location.KeyColumn, location.KeyType)
		args = append(args, cursor)
	}
	query += fmt.Sprintf(" ORDER BY %s LIMIT ?", location.KeyColumn)
	args = append(args, limit)
	rows, err := tx.Raw(query, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	page := make([]EncryptedStorageRow, 0, limit)
	for rows.Next() {
		var row EncryptedStorageRow
		if err := rows.Scan(&row.Cursor, &row.Value, &row.AssociatedData, &row.AppName); err != nil {
			return nil, err
		}
		page = append(page, row)
	}
	return page, rows.Err()
}

func ReplaceEncryptedStorageValue(tx *gorm.DB, location EncryptedStorageLocation, row EncryptedStorageRow, replacement []byte) (bool, error) {
	oldValue, newValue := any(row.Value), any(replacement)
	oldExpr, newExpr := "?", "?"
	if location.Encoding == "integration_json" {
		oldExpr, newExpr = "CAST(? AS jsonb)", "CAST(? AS jsonb)"
		oldValue, newValue = string(row.Value), string(replacement)
	} else if location.Encoding == "base64" {
		oldValue, newValue = string(row.Value), string(replacement)
	}
	query := fmt.Sprintf("UPDATE %s SET %s = %s WHERE %s = CAST(? AS %s) AND %s = %s", location.Table, location.ValueColumn, newExpr, location.KeyColumn, location.KeyType, location.ValueColumn, oldExpr)
	result := tx.Exec(query, newValue, row.Cursor, oldValue)
	return result.RowsAffected == 1, result.Error
}
