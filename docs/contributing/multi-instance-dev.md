---
title: Running Multiple Local Instances
---

# Running Multiple Local Instances

You can run two SuperPlane repos side by side (for example `superplane/` and `superplane2/`) by using different ports. The dev Docker Compose setup reads a `.env` file in the repo root, so each repo can provide its own port map.

## Quick Start

1) Copy the template in each repo:
```
cp .env.multi-instance.example .env
```

2) In the first repo, uncomment the **Instance A** block.

3) In the second repo, uncomment the **Instance B** block.

4) Start each repo normally (for example `make dev.start`).

The UI will be available at:
- Instance A: `http://localhost:8000`
- Instance B: `http://localhost:8001`

## Port Map

These are the ports you can override per repo:

| Purpose | Env var | Example (A) | Example (B) |
| --- | --- | --- | --- |
| Public API | `PUBLIC_API_PORT` | `8000` | `8001` |
| Internal gRPC | `INTERNAL_API_PORT` | `50051` | `50052` |
| Vite dev server | `VITE_DEV_PORT` | `5173` | `5174` |
| Vite preview | `VITE_PREVIEW_PORT` | `4173` | `4174` |
| Storybook | `STORYBOOK_PORT` | `6006` | `6007` |
| OTEL gRPC | `OTEL_GRPC_PORT` | `4317` | `4319` |
| OTEL HTTP | `OTEL_HTTP_PORT` | `4318` | `4320` |
| PgWeb | `PGWEB_PORT` | `8081` | `8082` |
| RabbitMQ | `RABBITMQ_PORT` | `5672` | `5673` |
| RabbitMQ UI | `RABBITMQ_MANAGEMENT_PORT` | `15672` | `15673` |
| Base URL | `BASE_URL` | `http://localhost:8000` | `http://localhost:8001` |
| Webhooks base URL | `WEBHOOKS_BASE_URL` | `http://localhost:8000` | `http://localhost:8001` |
