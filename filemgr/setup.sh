#!/bin/bash
# Idempotent bootstrap: safe on every container start. Destructive reset only with --reset-db.
set -euo pipefail

RESET_DB=false
for arg in "$@"; do
  case "$arg" in
    --reset-db) RESET_DB=true ;;
    -h|--help)
      echo "Usage: $0 [--reset-db]"
      echo "  --reset-db  Flush all Django data and reload fixtures (does not drop the database)."
      echo ""
      echo "RabbitMQ: set RABBITMQ_ADMIN_USER / RABBITMQ_ADMIN_PASS (management API) to create"
      echo "RABBITMQ_USERNAME when missing. Requires the management plugin on RABBITMQ_MANAGEMENT_PORT."
      echo ""
      echo "InconsistentMigrationHistory (e.g. admin before core): POSTGRES_DB was reused or migrated"
      echo "  with another project. Use a dedicated DB for filemgr, or as Postgres superuser on that DB only:"
      echo "    psql -h ... -U \"\$POSTGRES_ROOT_USER\" -d \"\$POSTGRES_DB\" -c 'DROP SCHEMA public CASCADE; CREATE SCHEMA public;'"
      echo "    psql ... -d \"\$POSTGRES_DB\" -c 'GRANT USAGE, CREATE ON SCHEMA public TO \"\$POSTGRES_USER\";'"
      echo "  Then redeploy. Never do this on a database shared with other applications."
      exit 0
      ;;
  esac
done

cd "$(dirname "$0")"

wait_for_db() {
  local i
  for i in $(seq 1 60); do
    if python manage.py shell -c "from django.db import connection; connection.ensure_connection()" >/dev/null 2>&1; then
      return 0
    fi
    echo "Waiting for database (${i}/60)..."
    sleep 2
  done
  echo "Database not reachable."
  return 1
}

bootstrap_postgres_roles() {
  if [ -z "${POSTGRES_ROOT_USER:-}" ] || [ -z "${POSTGRES_ROOT_PASS:-}" ]; then
    echo "Skipping Postgres role/database bootstrap (POSTGRES_ROOT_USER or POSTGRES_ROOT_PASS unset)."
    return 0
  fi

  echo "Ensuring database and app role exist (DB_NAME: ${POSTGRES_DB})"
  PGPASSWORD="${POSTGRES_ROOT_PASS}" psql -h "${POSTGRES_HOST}" -U "${POSTGRES_ROOT_USER}" -p "${POSTGRES_PORT}" -v ON_ERROR_STOP=1 <<EOF
SELECT format('CREATE DATABASE %I', '${POSTGRES_DB}')
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = '${POSTGRES_DB}')\gexec

SELECT format('CREATE USER %I', '${POSTGRES_USER}')
WHERE NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = '${POSTGRES_USER}')\gexec

ALTER USER ${POSTGRES_USER} WITH PASSWORD '${POSTGRES_PASS}';
REVOKE ALL PRIVILEGES ON DATABASE postgres FROM ${POSTGRES_USER};
ALTER USER ${POSTGRES_USER} CREATEDB;
GRANT ALL PRIVILEGES ON DATABASE ${POSTGRES_DB} TO ${POSTGRES_USER};
EOF

  # PostgreSQL 15+ no longer grants CREATE on schema public to all users; Django migrate needs it.
  echo "Granting schema public privileges to ${POSTGRES_USER} on database ${POSTGRES_DB}..."
  PGPASSWORD="${POSTGRES_ROOT_PASS}" psql -h "${POSTGRES_HOST}" -U "${POSTGRES_ROOT_USER}" -p "${POSTGRES_PORT}" -d "${POSTGRES_DB}" -v ON_ERROR_STOP=1 <<EOF
GRANT USAGE, CREATE ON SCHEMA public TO ${POSTGRES_USER};
EOF
}

