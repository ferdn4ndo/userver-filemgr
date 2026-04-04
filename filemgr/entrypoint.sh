#!/bin/bash
set -eo pipefail

cd /code

bash ./setup.sh

if [ "${ENV_MODE}" == "prod" ]
then
  echo "Starting in production mode..."

  gunicorn --access-logfile /code/logs/access.log --worker-tmp-dir /dev/shm --workers "${GUVICORN_WORKERS:-3}" --bind 0.0.0.0:5000 api.wsgi:application
else
  echo "Starting in development mode..."

  python manage.py runserver 0.0.0.0:5000
fi
