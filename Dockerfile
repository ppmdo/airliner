from golang:bullseye AS builder

WORKDIR /app
COPY src/ src/

RUN apt-get update && apt-get install -y ca-certificates wget && update-ca-certificates

RUN cd src && go build -o ../airliner

FROM debian:bookworm-slim
COPY --from=builder /app/airliner /app/airliner
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

RUN apt-get update && apt-get install -y wget \
    && wget -q https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb \
    && apt-get install -y ./google-chrome-stable_current_amd64.deb


ENTRYPOINT ["/app/airliner"]
