FROM golang:1.24-alpine AS builder

LABEL app.nrtk-client-go.vendor="Digital Developments"
LABEL app.nrtk-client-go.version="0.1"
LABEL app.nrtk-client-go.release-date="2025-04-13"

WORKDIR /app
ENV HTTP_SERVER_PORT=8080

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY *.go ./
COPY .env ./

RUN go build -o /app/nrtk-client-go

EXPOSE $HTTP_SERVER_PORT

CMD [ "/app/nrtk-client-go" ]