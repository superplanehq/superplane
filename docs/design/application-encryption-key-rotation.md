# Application Encryption Key Rotation

## Decision

SuperPlane will store versioned data encryption keys in PostgreSQL. A master key
will encrypt each stored data key. Go's AES-GCM implementation will encrypt
application values. The master key can come from a local secret or a KMS key.

Ciphertext will include a SuperPlane format marker and a data key identifier.
Readers will support the current legacy format during conversion. Writers will
use the new format after the rollout completes.

Installation admins will manage data keys through `superplane admin keys`.
An installation worker will migrate stored ciphertext in bounded batches.
PostgreSQL will store job progress and coordinate worker pods.

## Goals

- Rotate data keys without making existing ciphertext unreadable.
- Convert ciphertext created with the current `ENCRYPTION_KEY`.
- Re-encrypt stored values with an active data key when an admin requests it.
- Resume migration after a worker or database connection fails.
- Run one migration job across multiple worker pods without duplicate commits.
- Support a local master key for single-VM installations and KMS for SaaS.
- Rewrap data keys without re-encrypting stored application values.

## Key Model

The key registry holds one active data key and older keys needed for decryption.
PostgreSQL stores each wrapped data key and its master key reference. The master
provider wraps and unwraps data keys. It does not encrypt each application value.

Use Go's AES-256-GCM implementation for data values. Store a format marker
outside the ciphertext so readers can distinguish it from the legacy nonce and
ciphertext format. The new envelope identifies the data key. Authenticate
the envelope header together with the caller's existing associated data.
Preserve the current associated data for each storage location.

`SPENC1` means SuperPlane encrypted value format version 1. The stored bytes
contain this marker, a 16-byte data key ID, and AES-GCM ciphertext. The format
also binds the marker and key ID as authenticated data.

The new decryptor will use this order:

1. If the value has the new format marker, parse and authenticate the envelope.
2. If the value has no marker, decrypt it with the legacy `ENCRYPTION_KEY`.
3. If a new-format value fails authentication, return an error. Do not try the
   legacy key.

The new encryptor will always read the active data key ID from PostgreSQL.
An in-flight write can finish with the previous key after rotation returns.
This is safe because the old key remains available. The migration's final
verification pass must catch such writes. Do not retire old data keys while
retained backups or in-flight writes can reference them.

The local master provider will read a map of named keys from a secret supplied
at startup. The stored master key reference selects one entry in that map.
The SaaS provider will use a Google Cloud KMS CryptoKey URI and workload
identity. Neither provider will write plaintext data keys to disk or logs.
SuperPlane will fail startup if it cannot unwrap a stored data key. It will not
create replacement keys over unreadable ones.

Use `ENCRYPTION_LOCAL_MASTER_KEYS` for the local provider. Its value will be a
comma-separated list of `key-id:base64-key` entries. Each key must contain
32 random bytes before base64 encoding. For example,
`local-v1:<base64-key>,local-v2:<base64-key>`. Permit only letters, digits,
hyphens, and underscores in key IDs. Reject duplicate IDs, invalid base64,
and keys of the wrong length at startup. Use
`ENCRYPTION_KMS_KEY_URIS` for the KMS provider. Its value will use the
`gcp-kms://projects/.../locations/.../keyRings/.../cryptoKeys/...` form.
Keep `ENCRYPTION_KEY` for legacy decryption during conversion. A new
installation can use a master key without setting `ENCRYPTION_KEY`.

For a single-VM installation, generate a local master key:

```sh
openssl rand -base64 32
```

Then add the result to the application environment:

```dotenv
ENCRYPTION_LOCAL_MASTER_KEYS=local-v1:<base64-output>
ENCRYPTION_INITIAL_MASTER_KEY_REF=local-v1
```

For a Helm installation with Google Cloud KMS, set these values and bind the
pod service account to a Google service account that can encrypt and decrypt
with the CryptoKey:

```yaml
encryption:
  kmsKeyURIs: gcp-kms://projects/PROJECT/locations/global/keyRings/RING/cryptoKeys/KEY
  initialMasterKeyRef: gcp-kms://projects/PROJECT/locations/global/keyRings/RING/cryptoKeys/KEY
```

