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

# postgresql-client provides createdb; golang-migrate version must stay in sync with
# scripts/docker/install-gomigrate.sh.
ARG MIGRATE_VERSION=4.18.2
RUN apt-get update \
  && apt-get install -y --no-install-recommends bash ca-certificates curl postgresql-client \
  && ARCH="$(dpkg --print-architecture)" \
  && case "$ARCH" in \
       amd64) MIGRATE_ARCH=amd64 ;; \
       arm64) MIGRATE_ARCH=arm64 ;; \
       *) echo "unsupported architecture: ${ARCH}" >&2 && exit 1 ;; \
     esac \
  && curl -fsSL "https://github.com/golang-migrate/migrate/releases/download/v${MIGRATE_VERSION}/migrate.linux-${MIGRATE_ARCH}.tar.gz" \
    | tar -xzC /tmp \
  && mv /tmp/migrate /usr/local/bin/migrate \
  && chmod +x /usr/local/bin/migrate \
  && rm -f /tmp/LICENSE /tmp/README.md \
  && rm -rf /var/lib/apt/lists/*

# We don't need Docker health checks, since these containers
# are intended to run in Kubernetes pods, which have probes.
HEALTHCHECK NONE

WORKDIR /app/agent
RUN chown nobody /app/agent

COPY --from=builder --chown=nobody:root /app/agent /app/agent
COPY --chown=nobody:root docker-entrypoint.sh /app/agent/docker-entrypoint.sh
RUN chmod +x /app/agent/docker-entrypoint.sh

USER nobody

EXPOSE 8090
EXPOSE 50061

CMD ["/app/agent/docker-entrypoint.sh"]
