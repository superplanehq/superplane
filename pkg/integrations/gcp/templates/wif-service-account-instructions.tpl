## Grant SuperPlane access

**1. Create a service account** and assign it the IAM roles your workflows need.

**2. Grant `roles/iam.workloadIdentityUser`** on that service account to:

~~~
{{ .Principal }}
~~~

**Via the GCP Console:** Open the service account → **Permissions** → **Grant Access**, paste the principal above and assign `roles/iam.workloadIdentityUser`.

**Via gcloud:**
~~~bash
gcloud iam service-accounts add-iam-policy-binding SERVICE_ACCOUNT_EMAIL \
  --role=roles/iam.workloadIdentityUser \
  --member="{{ .Principal }}"
~~~

**3. Enter the service account email below.**
