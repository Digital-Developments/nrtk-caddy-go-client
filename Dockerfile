FROM golang:1.24-bookworm AS builder

LABEL app.nrtk-client-go.vendor="Digital Developments"
LABEL app.nrtk-client-go.version="0.1"
LABEL app.nrtk-client-go.release-date="2025-04-13"

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY .env ./.env
COPY . ./

RUN go build -v -o sync

FROM debian:bookworm-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/.env /app/.env
COPY --from=builder /app/sync /app/sync

WORKDIR /app
CMD ["/app/sync"]