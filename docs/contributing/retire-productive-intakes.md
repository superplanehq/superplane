# Retire Productive Factory intakes

Productive Factory intake is retired. Standalone Productive **On Task** workflows remain supported.
Existing work orders, source links, files, canvases, and versions remain available.

## Inventory and owner notice

Run this read-only query on each deployment database before retirement. Save the result in the deployment record.
It includes active, paused, and soft-deleted intake canvases. Remote webhook IDs can use either metadata format.

```sql
SELECT i.organization_id, i.factory_id, i.id AS intake_id, i.canvas_id,
       i.paused_at, i.initial_import_status, w.deleted_at,
       n.node_id, n.app_installation_id, n.configuration->>'project' AS productive_project,
       n.webhook_id, h.metadata->'ids' AS remote_ids, h.metadata->>'id' AS legacy_remote_id,
       (SELECT count(*) FROM workflow_nodes other_node
        JOIN workflows other_canvas ON other_canvas.id = other_node.workflow_id
        WHERE other_node.webhook_id = n.webhook_id
          AND other_node.workflow_id <> i.canvas_id
          AND other_node.deleted_at IS NULL AND other_canvas.deleted_at IS NULL) AS other_trigger_references
FROM factory_intakes i
JOIN workflows w ON w.id = i.canvas_id
LEFT JOIN workflow_nodes n ON n.workflow_id = i.canvas_id AND n.type = 'trigger'
LEFT JOIN webhooks h ON h.id = n.webhook_id
WHERE i.source = 'productive-tasks'
ORDER BY i.organization_id, i.factory_id, i.id, n.node_id;
```

Notify each affected workspace owner before retirement. Include their workspace, Productive project, and retirement date.
Explain that existing tasks, files, source links, and history remain available. Explain that new Productive tasks will stop entering Backlog.
Record notice delivery in the deployment record. Do not include credentials in the inventory or notice.

Intakes whose records and history were already hard-deleted cannot be recovered by this operation.
Do not select arbitrary Productive canvases for retirement. Custom canvases still use the standalone trigger.

## Deployment order

1. Apply the schema and flag migrations. They do not delete intakes, work orders, or files.
2. Deploy this version to every API instance and worker process. Replace old instances before retirement.
3. Verify that source enum value `4` cannot create an intake. Verify that old setup bookmarks use the route fallback.
4. Verify that updates, imports, reseeding, and canvas publication cannot restart the affected managed intakes.
5. Coordinate regenerated gRPC clients with this deployment. Factory settings fields after field 14 have new numbers.
6. Complete the inventory and owner notices before the next step.
7. Run the maintenance command once with the normal API process configuration:

   ```sh
   /app/build/superplane retire-productive-intakes
   ```

The command uses the existing database, encryption, authorization, and webhook URL configuration. It requires no new configuration keys.
Run it with the deployment image and Secret references used by the API. Override the container command to invoke the binary directly.
The maintenance process exits after retirement. It does not start API listeners or regular workers.

The operation scans 100 intake records at a time. Each canvas uses a separate transaction.
It saves the retirement identity, publishes an empty graph, cancels executions, removes queues and subscriptions, and deletes the intake record.
It also removes nodes that were absent from the live version. Soft-deleted canvases with intake records retain their remaining history.
A failed transaction rolls back. Rerun the same command after correcting the reported failure.
Already committed retirements are skipped. Their saved identities block publication of old graphs.

Keep upgraded API and worker processes running after retirement. Do not roll back to a version that lacks the execution and publication blocks.
The cleanup worker removes unused remote Productive registrations after the transaction commits.
Shared webhooks and Productive connections remain available. A remote error retains the local webhook and its IDs for retry.
A retry accepts a remote 404 because an earlier attempt can delete only one registration.

## Verification

Confirm that no intake records remain:

```sql
SELECT count(*) FROM factory_intakes WHERE source = 'productive-tasks';
```

Inspect the retirement inventory and cleanup state:

```sql
SELECT r.organization_id, r.factory_id, r.intake_id, r.canvas_id, r.retired_at,
       entry->>'webhookId' AS webhook_id, entry->'remoteIds' AS recorded_remote_ids,
       h.deleted_at, h.metadata AS remaining_remote_ids
FROM factory_intake_retirements r
CROSS JOIN LATERAL jsonb_array_elements(r.inventory) entry
LEFT JOIN webhooks h ON h.id::text = entry->>'webhookId';
```

A shared webhook remains live. An unused webhook remains soft-deleted while cleanup retries.
A missing webhook row means that remote cleanup succeeded. Confirm remote removal with Productive when deployment verification requires independent evidence.
Do not delete retry rows manually. Check cleanup errors for token permissions and connection failures.

Confirm that retired canvases have empty live nodes and edges. Confirm that their executions finish cancellation and queues remain empty.
Open an existing work order. Check its Productive link, attachment download, source query, and referenced run history.
Check Backlog Refresh with a supported provider. Historical Productive work orders must remain available.
Send a task event to a standalone Productive **On Task** workflow and confirm delivery.

Check local Compose, the self-hosted Helm chart, the hosted Helm chart, and Launchpad for stale flag references.
API and worker deployments both need the schema migration and execution blocks. No runner configuration changes are required.
Live inventory, owner notices, shared remote ownership, and deployed configuration require deployment access.
