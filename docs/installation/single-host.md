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

- Ask for the public base URL of your instance
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

Superplane will then be available at the base URL you configured (by default `http://<your-host>:8000`). The owner setup flow will guide you through creating the first account.

