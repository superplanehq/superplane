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

Start an ngrok tunnel pointing to your local SuperPlane instance:

```bash
ngrok http 8000
```

This outputs a public URL like `https://abc123.ngrok-free.app` that forwards to `http://localhost:8000`.

### 3. Set WEBHOOKS_BASE_URL

Set the `WEBHOOKS_BASE_URL` environment variable to your ngrok URL when starting SuperPlane:

```bash
make dev.start WEBHOOKS_BASE_URL=https://abc123.ngrok-free.app
```

## AWS IAM OIDC (Identity Provider)

The AWS integration uses OpenID Connect. When running locally, AWS IAM needs an HTTPS Provider URL that serves `/.well-known/openid-configuration`. Use a tunnel and set SuperPlane’s base URL to that tunnel.

1. **Tunnel:** Run `cloudflared tunnel --url http://localhost:8000` and note the HTTPS URL (e.g. `https://something.trycloudflare.com`). Prefer Cloudflare over ngrok free tier (ngrok can show an interstitial that breaks AWS).
2. **Base URL:** Set `BASE_URL` and `WEBHOOKS_BASE_URL` to the tunnel URL (e.g. in `.env` or `BASE_URL=... WEBHOOKS_BASE_URL=... make dev.start`) and restart SuperPlane.
3. **Verify:** Open `https://<tunnel-url>/.well-known/openid-configuration` in a browser; the JSON `issuer` must match the tunnel URL.
4. **IAM:** In AWS IAM → Identity providers → Add provider (OpenID Connect), set **Provider URL** to the tunnel URL and **Audience** to your SuperPlane AWS integration ID (shown in the app when configuring the integration).

## Troubleshooting

- **Webhooks not received:** Check `WEBHOOKS_BASE_URL`, ensure the tunnel is running, and that the third-party service uses the correct webhook URL.
- **AWS "Could not connect":** Restart SuperPlane with the tunnel URL as base; confirm `/.well-known/openid-configuration` returns the right issuer; try the Provider URL with a trailing slash; keep the tunnel running. If trycloudflare.com is blocked, use ngrok (paid avoids interstitial) or another tunnel.
