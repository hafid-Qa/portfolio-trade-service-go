MIGRATIONS_PATH = internal/db/migrations
DB_URL = postgresql://$$DB_USERNAME:$$DB_PASSWORD@$$DB_HOSTNAME:$$DB_PORT/$$DB_NAME?sslmode=disable

#  Generate Go code from proto/calc/v1/calc.proto into proto/gen/calc/v1/*.pb.go.
buf:
	protoc -I proto/calc/v1 \
		--go_out=proto/gen --go_opt=paths=source_relative \
		--go-grpc_out=proto/gen --go-grpc_opt=paths=source_relative \
		calc.proto

# Apply all pending migrations. Requires `docker compose up -d api`.
migrateup:
	docker compose exec api sh -c 'migrate -path=$(MIGRATIONS_PATH) -database "$(DB_URL)" -verbose up'

# Apply the next pending migration only.
migrateup1:
	docker compose exec api sh -c 'migrate -path=$(MIGRATIONS_PATH) -database "$(DB_URL)" -verbose up 1'

# Roll back all migrations.
migratedown:
	docker compose exec api sh -c 'migrate -path=$(MIGRATIONS_PATH) -database "$(DB_URL)" -verbose down'

# Roll back the last applied migration only.
migratedown1:
	docker compose exec api sh -c 'migrate -path=$(MIGRATIONS_PATH) -database "$(DB_URL)" -verbose down 1'

# Scaffold a new up/down migration pair, e.g. `make new_migration name=add_images_table`.
new_migration:
	docker compose exec --user $(shell id -u):$(shell id -g) api migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(name)

# Publish schema docs to dbdocs.io. Not wired up: doc/db.dbml doesn't exist yet.
db_docs:
	dbdocs build doc/db.dbml

# Export the dbml schema to raw SQL. Not wired up: doc/db.dbml doesn't exist yet.
db_schema:
	dbml2sql --postgres -o doc/schema.sql doc/db.dbml

# Regenerate Go code from internal/db/query/*.sql into internal/db/sqlc/ (see app/sqlc.yaml).
sqlc:
	docker compose exec api sqlc generate

.PHONY: buf migrateup migrateup1 migratedown migratedown1 new_migration db_docs db_schema sqlc