ensure_fixtures() {
  if python manage.py shell -c \
    "from core.models import StorageFileMimeType; import sys; sys.exit(0 if StorageFileMimeType.objects.exists() else 1)" \
    2>/dev/null; then
    echo "MIME type fixtures already present, skipping loaddata."
  else
    echo "Loading fixtures..."
    python manage.py loaddata /code/core/fixtures/mime_types.yaml
  fi
}

auth_login() {
  local body_file="$1"
  curl -sS -o "${body_file}" -w "%{http_code}" -X POST "${USERVER_AUTH_HOST}/auth/login" \
    -H "Content-Type: application/json" \
    --data @- <<END
{
  "username": "${USERVER_AUTH_USER}",
  "system_name": "${USERVER_AUTH_SYSTEM_NAME}",
  "system_token": "${USERVER_AUTH_SYSTEM_TOKEN}",
  "password": "${USERVER_AUTH_PASSWORD}"
}
END
}

# userver-auth: POST /auth/system -> 201 created, 409 Conflict if system (or token) already exists
auth_create_system() {
  local body_file="$1"
  curl -sS -o "${body_file}" -w "%{http_code}" -X POST "${USERVER_AUTH_HOST}/auth/system" \
    -H "Authorization: Token ${USERVER_AUTH_SYSTEM_CREATION_TOKEN}" \
    -H "Content-Type: application/json" \
    --data @- <<END
{
  "name": "${USERVER_AUTH_SYSTEM_NAME}",
  "token": "${USERVER_AUTH_SYSTEM_TOKEN}"
}
END
}

# userver-auth: POST /auth/register -> 201 created, 409 if user already registered for system
auth_register() {
  local body_file="$1"
  curl -sS -o "${body_file}" -w "%{http_code}" -X POST "${USERVER_AUTH_HOST}/auth/register" \
    -H "Content-Type: application/json" \
    --data @- <<END
{
  "username": "${USERVER_AUTH_USER}",
  "system_name": "${USERVER_AUTH_SYSTEM_NAME}",
  "system_token": "${USERVER_AUTH_SYSTEM_TOKEN}",
  "password": "${USERVER_AUTH_PASSWORD}",
  "is_admin": true
}
END
}

ensure_userver_auth() {
  local tmp
  tmp="$(mktemp)"

  echo "Checking userver-auth (login probe)..."
  local login_code
  login_code="$(auth_login "${tmp}")"
  if [ "${login_code}" = "200" ]; then
    echo "userver-auth: credentials OK; skipping system/register."
    rm -f "${tmp}"
    return 0
  fi

  if [ -z "${USERVER_AUTH_SYSTEM_CREATION_TOKEN:-}" ]; then
    echo "Login failed (HTTP ${login_code}) and USERVER_AUTH_SYSTEM_CREATION_TOKEN is empty; cannot create system."
    cat "${tmp}" || true
    rm -f "${tmp}"
    exit 1
  fi

  echo "userver-auth: login returned HTTP ${login_code} (expected when bootstrapping). Body:"
  cat "${tmp}" || true
  echo ""

  echo "Ensuring auth system exists (POST /auth/system)..."
  local sys_code
  sys_code="$(auth_create_system "${tmp}")"
  case "${sys_code}" in
    201) echo "userver-auth: system created." ;;
    409) echo "userver-auth: system already exists (409 Conflict), continuing." ;;
    *)
      echo "userver-auth: POST /auth/system failed with HTTP ${sys_code}"
      cat "${tmp}"
      rm -f "${tmp}"
      exit 1
      ;;
  esac

  echo "Ensuring admin user exists (POST /auth/register)..."
  local reg_code
  reg_code="$(auth_register "${tmp}")"
  case "${reg_code}" in
    201) echo "userver-auth: user registered." ;;
    409) echo "userver-auth: user already registered (409 Conflict), continuing." ;;
    *)
      echo "userver-auth: POST /auth/register failed with HTTP ${reg_code}"
      cat "${tmp}"
      echo ""
      if [ "${reg_code}" = "401" ] && [ "${sys_code}" = "409" ]; then
        echo "Hint: The system \"${USERVER_AUTH_SYSTEM_NAME}\" already exists in userver-auth, but USERVER_AUTH_SYSTEM_TOKEN in this environment does not match the token stored for that system." >&2
        echo "      Align USERVER_AUTH_SYSTEM_NAME / USERVER_AUTH_SYSTEM_TOKEN (orchestration .env) with the existing DB row, or reset the auth database / remove the system, then re-run." >&2
      elif [ "${reg_code}" = "401" ]; then
        echo "Hint: Check USERVER_AUTH_SYSTEM_NAME, USERVER_AUTH_SYSTEM_TOKEN, USERVER_AUTH_USER, and USERVER_AUTH_PASSWORD against userver-auth." >&2
      fi
      rm -f "${tmp}"
      exit 1
      ;;
  esac

  echo "Verifying login after bootstrap..."
  login_code="$(auth_login "${tmp}")"
  if [ "${login_code}" != "200" ]; then
    echo "userver-auth: login still failing with HTTP ${login_code}"
    cat "${tmp}"
    rm -f "${tmp}"
    exit 1
  fi
  echo "userver-auth: login OK."
  rm -f "${tmp}"
}

