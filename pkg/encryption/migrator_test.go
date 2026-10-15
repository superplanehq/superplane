package encryption

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	_ "github.com/superplanehq/superplane/pkg/integrations/openai"
	_ "github.com/superplanehq/superplane/pkg/integrations/openrouter"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

func TestMigratorConvertsLegacyCiphertextAndUnblocksRotation(t *testing.T) {
	for _, source := range []struct {
		name      string
		encryptor crypto.Encryptor
	}{
		{"aes-gcm", crypto.NewAESGCMEncryptor(bytes.Repeat([]byte{4}, 32))},
		{"plaintext", crypto.NewNoOpEncryptor()},
	} {
		t.Run(source.name, func(t *testing.T) {
			testMigratorConvertsLegacyValues(t, source.encryptor)
		})
	}
}

func testMigratorConvertsLegacyValues(t *testing.T, legacy crypto.Encryptor) {
	ctx := t.Context()
	db := database.DB(context.Background())
	require.NoError(t, database.VerifyTestDatabase(db))
	clean := func() {
		require.NoError(t, db.Exec("DELETE FROM application_encryption_migrations").Error)
		require.NoError(t, db.Exec("DELETE FROM application_encryption_keys").Error)
	}
	clean()
	t.Cleanup(clean)

	master := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{3}, 32))
	masters, err := crypto.ParseLocalMasterKeys("local-v1:" + master)
	require.NoError(t, err)
	store := NewStore(masters, "local-v1")
	require.NoError(t, store.Initialize(ctx))
	encryptor := crypto.NewVersionedEncryptor(store, legacy)
	table := "encryption_migration_test_" + uuid.NewString()[:8]
	require.NoError(t, db.Exec(fmt.Sprintf("CREATE TABLE %s (id uuid PRIMARY KEY, secret bytea NOT NULL, configuration jsonb NOT NULL, app_name text NOT NULL, access_token text NOT NULL)", table)).Error)
	t.Cleanup(func() { require.NoError(t, db.Exec("DROP TABLE "+table).Error) })
	location := testEncryptedStorageLocation(table, "secret", "'smtp_password'", "''", models.EncryptedStorageBytes)
	configurationLocation := testEncryptedStorageLocation(table, "configuration", "id::text", "app_name", models.EncryptedStorageIntegrationJSON)
	base64Location := testEncryptedStorageLocation(table, "access_token", "id::text", "''", models.EncryptedStorageBase64)
	integrations, err := registry.NewRegistry(encryptor, registry.HTTPOptions{})
	require.NoError(t, err)
	migrator := NewMigrator(encryptor, integrations, []models.EncryptedStorageLocation{location, configurationLocation, base64Location})

	ciphertext, err := legacy.Encrypt(ctx, []byte("smtp secret"), []byte("smtp_password"))
	require.NoError(t, err)
	id := uuid.New()
	apiKey, err := legacy.Encrypt(ctx, []byte("api secret"), []byte(id.String()))
	require.NoError(t, err)
	adminKey, err := legacy.Encrypt(ctx, []byte("admin secret"), []byte(id.String()))
	require.NoError(t, err)
	configuration, err := json.Marshal(map[string]any{
		"apiKey":      base64.StdEncoding.EncodeToString(apiKey),
		"adminKey":    base64.StdEncoding.EncodeToString(adminKey),
		"baseURL":     "https://example.com",
		"largeNumber": int64(9007199254740993),
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec("INSERT INTO "+table+" (id, secret, configuration, app_name, access_token) VALUES (?, ?, ?, 'openai', ?)", id, ciphertext, string(configuration), base64.StdEncoding.EncodeToString(apiKey)).Error)
	openRouterID := uuid.New()
	openRouterKey, err := legacy.Encrypt(ctx, []byte("router secret"), []byte(openRouterID.String()))
	require.NoError(t, err)
	openRouterConfig, err := json.Marshal(map[string]string{"apiKey": base64.StdEncoding.EncodeToString(openRouterKey)})
	require.NoError(t, err)
	require.NoError(t, db.Exec("INSERT INTO "+table+" (id, secret, configuration, app_name, access_token) VALUES (?, ?, ?, 'openrouter', ?)", openRouterID, ciphertext, string(openRouterConfig), base64.StdEncoding.EncodeToString(openRouterKey)).Error)

	job, err := migrator.Start(ctx)
	require.NoError(t, err)
	_, err = store.Rotate(ctx)
	require.ErrorIs(t, err, ErrMigrationUnfinished)

	for range 100 {
		require.NoError(t, migrator.RunBatch(ctx))
		job, err = migrator.Get(ctx, job.ID)
		require.NoError(t, err)
		if job.State == "completed" {
			break
		}
	}
	require.Equal(t, "completed", job.State, job.ErrorMessage)
	var stored []byte
	require.NoError(t, db.Raw("SELECT secret FROM "+table+" WHERE id = ?", id).Row().Scan(&stored))
	keyID, versioned, err := crypto.EncryptedDataKeyID(stored)
	require.NoError(t, err)
	require.True(t, versioned)
	active, err := models.LoadActiveApplicationEncryptionKey(db)
	require.NoError(t, err)
	require.Equal(t, active.ID, keyID)
	plain, err := encryptor.Decrypt(ctx, stored, []byte("smtp_password"))
	require.NoError(t, err)
	require.Equal(t, "smtp secret", string(plain))
	var storedConfiguration []byte
	require.NoError(t, db.Raw("SELECT configuration FROM "+table+" WHERE id = ?", id).Row().Scan(&storedConfiguration))
	var config map[string]any
	decoder := json.NewDecoder(bytes.NewReader(storedConfiguration))
	decoder.UseNumber()
	require.NoError(t, decoder.Decode(&config))
	require.Equal(t, "https://example.com", config["baseURL"])
	require.Equal(t, json.Number("9007199254740993"), config["largeNumber"])
	for field, expected := range map[string]string{"apiKey": "api secret", "adminKey": "admin secret"} {
		decoded, err := base64.StdEncoding.DecodeString(config[field].(string))
		require.NoError(t, err)
		keyID, versioned, err := crypto.EncryptedDataKeyID(decoded)
		require.NoError(t, err)
		require.True(t, versioned)
		require.Equal(t, active.ID, keyID)
		plain, err := encryptor.Decrypt(ctx, decoded, []byte(id.String()))
		require.NoError(t, err)
		require.Equal(t, expected, string(plain))
	}
	require.NoError(t, db.Raw("SELECT configuration FROM "+table+" WHERE id = ?", openRouterID).Row().Scan(&storedConfiguration))
	require.NoError(t, json.Unmarshal(storedConfiguration, &config))
	decoded, err := base64.StdEncoding.DecodeString(config["apiKey"].(string))
	require.NoError(t, err)
	plain, err = encryptor.Decrypt(ctx, decoded, []byte(openRouterID.String()))
	require.NoError(t, err)
	require.Equal(t, "router secret", string(plain))
	var storedToken string
	require.NoError(t, db.Raw("SELECT access_token FROM "+table+" WHERE id = ?", id).Row().Scan(&storedToken))
	decoded, err = base64.StdEncoding.DecodeString(storedToken)
	require.NoError(t, err)
	keyID, versioned, err = crypto.EncryptedDataKeyID(decoded)
	require.NoError(t, err)
	require.True(t, versioned)
	require.Equal(t, active.ID, keyID)
	plain, err = encryptor.Decrypt(ctx, decoded, []byte(id.String()))
	require.NoError(t, err)
	require.Equal(t, "api secret", string(plain))
	_, err = store.Rotate(ctx)
	require.NoError(t, err)
	if _, plaintext := legacy.(*crypto.NoOpEncryptor); plaintext {
		return
	}
	require.NoError(t, db.Exec("UPDATE "+table+" SET secret = ? WHERE id = ?", []byte("invalid ciphertext"), id).Error)
	failedJob, err := migrator.Start(ctx)
	require.NoError(t, err)
	require.NoError(t, migrator.RunBatch(ctx))
	failedJob, err = migrator.Get(ctx, failedJob.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", failedJob.State)
	require.Contains(t, failedJob.ErrorMessage, location.Name)
	require.NoError(t, db.Exec("UPDATE "+table+" SET secret = ? WHERE id = ?", stored, id).Error)
	_, err = migrator.Retry(ctx, failedJob.ID)
	require.NoError(t, err)
	for range 100 {
		require.NoError(t, migrator.RunBatch(ctx))
		failedJob, err = migrator.Get(ctx, failedJob.ID)
		require.NoError(t, err)
		if failedJob.State == "completed" {
			break
		}
	}
	require.Equal(t, "completed", failedJob.State, failedJob.ErrorMessage)
}

func testEncryptedStorageLocation(table, column, associatedData, appName string, format models.EncryptedStorageFormat) models.EncryptedStorageLocation {
	return models.EncryptedStorageLocation{
		Name:   table + "." + column,
		Format: format,
		ReadPage: func(tx *gorm.DB, cursor string, limit int) ([]models.EncryptedStorageRow, error) {
			query := fmt.Sprintf("SELECT id::text, %s, %s, %s FROM %s WHERE %s IS NOT NULL", column, associatedData, appName, table, column)
			args := []any{}
			if cursor != "" {
				query += " AND id > CAST(? AS uuid)"
				args = append(args, cursor)
			}
			query += " ORDER BY id LIMIT ?"
			args = append(args, limit)
			rows, err := tx.Raw(query, args...).Rows()
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			page := make([]models.EncryptedStorageRow, 0, limit)
			for rows.Next() {
				var row models.EncryptedStorageRow
				if err := rows.Scan(&row.Cursor, &row.Value, &row.AssociatedData, &row.AppName); err != nil {
					return nil, err
				}
				page = append(page, row)
			}
			return page, rows.Err()
		},
		Replace: func(tx *gorm.DB, row models.EncryptedStorageRow, replacement []byte) (bool, error) {
			oldValue, newValue := any(row.Value), any(replacement)
			valueExpr := "?"
			if format == models.EncryptedStorageIntegrationJSON {
				oldValue, newValue = string(row.Value), string(replacement)
				valueExpr = "CAST(? AS jsonb)"
			} else if format == models.EncryptedStorageBase64 {
				oldValue, newValue = string(row.Value), string(replacement)
			}
			query := fmt.Sprintf("UPDATE %s SET %s = %s WHERE id = CAST(? AS uuid) AND %s = %s", table, column, valueExpr, column, valueExpr)
			result := tx.Exec(query, newValue, row.Cursor, oldValue)
			return result.RowsAffected == 1, result.Error
		},
	}
}
