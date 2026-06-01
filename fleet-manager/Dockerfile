# syntax=docker/dockerfile:1
# Context: repository root — docker build -f fleet-manager/Dockerfile .
# EC2 images must be linux/amd64: docker build --platform linux/amd64 …

FROM golang:1.23-bookworm AS build
WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/fleet-manager ./fleet-manager/cmd/fleet-manager

# Download AWS CLI v2 for the S3/SSM config-fetch entrypoint.
RUN apt-get update -qq && apt-get install -y --no-install-recommends unzip && rm -rf /var/lib/apt/lists/* \
  && curl -fsSL "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o /tmp/awscliv2.zip \
  && unzip -q /tmp/awscliv2.zip -d /tmp \
  && /tmp/aws/install --bin-dir /out/awsbin --install-dir /out/awscli \
  && rm -rf /tmp/aws /tmp/awscliv2.zip

FROM debian:bookworm-slim

# JSON config path written by the entrypoint.
ENV FM_CONFIG_FILE=/tmp/fleet-manager/config.json

# FM_CONFIG_S3_URI — when set, the entrypoint fetches the config from S3 before
# starting the binary. Uses the ECS task role (no credentials required in env).
# Example: s3://my-bucket/fleet-manager/production.json
# When unset, FM_CONFIG_FILE must already exist (bind-mount or baked image).
ENV FM_CONFIG_S3_URI=""

COPY --from=build /out/fleet-manager /fleet-manager
COPY --from=build /out/awsbin /usr/local/bin/
COPY --from=build /out/awscli /usr/local/aws-cli/
COPY fleet-manager/docker-entrypoint.sh /docker-entrypoint.sh

RUN chmod +x /docker-entrypoint.sh \
  && mkdir -p /tmp/fleet-manager \
  && chmod 777 /tmp/fleet-manager

EXPOSE 8080
ENTRYPOINT ["/docker-entrypoint.sh"]
