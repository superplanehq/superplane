# Application Encryption Key Rotation

## Decision

SuperPlane will store versioned data encryption keys in PostgreSQL. A master key
will encrypt each stored data key. Go's AES-GCM implementation will encrypt
application values. The master key can come from a local secret or a KMS key.

Ciphertext includes a SuperPlane format marker and a data key identifier.
Readers require the versioned format. Writers use that format for every new value.

Installation admins will manage data keys in the admin UI.
An installation worker will migrate stored ciphertext in bounded batches.
PostgreSQL will store job progress and coordinate worker pods.

## Goals

- Create active data keys without making existing ciphertext unreadable.
- Re-encrypt stored values with an active data key when an admin requests it.
- Resume migration after a worker or database connection fails.
- Run one migration job across multiple worker pods without duplicate commits.
- Support a local master key for single-VM installations and KMS for SaaS.
- Rewrap data keys without re-encrypting stored application values.

## Key Model

The `application_encryption_keys` table stores each key with `active` or
`available` state. A unique index permits at most one active key. PostgreSQL
also stores each wrapped data key and its master key reference. The master
provider wraps and unwraps data keys. It does not encrypt each application value.

Use Go's AES-256-GCM implementation for data values. Store a format marker
outside the ciphertext. The envelope identifies the data key. Authenticate
the envelope header together with the caller's existing associated data.
Preserve the current associated data for each storage location.

`SPENC1` means SuperPlane encrypted value format version 1. The stored bytes
contain this marker, a 16-byte data key ID, and AES-GCM ciphertext. The format
also binds the marker and key ID as authenticated data.

The decryptor requires the format marker, parses the envelope, and authenticates
the value. It rejects unmarked values and authentication failures.

The new encryptor will always read the active data key ID from PostgreSQL.
An in-flight write can finish with the previous key after key creation returns.
This is safe because the old key remains available. A read-only audit after
migration finds any values that still use an old key. Do not retire old data
keys while retained backups or in-flight writes can reference them.

The local master provider will read a map of named keys from a secret supplied
at startup. The stored master key reference selects one entry in that map.
The SaaS provider will use a Google Cloud KMS CryptoKey URI and workload
identity. Neither provider will write plaintext data keys to disk or logs.
SuperPlane will fail startup if it cannot unwrap a stored data key. It will not
create replacement keys over unreadable ones.

Master keys come from one ordered `ENCRYPTION_MASTER_KEYS` value. The first
entry is primary for new data keys and rewraps. A local entry has the form
`env://<id>:<base64-key>` and contains 32 random key bytes. A KMS entry has
the form `gcp-kms://projects/.../locations/.../keyRings/.../cryptoKeys/...`.
The database stores `env://<id>` or the KMS URI as `master_key_ref`; it never
stores local key material. Keep old entries until rewrap and backup retention
permit removal.

The KMS provider uses workload identity with permission to encrypt and decrypt
with the configured CryptoKey.

## Encrypted Storage Registry

Define a registry in Go. Each entry has a stable identifier and code that can:

- Select a page of stored values in a stable order.
- Find each encrypted value in the row.
- Reconstruct its existing associated data.
- Decode and encode its storage form, including base64 values in JSON or text.
- Replace a value without overwriting a concurrent application update.

The registry covers every field encrypted through `crypto.Encryptor`:

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

The registry includes sensitive values stored inside JSON configuration. Some tables
have more than one encrypted value per row. Each value needs its own migration
check and update. Do not infer encryption from column types or names alone.

Registry identifiers are code identifiers, such as `secrets.data` and
`app_installations.configuration.sensitive_fields`. They are not SQL stored in
the database. Keep these identifiers stable while a job can reference them.
Do not remove a handler until all installations have completed jobs that use it.

## Admin Interface

Add an Encryption page to the installation admin UI. Installation admin
authorization will apply to all endpoints. Never return key material in an API
response.

