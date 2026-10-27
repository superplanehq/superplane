package models

import "gorm.io/gorm"

type EncryptedStorageLocation struct {
	Name     string
	Format   EncryptedStorageFormat
	ReadPage func(*gorm.DB, string, int) ([]EncryptedStorageRow, error)
	Replace  func(*gorm.DB, EncryptedStorageRow, []byte) (bool, error)
}

type EncryptedStorageFormat uint8

const (
	EncryptedStorageBytes EncryptedStorageFormat = iota
	EncryptedStorageIntegrationJSON
)

type EncryptedStorageRow struct {
	Cursor         string
	Value          []byte
	AssociatedData string
	AppName        string
}

// ApplicationEncryptedStorage is the reviewed inventory of ciphertext stores.
func ApplicationEncryptedStorage() []EncryptedStorageLocation {
	return []EncryptedStorageLocation{
		{
			Name:     "secrets.data",
			Format:   EncryptedStorageBytes,
			ReadPage: ReadSecretDataPage,
			Replace:  ReplaceSecretData,
		},
		{
			Name:     "app_installation_secrets.value",
			Format:   EncryptedStorageBytes,
			ReadPage: ReadAppInstallationSecretValuePage,
			Replace:  ReplaceAppInstallationSecretValue,
		},
		{
			Name:     "app_installations.configuration",
			Format:   EncryptedStorageIntegrationJSON,
			ReadPage: ReadAppInstallationConfigurationPage,
			Replace:  ReplaceAppInstallationConfiguration,
		},
		{
			Name:     "webhooks.secret",
			Format:   EncryptedStorageBytes,
			ReadPage: ReadWebhookSecretPage,
			Replace:  ReplaceWebhookSecret,
		},
		{
			Name:     "runner_tasks.payload_ciphertext",
			Format:   EncryptedStorageBytes,
			ReadPage: ReadRunnerTaskPayloadPage,
			Replace:  ReplaceRunnerTaskPayload,
		},
		{
			Name:     "factory_agent_resource_secrets.value",
			Format:   EncryptedStorageBytes,
			ReadPage: ReadFactoryAgentResourceSecretValuePage,
			Replace:  ReplaceFactoryAgentResourceSecretValue,
		},
		{
			Name:     "hosted_llm_providers.api_key",
			Format:   EncryptedStorageBytes,
			ReadPage: ReadHostedLLMProviderAPIKeyPage,
			Replace:  ReplaceHostedLLMProviderAPIKey,
		},
		{
			Name:     "hosted_llm_providers.management_key",
			Format:   EncryptedStorageBytes,
			ReadPage: ReadHostedLLMProviderManagementKeyPage,
			Replace:  ReplaceHostedLLMProviderManagementKey,
		},
		{
			Name:     "installation_github_apps.encrypted_private_key",
			Format:   EncryptedStorageBytes,
			ReadPage: ReadInstallationGitHubAppPrivateKeyPage,
			Replace:  ReplaceInstallationGitHubAppPrivateKey,
		},
		{
			Name:     "installation_github_apps.encrypted_webhook_secret",
			Format:   EncryptedStorageBytes,
			ReadPage: ReadInstallationGitHubAppWebhookSecretPage,
			Replace:  ReplaceInstallationGitHubAppWebhookSecret,
		},
		{
			Name:     "installation_github_apps.encrypted_client_secret",
			Format:   EncryptedStorageBytes,
			ReadPage: ReadInstallationGitHubAppClientSecretPage,
			Replace:  ReplaceInstallationGitHubAppClientSecret,
		},
		{
			Name:     "account_github_authorizations.encrypted_access_token",
			Format:   EncryptedStorageBytes,
			ReadPage: ReadAccountGitHubAuthorizationAccessTokenPage,
			Replace:  ReplaceAccountGitHubAuthorizationAccessToken,
		},
		{
			Name:     "account_github_authorizations.encrypted_refresh_token",
			Format:   EncryptedStorageBytes,
			ReadPage: ReadAccountGitHubAuthorizationRefreshTokenPage,
			Replace:  ReplaceAccountGitHubAuthorizationRefreshToken,
		},
		{
			Name:     "installation_licenses.encrypted_license",
			Format:   EncryptedStorageBytes,
			ReadPage: ReadInstallationLicensePage,
			Replace:  ReplaceInstallationLicense,
		},
		{
			Name:     "email_settings.smtp_password",
			Format:   EncryptedStorageBytes,
			ReadPage: ReadEmailSettingsSMTPPasswordPage,
			Replace:  ReplaceEmailSettingsSMTPPassword,
		},
		{
			Name:     "bitbucket_forge_installations.system_token",
			Format:   EncryptedStorageBytes,
			ReadPage: ReadBitbucketForgeInstallationSystemTokenPage,
			Replace:  ReplaceBitbucketForgeInstallationSystemToken,
		},
		{
			Name:     "sentry_app_install_grants.access_token",
			Format:   EncryptedStorageBytes,
			ReadPage: ReadSentryAppInstallGrantAccessTokenPage,
			Replace:  ReplaceSentryAppInstallGrantAccessToken,
		},
		{
			Name:     "sentry_app_install_grants.refresh_token",
			Format:   EncryptedStorageBytes,
			ReadPage: ReadSentryAppInstallGrantRefreshTokenPage,
			Replace:  ReplaceSentryAppInstallGrantRefreshToken,
		},
	}
}

