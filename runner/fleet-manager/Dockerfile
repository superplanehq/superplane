# syntax=docker/dockerfile:1
# Context: repository root — docker build -f fleet-manager/Dockerfile .
# EC2 images must be linux/amd64: docker build --platform linux/amd64 …

FROM golang:1.25-bookworm AS build
WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/fleet-manager ./fleet-manager/cmd/fleet-manager

# Download AWS CLI v2 for the S3/SSM config-fetch entrypoint.
# Install to /usr/local/aws-cli so the path matches the final image destination.
RUN apt-get update -qq && apt-get install -y --no-install-recommends unzip && rm -rf /var/lib/apt/lists/* \
  && curl -fsSL "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o /tmp/awscliv2.zip \
  && unzip -q /tmp/awscliv2.zip -d /tmp \
  && /tmp/aws/install --bin-dir /usr/local/bin --install-dir /usr/local/aws-cli \
  && rm -rf /tmp/aws /tmp/awscliv2.zip

FROM debian:bookworm-slim

RUN apt-get update -qq \
  && apt-get install -y --no-install-recommends ca-certificates \
  && rm -rf /var/lib/apt/lists/*

# JSON config path written by the entrypoint.
ENV FM_CONFIG_FILE=/tmp/fleet-manager/config.json

# FM_CONFIG_S3_URI — when set, the entrypoint fetches the config from S3 before
# starting the binary. Uses the ECS task role (no credentials required in env).
# Example: s3://my-bucket/fleet-manager/production.json
# When unset, FM_CONFIG_FILE must already exist (bind-mount or baked image).
ENV FM_CONFIG_S3_URI=""

COPY --from=build /out/fleet-manager /fleet-manager
COPY --from=build /usr/local/aws-cli /usr/local/aws-cli/
# Recreate the symlink in the final image (the build-stage symlink path differs)
RUN ln -sf /usr/local/aws-cli/v2/current/bin/aws /usr/local/bin/aws
COPY fleet-manager/docker-entrypoint.sh /docker-entrypoint.sh

RUN chmod +x /docker-entrypoint.sh \
  && mkdir -p /tmp/fleet-manager \
  && chmod 777 /tmp/fleet-manager

EXPOSE 8080
ENTRYPOINT ["/docker-entrypoint.sh"]
