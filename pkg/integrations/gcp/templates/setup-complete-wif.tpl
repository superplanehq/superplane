Your SuperPlane integration is configured to use Workload Identity Federation.

| Property | Value |
|----------|-------|
| Project | `{{ .ProjectID }}` |
| Service Account | `{{ .ServiceAccountEmail }}` |
| Auth Method | Workload Identity Federation |
| Integration Subject | `app-installation:{{ .IntegrationID }}` |

The first sync will exchange a federated token and impersonate the service account. Make sure the `roles/iam.workloadIdentityUser` binding is in place before the integration syncs.