For an existing installation, keep its `ENCRYPTION_KEY` in the encryption
Secret while legacy ciphertext or retained backups still need it. If the chart
uses `encryption.secretName`, add the new environment variables to that Secret.

## Encrypted Storage Registry

Define a registry in Go. Each entry has a stable identifier and code that can:

- Select a page of stored values in a stable order.
- Find each encrypted value in the row.
- Reconstruct its existing associated data.
- Decode and encode its storage form, including base64 values in JSON or text.
- Replace a value without overwriting a concurrent application update.
- Check whether any value still uses a legacy or non-target data key.

The registry must cover every field encrypted through `crypto.Encryptor`.
This initial inventory requires confirmation during implementation:

| Storage location | Encrypted values |
| --- | --- |
| `secrets` | `data` |
| `app_installation_secrets` | `value` |
| `app_installations` | Sensitive fields in `configuration` JSON |
| `webhooks` | `secret` |
| `runner_tasks` | `payload_ciphertext` |
| `factory_agent_resource_secrets` | `value` |
| `hosted_llm_providers` | `api_key`, `management_key` |
| `installation_github_apps` | `encrypted_private_key`, `encrypted_webhook_secret` |
| `installation_licenses` | `encrypted_license` |
| `email_settings` | `smtp_password` |
| `bitbucket_forge_installations` | `system_token` |
| `sentry_app_install_grants` | `access_token`, `refresh_token` |
| `account_providers` | Base64 ciphertext in `access_token` |

The implementation audit must inspect all encryption and decryption call sites.
It must also find sensitive values stored inside JSON configuration. Some tables
have more than one encrypted value per row. Each value needs its own migration
check and update. Do not infer encryption from column types or names alone.

Registry identifiers are code identifiers, such as `secrets.data` and
`app_installations.configuration.sensitive_fields`. They are not SQL stored in
the database. Keep these identifiers stable while a job can reference them.
Do not remove a handler until all installations have completed jobs that use it.

## Admin Interface

Add these commands to the regular `superplane` CLI. Use its existing context
URL and personal API token. Installation admin authorization will apply to all
endpoints. Never return key material in an API response.

| Command | Endpoint | Result |
| --- | --- | --- |
| `superplane admin keys list` | `GET /admin/api/installation/keys` | List data key IDs, creation times, and the active ID. |
| `superplane admin keys rotate` | `POST /admin/api/installation/keys/rotate` | Add and activate a data key. |
| `superplane admin keys migrate` | `POST /admin/api/installation/keys/migrations` | Create a migration job for the current active key. Return its ID. |
| `superplane admin keys migrate status <id>` | `GET /admin/api/installation/keys/migrations/{id}` | Show state, current storage location, and failures. |
| `superplane admin keys migrate retry <id>` | `POST /admin/api/installation/keys/migrations/{id}/retry` | Resume a failed job. |
| `superplane admin keys rewrap [--master-key-ref <ref>]` | `POST /admin/api/installation/keys/rewrap` | Rewrap data keys with a configured master key. |

The `migrate` command starts a server job and exits after it receives the job
ID. The CLI does not hold decrypted values or run migration batches.

Reject `keys rotate` while a migration job is pending, running, or failed.
Enforce this rule in the same database transaction that activates a new key.
An admin must retry a failed job and complete it before another rotation.
Keep old data keys available while stored values or backups reference them.
Permit only one unfinished migration job per installation.

## Migration Job

Use one PostgreSQL row per job. The row stores:

- Job ID, target data key ID, and registry version.
- State: `pending`, `running`, `failed`, `completed`, or `abandoned`.
- Current storage location identifier, page cursor, and pass.
- Creation, update, and completion times.
- A short error code and message, without plaintext or ciphertext.

Store records for individual failures only when a value needs investigation.
Do not store one progress row per migrated value. Do not require scanned or
migrated counters for correctness.

The target key ID is fixed when the job starts. The first pass converts values.
Later passes verify that all values use the target key. A cursor identifies the last examined row
within the current storage location. Process registry handlers sequentially in
the first version. Separate progress rows and parallel handlers are optional
future work. A worker must refuse to resume a job if its code lacks the job's
registry version or current handler identifier.

For each batch, the worker will:

