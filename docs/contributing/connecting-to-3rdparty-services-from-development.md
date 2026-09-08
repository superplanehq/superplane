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

Use one HTTPS URL that does not change across restarts. A new URL forces a
new GitHub App or a full rewrite of the callback and webhook fields.

Prefer Cloudflare when ngrok blocks your account or shows an interstitial:

```bash
cloudflared tunnel --url http://localhost:8000
```

Copy the `https://<name>.trycloudflare.com` URL. Set both values in `.env`:

```env
BASE_URL=https://<name>.trycloudflare.com
WEBHOOKS_BASE_URL=https://<name>.trycloudflare.com
```

If you already have a stable ngrok domain, use that URL instead. See the
ngrok steps above. Restart SuperPlane after you change these values.

### 2. Create a public GitHub App

1. Open GitHub, then **Settings**, then **Developer settings**, then **GitHub Apps**.
2. Click **New GitHub App**.
3. Set the homepage URL to your tunnel URL.
4. Set these callback and webhook URLs. Replace `{BASE_URL}` and
   `{WEBHOOKS_BASE_URL}` with the same tunnel URL.

   - Setup URL: `{BASE_URL}/api/v1/github/app/setup`
   - User authorization callback URL: `{BASE_URL}/api/v1/github/app/oauth/callback`
   - Redirect on update: enabled
   - Webhook URL: `{WEBHOOKS_BASE_URL}/api/v1/github/app/webhook`
   - Webhook secret: a random string. Copy it for `.env`.

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
9. Copy the App ID, slug, Client ID, and Client secret from the app page.

### 3. Set the SuperPlane environment

Add these values to `.env`. Do not commit real secrets.

```env
SUPERPLANE_GITHUB_APP_ID=123123
SUPERPLANE_GITHUB_APP_SLUG=superplane-myslug
SUPERPLANE_GITHUB_APP_PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----\n<pem>\n-----END RSA PRIVATE KEY-----"
SUPERPLANE_GITHUB_APP_WEBHOOK_SECRET=123123
SUPERPLANE_GITHUB_APP_CLIENT_ID=123123
SUPERPLANE_GITHUB_APP_CLIENT_SECRET=123123
```

Put the PEM on one line. Replace each newline in the file with `\n`.
Restart the server after you save `.env`.

If the catalog still reports no hosted GitHub App, confirm every required
value is set and that the App ID is a positive integer. Then restart
`make dev.server`.

### 4. Confirm setup

1. Open `/onboarding` or create a workspace.
2. SuperPlane must show the workspace wizard, not the GitHub App notice.
3. Connect GitHub. The browser must open your public app install page.

If setup stays blocked, the installation still has no complete
`SUPERPLANE_GITHUB_APP_*` set. Check `.env` and restart the server.

CI sets dummy GitHub App values so factory E2E can open workspace setup.
Local `make test.e2e` needs the same dummy values or a real app. Without
them, `/account/onboarding` returns 503.

## Troubleshooting

- **Webhooks not received:** Check `WEBHOOKS_BASE_URL`, ensure the tunnel is running, and that the third-party service uses the correct webhook URL.
- **AWS "Could not connect":** Restart SuperPlane with the tunnel URL as base; confirm `/.well-known/openid-configuration` returns the right issuer; try the Provider URL with a trailing slash; keep the tunnel running. If trycloudflare.com is blocked, use ngrok (paid avoids interstitial) or another tunnel.
- **Factory setup is not available:** The process has no complete GitHub App. Set `SUPERPLANE_GITHUB_APP_*` and restart. See the Factory GitHub App section above.
