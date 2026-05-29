Google Cloud is connected via Workload Identity Federation.

| | |
|---|---|
| Project | `{{ .ProjectID }}` |
| Service Account | `{{ .ServiceAccountEmail }}` |

The first sync runs automatically. If it fails with a permission error, wait a minute and click **Resync** — IAM bindings can take time to propagate.