func ReadSecretDataPage(tx *gorm.DB, cursor string, limit int) ([]EncryptedStorageRow, error) {
	query := tx.Table("secrets").
		Select("id::text, data, name, ''").
		Where("data IS NOT NULL")
	if cursor != "" {
		query = query.Where("id > CAST(? AS uuid)", cursor)
	}
	return scanEncryptedStorageRows(query.Order("id").Limit(limit), limit)
}

func ReplaceSecretData(tx *gorm.DB, row EncryptedStorageRow, replacement []byte) (bool, error) {
	result := tx.Exec("UPDATE secrets SET data = ? WHERE id = CAST(? AS uuid) AND data = ?", replacement, row.Cursor, row.Value)
	return result.RowsAffected == 1, result.Error
}

func ReadAppInstallationSecretValuePage(tx *gorm.DB, cursor string, limit int) ([]EncryptedStorageRow, error) {
	query := tx.Table("app_installation_secrets").
		Select("id::text, value, installation_id::text, ''").
		Where("value IS NOT NULL")
	if cursor != "" {
		query = query.Where("id > CAST(? AS uuid)", cursor)
	}
	return scanEncryptedStorageRows(query.Order("id").Limit(limit), limit)
}

func ReplaceAppInstallationSecretValue(tx *gorm.DB, row EncryptedStorageRow, replacement []byte) (bool, error) {
	result := tx.Exec("UPDATE app_installation_secrets SET value = ? WHERE id = CAST(? AS uuid) AND value = ?", replacement, row.Cursor, row.Value)
	return result.RowsAffected == 1, result.Error
}

func ReadAppInstallationConfigurationPage(tx *gorm.DB, cursor string, limit int) ([]EncryptedStorageRow, error) {
	query := tx.Table("app_installations").
		Select("id::text, configuration, id::text, app_name").
		Where("configuration IS NOT NULL")
	if cursor != "" {
		query = query.Where("id > CAST(? AS uuid)", cursor)
	}
	return scanEncryptedStorageRows(query.Order("id").Limit(limit), limit)
}

func ReplaceAppInstallationConfiguration(tx *gorm.DB, row EncryptedStorageRow, replacement []byte) (bool, error) {
	result := tx.Exec("UPDATE app_installations SET configuration = CAST(? AS jsonb) WHERE id = CAST(? AS uuid) AND configuration = CAST(? AS jsonb)", string(replacement), row.Cursor, string(row.Value))
	return result.RowsAffected == 1, result.Error
}

func ReadWebhookSecretPage(tx *gorm.DB, cursor string, limit int) ([]EncryptedStorageRow, error) {
	query := tx.Table("webhooks").
		Select("id::text, secret, id::text, ''").
		Where("secret IS NOT NULL")
	if cursor != "" {
		query = query.Where("id > CAST(? AS uuid)", cursor)
	}
	return scanEncryptedStorageRows(query.Order("id").Limit(limit), limit)
}

func ReplaceWebhookSecret(tx *gorm.DB, row EncryptedStorageRow, replacement []byte) (bool, error) {
	result := tx.Exec("UPDATE webhooks SET secret = ? WHERE id = CAST(? AS uuid) AND secret = ?", replacement, row.Cursor, row.Value)
	return result.RowsAffected == 1, result.Error
}

