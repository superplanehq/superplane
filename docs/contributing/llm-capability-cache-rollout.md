# LLM capability cache rollout

The cache stores provider metadata separately from billing prices. It does not
change model selection, model permissions, API responses, or Start admission.

## Processes and configuration

| Process | Required access and configuration |
| --- | --- |
| API | Database access, saved hosted credentials, Claude integration credentials, `ENCRYPTION_MASTER_KEYS`, and provider egress for saves and discovery. |
| Catalog worker | The same database, saved credentials, encryption keys, and provider egress. Set `START_PRICE_BOOK_SYNC_WORKER=yes`. |
| Runner | No configuration change in this phase. |

Database configuration uses the existing `DB_HOST`, `DB_PORT`, `DB_NAME`,
`DB_USERNAME`, `DB_PASSWORD`, and `POSTGRES_DB_SSL` settings. Preserve the encryption key versions
that decrypt existing hosted and integration credentials. Credentials stay in
existing encrypted storage. Capability rows contain no credentials.

Allow requests to `https://openrouter.ai/api/v1/models`,
`https://api.anthropic.com/v1/models`, and each saved provider endpoint.
Anthropic pagination uses `after_id` on the same endpoint. Continue to apply
existing endpoint validation and HTTP network policy.

## Configuration record

- Inspected `docker-compose.dev.yml`. It enables the worker flag and supplies
  database settings and `ENCRYPTION_MASTER_KEYS` to the application process.
- Inspected repository Helm templates `api.yaml`, `workers.yaml`, and
  `encryption.yaml`. API and worker pods import the database and encryption
  Secrets. Workers already enable `START_PRICE_BOOK_SYNC_WORKER`. These
  requirements need no template change. Repository values default to one worker
  replica. This inspection does not verify a
  published chart or deployed Secret.
- Read access through `gh api repos/superplanehq/helm-chart` and
  `gh api repos/superplanehq/launchpad` returned HTTP 404. No companion PR was
  prepared or published. A deployment owner must inspect both repositories.
  Add the worker flag or database/encryption Secret references if absent.
- Deployed Secrets, saved credentials, provider egress, and live catalog
  population remain unverified. These are production activation gates.

## Refresh behavior

The worker refreshes on startup and daily after a complete scan. Each scan has
at most five minutes. Each provider refresh has at most 30 seconds, including
pagination and publication. Fetches run outside database transactions.

Claude scans read at most 100 ready integrations per window, ordered by ID.
They exclude deleted integrations and deleted or suspended organizations.
The worker advances its cursor before each attempt, including failed
credentials. It continues unfinished scans after one minute. Repeated scan
cancellations therefore do not restart with the same first integrations.
A process restart starts a new pass. The cursor contains no credentials and
needs no durable scheduling records.

Replicas can repeat idempotent provider requests. Source publications use a
transaction and a database advisory lock to prevent conflicting writes.
Replica exclusion is not required for correctness. Monitor provider rate limits
and refresh duration before increasing worker replicas. Configured sources also
refresh during continuation windows; the public OpenRouter request is skipped
when a successful saved refresh already covers the public endpoint.

## Deployment order

1. Apply the additive migration before API and worker deployment.
2. Verify database and credential access for API and worker pods independently.
3. Verify encryption Secret references and provider egress in both deployment repositories.
4. Deploy the API and workers with the existing worker flag enabled.
5. Verify successful refresh logs and populated catalogs for each provider endpoint.
6. Complete source lookup and admission verification before later-phase activation.

Use a database query such as the following to check source population without
reading credentials:

```sql
SELECT provider, base_url, count(*) AS models, max(refreshed_at) AS refreshed_at
FROM llm_model_capabilities
GROUP BY provider, base_url
ORDER BY provider, base_url;
```

Verify the hosted endpoint separately from the public BYOK endpoint. Confirm
that a failed refresh preserves rows and their previous successful metadata.
Passing repository tests does not establish live deployment readiness.

Keep the additive cache table during an application rollback.
