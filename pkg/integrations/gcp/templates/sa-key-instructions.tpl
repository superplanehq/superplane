To get a Service Account JSON key:

1. Go to [IAM & Admin → Service Accounts](https://console.cloud.google.com/iam-admin/serviceaccounts) in the Google Cloud Console.
2. Select or create a service account and assign it the IAM roles your workflows need.
3. Click the **Keys** tab → **Add Key** → **Create new key** → **JSON**.
4. Download the file and paste its contents below.

## Required IAM roles

- `roles/viewer` — validate project access and check enabled APIs
- `roles/logging.configWriter` — create logging sinks for event triggers
- `roles/pubsub.admin` — manage Pub/Sub topics, subscriptions, and IAM policies
- Additional roles depending on selected capabilities (e.g. `roles/compute.admin`, `roles/cloudbuild.builds.editor`, `roles/cloudfunctions.developer`)
