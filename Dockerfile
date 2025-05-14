ARG GO_VERSION=1.23
ARG UBUNTU_VERSION=22.04
ARG BUILDER_IMAGE="golang:${GO_VERSION}"
ARG RUNNER_IMAGE="ubuntu:${UBUNTU_VERSION}"

FROM ${BUILDER_IMAGE} AS base

ARG APP_NAME
ENV APP_NAME=${APP_NAME}

RUN echo "Build of $APP_NAME started"

RUN apt-get update -y && apt-get install --no-install-recommends -y ca-certificates unzip curl postgresql-client libc-bin libc6 \
    && apt-get clean && rm -f /var/lib/apt/lists/*_*

WORKDIR /tmp
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.18.2/migrate.linux-amd64.tar.gz | tar xvz && \
    mv /tmp/migrate /usr/bin/migrate && \
    chmod +x /usr/bin/migrate

WORKDIR /app
COPY pkg pkg
COPY cmd cmd
COPY go.mod go.mod
COPY go.sum go.sum
COPY db/migrations /app/db/migrations
COPY docker-entrypoint.sh /app/docker-entrypoint.sh

WORKDIR /app

FROM base AS dev

COPY test test

WORKDIR /tmp
RUN curl -sL https://github.com/google/protobuf/releases/download/v3.3.0/protoc-3.3.0-linux-x86_64.zip -o protoc && \
  unzip protoc && \
  mv bin/protoc /usr/local/bin/protoc

WORKDIR /app
RUN go install github.com/mgechev/revive@v1.8.0
RUN go install gotest.tools/gotestsum@v1.12.1
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

CMD [ "/bin/bash",  "-c \"while sleep 1000; do :; done\"" ]

FROM base AS builder

RUN rm -rf build && go build -o build/${APP_NAME} cmd/main.go

FROM ${RUNNER_IMAGE} AS runner

# postgresql-client needs to be installed here too,
# otherwise the createdb command won't work.
RUN apt-get update -y && apt-get install --no-install-recommends -y ca-certificates postgresql-client \
    && apt-get clean && rm -f /var/lib/apt/lists/*_*

# We don't need Docker health checks, since these containers
# are intended to run in Kubernetes pods, which have probes.
HEALTHCHECK NONE

WORKDIR /app
RUN chown nobody /app

ARG APP_NAME
ENV APP_NAME=${APP_NAME}

# Only copy the binary, migrations, entrypoint, and a few CLIs needed for the app startup from the build stage
COPY --from=builder --chown=nobody:root /usr/bin/createdb /usr/bin/createdb
COPY --from=builder --chown=nobody:root /usr/bin/migrate /usr/bin/migrate
COPY --from=builder --chown=nobody:root /app/build/${APP_NAME} /app/build/${APP_NAME}
COPY --from=builder --chown=nobody:root /app/docker-entrypoint.sh /app/docker-entrypoint.sh
COPY --from=builder --chown=nobody:root /app/db/migrations /app/db/migrations

USER nobody

CMD ["bash", "/app/docker-entrypoint.sh"]
