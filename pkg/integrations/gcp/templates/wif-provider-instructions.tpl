## Set up Workload Identity Federation

**1. Enable these APIs** in your GCP project:
- [Security Token Service API](https://console.cloud.google.com/apis/library/sts.googleapis.com)
- [IAM Service Account Credentials API](https://console.cloud.google.com/apis/library/iamcredentials.googleapis.com)
- [Cloud Resource Manager API](https://console.cloud.google.com/apis/library/cloudresourcemanager.googleapis.com)
- [Pub/Sub API](https://console.cloud.google.com/apis/library/pubsub.googleapis.com)

**2. Create a Workload Identity Pool** under **IAM & Admin → Workload Identity Federation**.

**3. Add an OIDC provider** to the pool with these settings:
- **Issuer URL:** `{{.IssuerURL}}`
- **Audience:** the pool provider resource name
- **Attribute mapping:** `google.subject=assertion.sub`

**4. Copy the provider identifier** from the provider details page and enter it below. SuperPlane accepts the resource name (`//iam.googleapis.com/…`) or the full IAM URL from the console.