func ReadRunnerTaskPayloadPage(tx *gorm.DB, cursor string, limit int) ([]EncryptedStorageRow, error) {
	query := tx.Table("runner_tasks").
		Select("id::text, payload_ciphertext, id::text, ''").
		Where("payload_ciphertext IS NOT NULL")
	if cursor != "" {
		query = query.Where("id > CAST(? AS uuid)", cursor)
	}
	return scanEncryptedStorageRows(query.Order("id").Limit(limit), limit)
}

func ReplaceRunnerTaskPayload(tx *gorm.DB, row EncryptedStorageRow, replacement []byte) (bool, error) {
	result := tx.Exec("UPDATE runner_tasks SET payload_ciphertext = ? WHERE id = CAST(? AS uuid) AND payload_ciphertext = ?", replacement, row.Cursor, row.Value)
	return result.RowsAffected == 1, result.Error
}

func ReadFactoryAgentResourceSecretValuePage(tx *gorm.DB, cursor string, limit int) ([]EncryptedStorageRow, error) {
	query := tx.Table("factory_agent_resource_secrets").
		Select("id::text, value, resource_id::text, ''").
		Where("value IS NOT NULL")
	if cursor != "" {
		query = query.Where("id > CAST(? AS uuid)", cursor)
	}
	return scanEncryptedStorageRows(query.Order("id").Limit(limit), limit)
}

func ReplaceFactoryAgentResourceSecretValue(tx *gorm.DB, row EncryptedStorageRow, replacement []byte) (bool, error) {
	result := tx.Exec("UPDATE factory_agent_resource_secrets SET value = ? WHERE id = CAST(? AS uuid) AND value = ?", replacement, row.Cursor, row.Value)
	return result.RowsAffected == 1, result.Error
}

func ReadHostedLLMProviderAPIKeyPage(tx *gorm.DB, cursor string, limit int) ([]EncryptedStorageRow, error) {
	query := tx.Table("hosted_llm_providers").
		Select("id::text, api_key, 'hosted_llm_key:' || provider, ''").
		Where("api_key IS NOT NULL")
	if cursor != "" {
		query = query.Where("id > CAST(? AS uuid)", cursor)
	}
	return scanEncryptedStorageRows(query.Order("id").Limit(limit), limit)
}

func ReplaceHostedLLMProviderAPIKey(tx *gorm.DB, row EncryptedStorageRow, replacement []byte) (bool, error) {
	result := tx.Exec("UPDATE hosted_llm_providers SET api_key = ? WHERE id = CAST(? AS uuid) AND api_key = ?", replacement, row.Cursor, row.Value)
	return result.RowsAffected == 1, result.Error
}

func ReadHostedLLMProviderManagementKeyPage(tx *gorm.DB, cursor string, limit int) ([]EncryptedStorageRow, error) {
	query := tx.Table("hosted_llm_providers").
		Select("id::text, management_key, 'hosted_llm_mgmt_key:' || provider, ''").
		Where("management_key IS NOT NULL")
	if cursor != "" {
		query = query.Where("id > CAST(? AS uuid)", cursor)
	}
	return scanEncryptedStorageRows(query.Order("id").Limit(limit), limit)
}

func ReplaceHostedLLMProviderManagementKey(tx *gorm.DB, row EncryptedStorageRow, replacement []byte) (bool, error) {
	result := tx.Exec("UPDATE hosted_llm_providers SET management_key = ? WHERE id = CAST(? AS uuid) AND management_key = ?", replacement, row.Cursor, row.Value)
	return result.RowsAffected == 1, result.Error
}

func ReadInstallationGitHubAppPrivateKeyPage(tx *gorm.DB, cursor string, limit int) ([]EncryptedStorageRow, error) {
	query := tx.Table("installation_github_apps").
		Select("id::text, encrypted_private_key, 'installation_github_app_private_key', ''").
		Where("encrypted_private_key IS NOT NULL")
	if cursor != "" {
		query = query.Where("id > CAST(? AS integer)", cursor)
	}
	return scanEncryptedStorageRows(query.Order("id").Limit(limit), limit)
}

func ReplaceInstallationGitHubAppPrivateKey(tx *gorm.DB, row EncryptedStorageRow, replacement []byte) (bool, error) {
	result := tx.Exec("UPDATE installation_github_apps SET encrypted_private_key = ? WHERE id = CAST(? AS integer) AND encrypted_private_key = ?", replacement, row.Cursor, row.Value)
	return result.RowsAffected == 1, result.Error
}

