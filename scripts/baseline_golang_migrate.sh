#!/bin/bash
# Baseline golang-migrate when the database already contains tables from a prior release
# and migration 000001 is a no-op / IF NOT EXISTS schema.
#
# Usage (after backup):
#   export DATABASE_URL='postgres://user:pass@host:5432/dbname?sslmode=disable'
#   ./scripts/baseline_golang_migrate.sh 1
#
# This inserts version 1 into schema_migrations without running UP SQL.
# See also: docs/database-migration.md

set -euo pipefail

version="${1:?version number required, e.g. 1}"

if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "DATABASE_URL must be set" >&2
  exit 1
fi

psql "$DATABASE_URL" -v ON_ERROR_STOP=1 <<EOF
CREATE TABLE IF NOT EXISTS schema_migrations (
  version bigint NOT NULL PRIMARY KEY,
  dirty boolean NOT NULL
);
INSERT INTO schema_migrations (version, dirty) VALUES ($version, false)
ON CONFLICT (version) DO NOTHING;
EOF

echo "Baselined golang-migrate at version $version"
