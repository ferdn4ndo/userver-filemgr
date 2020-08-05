# uServer FileMgr

File management microservice based on django-rest-framework with AWS S3 integration

It's part of the [uServer](https://github.com/users/ferdn4ndo/projects/1) stack project.

## Using It

### Prepare the environment

Copy `.env.template` to `.env` and edit it accordingly.

### Run the Application

```sh
docker-compose up --build
```

### Setup it (first run only)

Take a look at `setup.sh` and adjust it accordingly if needed, then run:

```sh
docker exec -it userver-filemgr sh -c "./setup.sh"
```

Access the application at the address [http://localhost:5000/](http://localhost:5000/) or any other environment configuration you made.

## Documentation

The API schema is provided in two formats:

* with a GET at `<host>/docs/openapi/`, which will retrieve the [OpenAPI](https://swagger.io/specification/) YAML specification file with all the endpoints;

* with a GET at `<host>/docs/red/`, which will retrieve the [ReDoc](https://github.com/Redocly/redoc) API Reference Documentation for userver-filemgr, in an interactive interface;
 

## Testing

### Without coverage:

```sh
docker exec -it userver-filemgr sh -c "python manage.py test filemgr"
```

### With coverage:

```sh
docker exec -it userver-filemgr sh -c "python manage.py cov filemgr"
```


## Utils

General commands for userver-filemgr. Most of them are based on [django-extensions](https://github.com/django-extensions/django-extensions).

### Create DB models chart

Generate (and view) a graphviz graph of app models:

```
docker exec -it userver-filemgr sh -c "python manage.py graph_models -a -o filemgr_models.png"
```

### Generate list of URLs

Produce a tab-separated list of (url_pattern, view_function, name) tuples for a project:

```
docker exec -it userver-filemgr sh -c "python manage.py show_urls"
```

### Check templates

Check templates for rendering errors:

```
docker exec -it userver-filemgr sh -c "python manage.py validate_templates"
```

### Enhanced Django shell

Run the enhanced django shell:

```
docker exec -it userver-filemgr sh -c "python manage.py shell_plus"
```


### Updating OpenAPI schema

To update the OpenAPI schema:

```sh
docker exec -it userver-filemgr sh -c "python manage.py generateschema" > openapi-schema.yaml
```


