## Setting up Workload Identity Federation

1. Enable the following APIs in your GCP project (each name links to that API in Google Cloud Console):
   - [**Security Token Service API**](https://console.cloud.google.com/apis/library/sts.googleapis.com) (`sts.googleapis.com`)
   - [**IAM Service Account Credentials API**](https://console.cloud.google.com/apis/library/iamcredentials.googleapis.com) (`iamcredentials.googleapis.com`)
   - [**Cloud Resource Manager API**](https://console.cloud.google.com/apis/library/cloudresourcemanager.googleapis.com) (`cloudresourcemanager.googleapis.com`)
   - [**Pub/Sub API**](https://console.cloud.google.com/apis/library/pubsub.googleapis.com) (`pubsub.googleapis.com`)

2. Go to **IAM & Admin → Workload Identity Federation** and create a pool.

3. Add an **OIDC provider** to the pool:
   - Set the **Issuer URL** to `{{.IssuerURL}}` (no trailing slash; must match the `"issuer"` field from discovery below).
   - Set **Audiences** to the pool provider resource name.
   - Set **Attribute mapping** to `google.subject=assertion.sub`

   **Issuer URL must match OIDC discovery:** Open `{{.IssuerURL}}/.well-known/openid-configuration` in a browser. The JSON field `"issuer"` must be identical to what you paste as Issuer URL in GCP (scheme, host, no trailing slash).

4. Copy the provider identifier from the provider details page. SuperPlane accepts either:
   - The **resource name**:
     ~~~
     //iam.googleapis.com/projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/POOL_ID/providers/PROVIDER_ID
     ~~~
   - Or the **IAM API URL** shown in the console or REST docs (for example `https://iam.googleapis.com/v1/projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/POOL_ID/providers/PROVIDER_ID`). SuperPlane extracts the resource name automatically.
