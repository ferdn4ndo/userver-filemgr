#!/usr/bin/env bash
# Merge selected keys from .env.template into .env when missing or empty (bootstrap / auth block).
# Skips values that look like placeholders (e.g. <POSTGRES_USER>).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${ENV_FILE:-$ROOT/.env}"
TEMPLATE="${ENV_TEMPLATE:-$ROOT/.env.template}"

# Keys aligned with .env.template bootstrap section (and related lines).
AUTOFILL_KEYS=(
  SKIP_AUTH_BOOTSTRAP
  SYSTEM_CREATION_TOKEN
  FILEMGR_BOOTSTRAP_SYSTEM_NAME
  FILEMGR_BOOTSTRAP_CUSTOM_SYSTEM_TOKEN
  FILEMGR_SYSTEM_TOKEN
  FILEMGR_BOOTSTRAP_ADMIN_USERNAME
  FILEMGR_BOOTSTRAP_ADMIN_PASSWORD
  FILEMGR_BOOTSTRAP_ADMIN_IS_ADMIN
)

is_placeholder() {
  local v="$1"
  [[ "$v" == *'<'* ]] && [[ "$v" == *'>'* ]]
}

trim() {
  local s="$1"
  s="${s#"${s%%[![:space:]]*}"}"
  s="${s%"${s##*[![:space:]]}"}"
  printf '%s' "$s"
}

# Current value for KEY in env file (last occurrence), or empty.
current_val() {
  local f="$1" key="$2"
  local line
  line="$(grep -E "^${key}=" "$f" 2>/dev/null | tail -n1 || true)"
  [[ -z "$line" ]] && { printf ''; return; }
  local v="${line#*=}"
  v="${v%$'\r'}"
  # strip optional double quotes
  if [[ "$v" == \"*\" ]]; then
    v="${v#\"}"
    v="${v%\"}"
  fi
  printf '%s' "$v"
}

# Load defaults from template: key -> value (only keys we care about).
declare -A tmpl_val=()
declare -A tmpl_has=()

load_template_defaults() {
  local line key val
  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    [[ "$line" =~ ^([A-Za-z_][A-Za-z0-9_]*)=(.*)$ ]] || continue
    key="${BASH_REMATCH[1]}"
    val="${BASH_REMATCH[2]}"
    val="${val%$'\r'}"
    tmpl_val["$key"]="$val"
    tmpl_has["$key"]=1
  done < "$TEMPLATE"
}

upsert_line() {
  local file="$1" key="$2" val="$3"
  local tmp found=0
  tmp="$(mktemp)"
  if [[ -s "$file" ]]; then
    while IFS= read -r line || [[ -n "$line" ]]; do
      if [[ "$line" =~ ^${key}= ]]; then
        printf '%s=%s\n' "$key" "$val" >> "$tmp"
        found=1
      else
        printf '%s\n' "$line" >> "$tmp"
      fi
    done < "$file"
  fi
  if [[ "$found" -eq 0 ]]; then
    printf '%s=%s\n' "$key" "$val" >> "$tmp"
  fi
  cat "$tmp" > "$file"
  rm -f "$tmp"
}

main() {
  if [[ ! -f "$TEMPLATE" ]]; then
    return 0
  fi

  local dir
  dir="$(dirname "$ENV_FILE")"
  if [[ ! -d "$dir" ]]; then
    return 0
  fi

  if [[ -f "$ENV_FILE" ]] && [[ ! -w "$ENV_FILE" ]]; then
    echo "env_autofill: .env not writable, skipping"
    return 0
  fi

  if [[ ! -f "$ENV_FILE" ]]; then
    if ! : >"$ENV_FILE" 2>/dev/null; then
      echo "env_autofill: cannot create .env, skipping"
      return 0
    fi
  fi

  load_template_defaults

  local key def cur
  for key in "${AUTOFILL_KEYS[@]}"; do
    [[ -z "${tmpl_has[$key]:-}" ]] && continue
    def="${tmpl_val[$key]}"
    def="$(trim "$def")"
    [[ -z "$def" ]] && continue
    if is_placeholder "$def"; then
      continue
    fi
    cur="$(current_val "$ENV_FILE" "$key")"
    cur="$(trim "$cur")"
    if [[ -n "$cur" ]]; then
      continue
    fi
    echo "env_autofill: setting ${key} from .env.template (was empty)"
    upsert_line "$ENV_FILE" "$key" "$def"
  done
}

main "$@"
