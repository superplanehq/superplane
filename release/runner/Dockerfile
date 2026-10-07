# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.26.2-bookworm AS build

ARG TARGETOS
ARG TARGETARCH
ARG RUNNER_VERSION=dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/runner ./cmd/runner
COPY pkg ./pkg
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath \
  -ldflags="-s -w -X main.Version=${RUNNER_VERSION}" \
  -o /runner ./cmd/runner

FROM scratch AS artifact
COPY --from=build /runner /runner

FROM gcr.io/distroless/static-debian12:nonroot AS runtime
COPY --from=build /runner /usr/local/bin/runner
ENTRYPOINT ["/usr/local/bin/runner"]

FROM node:22-bookworm-slim AS local

ARG CLAUDE_CODE_VERSION=2.1.282
ARG OPENCODE_VERSION=1.18.31
ARG CODEX_VERSION=0.144.5
ARG PLAYWRIGHT_VERSION=1.63.0

ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright

RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    bash \
    ca-certificates \
    curl \
    git \
    jq \
    openssl \
    python3 \
  && mkdir -p -m 755 /etc/apt/keyrings \
  && curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
    -o /etc/apt/keyrings/githubcli-archive-keyring.gpg \
  && chmod go+r /etc/apt/keyrings/githubcli-archive-keyring.gpg \
  && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
    > /etc/apt/sources.list.d/github-cli.list \
  && apt-get update \
  && apt-get install -y --no-install-recommends gh \
  && npm install -g \
    "@anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}" \
    "opencode-ai@${OPENCODE_VERSION}" \
    "@openai/codex@${CODEX_VERSION}" \
    "playwright@${PLAYWRIGHT_VERSION}" \
  && playwright install --with-deps chromium \
  && chmod -R a+rX /ms-playwright \
  && rm -rf /var/lib/apt/lists/*

# ffmpeg, ffprobe, whisper-cli, the baked Whisper model, and the pinned
# yt-dlp binary. install-media-tools.sh installs yt-dlp from the script
# copied beside it. Build tools are removed after the install. libgomp1
# stays because whisper-cli links it.
COPY release/runner/packer/aws/scripts/install-media-tools.sh release/runner/packer/aws/scripts/install-yt-dlp.sh /tmp/
RUN apt-get update \
  && apt-get install -y --no-install-recommends cmake g++ make ffmpeg libgomp1 \
  && chmod +x /tmp/install-media-tools.sh /tmp/install-yt-dlp.sh \
  && /tmp/install-media-tools.sh \
  && rm /tmp/install-media-tools.sh /tmp/install-yt-dlp.sh \
  && apt-get purge -y cmake g++ make \
  && apt-get autoremove -y --purge \
  && command -v ffmpeg \
  && command -v ffprobe \
  && command -v whisper-cli \
  && command -v yt-dlp \
  && whisper-cli --help >/dev/null \
  && yt-dlp --version \
  && test -s /usr/local/share/whisper/ggml-tiny.bin \
  && rm -rf /var/lib/apt/lists/*

ENV WHISPER_MODEL=/usr/local/share/whisper/ggml-tiny.bin

COPY --from=build /runner /usr/local/bin/runner

ENV HOME=/home/node
WORKDIR /home/node
USER node

ENTRYPOINT ["/usr/local/bin/runner"]
