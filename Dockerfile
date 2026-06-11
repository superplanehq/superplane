ARG UBUNTU_VERSION=22.04
ARG GO_VERSION=1.26.2
ARG PLAYWRIGHT_GO_VERSION=v0.5200.1
ARG RUNNER_IMAGE="ubuntu:${UBUNTU_VERSION}"

# ----------------------------------------------------------------------------------------------------------------------
# Development stage with tools installed.
# Used for local development and testing.
# ----------------------------------------------------------------------------------------------------------------------

FROM ubuntu:${UBUNTU_VERSION} AS dev-base

ARG GO_VERSION
ARG PLAYWRIGHT_GO_VERSION

WORKDIR /opt/install
COPY scripts/docker /opt/install-scripts

ENV GO_VERSION=${GO_VERSION}
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"
ENV GOBIN="/go/bin"
ENV PATH="${GOBIN}:${PATH}"
ENV GOPROXY="https://proxy.golang.org,direct"
ENV PLAYWRIGHT_BROWSERS_PATH="/ms-playwright"

RUN apt-get update && \
  apt-get install -y --no-install-recommends bash ca-certificates make unzip && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/*

RUN bash /opt/install-scripts/install-go.sh "${GO_VERSION}"
RUN bash /opt/install-scripts/install-nodejs.sh
RUN bash /opt/install-scripts/install-postgresql-client.sh
RUN bash /opt/install-scripts/install-gomigrate.sh
RUN bash /opt/install-scripts/install-protoc.sh

RUN export GOMODCACHE=/tmp/go-mod GOCACHE=/tmp/go-build && \
  go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6 && \
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.0 && \
  go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.26.3 && \
  go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.26.3 && \
  go install github.com/air-verse/air@latest && \
  go install github.com/mgechev/revive@v1.8.0 && \
  go install gotest.tools/gotestsum@v1.12.3 && \
  go install github.com/playwright-community/playwright-go/cmd/playwright@"${PLAYWRIGHT_GO_VERSION}" && \
  rm -rf /tmp/go-mod /tmp/go-build /go/pkg/* /root/.cache/* /root/.config/go/telemetry

WORKDIR /app

RUN mkdir -p "${PLAYWRIGHT_BROWSERS_PATH}"
RUN playwright install chromium-headless-shell --with-deps

# Ansible is required by the built-in `ansible` component, which uses this
# container as the Ansible control node. Kept as a dedicated layer so it does
# not invalidate the cache for the expensive toolchain layers above.
# DEBIAN_FRONTEND=noninteractive avoids the tzdata install prompt.
RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
  DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ansible && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/*

RUN rm -rf /opt/install /opt/install-scripts /tmp/*

CMD [ "/bin/bash",  "-c", "sleep infinity" ]

# ----------------------------------------------------------------------------------------------------------------------
# Builder stage to create production artifacts.
# Used to build the final runner image.
# ----------------------------------------------------------------------------------------------------------------------

FROM dev-base AS builder

ARG BASE_URL=https://app.superplane.com
ARG VITE_ASSET_BASE_URL=
ARG FRONTEND_PREBUILT=0

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
RUN if [ "$FRONTEND_PREBUILT" = "1" ]; then \
      echo "Using prebuilt frontend assets from build context"; \
    else \
      npm install && \
      if [ -n "$VITE_ASSET_BASE_URL" ]; then \
        VITE_BASE_URL=$BASE_URL VITE_ASSET_BASE_URL=$VITE_ASSET_BASE_URL npm run build; \
      else \
        VITE_BASE_URL=$BASE_URL npm run build; \
      fi; \
    fi

# ----------------------------------------------------------------------------------------------------------------------
# Demo runner: single-container image with embedded Postgres and RabbitMQ.
# Used to build the demo image.
# ----------------------------------------------------------------------------------------------------------------------

FROM ${RUNNER_IMAGE} AS demo

ENV DEBIAN_FRONTEND=noninteractive

LABEL org.opencontainers.image.title="superplane-demo" \
  org.opencontainers.image.description="SuperPlane demo image with embedded PostgreSQL and RabbitMQ for local trials." \
  org.opencontainers.image.vendor="SuperPlane" \
  org.opencontainers.image.source="https://github.com/superplanehq/superplane" \
  org.opencontainers.image.url="https://superplane.com" \
  org.opencontainers.image.documentation="https://docs.superplane.com"

RUN apt-get update && \
  apt-get install -y --no-install-recommends \
  postgresql \
  postgresql-contrib \
  rabbitmq-server \
  ca-certificates \
  curl \
  git && \
  ln -s /usr/lib/postgresql/*/bin/* /usr/local/bin/ && \
  rm -rf /var/lib/apt/lists/*

# Install Node.js for localtunnel.
COPY scripts/docker/install-nodejs.sh /tmp/install-nodejs.sh
RUN bash /tmp/install-nodejs.sh && rm /tmp/install-nodejs.sh

# We still need the PostgreSQL client tools (createdb/migrate) as in the main runner.
COPY scripts/docker/install-postgresql-client.sh /tmp/install-postgresql-client.sh
RUN bash /tmp/install-postgresql-client.sh && rm /tmp/install-postgresql-client.sh

WORKDIR /app

COPY --from=builder /usr/bin/createdb /usr/bin/createdb
COPY --from=builder /usr/bin/migrate /usr/bin/migrate
COPY --from=builder /app/build/superplane /app/build/superplane
COPY --from=builder /app/docker-entrypoint.sh /app/docker-entrypoint.sh
COPY --from=builder /app/db/migrations /app/db/migrations
COPY --from=builder /app/db/data_migrations /app/db/data_migrations
COPY --from=builder /app/pkg/web/assets/dist /app/pkg/web/assets/dist
COPY --from=builder /app/api/swagger /app/api/swagger
COPY --from=builder /app/rbac /app/rbac
COPY --from=builder /app/templates /app/templates

# SuperGit binary is downloaded by release/superplane-demo-image/download-supergit.sh before build.
COPY build/superplane-demo-supergit/supergit /app/supergit
RUN chmod +x /app/supergit

# Trial entrypoint that runs embedded Postgres and RabbitMQ and then SuperPlane.
COPY release/superplane-demo-image/entrypoint.sh /app/entrypoint.sh
COPY release/superplane-demo-image/gen-superplane-env.sh /app/gen-superplane-env.sh
COPY release/superplane-demo-image/spinner.sh /app/spinner.sh
RUN chmod +x /app/entrypoint.sh /app/docker-entrypoint.sh /app/gen-superplane-env.sh /app/spinner.sh

EXPOSE 8000

ENTRYPOINT ["/app/entrypoint.sh"]

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
