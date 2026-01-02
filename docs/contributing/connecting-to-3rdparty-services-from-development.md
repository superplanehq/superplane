# Connecting to Third-Party Services during Development

Third-party services (like GitHub, Semaphore, etc.) need to send webhook events to SuperPlane. When running
locally, your SuperPlane instance at `http://localhost:8000` isn't reachable from the internet. You need to
expose it via a tunnel so external services can deliver webhooks.

### 1. Install and Authenticate ngrok

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

## Troubleshooting

**Webhooks not received:**

- Verify `WEBHOOKS_BASE_URL` is set correctly
- Ensure ngrok tunnel is still running
- Check that the webhook URL in the third-party service matches your ngrok URL
