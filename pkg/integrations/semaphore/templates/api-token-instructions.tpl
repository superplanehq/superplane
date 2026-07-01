There are two ways to provide a Semaphore API token:
1. Use a service account - **recommended**
2. Use a personal API token
---
## 1. Use a service account
If your organization has access to service accounts, you can use one of them to connect to SuperPlane.
- Go to {{ .OrganizationURL }}/people
- Create a service account, with the **Admin** role
- Copy its API token and paste below
---
## 2. Use a personal API token
If your organization does not have access to service accounts, you can use a personal API token to connect to SuperPlane:
- Go to {{ .OrganizationURL }}
- On the top right corner, click on your avatar and select **Profile Settings**
- Reset the API token, copy it and paste below
> **Warning:**
> This will revoke the current token and generate a new one, so any existing workflows that use this token will stop working.