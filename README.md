# uServer FileMgr

[![Unit and integration tests](https://github.com/ferdn4ndo/userver-filemgr/actions/workflows/test_ut_e2e.yaml/badge.svg?branch=main)](https://github.com/ferdn4ndo/userver-filemgr/actions/workflows/test_ut_e2e.yaml)
[![GitLeaks](https://github.com/ferdn4ndo/userver-filemgr/actions/workflows/test_code_leaks.yaml/badge.svg?branch=main)](https://github.com/ferdn4ndo/userver-filemgr/actions/workflows/test_code_leaks.yaml)
[![ShellCheck](https://github.com/ferdn4ndo/userver-filemgr/actions/workflows/test_code_quality.yaml/badge.svg?branch=main)](https://github.com/ferdn4ndo/userver-filemgr/actions/workflows/test_code_quality.yaml)
[![Release](https://img.shields.io/github/v/release/ferdn4ndo/userver-filemgr)](https://github.com/ferdn4ndo/userver-filemgr/releases)
[![MIT license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)

File management microservice written in **Go** (Gin) with **PostgreSQL** metadata and **Amazon S3** or **local disk** object storage. It is part of the [uServer](https://github.com/users/ferdn4ndo/projects/1) stack.

## Requirements

- **Go 1.25+** (CI and release images use **Go 1.26**).
- **PostgreSQL** for `core_*` tables (compatible with existing deployments; see `docs/database-migration.md`).
- **uServer-Auth** (or compatible) for `GET /auth/me` when a token is not yet cached locally.

## Configuration

Copy `.env.template` to `.env` and adjust:

| Variable | Purpose |
|----------|---------|
| `POSTGRES_*` | Application database connection |
| `POSTGRES_ROOT_USER` / `POSTGRES_ROOT_PASS` | Optional: `setup.sh` creates the app role and databases when set |
| `USERVER_AUTH_HOST` | Base URL of **userver-auth** (e.g. `http://userver-auth:5000`). Alias: **`AUTH_HOST`** if this is empty. |
| `USERVER_AUTH_TIMEOUT_SECS` | HTTP timeout for Auth calls (default `15`, clamped 5–120). |
| `SYSTEM_CREATION_TOKEN` / `FILEMGR_BOOTSTRAP_*` | Optional: `setup.sh` runs **`bootstrap:auth`** after migrations to call Auth `POST /auth/system` and optionally `POST /auth/register` (same contract as **userver-auth**). Set `SKIP_AUTH_BOOTSTRAP=1` to skip. On success, **`bootstrap:auth`** fills only **empty or placeholder** (`<...>`) keys in **`ENV_FILE`** / **`FILEMGR_ENV_FILE`** (default `.env`); it does **not** overwrite existing non-empty values. Set **`FILEMGR_SKIP_PERSIST_BOOTSTRAP_ENV=1`** to disable writes. See `.env.template`. |
| `LOCAL_STORAGE_ROOT` | Default root for `LOCAL` storages |
| `APP_PUBLIC_BASE_URL` | Optional absolute base for local download URLs |
| `DOWNLOAD_EXP_BYTES_SECS_RATIO` | Size-based TTL for download links (legacy formula) |

## Run with Docker Compose

Same layout as **userver-auth**: Compose only runs the app service; build/tests use **`make go-build`** / **`make go-test`** (Dockerfile `dev` target + `docker run`, no compile/postgres services in Compose).

The Compose file defaults **`MIGRATE_BIN`** and **`APP_BIN`** to **`/app/main`** (the static binary in the image). A host-built **`./out/userver-filemgr`** is often linked against glibc and will not run on the Alpine runtime—override those variables only if you use a matching dev image or binary.

```sh
docker compose up --build userver-filemgr
```

Compose attaches to the **`nginx-proxy`** external network (same as [userver-web](https://github.com/ferdn4ndo/userver-web)). Create it if needed: `docker network create nginx-proxy`. Point `POSTGRES_*` in `.env` at your Postgres (e.g. another service on that network).

The container runs `setup.sh` from `entrypoint.sh` (unless `SKIP_DB_SETUP=1`), then `app:serve` on port **5000**. `setup.sh` applies migrations and then runs **`bootstrap:auth`** when bootstrap-related env vars are set (or no-ops when unset).

## API

- `GET /healthz` — health check (used by Docker `HEALTHCHECK` and `health:probe`).
- Authenticated routes accept **`Authorization: Token <access_token>`** (legacy) or **`Authorization: Bearer <access_token>`** (JWT from userver-auth). Tokens are resolved against `core_usertoken` or validated via **`GET {USERVER_AUTH_HOST}/auth/me`** with `Bearer` (same as userver-auth).
- Primary resources live under `/storages/...` (list/create storages, files, trash, storage users, uploads, downloads).

A static OpenAPI snapshot is kept at `docs/openapi-schema-reference.yaml` for reference.

## SQL migrations

Migrations live in `migrations/` and are applied with **golang-migrate** (`migrate:up` / `migrate:down` subcommands, also invoked from `setup.sh`). See `docs/database-migration.md` and `scripts/baseline_golang_migrate.sh` for upgrading existing databases.

## Testing

```sh
make test                         # host Go + coverage.out
make go-test                      # tests inside Dockerfile dev stage (pinned Go)
make go-test-coverage             # same, with coverage.out in the repo dir
DOCKER_NETWORK=nginx-proxy make go-test-integration   # needs POSTGRES_TEST_URL in .env
```

CI runs unit + integration tests inside `golang:1.26-bookworm`; locally, `make go-test` matches the pinned toolchain without adding Compose services.

## Documentation

- [`docs/architecture.md`](docs/architecture.md) — components and topology
- [`docs/database-migration.md`](docs/database-migration.md) — database cutover notes

## Continuous integration

- **test_ut_e2e.yaml** — unit tests + Postgres-backed **integration** tests (`-tags=integration`) inside `golang:1.26-bookworm`, plus a Docker image build smoke step.
- **test_grype_scan.yaml** / **create_release_container.yaml** — build context is the **repository root** (`Dockerfile`).

## License

MIT
