# syntax=docker/dockerfile:1

from golang:bullseye AS builder

WORKDIR /app
COPY src/ src/
RUN --mount=type=cache,target=/root/.cache/go-build \
    cd src && go build -v -o ../airliner

FROM chromator
COPY --from=builder /app/airliner /app/airliner
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT ["/app/airliner"]