func ReadInstallationGitHubAppWebhookSecretPage(tx *gorm.DB, cursor string, limit int) ([]EncryptedStorageRow, error) {
	query := tx.Table("installation_github_apps").
		Select("id::text, encrypted_webhook_secret, 'installation_github_app_webhook_secret', ''").
		Where("encrypted_webhook_secret IS NOT NULL")
	if cursor != "" {
		query = query.Where("id > CAST(? AS integer)", cursor)
	}
	return scanEncryptedStorageRows(query.Order("id").Limit(limit), limit)
}

func ReplaceInstallationGitHubAppWebhookSecret(tx *gorm.DB, row EncryptedStorageRow, replacement []byte) (bool, error) {
	result := tx.Exec("UPDATE installation_github_apps SET encrypted_webhook_secret = ? WHERE id = CAST(? AS integer) AND encrypted_webhook_secret = ?", replacement, row.Cursor, row.Value)
	return result.RowsAffected == 1, result.Error
}

func ReadInstallationGitHubAppClientSecretPage(tx *gorm.DB, cursor string, limit int) ([]EncryptedStorageRow, error) {
	query := tx.Table("installation_github_apps").
		Select("id::text, encrypted_client_secret, 'installation_github_app_client_secret', ''").
		Where("length(encrypted_client_secret) > 0")
	if cursor != "" {
		query = query.Where("id > CAST(? AS integer)", cursor)
	}
	return scanEncryptedStorageRows(query.Order("id").Limit(limit), limit)
}

func ReplaceInstallationGitHubAppClientSecret(tx *gorm.DB, row EncryptedStorageRow, replacement []byte) (bool, error) {
	result := tx.Exec("UPDATE installation_github_apps SET encrypted_client_secret = ? WHERE id = CAST(? AS integer) AND encrypted_client_secret = ?", replacement, row.Cursor, row.Value)
	return result.RowsAffected == 1, result.Error
}

func ReadAccountGitHubAuthorizationAccessTokenPage(tx *gorm.DB, cursor string, limit int) ([]EncryptedStorageRow, error) {
	query := tx.Table("account_github_authorizations").
		Select("account_id::text, encrypted_access_token, 'account_github_authorization:' || account_id::text, ''").
		Where("length(encrypted_access_token) > 0")
	if cursor != "" {
		query = query.Where("account_id > CAST(? AS uuid)", cursor)
	}
	return scanEncryptedStorageRows(query.Order("account_id").Limit(limit), limit)
}

func ReplaceAccountGitHubAuthorizationAccessToken(tx *gorm.DB, row EncryptedStorageRow, replacement []byte) (bool, error) {
	result := tx.Exec("UPDATE account_github_authorizations SET encrypted_access_token = ? WHERE account_id = CAST(? AS uuid) AND encrypted_access_token = ?", replacement, row.Cursor, row.Value)
	return result.RowsAffected == 1, result.Error
}

func ReadAccountGitHubAuthorizationRefreshTokenPage(tx *gorm.DB, cursor string, limit int) ([]EncryptedStorageRow, error) {
	query := tx.Table("account_github_authorizations").
		Select("account_id::text, encrypted_refresh_token, 'account_github_authorization:' || account_id::text, ''").
		Where("length(encrypted_refresh_token) > 0")
	if cursor != "" {
		query = query.Where("account_id > CAST(? AS uuid)", cursor)
	}
	return scanEncryptedStorageRows(query.Order("account_id").Limit(limit), limit)
}

func ReplaceAccountGitHubAuthorizationRefreshToken(tx *gorm.DB, row EncryptedStorageRow, replacement []byte) (bool, error) {
	result := tx.Exec("UPDATE account_github_authorizations SET encrypted_refresh_token = ? WHERE account_id = CAST(? AS uuid) AND encrypted_refresh_token = ?", replacement, row.Cursor, row.Value)
	return result.RowsAffected == 1, result.Error
}

func ReadInstallationLicensePage(tx *gorm.DB, cursor string, limit int) ([]EncryptedStorageRow, error) {
	query := tx.Table("installation_licenses").
		Select("id::text, encrypted_license, 'superplane:installation-license:v1', ''").
		Where("encrypted_license IS NOT NULL")
	if cursor != "" {
		query = query.Where("id > CAST(? AS integer)", cursor)
	}
	return scanEncryptedStorageRows(query.Order("id").Limit(limit), limit)
}

