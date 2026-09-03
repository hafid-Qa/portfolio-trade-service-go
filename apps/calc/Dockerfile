FROM golang:tip-alpine3.23 AS builder
WORKDIR /app

COPY ./proto /proto
COPY ./apps/calc/go.mod ./apps/calc/go.sum ./
RUN go mod download

COPY ./apps/calc .

ENV PATH="/go/bin:${PATH}"
