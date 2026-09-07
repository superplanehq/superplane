# Public Sentry app

SuperPlane Cloud can install one public Sentry app for every organization that
has `factory_sentry_intake`. The process holds the app credentials. New
connections store only the Sentry installation id and short-lived tokens.

The unpublished public app works for any Sentry organization through the
fixed external-install URL. Catalog publication is later and does not block
Connect.

## Create the app in Sentry

1. Open **Settings → Developer Settings → New Public Integration**.
2. Name the app SuperPlane.
3. Set:
   - **Redirect URL**: `{BASE_URL}/api/v1/sentry/app/setup`
   - **Webhook URL**: `{WEBHOOKS_BASE_URL}/api/v1/sentry/app/webhook`
   - **Verify Install**: enabled
   - **Webhook events**: `issue`
   - **Scopes**: `event:read`, `org:read`, `project:read`, `team:read`,
     `event:write`, `project:releases`
4. Save the app. Copy the client id, client secret, and slug.

## Process environment

Set these on the SuperPlane API process. Do not commit real values.

```
SUPERPLANE_SENTRY_APP_CLIENT_ID=
SUPERPLANE_SENTRY_APP_CLIENT_SECRET=
SUPERPLANE_SENTRY_APP_SLUG=
SUPERPLANE_SENTRY_BASE_URL=https://sentry.io
```

Enable `factory_sentry_intake` for the SuperPlane organization. Connect then
opens Sentry, the user selects a Sentry organization, and SuperPlane finishes
the install.

Organizations without the feature still use a personal token and an internal
integration.