# Uses RabbitMQ Management HTTP API (management plugin). Idempotent: creates user only if GET returns 404.
ensure_rabbitmq_user() {
  if [ -z "${RABBITMQ_HOST:-}" ] || [ "${SKIP_RABBITMQ_SETUP:-0}" = "1" ]; then
    echo "Skipping RabbitMQ user bootstrap (unset RABBITMQ_HOST or SKIP_RABBITMQ_SETUP=1)."
    return 0
  fi

  if [ -z "${RABBITMQ_ADMIN_USER:-}" ] || [ -z "${RABBITMQ_ADMIN_PASS:-}" ]; then
    echo "Skipping RabbitMQ user bootstrap (RABBITMQ_ADMIN_USER / RABBITMQ_ADMIN_PASS unset)."
    return 0
  fi

  if [ -z "${RABBITMQ_USERNAME:-}" ] || [ -z "${RABBITMQ_PASSWORD:-}" ]; then
    echo "Skipping RabbitMQ user bootstrap (RABBITMQ_USERNAME or RABBITMQ_PASSWORD unset)."
    return 0
  fi

  local mgmt_port="${RABBITMQ_MANAGEMENT_PORT:-15672}"
  local base="http://${RABBITMQ_HOST}:${mgmt_port}/api"
  local tmp vhost user_enc vh_enc code user_json perm_json

  tmp="$(mktemp)"

  vhost="${RABBITMQ_VHOST:-/}"
  user_enc="$(
    RABBITMQ_USERNAME="${RABBITMQ_USERNAME}" python -c \
      'import os, urllib.parse; print(urllib.parse.quote(os.environ["RABBITMQ_USERNAME"], safe=""))'
  )"
  vh_enc="$(
    RABBITMQ_VHOST="${vhost}" python -c \
      'import os, urllib.parse; print(urllib.parse.quote(os.environ["RABBITMQ_VHOST"], safe=""))'
  )"

  echo "Waiting for RabbitMQ management API at ${RABBITMQ_HOST}:${mgmt_port}..."
  for i in $(seq 1 60); do
    code="$(
      curl -sS -o /dev/null -w "%{http_code}" \
        -u "${RABBITMQ_ADMIN_USER}:${RABBITMQ_ADMIN_PASS}" \
        "${base}/overview" 2>/dev/null || true
    )"
    code="${code:-000}"
    if [ "${code}" = "200" ]; then
      break
    fi
    if [ "${code}" = "401" ] || [ "${code}" = "403" ]; then
      echo "RabbitMQ management API rejected admin credentials (HTTP ${code})."
      rm -f "${tmp}"
      exit 1
    fi
    if [ "$i" -eq 60 ]; then
      echo "RabbitMQ management API not reachable (last HTTP code: ${code}). Is the management plugin enabled and RABBITMQ_MANAGEMENT_PORT correct?"
      rm -f "${tmp}"
      exit 1
    fi
    echo "  retry ${i}/60..."
    sleep 2
  done

  code="$(
    curl -sS -o "${tmp}" -w "%{http_code}" \
      -u "${RABBITMQ_ADMIN_USER}:${RABBITMQ_ADMIN_PASS}" \
      "${base}/users/${user_enc}" 2>/dev/null || true
  )"
  code="${code:-000}"

  case "${code}" in
    200)
      echo "RabbitMQ user '${RABBITMQ_USERNAME}' already exists."
      ;;
    404)
      echo "Creating RabbitMQ user '${RABBITMQ_USERNAME}'..."
      user_json="$(jq -n --arg p "${RABBITMQ_PASSWORD}" '{password: $p, tags: ""}')"
      code="$(
        curl -sS -o "${tmp}" -w "%{http_code}" \
          -u "${RABBITMQ_ADMIN_USER}:${RABBITMQ_ADMIN_PASS}" \
          -X PUT \
          -H "Content-Type: application/json" \
          --data "${user_json}" \
          "${base}/users/${user_enc}"
      )"
      if [ "${code}" != "201" ] && [ "${code}" != "204" ]; then
        echo "RabbitMQ PUT /users failed with HTTP ${code}"
        cat "${tmp}"
        rm -f "${tmp}"
        exit 1
      fi
      ;;
    *)
      echo "RabbitMQ GET /users/${RABBITMQ_USERNAME} failed with HTTP ${code}"
      cat "${tmp}"
      rm -f "${tmp}"
      exit 1
      ;;
  esac

  perm_json='{"configure":".*","write":".*","read":".*"}'
  code="$(
    curl -sS -o "${tmp}" -w "%{http_code}" \
      -u "${RABBITMQ_ADMIN_USER}:${RABBITMQ_ADMIN_PASS}" \
      -X PUT \
      -H "Content-Type: application/json" \
      --data "${perm_json}" \
      "${base}/permissions/${vh_enc}/${user_enc}"
  )"
  if [ "${code}" != "201" ] && [ "${code}" != "204" ]; then
    echo "RabbitMQ PUT /permissions failed with HTTP ${code}"
    cat "${tmp}"
    rm -f "${tmp}"
    exit 1
  fi
  echo "RabbitMQ permissions ensured for vhost '${vhost}'."

  rm -f "${tmp}"
}

