## Grant SuperPlane access

**1. Create a service account.** In the GCP console, go to [IAM & Admin → Service Accounts](https://console.cloud.google.com/iam-admin/serviceaccounts) and click **Create Service Account**.

- **Service account name:** any descriptive name (e.g. `superplane`) — the **Service account ID** is generated from the name
- Click **Create and continue**

**2. Grant the service account the IAM roles SuperPlane needs.** On the **Grant this service account access to project** step, add the base roles below, plus any capability-specific roles for the capabilities you enabled:

Base roles (always required):
- **Viewer** (`roles/viewer`) — validate project access
- **Logs Configuration Writer** (`roles/logging.configWriter`) — create logging sinks for event triggers
- **Pub/Sub Admin** (`roles/pubsub.admin`) — manage Pub/Sub topics and subscriptions

Capability-specific roles (add the ones that match your selection):
- **Compute Admin** (`roles/compute.admin`) — Compute Engine VMs (`createVM`, `deleteVMInstance`, `onVMInstance`)
- **Cloud Build Editor** (`roles/cloudbuild.builds.editor`) — Cloud Build (`createBuild`, `getBuild`, `runTrigger`, `onBuildComplete`)
- **Artifact Registry Reader** (`roles/artifactregistry.reader`) and **Container Analysis Occurrences Viewer** (`roles/containeranalysis.occurrences.viewer`) — Artifact Registry (`getArtifact`, `getArtifactAnalysis`, `onArtifactPush`, `onArtifactAnalysis`)
- **Cloud Functions Developer** (`roles/cloudfunctions.developer`) — `invokeFunction`
- **DNS Administrator** (`roles/dns.admin`) — Cloud DNS (`createRecord`, `updateRecord`, `deleteRecord`)

Click **Continue**, then **Done** to finish creating the service account.

**3. Allow SuperPlane to impersonate the service account.** From the service accounts list, open the one you just created → **Permissions** tab → **Grant access**.

- **New principals:** paste the principal below
- **Assign role:** select **Workload Identity User**

~~~
{{ .Principal }}
~~~

Click **Save**.

**Via gcloud (alternative):**
~~~bash
gcloud iam service-accounts add-iam-policy-binding SERVICE_ACCOUNT_EMAIL \
  --role=roles/iam.workloadIdentityUser \
  --member="{{ .Principal }}"
~~~

**4. Enter the service account email above** to finish connecting SuperPlane.
