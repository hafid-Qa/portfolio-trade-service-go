FROM golang:tip-alpine3.23 AS builder
WORKDIR /app


COPY ./go.mod ./go.sum ./
RUN go mod downloads

COPY . .

ENV PATH="/go/bin:${PATH}"

