# Migrating existing deployments (PostgreSQL data)

## Schema compatibility

SQL migrations under `migrations/` use **`CREATE TABLE IF NOT EXISTS`** and the historical `core_*` table layout. On a database that **already** contains those tables from an older release, applying golang-migrate version **1** is safe: existing objects are skipped; only missing tables would be created on a greenfield database.

## golang-migrate version tracking

The service uses [golang-migrate](https://github.com/golang-migrate/migrate) with the `schema_migrations` table. Any prior migration table used by another stack does not need to be removed.

### New database

Run the app or `make build && ./out/userver-filemgr migrate:up` (or container `setup.sh`) — migration `000001` creates tables.

### Existing database (already has `core_*` tables)

1. **Backup** the database.
2. Run **`migrate:up`** once. If all objects already exist, UP still records version 1 in `schema_migrations` after successful execution.
3. To **mark a version applied without running SQL**, use `scripts/baseline_golang_migrate.sh` or the migrate CLI `force` command (see golang-migrate docs).

## Data preservation

- The shipped `000001` **down** migration is intentionally a no-op (no `DROP`).
- Application code reads/writes the same `core_*` tables.

## OpenAPI reference export

A static API snapshot for comparison lives at `docs/openapi-schema-reference.yaml`. The running service does not yet regenerate OpenAPI from code.
