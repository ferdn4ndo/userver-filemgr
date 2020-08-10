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


echo "Reseting DB..."

PGPASSWORD=${POSTGRES_ROOT_PASS} psql -h "${POSTGRES_HOST}" -U "${POSTGRES_ROOT_USER}" -p "${POSTGRES_PORT}" <<EOF
  REVOKE CONNECT ON DATABASE ${POSTGRES_DB} FROM public;

  SELECT pid, pg_terminate_backend(pid)
  FROM pg_stat_activity
  WHERE pg_stat_activity.datname = '${POSTGRES_DB}' AND pid <> pg_backend_pid();

  DROP DATABASE IF EXISTS ${POSTGRES_DB};
  CREATE DATABASE ${POSTGRES_DB};

  GRANT CONNECT ON DATABASE ${POSTGRES_DB} TO public;


  DROP OWNED BY ${POSTGRES_USER};
  DROP USER IF EXISTS ${POSTGRES_USER};
  CREATE USER ${POSTGRES_USER} WITH ENCRYPTED PASSWORD '${POSTGRES_PASS}';

  REVOKE ALL PRIVILEGES ON DATABASE postgres FROM ${POSTGRES_USER};
  ALTER USER ${POSTGRES_USER} CREATEDB;
  GRANT ALL PRIVILEGES ON DATABASE ${POSTGRES_DB} TO ${POSTGRES_USER};
\gexec
EOF

echo "Reseting migrations"
#python manage.py flush --noinput
rm -f /code/filemgr/migrations/*.py

python manage.py makemigrations filemgr
python manage.py migrate filemgr

echo "Migrating..."
python manage.py migrate --run-syncdb

echo "Adding fixtures..."
python manage.py loaddata /code/filemgr/fixtures/image_sizes.yaml
python manage.py loaddata /code/filemgr/fixtures/mime_types.yaml

echo "Creating local test storage folder"
mkdir -p "${LOCAL_TEST_STORAGE_ROOT}"

echo "Creating auth system"

[[ -z "${LETSENCRYPT_HOST}" ]] && protocol="http" || protocol="https"

system_resp=$(
  curl -sS -X POST "${protocol}://${USERVER_AUTH_HOST}/auth/system" \
    -H "Authorization: Token ${USERVER_AUTH_SYSTEM_CREATION_TOKEN}" \
    -H "Content-Type: application/json" \
    --data @- <<END
{
  "name": "${USERVER_AUTH_SYSTEM_NAME}",
  "token": "${USERVER_AUTH_SYSTEM_TOKEN}"
}
END
)
echo "System response:"
echo "${system_resp}"

echo "Creating auth user"

reg_resp=$(
  curl -sS -X POST "${protocol}://${USERVER_AUTH_HOST}/auth/register" \
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
)

echo "Register response:"
echo "${reg_resp}"

login_resp=$(
  curl -sS -X POST "${protocol}://${USERVER_AUTH_HOST}/auth/login" \
    -H "Content-Type: application/json" \
    --data @- <<END
{
  "username": "${USERVER_AUTH_USER}",
  "system_name": "${USERVER_AUTH_SYSTEM_NAME}",
  "system_token": "${USERVER_AUTH_SYSTEM_TOKEN}",
  "password": "${USERVER_AUTH_PASSWORD}"
}
END
)

echo "Login response:"
echo "${login_resp}"

echo "SETUP COMPLETED!"
exit 0
