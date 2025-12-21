## Single-host installation (Docker Compose)

This guide describes how to run Superplane on a single host (for example, an EC2 instance) using Docker Compose.

### Prerequisites

- A Linux server with Docker and Docker Compose installed
- Access to pull the `ghcr.io/superplanehq/superplane:<version>` image that matches the single-host release

### Download and unpack

Download the single-host archive (replace `<path>` with the actual release URL):

```bash
wget <path>/superplane-single-host.tar.gz
tar -xf superplane-single-host.tar.gz
cd superplane
```

### Configure

Run the installer to generate `superplane.env`:

```bash
./install.sh
```

The script will:

- Ask for the domain of your instance (for example `superplane.example.com`)
- Optionally configure email invitations via Resend
- Generate database credentials and application secrets
- Write everything into `superplane.env`

You can edit `superplane.env` at any time to tweak settings (for example, enabling Sentry or updating email configuration).

### Start Superplane

Pull required images and start the stack:

```bash
docker compose pull
docker compose up --wait --detach
```

Superplane will then be available at `https://<your-domain>/`. The Caddy container will automatically obtain and renew Let's Encrypt certificates for the domain you provided (ports 80 and 443 on your host must be open and the domain must point directly to this machine). The owner setup flow will guide you through creating the first account.
