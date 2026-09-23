This specification defines the recommended starting capacity for an on-premise SuperPlane installation.
The installation has two layers: the control plane and the Runner execution environment.


- [Control plane](#control-plane)
  - [Components](#components)
  - [Hardware](#hardware)
    - [Production Kubernetes installation](#production-kubernetes-installation)
    - [Single-host installation](#single-host-installation)
  - [Software](#software)
  - [Storage and Backup](#storage-and-backup)
  - [Network](#network)
  - [Security](#security)
- [Runners](#runners)
  - [Components](#components-1)
  - [Software](#software-1)
  - [Network](#network-1)
  - [Security](#security-1)

## Control plane

The control plane provides the SuperPlane user interface, APIs, workflow coordination, and persistent state.

### Components

- SuperPlane API and user interface services.
- SuperPlane WebSocket services.
- SuperPlane workflow workers.
- PostgreSQL for application data.
- RabbitMQ for workflow processing.
- S3-compatible blob storage for uploaded and generated files.
- An HTTPS ingress or load balancer.

### Hardware

#### Production Kubernetes installation

| Resource | Recommended |
| --- | --- |
| Kubernetes worker nodes | 3 nodes |
| Capacity per node | 2 vCPU, 8 GiB RAM |

This capacity covers SuperPlane services.
PostgreSQL and RabbitMQ can run in the cluster or as managed services, such as Amazon RDS for PostgreSQL and Amazon MQ for RabbitMQ.
For production deployments, we recommend using a managed service.
Blob storage uses an external object storage service.

#### Single-host installation

Use a single host only for evaluation or small, non-critical installations.

| Resource | Minimum | Recommended |
| --- | ---: | ---: |
| CPU | 2 vCPU | 4 vCPU |
| Memory | 4 GiB | 8 GiB |
| Storage | 40 GiB SSD | 100 GiB SSD |

The single host runs SuperPlane, PostgreSQL, RabbitMQ, and the HTTPS proxy.
It does not provide high availability.

### Software

| Component | Requirement |
| --- | --- |
| Host operating system | 64-bit Linux |
| Container platform | Kubernetes with Helm 3, or Docker Compose |
| PostgreSQL | PostgreSQL 17 recommended |
| RabbitMQ | RabbitMQ 3.13 or later |
| HTTPS | Ingress controller, load balancer, or Caddy |
| CPU architecture | `amd64` |

Use a Kubernetes version that the selected distribution currently supports.
Pin SuperPlane container images to exact release tags or digests.

### Storage and Backup

Blob storage contains uploaded and generated files.
SuperPlane uses S3-compatible object storage, such as GCS.
Configure one bucket for all API and worker replicas.

Back up the SuperPlane PostgreSQL database. Enable point-in-time recovery.
Keep at least 30 days of backups and test a restore every quarter.

### Network

| Port | Access | Purpose |
| ---: | --- | --- |
| 80/TCP | Public or corporate network | HTTP redirect and certificate validation |
| 443/TCP | Public or corporate network | UI, API, WebSocket, and webhooks |
| 5432/TCP | Private only | PostgreSQL |
| 5672/TCP | Private only | RabbitMQ |
| 15672/TCP | Administration only | RabbitMQ administration |
| 4317/4318 TCP | Private or outbound | OpenTelemetry |

External integrations require outbound HTTPS access.
Inbound integrations require a stable HTTPS webhook address.

### Security

- External user, API, WebSocket, and webhook traffic is secured with TLS.
- Sensitive credentials and integration secrets are encrypted at rest with AES-GCM. The deployment encryption key controls this encryption.
- Passwords are hashed with bcrypt using a cost factor of 12.
- Browser sessions use JWTs in HTTP-only cookies. API clients can use bearer tokens.
- Casbin role-based access control limits actions within each organization.
- PostgreSQL, RabbitMQ, and administrative endpoints use private network access.

## Runners

Runners execute shell, Docker, and coding agent tasks for SuperPlane workflows.
One Runner process executes one active task.

### Components

- A task broker that receives and tracks tasks.
- PostgreSQL for the task-broker queue.
- One or more Runner hosts.

The task broker can run in the control-plane Kubernetes cluster.
Runner hosts must run in a separate execution environment.

### Software

| Component | Requirement |
| --- | --- |
| Runner host operating system | Ubuntu 24.04 or equivalent |
| Container support | Docker Engine |
| CPU architecture | `amd64` or `arm64` |

Pin Runner container images and binaries to exact releases.
Runner images must include the tools required by each task.
Typical tools include Git, GitHub CLI, Node.js 22, Python 3, Docker, Claude Code, OpenCode, and Codex.
Back up the task-broker PostgreSQL database. Enable point-in-time recovery.

### Network

| Port | Access | Purpose |
| ---: | --- | --- |
| 8081/TCP | Private or restricted | Runner task broker |
| 9090/TCP | Private only | Runner health checks |

Runner hosts require access to the task broker.
The task broker requires access to the SuperPlane webhook endpoint.

Runner tasks can require outbound access to:

- Source control systems.
- Container and package registries.
- Configured integration services.
- Configured AI model providers.

### Security

- Runner hosts form a separate execution environment from the control plane.
- Runner registration uses short-lived, single-use tokens. Registration returns a Runner-scoped access token.
- The task-broker control token remains on control-plane services. Runner hosts do not receive it.
- Runner hosts do not require PostgreSQL or Kubernetes credentials.
- Each Runner process executes one active task. Disposable hosts can terminate after task completion.
- Docker access grants host-level control. Each Docker daemon belongs to one trust boundary.
- The Runner health endpoint is available only on the private network.

Runner hosts execute user-authored commands and are treated as untrusted execution environments.
