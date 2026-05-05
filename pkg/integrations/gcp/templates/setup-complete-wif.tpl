Your SuperPlane integration is configured to use Workload Identity Federation.

| Property | Value |
|----------|-------|
| Project | `{{ .ProjectID }}` |
| Service Account | `{{ .ServiceAccountEmail }}` |
| Auth Method | Workload Identity Federation |
| Integration Subject | `app-installation:{{ .IntegrationID }}` |

The first sync runs shortly after setup (after a short delay so new IAM bindings can propagate). It exchanges a federated token and impersonates the service account. Ensure `roles/iam.workloadIdentityUser` is granted before sync completes; if you still see errors, wait a minute and **Resync**.
