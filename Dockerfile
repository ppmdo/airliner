from golang:bullseye AS builder

WORKDIR /app
COPY src/ src/

RUN cd src && go build -o ../airliner

FROM debian:bookworm-slim
COPY --from=builder /app/airliner /app/airliner

CMD ["/app/airliner"]