1. Lock the job row with `FOR UPDATE SKIP LOCKED`.
2. Read a bounded page from the current handler.
3. Skip values that already use the target data key.
4. Decrypt and encrypt values that need migration.
5. Update each value only if its stored ciphertext still matches the value read.
6. Commit value updates and the new cursor in one transaction.
7. Commit and release the row lock so another worker pod can run the next batch.

The transaction holds the job row lock while it processes one bounded batch.
Keep batches short. A worker can process several batches, but it must acquire
the row lock for each batch. Other worker pods skip the locked row.

If a pod stops before commit, PostgreSQL rolls back and another pod repeats the batch.
If it stops after commit, the next pod resumes from the saved cursor. A
conditional update that loses a race with an application request must not
overwrite the request. A later verification scan will revisit that value if it
still uses an old key.

After the initial walk, scan every registered location again for legacy values
or values that do not use the target key. Repeat bounded passes until none
remain. Mark the job completed only when all locations pass verification and
no unresolved failures remain. Report decryption failures. Never mark them as
skipped successes.

## Rollout and Operations

### Convert legacy ciphertext

1. Add the data key registry, master providers, new envelope, and dual-format reader.
2. Add the storage registry, admin APIs, CLI commands, and migration worker.
3. Deploy the new reader to every API and worker instance.
4. Switch every writer to the new envelope and active data key.
5. Confirm that old application versions can no longer write legacy values.
6. Start `superplane admin keys migrate`.
7. Inspect failures and retry the job until it completes.
8. Keep `ENCRYPTION_KEY` available while legacy values or retained backups need it.

Do not remove the legacy decryptor only because the current database is clean.
An older backup can restore legacy ciphertext. Define backup restore support
before retiring the old key or code path. Development installations that use
`NO_ENCRYPTION=yes` need a separate conversion decision; their stored values
do not have legacy AES-GCM ciphertext.

### Rotate a data key

1. Run `superplane admin keys rotate`.
2. New writes use the new active key. Old ciphertext stays readable.
3. Run `superplane admin keys migrate` only when re-encryption is required.
4. Keep old data keys while current data or retained backups need them.

Rotation does not require a migration job. Migration is an explicit operation.
Do not remove an old data key until verification and backup retention allow it.

### Rotate a master key

For a new version of the same KMS CryptoKey, rotate the KMS key version first.
Then run `superplane admin keys rewrap`. The endpoint unwraps the stored data
keys and wraps them with the new primary KMS version. It atomically
replaces the wrapped keys and verifies that it can read them. Application
ciphertext and data key IDs do not change. If the flag is absent, use the stored
master key reference.

For a new KMS URI or local master key, deploy both old and new key access to
every API and worker instance before rewrap. The stored master key reference
selects the old key. Run `superplane admin keys rewrap --master-key-ref <new-ref>`.
The server selects the new key from its configuration. It updates the wrapped
data keys and stored references in one transaction. Afterward, verify a fresh
instance can start. Remove old access only after backup retention permits it.
The CLI and API must not accept raw master keys. A rewrap failure must keep
the prior data keys and references intact.

## Validation

- Verify encryption and decryption across active and old data keys.
- Verify legacy decryption and conversion for each registry handler.
- Verify JSON handlers preserve unrelated fields and handle missing values.
- Verify associated data remains correct for every storage location.
- Verify concurrent updates cannot lose application writes.
- Verify a worker crash before and after a batch commit.
- Verify a second worker skips a locked job row and another worker resumes after rollback.
- Verify rotation fails while a migration remains unfinished.
- Verify final scans catch values changed during the initial walk.
- Verify a KMS or local master rewrap preserves all data key IDs.
- Verify restore from a backup that still contains legacy or old-key values.

## Implementation Order

1. Complete the encryption call-site audit and registry inventory.
2. Implement the data key registry, master providers, and versioned encryptor.
3. Add dual-format decryption and switch all writers to the new format.
4. Add installation admin APIs and regular CLI commands.
5. Add the durable migration job, worker row lock, and storage handlers.
6. Run legacy conversion and verify every registered location.
7. Add the master rewrap operation and document backup key retention.

## References

- [Google Cloud KMS key rotation](https://docs.cloud.google.com/kms/docs/key-rotation)
