#!/bin/bash

echo "WARNING: THIS PROCESS WILL DESTROY ANY EXISTING USERVER-FILEMGR DATABASE!"
echo "THIS IS IRREVERSIBLE!"

if [[ $ENV_MODE != "dev" ]]; then
  echo "!!!!! YOU ARE NOT UNDER A DEVELOPMENT ENVIRONMENT !!!!!"
  read -p "Are you sure you want to continue? (LAST CHANCE!)" -n 1 -r
  echo    # (optional) move to a new line
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      [[ "$0" = "$BASH_SOURCE" ]] && exit 1 || return 1 # handle exits from shell or function but don't exit interactive shell
  fi
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

echo "Resetting migrations..."
ls /code/app/migrations | grep -E "^[0-9]{4}_[0-9a-zA-Z_-]+.py$" | xargs -I STR rm /code/app/migrations/STR

echo "===== Reset Complete ====="
