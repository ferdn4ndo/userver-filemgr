# uServer FileMgr

[![UT and E2E tests](https://github.com/ferdn4ndo/userver-filemgr/actions/workflows/test_ut_e2e.yaml/badge.svg?branch=main)](https://github.com/ferdn4ndo/userver-filemgr/actions/workflows/test_ut_e2e.yml)
[![GitLeaks](https://github.com/ferdn4ndo/userver-filemgr/actions/workflows/test_code_leaks.yaml/badge.svg?branch=main)](https://github.com/ferdn4ndo/userver-filemgr/actions/workflows/test_code_leaks.yml)
[![ShellCheck](https://github.com/ferdn4ndo/userver-filemgr/actions/workflows/test_code_quality.yaml/badge.svg?branch=main)](https://github.com/ferdn4ndo/userver-filemgr/actions/workflows/test_code_quality.yml)
[![Release](https://img.shields.io/github/v/release/ferdn4ndo/userver-filemgr)](https://github.com/ferdn4ndo/userver-filemgr/releases)
[![MIT license](https://img.shields.io/badge/license-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)

File management microservice based on django-rest-framework with AWS S3 integration.

It's part of the [uServer](https://github.com/users/ferdn4ndo/projects/1) stack project.

## Using It

### Prepare the environment

Copy `filemgr/.env.template` to `filemgr/.env` and edit it accordingly. Key variables:

- **Database**: `POSTGRES_*` for the app user and database; optional **`POSTGRES_ROOT_USER`** / **`POSTGRES_ROOT_PASS`** so `setup.sh` can create the database and role on a local Postgres (omit both if the database is already provisioned). With those set, setup also runs **`GRANT USAGE, CREATE ON SCHEMA public`** to the app user, which **PostgreSQL 15+** requires for **`CREATE TABLE`** (including **`django_migrations`**). If you provision Postgres yourself, grant that (or equivalent) on your app database.
- **uServer-Auth**: `USERVER_AUTH_HOST` (full base URL, e.g. `http://userver-auth:5000`), system and user credentials, and **`USERVER_AUTH_SYSTEM_CREATION_TOKEN`** (required only when login fails and the system must be created). Set **`SKIP_USERVER_AUTH_SETUP=1`** to skip all auth HTTP bootstrap (e.g. offline work).
- **RabbitMQ**: `RABBITMQ_HOST`, **`RABBITMQ_PORT`** (AMQP, usually **5672**), **`RABBITMQ_USERNAME`** / **`RABBITMQ_PASSWORD`** for the app. Optional bootstrap: **`RABBITMQ_ADMIN_USER`** / **`RABBITMQ_ADMIN_PASS`** (a management-capable account) and **`RABBITMQ_MANAGEMENT_PORT`** (HTTP API, usually **15672**) so `setup.sh` can create the app user via the **management plugin** when it does not exist yet. Set **`SKIP_RABBITMQ_SETUP=1`** to skip. **`RABBITMQ_VHOST`** defaults to **`/`**.
- **Django**: `DJANGO_SECRET_KEY`, `VIRTUAL_HOST`, `LOCAL_TEST_STORAGE_ROOT`, and the rest as in the template.

### Run the application

From the repository root:

```sh
docker compose up --build
```

The **web** container runs **`setup.sh` automatically** on every start (via `entrypoint.sh`): it waits for Postgres, applies migrations, loads MIME fixtures only when the table is empty, ensures the local storage directory exists, reconciles **uServer-Auth** (see below), and optionally ensures a **RabbitMQ** application user exists (see below). Then it starts Gunicorn (production) or `runserver` (development) according to **`ENV_MODE`**.

The **qcluster** service only runs **`manage.py qcluster`**. It **`depends_on`** the web service with **`condition: service_healthy`**: the web container exposes port **5000** only after **`setup.sh`** finishes (including Postgres grants and **`migrate`**), which avoids **`qcluster`** racing migrations before **PostgreSQL 15+** **`public`** schema privileges are applied.

Access the application at [http://localhost:5000/](http://localhost:5000/) or whatever you configure (for example behind nginx-proxy via `VIRTUAL_HOST`).

### Bootstrap and `setup.sh`

`filemgr/setup.sh` is **idempotent**: safe to run on every container start. It does **not** run `makemigrations` in the container (generate migrations in development and commit them).

**uServer-Auth** integration follows the HTTP API implemented by the uServer-Auth service: it calls **`POST /auth/login`** first. If that returns **200**, system and user are already valid and **no** create/register calls are made. Otherwise it calls **`POST /auth/system`** (accepts **201** or **409 Conflict** if the system or token already exists) and **`POST /auth/register`** (accepts **201** or **409** if the user already exists), then verifies login again with **200**.

**RabbitMQ** bootstrap uses the broker’s **Management HTTP API** (the [management plugin](https://www.rabbitmq.com/docs/management) must be enabled and reachable from the filemgr container on **`RABBITMQ_MANAGEMENT_PORT`**). If **`RABBITMQ_ADMIN_USER`** / **`RABBITMQ_ADMIN_PASS`** are unset, this step is skipped (for example when the broker already provisions users via **`RABBITMQ_DEFAULT_USER`** or another process). Otherwise the script waits for **`GET /api/overview`**, then **`GET /api/users/{name}`**: **404** means it **`PUT`s** the user with **`RABBITMQ_PASSWORD`**, and in all cases it **`PUT`s** full configure/write/read permissions on **`RABBITMQ_VHOST`** so publishing from **`MessageBrokerService`** works.

To **wipe application data** inside the existing database (flush Django tables and reload fixtures), run setup manually with **`--reset-db`**:

```sh
docker exec -it userver-filemgr bash -c './setup.sh --reset-db'
```

For options:

```sh
docker exec -it userver-filemgr bash -c './setup.sh --help'
```

### Python dependencies (notable pins)

- **django-q2** (not the unmaintained **django-q** package): **Django 6** removed **`django.utils.baseconv`**, which **django-q 1.3.x** still imported; **django-q2** is the maintained fork and stays compatible through **Django 6**. The installed Python module name remains **`django_q`** (for example **`INSTALLED_APPS`**, **`manage.py qcluster`**, **`django_q.tasks`**).
- **django-redis 6.x** and **redis 5.x** go with **django-q2**’s supported stack (the old **django-q** + **redis 3.x** pin is no longer needed).
- **setuptools** is pinned so **`pkg_resources`** exists on minimal images if a dependency still expects it.

Application code and **`requirements.txt`** live under **`filemgr/`**; the Docker build context is **`filemgr/`** (see `docker-compose.yml`).

## Continuous integration

The **`.github/`** workflows are tailored to this repository:

- **Dependabot** watches **`filemgr/`** for pip and Docker (where `requirements.txt` and the `Dockerfile` live), and the repo root for GitHub Actions.
- **Unit tests** build from **`filemgr/Dockerfile`**, use Postgres and Redis services, run migrations, then **`python manage.py test`**.
- **Grype** builds the same image path and scans it; **ShellCheck**, **GitLeaks**, and **release** workflows use **`userver-filemgr`** / **`ferdn4ndo/userver-filemgr`** naming where relevant.
- Workflows trigger on pushes to **`main`** (and pull requests), aside from release triggers.

Adjust image names and secrets in **`.github/workflows/create_release_container.yaml`** if your Docker Hub repository differs.

## Documentation

The API schema is provided in two formats:

- with a GET at `<host>/docs/openapi/`, which will retrieve the [OpenAPI](https://swagger.io/specification/) YAML specification file with all the endpoints;

- with a GET at `<host>/docs/redoc/`, which will retrieve the [ReDoc](https://github.com/Redocly/redoc) API Reference Documentation for userver-filemgr, in an interactive interface;

### Endpoints Summary

#### `/media-images/` Namespace

- **GET** `/media-images/`: List images;
- **GET** `/media-images/<media-image-id>/`: Show image metadata and available sizes;
- **POST** `/media-images/<media-image-id>/download/`: Download the biggest size tag;
- **POST** `/media-images/<media-image-id>/download/<size-tag>/`: Download a specific size tag;

#### `/storages/` Namespace

- **GET** `/storages/`: List storages;
- **POST** `/storages/`: Create a new storage;
- **GET** `/storages/<storage-id>/`: Read a storage;
- **PATCH** `/storages/<storage-id>/`: Update a storage;
- **DELETE** `/storages/<storage-id>/`: Delete a storage;
- **GET** `/storages/<storage-id>/files/`: List the storage files;
- **GET** `/storages/<storage-id>/files/<file-id>/`: Read a file info;
- **PATCH** `/storages/<storage-id>/files/<file-id>/`: Update (metadata only) a file;
- **DELETE** `/storages/<storage-id>/files/<file-id>/`: Delete a file (move to trash);
- **POST** `/storages/<storage-id>/files/<file-id>/download/`: Download the raw storage file;
- **POST** `/storages/<storage-id>/upload-from-url`: Upload a new file to the storage from a URL;
- **POST** `/storages/<storage-id>/upload-from-file`: Upload a new file to the storage from a local one;
- **GET** `/storages/<storage-id>/trash/`: List the files in the recycled bin;
- **GET** `/storages/<storage-id>/trash/<file-id>/`: Read a file in the recycled bin/Permanently delete the file;
- **DELETE** `/storages/<storage-id>/trash/<file-id>/`: Permanently delete the file;

## Testing

Run the test suite inside the web container:

```sh
docker exec -it userver-filemgr bash -c "python manage.py test"
```

Limit to an app label if needed, for example:

```sh
docker exec -it userver-filemgr bash -c "python manage.py test api"
```

## Utils

General commands for userver-filemgr. Most of them are based on [django-extensions](https://github.com/django-extensions/django-extensions).

### Create DB models chart

Generate (and view) a graphviz graph of app models:

```
docker exec -it userver-filemgr bash -c "python manage.py graph_models -a -o filemgr_models.png"
```

### Generate list of URLs

Produce a tab-separated list of (url_pattern, view_function, name) tuples for a project:

```
docker exec -it userver-filemgr bash -c "python manage.py show_urls"
```

### Check templates

Check templates for rendering errors:

```
docker exec -it userver-filemgr bash -c "python manage.py validate_templates"
```

### Enhanced Django shell

Run the enhanced django shell:

```
docker exec -it userver-filemgr bash -c "python manage.py shell_plus"
```

### Updating OpenAPI schema

To update the OpenAPI schema:

```sh
docker exec -it userver-filemgr bash -c "python manage.py generateschema" > openapi-schema.yaml
```