| UI action | Endpoint | Result |
| --- | --- | --- |
| View data keys | `GET /admin/api/installation/keys` | List data key IDs, creation times, and the active ID. |
| Create data key | `POST /admin/api/installation/keys` | Create and activate a data key. |
| Start migration | `POST /admin/api/installation/keys/migrations` | Create a migration job for the current active key. Return its ID. |
| View migrations | `GET /admin/api/installation/keys/migrations` | List recent jobs so progress remains visible after a page reload. |
| Fetch one job | `GET /admin/api/installation/keys/migrations/{id}` | Show state, current storage location, and failures. |
| Rewrap data keys | `POST /admin/api/installation/keys/rewrap` | Rewrap data keys with a configured master key. |

The UI starts a server job and shows its saved progress. The browser does not
hold decrypted values or run migration batches.

Reject data key creation while a migration job is pending, running, or auditing.
Enforce this rule in the same database transaction that activates a new key.
An admin can start a new job after a failed job. The failed job remains in the history.
Keep old data keys available while stored values or backups reference them.
Permit only one unfinished migration job per installation.

## Migration Job

Use one PostgreSQL row per job. The row stores:

- Job ID, target data key ID, and registry version.
- State: `pending`, `running`, `auditing`, `failed`, or `completed`.
- Current storage location identifier and page cursor.
- Creation, update, and completion times.
- A short error code and message, without plaintext or ciphertext.

Log unreadable values with their storage location and row ID. Keep logs free of
plaintext and ciphertext. Do not store one progress row per migrated value.
Do not require scanned or migrated counters for correctness.

Log job start, location changes, completion, and failure. Log each
committed batch at info level. Include the job ID and target key ID so an
operator can follow one job across worker pods. Emit progress and outcome logs
after the transaction commits. Log transaction errors when they occur.

The target key ID is fixed when the job starts. The job scans each registered
location once. A cursor identifies the last examined row
within the current storage location. Process registry handlers sequentially in
the first version. Separate progress rows and parallel handlers are optional
future work. A worker must refuse to resume a job if its code lacks the job's
registry version or current handler identifier.

The admin API creates and reads job rows through the models. After it creates
a job, it publishes a RabbitMQ wake-up message. A worker in `pkg/workers`
processes batches until no unfinished job remains. Workers also check at
startup and every five minutes to recover jobs after a missed message or a
worker crash. For each locked batch, the worker creates a job-bound migrator
or auditor. The migrator changes stored values; the auditor only reads them.
The worker saves progress and logs results after commit.

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
overwrite the request. The read-only audit can find that value if it still
uses an old key.

Continue past unreadable values and process the remaining rows. Mark the job
failed after the migration scan if any value could not be migrated. Keep
successful changes. After the operator resolves errors, start a new job. The
new job scans all locations and targets the active data key.

After the migration scan, the worker audits the same storage registry in
bounded read-only batches. It marks the job completed only if the audit finds
no value with another key. It marks the job failed if it finds a
mismatch, and logs the affected storage location and row ID. An operator can
also run [the SQL query](application-encryption-audit.sql) with a job ID to
investigate. The audit checks stored key IDs; it cannot prove that a value
decrypts or that a later write will use the same key. Keep old keys available
until backup retention and any remaining writes permit removal.

## Master Key Rewrap

A change to the primary master key requires every API and worker instance to
have both the old and new master keys before data key creation or rewrap.
Otherwise, an instance can fail at startup when it cannot unwrap a stored data
key. The old master key remains available until rewrap and backup retention
permit its removal.

A rewrap decrypts each stored data key with its recorded master key reference
and encrypts it with the configured primary master key. The operation updates
all wrapped data keys and their references in one database transaction. A
failure leaves the prior keys and references intact. Application ciphertext
and data key IDs do not change. The API does not accept raw master keys.

## Validation

- Verify encryption and decryption across active and old data keys.
- Verify rotation between versioned data keys for each registry handler.
- Verify JSON handlers preserve unrelated fields and handle missing values.
- Verify associated data remains correct for every storage location.
- Verify concurrent updates cannot lose application writes.
- Verify a worker crash before and after a batch commit.
- Verify a second worker skips a locked job row and another worker resumes after rollback.
- Verify data key creation fails while a migration remains unfinished.
- Verify the read-only audit finds values that do not use the target key.
- Verify a KMS or local master rewrap preserves all data key IDs.
- Verify restore from a backup that contains values encrypted with an old data key.

## References

- [Google Cloud KMS key rotation](https://docs.cloud.google.com/kms/docs/key-rotation)
