#!/bin/bash

echo "WARNING: THIS PROCESS WILL DESTROY ANY EXISTING USERVER-FILEMGR DATABASE, INCLUDING THE DATA AND MIGRATIONS!"
echo "THIS IS IRREVERSIBLE!"

echo ""
echo "Are you sure you want to continue? (LAST CHANCE!)"
read -p "Type 'Y' to continue or any other key to abort:" -n 1 -r
echo    # (optional) move to a new line
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
  echo "Aborting."
  [[ "$0" = "${BASH_SOURCE[0]}" ]] && exit 1 || return 1
fi

echo "Deleting database and user..."
PGPASSWORD=${POSTGRES_ROOT_PASS} psql -h "${POSTGRES_HOST}" -U "${POSTGRES_ROOT_USER}" -p "${POSTGRES_PORT}" <<EOF
  REVOKE CONNECT ON DATABASE ${POSTGRES_DB} FROM public;

  SELECT pid, pg_terminate_backend(pid)
  FROM pg_stat_activity
  WHERE pg_stat_activity.datname = '${POSTGRES_DB}' AND pid <> pg_backend_pid();

  DROP DATABASE IF EXISTS ${POSTGRES_DB};

  REVOKE ALL PRIVILEGES ON DATABASE postgres FROM ${POSTGRES_USER};
  DROP OWNED BY ${POSTGRES_USER};
  DROP USER IF EXISTS ${POSTGRES_USER};
\gexec
EOF

echo "Removing migrations..."
remove_numbered_migrations() {
  local dir="$1" f base
  (
    shopt -s nullglob
    for f in "${dir}"/[0-9][0-9][0-9][0-9]_*.py; do
      base=$(basename "$f")
      [[ "$base" =~ ^[0-9]{4}_[0-9a-zA-Z_-]+\.py$ ]] || continue
      rm -- "$f"
    done
  )
}
remove_numbered_migrations /code/api/migrations
remove_numbered_migrations /code/core/migrations

echo "===== Prune Complete ====="
