## Grant service account impersonation

Create a service account and grant it the IAM roles your workflows need. Then grant SuperPlane permission to impersonate it.

Grant `roles/iam.workloadIdentityUser` on the service account to the following principal:

~~~
{{ .Principal }}
~~~

**Via the GCP Console:**
1. Go to **IAM & Admin → Service Accounts** and select your service account.
2. Open the **Permissions** tab and click **Grant Access**.
3. Paste the principal above and assign `roles/iam.workloadIdentityUser`.

**Via `gcloud`:**
~~~bash
gcloud iam service-accounts add-iam-policy-binding SERVICE_ACCOUNT_EMAIL \
  --role=roles/iam.workloadIdentityUser \
  --member="{{ .Principal }}"
~~~

**Note:** IAM changes often take a minute or longer to fully propagate in Google Cloud. SuperPlane waits a short interval before running the first sync after setup; if you still see a permission error, wait briefly and use **Resync** from this integration.
