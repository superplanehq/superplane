# Runtime configuration

SuperPlane reads a small set of environment variables at runtime to tune execution limits. These apply to self-hosted deployments.

## Execution limits

| Variable | Default | Description |
| --- | --- | --- |
| `SUPERPLANE_MAX_EMIT_COUNT` | `100` | Maximum number of events a single component execution may emit at once. Applies to fan-out components such as **For Each** (one event per array item) and **Read Memory** when emit mode is **One By One**. |
| `SUPERPLANE_MAX_PAYLOAD_SIZE` | `524288` (512 KiB) | Maximum serialized size of an emitted event payload, in bytes. |

Invalid or non-positive values are ignored; the default is used instead.

Set these on the SuperPlane API and worker processes. Changes take effect on the next process start (or immediately on the next read, depending on the variable).