bootstrap_postgres_roles

wait_for_db

echo "Applying migrations..."
migrate_out=""
for i in $(seq 1 60); do
  set +e
  migrate_out="$(python manage.py migrate --noinput 2>&1)"
  migration_status=$?
  set -e
  if [ "$migration_status" -eq 0 ]; then
    printf '%s\n' "${migrate_out}"
    break
  fi
  printf '%s\n' "${migrate_out}"
  if printf '%s' "${migrate_out}" | grep -qF 'InconsistentMigrationHistory'; then
    echo ""
    echo "migrate will not recover by retrying: django_migrations does not match this project (custom user model + admin)."
    echo "  Typical cause: POSTGRES_DB points at a database that already had Django tables from another service."
    echo "  Fix: use a dedicated database name for filemgr, or reset schema on that DB only (see: $0 --help)."
    exit 1
  fi
  if [ "$i" -eq 60 ]; then
    echo "migrate failed after 60 attempts."
    exit 1
  fi
  echo "migrate failed, retrying (${i}/60)..."
  sleep 2
done

if [ "${RESET_DB}" = true ]; then
  echo "Flushing database (--reset-db)..."
  python manage.py flush --no-input
fi

ensure_fixtures

echo "Ensuring local test storage directory exists..."
mkdir -p "${LOCAL_TEST_STORAGE_ROOT}"

if [ -n "${USERVER_AUTH_HOST:-}" ] && [ "${SKIP_USERVER_AUTH_SETUP:-0}" != "1" ]; then
  ensure_userver_auth
else
  echo "Skipping userver-auth bootstrap (unset USERVER_AUTH_HOST or SKIP_USERVER_AUTH_SETUP=1)."
fi

ensure_rabbitmq_user

echo "===== setup completed ====="
