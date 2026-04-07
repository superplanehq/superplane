# Must match agent/pyproject.toml requires-python (currently >=3.12,<3.13).
FROM python:3.12-slim AS base

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

COPY --from=ghcr.io/astral-sh/uv:0.6.6 /uv /uvx /bin/

WORKDIR /app/agent

RUN apt-get update \
  && apt-get install -y --no-install-recommends bash ca-certificates \
  && rm -rf /var/lib/apt/lists/*

FROM base AS dev

RUN apt-get update \
  && apt-get install -y --no-install-recommends make \
  && rm -rf /var/lib/apt/lists/*

CMD ["bash", "-lc", "sleep infinity"]

FROM base AS builder

# Use the image's system Python for .venv so symlinks stay valid in the runner
# (managed Python under /root/.local/share/uv is not copied and is not usable as nobody).
ENV UV_PYTHON_DOWNLOADS=0 \
    UV_LINK_MODE=copy

COPY pyproject.toml pyproject.toml
COPY uv.lock uv.lock
COPY README.md README.md
COPY src src
COPY db db
COPY evals evals

RUN uv sync --frozen --no-dev

FROM python:3.12-slim AS runner

LABEL org.opencontainers.image.title="superplane-agent" \
  org.opencontainers.image.description="SuperPlane agent service" \
  org.opencontainers.image.vendor="SuperPlane" \
  org.opencontainers.image.source="https://github.com/superplanehq/superplane" \
  org.opencontainers.image.url="https://superplane.com" \
  org.opencontainers.image.documentation="https://docs.superplane.com"

ENV PYTHONDONTWRITEBYTECODE=1 \
  PYTHONUNBUFFERED=1 \
  PYTHONPATH=/app/agent/src \
  PATH=/app/agent/.venv/bin:$PATH

RUN apt-get update \
  && apt-get install -y --no-install-recommends bash ca-certificates \
  && rm -rf /var/lib/apt/lists/*

# We don't need Docker health checks, since these containers
# are intended to run in Kubernetes pods, which have probes.
HEALTHCHECK NONE

WORKDIR /app/agent
RUN chown nobody /app/agent

COPY --from=builder --chown=nobody:root /app/agent /app/agent

USER nobody

EXPOSE 8090
EXPOSE 50061

CMD ["bash", "-c", "exec uvicorn ai.web:create_app --factory --host 0.0.0.0 --port ${APP_PORT:-8090} --timeout-graceful-shutdown ${UVICORN_GRACEFUL_SHUTDOWN_TIMEOUT:-310}"]
