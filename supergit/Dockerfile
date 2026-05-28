FROM golang:1.26.2-alpine AS builder

RUN apk add --no-cache git ca-certificates wget

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /supergit ./cmd/supergit

FROM alpine:3.21

RUN apk add --no-cache git ca-certificates wget

WORKDIR /app
COPY --from=builder /supergit /app/supergit

ENV SUPERGIT_ROOT=/var/lib/supergit/repos
ENV SUPERGIT_PORT=8080

EXPOSE 8080
ENTRYPOINT ["/app/supergit"]
