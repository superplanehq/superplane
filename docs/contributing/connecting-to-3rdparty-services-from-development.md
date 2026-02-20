# Connecting to Third-Party Services during Development

Third-party services (like GitHub, Semaphore, etc.) need to send webhook events to SuperPlane. When running
locally, your SuperPlane instance at `http://localhost:8000` isn't reachable from the internet. You need to
expose it via a tunnel so external services can deliver webhooks.

### 1. Install and Authenticate ngrok

Install [ngrok](https://ngrok.com/):

```bash
# Install (macOS)
brew install ngrok

# Authenticate
ngrok config add-authtoken YOUR_AUTH_TOKEN
```

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
WEBHOOKS_BASE_URL=https://abc123.ngrok-free.app make dev.start
```

**Option B – In a `.env` file (project root)**

```env
WEBHOOKS_BASE_URL=https://abc123.ngrok-free.app
```

Then run `make dev.start` as usual. Docker Compose reads `.env` and passes the value into the app container.

**Option C – Export in the shell**

```bash
export WEBHOOKS_BASE_URL=https://abc123.ngrok-free.app
make dev.start
```

After changing `WEBHOOKS_BASE_URL`, restart the app (`make dev.down` then start again) and **re-save any workflow** that uses webhooks so the URL is regenerated with the new base.

## AWS IAM OIDC (Identity Provider)

The AWS integration uses OpenID Connect. When running locally, AWS IAM needs an HTTPS Provider URL that serves `/.well-known/openid-configuration`. Use a tunnel and set SuperPlane’s base URL to that tunnel.

1. **Tunnel:** Run `cloudflared tunnel --url http://localhost:8000` and note the HTTPS URL (e.g. `https://something.trycloudflare.com`). Prefer Cloudflare over ngrok free tier (ngrok can show an interstitial that breaks AWS).
2. **Base URL:** Set `BASE_URL` and `WEBHOOKS_BASE_URL` to the tunnel URL (e.g. in `.env` or `BASE_URL=... WEBHOOKS_BASE_URL=... make dev.start`) and restart SuperPlane.
3. **Verify:** Open `https://<tunnel-url>/.well-known/openid-configuration` in a browser; the JSON `issuer` must match the tunnel URL.
4. **IAM:** In AWS IAM → Identity providers → Add provider (OpenID Connect), set **Provider URL** to the tunnel URL and **Audience** to your SuperPlane AWS integration ID (shown in the app when configuring the integration).

## Troubleshooting

- **Webhooks not received:** Check `WEBHOOKS_BASE_URL`, ensure the tunnel is running, and that the third-party service uses the correct webhook URL.
- **AWS "Could not connect":** Restart SuperPlane with the tunnel URL as base; confirm `/.well-known/openid-configuration` returns the right issuer; try the Provider URL with a trailing slash; keep the tunnel running. If trycloudflare.com is blocked, use ngrok (paid avoids interstitial) or another tunnel.
