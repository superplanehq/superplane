## Set up Workload Identity Federation

**1. Enable these APIs** in your GCP project:
- [Security Token Service API](https://console.cloud.google.com/apis/library/sts.googleapis.com)
- [IAM Service Account Credentials API](https://console.cloud.google.com/apis/library/iamcredentials.googleapis.com)
- [Cloud Resource Manager API](https://console.cloud.google.com/apis/library/cloudresourcemanager.googleapis.com)
- [Pub/Sub API](https://console.cloud.google.com/apis/library/pubsub.googleapis.com)

**2. Find your Project ID.** Open the project picker at the top of the [Google Cloud Console](https://console.cloud.google.com/) — the **Project ID** is the value shown next to your project (not the display name). You can also find it on the [Dashboard](https://console.cloud.google.com/home/dashboard) under **Project info**. Paste it into the **Project ID** field above.

**3. Create a Workload Identity Pool** under [IAM & Admin → Workload Identity Federation](https://console.cloud.google.com/iam-admin/workload-identity-pools). Click **Create Pool**, give it a name (e.g. `superplane`), and click **Continue**.

**4. Add an OIDC provider** to the pool with these settings:
- **Select a provider:** OpenID Connect (OIDC)
- **Provider name / ID:** any value (e.g. `superplane`)
- **Issuer (URL):** `{{ .IssuerURL }}`
- **Audience:** select **Default audience**

Copy the provider's IAM URL _before_ configuring the provider attributes on the next step and paste it in the **Pool Provider** input above. e.g., `https://iam.googleapis.com/projects/<project-number>/locations/global/workloadIdentityPools/superplane/providers/<providerId>`.

Click **Continue**.

**5. Configure attribute mapping.** Add a single mapping where the **Google 1** column is `google.subject` and the **OIDC 1** column is `assertion.sub` — so `assertion.sub` is the value you enter for the `google.subject` attribute.
