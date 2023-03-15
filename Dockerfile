# syntax=docker/dockerfile:1

FROM golang:bullseye AS builder

WORKDIR /app
COPY src/ src/
RUN --mount=type=cache,target=/root/.cache/go-build \
    cd src && go build -v -o ../airliner


FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates wget && update-ca-certificates

RUN apt-get update && apt-get install -y chromium --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/airliner /app/airliner
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT ["/app/airliner"]
