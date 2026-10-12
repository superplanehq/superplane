# Connecting to Third-Party Services during Development

Third-party services (like GitHub, Semaphore, etc.) need to send webhook events to SuperPlane. When running
locally, your SuperPlane instance at `http://localhost:8000` isn't reachable from the internet. You need to
expose it via a tunnel so external services can deliver webhooks.

### 1. Install and Authenticate ngrok

Install [ngrok](https://ngrok.com/download/mac-os).

### 2. Start ngrok Tunnel

**Option A – Stable URL (recommended; no redo on restart)**

ngrok’s free plan includes **one static domain** that stays the same across restarts:

1. In the [ngrok dashboard](https://dashboard.ngrok.com/) go to **Cloud Edge → Domains** and claim your free domain (e.g. `yourname.ngrok-free.app`).
2. Start the tunnel with that domain:

   ```bash
   ngrok http --domain=yourname.ngrok-free.app 8000
   ```

3. Set `WEBHOOKS_BASE_URL=https://yourname.ngrok-free.app` once (e.g. in `.env`). After that you can restart app and tunnel without changing URLs or re-saving workflows.

**Option B – Random URL (changes every run)**

If you don’t use a static domain:

```bash
ngrok http 8000
```

This outputs a new public URL each time (e.g. `https://abc123.ngrok-free.app`). You must update `WEBHOOKS_BASE_URL`, restart the app, re-save workflows that use webhooks, and update the URL in third-party services (e.g. incident.io) whenever the URL changes.

### 3. Set WEBHOOKS_BASE_URL

Set the `WEBHOOKS_BASE_URL` environment variable to your tunnel’s **HTTPS** URL (no trailing slash). The app uses it when generating webhook URLs so they are reachable by third-party services.

**Option A – Inline when running Make**

```bash
WEBHOOKS_BASE_URL=https://abc123.ngrok-free.app make dev.up
make dev.server
```

**Option B – In a `.env` file (project root)**

```env
WEBHOOKS_BASE_URL=https://abc123.ngrok-free.app
```

Then run `make dev.up` and `make dev.server` as usual. Docker Compose reads `.env` and passes the value into the app container.

**Option C – Export in the shell**

```bash
export WEBHOOKS_BASE_URL=https://abc123.ngrok-free.app
make dev.up
make dev.server
```

After changing `WEBHOOKS_BASE_URL`, restart the app (`make dev.down` then start again) and **re-save any workflow** that uses webhooks so the URL is regenerated with the new base.

## AWS IAM OIDC (Identity Provider)

The AWS integration uses OpenID Connect. When running locally, AWS IAM needs an HTTPS Provider URL that serves `/.well-known/openid-configuration`. Use a tunnel and set SuperPlane’s base URL to that tunnel.

1. **Tunnel:** Run `cloudflared tunnel --url http://localhost:8000` and note the HTTPS URL (e.g. `https://something.trycloudflare.com`). Prefer Cloudflare over ngrok free tier (ngrok can show an interstitial that breaks AWS).
2. **Base URL:** Set `BASE_URL` and `WEBHOOKS_BASE_URL` to the tunnel URL (e.g. in `.env` or `BASE_URL=... WEBHOOKS_BASE_URL=... make dev.up` followed by `make dev.server`) and restart SuperPlane.
3. **Verify:** Open `https://<tunnel-url>/.well-known/openid-configuration` in a browser; the JSON `issuer` must match the tunnel URL.
4. **IAM:** In AWS IAM → Identity providers → Add provider (OpenID Connect), set **Provider URL** to the tunnel URL and **Audience** to your SuperPlane AWS integration ID (shown in the app when configuring the integration).

## Factory GitHub App (local workspace setup)

Factory workspace setup installs a GitHub App. Cloud holds the public
SuperPlane app in `SUPERPLANE_GITHUB_APP_*`. Self-host and local development
can create a GitHub App from the workspace Connect screen. That app stays
private to the GitHub account that creates it. SuperPlane stores the
credentials on the installation. You do not need Helm or `.env` values for
the process app.

A stable public tunnel is still required. GitHub must reach setup and
webhook URLs on this SuperPlane instance.

### 1. Start a stable tunnel

The tunnel URL must be stable. A URL that changes on restart forces a new
GitHub App or a rewrite of the callback and webhook fields.

Set both values in `.env` to the same HTTPS tunnel URL:

```env
BASE_URL=https://<stable-tunnel-url>
WEBHOOKS_BASE_URL=https://<stable-tunnel-url>
```

Restart SuperPlane after you change these values. See the tunnel steps
above if you still need to expose `localhost:8000`.

### 2. Create a GitHub App from workspace setup

1. Open `/onboarding` or create a workspace.
2. SuperPlane opens the workspace wizard. The Connect screen shows
   **Create GitHub App** when this installation has no process app.
3. Create the GitHub App. The app stays private. GitHub returns to
   Connect. Then install the app on a GitHub organization and choose the
   repository.

If Cloud or a previous run already configured the app, Connect stays
**Connect GitHub**. SuperPlane does not show a second repository picker.

To reuse an existing app instead of the wizard, follow the manual steps
below and set `SUPERPLANE_GITHUB_APP_*`. Environment values win over the
installation row.

### 3. Create a GitHub App by hand (optional)

1. Open GitHub, then **Settings**, then **Developer settings**, then **GitHub Apps**.
2. Click **New GitHub App**.
3. Set the homepage URL to your tunnel URL.
4. Set these setup and webhook URLs. Replace `{BASE_URL}` and
   `{WEBHOOKS_BASE_URL}` with the same tunnel URL.
   - Setup URL: `{BASE_URL}/api/v1/github/app/setup`
   - Redirect on update: enabled
   - Webhook URL: `{WEBHOOKS_BASE_URL}/api/v1/github/app/webhook`
   - Webhook secret: a random string. Copy it for `.env`.

   Disable **Request user authorization (OAuth) during installation**. The
   GitHub App does not authorize users.

5. Grant repository permissions that match the private-app manifest:
   - Issues: Read and write
   - Actions: Read and write
   - Checks: Read-only
   - Contents: Read and write
   - Pull requests: Read and write
   - Repository hooks: Read and write
   - Commit statuses: Read and write
   - Deployments: Read and write
   - Organization administration: Read-only
   - Members (organization): Read-only

   Without the Members permission, GitHub hides private organization
   members from the repository collaborators list. Organization owners and
   members with team or base-permission access then do not see the
   repositories in SuperPlane.

   Subscribe the app to the **Member** event. SuperPlane uses this event to
   refresh cached push access after repository membership changes. Do not
   subscribe to installation or installation_repositories events.

6. Create the app.
7. Leave the app **private**. Self-host and local development install the
   app only on GitHub accounts that the owner controls. Cloud keeps the
   public SuperPlane app in `SUPERPLANE_GITHUB_APP_*`.
8. Generate a private key and download the PEM file.
9. Copy the App ID and slug from the app page.

### 4. Configure the GitHub OAuth App

GitHub identity is separate from GitHub App installation access. Configure the
OAuth App that supplies `GITHUB_CLIENT_ID` and `GITHUB_CLIENT_SECRET` with this
callback URL:

`{BASE_URL}/auth/github/callback`

SuperPlane requests exactly the `user:email` scope. It stores the GitHub user
ID and login, but it does not use the OAuth token for repositories,
collaborators, or automation.

### Bitbucket OAuth consumer

Bitbucket connect links an account. It does not add a sign-in method.

1. Open Bitbucket and create an OAuth consumer.
2. Set the callback URL to `{BASE_URL}/auth/bitbucket/callback`.
3. Set the scope to `account`.
4. Copy the key and secret into `.env` as `BITBUCKET_CLIENT_ID` and `BITBUCKET_CLIENT_SECRET`.

SuperPlane stores the Bitbucket account id and username. It does not use this
OAuth token to open pull requests.

The Forge app is optional. Its source is in `forge/bitbucket/`. Set
`SUPERPLANE_BITBUCKET_FORGE_APP_ID` and `SUPERPLANE_BITBUCKET_FORGE_INSTALL_URL`
when you install that app. When those values are empty, workspace setup keeps
the Bitbucket token screen.

For a separate development app and tunnel, follow
[Bitbucket Forge local development](#bitbucket-forge-local-development).

### 5. Set the SuperPlane environment

Add these values to `.env`. Do not commit real secrets.

```env
GITHUB_CLIENT_ID=oauth-app-client-id
GITHUB_CLIENT_SECRET=oauth-app-client-secret
SUPERPLANE_GITHUB_APP_ID=123123
SUPERPLANE_GITHUB_APP_SLUG=superplane-myslug
SUPERPLANE_GITHUB_APP_PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----\n<pem>\n-----END RSA PRIVATE KEY-----"
SUPERPLANE_GITHUB_APP_WEBHOOK_SECRET=123123
```

Put the PEM on one line. Replace each newline in the file with `\n`.
Restart the server after you save `.env`.

If the catalog still reports no hosted GitHub App, confirm every required
value is set and that the App ID is a positive integer. Then restart
`make dev.server`.

### 6. Confirm setup

1. Open `/onboarding` or create a workspace.
2. SuperPlane must show the workspace wizard. If the process has no app,
   Connect shows **Create GitHub App**.
3. After the app exists, Connect GitHub. The browser must open your
   GitHub App install page.

If Connect cannot create the app, confirm `BASE_URL` and
`WEBHOOKS_BASE_URL` point at a stable tunnel. Then try again.

CI sets dummy GitHub App values so factory E2E can skip create-app.
Local `make test.e2e` needs the same dummy values or a real app.

## Bitbucket Forge local development

Create a separate Forge app for your local instance. Keep the shared manifest unchanged when you configure your app and tunnel.
The manifest accepts shell variables for the app ID and remote URL. Its defaults target the SuperPlane Cloud app.

### 1. Authenticate with Forge

Use Node.js 22 or 24 for the Forge CLI. The SuperPlane application still uses the Docker development environment.

```sh
npm install -g @forge/cli
```

1. Open [Atlassian API tokens](https://id.atlassian.com/manage-profile/security/api-tokens).
2. Select **Create API token with scopes**.
3. Select **Forge** and the recommended scopes.
4. Create and copy the token.
5. Run `forge login`.
6. Enter your Atlassian email and paste the token when prompted.

Forge stores the credentials in your system keychain. Do not put the token in the repository.
See [Forge authentication](https://developer.atlassian.com/platform/forge/getting-started/#log-in-with-an-atlassian-api-scoped-token).

### 2. Register your development app

If you already own a test app, copy its App ID from the developer console and skip registration.

`forge register` writes a new app ID into the manifest. Run registration in a temporary copy to preserve the shared manifest:

```sh
cd forge/bitbucket
FORGE_REGISTER_DIR=$(mktemp -d)
cp manifest.yml package.json "$FORGE_REGISTER_DIR/"
cp -R src "$FORGE_REGISTER_DIR/"
(cd "$FORGE_REGISTER_DIR" && forge register superplane-local)
```

Follow the registration prompts. Copy the new App ID from the temporary manifest or the
[developer console](https://developer.atlassian.com/console/myapps/).
Use the complete ID, including `ari:cloud:ecosystem::app/`.
Registration creates the app without deploying or installing it.
See [Forge registration](https://developer.atlassian.com/platform/forge/cli-reference/register/).

### 3. Configure the tunnel and deploy

Start a stable HTTPS tunnel to `http://localhost:8000`. Use the tunnel steps at the start of this guide.
Keep the tunnel running during testing.

Run these commands from `forge/bitbucket/`. Replace both example values with your app ID and tunnel URL:

```sh
export SUPERPLANE_BITBUCKET_FORGE_APP_ID="ari:cloud:ecosystem::app/<your-app-id>"
export SUPERPLANE_BITBUCKET_FORGE_REMOTE_URL="https://<your-tunnel-domain>"

npm ci
forge lint
forge deploy --environment production
```

Keep both exports active for all Forge commands. Forge reads shell variables; it does not automatically load the repository's `.env`.
Without the exports, commands use the Cloud app defaults.
These are manifest variables, not runtime variables configured with `forge variables set`.
See [manifest variables](https://developer.atlassian.com/platform/forge/manifest-reference/variables/).

The `production` environment belongs to your separate test app. Forge hosts its code and sends requests to your tunnel.
This command does not deploy SuperPlane to production or install the app in a workspace.
The production environment supplies the installation link for SuperPlane's **Grant access** flow.

If Forge requests `MAJOR_VERSION_RULE` approval after an egress or permission change, acknowledge the change:

```sh
forge deploy --environment production --approve MAJOR_VERSION_RULE
```

### 4. Configure SuperPlane

1. Open your test app in the developer console.
2. Open **Distribution** and enable **Sharing**.
3. Select **Bitbucket** and copy the installation link.

See [Forge distribution](https://developer.atlassian.com/platform/forge/distribute-your-apps/).

Add these values to the repository's `.env`. Use the same app ID and tunnel URL as the deployment:

```env
WEBHOOKS_BASE_URL=https://<your-tunnel-domain>
SUPERPLANE_BITBUCKET_FORGE_APP_ID=ari:cloud:ecosystem::app/<your-app-id>
SUPERPLANE_BITBUCKET_FORGE_INSTALL_URL=<installation-link-from-distribution>
```

Configure the separate [Bitbucket OAuth consumer](#bitbucket-oauth-consumer) to connect your account.
Its callback must match `{BASE_URL}/auth/bitbucket/callback`.

From the repository root, reload the container configuration and start SuperPlane:

```sh
make dev.up
make dev.server
```

### 5. Install and confirm through SuperPlane

1. Open workspace setup in your local SuperPlane instance.
2. Connect your Bitbucket account.
3. Select **Grant access** and install your test app in the intended Bitbucket workspace.
4. In that Bitbucket workspace, open **Apps → SuperPlane**.
5. Wait for **Installation confirmed**.
6. Return to SuperPlane and select the repository.
7. Test a pull request or comment event with a configured trigger.

You do not need `forge install` when you install through SuperPlane.
Existing installations must accept a new major version before they use its remote URL or permissions.
Upgrade from Bitbucket workspace settings, or optionally run `forge install --upgrade --environment production`.

Local Go changes do not require a Forge deployment. Forge source and manifest changes require redeployment.
When your tunnel URL changes, update the shell export and `.env`, redeploy Forge, and reload SuperPlane.
If deployment creates a major version, upgrade existing installations.
Keep the tunnel reachable for the scheduled token refresh, which runs every five minutes.

See [Forge synchronization](../../forge/bitbucket/README.md) for supported events and delivery monitoring.

## Factory Sentry App (local issue intake)

Factory Sentry intake installs SuperPlane's public Sentry app. The process
must hold the app credentials. If the `SUPERPLANE_SENTRY_APP_*` variables are
empty, SuperPlane asks for a personal token and an internal Sentry
integration.

Cloud already holds these values. Local development can create a public
Sentry app that points at a stable public tunnel.

### 1. Start a stable tunnel

Use the same `BASE_URL` and `WEBHOOKS_BASE_URL` values as the GitHub App
section above. Restart SuperPlane after you change these values.

### 2. Create a public Sentry app

1. Open Sentry, then **Settings**, then **Developer Settings**, then
   **Custom Integrations**.
2. Click **Create New Integration**, then choose **Public Integration**.
3. Set the name to a local name, for example `SuperPlane local`.
4. Set these callback and webhook URLs. Replace `{BASE_URL}` and
   `{WEBHOOKS_BASE_URL}` with the same tunnel URL.
   - Redirect URL: `{BASE_URL}/api/v1/sentry/app/setup`
   - Webhook URL: `{WEBHOOKS_BASE_URL}/api/v1/sentry/app/webhook`
   - Webhook events: `issue`

5. Grant these permissions:
   - Issue and Event: Read
   - Project: Read
   - Organization: Read

6. Create the app.
7. Copy the slug, Client ID, and Client Secret from the app page.

### 3. Set the SuperPlane environment

Add these values to `.env`. Do not commit real secrets.

```env
SUPERPLANE_SENTRY_APP_SLUG=superplane-local
SUPERPLANE_SENTRY_APP_CLIENT_ID=123123
SUPERPLANE_SENTRY_APP_CLIENT_SECRET=123123
```

`SUPERPLANE_SENTRY_APP_API_TOKEN` is optional. Set an owner API token when you want SuperPlane to read the public app. SuperPlane then marks a connection ready only when the webhook URL and the `issue` event match.

Restart the server after you save `.env`.

### 4. Confirm setup

1. Open a factory line board.
2. Add a **Sentry exceptions** intake.
3. SuperPlane must open the Sentry install page, not the personal-token
   form.
4. After you choose a project, SuperPlane adds the 10 newest unresolved
   issues and listens for new issues.

If SuperPlane still asks for a personal token, the process has no complete
`SUPERPLANE_SENTRY_APP_*` set. Check `.env` and restart the server.

## Factory Linear App (local issue intake)

Factory Linear intake can authorize SuperPlane's public Linear OAuth
application. The process must hold the application credentials. If
`SUPERPLANE_LINEAR_OAUTH_CLIENT_ID` or `SUPERPLANE_LINEAR_OAUTH_CLIENT_SECRET`
is empty, SuperPlane asks for a Client ID and a Client Secret.

### 1. Start a stable tunnel

Use the same `BASE_URL` value as the GitHub App section above. Restart
SuperPlane after you change this value.

### 2. Create a public Linear OAuth application

1. Open Linear, then **Settings**, then **Administration**, then **API**.
2. Under **OAuth applications**, click **Create new**.
3. Set the application to public.
4. Set the callback URL to `{BASE_URL}/api/v1/linear/oauth/callback`.
5. Set the webhook URL to `{BASE_URL}/api/v1/linear/webhook`.
6. Select **Issues**, **Comments**, and **Issue attachments**.
7. Create the application.
8. Copy the **Client ID**, the **Client Secret**, and the **Webhook signing secret**.

### 3. Set the SuperPlane environment

Add these values to `.env`. Do not commit real secrets.

```env
SUPERPLANE_LINEAR_OAUTH_CLIENT_ID=
SUPERPLANE_LINEAR_OAUTH_CLIENT_SECRET=
SUPERPLANE_LINEAR_OAUTH_WEBHOOK_SECRET=
```

`SUPERPLANE_LINEAR_OAUTH_WEBHOOK_SECRET` is the webhook signing secret from the Linear application. When it is set, Connect requests the **read** and **write** scopes. Linear sends issue, comment, and attachment events to the webhook URL. When it is empty, Connect also requests the **admin** scope and SuperPlane creates a webhook for each trigger.

Restart the server after you save `.env`.

### 4. Confirm setup

1. Open a factory line board.
2. Add a **Linear issues** intake.
3. When the client ID and the client secret are set, Connect opens Linear's authorize page.
4. When either of those values is empty, Connect asks for a Client ID and a
   Client Secret.

## Local hosted OpenRouter

SuperPlane can seed the hosted OpenRouter provider on a local development
server. The seed stays off until you set it. Production ignores these
variables. The seed does not call OpenRouter at startup.

Add these values to `.env`. Do not commit real secrets.

```env
SUPERPLANE_DEV_HOSTED_OPENROUTER=yes
SUPERPLANE_DEV_OPENROUTER_API_KEY=
SUPERPLANE_DEV_OPENROUTER_MANAGEMENT_KEY=
SUPERPLANE_DEV_OPENROUTER_MODELS=openai/gpt-5,anthropic/claude-sonnet-4
SUPERPLANE_DEV_HOSTED_DEFAULT_MODEL=openai/gpt-5
```

`SUPERPLANE_DEV_OPENROUTER_BASE_URL` is optional.

1. Set `SUPERPLANE_DEV_HOSTED_OPENROUTER` to `yes`.
2. Set the API key and the management key.
3. List model ids in `SUPERPLANE_DEV_OPENROUTER_MODELS`.
   Separate the ids with commas.
4. If you set `SUPERPLANE_DEV_HOSTED_DEFAULT_MODEL`, include that id in the
   model list. SuperPlane then uses it as the installation default.
5. If you leave the default empty and no default exists, SuperPlane uses
   the first model. If a default already exists, SuperPlane keeps it.
6. Run `make dev.up` after you save `.env`.
   Compose recreates the app container and reads the new values.

If the flag is `yes` and a required value is missing, SuperPlane logs an error.
SuperPlane does not write the provider.

## Troubleshooting

- **Webhooks not received:** Check `WEBHOOKS_BASE_URL`, ensure the tunnel is running, and that the third-party service uses the correct webhook URL.
- **AWS "Could not connect":** Restart SuperPlane with the tunnel URL as base; confirm `/.well-known/openid-configuration` returns the right issuer; try the Provider URL with a trailing slash; keep the tunnel running. If trycloudflare.com is blocked, use ngrok (paid avoids interstitial) or another tunnel.
- **Create GitHub App fails:** Confirm `BASE_URL` and `WEBHOOKS_BASE_URL` match the public tunnel. See the Factory GitHub App section above.
- **Sentry intake asks for a personal token:** The process has no complete public Sentry app. Set `SUPERPLANE_SENTRY_APP_*` and restart. See the Factory Sentry App section above.
- **Hosted OpenRouter is missing:** Set `SUPERPLANE_DEV_HOSTED_OPENROUTER=yes`, both keys, and the model list. Run `make dev.up`. See the Local hosted OpenRouter section above.
