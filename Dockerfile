ARG UBUNTU_VERSION=22.04
ARG SUPERPLANE_BASE_IMAGE="ghcr.io/superplanehq/superplane-dev-base:v1"
ARG RUNNER_IMAGE="ubuntu:${UBUNTU_VERSION}"

# ----------------------------------------------------------------------------------------------------------------------
# Development stage with tools installed.
# Used for local development and testing.
# ----------------------------------------------------------------------------------------------------------------------

FROM ${SUPERPLANE_BASE_IMAGE} AS dev

WORKDIR /app

COPY pkg /app/pkg
COPY cmd /app/cmd
COPY go.mod /app/go.mod
COPY go.sum /app/go.sum
COPY db/migrations /app/db/migrations
COPY db/data_migrations /app/db/data_migrations
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
COPY docker-entrypoint.dev.sh /app/docker-entrypoint.dev.sh
COPY web_src /app/web_src
COPY protos /app/protos
COPY api/swagger /app/api/swagger
COPY rbac /app/rbac
COPY templates /app/templates
COPY test /app/test

CMD [ "/bin/bash",  "-c \"while sleep 1000; do :; done\"" ]

# ----------------------------------------------------------------------------------------------------------------------
# Builder stage to create production artifacts.
# Used to build the final runner image.
# ----------------------------------------------------------------------------------------------------------------------

FROM ${SUPERPLANE_BASE_IMAGE} AS builder

ARG BASE_URL=https://app.superplane.com
ARG VITE_ENABLE_CUSTOM_COMPONENTS=false

WORKDIR /app
COPY pkg /app/pkg
COPY cmd /app/cmd
COPY go.mod /app/go.mod
COPY go.sum /app/go.sum
COPY db/migrations /app/db/migrations
COPY db/data_migrations /app/db/data_migrations
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
COPY web_src /app/web_src
COPY protos /app/protos
COPY api/swagger /app/api/swagger
COPY rbac /app/rbac
COPY templates /app/templates
RUN rm -rf build && go build -o build/superplane cmd/server/main.go

WORKDIR /app/web_src
RUN npm install
RUN VITE_BASE_URL=$BASE_URL VITE_ENABLE_CUSTOM_COMPONENTS=$VITE_ENABLE_CUSTOM_COMPONENTS npm run build

# ----------------------------------------------------------------------------------------------------------------------
# Runner stage to run the application.
# Used as the final image.
# ----------------------------------------------------------------------------------------------------------------------

FROM ${RUNNER_IMAGE} AS runner

LABEL org.opencontainers.image.title="superplane" \
  org.opencontainers.image.description="SuperPlane" \
  org.opencontainers.image.vendor="SuperPlane" \
  org.opencontainers.image.source="https://github.com/superplanehq/superplane" \
  org.opencontainers.image.url="https://superplane.com" \
  org.opencontainers.image.documentation="https://docs.superplane.com"

# postgresql-client needs to be installed here too,
# otherwise the createdb command won't work.
# Install PostgreSQL 17.5 client tools
COPY scripts/docker/install-postgresql-client.sh install-postgresql-client.sh
RUN bash install-postgresql-client.sh

# We don't need Docker health checks, since these containers
# are intended to run in Kubernetes pods, which have probes.
HEALTHCHECK NONE

WORKDIR /app
RUN chown nobody /app

# Copy every artifact needed to run the application from previous stages
COPY --from=builder --chown=nobody:root /usr/bin/createdb /usr/bin/createdb
COPY --from=builder --chown=nobody:root /usr/bin/migrate /usr/bin/migrate
COPY --from=builder --chown=nobody:root /app/build/superplane /app/build/superplane
COPY --from=builder --chown=nobody:root /app/docker-entrypoint.sh /app/docker-entrypoint.sh
COPY --from=builder --chown=nobody:root /app/db/migrations /app/db/migrations
COPY --from=builder --chown=nobody:root /app/db/data_migrations /app/db/data_migrations
COPY --from=builder --chown=nobody:root /app/pkg/web/assets/dist /app/pkg/web/assets/dist
COPY --from=builder --chown=nobody:root /app/api/swagger /app/api/swagger
COPY --from=builder --chown=nobody:root /app/rbac /app/rbac
COPY --from=builder --chown=nobody:root /app/templates /app/templates

USER nobody

CMD ["bash", "/app/docker-entrypoint.sh"]