func ReplaceInstallationLicense(tx *gorm.DB, row EncryptedStorageRow, replacement []byte) (bool, error) {
	result := tx.Exec("UPDATE installation_licenses SET encrypted_license = ? WHERE id = CAST(? AS integer) AND encrypted_license = ?", replacement, row.Cursor, row.Value)
	return result.RowsAffected == 1, result.Error
}

func ReadEmailSettingsSMTPPasswordPage(tx *gorm.DB, cursor string, limit int) ([]EncryptedStorageRow, error) {
	query := tx.Table("email_settings").
		Select("id::text, smtp_password, 'smtp_password', ''").
		Where("smtp_password IS NOT NULL")
	if cursor != "" {
		query = query.Where("id > CAST(? AS uuid)", cursor)
	}
	return scanEncryptedStorageRows(query.Order("id").Limit(limit), limit)
}

func ReplaceEmailSettingsSMTPPassword(tx *gorm.DB, row EncryptedStorageRow, replacement []byte) (bool, error) {
	result := tx.Exec("UPDATE email_settings SET smtp_password = ? WHERE id = CAST(? AS uuid) AND smtp_password = ?", replacement, row.Cursor, row.Value)
	return result.RowsAffected == 1, result.Error
}

func ReadBitbucketForgeInstallationSystemTokenPage(tx *gorm.DB, cursor string, limit int) ([]EncryptedStorageRow, error) {
	query := tx.Table("bitbucket_forge_installations").
		Select("installation_id::text, system_token, installation_id, ''").
		Where("system_token IS NOT NULL")
	if cursor != "" {
		query = query.Where("installation_id > ?", cursor)
	}
	return scanEncryptedStorageRows(query.Order("installation_id").Limit(limit), limit)
}

func ReplaceBitbucketForgeInstallationSystemToken(tx *gorm.DB, row EncryptedStorageRow, replacement []byte) (bool, error) {
	result := tx.Exec("UPDATE bitbucket_forge_installations SET system_token = ? WHERE installation_id = ? AND system_token = ?", replacement, row.Cursor, row.Value)
	return result.RowsAffected == 1, result.Error
}

func ReadSentryAppInstallGrantAccessTokenPage(tx *gorm.DB, cursor string, limit int) ([]EncryptedStorageRow, error) {
	query := tx.Table("sentry_app_install_grants").
		Select("installation_uuid::text, access_token, installation_uuid, ''").
		Where("access_token IS NOT NULL")
	if cursor != "" {
		query = query.Where("installation_uuid > ?", cursor)
	}
	return scanEncryptedStorageRows(query.Order("installation_uuid").Limit(limit), limit)
}

func ReplaceSentryAppInstallGrantAccessToken(tx *gorm.DB, row EncryptedStorageRow, replacement []byte) (bool, error) {
	result := tx.Exec("UPDATE sentry_app_install_grants SET access_token = ? WHERE installation_uuid = ? AND access_token = ?", replacement, row.Cursor, row.Value)
	return result.RowsAffected == 1, result.Error
}

func ReadSentryAppInstallGrantRefreshTokenPage(tx *gorm.DB, cursor string, limit int) ([]EncryptedStorageRow, error) {
	query := tx.Table("sentry_app_install_grants").
		Select("installation_uuid::text, refresh_token, installation_uuid, ''").
		Where("refresh_token IS NOT NULL")
	if cursor != "" {
		query = query.Where("installation_uuid > ?", cursor)
	}
	return scanEncryptedStorageRows(query.Order("installation_uuid").Limit(limit), limit)
}

func ReplaceSentryAppInstallGrantRefreshToken(tx *gorm.DB, row EncryptedStorageRow, replacement []byte) (bool, error) {
	result := tx.Exec("UPDATE sentry_app_install_grants SET refresh_token = ? WHERE installation_uuid = ? AND refresh_token = ?", replacement, row.Cursor, row.Value)
	return result.RowsAffected == 1, result.Error
}

func scanEncryptedStorageRows(query *gorm.DB, limit int) ([]EncryptedStorageRow, error) {
	rows, err := query.Rows()
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
