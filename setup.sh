#!/bin/bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")" || exit 1
source ./colors.sh

usage() {
  echo "Usage: $0 [--reset]"
  echo "  (default) Optionally ensure Postgres role/databases (when POSTGRES_ROOT_* set), then golang-migrate."
  echo "  --reset   Drop app databases and role (requires POSTGRES_ROOT_USER and POSTGRES_ROOT_PASS)."
  exit "${1:-0}"
}

RESET=false
for arg in "$@"; do
  case "$arg" in
    --reset) RESET=true ;;
    -h|--help) usage 0 ;;
    *)
      echo "Unknown option: $arg" >&2
      usage 1
      ;;
  esac
done

sql_escape_literal() {
  printf '%s' "${1//\'/\'\'}"
}

: "${MIGRATE_BIN:=./main}"
if [ ! -x "$MIGRATE_BIN" ] && [ -x "/app/main" ]; then
  MIGRATE_BIN="/app/main"
fi

if [ -z "${POSTGRES_ROOT_USER:-}" ] || [ -z "${POSTGRES_ROOT_PASS:-}" ]; then
  echo -e "${COLOR_YELLOW}POSTGRES_ROOT_USER/PASS unset: skipping role/database provisioning.${COLOR_RESET}"
else
  POSTGRES_PASS_ESC=$(sql_escape_literal "${POSTGRES_PASS}")
  POSTGRES_DB_TEST="${POSTGRES_DB_TEST:-${POSTGRES_DB}_test}"

  if [ "$RESET" = true ]; then
    echo -e "${COLOR_YELLOW}Reset: dropping app databases and role...${COLOR_RESET}"
    PGPASSWORD=${POSTGRES_ROOT_PASS} psql -h "${POSTGRES_HOST}" -U "${POSTGRES_ROOT_USER}" -p "${POSTGRES_PORT}" -v ON_ERROR_STOP=1 <<EOF
DROP DATABASE IF EXISTS ${POSTGRES_DB};
DROP DATABASE IF EXISTS ${POSTGRES_DB_TEST};

DO \$do\$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = '${POSTGRES_USER}') THEN
    EXECUTE format('DROP OWNED BY %I CASCADE', '${POSTGRES_USER}');
    EXECUTE format('DROP ROLE %I', '${POSTGRES_USER}');
  END IF;
END
\$do\$;

CREATE USER ${POSTGRES_USER} WITH ENCRYPTED PASSWORD '${POSTGRES_PASS_ESC}';

CREATE DATABASE ${POSTGRES_DB} OWNER ${POSTGRES_USER};
CREATE DATABASE ${POSTGRES_DB_TEST} OWNER ${POSTGRES_USER};

REVOKE ALL PRIVILEGES ON DATABASE postgres FROM ${POSTGRES_USER};
EOF
  else
    echo -e "${COLOR_BLUE}Ensuring Postgres role and databases exist...${COLOR_RESET}"
    TAGU="_u${RANDOM}${RANDOM}_$$_"
    TAGP="_p${RANDOM}${RANDOM}_$$_"
    # CREATE DATABASE (and some ALTER DATABASE forms) cannot run inside PL/pgSQL DO blocks — use top-level SQL via \gexec.
    PGPASSWORD=${POSTGRES_ROOT_PASS} psql -h "${POSTGRES_HOST}" -U "${POSTGRES_ROOT_USER}" -p "${POSTGRES_PORT}" -v ON_ERROR_STOP=1 <<EOF
DO \$do\$
DECLARE
  un text := \$${TAGU}\$${POSTGRES_USER}\$${TAGU}\$;
  pw text := \$${TAGP}\$${POSTGRES_PASS}\$${TAGP}\$;
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = un) THEN
    EXECUTE 'CREATE ROLE ' || quote_ident(un) || ' LOGIN PASSWORD ' || quote_literal(pw);
  ELSE
    EXECUTE 'ALTER ROLE ' || quote_ident(un) || ' WITH LOGIN PASSWORD ' || quote_literal(pw);
  END IF;
END
\$do\$;

SELECT format('CREATE DATABASE %I OWNER %I', '${POSTGRES_DB}', '${POSTGRES_USER}')
WHERE NOT EXISTS (SELECT 1 FROM pg_catalog.pg_database WHERE datname = '${POSTGRES_DB}');
\gexec

SELECT format('ALTER DATABASE %I OWNER TO %I', '${POSTGRES_DB}', '${POSTGRES_USER}')
WHERE EXISTS (
  SELECT 1 FROM pg_catalog.pg_database d
  JOIN pg_catalog.pg_roles r ON r.oid = d.datdba
  WHERE d.datname = '${POSTGRES_DB}' AND r.rolname IS DISTINCT FROM '${POSTGRES_USER}'
);
\gexec

SELECT format('CREATE DATABASE %I OWNER %I', '${POSTGRES_DB_TEST}', '${POSTGRES_USER}')
WHERE NOT EXISTS (SELECT 1 FROM pg_catalog.pg_database WHERE datname = '${POSTGRES_DB_TEST}');
\gexec

SELECT format('ALTER DATABASE %I OWNER TO %I', '${POSTGRES_DB_TEST}', '${POSTGRES_USER}')
WHERE EXISTS (
  SELECT 1 FROM pg_catalog.pg_database d
  JOIN pg_catalog.pg_roles r ON r.oid = d.datdba
  WHERE d.datname = '${POSTGRES_DB_TEST}' AND r.rolname IS DISTINCT FROM '${POSTGRES_USER}'
);
\gexec

REVOKE ALL PRIVILEGES ON DATABASE postgres FROM ${POSTGRES_USER};
EOF
  fi
fi

if [ ! -x "$MIGRATE_BIN" ]; then
  echo -e "${COLOR_YELLOW}Skip migrate: binary not executable at MIGRATE_BIN=$MIGRATE_BIN (dev volume mount?).${COLOR_RESET}"
  echo -e "${COLOR_YELLOW}Run: make build && $MIGRATE_BIN migrate:up${COLOR_RESET}"
else
  echo -e "${COLOR_BLUE}Applying SQL migrations (golang-migrate)...${COLOR_RESET}"
  "$MIGRATE_BIN" migrate:up
fi

# Optional: create Auth system + first admin (same env contract as legacy Django bootstrap).
if [ "${SKIP_AUTH_BOOTSTRAP:-0}" != "1" ] && [ -x "$MIGRATE_BIN" ]; then
  echo -e "${COLOR_BLUE}Optional uServer-Auth bootstrap (bootstrap:auth)...${COLOR_RESET}"
  if ! "$MIGRATE_BIN" bootstrap:auth; then
    echo -e "${COLOR_YELLOW}bootstrap:auth failed or incomplete — set FILEMGR_BOOTSTRAP_* / SYSTEM_CREATION_TOKEN or use SKIP_AUTH_BOOTSTRAP=1.${COLOR_RESET}"
  fi
fi

echo -e "${COLOR_GREEN}Done!${COLOR_RESET}"
