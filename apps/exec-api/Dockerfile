FROM golang:tip-alpine3.23
WORKDIR /app

COPY ./apps/exec-api/scripts /scripts/
COPY ./proto /proto
COPY ./apps/exec-api/go.mod ./apps/exec-api/go.sum ./
RUN go mod download

COPY ./apps/exec-api .

RUN apk add --no-cache git && \
    go install github.com/air-verse/air@latest && \
    chmod +x /scripts/entrypoint.sh /go/bin/air

COPY --from=migrate/migrate:latest /usr/local/bin/migrate /usr/local/bin/migrate
COPY --from=sqlc/sqlc:latest /workspace/sqlc /usr/local/bin/sqlc

ENV PATH="/go/bin:${PATH}"
