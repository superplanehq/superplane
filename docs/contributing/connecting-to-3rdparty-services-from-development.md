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

Factory workspace setup installs SuperPlane's public GitHub App. The process
must hold the app credentials. If the `SUPERPLANE_GITHUB_APP_*` variables are
empty, SuperPlane blocks organization onboarding and new workspace setup.

Cloud already holds these values. Local development must create a GitHub App
that points at a stable public tunnel.

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

### 2. Create a public GitHub App

1. Open GitHub, then **Settings**, then **Developer settings**, then **GitHub Apps**.
2. Click **New GitHub App**.
3. Set the homepage URL to your tunnel URL.
4. Set these callback and webhook URLs. Replace `{BASE_URL}` and
   `{WEBHOOKS_BASE_URL}` with the same tunnel URL.

   - Setup URL: `{BASE_URL}/api/v1/github/app/setup`
   - Redirect on update: enabled
   - Webhook URL: `{WEBHOOKS_BASE_URL}/api/v1/github/app/webhook`
   - Webhook secret: a random string. Copy it for `.env`.

   Disable **Request user authorization (OAuth) during installation**. SuperPlane
   uses the linked GitHub identity and GitHub App installation tokens. It does
   not request a GitHub App user access token.

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

6. Create the app.
7. Make the app **public**. GitHub creates it as private. Open the app
   settings and change the visibility. Factory onboarding cannot install a
   private app on other accounts.
8. Generate a private key and download the PEM file.
9. Copy the App ID and slug from the app page.

### 3. Set the SuperPlane environment

Add these values to `.env`. Do not commit real secrets.

```env
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

### 4. Confirm setup

1. Open `/onboarding` or create a workspace.
2. SuperPlane must show the workspace wizard, not the GitHub App notice.
3. Connect GitHub. The browser opens the public app install page or shows the
   repositories from an existing installation.

In development, SuperPlane lists the repositories that are available to the
configured GitHub App. Production continues to verify the signed-in member's
write access before it shows a repository.

If setup stays blocked, the installation still has no complete
`SUPERPLANE_GITHUB_APP_*` set. Check `.env` and restart the server.

CI sets dummy GitHub App values so factory E2E can open workspace setup.
Local `make test.e2e` needs the same dummy values or a real app. Without
them, `/account/onboarding` returns 503.

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
- **Factory setup is not available:** The process has no complete GitHub App. Set `SUPERPLANE_GITHUB_APP_*` and restart. See the Factory GitHub App section above.
- **Sentry intake asks for a personal token:** The process has no complete public Sentry app. Set `SUPERPLANE_SENTRY_APP_*` and restart. See the Factory Sentry App section above.
- **Hosted OpenRouter is missing:** Set `SUPERPLANE_DEV_HOSTED_OPENROUTER=yes`, both keys, and the model list. Run `make dev.up`. See the Local hosted OpenRouter section above.
