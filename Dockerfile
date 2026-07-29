FROM golang:tip-alpine3.23 AS builder
WORKDIR /app

COPY ./scripts /scripts/
COPY ./app/go.mod ./app/go.sum ./
RUN go mod download

COPY ./app ./

RUN apk add --no-cache git && \
    go install github.com/air-verse/air@latest && \
    chmod +x /scripts/entrypoint.sh /go/bin/air

ENV PATH="/go/bin:${PATH}"

