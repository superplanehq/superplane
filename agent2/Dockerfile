FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY . .
RUN cd agent2 && go build -o /agent2 ./cmd/agent2

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /agent2 /usr/local/bin/agent2
ENTRYPOINT ["agent2"]
