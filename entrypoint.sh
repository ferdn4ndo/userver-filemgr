#!/bin/bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")" || exit 1
source ./colors.sh

# Fill empty bootstrap/auth keys in .env from .env.template (non-placeholder values only), then export for setup/bootstrap.
if [ -f ./scripts/env_autofill.sh ]; then
  if ! bash ./scripts/env_autofill.sh; then
    echo -e "${COLOR_YELLOW}env_autofill.sh failed (non-fatal).${COLOR_RESET}"
  fi
fi
if [ -f ./.env ]; then
  set -a
  # shellcheck disable=SC1091
  . ./.env
  set +a
fi

case "${ENV_MODE:-development}" in
  prod)
    echo -e "${COLOR_BLUE}Starting in production mode (Go)...${COLOR_RESET}"
    ;;
  *)
    echo -e "${COLOR_BLUE}Starting in development mode (Go)...${COLOR_RESET}"
    ;;
esac

if [ "${SKIP_DB_SETUP:-0}" = "1" ]; then
  echo -e "${COLOR_YELLOW}SKIP_DB_SETUP=1: skipping database setup (migrations not applied).${COLOR_RESET}"
else
  bash ./setup.sh
fi

: "${APP_BIN:=./main}"
if [ ! -x "$APP_BIN" ] && [ -x "/app/main" ]; then
  APP_BIN="/app/main"
fi
if [ ! -x "$APP_BIN" ]; then
  echo -e "${COLOR_RED}No application binary found (set APP_BIN or place ./main).${COLOR_RESET}" >&2
  exit 1
fi

PORT="${APP_PORT:-${PORT:-${FLASK_PORT:-5000}}}"
export APP_PORT="$PORT"

echo -e "${COLOR_BLUE}Starting API on 0.0.0.0:${PORT}...${COLOR_RESET}"
exec "$APP_BIN" app:serve
