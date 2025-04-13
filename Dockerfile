FROM golang:1.24-bookworm AS builder

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