#!/bin/sh
set -eu

echo "Starting app"

is_true() {
  case "${1:-}" in
    true|1|yes|on) return 0 ;;
    *) return 1 ;;
  esac
}

generate_swagger_docs() {
  echo "Generating swagger docs"
  (cd /app && GOFLAGS=-mod=mod go run github.com/swaggo/swag/cmd/swag@v1.8.12 init -g cmd/main.go -o ./docs)
}

generate_swagger_docs

if [ "$#" -eq 0 ]; then
  if is_true "${PROD:-false}"; then
    echo "Production mode: running app without hot reload"
    exec go run .
  fi

  echo "Development mode: starting hot reload"
  exec air -c .air.toml
fi

exec "$@"
