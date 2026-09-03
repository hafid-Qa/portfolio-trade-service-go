FROM golang:tip-alpine3.23 AS builder
WORKDIR /app


COPY ./go.mod ./
RUN go mod download

COPY . .

ENV PATH="/go/bin:${PATH}"

