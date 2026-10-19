package workers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/encryption"
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
	store := encryption.NewStore(masters, "local-v1")
	require.NoError(t, store.Initialize(ctx))
	encryptor := encryption.NewApplicationEncryptor(crypto.NewVersionedEncryptor(store, legacy), store)
	table := "encryption_migration_test_" + uuid.NewString()[:8]
	require.NoError(t, db.Exec(fmt.Sprintf("CREATE TABLE %s (id uuid PRIMARY KEY, secret bytea NOT NULL, configuration jsonb NOT NULL, app_name text NOT NULL)", table)).Error)
	t.Cleanup(func() { require.NoError(t, db.Exec("DROP TABLE "+table).Error) })
	location := testEncryptedStorageLocation(table, "secret", "'smtp_password'", "''", models.EncryptedStorageBytes)
	configurationLocation := testEncryptedStorageLocation(table, "configuration", "id::text", "app_name", models.EncryptedStorageIntegrationJSON)
	integrations, err := registry.NewRegistry(encryptor, registry.HTTPOptions{})
	require.NoError(t, err)
	worker := NewApplicationEncryptionMigrationWorker("", encryptor, integrations, []models.EncryptedStorageLocation{location, configurationLocation})
	startJob := func() (*models.ApplicationEncryptionMigration, error) {
		active, err := models.LoadActiveApplicationEncryptionKey(db)
		if err != nil {
			return nil, err
		}
		job := &models.ApplicationEncryptionMigration{
			ID: uuid.New(), TargetKeyID: active.ID, State: "pending",
			RegistryVersion: encryption.RegistryVersion,
			StorageLocation: location.Name,
		}
		return job, db.Create(job).Error
	}
	getJob := func(id uuid.UUID) (*models.ApplicationEncryptionMigration, error) {
		return models.LoadApplicationEncryptionMigration(db, id)
	}

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
	require.NoError(t, db.Exec("INSERT INTO "+table+" (id, secret, configuration, app_name) VALUES (?, ?, ?, 'openai')", id, ciphertext, string(configuration)).Error)
	openRouterID := uuid.New()
	openRouterKey, err := legacy.Encrypt(ctx, []byte("router secret"), []byte(openRouterID.String()))
	require.NoError(t, err)
	openRouterConfig, err := json.Marshal(map[string]string{"apiKey": base64.StdEncoding.EncodeToString(openRouterKey)})
	require.NoError(t, err)
	require.NoError(t, db.Exec("INSERT INTO "+table+" (id, secret, configuration, app_name) VALUES (?, ?, ?, 'openrouter')", openRouterID, ciphertext, string(openRouterConfig)).Error)

	job, err := startJob()
	require.NoError(t, err)
	_, err = store.CreateAndActivateDataKey(ctx)
	require.ErrorIs(t, err, encryption.ErrMigrationUnfinished)

	for range 2 {
		_, err = worker.RunBatch(ctx)
		require.NoError(t, err)
		job, err = getJob(job.ID)
		require.NoError(t, err)
	}
	require.Equal(t, "auditing", job.State)
	_, err = store.CreateAndActivateDataKey(ctx)
	require.ErrorIs(t, err, encryption.ErrMigrationUnfinished)
	for range 2 {
		_, err = worker.RunBatch(ctx)
		require.NoError(t, err)
		job, err = getJob(job.ID)
		require.NoError(t, err)
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
	newKeyID, err := store.CreateAndActivateDataKey(ctx)
	require.NoError(t, err)
	if _, plaintext := legacy.(*crypto.NoOpEncryptor); plaintext {
		return
	}
	corruptID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	require.NoError(t, db.Exec("INSERT INTO "+table+" (id, secret, configuration, app_name) VALUES (?, ?, '{}', 'openai')", corruptID, []byte("invalid ciphertext")).Error)
	failedJob, err := startJob()
	require.NoError(t, err)
	for range 100 {
		_, err = worker.RunBatch(ctx)
		require.NoError(t, err)
		failedJob, err = getJob(failedJob.ID)
		require.NoError(t, err)
		if failedJob.State == "failed" {
			break
		}
	}
	require.Equal(t, "failed", failedJob.State)
	require.NotEmpty(t, failedJob.ErrorMessage)
	require.NoError(t, db.Raw("SELECT secret FROM "+table+" WHERE id = ?", id).Row().Scan(&stored))
	keyID, versioned, err = crypto.EncryptedDataKeyID(stored)
	require.NoError(t, err)
	require.True(t, versioned)
	require.Equal(t, newKeyID, keyID)

	repaired, err := legacy.Encrypt(ctx, []byte("repaired secret"), []byte("smtp_password"))
	require.NoError(t, err)
	require.NoError(t, db.Exec("UPDATE "+table+" SET secret = ? WHERE id = ?", repaired, corruptID).Error)
	newJob, err := startJob()
	require.NoError(t, err)
	require.NotEqual(t, failedJob.ID, newJob.ID)
	require.Equal(t, failedJob.TargetKeyID, newJob.TargetKeyID)
	for range 2 {
		_, err = worker.RunBatch(ctx)
		require.NoError(t, err)
		newJob, err = getJob(newJob.ID)
		require.NoError(t, err)
	}
	require.Equal(t, "auditing", newJob.State)
	require.NoError(t, db.Exec("UPDATE "+table+" SET secret = ? WHERE id = ?", ciphertext, id).Error)
	for range 2 {
		_, err = worker.RunBatch(ctx)
		require.NoError(t, err)
		newJob, err = getJob(newJob.ID)
		require.NoError(t, err)
	}
	require.Equal(t, "failed", newJob.State)
	require.Contains(t, newJob.ErrorMessage, "target data key")

	retryJob, err := startJob()
	require.NoError(t, err)
	require.NoError(t, worker.consumeMigrationRequested(ctx, tackle.NewFakeDelivery(nil)))
	retryJob, err = getJob(retryJob.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", retryJob.State, retryJob.ErrorMessage)
	failedJob, err = getJob(failedJob.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", failedJob.State)
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
			}
			query := fmt.Sprintf("UPDATE %s SET %s = %s WHERE id = CAST(? AS uuid) AND %s = %s", table, column, valueExpr, column, valueExpr)
			result := tx.Exec(query, newValue, row.Cursor, oldValue)
			return result.RowsAffected == 1, result.Error
		},
	}
}
